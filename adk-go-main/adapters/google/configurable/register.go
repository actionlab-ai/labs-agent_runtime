// Copyright 2026 Google LLC
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

// Package configurable registers Google adapter integrations for the YAML
// configurable-agent runtime. Import this package when Google/Gemini providers
// or native tools should be available to configuration files.
package configurable

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"

	"google.golang.org/adk/adapters/google/model/gemini"
	"google.golang.org/adk/adapters/google/tool/geminitool"
	coreconfig "google.golang.org/adk/internal/configurable"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
)

// Register installs Google/Gemini model and native-tool factories into the
// configurable runtime. It is intentionally explicit so core startup does not
// enable provider-specific tools by default.
func Register() error {
	if err := coreconfig.RegisterModelFactory("google", newGeminiModel); err != nil {
		return err
	}
	if err := coreconfig.RegisterToolFactory("google_search", func(_ context.Context, _ map[string]any) (tool.Tool, error) {
		return geminitool.GoogleSearch{}, nil
	}); err != nil {
		return err
	}
	if err := coreconfig.RegisterToolFactory("url_context", func(_ context.Context, _ map[string]any) (tool.Tool, error) {
		return geminitool.New("url_context", "URL context", &genai.Tool{URLContext: &genai.URLContext{}}), nil
	}); err != nil {
		return err
	}
	if err := coreconfig.RegisterToolFactory("google_maps_grounding", func(_ context.Context, _ map[string]any) (tool.Tool, error) {
		return geminitool.New("google_maps_grounding", "Google Maps grounding", &genai.Tool{GoogleMaps: &genai.GoogleMaps{}}), nil
	}); err != nil {
		return err
	}
	return nil
}

func newGeminiModel(ctx context.Context, modelName string) (model.LLM, error) {
	if modelName == "" {
		return nil, fmt.Errorf("model name is required")
	}
	return gemini.NewModel(ctx, modelName, &genai.ClientConfig{APIKey: os.Getenv("GOOGLE_API_KEY")})
}
