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
package artifact

// 代码走读：artifact 包是文件/二进制内容的会话级存储抽象。
//
// 业务理解：
// 1. Artifact 用来保存图片、音频、上传文件、模型生成文件等不适合直接塞进 prompt 的内容。
// 2. Runner 可以把用户输入中的 InlineData 保存为 Artifact，再把消息替换为文本占位符。
// 3. 工具和 Agent 可以通过 agent.Artifacts 读取/保存这些文件。
//
// 技术设计模式：
// - Storage Interface：Service 屏蔽内存、本地、GCS 等不同存储后端。
// - Versioned Artifact：同名文件可以有多个版本，便于生成式任务迭代。
