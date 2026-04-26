# 文档总览

这是当前 `backend/docs` 的主入口。

如果你只想先知道“现在整体做到哪了”，先看这份。

## 当前整体进度

截至 `2026-04-27`，当前进度可以分成几层：

### Tool Layer

- 已完成：`tool_search -> activation -> retained pool -> skill execution`
- 已完成：`tool_reference_like`
- 已完成：skill-specific `tool_input_schema`
- 已完成：skill executor 内部 `Read / Write / Edit / Glob / Bash / PowerShell`
- 已完成：router / skill executor request assembly 落盘
- 已完成：默认文档落地目录 [docs/08-generated-drafts](../../docs/08-generated-drafts/README.md)
- 已完成：DeepSeek 调试链路与 `-debug` 落盘
- 待做：未 discover 先误调用时的自修复提示
- 暂缓：跨轮 discovered memory，等 mem 系统稳定后推进

### Skill Layer

- 已完成：`webnovel-opening-sniper` 的基础 tool contract
- 已完成：`opening_v1` 的初步输出约束
- 已完成：`novel-idea-bootstrap` 的 `bootstrap_v1` 起盘与澄清约束
- 待做：`outline_v1`
- 待做：`rewrite_v1`
- 未开始：大规模小说 skill 库扩展

### Runtime 体验层

- 已完成：skill 成功后 direct return
- 已完成：debug 模式保存 router / skill executor 请求与响应
- 已完成：DeepSeek `reasoning_content` 兼容
- 已完成：DeepSeek token ceiling 保护

## 当前推荐阅读顺序

1. [CONFIG.md](CONFIG.md)
2. [FULL_REQUEST_FLOW.md](FULL_REQUEST_FLOW.md)
3. [IMPLEMENTATION.md](IMPLEMENTATION.md)
4. [FILE_TOOLS_AND_DOCUMENT_OUTPUT.md](FILE_TOOLS_AND_DOCUMENT_OUTPUT.md)
5. [SEARCH_PIPELINE.md](SEARCH_PIPELINE.md)
6. [TOOL_SEARCH_VS_NOVELCODE.md](TOOL_SEARCH_VS_NOVELCODE.md)
7. [DISCOVERED_SKILL_POOL.md](DISCOVERED_SKILL_POOL.md)
8. [SKILL_OUTPUT_CONTRACTS.md](SKILL_OUTPUT_CONTRACTS.md)
9. [NEXT_STEPS.md](NEXT_STEPS.md)

## 各文档现在各自负责什么

- `CONFIG.md`
  - 当前真实配置、环境变量覆盖和命令行参数
- `IMPLEMENTATION.md`
  - 当前运行时架构和调用链
- `FULL_REQUEST_FLOW.md`
  - 从用户输入到最终模型请求的完整 assembly 流程
- `FILE_TOOLS_AND_DOCUMENT_OUTPUT.md`
  - `Read / Write / Edit / Glob` 以及文档落地策略
- `SEARCH_PIPELINE.md`
  - `tool_search` 的搜索、排序、激活和 schema 暴露机制
- `TOOL_SEARCH_VS_NOVELCODE.md`
  - 和本地 `novelcode` 的真实差距
- `DISCOVERED_SKILL_POOL.md`
  - retained pool 为什么重要
- `SKILL_OUTPUT_CONTRACTS.md`
  - 为什么 tool 层稳定后还要做 skill 输出 contract
- `NEXT_STEPS.md`
  - 当前真实 roadmap
- `CHANGELOG-v0.2.md`
  - 历史归档，不代表当前状态

## 当前最关键的判断

现在项目不该再被理解成：

```text
一个简单的 skill 检索器
```

而应该理解成：

```text
一个正在往 novelcode 风格推进的 Go 版 deferred skill runtime
```

只是现在仍然处在：

- tool 层已基本成型
- memory 层还没开始接入
- 小说 skill 库还没大规模扩展
- 文件工具层已经具备“把 skill 产物真实落到文档”的基础能力
- 请求装配层已经具备“把每轮真实发给模型的内容逐轮落盘”的基础能力

这个阶段。
