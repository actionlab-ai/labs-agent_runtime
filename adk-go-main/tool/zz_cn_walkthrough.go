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
package tool

// 代码走读：tool 包定义了 Agent 可调用工具的通用协议。
//
// 业务理解：
// 1. Tool 代表“模型可以请求执行的一段外部能力”，如搜索、数据库查询、调用子 Agent、加载记忆等。
// 2. Toolset 是工具集合，可以按上下文动态返回工具列表。
// 3. tool.Context 让工具不仅能拿到参数，还能读写状态、搜索记忆、请求人工确认、触发 Agent 转交。
// 4. 工具不是直接返回自然语言，而是返回结构化 map，由框架包装成 FunctionResponse 事件给模型继续推理。
//
// 技术设计模式：
// - Capability Injection：Agent 通过工具获得外部能力，而不是把所有业务写死在 Prompt 里。
// - Policy Wrapper：FilterToolset/WithConfirmation 通过包装器给工具增加过滤和人工确认能力。
