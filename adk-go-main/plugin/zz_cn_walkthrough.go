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
package plugin

// 代码走读：plugin 包提供全局插件机制，是观测、审计、改写、限流、缓存的扩展点。
//
// 业务理解：
// 1. Plugin 可以在用户消息、Runner、Agent、Model、Tool、Event 等关键节点前后插入逻辑。
// 2. 它适合做透明代理日志、提示词审计、模型输出审查、工具调用安全校验、token 统计等横切能力。
// 3. 插件和 Agent 回调不同：插件更偏全局横切；Agent 回调更偏单个 Agent 的业务逻辑。
//
// 技术设计模式：
// - Interceptor/Middleware：在主流程关键节点前后拦截或改写。
// - Chain of Responsibility：多个插件按顺序执行，遇到返回值/错误时可短路。
