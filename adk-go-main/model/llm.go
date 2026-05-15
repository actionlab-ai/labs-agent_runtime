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

// Package model defines the Archinfra Agent Runtime LLM abstraction layer. It contains the interfaces and data structures used to connect Gemini, OpenAI-compatible models, Qwen, DeepSeek, or other custom LLM providers.
package model

import (
	"context"
	"iter"

	"google.golang.org/genai"
)

// ADK 不关心你后面接的是 Gemini、OpenAI、Qwen、DeepSeek、LiteLLM 还是本地模型。
// 只要你实现 model.LLM，ADK 就可以把它当成模型用。
type LLM interface {
	Name() string
	// iter迭代器
	// 非流式：yield 一次最终 LLMResponse
	// 流式：yield 多次 partial LLMResponse，最后 yield 一次 TurnComplete / final response
	GenerateContent(ctx context.Context, req *LLMRequest, stream bool) iter.Seq2[*LLMResponse, error]
}

// LLMRequest is the raw LLM request.
// LLMRequest = ADK 准备发给模型的一次请求包
// Config.Tools：给模型看的工具 schema
// Tools：给 ADK 自己执行的 Go 工具实例
type LLMRequest struct {
	Model    string           // 这次实际要用的模型。如果为空，就用 NewModel(...) 时创建模型对象的默认名称。这意味着 ADK 支持 callback 在调用前改模型：默认模型：qwen_qwen3.5-397b-a17b，BeforeModelCallback 里可以改成 deepseek-v4-flash
	Contents []*genai.Content // 这是模型上下文。它不是普通字符串，而是 ADK 当前复用的内部统一消息格式。
	/*
		SystemInstruction    系统提示词
		Temperature          温度
		MaxOutputTokens      最大输出
		Tools                工具声明
		ResponseSchema       输出结构约束
		SafetySettings       安全设置
		HTTPOptions          Gemini 相关 HTTP 配置
	*/
	// 对 OpenAI-compatible 最重要的是这两个：Config.SystemInstruction / Config.Tools
	// 也就是：SystemInstruction -> OpenAI messages[0] role=system / Tools             -> OpenAI tools
	Config *genai.GenerateContentConfig
	Tools  map[string]any `json:"-"` // 这个字段有：`json:"-"`,说明它不会发给模型。它是 ADK Runtime 内部用的，主要为了保留真实工具对象。
}

// LLMResponse is the raw LLM response.
// It provides the first candidate response from the model if available.
/*
普通文本响应：
Content.Role = "model"
Content.Parts[0].Text = "你好，我是 Qwen..."
FinishReason = "stop"
---------------------
工具调用响应：
Content.Role = "model"
Content.Parts[0].FunctionCall = &genai.FunctionCall{
    ID: "call_001",
    Name: "get_weather",
    Args: map[string]any{"city": "Beijing"},
}
FinishReason = "tool_calls"
*/
type LLMResponse struct {
	Content           *genai.Content // 模型输出内容
	CitationMetadata  *genai.CitationMetadata
	GroundingMetadata *genai.GroundingMetadata
	UsageMetadata     *genai.GenerateContentResponseUsageMetadata
	CustomMetadata    map[string]any
	LogprobsResult    *genai.LogprobsResult
	ModelVersion      string // 实际模型名
	// Partial indicates whether the content is part of a unfinished content stream.
	// Only used for streaming mode and when the content is plain text.
	// The Runner fully processes only the final non-partial event, partial
	// events are simply forwarded downstream (eg. to UI for display).
	Partial bool // 是否流式片段
	// Indicates whether the response from the model is complete.
	// Only used for streaming mode.
	TurnComplete bool
	// Flag indicating that LLM was interrupted when generating the content.
	// Usually it is due to user interruption during a bidi streaming.
	Interrupted  bool
	ErrorCode    string             // 错误码
	ErrorMessage string             // 错误信息
	FinishReason genai.FinishReason //  stop / tool_calls / safety 等
	AvgLogprobs  float64
}
