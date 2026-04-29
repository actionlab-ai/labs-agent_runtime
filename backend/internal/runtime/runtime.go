package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/model"
	"novel-agent-runtime/internal/runstore"
	"novel-agent-runtime/internal/skill"
)

type Runtime struct {
	Config         config.Config
	ModelConfig    ModelConfig
	Registry       *skill.Registry
	Model          *model.Client
	Store          *runstore.Store
	Debug          bool
	ProjectID      string
	ProjectContext string
	ProjectDocs    ProjectDocumentProvider
}

type RunResult struct {
	FinalText string
	RunDir    string
	RunID     string
}

const (
	SkillRunStatusCompleted  = "completed"
	SkillRunStatusNeedsInput = "needs_input"
)

type AskHumanOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type AskHumanQuestion struct {
	Field       string           `json:"field,omitempty"`
	Header      string           `json:"header,omitempty"`
	Question    string           `json:"question"`
	Options     []AskHumanOption `json:"options,omitempty"`
	MultiSelect bool             `json:"multi_select,omitempty"`
}

type AskHumanRequest struct {
	Reason    string             `json:"reason,omitempty"`
	Questions []AskHumanQuestion `json:"questions"`
}

type AskHumanAnswer struct {
	Answers map[string]string `json:"answers,omitempty"`
	Notes   string            `json:"notes,omitempty"`
}

type AskHumanPause struct {
	ToolCallID string          `json:"tool_call_id"`
	Request    AskHumanRequest `json:"request"`
}

func (p *AskHumanPause) Error() string {
	return "skill requested human input"
}

type SkillExecutionState struct {
	SkillID           string                   `json:"skill_id"`
	SafeID            string                   `json:"safe_id"`
	OriginalUserInput string                   `json:"original_user_input"`
	InvocationArgs    map[string]any           `json:"invocation_args,omitempty"`
	CompiledPrompt    string                   `json:"compiled_prompt"`
	Conversation      []model.Message          `json:"conversation"`
	NextRound         int                      `json:"next_round"`
	PendingToolCallID string                   `json:"pending_tool_call_id,omitempty"`
	ReadState         map[string]FileReadState `json:"read_state,omitempty"`
}

type SkillRunResult struct {
	Status   string               `json:"status"`
	Text     string               `json:"text,omitempty"`
	AskHuman *AskHumanRequest     `json:"ask_human,omitempty"`
	State    *SkillExecutionState `json:"state,omitempty"`
	RunDir   string               `json:"run_dir"`
	RunID    string               `json:"run_id"`
}

type ProjectDocumentWriteRequest struct {
	ProjectID string         `json:"project_id"`
	Kind      string         `json:"kind"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type ProjectDocumentSummary struct {
	ProjectID string `json:"project_id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	BodyBytes int    `json:"body_bytes"`
}

type ProjectDocumentReadResult struct {
	ProjectID string         `json:"project_id"`
	Kind      string         `json:"kind"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type ProjectDocumentWriteResult struct {
	ProjectID string `json:"project_id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	BodyBytes int    `json:"body_bytes"`
	Synced    bool   `json:"synced"`
}

type ProjectDocumentProvider interface {
	ListProjectDocuments(context.Context, string) ([]ProjectDocumentSummary, error)
	ReadProjectDocument(context.Context, string, string) (ProjectDocumentReadResult, error)
	WriteProjectDocument(context.Context, ProjectDocumentWriteRequest) (ProjectDocumentWriteResult, error)
}

type ModelConfig struct {
	Provider       string
	ID             string
	BaseURL        string
	APIKey         string
	APIKeyEnv      string
	ContextWindow  int
	MaxOutput      int
	Temperature    float64
	TimeoutSeconds int
}

type activatedSkillRef struct {
	SkillID          string  `json:"skill_id"`
	ToolName         string  `json:"tool_name"`
	Name             string  `json:"name,omitempty"`
	Score            float64 `json:"score,omitempty"`
	ActivationReason string  `json:"activation_reason,omitempty"`
}

type activationPlan struct {
	Policy             string              `json:"policy"`
	RetentionPolicy    string              `json:"retention_policy"`
	SearchIndex        int                 `json:"search_index"`
	QueryFingerprint   string              `json:"query_fingerprint,omitempty"`
	MaxActivatedSkills int                 `json:"max_activated_skills"`
	MaxRetainedSkills  int                 `json:"max_retained_skills"`
	ActivationMinScore float64             `json:"activation_min_score"`
	ScoreRatio         float64             `json:"activation_score_ratio"`
	FirstScore         float64             `json:"first_score,omitempty"`
	Threshold          float64             `json:"threshold,omitempty"`
	WindowSkills       []activatedSkillRef `json:"window_skills,omitempty"`
	NewlyActivated     []activatedSkillRef `json:"newly_activated_skills,omitempty"`
	ReusedSkills       []activatedSkillRef `json:"reused_skills,omitempty"`
	RetainedSkills     []activatedSkillRef `json:"retained_skills,omitempty"`
	EvictedSkills      []activatedSkillRef `json:"evicted_skills,omitempty"`
	SkippedSkills      []string            `json:"skipped_skills,omitempty"`
	Unchanged          bool                `json:"unchanged,omitempty"`
}

type runState struct {
	SearchPerformed       bool
	SearchCount           int
	LastSearchFingerprint string
	RetainedSkills        []activatedSkillRef
}

type toolCallOutcome struct {
	Content           string
	FinalText         string
	SearchPerformed   bool
	SearchFingerprint string
	RetainedSkills    []activatedSkillRef
}

func New(cfg config.Config, modelCfg ModelConfig) (*Runtime, error) {
	modelCfg.Provider = firstNonEmpty(modelCfg.Provider, "openai_compatible")
	if modelCfg.ContextWindow <= 0 {
		modelCfg.ContextWindow = 131072
	}
	if modelCfg.MaxOutput <= 0 {
		modelCfg.MaxOutput = 4096
	}
	if modelCfg.Temperature == 0 {
		modelCfg.Temperature = 0.7
	}
	if modelCfg.TimeoutSeconds <= 0 {
		modelCfg.TimeoutSeconds = 180
	}
	if strings.TrimSpace(modelCfg.ID) == "" {
		return nil, fmt.Errorf("runtime model.id is required")
	}
	if strings.TrimSpace(modelCfg.BaseURL) == "" {
		return nil, fmt.Errorf("runtime model.base_url is required")
	}
	reg, err := skill.LoadRegistry(cfg.Runtime.SkillsDir)
	if err != nil {
		return nil, err
	}
	store, err := runstore.New(cfg.Runtime.RunsDir)
	if err != nil {
		return nil, err
	}
	client := model.NewOpenAICompatible(modelCfg.BaseURL, modelCfg.APIKeyEnv, modelCfg.ID, modelCfg.TimeoutSeconds)
	if strings.TrimSpace(modelCfg.APIKey) != "" {
		client = model.NewOpenAICompatibleWithAPIKey(modelCfg.BaseURL, modelCfg.APIKey, modelCfg.ID, modelCfg.TimeoutSeconds)
	}
	return &Runtime{Config: cfg, ModelConfig: modelCfg, Registry: reg, Model: client, Store: store}, nil
}

func (r *Runtime) DryRun(input string) error {
	hits := r.Registry.Search(input, 10)
	_ = r.Store.WriteJSON("dry-run/skill-search.json", hits)
	_ = r.Store.WriteJSON("dry-run/loaded-skills.json", r.Registry.List())
	assembly := r.buildRouterAssembly(input, runState{}, []model.Message{{Role: "user", Content: input}}, 1)
	_ = r.Store.WriteJSON("dry-run/router-round-01-assembly.json", assembly)
	_ = r.Store.WriteText("dry-run/router-round-01-assembly.md", renderRouterAssemblyMarkdown(assembly))
	return nil
}

func (r *Runtime) Run(ctx context.Context, userInput string) (RunResult, error) {
	state := runState{}
	conversation := []model.Message{{Role: "user", Content: userInput}}
	_ = r.Store.WriteJSON("router/initial-conversation.json", conversation)

	var lastText string
	for round := 1; round <= r.Config.Runtime.MaxToolRounds; round++ {
		assembly := r.buildRouterAssembly(userInput, state, conversation, round)
		_ = r.Store.WriteJSON(fmt.Sprintf("router/round-%02d-assembly.json", round), assembly)
		_ = r.Store.WriteText(fmt.Sprintf("router/round-%02d-assembly.md", round), renderRouterAssemblyMarkdown(assembly))
		_ = r.Store.WriteJSON(fmt.Sprintf("router/round-%02d-tools.json", round), assembly.Tools)

		chatReq := assembly.ChatRequest
		r.writeChatDebug(fmt.Sprintf("router/round-%02d-chat-request.json", round), chatReq)

		resp, err := r.Model.Chat(ctx, chatReq)
		if err != nil {
			r.writeChatError(fmt.Sprintf("router/round-%02d-chat-error.txt", round), err)
			return RunResult{}, err
		}
		_ = r.Store.WriteJSON(fmt.Sprintf("router/round-%02d-response.json", round), json.RawMessage(resp.Raw))
		_ = r.Store.WriteJSON(fmt.Sprintf("router/round-%02d-response-analysis.json", round), analyzeChatResponse(resp))
		if len(resp.Choices) == 0 {
			return RunResult{}, fmt.Errorf("model returned no choices")
		}
		msg := resp.Choices[0].Message
		conversation = append(conversation, msg)
		lastText = msg.Content
		if len(msg.ToolCalls) == 0 {
			if r.Config.Runtime.FallbackSkillSearch && round == 1 {
				if out, ok, err := r.fallbackSkillCall(ctx, userInput); err != nil {
					return RunResult{}, err
				} else if ok {
					return RunResult{FinalText: out, RunDir: r.Store.Root, RunID: r.Store.RunID}, nil
				}
			}
			return RunResult{FinalText: msg.Content, RunDir: r.Store.Root, RunID: r.Store.RunID}, nil
		}
		for _, tc := range msg.ToolCalls {
			outcome, err := r.handleToolCall(ctx, userInput, tc, state)
			if err != nil {
				outcome.Content = `{"error":` + jsonQuote(err.Error()) + `}`
			}
			if outcome.SearchPerformed {
				state.SearchPerformed = true
				state.SearchCount++
				state.LastSearchFingerprint = outcome.SearchFingerprint
			}
			if outcome.RetainedSkills != nil {
				state.RetainedSkills = outcome.RetainedSkills
			}
			if r.Config.Runtime.ReturnSkillDirect && strings.TrimSpace(outcome.FinalText) != "" {
				return RunResult{
					FinalText: outcome.FinalText,
					RunDir:    r.Store.Root,
					RunID:     r.Store.RunID,
				}, nil
			}
			conversation = append(conversation, model.Message{Role: "tool", ToolCallID: tc.ID, Content: outcome.Content})
		}
	}
	return RunResult{FinalText: lastText, RunDir: r.Store.Root, RunID: r.Store.RunID}, nil
}

func (r *Runtime) fallbackSkillCall(ctx context.Context, userInput string) (string, bool, error) {
	hits := r.Registry.Search(userInput, 3)
	_ = r.Store.WriteJSON("fallback/skill-search.json", hits)
	if len(hits) == 0 || hits[0].Score < r.Config.Runtime.FallbackMinScore {
		return "", false, nil
	}
	out, err := r.ExecuteSkill(ctx, hits[0].ID, userInput, map[string]any{
		"task":             userInput,
		"fallback_reason":  "selected highest scoring skill",
		"search_reasoning": hits[0].Reason,
	})
	return out, true, err
}

func (r *Runtime) handleToolCall(ctx context.Context, originalUserInput string, tc model.ToolCall, state runState) (toolCallOutcome, error) {
	_ = r.Store.WriteJSON(fmt.Sprintf("tools/%s-%s-call.json", tc.Function.Name, tc.ID), tc)

	switch {
	case tc.Function.Name == "tool_search":
		var args struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return toolCallOutcome{}, err
		}
		if strings.TrimSpace(args.Query) == "" {
			args.Query = originalUserInput
		}
		hits := r.Registry.Search(args.Query, args.Limit)
		queryInfo := skill.ExplainQuery(args.Query)
		activation := r.activateSkills(state, queryInfo, hits)
		payload := map[string]any{
			"query":                    queryInfo,
			"hits":                     hits,
			"tool_references":          r.buildToolReferences(activation.WindowSkills),
			"retained_tool_references": r.buildToolReferences(activation.RetainedSkills),
			"activation":               activation,
			"note":                     "tool_references are this runtime's Go-style equivalent of novelcode deferred-tool activation. Retained tool references stay callable on later rounds.",
		}
		_ = r.Store.WriteJSON(fmt.Sprintf("tools/%s-%s-result.json", tc.Function.Name, tc.ID), payload)
		return toolCallOutcome{
			Content:           model.MustJSON(payload),
			SearchPerformed:   true,
			SearchFingerprint: activation.QueryFingerprint,
			RetainedSkills:    activation.RetainedSkills,
		}, nil

	case tc.Function.Name == "skill_call":
		if r.Config.Runtime.ForceToolSearchFirst && !state.SearchPerformed {
			return toolCallOutcome{}, fmt.Errorf("skill_call is blocked until tool_search runs at least once")
		}
		var args struct {
			SkillID   string         `json:"skill_id"`
			Task      string         `json:"task"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return toolCallOutcome{}, err
		}
		callArgs := cloneInvocationArgs(args.Arguments)
		if strings.TrimSpace(args.Task) != "" {
			callArgs["task"] = args.Task
		}
		if strings.TrimSpace(deriveSkillTask(callArgs, "")) == "" {
			callArgs["task"] = originalUserInput
		}
		out, err := r.ExecuteSkill(ctx, args.SkillID, originalUserInput, callArgs)
		if err != nil {
			return toolCallOutcome{}, err
		}
		payload := map[string]any{"skill_id": args.SkillID, "arguments": callArgs, "output": out, "run_dir": r.Store.Root}
		_ = r.Store.WriteJSON(fmt.Sprintf("tools/%s-%s-result.json", tc.Function.Name, tc.ID), payload)
		return toolCallOutcome{
			Content:   model.MustJSON(payload),
			FinalText: normalizeSkillOutput(out),
		}, nil

	default:
		if skillID, ok := activeSkillIDForToolName(tc.Function.Name, state.RetainedSkills); ok {
			callArgs, err := parseInvocationArgs(tc.Function.Arguments)
			if err != nil {
				return toolCallOutcome{}, err
			}
			if strings.TrimSpace(deriveSkillTask(callArgs, "")) == "" {
				callArgs["task"] = originalUserInput
			}
			out, err := r.ExecuteSkill(ctx, skillID, originalUserInput, callArgs)
			if err != nil {
				return toolCallOutcome{}, err
			}
			payload := map[string]any{
				"skill_id":  skillID,
				"tool_name": tc.Function.Name,
				"arguments": callArgs,
				"output":    out,
				"run_dir":   r.Store.Root,
			}
			_ = r.Store.WriteJSON(fmt.Sprintf("tools/%s-%s-result.json", tc.Function.Name, tc.ID), payload)
			return toolCallOutcome{
				Content:   model.MustJSON(payload),
				FinalText: normalizeSkillOutput(out),
			}, nil
		}
		return toolCallOutcome{}, fmt.Errorf("unknown tool: %s", tc.Function.Name)
	}
}

func (r *Runtime) ExecuteSkill(ctx context.Context, skillID, originalUserInput string, invocationArgs map[string]any) (string, error) {
	result, err := r.ExecuteSkillInteractive(ctx, skillID, originalUserInput, invocationArgs)
	if err != nil {
		return "", err
	}
	if result.Status == SkillRunStatusNeedsInput && result.AskHuman != nil {
		return renderAskHumanAsMarkdown(*result.AskHuman), nil
	}
	return result.Text, nil
}

func (r *Runtime) ExecuteSkillInteractive(ctx context.Context, skillID, originalUserInput string, invocationArgs map[string]any) (SkillRunResult, error) {
	cmd, err := r.Registry.LoadInvocationCommand(skillID)
	if err != nil {
		return SkillRunResult{}, err
	}
	safeID := strings.ReplaceAll(skillID, "/", "_")
	fileTools := newSkillFileToolSession(RuntimeConfigView{
		WorkspaceRoot:     r.Config.Runtime.WorkspaceRoot,
		DocumentOutputDir: r.Config.Runtime.DocumentOutputDir,
		ProjectID:         r.ProjectID,
		ProjectDocs:       r.ProjectDocs,
	}, r.Store, filepath.ToSlash(filepath.Join("skill-calls", safeID)))
	compiled := ComposeSkillPrompt(cmd, originalUserInput, invocationArgs, r.skillContextHint(cmd, fileTools))
	_ = r.Store.WriteText(fmt.Sprintf("skill-calls/%s/compiled-prompt.md", safeID), compiled)
	_ = r.Store.WriteJSON(fmt.Sprintf("skill-calls/%s/skill-metadata.json", safeID), cmd)
	conversation := []model.Message{{Role: "user", Content: compiled}}
	toolSpecs := fileTools.toolSpecs(cmd.AllowedTools)
	_ = r.Store.WriteJSON(fmt.Sprintf("skill-calls/%s/allowed-tools.json", safeID), toolSpecs)

	state := SkillExecutionState{
		SkillID:           skillID,
		SafeID:            safeID,
		OriginalUserInput: originalUserInput,
		InvocationArgs:    cloneInvocationArgs(invocationArgs),
		CompiledPrompt:    compiled,
		Conversation:      conversation,
		NextRound:         1,
	}
	return r.runSkillInteractiveLoop(ctx, cmd, fileTools, toolSpecs, state)
}

func (r *Runtime) ContinueSkillInteractive(ctx context.Context, state SkillExecutionState, answer AskHumanAnswer) (SkillRunResult, error) {
	if strings.TrimSpace(state.SkillID) == "" {
		return SkillRunResult{}, fmt.Errorf("skill session state is missing skill_id")
	}
	cmd, err := r.Registry.LoadInvocationCommand(state.SkillID)
	if err != nil {
		return SkillRunResult{}, err
	}
	safeID := state.SafeID
	if strings.TrimSpace(safeID) == "" {
		safeID = strings.ReplaceAll(state.SkillID, "/", "_")
		state.SafeID = safeID
	}
	fileTools := newSkillFileToolSession(RuntimeConfigView{
		WorkspaceRoot:     r.Config.Runtime.WorkspaceRoot,
		DocumentOutputDir: r.Config.Runtime.DocumentOutputDir,
		ProjectID:         r.ProjectID,
		ProjectDocs:       r.ProjectDocs,
	}, r.Store, filepath.ToSlash(filepath.Join("skill-calls", safeID)))
	if state.ReadState != nil {
		fileTools.ReadState = make(map[string]skillFileReadState, len(state.ReadState))
		for k, v := range state.ReadState {
			fileTools.ReadState[k] = v
		}
	}
	if strings.TrimSpace(state.PendingToolCallID) == "" {
		return SkillRunResult{}, fmt.Errorf("skill session has no pending AskHuman tool call")
	}
	state.Conversation = append(state.Conversation, model.Message{
		Role:       "tool",
		ToolCallID: state.PendingToolCallID,
		Content:    formatAskHumanToolResult(answer),
	})
	state.PendingToolCallID = ""
	if state.NextRound <= 0 {
		state.NextRound = 1
	}
	toolSpecs := fileTools.toolSpecs(cmd.AllowedTools)
	return r.runSkillInteractiveLoop(ctx, cmd, fileTools, toolSpecs, state)
}

func (r *Runtime) runSkillInteractiveLoop(ctx context.Context, cmd skill.Command, fileTools *skillFileToolSession, toolSpecs []model.ToolSpec, state SkillExecutionState) (SkillRunResult, error) {
	var lastText string
	safeID := strings.TrimSpace(state.SafeID)
	if safeID == "" {
		safeID = strings.ReplaceAll(state.SkillID, "/", "_")
		state.SafeID = safeID
	}
	maxRounds := maxInt(r.Config.Runtime.MaxSkillToolRounds, 1)
	if state.NextRound <= 0 {
		state.NextRound = 1
	}
	for round := state.NextRound; round <= maxRounds; round++ {
		assembly := r.buildSkillAssembly(cmd, state.CompiledPrompt, round, state.Conversation, toolSpecs)
		_ = r.Store.WriteJSON(fmt.Sprintf("skill-calls/%s/round-%02d-assembly.json", safeID, round), assembly)
		_ = r.Store.WriteText(fmt.Sprintf("skill-calls/%s/round-%02d-assembly.md", safeID, round), renderSkillAssemblyMarkdown(assembly))
		chatReq := assembly.ChatRequest
		r.writeChatDebug(fmt.Sprintf("skill-calls/%s/round-%02d-chat-request.json", safeID, round), chatReq)
		resp, err := r.Model.Chat(ctx, chatReq)
		if err != nil {
			r.writeChatError(fmt.Sprintf("skill-calls/%s/round-%02d-chat-error.txt", safeID, round), err)
			return SkillRunResult{}, err
		}
		_ = r.Store.WriteJSON(fmt.Sprintf("skill-calls/%s/round-%02d-response.json", safeID, round), json.RawMessage(resp.Raw))
		_ = r.Store.WriteJSON(fmt.Sprintf("skill-calls/%s/round-%02d-response-analysis.json", safeID, round), analyzeChatResponse(resp))
		_ = r.Store.WriteJSON(fmt.Sprintf("skill-calls/%s/model-raw.json", safeID), json.RawMessage(resp.Raw))
		if len(resp.Choices) == 0 {
			return SkillRunResult{}, fmt.Errorf("skill model returned no choices")
		}
		msg := resp.Choices[0].Message
		state.Conversation = append(state.Conversation, msg)
		lastText = msg.Content
		if len(msg.ToolCalls) == 0 {
			_ = r.Store.WriteText(fmt.Sprintf("skill-calls/%s/output.md", safeID), lastText)
			return SkillRunResult{Status: SkillRunStatusCompleted, Text: lastText, RunDir: r.Store.Root, RunID: r.Store.RunID}, nil
		}
		for _, tc := range msg.ToolCalls {
			content, err := fileTools.handleToolCallWithContext(ctx, tc)
			var pause *AskHumanPause
			if err != nil && asAskHumanPause(err, &pause) {
				state.NextRound = round + 1
				state.PendingToolCallID = pause.ToolCallID
				state.ReadState = copyReadState(fileTools.ReadState)
				return SkillRunResult{
					Status:   SkillRunStatusNeedsInput,
					Text:     msg.Content,
					AskHuman: &pause.Request,
					State:    &state,
					RunDir:   r.Store.Root,
					RunID:    r.Store.RunID,
				}, nil
			}
			if err != nil {
				content = `{"error":` + jsonQuote(err.Error()) + `}`
			}
			state.Conversation = append(state.Conversation, model.Message{Role: "tool", ToolCallID: tc.ID, Content: content})
		}
	}
	_ = r.Store.WriteText(fmt.Sprintf("skill-calls/%s/output.md", safeID), lastText)
	return SkillRunResult{Status: SkillRunStatusCompleted, Text: lastText, RunDir: r.Store.Root, RunID: r.Store.RunID}, nil
}

func asAskHumanPause(err error, target **AskHumanPause) bool {
	var pause *AskHumanPause
	if errors.As(err, &pause) {
		*target = pause
		return true
	}
	return false
}

func copyReadState(in map[string]skillFileReadState) map[string]FileReadState {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]FileReadState, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func formatAskHumanToolResult(answer AskHumanAnswer) string {
	answers := answer.Answers
	if answers == nil {
		answers = map[string]string{}
	}
	payload := map[string]any{
		"type":    "human_answers",
		"answers": answers,
	}
	if strings.TrimSpace(answer.Notes) != "" {
		payload["notes"] = strings.TrimSpace(answer.Notes)
	}
	return model.MustJSON(payload)
}

func renderAskHumanAsMarkdown(request AskHumanRequest) string {
	var b strings.Builder
	b.WriteString("## 需要补充信息\n\n")
	if strings.TrimSpace(request.Reason) != "" {
		b.WriteString(request.Reason)
		b.WriteString("\n\n")
	}
	for i, q := range request.Questions {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, q.Question))
		for _, opt := range q.Options {
			if strings.TrimSpace(opt.Description) != "" {
				b.WriteString(fmt.Sprintf("- %s: %s\n", opt.Label, opt.Description))
			} else {
				b.WriteString(fmt.Sprintf("- %s\n", opt.Label))
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func (r *Runtime) skillContextHint(cmd skill.Command, fileTools *skillFileToolSession) string {
	hint := skillDocumentHint(cmd, fileTools)
	if strings.TrimSpace(r.ProjectContext) != "" {
		_ = r.Store.WriteText("project-context.md", r.ProjectContext)
		if strings.TrimSpace(hint) == "" {
			return r.ProjectContext
		}
		return hint + "\n\n" + r.ProjectContext
	}
	return hint
}

func ComposeSkillPrompt(cmd skill.Command, originalUserInput string, invocationArgs map[string]any, toolingHint string) string {
	task := deriveSkillTask(invocationArgs, originalUserInput)
	var b strings.Builder
	b.WriteString("# Skill Metadata\n\n")
	b.WriteString("skill_id: " + cmd.ID + "\n")
	b.WriteString("skill_name: " + cmd.Name + "\n")
	b.WriteString("skill_root: " + cmd.SkillRoot + "\n")
	if cmd.Description != "" {
		b.WriteString("description: " + cmd.Description + "\n")
	}
	if cmd.WhenToUse != "" {
		b.WriteString("when_to_use: " + cmd.WhenToUse + "\n")
	}
	if cmd.ToolContract != "" {
		b.WriteString("tool_contract: " + cmd.ToolContract + "\n")
	}
	if cmd.ToolOutput != "" {
		b.WriteString("tool_output_contract: " + cmd.ToolOutput + "\n")
	}
	if cmd.ArgumentHint != "" {
		b.WriteString("argument_hint: " + cmd.ArgumentHint + "\n")
	}
	if strings.TrimSpace(toolingHint) != "" {
		b.WriteString("\n# Skill Tooling\n\n")
		b.WriteString(toolingHint)
		b.WriteString("\n")
	}
	b.WriteString("\n# Base Directory\n\n")
	b.WriteString("Base directory for this skill: " + cmd.SkillRoot + "\n\n")
	b.WriteString("# Activated Tool Arguments\n\n")
	if len(invocationArgs) == 0 {
		b.WriteString("{}\n\n")
	} else {
		b.WriteString(prettyJSON(invocationArgs))
		b.WriteString("\n\n")
	}
	b.WriteString("# Skill Instruction\n\n")
	b.WriteString(cmd.MarkdownContent)
	b.WriteString("\n\n# Original User Request\n\n")
	b.WriteString(originalUserInput)
	b.WriteString("\n\n# Current Skill Task\n\n")
	b.WriteString(task)
	b.WriteString("\n\n# Output Rule\n\n")
	b.WriteString("Follow this skill strictly. Respect the skill's own missing-context policy first. If the skill does not define one and the user context is incomplete, produce the best usable version from the provided facts and end with the 3 most valuable missing inputs for the next round.\n")
	return b.String()
}

func (r *Runtime) routerSystemPrompt() string {
	return `You are the Router for Novel Agent Runtime, not the final writer.

Responsibilities:
1. Classify the user's request.
2. If only tool_search is currently exposed, call tool_search first.
3. tool_search returns candidate skills, explicit tool-reference-like activation objects, a fresh activation window, and an updated retained skill pool for later rounds.
4. Prefer calling those retained skill tools directly because they expose the real skill-specific input schema. skill_call is only a compatibility shim.
5. Do not write the final skill output yourself unless tool_search fails to find a suitable skill.

Constraints:
- tool_search returns activation references and retained callable tool names, not the full SKILL.md.
- Once a skill tool is retained, it stays callable on later rounds until Runtime evicts it from the retained pool.`
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func (r *Runtime) writeChatDebug(rel string, req model.ChatRequest) {
	if !r.Debug {
		return
	}
	payload := map[string]any{
		"url":                  strings.TrimRight(r.Model.BaseURL, "/") + "/chat/completions",
		"http_timeout_seconds": r.ModelConfig.TimeoutSeconds,
		"headers": map[string]string{
			"Content-Type":  "application/json",
			"Authorization": maskAuthorization(r.Model.APIKey),
		},
		"request": req,
	}
	_ = r.Store.WriteJSON(rel, payload)
}

func (r *Runtime) writeChatError(rel string, err error) {
	if !r.Debug || err == nil {
		return
	}
	_ = r.Store.WriteText(rel, err.Error()+"\n")
}

func maskAuthorization(apiKey string) string {
	switch strings.TrimSpace(apiKey) {
	case "":
		return ""
	case "EMPTY":
		return "Bearer EMPTY"
	default:
		return "Bearer ***"
	}
}

func (r *Runtime) effectiveSkillMaxTokens() int {
	maxTokens := r.ModelConfig.MaxOutput
	if strings.Contains(strings.ToLower(r.ModelConfig.BaseURL), "deepseek.com") && maxTokens > 393216 {
		return 393216
	}
	return maxTokens
}

func normalizeSkillOutput(out string) string {
	out = strings.ReplaceAll(out, "\r\n", "\n")
	out = strings.TrimSpace(out)
	if section := extractMarkdownSection(out, "## 开篇正文"); section != "" {
		return strings.TrimSpace(section)
	}
	out = trimPresentationHeading(out)
	return out
}

func extractMarkdownSection(text, headingPrefix string) string {
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), headingPrefix) {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return ""
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if i > start && strings.HasPrefix(trimmed, "## ") {
			end = i
			break
		}
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

func trimPresentationHeading(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return text
	}
	first := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(first, "#") {
		return text
	}
	if !looksLikePresentationHeading(first) {
		return text
	}
	start := 1
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	trimmed := strings.TrimSpace(strings.Join(lines[start:], "\n"))
	if trimmed == "" {
		return text
	}
	return trimmed
}

func looksLikePresentationHeading(line string) bool {
	line = strings.ToLower(strings.TrimSpace(strings.TrimLeft(line, "#")))
	for _, marker := range []string{
		"正文输出",
		"开篇",
		"skill",
		"狙击手",
	} {
		if strings.Contains(line, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func parseInvocationArgs(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	return cloneInvocationArgs(args), nil
}

func cloneInvocationArgs(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	return skill.CloneMap(src)
}

func deriveSkillTask(args map[string]any, fallback string) string {
	if len(args) > 0 {
		for _, key := range []string{"task", "prompt", "request"} {
			if value, ok := args[key]; ok {
				if s := strings.TrimSpace(fmt.Sprint(value)); s != "" {
					return s
				}
			}
		}
	}
	return strings.TrimSpace(fallback)
}

func prettyJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}
