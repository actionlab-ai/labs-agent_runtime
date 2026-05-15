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

package openaicompat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/genai"

	"google.golang.org/adk/model"
)

func TestBuildChatCompletionRequest_TextAndSystem(t *testing.T) {
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText("你好，解释一下 model 模块", genai.RoleUser),
		},
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText("你是一个中文 Agent 框架老师。", genai.RoleUser),
		},
	}

	got, err := buildChatCompletionRequest("qwen-test", req)
	if err != nil {
		t.Fatalf("buildChatCompletionRequest() error = %v", err)
	}
	if got.Model != "qwen-test" {
		t.Fatalf("model = %q, want qwen-test", got.Model)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2: %#v", len(got.Messages), got.Messages)
	}
	if got.Messages[0].Role != "system" || !strings.Contains(got.Messages[0].Content, "中文 Agent") {
		t.Fatalf("system message mismatch: %#v", got.Messages[0])
	}
	if got.Messages[1].Role != "user" || !strings.Contains(got.Messages[1].Content, "model 模块") {
		t.Fatalf("user message mismatch: %#v", got.Messages[1])
	}
}

func TestBuildChatCompletionRequest_ToolDeclaration(t *testing.T) {
	req := &model.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText("北京天气怎么样？", genai.RoleUser)},
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{{
				FunctionDeclarations: []*genai.FunctionDeclaration{{
					Name:        "get_weather",
					Description: "查询城市天气",
					ParametersJsonSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"city": map[string]any{"type": "string"},
						},
						"required": []any{"city"},
					},
				}},
			}},
		},
	}

	got, err := buildChatCompletionRequest("qwen-test", req)
	if err != nil {
		t.Fatalf("buildChatCompletionRequest() error = %v", err)
	}
	if len(got.Tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(got.Tools))
	}
	if got.Tools[0].Type != "function" || got.Tools[0].Function.Name != "get_weather" {
		t.Fatalf("tool mismatch: %#v", got.Tools[0])
	}
	if got.ToolChoice != "auto" {
		t.Fatalf("tool_choice = %#v, want auto", got.ToolChoice)
	}
}

func TestMessagesFromContent_FunctionCallAndFunctionResponse(t *testing.T) {
	functionCallContent := &genai.Content{
		Role: genai.RoleModel,
		Parts: []*genai.Part{{FunctionCall: &genai.FunctionCall{
			ID:   "call_001",
			Name: "get_weather",
			Args: map[string]any{"city": "Beijing"},
		}}},
	}
	functionResponseContent := &genai.Content{
		Role: genai.RoleUser,
		Parts: []*genai.Part{{FunctionResponse: &genai.FunctionResponse{
			ID:       "call_001",
			Name:     "get_weather",
			Response: map[string]any{"temperature": "23C", "weather": "sunny"},
		}}},
	}

	callMessages, err := messagesFromContent(functionCallContent)
	if err != nil {
		t.Fatalf("messagesFromContent(functionCall) error = %v", err)
	}
	if len(callMessages) != 1 {
		t.Fatalf("function call messages len = %d, want 1", len(callMessages))
	}
	if callMessages[0].Role != "assistant" || len(callMessages[0].ToolCalls) != 1 {
		t.Fatalf("function call message mismatch: %#v", callMessages[0])
	}
	if callMessages[0].ToolCalls[0].ID != "call_001" || callMessages[0].ToolCalls[0].Function.Name != "get_weather" {
		t.Fatalf("tool call mismatch: %#v", callMessages[0].ToolCalls[0])
	}

	responseMessages, err := messagesFromContent(functionResponseContent)
	if err != nil {
		t.Fatalf("messagesFromContent(functionResponse) error = %v", err)
	}
	if len(responseMessages) != 1 {
		t.Fatalf("function response messages len = %d, want 1", len(responseMessages))
	}
	if responseMessages[0].Role != "tool" || responseMessages[0].ToolCallID != "call_001" {
		t.Fatalf("function response message mismatch: %#v", responseMessages[0])
	}
}

func TestToADKResponse_Text(t *testing.T) {
	got, err := toADKResponse(&chatCompletionResponse{
		ID:    "chatcmpl-test",
		Model: "qwen-test",
		Choices: []chatCompletionChoice{
			{Message: chatMessage{Role: "assistant", Content: "模型回答文本"}, FinishReason: "stop"},
		},
		Usage: &chatCompletionUsage{PromptTokens: 3, CompletionTokens: 4, TotalTokens: 7},
	})
	if err != nil {
		t.Fatalf("toADKResponse() error = %v", err)
	}
	if got.ModelVersion != "qwen-test" || got.Content.Role != genai.RoleModel {
		t.Fatalf("response header mismatch: %#v", got)
	}
	if len(got.Content.Parts) != 1 || got.Content.Parts[0].Text != "模型回答文本" {
		t.Fatalf("response content mismatch: %#v", got.Content.Parts)
	}
	if got.CustomMetadata == nil || got.CustomMetadata["openai_id"] != "chatcmpl-test" {
		t.Fatalf("custom metadata missing: %#v", got.CustomMetadata)
	}
}

func TestToADKResponse_ToolCall(t *testing.T) {
	got, err := toADKResponse(&chatCompletionResponse{
		Model: "qwen-test",
		Choices: []chatCompletionChoice{
			{Message: chatMessage{Role: "assistant", ToolCalls: []toolCall{{
				ID:   "call_001",
				Type: "function",
				Function: toolCallFunction{
					Name:      "get_weather",
					Arguments: `{"city":"Beijing"}`,
				},
			}}}, FinishReason: "tool_calls"},
		},
	})
	if err != nil {
		t.Fatalf("toADKResponse() error = %v", err)
	}
	if len(got.Content.Parts) != 1 || got.Content.Parts[0].FunctionCall == nil {
		t.Fatalf("tool call part missing: %#v", got.Content.Parts)
	}
	fc := got.Content.Parts[0].FunctionCall
	if fc.ID != "call_001" || fc.Name != "get_weather" || fc.Args["city"] != "Beijing" {
		t.Fatalf("function call mismatch: %#v", fc)
	}
}

func TestGenerateContent_UsesOpenAICompatibleEndpoint(t *testing.T) {
	var captured chatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q, want Bearer test-key", got)
		}
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Fatalf("User-Agent is empty")
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-test",
			"model":"qwen-test",
			"choices":[{"index":0,"message":{"role":"assistant","content":"ok from mock server"},"finish_reason":"stop"}]
		}`))
	}))
	defer server.Close()

	llm, err := NewModel(context.Background(), "qwen-test", WithBaseURL(server.URL+"/v1"), WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("NewModel() error = %v", err)
	}

	req := &model.LLMRequest{Contents: []*genai.Content{genai.NewContentFromText("ping", genai.RoleUser)}}
	var got *model.LLMResponse
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("GenerateContent() error = %v", err)
		}
		got = resp
	}
	if captured.Model != "qwen-test" || len(captured.Messages) != 1 || captured.Messages[0].Content != "ping" || captured.Stream {
		t.Fatalf("captured request mismatch: %#v", captured)
	}
	if got == nil || len(got.Content.Parts) != 1 || got.Content.Parts[0].Text != "ok from mock server" {
		t.Fatalf("response mismatch: %#v", got)
	}
}

func TestGenerateContent_StreamText(t *testing.T) {
	var captured chatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"chatcmpl-stream\",\"model\":\"deepseek_v4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprint(w, "data: {\"id\":\"chatcmpl-stream\",\"model\":\"deepseek_v4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"你好\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprint(w, "data: {\"id\":\"chatcmpl-stream\",\"model\":\"deepseek_v4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"，世界\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprint(w, "data: {\"id\":\"chatcmpl-stream\",\"model\":\"deepseek_v4\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	llm, err := NewModel(context.Background(), "deepseek_v4", WithBaseURL(server.URL+"/v1"), WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("NewModel() error = %v", err)
	}

	var partials []string
	var final *model.LLMResponse
	for resp, err := range llm.GenerateContent(context.Background(), &model.LLMRequest{Contents: []*genai.Content{genai.NewContentFromText("ping", genai.RoleUser)}}, true) {
		if err != nil {
			t.Fatalf("GenerateContent(stream) error = %v", err)
		}
		if resp.Partial {
			partials = append(partials, resp.Content.Parts[0].Text)
		} else {
			final = resp
		}
	}
	if !captured.Stream {
		t.Fatalf("captured stream = false, want true")
	}
	if strings.Join(partials, "") != "你好，世界" {
		t.Fatalf("partials = %#v", partials)
	}
	if final == nil || !final.TurnComplete || final.Content.Parts[0].Text != "你好，世界" || final.FinishReason != genai.FinishReasonStop {
		t.Fatalf("final mismatch: %#v", final)
	}
}

func TestGenerateContent_StreamToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"chatcmpl-tool\",\"model\":\"deepseek_v4\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_abc\",\"type\":\"function\",\"function\":{\"name\":\"get_weather\",\"arguments\":\"{\\\"city\\\":\"}}]},\"finish_reason\":null}]}\n\n")
		fmt.Fprint(w, "data: {\"id\":\"chatcmpl-tool\",\"model\":\"deepseek_v4\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"Beijing\\\"}\"}}]},\"finish_reason\":null}]}\n\n")
		fmt.Fprint(w, "data: {\"id\":\"chatcmpl-tool\",\"model\":\"deepseek_v4\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	llm, err := NewModel(context.Background(), "deepseek_v4", WithBaseURL(server.URL+"/v1"), WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("NewModel() error = %v", err)
	}

	var final *model.LLMResponse
	for resp, err := range llm.GenerateContent(context.Background(), &model.LLMRequest{Contents: []*genai.Content{genai.NewContentFromText("weather", genai.RoleUser)}}, true) {
		if err != nil {
			t.Fatalf("GenerateContent(stream tool) error = %v", err)
		}
		if !resp.Partial {
			final = resp
		}
	}
	if final == nil || len(final.Content.Parts) != 1 || final.Content.Parts[0].FunctionCall == nil {
		t.Fatalf("final tool call missing: %#v", final)
	}
	fc := final.Content.Parts[0].FunctionCall
	if fc.ID != "call_abc" || fc.Name != "get_weather" || fc.Args["city"] != "Beijing" {
		t.Fatalf("function call mismatch: %#v", fc)
	}
	if final.FinishReason != genai.FinishReason("tool_calls") {
		t.Fatalf("finish reason = %q", final.FinishReason)
	}
}

func TestDoChatCompletion_HTTPStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway from mock", http.StatusBadGateway)
	}))
	defer server.Close()

	llm, err := NewModel(context.Background(), "qwen-test", WithBaseURL(server.URL+"/v1"), WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("NewModel() error = %v", err)
	}
	m := llm.(*openAICompatModel)

	_, err = m.doChatCompletion(context.Background(), chatCompletionRequest{Model: "qwen-test", Messages: []chatMessage{{Role: "user", Content: "ping"}}})
	if !errors.Is(err, ErrEndpointStatus) {
		t.Fatalf("err = %v, want ErrEndpointStatus", err)
	}
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("status error mismatch: %#v", statusErr)
	}
}
