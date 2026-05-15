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
package context

// 代码走读：internal/context 是 InvocationContext/CallbackContext 的具体实现。
//
// 业务理解：
// 1. InvocationContext 是一轮用户请求的运行时背包，里面有当前 Agent、Session、用户输入、Artifact、Memory、RunConfig。
// 2. CallbackContext 是给回调/插件使用的受控上下文，支持记录 StateDelta 和 ArtifactDelta。
// 3. ToolContext 又在此基础上增加 FunctionCallID、ToolConfirmation、EventActions。
//
// 技术设计模式：
// - Context Object：避免每层函数传一长串参数。
// - Capability Scoping：不同上下文暴露不同能力，减少误用。
