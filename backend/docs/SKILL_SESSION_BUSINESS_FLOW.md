# Skill Session 业务流程全景

这份文档说明 `skill-session` 模块在产品里的完整作用：它怎么接住模型的反问，怎么等待人类补充信息，怎么带着原上下文继续发给模型，以及最后怎么通过 provider 写入项目资产。

当前约定已经收敛为：

- 所有新的小说类内容默认基于 `skill-session` 执行。
- `AskHuman` 视为 skill-session 的默认输入补全能力，不再把“文本式澄清后人工重提一次请求”当成主路径。
- 固定 workflow 和早期 bootstrap 路线先冻结，不作为当前新能力的落点。

## 结论

`skill-session` 是当前小说能力的主执行面，也是“单个 skill 的多轮交互控制器”。

它解决的问题是：

```text
用户提出任务
  -> 明确调用某个 skill
  -> 模型发现缺少关键信息
  -> 模型调用 AskHuman
  -> 服务端暂停 skill
  -> 前端展示问题
  -> 人类补充答案
  -> 服务端把答案作为 tool result 塞回原对话
  -> 模型继续完成 skill
  -> 如果需要持久化项目文档，仍然调用 WriteProjectDocument
```

它不替代 runtime，也不替代 provider。

- runtime 负责模型调用、prompt 组装、tool 暴露和 tool 执行。
- skill-session 负责暂停、保存会话状态、恢复会话。
- provider 负责项目文档的 PG / filesystem / Redis 同步。
- workflow 负责多个 skill 的产品级编排，但当前不再作为新内容的默认入口，也不应该自己实现 AskHuman 状态机。

## 当前落地策略

新 skill 的默认设计前提：

1. 先由 `skill-session` 启动单个 skill。
2. skill 缺信息时直接调用 `AskHuman`。
3. 人类补完后在同一 session 内继续，而不是重新起一次新任务。
4. 只有信息足够时才调用 `WriteProjectDocument` 落正式项目资产。

当前不作为主路径推进的东西：

- 固定 workflow 新增或扩展
- “先 bootstrap 再切主流程”的自举路线
- 依赖文本 `response_mode=clarification` 的一次性问答入口

## 核心到第二任务的数据流

当前最重要的业务链路是：

```text
第一步：用 skill-session 创建 novel_core
第二步：再开一个 skill-session 跑 novel-world-engine
第三步：runtime 根据 PG 里的 project_document_policy 自动把 novel_core 带进第二个 skill
```

这里的关键原则是：

- `novel_core` 是正式项目文档，不是聊天临时上下文。
- 保存位置以 `postgres.project_documents(project_id, kind)` 为准。
- skill 之间不靠文件路径传递产物。
- runtime 不硬编码“world-engine 必须读 novel_core”，而是读取 `postgres.app_settings.project_document_policy`。
- provider 负责把 PG、filesystem projection、Redis cache 串起来，skill 不直接碰底层存储。

### 1. 创建内核并保存

```mermaid
sequenceDiagram
  participant UI as Web UI
  participant HTTP as SkillSessionService
  participant Runtime as Runtime
  participant Model as Model
  participant Tool as WriteProjectDocument
  participant Provider as ProjectDocumentProvider
  participant PG as PostgreSQL project_documents
  participant FS as Filesystem Projection
  participant Redis as Redis Cache

  UI->>HTTP: POST /v1/skill-sessions<br/>skill_id=novel-emotional-core<br/>project=urban-rebirth
  HTTP->>Runtime: ExecuteSkillSession(project_id, skill_id, input)
  Runtime->>Model: compiled skill prompt + AskHuman + project tools
  alt 信息不足
    Model-->>Runtime: AskHuman(...)
    Runtime-->>HTTP: status=needs_input
    UI->>HTTP: POST /v1/skill-sessions/{id}/turns<br/>answers
    HTTP->>Runtime: ContinueSkillInteractive(...)
    Runtime->>Model: previous conversation + human tool result
  end
  Model-->>Runtime: WriteProjectDocument(kind="novel_core", body=...)
  Runtime->>Tool: execute tool call
  Tool->>Provider: UpsertProjectDocument(project_id, "novel_core")
  Provider->>PG: INSERT/UPDATE project_documents
  Provider->>FS: sync documents/novel_core.md
  Provider->>Redis: async cache project
  Provider-->>Runtime: body_bytes / synced
  Runtime-->>HTTP: session completed
  HTTP-->>UI: final_text + session snapshot
```

保存后的事实是：

```text
project_documents
  project_id = "urban-rebirth"
  kind       = "novel_core"
  title      = "小说情感内核"
  body       = "<正式内核正文>"
```

filesystem 只是 projection，用于人类查看和调试：

```text
projects/{storage_prefix}/documents/novel_core.md
```

后续 skill 不应该通过拼文件路径读取它，而应该走 provider。

### 2. 第二个任务如何带上第一个产物

第二次运行 `novel-world-engine` 时，请求本身只需要指定同一个 project：

```http
POST /v1/skill-sessions
Content-Type: application/json
```

```json
{
  "project": "urban-rebirth",
  "model": "deepseek-flash",
  "skill_id": "novel-world-engine",
  "input": "基于已经保存的情绪内核，设计这个世界如何持续制造冲突。",
  "arguments": {
    "document_kind": "world_engine"
  },
  "debug": true
}
```

服务端会在模型调用前做两层注入：

```mermaid
flowchart TD
  Req["POST /v1/skill-sessions<br/>project + skill_id=novel-world-engine"] --> Factory["runtimeFactory"]
  Factory --> Policy["Load project_document_policy<br/>from app_settings"]
  Factory --> Context["PostgresContextProvider.BuildContextWithPolicy"]
  Policy --> Context
  Context --> ListDocs["ListProjectDocuments(project_id)"]
  ListDocs --> PGDocs["project_documents"]
  Context --> ActiveCtx["Active Novel Project Context<br/>按 policy 排序"]
  Factory --> Runtime["Runtime(ProjectID, ProjectDocs, ProjectPolicy, ProjectContext)"]
  Policy --> Runtime
  ActiveCtx --> Runtime
  Runtime --> SkillDeps["ProjectPolicy.skill_documents[novel-world-engine]"]
  SkillDeps --> NeedCore["required kinds: novel_core"]
  NeedCore --> ReadCore["ProjectDocs.ReadProjectDocument(project_id, novel_core)"]
  ReadCore --> PGCore["project_documents(kind=novel_core)"]
  ReadCore --> Inject["Runtime-Loaded Project Documents<br/>## 小说情感内核 (novel_core)"]
  Inject --> Prompt["compiled skill prompt"]
  ActiveCtx --> Prompt
  Prompt --> Model["Model sees novel_core before drafting world_engine"]
```

这意味着第二个 skill 不需要自己猜路径，也不需要把“先读 novel_core”写成固定代码流程。

它只依赖一条配置：

```json
{
  "documents": [
    {"kind": "novel_core", "title": "小说情感内核", "priority": 0},
    {"kind": "world_engine", "title": "小说世界压力引擎", "priority": 50},
    {"kind": "world_rules", "title": "世界规则", "priority": 60},
    {"kind": "power_system", "title": "能力体系", "priority": 70}
  ],
  "skill_documents": {
    "novel-world-engine": ["novel_core"]
  }
}
```

这份配置的业务源是：

```text
postgres.app_settings
  key   = "project_document_policy"
  value = "<JSON policy>"
```

如果以后要让 `novel-world-engine` 同时带上 `reader_contract`，只改 PG 配置：

```json
{
  "skill_documents": {
    "novel-world-engine": ["novel_core", "reader_contract"]
  }
}
```

runtime 下一次启动 session 时会按新配置读取，不需要改 `SKILL.md`、SQL 排序、runtime 提示或 workflow 后处理。

### 3. 缺少 novel_core 时怎么办

如果 policy 要求 `novel-world-engine` 读取 `novel_core`，但 PG 里没有这份文档：

```mermaid
flowchart TD
  Runtime["Runtime before model call"] --> ReadCore["ReadProjectDocument(project_id, novel_core)"]
  ReadCore --> Missing["not found / empty"]
  Missing --> PromptHint["Prompt includes missing configured document notice"]
  PromptHint --> Model["Model runs novel-world-engine"]
  Model --> Ask["Call AskHuman<br/>ask for missing emotional core"]
  Ask --> Session["skill-session status=needs_input"]
```

此时不应该：

- 从 filesystem 猜 `novel_core.md` 路径；
- 用题材名瞎补一个内核；
- 直接输出正式 `world_engine`；
- 把缺信息的澄清文本当成最终文档写回。

正确路径是 `AskHuman`，人类补完后继续同一个 session。

## 模块边界

```mermaid
flowchart TD
  UI["Web UI / API Caller"] --> H["SkillSession HTTP Controller"]
  H --> S["skillsession.Manager"]
  H --> F["runtimeFactory"]
  F --> P["Project Context PG/Redis"]
  F --> M["Model Profile PG/Redis"]
  F --> R["Runtime"]
  S --> R
  R --> LLM["OpenAI-compatible Model"]
  R --> T["Runtime Built-in Tools"]
  T --> AH["AskHuman"]
  T --> PD["ProjectDocumentProvider"]
  PD --> PG["PostgreSQL"]
  PD --> FS["Filesystem Provider"]
  PD --> RC["Redis Cache"]
```

核心原则：

| 模块 | 负责 | 不负责 |
|---|---|---|
| `SkillSessionService` | HTTP 请求、解析 project/model、创建 runtime | 直接写文档、直接写 Redis |
| `skillsession.Manager` | session 状态、pending AskHuman、继续执行 | 理解小说业务、写项目资产 |
| `Runtime` | 调模型、暴露工具、执行工具、暂停/恢复 primitive | 多 skill 产品编排 |
| `AskHuman` | 请求人类补充信息 | 修改 PG/Redis/FS |
| `WriteProjectDocument` | 通过 provider 写项目文档 | 管理 session 状态 |
| `workflow` | 多 skill 顺序、产品流程、产物规则 | 自己实现多轮问答状态机、承接当前新小说能力主路径 |

## 当前 API

### 1. 创建 session

```http
POST /v1/skill-sessions
Content-Type: application/json
```

```json
{
  "project": "urban-rebirth",
  "model": "deepseek-flash",
  "skill_id": "novel-emotional-core",
  "input": "先做小说情感内核，缺少信息就问我，不要直接猜。",
  "arguments": {
    "document_kind": "novel_core"
  },
  "debug": true
}
```

返回有两种结果。

如果模型直接完成：

```json
{
  "session": {
    "id": "ss_20260429T011500.123456789",
    "status": "completed",
    "skill_id": "novel-emotional-core",
    "final_text": "..."
  }
}
```

如果模型需要人类补充：

```json
{
  "session": {
    "id": "ss_20260429T011500.123456789",
    "status": "needs_input",
    "skill_id": "novel-emotional-core",
    "ask_human": {
      "reason": "需要确认读者最核心的情绪回报。",
      "questions": [
        {
          "field": "payoff",
          "header": "爽点",
          "question": "主角最终给读者的核心情绪回报是什么？",
          "options": [
            {
              "label": "尊严",
              "description": "被重新看见、重新被尊重。"
            },
            {
              "label": "复仇",
              "description": "让曾经羞辱他的人付出代价。"
            }
          ]
        }
      ]
    }
  }
}
```

### 2. 查询 session

```http
GET /v1/skill-sessions/{id}
```

用途：

- 前端刷新页面后恢复当前状态。
- 查看是否还在等待人类输入。
- 查看历史 turns、run_id、run_dir。

注意：当前 session 存储是进程内存，服务重启后不能恢复。后续如果要多副本或重启恢复，需要把 `SkillExecutionState` 持久化到 PG 或 Redis。

### 3. 继续 session

```http
POST /v1/skill-sessions/{id}/turns
Content-Type: application/json
```

```json
{
  "input": "我选择尊严和重新被看见，不要简单暴富。",
  "answers": {
    "payoff": "尊严和重新被看见"
  },
  "notes": "主角是中年销售，被 KPI、债务、家庭责任压得喘不过气。"
}
```

服务端会把它转成 AskHuman 的 tool result：

```json
{
  "type": "human_answers",
  "answers": {
    "payoff": "尊严和重新被看见"
  },
  "notes": "主角是中年销售，被 KPI、债务、家庭责任压得喘不过气。"
}
```

然后追加到原 conversation 里继续请求模型。

## Start 流程

```mermaid
sequenceDiagram
  participant UI as Web UI
  participant HTTP as SkillSessionService
  participant Factory as runtimeFactory
  participant Session as skillsession.Manager
  participant Runtime as Runtime
  participant Model as Model

  UI->>HTTP: POST /v1/skill-sessions
  HTTP->>Factory: resolve project + model
  Factory->>Factory: load project context from PG/Redis
  Factory->>Factory: load model profile/default model from PG/Redis
  Factory->>Runtime: build runtime with ProjectDocs provider
  HTTP->>Session: Start(runtime, skill_id, input, arguments)
  Session->>Runtime: ExecuteSkillInteractive
  Runtime->>Runtime: compose skill prompt
  Runtime->>Model: chat/completions with tools
  Model-->>Runtime: final answer or tool call
  Runtime-->>Session: completed or needs_input
  Session-->>HTTP: session snapshot
  HTTP-->>UI: JSON response
```

关键点：

1. project 是必选的，因为 session 是项目上下文里的 skill 执行。
2. model 可以显式传，不传则使用数据库里的 default model。
3. runtime 会注入 `ProjectContext`，并挂上 `ProjectDocumentProvider`。
4. skill prompt 里包含 skill metadata、tooling hint、原始用户输入、当前任务。
5. `AskHuman` 是所有 skill execution 默认可用的内置工具。

## AskHuman 暂停流程

模型如果觉得缺少信息，不能自己瞎补，应调用：

```json
{
  "name": "AskHuman",
  "arguments": {
    "reason": "需要确认核心情绪回报。",
    "questions": [
      {
        "field": "payoff",
        "question": "主角最终要给读者什么情绪回报？",
        "options": [
          {"label": "尊严"},
          {"label": "复仇"}
        ]
      }
    ]
  }
}
```

runtime 收到这个 tool call 后不会继续调用模型，而是返回 `needs_input`。

保存的关键状态：

```text
SkillExecutionState
  - SkillID
  - SafeID
  - OriginalUserInput
  - InvocationArgs
  - CompiledPrompt
  - Conversation
  - NextRound
  - PendingToolCallID
  - ReadState
```

其中最重要的是：

- `Conversation`：保留模型此前看到和说过的上下文。
- `PendingToolCallID`：标记人类答案要回复给哪一个 AskHuman tool call。
- `NextRound`：继续时从下一轮模型调用开始。

这保证了“继续”不是重新开始，而是在同一条模型对话里补上工具结果。

## Continue 流程

```mermaid
sequenceDiagram
  participant UI as Web UI
  participant HTTP as SkillSessionService
  participant Session as skillsession.Manager
  participant Runtime as Runtime
  participant Model as Model

  UI->>HTTP: POST /v1/skill-sessions/{id}/turns
  HTTP->>Session: Continue(id, answers)
  Session->>Session: find pending SkillExecutionState
  Session->>Runtime: ContinueSkillInteractive(state, AskHumanAnswer)
  Runtime->>Runtime: append tool result to conversation
  Runtime->>Model: chat/completions with previous context
  Model-->>Runtime: final answer or next AskHuman/tool call
  Runtime-->>Session: completed or needs_input
  Session-->>HTTP: updated snapshot
  HTTP-->>UI: JSON response
```

runtime 实际续上的消息类似：

```text
assistant:
  tool_calls:
    - id: ask_1
      name: AskHuman
      arguments: ...

tool:
  tool_call_id: ask_1
  content:
    {
      "type": "human_answers",
      "answers": {...},
      "notes": "..."
    }
```

所以模型继续时能看到：

1. 原始 skill prompt；
2. 项目上下文；
3. 它刚才问过的问题；
4. 人类刚补充的答案；
5. 当前仍可用的工具。

## 文档写入流程

session 不写项目文档。模型如果最终要落项目资产，必须调用 runtime tool：

```text
WriteProjectDocument
```

完整链路：

```mermaid
flowchart TD
  M["Model calls WriteProjectDocument"] --> R["Runtime tool executor"]
  R --> P["runtimeProjectDocumentProvider"]
  P --> S["projectConfigStore.UpsertProjectDocument"]
  S --> PG["PostgreSQL project_documents"]
  S --> FS["Filesystem provider md/meta sync"]
  S --> RC["Redis project cache async set"]
  S --> OUT["Tool result body_bytes/synced"]
```

这和 workflow 的写文档链路是同一条：

```text
Runtime
  -> ProjectDocumentProvider
  -> projectConfigStore
  -> PostgreSQL
  -> filesystem provider
  -> Redis cache
```

因此 session 不会绕过 provider，也不会自己写本地文件。

## 和 Workflow 的关系

现在的口径已经简化：

```text
skill-session:
  当前唯一主执行面

workflow:
  历史入口 / 参考实现
```

当前建议很明确：

- 明确知道要跑哪个 skill，并且允许模型缺信息就问人：统一走 `/v1/skill-sessions`。
- 固定 workflow 先不继续扩，已有文档仅保留给历史实现和调试参考。
- 多步骤“一键初始化”暂不作为当前推进重点，避免在 AskHuman 状态机还没统一前再次分叉。

所以现阶段不再把“workflow 复用 session”当成近期强依赖，而是先把单 skill session 路径打稳。

## 前端应该怎么用

前端状态机可以很简单：

```text
idle
  -> start session
  -> running
  -> if completed: show final_text
  -> if needs_input: render ask_human.questions
  -> submit answers
  -> running
  -> repeat
```

UI 展示建议：

- `ask_human.reason` 放在问题组上方；
- 每个 `question.field` 作为表单字段 key；
- `options` 渲染为单选/多选；
- 允许用户补充 free text；
- 提交时同时发送：
  - `input`：人类自然语言总结；
  - `answers`：结构化字段；
  - `notes`：额外说明。

## 日志和调试

HTTP 日志：

- `skill_session.start.accepted`
- `skill_session.start.completed`
- `skill_session.continue.accepted`
- `skill_session.continue.completed`

runtime debug artifacts：

```text
runs/{run_id}/skill-calls/{skill_id}/compiled-prompt.md
runs/{run_id}/skill-calls/{skill_id}/allowed-tools.json
runs/{run_id}/skill-calls/{skill_id}/round-xx-assembly.json
runs/{run_id}/skill-calls/{skill_id}/round-xx-chat-request.json
runs/{run_id}/skill-calls/{skill_id}/round-xx-response.json
runs/{run_id}/skill-calls/{skill_id}/tools/AskHuman-{id}-request.json
```

这些文件用于解释：

- 模型当时看到了什么 prompt；
- 暴露了哪些工具；
- 模型为什么调用 AskHuman；
- 人类答案续回后模型怎么继续。

## 当前限制

1. session 当前是内存存储。
   - 服务重启会丢失 session。
   - 多副本部署时，同一个 session 必须回到同一个进程，否则找不到状态。

2. workflow 没有完全复用 skill-session。
   - 这在当前阶段是已知现状，但不是优先修复项。
   - 固定 workflow 暂时冻结，先不继续扩写这条链路。

3. session 没有独立过期策略。
   - 当前需要后续补 TTL、清理任务、取消接口。

4. session 没有持久化 transcript 到 PG。
   - 当前 snapshot 在内存里。
   - runstore 有 debug artifacts，但不是业务状态源。

## 后续演进建议

### 第一阶段：完善当前单 skill session

- 增加 `DELETE /v1/skill-sessions/{id}` 用于取消。
- 增加 session TTL。
- 增加 session list，仅用于调试或后台管理。
- 增加 verification script：创建 session -> AskHuman -> continue -> WriteProjectDocument。

### 第二阶段：session 持久化

建议表：

```text
skill_sessions
  id
  project_id
  skill_id
  status
  request
  arguments
  ask_human
  final_text
  runtime_state
  run_id
  run_dir
  created_at
  updated_at
  expires_at
```

如果使用 Redis：

```text
novelrt:skill_session:{id}
```

但原则仍然是：

- PG 更适合长期审计和恢复；
- Redis 更适合短期会话缓存；
- 不要让 Redis 成为唯一业务状态源，除非明确接受重启丢失。

### 第三阶段：workflow 复用 session（非当前范围）

这条线保留为远期参考，不纳入当前小说能力落地范围。

当前优先级仍然是：

- skill-session 稳定
- AskHuman 补问稳定
- WriteProjectDocument 落库稳定

## 验收标准

一个完整 session 流程应该满足：

- 能指定 project、model、skill_id 启动；
- 能拿到项目上下文；
- 能拿到数据库模型配置；
- 模型缺信息时能调用 AskHuman；
- 服务端返回 `needs_input` 而不是瞎生成；
- 人类答案能作为 tool result 回到同一 conversation；
- 模型能继续完成；
- 模型能通过 `WriteProjectDocument` 写项目文档；
- 文档写入走 PG -> filesystem provider -> Redis cache sync；
- runstore 能看到完整 prompt、tools、request、response、tool call 调试文件。
