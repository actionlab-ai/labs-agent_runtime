---
name: 小说开篇执行包设计器
description: 基于全部开书资产，生成 current_state 项目文档，明确第一章开场、初始状态、已确认 canon、禁止事项和下一步写作约束。
when_to_use: 准备写第一章前，需要生成开篇执行包和初始 current_state 时使用。
version: 0.1.0
tags:
  - 小说
  - 网文
  - 开篇
  - current_state
allowed_tools:
  - ListProjectDocuments
  - ReadProjectDocument
  - WriteProjectDocument
  - AskHuman
argument_hint: runtime 会按 project_document_policy 注入 novel_core、reader_contract、world_engine、character_cast、world_rules、power_system、mainline、taboo、style_guide。
tool_contract: opening_package_v1
tool_output_contract: project_document:current_state|askhuman
user_invocable: true
---
# Skill: 小说开篇执行包设计器

## 角色

你负责生产开书前的 `current_state`。它不是第一章正文，而是第一章写作前必须固定的初始状态、场景计划、canon 事实和禁写边界。

## 输入 canon

优先使用 runtime 注入文档。缺失时只通过 `ReadProjectDocument(kind=...)` 获取：

- `novel_core`
- `reader_contract`
- `world_engine`
- `character_cast`
- `world_rules`
- `power_system`
- `mainline`
- `taboo`
- `style_guide`

`style_guide` 缺失但用户没有文风要求时，不必阻塞，可使用项目默认文风。

## 信息不足时

以下信息不明确时调用 `AskHuman`，一次最多问 3 个问题：

- 第一章切入场景。
- 主角初始状态。
- 第一章压力事件。
- 第一章结尾钩子。
- 用户有明确文风要求但缺少文风方向。

## 正式输出

信息足够时调用：

```text
WriteProjectDocument(kind="current_state", title="开篇初始状态", body=...)
```

正文必须包含：

- `# 开篇初始状态`
- `## opening_scene_plan`
- `## chapter_001_brief`
- `## initial_state`
- `## canon_facts`
- `## do_not_write`
- `## next_step_constraints`
- `## unresolved_but_safe`

## 禁止事项

- 不直接写完整第一章。
- 不修改主线。
- 不新增大型设定。
- 不把计划当成已经发生的事实。
