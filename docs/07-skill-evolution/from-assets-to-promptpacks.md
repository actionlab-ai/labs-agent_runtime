# 从知识资产到 PromptPack / Skill

## 去神秘化

Skill 不是魔法。

```text
Skill / PromptPack = 可复用 prompt 规则 + 输入输出契约 + 参考资料 + 工具约定。
```

## 为什么还需要 Skill

如果只有一次写作，一个 prompt 就够。但长期小说工程需要：

```text
可检索
可版本化
可测试
可复用
可组合
可审计
可局部优化
```

所以我们用 PromptPack 管理任务规则。

## 推荐结构

```text
promptpacks/
  reader_experience_audit/
    PACK.md
    pack.yaml
    references/
      scoring-rubric.md
      anti-patterns.md
    schemas/
      output.yaml
    tests/
      bad-opening.md

  canon_retrieval_strategy/
    PACK.md
    pack.yaml
    references/
      retrieval-strategy-matrix.md
```

## pack.yaml 示例

```yaml
id: reader_experience_audit
name: 读者体验审稿
description: 检查弃书风险、爽点兑现、主角主动性、设定密度和 AI 味。
when_to_use: 用户要求从商业网文读者体验角度审查章节时使用。

input:
  required:
    - chapter_text
    - reader_contract
  optional:
    - scene_cards
    - chapter_brief
    - promise_ledger

references:
  - path: references/scoring-rubric.md
    load_policy: always
  - path: references/anti-patterns.md
    load_policy: auto
    triggers:
      keywords: [AI味, 弃书, 第一章, 不抓人]

tools:
  - audit.reader_experience_scan

output:
  format: markdown
  schema: schemas/output.yaml
```

## 核心 PromptPack 路线图

P0：

```text
context_builder
canon_retrieval_strategy
fact_resolver
reader_experience_audit
chapter_snapshot
```

P1：

```text
scene_beat_engine
chapter_draft
ai_tone_surgery
first_chapter_engine
```

P2：

```text
info_gap_audit
promise_probe
pacing_probe
subplot_return_planner
voice_probe
```

P3：

```text
retcon_manager
changeset_reviewer
snapshot_rebuilder
```

原则：先做 `context_builder + fact_resolver`，让系统能像 Codex 一样找资料。
