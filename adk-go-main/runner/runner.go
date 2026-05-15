// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package runner provides a runtime for ADK agents.
package runner

import (
	"context"
	"fmt"
	"iter"
	"log"
	"time"

	"google.golang.org/genai"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/artifact"
	"google.golang.org/adk/internal/agent/parentmap"
	"google.golang.org/adk/internal/agent/runconfig"
	artifactinternal "google.golang.org/adk/internal/artifact"
	icontext "google.golang.org/adk/internal/context"
	"google.golang.org/adk/internal/llminternal"
	imemory "google.golang.org/adk/internal/memory"
	"google.golang.org/adk/internal/plugininternal"
	"google.golang.org/adk/internal/utils"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/model"
	"google.golang.org/adk/plugin"
	"google.golang.org/adk/session"
)

// Config is used to create a [Runner].
type Config struct {
	AppName string
	// Root agent which starts the execution.
	Agent          agent.Agent
	SessionService session.Service

	// optional
	ArtifactService artifact.Service
	// optional
	MemoryService memory.Service
	// optional
	PluginConfig PluginConfig
	// optional
	AutoCreateSession bool
}

type PluginConfig struct {
	Plugins      []*plugin.Plugin
	CloseTimeout time.Duration
}

type RunOption func(*runOptions)

type runOptions struct {
	stateDelta map[string]any
}

// WithStateDelta sets a state delta for the run invocation.
func WithStateDelta(delta map[string]any) RunOption {
	return func(o *runOptions) {
		o.stateDelta = delta
	}
}

// New creates a new [Runner].
// 中文业务注释：New 只做运行时装配，不启动任何对话；这里会校验根 Agent/SessionService，构建 Agent 父子关系表，并初始化插件管理器。
func New(cfg Config) (*Runner, error) {
	if cfg.Agent == nil {
		return nil, fmt.Errorf("root agent is required")
	}

	if cfg.SessionService == nil {
		return nil, fmt.Errorf("session service is required")
	}

	parents, err := parentmap.New(cfg.Agent)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent tree: %w", err)
	}

	pluginManager, err := plugininternal.NewPluginManager(plugininternal.PluginConfig{
		Plugins:      cfg.PluginConfig.Plugins,
		CloseTimeout: cfg.PluginConfig.CloseTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin manager: %w", err)
	}

	return &Runner{
		appName:           cfg.AppName,
		rootAgent:         cfg.Agent,
		sessionService:    cfg.SessionService,
		artifactService:   cfg.ArtifactService,
		memoryService:     cfg.MemoryService,
		parents:           parents,
		pluginManager:     pluginManager,
		autoCreateSession: cfg.AutoCreateSession,
	}, nil
}

// Runner manages the execution of the agent within a session, handling message
// processing, event generation, and interaction with various services like
// artifact storage, session management, and memory.
type Runner struct {
	appName         string
	rootAgent       agent.Agent
	sessionService  session.Service
	artifactService artifact.Service
	memoryService   memory.Service

	parents           parentmap.Map
	pluginManager     *plugininternal.PluginManager
	autoCreateSession bool
}

// Run runs the agent for the given user input, yielding events from agents.
// For each user message it finds the proper agent within an agent tree to
// continue the conversation within the session.
// 中文业务注释：Run 是一次用户请求的总入口。它返回事件迭代器，所以调用方可以边读边推给前端，而不是等待整轮完成。
func (r *Runner) Run(ctx context.Context, userID, sessionID string, msg *genai.Content, cfg agent.RunConfig, opts ...RunOption) iter.Seq2[*session.Event, error] {
	// TODO(hakim): we need to validate whether cfg is compatible with the Agent.
	//   see adk-python/src/google/adk/runners.py Runner._new_invocation_context.
	// TODO: setup tracer.
	return func(yield func(*session.Event, error) bool) {
		options := runOptions{}
		for _, opt := range opts {
			opt(&options)
		}

		// 中文业务注释：先恢复已有会话。Agent 的上下文、历史消息、状态都从 SessionService 来；如果开启 AutoCreateSession，找不到会话时会自动创建。
		var storedSession session.Session
		getResp, err := r.sessionService.Get(ctx, &session.GetRequest{
			AppName:   r.appName,
			UserID:    userID,
			SessionID: sessionID,
		})
		if err != nil {
			if !r.autoCreateSession {
				yield(nil, err)
				return
			}
			createResp, err := r.sessionService.Create(ctx, &session.CreateRequest{
				AppName:   r.appName,
				UserID:    userID,
				SessionID: sessionID,
			})
			if err != nil {
				yield(nil, err)
				return
			}
			storedSession = createResp.Session
		} else {
			storedSession = getResp.Session
		}

		// 中文业务注释：根据历史事件决定本轮继续由哪个 Agent 接手。典型场景：上轮子 Agent 发起工具调用，本轮用户返回工具确认/函数响应，需要回到同一个子 Agent。
		agentToRun, err := r.findAgentToRun(storedSession, msg)
		if err != nil {
			yield(nil, err)
			return
		}

		ctx = parentmap.ToContext(ctx, r.parents)
		ctx = runconfig.ToContext(ctx, &runconfig.RunConfig{
			StreamingMode: runconfig.StreamingMode(cfg.StreamingMode),
		})
		ctx = plugininternal.ToContext(ctx, r.pluginManager)

		var artifacts agent.Artifacts
		if r.artifactService != nil {
			artifacts = &artifactinternal.Artifacts{
				Service:   r.artifactService,
				SessionID: storedSession.ID(),
				AppName:   storedSession.AppName(),
				UserID:    storedSession.UserID(),
			}
		}

		var memoryImpl agent.Memory = nil
		if r.memoryService != nil {
			memoryImpl = &imemory.Memory{
				Service:   r.memoryService,
				SessionID: storedSession.ID(),
				UserID:    storedSession.UserID(),
				AppName:   storedSession.AppName(),
			}
		}

		// 中文业务注释：组装 InvocationContext。后续 Agent、Tool、Plugin 都通过这个上下文拿到 Session/Artifact/Memory/用户输入等能力。
		ctx := icontext.NewInvocationContext(ctx, icontext.InvocationContextParams{
			Artifacts:   artifacts,
			Memory:      memoryImpl,
			Session:     storedSession,
			Agent:       agentToRun,
			UserContent: msg,
			RunConfig:   &cfg,
		})
		// 中文业务注释：先把用户消息作为 Event 追加到 Session，再让 Agent 运行。这样模型构造上下文时能看到最新用户输入。
		ctx, err = r.appendMessageToSession(ctx, storedSession, msg, cfg.SaveInputBlobsAsArtifacts, r.pluginManager, options.stateDelta)
		if err != nil {
			yield(nil, err)
			return
		}

		pluginManager := r.pluginManager
		if pluginManager != nil {
			// Defer the after run callbacks to perform global cleanup tasks or finalizing logs and metrics data.
			// This does NOT emit any event.
			defer pluginManager.RunAfterRunCallback(ctx)

			earlyExitResult, err := pluginManager.RunBeforeRunCallback(ctx)
			if earlyExitResult != nil || err != nil {
				earlyExitEvent := session.NewEvent(ctx.InvocationID())
				earlyExitEvent.Author = "user"
				earlyExitEvent.LLMResponse = model.LLMResponse{
					Content: msg,
				}
				if err := r.sessionService.AppendEvent(ctx, storedSession, earlyExitEvent); err != nil {
					yield(nil, fmt.Errorf("failed to add event to session: %w", err))
					return
				}
				yield(earlyExitEvent, err)
				return
			}
		}

		// 中文业务注释：开始消费 Agent 事件流。每个非 partial 事件都会落入 Session，partial 事件只用于前端实时展示。
		for event, err := range agentToRun.Run(ctx) {
			if err != nil {
				if !yield(event, err) {
					return
				}
				continue
			}

			if pluginManager != nil {
				modifiedEvent, err := pluginManager.RunOnEventCallback(ctx, event)
				if err != nil {
					if !yield(nil, err) {
						return
					}
					continue
				}
				if modifiedEvent != nil {
					event = modifiedEvent
				}
			}

			// only commit non-partial event to a session service
			if !event.LLMResponse.Partial {
				if err := r.sessionService.AppendEvent(ctx, storedSession, event); err != nil {
					yield(nil, fmt.Errorf("failed to add event to session: %w", err))
					return
				}
			}

			if !yield(event, nil) {
				return
			}
		}
	}
}

// 中文业务注释：把用户输入转换成 session.Event。这里也处理输入附件：如果配置了保存 InlineData，会先写入 Artifact，再用文本占位符替换原始二进制。
func (r *Runner) appendMessageToSession(ctx agent.InvocationContext, storedSession session.Session, msg *genai.Content, saveInputBlobsAsArtifacts bool, pluginManager *plugininternal.PluginManager, stateDelta map[string]any) (agent.InvocationContext, error) {
	if msg == nil {
		return ctx, nil
	}
	if pluginManager != nil {
		modifiedMsg, err := pluginManager.RunOnUserMessageCallback(ctx, msg)
		if err != nil {
			return ctx, fmt.Errorf("error running on run user message callback : %w", err)
		}
		if modifiedMsg != nil {
			msg = modifiedMsg
			// update ctx user message
			ctx = icontext.NewInvocationContext(ctx, icontext.InvocationContextParams{
				Artifacts:    ctx.Artifacts(),
				Memory:       ctx.Memory(),
				Session:      ctx.Session(),
				Agent:        ctx.Agent(),
				UserContent:  msg,
				RunConfig:    ctx.RunConfig(),
				InvocationID: ctx.InvocationID(),
			})
		}
	}

	artifactsService := ctx.Artifacts()
	if artifactsService != nil && saveInputBlobsAsArtifacts {
		for i, part := range msg.Parts {
			if part.InlineData == nil {
				continue
			}
			fileName := fmt.Sprintf("artifact_%s_%d", ctx.InvocationID(), i)
			if _, err := artifactsService.Save(ctx, fileName, part); err != nil {
				return ctx, fmt.Errorf("failed to save artifact %s: %w", fileName, err)
			}
			// Replace the part with a text placeholder
			msg.Parts[i] = &genai.Part{
				Text: fmt.Sprintf("Uploaded file: %s. It has been saved to the artifacts", fileName),
			}
		}
	}

	event := session.NewEvent(ctx.InvocationID())

	event.Author = "user"
	event.LLMResponse = model.LLMResponse{
		Content: msg,
	}
	if stateDelta != nil {
		event.Actions.StateDelta = stateDelta
	}

	if err := r.sessionService.AppendEvent(ctx, storedSession, event); err != nil {
		return ctx, fmt.Errorf("failed to append event to sessionService: %w", err)
	}
	return ctx, nil
}

// findAgentToRun returns the agent that should handle the next request based on
// session history.
// 中文业务注释：Agent 选择策略。优先处理“用户返回的函数响应/确认”对应的原始 Agent；否则回看最近事件，找可继续接手的子 Agent；最后退回根 Agent。
func (r *Runner) findAgentToRun(session session.Session, msg *genai.Content) (agent.Agent, error) {
	if event := handleUserFunctionCallResponse(session.Events(), msg); event != nil {
		subAgent := r.rootAgent.FindAgent(event.Author)
		if subAgent != nil {
			return subAgent, nil
		}
		log.Printf("Function call from an unknown agent: %s, event id: %s", event.Author, event.ID)
	}

	events := session.Events()
	for i := events.Len() - 1; i >= 0; i-- {
		event := events.At(i)

		if event.Author == "user" {
			continue
		}

		subAgent := r.rootAgent.FindAgent(event.Author)
		// Agent not found, continue looking for the other event.
		if subAgent == nil {
			log.Printf("Event from an unknown agent: %s, event id: %s", event.Author, event.ID)
			continue
		}

		if r.isTransferableAcrossAgentTree(subAgent) {
			return subAgent, nil
		}
	}

	// Falls back to root agent if no suitable agents are found in the session.
	return r.rootAgent, nil
}

// handleUserFunctionCallResponse finds the function call event that matches the function response id
// delivered by the user in the latest event.
func handleUserFunctionCallResponse(events session.Events, msg *genai.Content) *session.Event {
	if events.Len() == 0 {
		return nil
	}

	functionResponses := utils.FunctionResponses(msg)
	if len(functionResponses) == 0 {
		return nil
	}

	// This assumes that even if user provides multiple function responses, all the function calls
	// were made by the same agent. Otherwise it would be impossible to rearrange session events
	// such that every function response has a corresponding call filtering by author.
	callID := functionResponses[0].ID
	for i := events.Len() - 1; i >= 0; i-- {
		event := events.At(i)
		for _, part := range utils.FunctionCalls(event.Content) {
			if part.ID == callID {
				return event
			}
		}
	}
	return nil
}

// checks if the agent and its parent chain allow transfer up the tree.
func (r *Runner) isTransferableAcrossAgentTree(agentToRun agent.Agent) bool {
	for curAgent := agentToRun; curAgent != nil; curAgent = r.parents[curAgent.Name()] {
		llmAgent, ok := curAgent.(llminternal.Agent)
		if !ok {
			return false
		}

		if llminternal.Reveal(llmAgent).DisallowTransferToParent {
			return false
		}
	}

	return true
}
