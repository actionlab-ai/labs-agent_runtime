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
package runner

// 代码走读：runner 包是 Agent 框架的“运行时总控”。
//
// 业务理解：
// 1. Runner 接收用户输入，找到/创建 Session，然后决定本轮应该由哪个 Agent 继续对话。
// 2. Runner 把 Session、Artifact、Memory、Plugin、RunConfig 等能力组装进 InvocationContext。
// 3. Runner 负责把用户消息和 Agent 产出的最终事件落库到 SessionService。
// 4. Runner 不关心具体模型怎么调用，也不关心工具怎么实现；它只消费 Agent.Run 产生的事件流。
//
// 核心数据流：
// 用户请求 -> Runner.Run -> SessionService.Get/Create -> findAgentToRun -> appendMessageToSession
// -> Agent.Run -> 事件回调/落 Session -> 向上游 UI/API 持续 yield 事件。
//
// 技术设计模式：
// - Orchestrator：统一编排 session、agent、memory、artifact、plugin。
// - Event Sourcing 雏形：对话历史以 Event 追加，Session 状态由 StateDelta 推导更新。
// - Streaming Iterator：Runner.Run 本身不阻塞等待最终答案，而是持续产出 Event。
