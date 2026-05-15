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

package model

// ProviderBackend identifies a provider-specific serving backend without making
// the core runtime import provider SDKs.
type ProviderBackend string

const (
	ProviderBackendUnspecified     ProviderBackend = ""
	ProviderBackendGoogleGeminiAPI ProviderBackend = "google_gemini_api"
	ProviderBackendGoogleVertexAI  ProviderBackend = "google_vertex_ai"
)

// BackendProvider is implemented by optional model adapters that want to expose
// backend metadata to core runtime features such as telemetry and compatibility
// processors.
type BackendProvider interface {
	ProviderBackend() ProviderBackend
}

// GetProviderBackend returns provider backend metadata when an optional adapter
// exposes it. Core runtime code should depend on this neutral interface instead
// of importing provider-specific adapter packages.
func GetProviderBackend(llm LLM) ProviderBackend {
	if llm == nil {
		return ProviderBackendUnspecified
	}
	provider, ok := llm.(BackendProvider)
	if !ok {
		return ProviderBackendUnspecified
	}
	return provider.ProviderBackend()
}
