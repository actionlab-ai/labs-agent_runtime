# Master / SubAgent 与章节生产流程

## 1. Master / SubAgent 创作循环

Master 不应该靠印象写第 1000 章。遇到不确定细节时，应该派 SubAgent 查证据。

标准循环：

```text
1. Master 准备写当前章节
2. Master 或 Context Compiler 发现不确定点
3. 生成 fact query
4. Retrieval Strategy 选择策略
5. SubAgent 执行查询
6. SubAgent 返回 fact card
7. Context Compiler 编译上下文
8. Master 写作
9. 审稿
10. Snapshot / Ledger 入账
```

示例：

```text
Master:
第1000章要写主角摸出旧铜牌。不确定：铜牌上到底刻了什么字？

Retrieval Strategy:
query_type = item_origin_detail
strategy = origin_first_then_recent_confirmation

Canon Resolver:
读 item shard -> ch0312 -> ch0478 -> 返回 fact card

Master:
根据 fact card 写：“他指腹擦过那半个残缺的‘巡’字。”
```

## 2. 普通章节生成 Workflow

```text
inspect_project
  ↓
build_context_pack
  ↓
generate_chapter_brief
  ↓
generate_scene_cards
  ↓
validate_scene_cards
  ↓
draft_chapter
  ↓
reader_experience_audit
  ↓
continuity_audit
  ↓
ai_tone_surgery
  ↓
revise_chapter
  ↓
human_acceptance
  ↓
snapshot_chapter
  ↓
update_shards_and_ledgers
```

## 3. 第一章 Workflow

首章不是普通章节，它是商品转化页。

首 300 字必须出现：

```text
具体人物
具体冲突
具体损失或压迫
主角即时处境
读者站队理由
```

首章禁止：

```text
天气开头
气味开头
光影开头
主角醒来但无危机
长段世界观介绍
先讲王朝历史
先讲力量体系
```

质量门：

```text
前 300 字没有冲突：重写。
前 1000 字世界观解释超过 15%：重写。
前 1500 字主角没有主动选择：重写。
章尾没有未完成承诺：重写。
```

## 4. 支线回归 Workflow

支线回归前必须检查：

```text
支线上次出现在哪一章？
当时未完成的问题是什么？
读者还记得什么？
主角还知道什么？
支线角色当前状态是什么？
支线回归服务主线吗？
回归是否需要提醒性场景？
回归后是否要兑现之前承诺？
```

流程：

```text
detect_subplot_return
  ↓
read_plot_thread_shard
  ↓
timeline_sweep
  ↓
retrieve_last_appearance_raw_text
  ↓
retrieve_key_origin_raw_text
  ↓
build_return_context
  ↓
design_reentry_scene
  ↓
reader_experience_audit
```
