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
package llminternal

// 代码走读：internal/llminternal 是 LLM Agent 的“推理循环内核”。
//
// 业务理解：
// 1. Flow.Run 是 LLM Agent 的主循环：构造请求 -> 调模型 -> 生成事件 -> 如果模型要求工具调用则执行工具 -> 把工具结果回灌 -> 再调模型，直到最终回复。
// 2. RequestProcessors 类似一条请求加工流水线，负责注入系统指令、历史上下文、工具声明、Schema、转交工具等。
// 3. ResponseProcessors 类似响应后处理流水线，负责处理代码执行、规划标记等模型返回内容。
// 4. 工具调用是并发执行的，多个 function_call 会合并成一个 function_response event。
//
// 技术设计模式：
// - Pipeline：请求处理器和响应处理器按顺序执行。
// - ReAct/Tool Calling Loop：模型输出 function_call 后，框架执行工具，再把 function_response 作为新事件交回模型。
// - Hook Chain：模型前、模型后、模型错误、工具前、工具后、工具错误都可插拔。
//
// 读代码入口：
// - base_flow.go：runOneStep/callLLM/handleFunctionCalls/finalizeModelResponseEvent。
// - functions.go：工具确认事件与 function response 事件辅助逻辑。
// - contents_processor.go：如何把 Session 历史拼到 LLMRequest。
// - instruction_processor.go：如何注入 Instruction/GlobalInstruction。
