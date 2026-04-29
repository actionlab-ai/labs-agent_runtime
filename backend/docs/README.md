# 文档总览

截至 `2026-04-29`，后端当前可以分成三层理解：

1. 进程入口层
   - [main.go](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/cmd/novelrt/main.go)
   - 只负责读取配置并启动 HTTP 服务。
2. HTTP API 层
   - [internal/httpapi](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/httpapi)
   - 负责路由、workflow 编排、run 编排、项目文档落库。
3. 运行时与业务层
   - `internal/runtime`
   - `internal/workflow`
   - `internal/skill`
   - `internal/project`
   - `internal/store`

## 现在最关键的接口

- `POST /v1/skill-sessions`
  - 当前小说能力的主入口。
  - 单个 skill 在项目上下文里执行，缺信息时直接通过 `AskHuman` 补问。
- `POST /v1/skill-sessions/{id}/turns`
  - 继续同一个 session。
  - 人类补充答案后，模型沿着原对话继续，而不是重新开始。
- `POST /v1/runs`
  - 通用入口。
  - 允许 router、`tool_search` 和动态 skill tool 激活。
- `POST /v1/workflows/project-kickoff`
  - 历史固定入口，当前不作为新小说能力默认路径。
- `POST /v1/workflows/project-kernel`
  - 历史固定入口，当前不作为新小说能力默认路径。

## 这轮最重要的变化

- 新的小说类内容默认基于 `skill-session` 执行。
- `AskHuman` 现在是默认输入补全机制。
- 项目文档流转统一走 provider，不通过 skill 猜文件路径。
- `project_document_policy` 存在 PG 的 `app_settings` 里，是文档 kind、上下文排序、skill 依赖文档的统一配置源。
- 典型链路是：先用 `novel-emotional-core` 写入 `project_documents(kind=novel_core)`，再运行 `novel-world-engine`，runtime 按 policy 自动读取并注入 `novel_core`。
- 关键原则变成：
  - 信息不足先暂停并提问
  - 人类答案回到同一 session
  - 信息足够后再写正式项目文档
- 固定 workflow 和 bootstrap 路线先冻结，不作为当前推进重点。

## 推荐阅读顺序

1. [SKILL_SESSION_BUSINESS_FLOW.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/SKILL_SESSION_BUSINESS_FLOW.md)
   - 重点看“核心到第二任务的数据流”，里面有 `novel_core` 保存和 `world_engine` 自动带入的图。
2. [PROJECT_DOCUMENT_POLICY.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/PROJECT_DOCUMENT_POLICY.md)
   - 解释 `project_document_policy` 的业务意义、完整流程，以及它如何把第一个 skill 的产物带给第二个 skill。
3. [WEBNOVEL_LAUNCH_BLUEPRINT.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/WEBNOVEL_LAUNCH_BLUEPRINT.md)
   - 说明已有 `novel_core` 和 `world_engine` 后，正式开书还必须补哪些 skill、项目文档和引用关系。
4. [WEBNOVEL_SKILL_SESSION_CALL_EXAMPLES.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/WEBNOVEL_SKILL_SESSION_CALL_EXAMPLES.md)
   - 可直接复制的 skill-session curl 调用实例，覆盖初始化链路里每个 skill。
5. [SKILL_SESSION_CONTROLLER.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/SKILL_SESSION_CONTROLLER.md)
6. [BUSINESS_ARCHITECTURE.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/BUSINESS_ARCHITECTURE.md)
7. [IMPLEMENTATION.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/IMPLEMENTATION.md)
8. [FIXED_WORKFLOW_MODEL_REQUEST.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/FIXED_WORKFLOW_MODEL_REQUEST.md)
9. [SKILL_WORKFLOW_RUNTIME.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/SKILL_WORKFLOW_RUNTIME.md)
10. [FULL_REQUEST_FLOW.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/FULL_REQUEST_FLOW.md)
11. [SEARCH_PIPELINE.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/SEARCH_PIPELINE.md)
12. [FILE_TOOLS_AND_DOCUMENT_OUTPUT.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/FILE_TOOLS_AND_DOCUMENT_OUTPUT.md)
13. [SQL_RETRIEVAL_ARCHITECTURE.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/SQL_RETRIEVAL_ARCHITECTURE.md)
14. [CONFIG.md](/C:/Users/yuanyp8/Desktop/labs-agent_runtime/backend/docs/CONFIG.md)
