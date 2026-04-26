# Novel as Code：把小说当成工程项目

## 目标

让长篇小说项目像代码项目一样：

```text
可检索
可追踪
可验证
可重跑
可回滚
可审计
```

## 代码项目 vs 小说项目

| 代码项目 | 小说项目 |
|---|---|
| 源码文件 | 章节原文 |
| 函数 / 类 | 人物 / 物品 / 地点 / 能力 / 势力 |
| 调用链 | 伏笔链 / 支线链 / 关系链 |
| README | project brief / reader contract |
| 测试 | continuity audit / reader audit |
| 编译错误 | 设定矛盾 / 信息差错误 |
| git diff | state diff / changeset |
| LSP / symbol index | entity index / shard index |
| Codex | NovelCode |

## 推荐目录

```text
novel-project/
  00-project/
    project-brief.md
    reader-contract.md
    style-guide.md
    taboo.md

  01-bible/
    world-rules.md
    power-system.md
    factions.md
    locations.md

  02-outline/
    mainline.md
    volume-roadmap.md
    chapter-roadmap.md

  03-drafts/
    chapters/
    candidates/
    revisions/

  04-knowledge/
    characters/
    items/
    locations/
    abilities/
    factions/
    plot-threads/
    reader-promises/
    pacing/
    info-gap/

  05-snapshots/
  06-context-packs/
  07-audits/
  08-fact-cards/
  09-runs/
  99-ops/changesets/
```

## 项目资产层级

```text
L0 原文层：章节正文，最终证据源。
L1 摘要层：chapter snapshot，便于快速导航。
L2 结构化状态层：character ledger / item ledger / info gap / plot threads。
L3 知识 shard 层：每个角色、物品、地点、伏笔、爽点一个文件。
L4 检索层：search / read_window / fact.resolve。
L5 上下文层：context pack，给模型当前任务使用。
L6 执行层：promptpack / skill / workflow / agent。
```
