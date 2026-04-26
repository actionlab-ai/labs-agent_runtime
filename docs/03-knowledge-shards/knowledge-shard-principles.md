# Knowledge Shard 设计原则

## 什么是 Shard

Shard 是一个小说实体或状态对象的知识文件，例如：

```text
一个人物
一个物品
一个地点
一条伏笔
一个读者承诺
一个能力规则
一段关系
一个情绪债
```

每个 shard 的目标不是替代原文，而是让系统快速知道：

```text
该去哪里查原文？
当前 canon 状态是什么？
哪些写法被允许？
哪些写法被禁止？
```

## Shard 的价值

```text
帮助搜索
指向原文锚点
记录当前状态
记录 aliases
记录 can_write / cannot_write
支持 SubAgent 精准回查
支持 Context Compiler 自动选上下文
```

## Shard 不是百科

不要写成：

```text
旧铜牌是一个神秘物品，象征……
```

要写成：

```yaml
first_seen: ch0312#p087
latest_seen: ch0478#p142
confirmed:
  - 半枚
  - 有火烧缺口
  - 有半个残缺的“巡”字
cannot_write:
  - 不能写成完整“巡查司”
```

## Shard 与原文关系

```text
shard 是索引
原文是证据
snapshot 是状态缓存
ledger 是当前状态表
```

## Shard 更新

每章结束后由 Snapshot / Consolidator 提出 shard 更新建议：

```text
new entity
new appearance
state changed
fact confirmed
fact contradicted
promise delivered
plot thread advanced
```

重大更新建议走 changeset。
