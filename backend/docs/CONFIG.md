# 配置说明

本文档描述当前 `backend/config.yaml` 的真实配置语义。

## 1. 当前正式配置

当前默认配置文件是：

- [config.yaml](../config.yaml)

当前默认服务配置是：

```yaml
runtime:
  skills_dir: ./skills
  runs_dir: ./runs
  workspace_root: ..
  document_output_dir: ../docs/08-generated-drafts
  max_tool_rounds: 4
  max_skill_tool_rounds: 6
  force_tool_search_first: true
  return_skill_output_direct: true
  fallback_skill_search: true
  fallback_min_score: 0.18
  max_activated_skills: 3
  max_retained_skills: 6
  activation_min_score: 0.18
  activation_score_ratio: 0.55
database:
  url: ""
  host: 36.138.61.152
  port: 30432
  name: codefly
  user: postgres
  password: ""
  sslmode: disable
  connect_timeout_seconds: 5
  migrations_dir: ./internal/db/migrations
  auto_migrate: true
redis:
  enabled: true
  required: false
  mode: cluster
  addrs:
    - 36.138.61.152:30001
    - 36.138.61.152:30002
    - 36.138.61.152:30003
    - 36.138.61.152:30004
    - 36.138.61.152:30005
    - 36.138.61.152:30006
  password: ""
  key_prefix: novelrt
  ttl_seconds: 300
logging:
  level: debug
  encoding: console
  development: true
```

## 2. `runtime` 配置项

- `skills_dir`
  - skill 扫描目录
- `runs_dir`
  - run 产物输出目录
- `workspace_root`
  - skill 文件工具允许读写的工作区根目录
- `document_output_dir`
  - 默认文档落地目录
- `max_tool_rounds`
  - router tool calling 最大轮数
- `max_skill_tool_rounds`
  - skill executor 内部文件工具回合上限
- `force_tool_search_first`
  - 首轮是否只暴露 `tool_search`
- `return_skill_output_direct`
  - skill 执行成功后是否直接把 skill 输出返回用户
- `fallback_skill_search`
  - 模型没走 tool calling 时，是否允许本地检索兜底
- `fallback_min_score`
  - 触发 fallback 的最低命中分
- `max_activated_skills`
  - 单次 search 的激活窗口大小上限
- `max_retained_skills`
  - retained discovered-skill pool 的总容量
- `activation_min_score`
  - 激活窗口的最低分地板
- `activation_score_ratio`
  - 激活窗口相对首个命中分数的比例阈值

## 3. `database` 配置项

- `url`
  - PostgreSQL 连接串
  - 不建议写进配置文件；优先用 `DATABASE_URL` 或 `NOVEL_DATABASE_URL`
- `host`
  - PostgreSQL 主机名或 IP
- `port`
  - PostgreSQL 端口
- `name`
  - 数据库名
- `user`
  - 用户名
- `password`
  - 密码
- `sslmode`
  - 连接参数，例如 `disable`、`require`
- `connect_timeout_seconds`
  - 建连超时，单位秒
- `migrations_dir`
  - golang-migrate 迁移目录
- `auto_migrate`
  - HTTP 服务启动时是否自动执行 migration

## 4. `redis` 配置项

Redis Cluster 只做共享缓存，不是模型配置的 source of truth。模型配置、默认模型设置仍然以 PostgreSQL 为准。

- `enabled`
  - 是否启用 Redis Cluster 缓存
- `required`
  - 是否要求 Redis 启动成功。默认 `false`，Redis 不可用时降级回 PostgreSQL；生产环境如果希望 Redis 不通就拒绝启动，可以设为 `true`
- `mode`
  - Redis 连接模式，支持 `cluster` 和 `standalone`。当前集群使用 `cluster`；后续切单点时改成 `standalone` 并只保留一个地址即可
- `addrs`
  - Redis 地址列表。`cluster` 模式填写多个节点；`standalone` 模式使用第一个地址
- `password`
  - Redis 密码；不建议写进配置文件，优先用 `REDIS_PASSWORD` 或 `NOVEL_REDIS_PASSWORD`
- `key_prefix`
  - 缓存 key 前缀，默认 `novelrt`
- `ttl_seconds`
  - 缓存过期时间，默认 300 秒

当前缓存内容：

- `projects`
- `model_profiles`
- `app_settings.default_model_id`

写入项目、模型配置或默认模型时，流程是先写 PostgreSQL，再由后台 goroutine 同步 Redis。读取项目、模型或默认模型时，流程是先读 Redis，未命中再回源 PostgreSQL 并立即回填 Redis。

项目存储定位现在属于 `projects` 表字段：

- `storage_provider`: `filesystem` 或 `s3`
- `storage_bucket`: S3 bucket，`filesystem` 模式可为空
- `storage_prefix`: 本地文件夹名或 S3 key prefix

## 5. `logging` 配置项

日志使用 `zap`，启动后所有 HTTP 请求都会带 `request_id`。你查一个接口时，按同一个 `request_id` 过滤，就能看到它走了哪些步骤。

- `level`
  - 支持 `debug`、`info`、`warn`、`error`
  - 本地排查建议 `debug`
- `encoding`
  - 支持 `console` 和 `json`
  - 本地看日志用 `console` 更直观；生产采集建议 `json`
- `development`
  - 本地可以设为 `true`，日志更适合直接看
  - 生产建议 `false`

当前会记录：

- `http.request.start` / `http.request.completed`
- `project.cache.hit` / `project.cache.miss` / `project.pg.get`
- `model.cache.hit` / `model.cache.miss` / `model.pg.get`
- `default_model.cache.hit` / `default_model.pg.get`
- `run.request.accepted` / `run.model.selected` / `run.project_context.loaded`
- `cache.sync.start` / `cache.sync.done` / `cache.sync.failed`

不会记录 API Key、用户输入正文和模型完整响应，只记录长度、ID、状态、耗时等排查信息。

### `filesystem` 项目落盘

当项目的 `storage_provider=filesystem` 时：

- 创建项目会自动创建 `${runtime.workspace_root}/projects/<storage_prefix>/`
- 项目目录下会生成 `meta.json`
- 文档写入 `/v1/projects/:id/documents/:kind` 时，会同步到 `documents/<kind>.md`
- 同时会生成 `documents/<kind>.meta.json`
- 项目级 `meta.json` 会记录 `project_id`、`storage_*`、`document_count` 和 `document_kinds`

### skill 内置项目文档工具

项目模式下，skill executor 会额外暴露 3 个项目级工具：

- `ListProjectDocuments`
- `ReadProjectDocument`
- `WriteProjectDocument`

这些工具不直接读写本地路径，而是调用运行时注入的 `ProjectDocumentProvider`。当前 provider 的 HTTP/PG 实现最终会写入 PostgreSQL，并在 `filesystem` 项目上同步一份文件投影；后续切换到 S3 时，只需要替换 provider 实现，业务 skill 不需要关心底层是文件夹还是 bucket。

普通 `Read / Write / Edit / Glob` 仍然是 workspace 文件工具，适合调试文件、临时草稿和非项目状态文件；小说世界观、情感内核、角色卡、当前状态这类长期项目资料应该优先走项目文档工具。

## 6. 环境变量覆盖

配置系统使用 `NOVEL_` 前缀，`.` 会映射成 `_`。

示例：

```bash
export DATABASE_HOST=127.0.0.1
export DATABASE_PORT=5432
export DATABASE_NAME=codefly
export DATABASE_USER=postgres
export DATABASE_PASSWORD=your_real_password
export REDIS_PASSWORD=your_real_redis_password
export NOVEL_RUNTIME_MAX_RETAINED_SKILLS=8
```

## 7. 服务启动参数

入口在：
[main.go](../cmd/novelrt/main.go)

当前支持：

- `-config`
  - 配置文件路径
- `-debug`
  - 为 HTTP 请求保存详细请求、错误与中间产物
- `-addr`
  - HTTP API 监听地址，默认 `:8080`

## 8. HTTP API

启动：

```bash
./novelrt -config config.yaml -addr :8080
```

接口：

- `GET /healthz`
  - 健康检查
- `GET /v1/skills`
  - 返回当前加载到的 skill metadata
- `GET /v1/projects`
  - 列出项目
- `POST /v1/projects`
  - 创建小说项目
  - 请求体：`{"name":"都市异能悬疑"}`
- `GET /v1/projects/:id`
  - 查看项目
- `PATCH /v1/projects/:id`
  - 更新项目信息
- `DELETE /v1/projects/:id`
  - 软删除项目
- `GET /v1/projects/:id/documents`
  - 查看项目文档
- `PUT /v1/projects/:id/documents/:kind`
  - 写入或更新项目文档，例如 `world_rules`、`power_system`、`mainline`、`current_state`
- `GET /v1/models`
  - 列出模型配置
- `POST /v1/models`
  - 创建模型配置；直接保存 `api_key`，响应只返回 `api_key_set`，不会回显裸密钥
  - 请求体：`{"id":"deepseek-flash","name":"DeepSeek Flash","model_id":"deepseek-v4-flash","base_url":"https://api.deepseek.com","api_key":"sk-your-key"}`
- `GET /v1/models/:id`
  - 查看模型配置
- `PATCH /v1/models/:id`
  - 更新模型配置
- `DELETE /v1/models/:id`
  - 软删除模型配置
  - 如果它是当前默认模型，会拒绝删除，必须先切换或清空默认模型
- `GET /v1/settings/default-model`
  - 查看数据库里的默认模型设置
- `PUT /v1/settings/default-model`
  - 设置数据库默认模型
  - 请求体：`{"model":"deepseek-flash"}`
- `DELETE /v1/settings/default-model`
  - 清空数据库默认模型
- `POST /v1/runs`
  - 基于输入执行一次 runtime run
  - 请求体：`{"project":"都市异能悬疑","model":"deepseek-flash","input":"继续完善世界观","dry_run":false}`
  - `project` 可省略；省略时本次运行不绑定项目上下文。Web 界面应在打开某个项目后显式传当前项目 ID
  - `model` 默认必传；只有数据库里设置了默认模型时才可以省略
