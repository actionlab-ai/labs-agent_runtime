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
package parallelagent

// 代码走读：parallelagent 并发执行多个子 Agent，并通过 branch 隔离它们的历史视角。
//
// 业务理解：
// 适合“多专家并行”场景，例如：架构师/安全专家/性能专家同时审查同一段代码。
// 每个子 Agent 的事件会带不同 Branch，避免互相看到对方中间过程。
