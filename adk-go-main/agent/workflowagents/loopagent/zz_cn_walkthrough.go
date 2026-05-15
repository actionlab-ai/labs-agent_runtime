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
package loopagent

// 代码走读：loopagent 重复执行子 Agent，直到达到次数上限或事件触发 Escalate。
//
// 业务理解：
// 适合“迭代改进”场景，例如：生成代码 -> 测试 -> 修复 -> 再测试，直到通过或超出迭代次数。
