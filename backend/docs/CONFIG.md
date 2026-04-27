# 配置说明

本文档描述当前 `backend/config.yaml` 与 `backend/config.deepseek-debug.yaml` 所对应的真实配置语义。

## 1. 当前正式配置

当前默认配置文件是：

- [config.yaml](../config.yaml)

当前默认模型配置是：

```yaml
model:
  provider: openai_compatible
  id: deepseek-v4-flash
  base_url: https://api.deepseek.com
  api_key: ""
  api_key_env: DEEPSEEK_API_KEY
  context_window: 1000000
  max_output_tokens: 32768
  temperature: 0.7
  timeout_seconds: 180
runtime:
  skills_dir: ./skills
  runs_dir: ./runs
  workspace_root: ..
  document_output_dir: ../docs/08-generated-drafts
  project_id: ""
  project_root: ""
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
  migrations_dir: ./internal/db/migrations
  auto_migrate: true
```

## 2. `model` 配置项

- `provider`
  - 当前实现里主要是 OpenAI-compatible 调用链
- `id`
  - 请求 `/chat/completions` 时发送的模型名
- `base_url`
  - OpenAI-compatible 接口地址
- `api_key`
  - 直接使用的模型 API Key
  - 优先级高于 `api_key_env`
- `api_key_env`
  - 兼容旧配置的环境变量名兜底；新建模型配置时优先使用 `api_key`
- `context_window`
  - 模型上下文窗口上限
- `max_output_tokens`
  - 单次请求的输出 token 上限
- `temperature`
  - 采样温度
- `timeout_seconds`
  - HTTP 超时

## 3. `runtime` 配置项

- `skills_dir`
  - skill 扫描目录
- `runs_dir`
  - run 产物输出目录
- `workspace_root`
  - skill 文件工具允许读写的工作区根目录
- `document_output_dir`
  - 默认文档落地目录
- `project_id`
  - 可选。指定 HTTP `POST /v1/runs` 在请求体未传 `project` 时的默认项目 ID
- `project_root`
  - 兼容旧文件项目上下文的字段。PG 项目模式下通常为空
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

## 4. `database` 配置项

- `url`
  - PostgreSQL 连接串
  - 不建议写进配置文件；优先用 `DATABASE_URL` 或 `NOVEL_DATABASE_URL`
- `migrations_dir`
  - golang-migrate 迁移目录
- `auto_migrate`
  - HTTP 服务启动时是否自动执行 migration

## 5. Debug 配置

调试时可用：

- [config.deepseek-debug.yaml](../config.deepseek-debug.yaml)

这份配置主要用于：

- 开启 `-debug` 时保存详细请求/响应轨迹
- 做大 token 上限实验
- 排查模型兼容性问题

注意：

- skill 执行阶段对 DeepSeek 仍然会做运行时保护，不会真的盲目把超上限 token 打出去
- skill 如果使用文件工具，默认会把文档写到 `document_output_dir`

## 6. 环境变量覆盖

配置系统使用 `NOVEL_` 前缀，`.` 会映射成 `_`。

示例：

```bash
export DEEPSEEK_API_KEY=your_real_key
export DATABASE_URL=postgres://postgres:***@host:30432/codefly?sslmode=disable
export NOVEL_MODEL_ID=deepseek-v4-flash
export NOVEL_MODEL_TIMEOUT_SECONDS=180
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
- `POST /v1/runs`
  - 基于输入执行一次 runtime run
  - 请求体：`{"project":"都市异能悬疑","model":"deepseek-flash","input":"继续完善世界观","dry_run":false}`
  - `project` 可省略；省略时使用配置里的 `runtime.project_id`
  - `model` 可省略；省略时使用配置文件里的 `model`
