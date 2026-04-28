# Novel Agent Runtime

这是当前 `backend` 的总入口说明。

这套 runtime 现在已经不是最早的“`tool_search + skill_call` 两段式分发器”了，而是一套更接近 `novelcode` 思路的 `deferred skill discovery` 运行时：

1. 启动时只加载 skill metadata，不加载完整 `SKILL.md` 正文。
2. 首轮默认只暴露 `tool_search`。
3. `tool_search` 会返回：
   - 查询解析结果
   - 排序后的 skill 命中
   - `tool_reference_like` 激活对象
   - 本轮激活窗口
   - retained discovered-skill pool
4. 下一轮只暴露：
   - `tool_search`
   - retained skill tools
   - `skill_call` 兼容兜底工具
5. 如果 skill 在 frontmatter 里声明了 `tool_input_schema`，激活后的 skill tool 会直接暴露结构化输入 schema，而不是只给一个 `task:string`。
6. skill executor 内部现在也能走本地工具回合：
   - `Read`
   - `Write`
   - `Edit`
   - `Glob`
   - `Bash`
   - `PowerShell`
   - `ListProjectDocuments`
   - `ReadProjectDocument`
   - `WriteProjectDocument`
7. skill 真执行成功后，默认直接把 skill 输出返回给用户；如果 skill 已把内容落成文档，则返回的是简短文件摘要而不是整篇正文。
8. router 和 skill executor 每一轮在真正发模型前，都会先生成 request assembly，并把预处理后的 messages / tools / prompt / final chat request 落盘。

## 当前状态

截至 `2026-04-27`，当前阶段可以概括为：

- tool layer 主链已经跑通
- DeepSeek 实测调通
- `tool_search -> activation -> retained pool -> structured skill tool` 已落地
- `Read / Write / Edit / Glob / Bash / PowerShell` 已接入 skill executor
- `ListProjectDocuments / ReadProjectDocument / WriteProjectDocument` 已作为项目级内置工具接入 skill executor；它们只调用项目文档 provider，不直接操作本地文件路径
- `router / skill executor assembly -> final request -> response analysis` 已落地
- 小说产物默认可以落到 [docs/08-generated-drafts](../docs/08-generated-drafts/README.md)
- `opening_v1` 输出 contract 已开始落地
- `bootstrap_v1` 创意起盘 skill 已接入，可先做世界观 / 金手指 / 主角起点澄清与初稿
- 跨轮 discovered memory 还没做，等 mem 系统稳定后再推进
- 小说 skill 库的大规模扩展还没开始，当前重心仍然是把工具调用层磨到很稳

更详细的进度和文档导航见：
[docs/README.md](docs/README.md)

## 启动服务

```bash
cd backend
go build -o novelrt ./cmd/novelrt
```

```bash
export DATABASE_PASSWORD=your_real_database_password
export REDIS_PASSWORD=your_real_redis_password
./novelrt -config config.yaml -addr :8080
```

调试模式：

```bash
./novelrt -config config.yaml -addr :8080 -debug
```

常用接口：

```bash
curl -X POST http://localhost:8080/v1/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"都市异能悬疑"}'

curl http://localhost:8080/v1/projects

curl -X POST http://localhost:8080/v1/models \
  -H "Content-Type: application/json" \
  -d '{"id":"deepseek-flash","name":"DeepSeek Flash","model_id":"deepseek-v4-flash","base_url":"https://api.deepseek.com","api_key":"sk-your-key","max_output_tokens":8192}'

curl -X PUT http://localhost:8080/v1/projects/都市异能悬疑/documents/world_rules \
  -H "Content-Type: application/json" \
  -d '{"title":"世界规则","body":"遗物会保留死者最后三分钟的执念。"}'

curl -X POST http://localhost:8080/v1/runs \
  -H "Content-Type: application/json" \
  -d '{"project":"都市异能悬疑","model":"deepseek-flash","input":"继续完善这个项目的世界观和能力体系"}'
```

## 新的落盘产物

现在一次真实 run 里，除了原来的 raw request / raw response，还会多出：

- `router/round-XX-assembly.json`
- `router/round-XX-assembly.md`
- `router/round-XX-response-analysis.json`
- `skill-calls/<skill>/round-XX-assembly.json`
- `skill-calls/<skill>/round-XX-assembly.md`
- `skill-calls/<skill>/round-XX-response-analysis.json`

如果你想直接看“这一轮模型到底看到了什么、为什么会调这个工具”，先看这些文件。

## 配置提醒

- 模型管理接口直接传 `api_key`；响应只返回 `api_key_set`，不会回显裸密钥
- 配置文件不再携带模型的 `base_url / model_id / api_key`；这些都走 `/v1/models` 存数据库
- `/v1/runs` 默认需要传 `model`；如果想省略，需要先通过 `/v1/settings/default-model` 在数据库里设置默认模型
- 数据库连接既支持 `database.url`，也支持 `database.host / port / name / user / password / sslmode`
- Redis Cluster 当前缓存项目元数据、模型配置和默认模型设置；PostgreSQL 仍然是主存储，写接口会先写 PG 再由后台 goroutine 同步 Redis
- Redis 客户端支持 `cluster` 和 `standalone` 两种模式；以后切单点只改 `redis.mode` 和 `redis.addrs`
- 日志使用 zap；本地默认 `logging.level: debug`、`logging.encoding: console`，每个 API 请求都会带 `request_id`，可以串起 HTTP、Redis、PG、run 的调用链
- 进程内不缓存模型配置，所以模型变更不需要通知内存；下一次 `/v1/runs` 会从 Redis/PG 读取最新配置
- 当前只保留 HTTP 服务入口；项目管理、模型管理和运行都走 API
- 当前默认 workspace root 是仓库根目录，默认文档输出目录是 [docs/08-generated-drafts](../docs/08-generated-drafts/README.md)
- 项目信息、项目文档和 run 记录现在以 PostgreSQL 为主存储
- 项目记录包含 `storage_provider / storage_bucket / storage_prefix`，用于表达本地文件夹或 S3 bucket/prefix；后续换存储后端不需要改 HTTP 合同
- `filesystem` 项目会额外落盘到 `${runtime.workspace_root}/projects/<storage_prefix>/`，目录下包含 `meta.json`、`documents/*.md` 和对应的 `.meta.json`
- `document_output_dir` 仍用于 skill 文件工具输出草稿或调试文档，但不再是项目状态的 source of truth
- 项目模式下 runtime 会通过 `ContextProvider` 把 PG 中的 `project_documents` 注入到 skill 上下文里；后续世界观、开篇、章节等操作都会以这些数据库记录作为当前项目基线
- skill 定义当前仍来自本地 `skills/`，通过 `SkillProvider` 暴露；后续可以扩展 DB/Git/对象存储来源，不影响 HTTP 层
- 多 skill 编排的边界已经落在 `WorkflowRunner`，下一步会在它之上做项目初始化和章节创作 workflow

## 推荐阅读顺序

1. [docs/README.md](docs/README.md)
2. [docs/CONFIG.md](docs/CONFIG.md)
3. [docs/SKILL_WORKFLOW_RUNTIME.md](docs/SKILL_WORKFLOW_RUNTIME.md)
4. [docs/FULL_REQUEST_FLOW.md](docs/FULL_REQUEST_FLOW.md)
5. [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md)
6. [docs/FILE_TOOLS_AND_DOCUMENT_OUTPUT.md](docs/FILE_TOOLS_AND_DOCUMENT_OUTPUT.md)
7. [docs/SEARCH_PIPELINE.md](docs/SEARCH_PIPELINE.md)
8. [docs/TOOL_SEARCH_VS_NOVELCODE.md](docs/TOOL_SEARCH_VS_NOVELCODE.md)
9. [docs/NEXT_STEPS.md](docs/NEXT_STEPS.md)
