# Skill / Workflow Runtime 边界

这份文档说明当前先落地的 4 个接口概念，以及它们为什么能支撑后续“项目 -> 世界观/角色/章节 -> skill workflow”的产品主链路。

## 结论

当前不要把所有东西都塞进数据库，也不要继续让 HTTP handler 直接拼 prompt。

更稳的范式是：

- `SkillProvider`：负责发现和加载 skill。当前第一版读取本地 `skills/`，后续可以扩展为 DB、对象存储、Git 仓库或混合来源。
- `ContextProvider`：负责为一个项目构建“可喂给模型的上下文包”。当前第一版从 PostgreSQL 的 `projects` 和 `project_documents` 生成。
- `SkillRunner`：负责执行一个具体 skill。它不关心 skill 来自哪里，也不关心项目文档存在哪里，只消费 `SkillInput`。
- `WorkflowRunner`：负责编排多个 skill 步骤。第一版是顺序执行，后续可以扩展条件分支、状态机、人工确认节点。

代码入口在：

- [internal/workflow/interfaces.go](../internal/workflow/interfaces.go)
- [internal/workflow/providers.go](../internal/workflow/providers.go)

## 为什么不是全部 DB 化

skill 本质上更像“能力定义”和“可版本化流程资产”，包括：

- 什么时候使用
- 输入 schema
- 输出 contract
- 具体执行指令
- 允许的工具边界

这些内容适合先放本地文件，因为它们需要 code review、版本管理、回滚和批量迭代。数据库适合存业务状态：

- 项目信息
- 项目文档，比如世界观、角色卡、当前状态、章节摘要
- 模型配置和默认模型
- run 记录和 workflow 运行记录

Redis Cluster 当前只承担共享缓存角色：

- 缓存项目元数据和存储定位
- 缓存模型配置
- 缓存默认模型设置
- 让多实例读取同一份热配置

它不是主存储。写配置时先写 PostgreSQL，再更新 Redis；读运行配置时先读 Redis，未命中再回源 PostgreSQL。进程内不保存模型配置，所以不存在“配置变动后谁通知内存”的问题。

所以当前边界是：

```text
本地 skill 文件        PostgreSQL 项目数据
      |                       |
      v                       v
SkillProvider          ContextProvider
      |                       |
      +----------+------------+
                 v
             SkillRunner
                 |
                 v
            WorkflowRunner
```

## LLM 怎么知道项目下有什么文档

不是让 LLM 自己猜数据库里有什么文档。

正确做法是运行时先通过 `ContextProvider` 生成 `ContextPack`，里面包含：

- 项目 ID、名称、描述、状态
- 项目下已经保存的文档列表
- 拼装后的上下文文本
- 当前用户请求

然后 `SkillRunner` 把这包上下文注入 runtime。LLM 看到的是明确整理后的项目上下文，而不是一个模糊的“你自己去数据库找”。

第一版仍然是全量拼入项目文档。后续文档变多时，可以把 `ContextProvider` 升级为检索式：

- 按 `kind` 选择，例如 `world_rules`、`character_cast`、`current_state`
- 按 workflow step 选择，例如世界观 skill 只拿世界规则和当前设定
- 接入向量检索或关键词检索
- 给每个文档加优先级、更新时间和 token 预算

这样升级时只换 `ContextProvider`，不用重写 HTTP API 和 skill runtime。

## 4 个接口的职责

### SkillProvider

负责“有哪些能力可用”和“加载某个能力的完整定义”。

当前实现：

- `LocalSkillProvider`
- 从 `runtime.skills_dir` 扫描本地 `SKILL.md`
- `/v1/skills` 已经通过它返回 skill 列表

后续可以扩展：

- DB skill registry
- Git-backed skill registry
- 多来源聚合 provider

### ContextProvider

负责“某个项目当前应该给模型看的上下文是什么”。

当前实现：

- `PostgresContextProvider`
- 从 PostgreSQL 读取 `projects` 和 `project_documents`
- `/v1/runs` 已经通过它构建项目上下文

后续重点会放在这里升级上下文选择策略，而不是让 handler 拼字符串。

### SkillRunner

负责执行一个 skill。

当前实现：

- `RuntimeSkillRunner`
- 把 `ContextPack.Text` 临时注入 `runtime.ProjectContext`
- 调用现有 `runtime.ExecuteSkill`

它是后续“问答、创作、初始化世界观、生成章节”的统一执行入口。

### WorkflowRunner

负责把多个 skill 串成产品流程。

当前实现：

- `SequentialWorkflowRunner`
- 按步骤顺序调用 `SkillRunner`
- 合并 workflow 级参数和 step 级参数

后续可以演进为：

- `project_bootstrap_workflow`：世界观 -> 角色 -> 主线 -> 当前状态
- `chapter_draft_workflow`：上下文包 -> 章节 brief -> 正文 -> continuity audit -> snapshot
- `retcon_workflow`：影响面分析 -> 修改建议 -> 文档回写

## 当前 HTTP 层怎么用

目前已接入两个点：

- `GET /v1/skills`
  - 通过 `SkillProvider` 返回本地 skill metadata。
- `POST /v1/runs`
  - 通过 `ContextProvider` 加载项目上下文，再执行现有 runtime。

这次没有急着新增 workflow HTTP API，是因为先把内部边界定住更重要。下一步更适合新增：

- `POST /v1/workflows/project-bootstrap`
- `POST /v1/workflows/chapter-draft`
- `GET /v1/workflows/runs/:id`

这些接口会调用 `WorkflowRunner`，而不是让前端自己决定调用哪些 skill。

## 下一步建议

优先推进两件事：

1. 给 workflow run 建表，记录每个 step 的输入、输出、状态和错误。
2. 落第一个真实 workflow：`project-bootstrap`，把项目创建后的世界观、角色、主线、当前状态初始化成结构化项目文档。

这条路线可以继续复用现有 skill，不会把现在的本地 skill 体系推倒重来。

## Project Storage Provider Boundary

Project storage is now represented as metadata on the `projects` table:

- `storage_provider`: currently `filesystem` or `s3`.
- `storage_bucket`: S3 bucket name when provider is `s3`.
- `storage_prefix`: local folder name or S3 key prefix.

`ContextProvider` and `WorkflowRunner` should consume project metadata through the provider boundary instead of hard-coding local paths. That lets the first product workflow use local files now and move selected artifacts to S3 later without changing the HTTP contract or skill execution interface.

Skill execution has the same boundary. Business skills should not call raw file tools when they are creating durable project state. In project mode the runtime injects provider-backed tools:

- `ListProjectDocuments`
- `ReadProjectDocument`
- `WriteProjectDocument`

The skill only describes the document it wants to create or update, for example `kind=novel_core`; the provider decides whether that becomes a row plus a filesystem projection, an S3 object, or another storage backend later. This keeps webnovel skills focused on writing logic and keeps storage behavior in one reusable layer.

Cache rules:

- Project/model/default reads try Redis first and fall back to PostgreSQL.
- Cache misses are backfilled immediately after PostgreSQL reads.
- PostgreSQL mutations start a background goroutine to sync Redis.
- PostgreSQL remains the source of truth; Redis is not a primary store.
