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
package adkrest

// 代码走读：server/adkrest 包把 Agent Runtime 暴露成 REST/SSE HTTP 服务。
//
// 业务理解：
// 1. WebUI 或外部客户端通过 REST API 创建/读取 Session、发送消息、读取 Artifact、查看 Debug Trace。
// 2. RuntimeAPIController 内部会创建 Runner，然后用 SSE 把 Runner.Run 产生的事件流返回给前端。
// 3. DebugTelemetry 通过 OpenTelemetry Span/Log 保存运行轨迹，适合分析模型请求、工具调用、Agent 转交。
//
// 技术设计模式：
// - Adapter：HTTP 请求转换成 Runner 调用。
// - Router/Controller 分层：routers 只负责路由，controllers 负责业务处理。
// - SSE Streaming：适合把模型 partial/event 增量推给浏览器。
