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
package sequentialagent

// 代码走读：sequentialagent 是最简单的工作流 Agent：按顺序执行子 Agent。
//
// 业务理解：
// 适合“固定流水线”场景，例如：需求分析 -> 检索资料 -> 写初稿 -> 审稿 -> 输出。
// 它不自己调用模型，只负责把同一个 InvocationContext 传给每个子 Agent。
