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
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/genai"

	"google.golang.org/adk/model"
)

// TestIntegrationDeepSeekV4_TextAndStream 使用真实 OpenAI-compatible 模型做集成测试。
//
// 默认跳过，避免普通 go test 依赖外部网络和私有模型服务。
// Windows PowerShell 示例：
//
//	$env:OPENAI_COMPAT_INTEGRATION = "1"
//	$env:OPENAI_COMPAT_BASE_URL = "http://36.147.35.14:30080/v1"
//	$env:OPENAI_COMPAT_MODEL = "deepseek_v4"
//	$env:OPENAI_COMPAT_API_KEY = "EMPTY"
//	go test ./model/openaicompat -run TestIntegrationDeepSeekV4_TextAndStream -v -count=1
func TestIntegrationDeepSeekV4_TextAndStream(t *testing.T) {
	if os.Getenv("OPENAI_COMPAT_INTEGRATION") != "1" {
		t.Skip("set OPENAI_COMPAT_INTEGRATION=1 to run real model integration test")
	}

	baseURL := envOrDefault("OPENAI_COMPAT_BASE_URL", "http://36.147.35.14:30080/v1")
	modelName := envOrDefault("OPENAI_COMPAT_MODEL", "deepseek_v4")
	apiKey := envOrDefault("OPENAI_COMPAT_API_KEY", "EMPTY")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	llm, err := NewModel(ctx, modelName, WithBaseURL(baseURL), WithAPIKey(apiKey), WithDebug(true))
	if err != nil {
		t.Fatalf("NewModel() error = %v", err)
	}

	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText("用一句中文回答：你现在是否支持流式输出？", genai.RoleUser),
		},
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText("你是一个用于测试 OpenAI-compatible 流式输出的模型。回答要简短。", genai.RoleUser),
		},
	}

	var partialCount int
	var partialText strings.Builder
	var final *model.LLMResponse
	for resp, err := range llm.GenerateContent(ctx, req, true) {
		if err != nil {
			t.Fatalf("GenerateContent(stream=true) error = %v", err)
		}
		if resp == nil {
			continue
		}
		if resp.Partial {
			partialCount++
			if resp.Content != nil && len(resp.Content.Parts) > 0 {
				partialText.WriteString(resp.Content.Parts[0].Text)
			}
			continue
		}
		final = resp
	}

	if partialCount == 0 {
		t.Fatalf("stream produced no partial chunks")
	}
	if final == nil {
		t.Fatalf("stream produced no final aggregated response; partial=%q", partialText.String())
	}
	if final.Content == nil || len(final.Content.Parts) == 0 || strings.TrimSpace(final.Content.Parts[0].Text) == "" {
		t.Fatalf("final response has no text: %#v; partial=%q", final, partialText.String())
	}
	t.Logf("partial chunks=%d", partialCount)
	t.Logf("partial text=%s", partialText.String())
	t.Logf("final text=%s", final.Content.Parts[0].Text)
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
