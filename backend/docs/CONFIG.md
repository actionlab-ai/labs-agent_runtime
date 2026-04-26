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
```

## 2. `model` 配置项

- `provider`
  - 当前实现里主要是 OpenAI-compatible 调用链
- `id`
  - 请求 `/chat/completions` 时发送的模型名
- `base_url`
  - OpenAI-compatible 接口地址
- `api_key_env`
  - 读取 API Key 的环境变量名
  - 注意这里填的是环境变量名，不是裸密钥
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

## 4. Debug 配置

调试时可用：

- [config.deepseek-debug.yaml](../config.deepseek-debug.yaml)

这份配置主要用于：

- 开启 `-debug` 时保存详细请求/响应轨迹
- 做大 token 上限实验
- 排查模型兼容性问题

注意：

- skill 执行阶段对 DeepSeek 仍然会做运行时保护，不会真的盲目把超上限 token 打出去
- skill 如果使用文件工具，默认会把文档写到 `document_output_dir`

## 5. 环境变量覆盖

配置系统使用 `NOVEL_` 前缀，`.` 会映射成 `_`。

示例：

```bash
export DEEPSEEK_API_KEY=your_real_key
export NOVEL_MODEL_ID=deepseek-v4-flash
export NOVEL_MODEL_TIMEOUT_SECONDS=180
export NOVEL_RUNTIME_MAX_RETAINED_SKILLS=8
```

## 6. 命令行参数

入口在：
[main.go](../cmd/novelrt/main.go)

当前支持：

- `-config`
  - 配置文件路径
- `-input`
  - 用户输入
- `-dry-run`
  - 只跑本地检索，不请求模型
- `-list-skills`
  - 列出已加载的 skill metadata
- `-debug`
  - 保存详细请求、错误与中间产物
