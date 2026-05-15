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
package functiontool

// 代码走读：functiontool 包负责把一个普通 Go 函数包装成 LLM 可调用工具。
//
// 业务理解：
// 1. 业务开发者写 func(tool.Context, Args) (Result, error)，框架自动推导 JSON Schema。
// 2. 模型看到的是 FunctionDeclaration，包括工具名、描述、参数 Schema、返回 Schema。
// 3. 运行时把模型传来的 map 参数转换成强类型 Args，执行函数，再把 Result 转回 map。
// 4. 支持长任务工具和 Human-in-the-Loop 确认，适合支付、删除、发邮件等敏感操作。
//
// 技术设计模式：
// - Reflection + Generics：用 Go 泛型和反射自动推导参数/返回结构。
// - Adapter：把强类型 Go 函数适配为 tool.Tool + internal FunctionTool。
// - Panic Boundary：Run 中 recover，避免业务函数 panic 直接打崩 Agent 主循环。
