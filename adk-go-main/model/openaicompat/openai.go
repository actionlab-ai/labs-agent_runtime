// Copyright 2026 Archinfra / yuanyp8
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

// Package openaicompat implements ADK's model.LLM interface for OpenAI-compatible
// Chat Completions endpoints.
//
// 业务说明：
//
//	ADK Go 的核心运行时只依赖 model.LLM 接口，不强绑定 Gemini。
//	这个适配器把 ADK 内部的 genai.Content / FunctionDeclaration 转换成
//	OpenAI Chat Completions 的 messages / tools，再把模型返回的文本或
//	tool_calls 转回 ADK 的 genai.Content。这样自建 OpenAI 格式模型可以
//	参与 Runner、Session、Tool Calling、REST/SSE 等完整链路。
//
// 设计原则：
//
//  1. ADK 内部继续使用 genai 作为统一中间协议。
//  2. OpenAI-compatible 只存在于模型适配器边界。
//  3. 不静默丢弃不支持的 Part 类型；遇到图片、文件等暂未支持内容时显式报错。
//  4. 错误保留错误链，方便上层使用 errors.Is / errors.As 判断。
package openaicompat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/genai"

	"google.golang.org/adk/model"
)

const (
	baseURLEnv = "OPENAI_COMPAT_BASE_URL"
	apiKeyEnv  = "OPENAI_COMPAT_API_KEY"

	defaultTimeout   = 180 * time.Second
	defaultUserAgent = "archinfra-adk/openai-compatible"
)

var (
	// ErrStreamDecode 表示 OpenAI-compatible SSE 流式响应解析失败。
	ErrStreamDecode = errors.New("failed to decode openai-compatible stream")

	// ErrEndpointStatus 表示 HTTP endpoint 返回了非 2xx 状态码。
	//
	// 可通过 errors.Is(err, ErrEndpointStatus) 判断。
	ErrEndpointStatus = errors.New("openai-compatible endpoint returned non-2xx status")

	// ErrAPIResponse 表示 OpenAI-compatible endpoint 返回了 200，但 JSON body 中带 error 字段。
	//
	// 可通过 errors.Is(err, ErrAPIResponse) 判断。
	ErrAPIResponse = errors.New("openai-compatible endpoint returned api error")

	// ErrUnsupportedPart 表示 ADK genai.Part 中存在当前适配器还不能转换成 OpenAI messages 的内容。
	//
	// 常见场景：InlineData、FileData、ExecutableCode 等多模态/代码执行类型。
	ErrUnsupportedPart = errors.New("unsupported genai part for openai-compatible adapter")

	// ErrEmptyResponse 表示 endpoint 返回结构不完整，例如 choices 为空。
	ErrEmptyResponse = errors.New("empty openai-compatible response")
)

// Config 是 OpenAI-compatible 模型适配器配置。
//
// 业务注释：
//   - BaseURL 填到 /v1 级别，例如：http://36.147.35.14:30080/v1
//   - ModelName 填模型 id，例如：qwen_qwen3.5-397b-a17b
//   - APIKey 如果网关不校验，可以填 EMPTY；如果校验，就填真实 key。
//   - HTTPClient 可用于测试、代理、自定义超时、连接池等场景。
//   - Headers 用于透传额外网关头，但不要把 API key 写进普通日志。
type Config struct {
	BaseURL    string
	APIKey     string
	ModelName  string
	HTTPClient *http.Client
	Headers    http.Header
	UserAgent  string
	Logger     *slog.Logger
	Debug      bool
}

// Option 用于可选配置。
type Option func(*Config)

// WithBaseURL 设置 OpenAI-compatible base url。
func WithBaseURL(baseURL string) Option {
	return func(c *Config) { c.BaseURL = baseURL }
}

// WithAPIKey 设置 Bearer token。
func WithAPIKey(apiKey string) Option {
	return func(c *Config) { c.APIKey = apiKey }
}

// WithHTTPClient 设置自定义 HTTP client，测试或代理场景可用。
func WithHTTPClient(client *http.Client) Option {
	return func(c *Config) { c.HTTPClient = client }
}

// WithHeaders 设置额外请求头。
func WithHeaders(headers http.Header) Option {
	return func(c *Config) { c.Headers = cloneHeader(headers) }
}

// WithUserAgent 设置请求 User-Agent，便于服务端网关识别调用来源。
func WithUserAgent(userAgent string) Option {
	return func(c *Config) { c.UserAgent = userAgent }
}

// WithLogger 设置日志器。
//
// 说明：默认不强制输出日志。只有启用 WithDebug(true) 后，才会输出请求 URL、模型名、状态码、耗时等元信息。
func WithLogger(logger *slog.Logger) Option {
	return func(c *Config) { c.Logger = logger }
}

// WithDebug 启用调试日志。
//
// 注意：调试日志默认不打印 API key。响应错误体会进入错误信息，生产环境如包含敏感信息可在上层脱敏。
func WithDebug(debug bool) Option {
	return func(c *Config) { c.Debug = debug }
}

type openAICompatModel struct {
	baseURL   string
	apiKey    string
	name      string
	client    *http.Client
	headers   http.Header
	userAgent string
	logger    *slog.Logger
	debug     bool
}

var _ model.LLM = (*openAICompatModel)(nil)

// NewModel 创建一个 ADK model.LLM。
//
// 业务注释：
//
//	ADK 的 llmagent.Config.Model 只要求实现 model.LLM 接口，所以这里无需改
//	Runner、Session、Flow、Tool 的任何代码。模型适配层就是“外部模型协议”和
//	“ADK 内部事件协议”的边界。
func NewModel(ctx context.Context, modelName string, opts ...Option) (model.LLM, error) {
	_ = ctx

	cfg := &Config{
		BaseURL:    os.Getenv(baseURLEnv),
		APIKey:     os.Getenv(apiKeyEnv),
		ModelName:  modelName,
		HTTPClient: &http.Client{Timeout: defaultTimeout},
		UserAgent:  defaultUserAgent,
		Logger:     slog.Default(),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if strings.TrimSpace(cfg.ModelName) == "" {
		return nil, fmt.Errorf("model name is required")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("%s is required", baseURLEnv)
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: defaultTimeout}
	}
	if strings.TrimSpace(cfg.UserAgent) == "" {
		cfg.UserAgent = defaultUserAgent
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &openAICompatModel{
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:    cfg.APIKey,
		name:      cfg.ModelName,
		client:    cfg.HTTPClient,
		headers:   cloneHeader(cfg.Headers),
		userAgent: cfg.UserAgent,
		logger:    cfg.Logger,
		debug:     cfg.Debug,
	}, nil
}

func (m *openAICompatModel) Name() string { return m.name }

// GenerateContent 实现 ADK model.LLM。
//
// 功能注释：
//
//	ADK Flow 会把用户消息、历史事件、系统指令、工具声明全部整理进 LLMRequest。
//	这里做三件事：
//	  1. ADK LLMRequest -> OpenAI ChatCompletionRequest
//	  2. POST /chat/completions
//	  3. OpenAI ChatCompletionResponse -> ADK LLMResponse
//
// 当前策略：
//
//	stream=false 时走普通 JSON 响应。
//	stream=true 时走 OpenAI SSE：逐个解析 data: {...} delta，向 ADK yield Partial 文本事件，
//	并在 [DONE] 或流结束时 yield 一个非 Partial 的聚合响应，供 Runner 写入 Session。
func (m *openAICompatModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		openAIReq, err := buildChatCompletionRequest(m.modelName(req), req)
		if err != nil {
			yield(nil, err)
			return
		}

		if stream {
			openAIReq.Stream = true
			for llmResp, err := range m.doChatCompletionStream(ctx, openAIReq) {
				if !yield(llmResp, err) {
					return
				}
			}
			return
		}

		resp, err := m.doChatCompletion(ctx, openAIReq)
		if err != nil {
			yield(nil, err)
			return
		}

		llmResp, err := toADKResponse(resp)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(llmResp, nil)
	}
}

func (m *openAICompatModel) modelName(req *model.LLMRequest) string {
	if req != nil && strings.TrimSpace(req.Model) != "" {
		return req.Model
	}
	return m.name
}

func (m *openAICompatModel) doChatCompletion(ctx context.Context, body chatCompletionRequest) (*chatCompletionResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal openai-compatible request: %w", err)
	}

	url := m.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create openai-compatible request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", m.userAgent)
	if strings.TrimSpace(m.apiKey) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)
	}
	for k, values := range m.headers {
		for _, v := range values {
			httpReq.Header.Add(k, v)
		}
	}

	start := time.Now()
	if m.debug {
		m.logger.Info("openai-compatible request start", "url", url, "model", body.Model, "messages", len(body.Messages), "tools", len(body.Tools))
	}

	httpResp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call openai-compatible endpoint: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read openai-compatible response body: %w", err)
	}

	duration := time.Since(start)
	if m.debug {
		m.logger.Info("openai-compatible request finished", "url", url, "model", body.Model, "status", httpResp.Status, "duration", duration.String())
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, &HTTPStatusError{
			StatusCode: httpResp.StatusCode,
			Status:     httpResp.Status,
			Body:       string(respBytes),
			URL:        url,
		}
	}

	var out chatCompletionResponse
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("decode openai-compatible response: %w; raw=%s", err, truncateForError(respBytes, 4096))
	}
	if out.Error != nil {
		return nil, &APIError{
			Message: out.Error.Message,
			Type:    out.Error.Type,
			Code:    out.Error.Code,
		}
	}
	return &out, nil
}

func (m *openAICompatModel) doChatCompletionStream(ctx context.Context, body chatCompletionRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		body.Stream = true
		payload, err := json.Marshal(body)
		if err != nil {
			yield(nil, fmt.Errorf("marshal openai-compatible stream request: %w", err))
			return
		}

		url := m.baseURL + "/chat/completions"
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			yield(nil, fmt.Errorf("create openai-compatible stream request: %w", err))
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("Cache-Control", "no-cache")
		httpReq.Header.Set("User-Agent", m.userAgent)
		if strings.TrimSpace(m.apiKey) != "" {
			httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)
		}
		for k, values := range m.headers {
			for _, v := range values {
				httpReq.Header.Add(k, v)
			}
		}

		start := time.Now()
		if m.debug {
			m.logger.Info("openai-compatible stream request start", "url", url, "model", body.Model, "messages", len(body.Messages), "tools", len(body.Tools))
		}

		httpResp, err := m.client.Do(httpReq)
		if err != nil {
			yield(nil, fmt.Errorf("call openai-compatible stream endpoint: %w", err))
			return
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
			respBytes, readErr := io.ReadAll(httpResp.Body)
			if readErr != nil {
				yield(nil, fmt.Errorf("read openai-compatible stream error body: %w", readErr))
				return
			}
			yield(nil, &HTTPStatusError{StatusCode: httpResp.StatusCode, Status: httpResp.Status, Body: string(respBytes), URL: url})
			return
		}

		acc := newStreamAccumulator()
		scanner := bufio.NewScanner(httpResp.Body)
		// OpenAI-compatible 网关有时会把较长 tool arguments 放进单行 data 中。
		// Scanner 默认 token 上限 64K，这里放大到 10MB，避免大参数被截断。
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" {
				continue
			}
			if data == "[DONE]" {
				break
			}

			var chunk chatCompletionStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				yield(nil, fmt.Errorf("%w: %v; raw=%s", ErrStreamDecode, err, truncateStringForError(data, 4096)))
				return
			}
			if chunk.Error != nil {
				yield(nil, &APIError{Message: chunk.Error.Message, Type: chunk.Error.Type, Code: chunk.Error.Code})
				return
			}

			partials, err := acc.AddChunk(&chunk)
			if err != nil {
				yield(nil, err)
				return
			}
			for _, partial := range partials {
				if !yield(partial, nil) {
					return
				}
			}
		}
		if err := scanner.Err(); err != nil {
			yield(nil, fmt.Errorf("read openai-compatible stream: %w", err))
			return
		}

		finalResp, err := acc.FinalResponse()
		if err != nil {
			yield(nil, err)
			return
		}
		if finalResp != nil {
			if m.debug {
				m.logger.Info("openai-compatible stream request finished", "url", url, "model", body.Model, "duration", time.Since(start).String(), "finish_reason", finalResp.FinishReason)
			}
			yield(finalResp, nil)
		}
	}
}

// HTTPStatusError 表示 endpoint 返回非 2xx 状态。
//
// 上层可以：
//
//	if errors.Is(err, openaicompat.ErrEndpointStatus) { ... }
//	var statusErr *openaicompat.HTTPStatusError
//	if errors.As(err, &statusErr) { fmt.Println(statusErr.StatusCode, statusErr.Body) }
type HTTPStatusError struct {
	StatusCode int
	Status     string
	Body       string
	URL        string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("%v: %s url=%s body=%s", ErrEndpointStatus, e.Status, e.URL, truncateStringForError(e.Body, 4096))
}

func (e *HTTPStatusError) Unwrap() error { return ErrEndpointStatus }

// APIError 表示 endpoint 返回 200，但 JSON body 内含 OpenAI error 字段。
type APIError struct {
	Message string
	Type    string
	Code    any
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%v: type=%s code=%v message=%s", ErrAPIResponse, e.Type, e.Code, e.Message)
}

func (e *APIError) Unwrap() error { return ErrAPIResponse }

// ---- OpenAI Chat Completions request/response DTOs ----

type chatCompletionRequest struct {
	Model      string        `json:"model"`
	Messages   []chatMessage `json:"messages"`
	Tools      []chatTool    `json:"tools,omitempty"`
	ToolChoice any           `json:"tool_choice,omitempty"`
	Stream     bool          `json:"stream"`
}

type chatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type chatTool struct {
	Type     string       `json:"type"`
	Function chatFunction `json:"function"`
}

type chatFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type toolCall struct {
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type"`
	Function toolCallFunction `json:"function"`
}

type toolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatCompletionResponse struct {
	ID      string                 `json:"id"`
	Model   string                 `json:"model"`
	Choices []chatCompletionChoice `json:"choices"`
	Usage   *chatCompletionUsage   `json:"usage,omitempty"`
	Error   *chatCompletionError   `json:"error,omitempty"`
}

type chatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

type chatCompletionError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"`
}

type chatCompletionStreamChunk struct {
	ID      string                       `json:"id"`
	Model   string                       `json:"model"`
	Choices []chatCompletionStreamChoice `json:"choices"`
	Usage   *chatCompletionUsage         `json:"usage,omitempty"`
	Error   *chatCompletionError         `json:"error,omitempty"`
}

type chatCompletionStreamChoice struct {
	Index        int              `json:"index"`
	Delta        chatMessageDelta `json:"delta"`
	FinishReason string           `json:"finish_reason"`
}

type chatMessageDelta struct {
	Role      string           `json:"role,omitempty"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []streamToolCall `json:"tool_calls,omitempty"`
}

type streamToolCall struct {
	Index    int                    `json:"index"`
	ID       string                 `json:"id,omitempty"`
	Type     string                 `json:"type,omitempty"`
	Function streamToolCallFunction `json:"function,omitempty"`
}

type streamToolCallFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type streamToolCallAccumulator struct {
	ID        string
	Type      string
	Name      string
	Arguments strings.Builder
}

type streamAccumulator struct {
	id           string
	modelVersion string
	text         strings.Builder
	toolCalls    map[int]*streamToolCallAccumulator
	toolOrder    []int
	finishReason string
	usage        *chatCompletionUsage
	seenChunk    bool
}

func newStreamAccumulator() *streamAccumulator {
	return &streamAccumulator{toolCalls: map[int]*streamToolCallAccumulator{}}
}

func (a *streamAccumulator) AddChunk(chunk *chatCompletionStreamChunk) ([]*model.LLMResponse, error) {
	if chunk == nil {
		return nil, nil
	}
	a.seenChunk = true
	if chunk.ID != "" {
		a.id = chunk.ID
	}
	if chunk.Model != "" {
		a.modelVersion = chunk.Model
	}
	if chunk.Usage != nil {
		a.usage = chunk.Usage
	}

	var out []*model.LLMResponse
	for _, choice := range chunk.Choices {
		if choice.FinishReason != "" {
			a.finishReason = choice.FinishReason
		}
		if choice.Delta.Content != "" {
			a.text.WriteString(choice.Delta.Content)
			out = append(out, &model.LLMResponse{
				Content:      &genai.Content{Role: genai.RoleModel, Parts: []*genai.Part{genai.NewPartFromText(choice.Delta.Content)}},
				ModelVersion: a.modelVersion,
				Partial:      true,
			})
		}
		for _, tc := range choice.Delta.ToolCalls {
			entry, ok := a.toolCalls[tc.Index]
			if !ok {
				entry = &streamToolCallAccumulator{}
				a.toolCalls[tc.Index] = entry
				a.toolOrder = append(a.toolOrder, tc.Index)
			}
			if tc.ID != "" {
				entry.ID = tc.ID
			}
			if tc.Type != "" {
				entry.Type = tc.Type
			}
			if tc.Function.Name != "" {
				entry.Name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				entry.Arguments.WriteString(tc.Function.Arguments)
			}
		}
	}
	return out, nil
}

func (a *streamAccumulator) FinalResponse() (*model.LLMResponse, error) {
	if !a.seenChunk {
		return nil, fmt.Errorf("%w: empty streaming response", ErrEmptyResponse)
	}

	parts := make([]*genai.Part, 0, 1+len(a.toolOrder))
	if a.text.Len() > 0 {
		parts = append(parts, genai.NewPartFromText(a.text.String()))
	}
	for _, idx := range a.toolOrder {
		entry := a.toolCalls[idx]
		if entry == nil {
			continue
		}
		args := map[string]any{}
		argText := strings.TrimSpace(entry.Arguments.String())
		if argText != "" {
			if err := json.Unmarshal([]byte(argText), &args); err != nil {
				args = map[string]any{"_raw_arguments": argText}
			}
		}
		id := entry.ID
		if id == "" {
			id = "call_" + entry.Name
		}
		parts = append(parts, &genai.Part{FunctionCall: &genai.FunctionCall{ID: id, Name: entry.Name, Args: args}})
	}
	if len(parts) == 0 {
		parts = append(parts, genai.NewPartFromText(""))
	}

	finishReason := a.finishReason
	if finishReason == "" && len(a.toolOrder) > 0 {
		finishReason = "tool_calls"
	}
	if finishReason == "" {
		finishReason = "stop"
	}

	customMetadata := map[string]any{}
	if a.id != "" {
		customMetadata["openai_id"] = a.id
	}
	if a.usage != nil {
		customMetadata["openai_usage"] = map[string]any{
			"prompt_tokens":     a.usage.PromptTokens,
			"completion_tokens": a.usage.CompletionTokens,
			"total_tokens":      a.usage.TotalTokens,
		}
	}
	if len(customMetadata) == 0 {
		customMetadata = nil
	}

	return &model.LLMResponse{
		Content:        &genai.Content{Role: genai.RoleModel, Parts: parts},
		ModelVersion:   a.modelVersion,
		FinishReason:   genai.FinishReason(finishReason),
		TurnComplete:   true,
		CustomMetadata: customMetadata,
	}, nil
}

func buildChatCompletionRequest(modelName string, req *model.LLMRequest) (chatCompletionRequest, error) {
	if req == nil {
		return chatCompletionRequest{}, fmt.Errorf("nil LLMRequest")
	}

	messages := make([]chatMessage, 0, len(req.Contents)+1)
	if req.Config != nil && req.Config.SystemInstruction != nil {
		text, err := contentTextStrict(req.Config.SystemInstruction)
		if err != nil {
			return chatCompletionRequest{}, fmt.Errorf("convert system instruction: %w", err)
		}
		if strings.TrimSpace(text) != "" {
			messages = append(messages, chatMessage{Role: "system", Content: text})
		}
	}

	for i, c := range req.Contents {
		converted, err := messagesFromContent(c)
		if err != nil {
			return chatCompletionRequest{}, fmt.Errorf("convert content[%d]: %w", i, err)
		}
		messages = append(messages, converted...)
	}
	messages = maybeAppendContinuationPrompt(messages)

	tools, err := toolsFromConfig(req.Config)
	if err != nil {
		return chatCompletionRequest{}, err
	}

	out := chatCompletionRequest{
		Model:    modelName,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
	}
	if len(tools) > 0 {
		out.ToolChoice = "auto"
	}
	return out, nil
}

func maybeAppendContinuationPrompt(messages []chatMessage) []chatMessage {
	if len(messages) == 0 {
		return append(messages, chatMessage{Role: "user", Content: "Continue."})
	}

	last := messages[len(messages)-1]
	switch last.Role {
	case "system":
		return append(messages, chatMessage{Role: "user", Content: "Continue."})
	case "assistant":
		// 如果最后一条是 assistant 文本，模型下一步通常需要一个用户侧继续指令。
		// 但如果最后一条 assistant 带 tool_calls，后面应该接 tool message，不在这里补 user。
		if len(last.ToolCalls) == 0 {
			return append(messages, chatMessage{Role: "user", Content: "Continue."})
		}
	}
	return messages
}

func messagesFromContent(c *genai.Content) ([]chatMessage, error) {
	if c == nil || len(c.Parts) == 0 {
		return nil, nil
	}

	// FunctionResponse 在 OpenAI 协议里必须作为 role=tool 的独立消息。
	var out []chatMessage
	var textParts []string
	var toolCalls []toolCall

	for i, part := range c.Parts {
		if part == nil {
			continue
		}

		switch {
		case part.Text != "":
			textParts = append(textParts, part.Text)

		case part.FunctionCall != nil:
			fc := part.FunctionCall
			args, err := marshalJSONObject(emptyMapIfNil(fc.Args))
			if err != nil {
				return nil, fmt.Errorf("marshal function call args for %q: %w", fc.Name, err)
			}
			id := fc.ID
			if id == "" {
				id = "call_" + fc.Name
			}
			toolCalls = append(toolCalls, toolCall{
				ID:   id,
				Type: "function",
				Function: toolCallFunction{
					Name:      fc.Name,
					Arguments: args,
				},
			})

		case part.FunctionResponse != nil:
			fr := part.FunctionResponse
			response, err := marshalJSONObject(emptyMapIfNil(fr.Response))
			if err != nil {
				return nil, fmt.Errorf("marshal function response for %q: %w", fr.Name, err)
			}
			id := fr.ID
			if id == "" {
				id = "call_" + fr.Name
			}
			out = append(out, chatMessage{
				Role:       "tool",
				ToolCallID: id,
				Name:       fr.Name,
				Content:    response,
			})

		case isEmptyPart(part):
			continue

		default:
			return nil, fmt.Errorf("%w: role=%q part_index=%d json=%s", ErrUnsupportedPart, c.Role, i, compactJSONForDebug(part))
		}
	}

	if len(textParts) > 0 || len(toolCalls) > 0 {
		out = append([]chatMessage{{
			Role:      openAIRole(c.Role),
			Content:   strings.Join(textParts, "\n"),
			ToolCalls: toolCalls,
		}}, out...)
	}
	return out, nil
}

func openAIRole(role string) string {
	switch role {
	case "model":
		return "assistant"
	case "user":
		return "user"
	default:
		if role == "" {
			return "user"
		}
		return role
	}
}

func contentTextStrict(c *genai.Content) (string, error) {
	if c == nil {
		return "", nil
	}
	var parts []string
	for i, p := range c.Parts {
		if p == nil {
			continue
		}
		if p.Text != "" {
			parts = append(parts, p.Text)
			continue
		}
		if isEmptyPart(p) {
			continue
		}
		return "", fmt.Errorf("%w: system instruction part_index=%d json=%s", ErrUnsupportedPart, i, compactJSONForDebug(p))
	}
	return strings.Join(parts, "\n"), nil
}

func toolsFromConfig(cfg *genai.GenerateContentConfig) ([]chatTool, error) {
	if cfg == nil {
		return nil, nil
	}
	var tools []chatTool
	for _, t := range cfg.Tools {
		if t == nil {
			continue
		}
		for _, decl := range t.FunctionDeclarations {
			if decl == nil || decl.Name == "" {
				continue
			}

			params := any(decl.ParametersJsonSchema)
			if params == nil {
				params = map[string]any{"type": "object", "properties": map[string]any{}}
			} else {
				cloned, err := cloneJSONValue(params)
				if err != nil {
					return nil, fmt.Errorf("clone json schema for tool %q: %w", decl.Name, err)
				}
				params = cloned
			}

			tools = append(tools, chatTool{
				Type: "function",
				Function: chatFunction{
					Name:        decl.Name,
					Description: decl.Description,
					Parameters:  params,
				},
			})
		}
	}
	return tools, nil
}

func toADKResponse(resp *chatCompletionResponse) (*model.LLMResponse, error) {
	if resp == nil {
		return nil, fmt.Errorf("%w: nil chat completion response", ErrEmptyResponse)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("%w: empty choices in chat completion response", ErrEmptyResponse)
	}
	choice := resp.Choices[0]
	msg := choice.Message

	parts := make([]*genai.Part, 0, 1+len(msg.ToolCalls))
	if msg.Content != "" {
		parts = append(parts, genai.NewPartFromText(msg.Content))
	}
	for _, tc := range msg.ToolCalls {
		args := map[string]any{}
		if strings.TrimSpace(tc.Function.Arguments) != "" {
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				// 国产模型偶尔会返回非严格 JSON。这里不直接失败，而是把原始参数保留下来，
				// 让上层工具或调试日志能看到模型到底返回了什么。
				args = map[string]any{"_raw_arguments": tc.Function.Arguments}
			}
		}
		parts = append(parts, &genai.Part{FunctionCall: &genai.FunctionCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: args,
		}})
	}
	if len(parts) == 0 {
		parts = append(parts, genai.NewPartFromText(""))
	}

	finishReason := choice.FinishReason
	if finishReason == "" && len(msg.ToolCalls) > 0 {
		finishReason = "tool_calls"
	}
	if finishReason == "" {
		finishReason = "stop"
	}

	customMetadata := map[string]any{}
	if resp.ID != "" {
		customMetadata["openai_id"] = resp.ID
	}
	if resp.Usage != nil {
		customMetadata["openai_usage"] = map[string]any{
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		}
	}
	if len(customMetadata) == 0 {
		customMetadata = nil
	}

	return &model.LLMResponse{
		Content: &genai.Content{
			Role:  genai.RoleModel,
			Parts: parts,
		},
		ModelVersion:   resp.Model,
		FinishReason:   genai.FinishReason(finishReason),
		CustomMetadata: customMetadata,
	}, nil
}

func emptyMapIfNil(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

func marshalJSONObject(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func cloneJSONValue(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func isEmptyPart(part *genai.Part) bool {
	if part == nil {
		return true
	}
	b, err := json.Marshal(part)
	if err != nil {
		return false
	}
	s := strings.TrimSpace(string(b))
	return s == "{}" || s == "null" || s == `{"text":""}`
}

func compactJSONForDebug(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("<marshal error: %v>", err)
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		return string(b)
	}
	return truncateStringForError(buf.String(), 1024)
}

func truncateForError(b []byte, limit int) string {
	return truncateStringForError(string(b), limit)
}

func truncateStringForError(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	return s[:limit] + "...(truncated)"
}

func cloneHeader(in http.Header) http.Header {
	if in == nil {
		return nil
	}
	out := make(http.Header, len(in))
	for k, values := range in {
		out[k] = append([]string(nil), values...)
	}
	return out
}
