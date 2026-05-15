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

package model_test

import (
	"reflect"
	"testing"

	"google.golang.org/genai"

	"google.golang.org/adk/internal/llminternal/converters"
	"google.golang.org/adk/model"
)

// 定义一些常量，模拟 genai 中的 FinishReason 和 BlockedReason 枚举值。
// 这些常量主要用于测试用例中构建输入和期望输出，避免硬编码字符串。
const (
	FinishReasonStop       genai.FinishReason = "STOP"       // 正常结束
	FinishReasonSafety     genai.FinishReason = "SAFETY"     // 因安全策略停止
	FinishReasonRecitation genai.FinishReason = "RECITATION" // 因重复内容停止
)

const (
	BlockedReasonSafety genai.BlockedReason = "SAFETY" // 因安全策略被阻止
)

// TestCreateResponse 测试 Genai2LLMResponse 转换函数的正确性。
// 该函数负责将 Google GenAI SDK 返回的原始响应 (genai.GenerateContentResponse)
// 转换为 ADK 内部使用的 LLMResponse 结构。
func TestCreateResponse(t *testing.T) {
	// 预先定义一些复杂的 Logprobs 结构，以便在多个测试用例中复用
	emptyLogprobs := &genai.LogprobsResult{
		ChosenCandidates: []*genai.LogprobsResultCandidate{},
		TopCandidates:    []*genai.LogprobsResultTopCandidates{},
	}
	// concreteLogprobs 验证“复杂真实数据”的转换正确性。
	concreteLogprobs := &genai.LogprobsResult{
		ChosenCandidates: []*genai.LogprobsResultCandidate{
			{Token: "The", LogProbability: -0.1, TokenID: 123},
			{Token: " capital", LogProbability: -0.5, TokenID: 456},
			{Token: " of", LogProbability: -0.2, TokenID: 789},
		},
		TopCandidates: []*genai.LogprobsResultTopCandidates{
			{Candidates: []*genai.LogprobsResultCandidate{{Token: "The"}, {Token: "A"}, {Token: "This"}}},
			{Candidates: []*genai.LogprobsResultCandidate{{Token: " capital"}, {Token: " city"}, {Token: " main"}}},
		},
	}
	partialLogprobs := &genai.LogprobsResult{
		ChosenCandidates: []*genai.LogprobsResultCandidate{
			{Token: "Hello", LogProbability: -0.05, TokenID: 111},
			{Token: " world", LogProbability: -0.8, TokenID: 222},
		},
		TopCandidates: []*genai.LogprobsResultTopCandidates{},
	}
	citationMeta := &genai.CitationMetadata{
		Citations: []*genai.Citation{{StartIndex: 0, EndIndex: 10, URI: "https://example.com"}},
	}

	// 定义测试用例表，每个用例包含名称、输入和期望输出
	testCases := []struct {
		name  string
		input genai.GenerateContentResponse
		want  model.LLMResponse
	}{
		{
			// 测试场景：正常响应，并且包含空的 Logprobs 结果（代表开启了 Logprobs 但没有内容）
			name: "CreateWithLogprobs",
			input: genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content:        &genai.Content{Parts: []*genai.Part{{Text: "Response text"}}},
					FinishReason:   FinishReasonStop,
					AvgLogprobs:    -0.75,
					LogprobsResult: emptyLogprobs,
				}},
			},
			want: model.LLMResponse{
				Content:        &genai.Content{Parts: []*genai.Part{{Text: "Response text"}}},
				FinishReason:   FinishReasonStop,
				AvgLogprobs:    -0.75,
				LogprobsResult: emptyLogprobs,
			},
		},
		{
			// 测试场景：正常响应，但没有 Logprobs 信息（未开启 Logprobs）
			name: "CreateWithoutLogprobs",
			input: genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content:      &genai.Content{Parts: []*genai.Part{{Text: "Response text"}}},
					FinishReason: FinishReasonStop,
				}},
			},
			want: model.LLMResponse{
				Content:      &genai.Content{Parts: []*genai.Part{{Text: "Response text"}}},
				FinishReason: FinishReasonStop,
			},
		},
		{
			// 测试场景：错误情况 —— 候选内容为空，但带有安全过滤的错误信息，同时包含 Logprobs 平均值。
			// 期望 LLMResponse 的 ErrorCode 和 ErrorMessage 被正确填充。
			name: "CreateErrorCaseWithLogprobs",
			input: genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					FinishReason:  FinishReasonSafety,
					FinishMessage: "Safety filter triggered",
					AvgLogprobs:   -2.1,
				}},
			},
			want: model.LLMResponse{
				ErrorCode:    string(FinishReasonSafety),
				ErrorMessage: "Safety filter triggered",
				AvgLogprobs:  -2.1,
				FinishReason: FinishReasonSafety,
			},
		},
		{
			// 测试场景：完全没有候选回复，但 PromptFeedback 指示提示词被安全策略阻止。
			// 期望错误信息取自 PromptFeedback。
			name: "CreateNoCandidates",
			input: genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{},
				PromptFeedback: &genai.GenerateContentResponsePromptFeedback{
					BlockReason:        BlockedReasonSafety,
					BlockReasonMessage: "Prompt blocked for safety",
				},
			},
			want: model.LLMResponse{
				ErrorCode:    string(BlockedReasonSafety),
				ErrorMessage: "Prompt blocked for safety",
			},
		},
		{
			// 测试场景：正常响应，包含具体的 Logprobs 结果（包含 chosen 和 top candidates）。
			// 验证复杂的嵌套结构能被正确传递。
			name: "CreateWithConcreteLogprobsResult",
			input: genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content:        &genai.Content{Parts: []*genai.Part{{Text: "The capital of France is Paris."}}},
					FinishReason:   FinishReasonStop,
					AvgLogprobs:    -0.27,
					LogprobsResult: concreteLogprobs,
				}},
			},
			want: model.LLMResponse{
				Content:        &genai.Content{Parts: []*genai.Part{{Text: "The capital of France is Paris."}}},
				FinishReason:   FinishReasonStop,
				AvgLogprobs:    -0.27,
				LogprobsResult: concreteLogprobs,
			},
		},
		{
			// 测试场景：部分 Logprobs 结果（只有 chosen candidates，top candidates 为空）。
			// 确保转换不会丢失字段。
			name: "CreateWithPartialLogprobsResult",
			input: genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content:        &genai.Content{Parts: []*genai.Part{{Text: "Hello world"}}},
					FinishReason:   FinishReasonStop,
					AvgLogprobs:    -0.425,
					LogprobsResult: partialLogprobs,
				}},
			},
			want: model.LLMResponse{
				Content:        &genai.Content{Parts: []*genai.Part{{Text: "Hello world"}}},
				FinishReason:   FinishReasonStop,
				AvgLogprobs:    -0.425,
				LogprobsResult: partialLogprobs,
			},
		},
		{
			// 测试场景：正常响应，包含引用元数据 (CitationMetadata)
			name: "CreateWithCitationMetadata",
			input: genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content:          &genai.Content{Parts: []*genai.Part{{Text: "Response text"}}},
					FinishReason:     FinishReasonStop,
					CitationMetadata: citationMeta,
				}},
			},
			want: model.LLMResponse{
				Content:          &genai.Content{Parts: []*genai.Part{{Text: "Response text"}}},
				FinishReason:     FinishReasonStop,
				CitationMetadata: citationMeta,
			},
		},
		{
			// 测试场景：正常响应，没有引用元数据，确认其为 nil
			name: "CreateWithoutCitationMetadata",
			input: genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content:      &genai.Content{Parts: []*genai.Part{{Text: "Response text"}}},
					FinishReason: FinishReasonStop,
				}},
			},
			want: model.LLMResponse{
				Content:      &genai.Content{Parts: []*genai.Part{{Text: "Response text"}}},
				FinishReason: FinishReasonStop,
			},
		},
		{
			// 测试场景：错误情况，因重复内容被阻止，同时包含引用元数据。
			// 验证错误码和元数据都能正确转换。
			name: "CreateErrorCaseWithCitationMetadata",
			input: genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					FinishReason:     FinishReasonRecitation,
					FinishMessage:    "Response blocked due to recitation triggered",
					CitationMetadata: citationMeta,
				}},
			},
			want: model.LLMResponse{
				ErrorCode:        string(FinishReasonRecitation),
				ErrorMessage:     "Response blocked due to recitation triggered",
				CitationMetadata: citationMeta,
				FinishReason:     FinishReasonRecitation,
			},
		},
	}

	// 遍历所有测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 调用被测试的转换函数
			got := converters.Genai2LLMResponse(&tc.input)

			// 验证 AvgLogprobs 字段是否一致
			if tc.want.AvgLogprobs != got.AvgLogprobs {
				t.Errorf("AvgLogprobs mismatch: want %f, got %f", tc.want.AvgLogprobs, got.AvgLogprobs)
			}

			// 验证错误码
			if got.ErrorCode != tc.want.ErrorCode {
				t.Errorf("ErrorCode mismatch: want %v, got %v", tc.want.ErrorCode, got.ErrorCode)
			}

			// 验证错误信息
			if got.ErrorMessage != tc.want.ErrorMessage {
				t.Errorf("ErrorMessage mismatch: want '%s', got '%s'", tc.want.ErrorMessage, got.ErrorMessage)
			}

			// 验证结束原因
			if got.FinishReason != tc.want.FinishReason {
				t.Errorf("FinishReason mismatch: want %s, got %s", tc.want.FinishReason, got.FinishReason)
			}

			// 使用 DeepEqual 深度比较复杂的嵌套结构体：Content、LogprobsResult、CitationMetadata
			if !reflect.DeepEqual(got.Content, tc.want.Content) {
				t.Errorf("Content mismatch: want %+v, got %+v", tc.want.Content, got.Content)
			}

			if !reflect.DeepEqual(got.LogprobsResult, tc.want.LogprobsResult) {
				t.Errorf("LogprobsResult mismatch: want %+v, got %+v", tc.want.LogprobsResult, got.LogprobsResult)
			}

			if !reflect.DeepEqual(got.CitationMetadata, tc.want.CitationMetadata) {
				t.Errorf("CitationMetadata mismatch: want %+v, got %+v", tc.want.CitationMetadata, got.CitationMetadata)
			}
		})
	}
}
