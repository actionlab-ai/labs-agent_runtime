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
package agent

// 代码走读：agent 包是整个框架的“抽象层”。
//
// 业务理解：
// 1. 一个 Agent 可以理解为“可执行的智能体节点”。它既可以是直接调用大模型的 LLMAgent，
//    也可以是自定义逻辑 Agent、顺序/并行/循环工作流 Agent、远程 Agent。
// 2. Agent 不直接关心 HTTP、数据库、WebUI，它只暴露 Run(ctx) 这个流式事件接口。
// 3. Agent 的产物不是简单字符串，而是 session.Event；Event 可以表示模型回复、工具调用、工具结果、
//    状态变更、跨 Agent 转交、人工确认等。
//
// 技术设计模式：
// - Interface + Constructor：外部只依赖 Agent 接口，通过 New/llmagent.New 等构造函数创建。
// - Tree/Composite：Agent 可以有 SubAgents，形成一棵可转交、可编排的 Agent 树。
// - Iterator/Streaming：Run 返回 iter.Seq2[*session.Event, error]，天然支持边生成边输出。
// - Callback Hook：BeforeAgentCallbacks / AfterAgentCallbacks 允许在 Agent 执行前后插入业务逻辑。
//
// 读代码入口：
// - agent.go：Agent 接口、基础 Agent 实现、执行前后回调。
// - context.go：InvocationContext 的生命周期定义，是理解 Runner/Agent/Tool 之间传参的关键。
