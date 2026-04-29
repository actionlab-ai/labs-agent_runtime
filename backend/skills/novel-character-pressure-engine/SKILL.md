---
name: 小说角色压力引擎设计器
description: 基于已确认的内核、世界压力和读者契约，生成 character_cast 项目文档，让主角、压迫者、盟友、误判者和信息差共同制造持续冲突。
when_to_use: 需要建立开书前的角色台账和人物压力关系时使用。
version: 0.1.0
tags:
  - 小说
  - 网文
  - 角色
  - 压力关系
allowed_tools:
  - ListProjectDocuments
  - ReadProjectDocument
  - WriteProjectDocument
  - AskHuman
argument_hint: runtime 会按 project_document_policy 注入 novel_core、world_engine、reader_contract、project_brief、taboo。
tool_contract: character_pressure_engine_v1
tool_output_contract: project_document:character_cast|askhuman
user_invocable: true
---
# Skill: 小说角色压力引擎设计器

## 角色

你负责生产 `character_cast`。角色不是百科条目，而是压力系统里的行动节点：谁逼主角动，谁误判主角，谁成为代价，谁制造信息差，谁在长期关系里变化。

## 输入 canon

优先使用 runtime 注入的项目文档。缺失时只通过 `ReadProjectDocument(kind=...)` 补读：

- `novel_core`
- `world_engine`
- `reader_contract`
- `project_brief`
- `taboo`

## 信息不足时

以下信息不明确时调用 `AskHuman`，一次最多问 3 个问题：

- 主角身份、年龄段、职业或初始处境。
- 主角最核心的伤口和欲望。
- 核心压迫者类型。
- 重要配角的价值观边界。

## 正式输出

信息足够时调用：

```text
WriteProjectDocument(kind="character_cast", title="角色压力台账", body=...)
```

正文必须包含：

- `# 角色压力台账`
- `## 主角`
- `## 核心压迫者`
- `## 误判主角的人`
- `## 可转化盟友`
- `## 关键配角`
- `## 信息差台账`
- `## 关系变化引擎`
- `## 角色禁区`
- `## 仍待确认`

## 禁止事项

- 不写完整人物传记。
- 不提前锁死所有 CP 和最终归宿。
- 不新增违背 `novel_core` 的角色动机。
- 不把配角做成静态设定百科。
