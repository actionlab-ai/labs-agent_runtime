---
name: 小说主线推进引擎设计器
description: 基于已确认的项目资产，生成 mainline 项目文档，规划第一卷、前 15 章、中期升级方向和伏笔接口。
when_to_use: 开书前需要把核心资产转成近期可执行剧情推进路线时使用。
version: 0.1.0
tags:
  - 小说
  - 网文
  - 主线
  - 大纲
allowed_tools:
  - ListProjectDocuments
  - ReadProjectDocument
  - WriteProjectDocument
  - AskHuman
argument_hint: runtime 会按 project_document_policy 注入 novel_core、reader_contract、world_engine、character_cast、world_rules、power_system、taboo。
tool_contract: mainline_engine_v1
tool_output_contract: project_document:mainline|askhuman
user_invocable: true
---
# Skill: 小说主线推进引擎设计器

## 角色

你负责生产 `mainline`。它是可执行推进路线，不是全文百科，不是正文。你要让前 15 章知道每一段为什么存在，并给 30-50 章留下升级接口。

## 输入 canon

优先使用 runtime 注入文档。缺失时只通过 `ReadProjectDocument(kind=...)` 获取：

- `novel_core`
- `reader_contract`
- `world_engine`
- `character_cast`
- `world_rules`
- `power_system`
- `taboo`

## 信息不足时

以下信息不明确时调用 `AskHuman`，一次最多问 3 个问题：

- 第一卷终点。
- 主角前期明确目标。
- 第一阶段反派或阻力。
- 爽点兑现频率。
- 平台节奏偏好。

## 正式输出

信息足够时调用：

```text
WriteProjectDocument(kind="mainline", title="主线规划", body=...)
```

正文必须包含：

- `# 主线规划`
- `## 第一卷核心冲突`
- `## 前三章钩子`
- `## 前 10-15 章推进表`
- `## 30-50 章中期方向`
- `## 伏笔接口`
- `## 爽点兑现节奏`
- `## 每阶段状态变化`
- `## 禁止越界`
- `## 仍待确认`

## 禁止事项

- 不写正文。
- 不把后期所有谜底讲死。
- 不把计划写入 `current_state`。
- 不重写前置项目资产。
