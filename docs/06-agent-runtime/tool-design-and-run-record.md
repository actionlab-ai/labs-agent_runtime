# Tool 设计与 Run Record

## 1. knowledge.search

搜索 shard。

```yaml
input:
  query: 旧铜牌 巡字
  types: [item, plot_thread]
  limit: 5
output:
  hits:
    - path: 04-knowledge/items/old-copper-token.yaml
      score: 0.94
      reason: aliases contain 铜牌 and canonical facts contain 巡字
```

## 2. knowledge.read_shard

读取 shard 文件。

```yaml
input:
  path: 04-knowledge/items/old-copper-token.yaml
output:
  shard:
    id:
    first_seen:
    latest_seen:
    retrieval_policy:
```

## 3. chapter.search

搜索章节原文。

```yaml
input:
  query: 旧铜牌 OR 半枚铜牌 OR 巡牌
  chapter_range: ch0001..ch0999
  order: desc
  limit: 10
```

## 4. chapter.read_window

读取原文窗口。

```yaml
input:
  chapter: ch0312
  anchor: ch0312#p087
  window_paragraphs_before: 8
  window_paragraphs_after: 10
```

## 5. fact.resolve

自动完成事实查证。

```yaml
input:
  query: 旧铜牌上刻了什么字？
  current_chapter: ch1000
output:
  fact_card:
```

## 6. context.build

编译 context pack。

```yaml
input:
  task_type: draft
  chapter: ch1000
  focus_entities:
    - item.old_copper_token
output:
  context_pack:
  included_sources:
  token_budget:
```

## 7. Run Record

每次模型调用必须可追踪：

```text
用了哪个 PromptPack
用了哪些上下文
读了哪些 shard
读了哪些原文窗口
调用了哪些工具
模型输出是什么
人工是否接受
```

标准目录：

```text
09-runs/run-20260425-xxxx/
  run.yaml
  nodes/
    build_context/
      node.yaml
      context_pack.md
      included_sources.yaml
    fact_resolve_old_copper_token/
      node.yaml
      tool_calls.yaml
      fact_card.yaml
    draft_chapter/
      node.yaml
      prompt.md
      output.md
    reader_audit/
      node.yaml
      audit.md
```

## 8. 局部重跑

用户不满意某节点：

```text
场景卡不够狠，从 scene_cards 之后重跑。
```

系统应该保留上游节点，失效当前节点和下游节点，从指定节点重新执行。
