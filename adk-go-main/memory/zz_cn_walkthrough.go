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
package memory

// 代码走读：memory 包定义跨 Session 的长期记忆接口。
//
// 业务理解：
// 1. Session 是短中期会话历史，Memory 是跨会话可搜索知识。
// 2. AddSessionToMemory 可以把一条会话吸收进长期记忆。
// 3. SearchMemory 按 query 检索相关记忆，常由 LoadMemoryTool/PreloadMemoryTool 暴露给 Agent。
//
// 技术设计模式：
// - Retrieval Interface：具体可接向量库、Vertex AI Memory、数据库全文索引等实现。
