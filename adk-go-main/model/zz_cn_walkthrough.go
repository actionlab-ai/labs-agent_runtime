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
package model

// 代码走读：model 包是 LLM Provider 的最小抽象层。
//
// 业务理解：
// 1. LLMRequest 是框架内部统一的大模型请求对象。
// 2. LLMResponse 是框架内部统一的大模型响应对象，支持流式 partial、token usage、引用、grounding、错误码等。
// 3. 具体模型实现只需要实现 GenerateContent，框架上层就能复用 Runner/Agent/Tool/Session 流程。
//
// 技术设计模式：
// - Provider Adapter：Gemini、OpenAI 兼容模型或自定义模型都可以适配成 model.LLM。
// - Streaming Abstraction：通过 iter.Seq2 统一非流式和流式输出。
