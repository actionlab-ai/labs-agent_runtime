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
﻿// Copyright 2025 Google LLC
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

// Package main demonstrates ADK REST server with an OpenAI-compatible model.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/openaicompat"
	"google.golang.org/adk/server/adkrest"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

const (
	defaultModelName = "qwen_qwen3.5-397b-a17b"
	defaultBaseURL   = "http://36.147.35.14:30080/v1"
)

func main() {
	ctx := context.Background()
	if os.Getenv("OPENAI_COMPAT_BASE_URL") == "" {
		_ = os.Setenv("OPENAI_COMPAT_BASE_URL", defaultBaseURL)
	}
	modelName := os.Getenv("OPENAI_COMPAT_MODEL")
	if modelName == "" {
		modelName = defaultModelName
	}

	llm, err := openaicompat.NewModel(ctx, modelName)
	if err != nil {
		log.Fatalf("failed to create OpenAI-compatible model: %v", err)
	}

	weatherTool, err := functiontool.New(functiontool.Config{
		Name:        "get_weather",
		Description: "查询指定城市的模拟天气。用于学习 ADK Tool Calling 数据流。",
	}, getWeather)
	if err != nil {
		log.Fatalf("failed to create weather tool: %v", err)
	}

	timeTool, err := functiontool.New(functiontool.Config{
		Name:        "get_time",
		Description: "查询指定城市的模拟时间。用于学习 ADK Tool Calling 数据流。",
	}, getTime)
	if err != nil {
		log.Fatalf("failed to create time tool: %v", err)
	}

	trace := newTraceDumper("_study_logs_round3")
	a, err := llmagent.New(llmagent.Config{
		Name:        "qwen_weather_time_agent",
		Model:       llm,
		Description: "使用自建 OpenAI-compatible Qwen 模型回答天气和时间问题。",
		Instruction: `你是一个用于学习 ADK Go Agent Runtime 的中文调试 Agent。

规则：
1. 如果用户问天气，必须调用 get_weather 工具，不要自己编造天气。
2. 如果用户问时间，必须调用 get_time 工具，不要自己编造时间。
3. 工具返回后，用中文总结给用户。
4. 如果用户只是问候或询问框架原理，可以直接回答。`,
		Tools: []tool.Tool{weatherTool, timeTool},
		BeforeModelCallbacks: []llmagent.BeforeModelCallback{
			trace.beforeModel,
		},
		AfterModelCallbacks: []llmagent.AfterModelCallback{
			trace.afterModel,
		},
		BeforeToolCallbacks: []llmagent.BeforeToolCallback{
			func(ctx tool.Context, t tool.Tool, args map[string]any) (map[string]any, error) {
				trace.dump("before-tool-"+t.Name(), map[string]any{"tool": t.Name(), "args": args})
				return nil, nil
			},
		},
		AfterToolCallbacks: []llmagent.AfterToolCallback{
			func(ctx tool.Context, t tool.Tool, args, result map[string]any, err error) (map[string]any, error) {
				trace.dump("after-tool-"+t.Name(), map[string]any{"tool": t.Name(), "args": args, "result": result, "error": errorString(err)})
				return nil, nil
			},
		},
	})
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	restServer, err := adkrest.NewServer(adkrest.ServerConfig{
		AgentLoader:     agent.NewSingleLoader(a),
		SessionService:  session.InMemoryService(),
		SSEWriteTimeout: 120 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to create REST API server: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", restServer))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	log.Printf("model=%s base_url=%s", modelName, os.Getenv("OPENAI_COMPAT_BASE_URL"))
	log.Println("starting server on :8080")
	log.Println("API available at http://localhost:8080/api/")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

type WeatherInput struct {
	City string `json:"city" jsonschema:"city name, for example: Beijing, Shanghai, London"`
}

type WeatherOutput struct {
	City        string `json:"city"`
	Weather     string `json:"weather"`
	Temperature string `json:"temperature"`
	Source      string `json:"source"`
}

func getWeather(ctx tool.Context, input WeatherInput) (WeatherOutput, error) {
	city := input.City
	if city == "" {
		city = "未知城市"
	}
	return WeatherOutput{
		City:        city,
		Weather:     "晴，微风，适合学习 Agent 框架",
		Temperature: "23°C",
		Source:      "local-demo-tool",
	}, nil
}

type TimeInput struct {
	City string `json:"city" jsonschema:"city name, for example: Beijing, Shanghai, London"`
}

type TimeOutput struct {
	City   string `json:"city"`
	Time   string `json:"time"`
	Note   string `json:"note"`
	Source string `json:"source"`
}

func getTime(ctx tool.Context, input TimeInput) (TimeOutput, error) {
	city := input.City
	if city == "" {
		city = "未知城市"
	}
	return TimeOutput{
		City:   city,
		Time:   time.Now().Format(time.RFC3339),
		Note:   "这是本地示例工具返回的服务器时间，不是真实城市时区换算。",
		Source: "local-demo-tool",
	}, nil
}

type traceDumper struct {
	dir string
	seq atomic.Int64
}

func newTraceDumper(dir string) *traceDumper {
	_ = os.MkdirAll(dir, 0755)
	return &traceDumper{dir: dir}
}

func (d *traceDumper) beforeModel(ctx agent.CallbackContext, req *model.LLMRequest) (*model.LLMResponse, error) {
	d.dump("before-model", req)
	return nil, nil
}

func (d *traceDumper) afterModel(ctx agent.CallbackContext, resp *model.LLMResponse, llmErr error) (*model.LLMResponse, error) {
	d.dump("after-model", map[string]any{"response": resp, "error": errorString(llmErr)})
	return nil, nil
}

func (d *traceDumper) dump(prefix string, v any) {
	id := d.seq.Add(1)
	name := filepath.Join(d.dir, time.Now().Format("150405")+"-"+formatSeq(id)+"-"+prefix+".json")
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		b = []byte(`{"marshal_error":` + jsonQuote(err.Error()) + `}`)
	}
	if err := os.WriteFile(name, b, 0644); err != nil {
		log.Printf("failed to write trace %s: %v", name, err)
	}
}

func formatSeq(id int64) string {
	if id < 10 {
		return "00" + jsonNumber(id)
	}
	if id < 100 {
		return "0" + jsonNumber(id)
	}
	return jsonNumber(id)
}

func jsonNumber(id int64) string {
	b, _ := json.Marshal(id)
	return string(b)
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

