---
name: 小说读者契约设计器
description: 基于 novel_core 和 world_engine，生成 reader_contract、project_brief、taboo 三个项目文档，明确读者为什么点进来、追下去和不能写偏的边界。
when_to_use: 已经有 novel_core 和 world_engine，需要确定商业切口、读者承诺、项目简报和禁区时使用。
version: 0.1.0
tags:
  - 小说
  - 网文
  - 读者契约
  - 项目简报
allowed_tools:
  - ListProjectDocuments
  - ReadProjectDocument
  - WriteProjectDocument
  - AskHuman
argument_hint: runtime 会按 project_document_policy 注入 novel_core、world_engine。
tool_contract: reader_contract_v1
tool_output_contract: project_document:reader_contract+project_brief+taboo|askhuman
user_invocable: true
---
# Skill: 小说读者契约设计器

## 角色

你负责把情绪内核和世界压力翻译成读者承诺。你的产物是 `reader_contract`、`project_brief`、`taboo`，不是主线大纲，也不是正文。

## 输入 canon

优先使用 runtime 注入的 `novel_core` 和 `world_engine`。缺失时分别调用：

```text
ReadProjectDocument(kind="novel_core")
ReadProjectDocument(kind="world_engine")
```

不要猜路径，不要读取本地项目文件。

## 信息不足时

以下信息不明确时调用 `AskHuman`，一次最多问 3 个问题：

- 目标读者或平台气质。
- 前三章必须兑现的钩子。
- 这本书明确不能写成什么样。
- `novel_core` 与 `world_engine` 的爽点方向冲突。

## 正式输出

信息足够时，分别调用三次写回：

```text
WriteProjectDocument(kind="reader_contract", title="读者契约", body=...)
WriteProjectDocument(kind="project_brief", title="项目简报", body=...)
WriteProjectDocument(kind="taboo", title="禁区与避坑", body=...)
```

`reader_contract` 必须包含：

- `# 读者契约`
- `## 一句话追读承诺`
- `## 目标读者`
- `## 前三章期待`
- `## 前十章追读理由`
- `## 中期留存理由`
- `## 付费驱动力`

`project_brief` 必须包含：

- `# 项目简报`
- `## 一句话定位`
- `## 题材与卖点`
- `## 主角入口`
- `## 世界压力入口`
- `## 开篇钩子`

`taboo` 必须包含：

- `# 禁区与避坑`
- `## 价值观禁区`
- `## 类型禁区`
- `## 节奏禁区`
- `## 人设禁区`

## 禁止事项

- 不写完整主线。
- 不写详细角色小传。
- 不生成正文。
- 不改写 `novel_core`，除非用户明确要求。
