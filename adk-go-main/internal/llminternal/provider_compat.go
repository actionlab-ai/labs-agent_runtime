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

package llminternal

import (
	"regexp"
	"strconv"
	"strings"

	"google.golang.org/adk/model"
)

var geminiModelVersionRegex = regexp.MustCompile(`^gemini-(\d+(\.\d+)?)`)

func needsOutputSchemaProcessorForModel(llm model.LLM) bool {
	if llm == nil || model.GetProviderBackend(llm) != model.ProviderBackendGoogleGeminiAPI {
		return false
	}
	name := extractModelName(llm.Name())
	if !strings.HasPrefix(name, "gemini-") {
		return false
	}
	matches := geminiModelVersionRegex.FindStringSubmatch(name)
	if len(matches) < 2 {
		return false
	}
	version, err := strconv.ParseFloat(matches[1], 64)
	return err == nil && version <= 2.5
}

func extractModelName(model string) string {
	return strings.ToLower(model[strings.LastIndex(model, "/")+1:])
}
