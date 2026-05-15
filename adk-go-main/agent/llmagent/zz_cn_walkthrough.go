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
package llmagent

// 代码走读：llmagent 包负责把“一个大模型驱动的 Agent”组装出来。
//
// 业务理解：
// 1. LLMAgent 是最常用的智能体类型：它有模型、指令、工具、子 Agent、输入/输出 Schema、回调等。
// 2. New(cfg) 并不是直接暴露内部执行细节，而是把配置转成 internal/llminternal.Flow 所需的状态。
// 3. 真正的 LLM 请求构造、工具调用、函数响应回灌、TransferToAgent 发生在 internal/llminternal。
//
// 技术设计模式：
// - Facade：llmagent.Config 对业务开发者友好，隐藏底层 Flow/Processor/Callback 复杂度。
// - Adapter：把公开的 llmagent 回调类型转换成 internal/llminternal 回调类型。
// - State Reveal：通过 internal state 保存 Agent 类型和配置，供调试、REST、图生成等模块读取。
//
// 读代码入口：
// - llmagent.go：配置字段含义、构造流程、LLM Agent 与基础 Agent 的组合关系。
// - internal/llminternal/base_flow.go：LLM Agent 的真实执行循环。
