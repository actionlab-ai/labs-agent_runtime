# 文档总览

截至 `2026-04-28`，后端当前可以分成三层理解：

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

- `POST /v1/runs`
  - 通用入口。
  - 允许 router、`tool_search` 和动态 skill tool 激活。
- `POST /v1/workflows/project-kickoff`
  - 固定执行 `novel-project-kickoff`。
  - 负责项目定调。
- `POST /v1/workflows/project-kernel`
  - 固定执行 `novel-emotional-core`。
  - 负责情感内核。
  - 现在支持“信息不足先反问，不落正式 novel_core”。

## 这轮最重要的变化

- `novel-emotional-core` 改成了明确的 `clarify_first` 契约。
- `project-kernel` 现在会返回：
  - `response_mode`
  - `needs_input`
- 当模型输出的是澄清问题时：
  - `response_mode=clarification`
  - `needs_input=true`
  - 不会把结果落成正式 `novel_core`
- 当模型输出的是正式内核文档时：
  - `response_mode=document`
  - `needs_input=false`
  - 会落库为 `novel_core`

## 推荐阅读顺序

1. [BUSINESS_ARCHITECTURE.md](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/docs/BUSINESS_ARCHITECTURE.md)
2. [IMPLEMENTATION.md](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/docs/IMPLEMENTATION.md)
3. [PROJECT_KICKOFF_FLOW.md](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/docs/PROJECT_KICKOFF_FLOW.md)
4. [FIXED_WORKFLOW_MODEL_REQUEST.md](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/docs/FIXED_WORKFLOW_MODEL_REQUEST.md)
5. [SKILL_WORKFLOW_RUNTIME.md](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/docs/SKILL_WORKFLOW_RUNTIME.md)
6. [FULL_REQUEST_FLOW.md](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/docs/FULL_REQUEST_FLOW.md)
7. [SEARCH_PIPELINE.md](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/docs/SEARCH_PIPELINE.md)
8. [FILE_TOOLS_AND_DOCUMENT_OUTPUT.md](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/docs/FILE_TOOLS_AND_DOCUMENT_OUTPUT.md)
9. [SQL_RETRIEVAL_ARCHITECTURE.md](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/docs/SQL_RETRIEVAL_ARCHITECTURE.md)
10. [CONFIG.md](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/docs/CONFIG.md)
11. [NEXT_STEPS.md](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/docs/NEXT_STEPS.md)
