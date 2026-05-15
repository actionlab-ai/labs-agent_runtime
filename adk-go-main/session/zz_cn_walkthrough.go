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
package session

// 代码走读：session 包是“对话状态与事件历史”的核心存储抽象。
//
// 业务理解：
// 1. Session 表示某个 user 在某个 app 下的一条会话线程。
// 2. Event 表示会话中的一次原子事件：用户输入、模型输出、函数调用、函数响应、转交、状态变更等。
// 3. State 是会话的 KV 状态，支持 app/user/session/temp 四类作用域语义。
// 4. Service 是存储接口，inmemory/database/vertexai 等实现可以替换。
//
// 状态作用域：
// - app:    应用级状态，同一个 app 内共享。
// - user:   用户级状态，同一个 app + user 下多 session 共享。
// - 普通 key：当前 session 私有状态。
// - temp:   单轮 invocation 临时状态，追加事件前会被裁剪，不长期保存。
//
// 读代码入口：
// - session.go：Session/Event/State 的领域模型。
// - service.go：SessionService 接口。
// - inmemory.go：内存版实现，是理解数据结构和状态合并规则的最佳入口。
