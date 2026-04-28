# SQL 检索架构：原文、索引、证据三层分离

这份文档专门回答一个现在已经很现实的问题：

```text
原来全是 Markdown 的时候，
“物品 -> 事件 -> 原文窗口 -> fact card” 这条链很直觉。

引入 SQL 以后，
到底该让数据库存什么，原文还算不算 source of truth？
```

结论很明确：

```text
SQL 不应该替代原文，
而应该把“导航、定位、关联、过滤、排序”做强。
```

也就是说：

- 原文章节仍然是事实源头
- SQL 负责把 shard / event / chapter anchor 串起来
- fact card 仍然必须落到证据链，而不是直接相信一条 SQL 记录

## 1. 先看当前代码到底到了哪一步

当前这版 PostgreSQL 已经有这些能力：

- `projects`
  - 项目主表
- `project_documents`
  - 项目级文档仓
- `sessions / messages`
  - 会话与消息
- `runs`
  - 运行记录

但关键点是：  
当前 `project_documents` 只有一层“按 `kind` 存整份文档”的结构。

这意味着它很适合存：

- `project_brief`
- `world_rules`
- `power_system`
- `mainline`
- `current_state`

它不适合直接承担：

- 某个物品出现过哪些章节
- 某个事件引用了哪些原文窗口
- 某条 fact card 的证据链来自哪些 anchor

原因很简单：  
这类信息不是“项目级大文档”，而是“可 join 的细颗粒关系数据”。

## 2. 你原来那套思想其实没错，不该被 SQL 推翻

你原来的顶层原则是对的：

```text
原文是 source of truth
shard 是索引
fact card 是证据结果
```

SQL 进来以后，不是把这套推翻，而是把这套做成更强的“机器可查关系图”：

```text
SQL = shard / event / anchor / relation 的结构化索引层
Raw chapter text = 最终事实核验层
Fact card = 基于索引找到原文后生成的证据结果
```

所以你不该问：

```text
既然有 SQL 了，原文是不是不用查了？
```

你该问的是：

```text
既然有 SQL 了，怎么更快地定位到必须读的那几个原文窗口？
```

## 3. 当前最合理的分层

### 第 1 层：项目级 canon 文档层

继续用现在的 `project_documents`。

这一层负责放“面向创作的长期稳定文档”：

- 项目简介
- 世界规则
- 能力体系
- 主线规划
- 当前状态

特点：

- 文本较长
- 人可以直接读
- 适合注入模型上下文
- 不适合细粒度追溯

### 第 2 层：检索索引层

这一层要新建 SQL 表，不要硬塞进 `project_documents`。

这一层负责：

- entity
  - 人物、物品、地点、势力、线索
- event
  - 章节事件、场景事件、关键动作
- mention / appearance
  - 某实体在哪个事件、哪个章节锚点出现
- relation
  - 实体和实体、实体和事件之间的关系

这一层是“导航图”，不是“最终事实”。

### 第 3 层：原文证据层

这一层保存原文章节和可定位窗口。

至少要能表达：

- chapter_id
- scene_id
- anchor
- raw_text
- paragraph range / offset

只有查到这一层，才能真正回答：

- 这个物品第一次出现时原文到底怎么写的？
- 那个事件里主角到底有没有确认这个事实？
- 这个细节是不是只是角色猜测，不是客观事实？

### 第 4 层：fact card 结果层

fact card 是检索和查证后的结果，不是底层真相本身。

它可以存 SQL，方便复用；  
但它的每条结论都必须带 source anchor。

## 4. “查某个物品” 的标准执行链

你之前的想法本质上是对的，SQL 版只是把它结构化：

```text
用户问：旧铜牌上到底刻了什么字？

1. 先查 entity / item
2. 找到 item 对应的 appearances
3. 从 appearances 找到关联 event
4. 从 event 找到 chapter anchors
5. 按策略读原文窗口
6. 生成带 evidence 的 fact card
```

这条链里：

- SQL 负责 1~4 的定位和过滤
- 原文窗口负责 5 的核验
- fact card 是 6 的输出

所以 SQL 不是让流程变了，而是让：

- 不用全文盲搜
- 不用每次从大 markdown 手工找
- 可以稳定复现查询路径

## 5. 建议你新增的表，不要一步到位搞太大

先别做一个“宇宙级 schema”。  
先做能支持“物品 -> 事件 -> 原文 anchor -> fact card”的最小闭环。

### 5.1 `chapter_sources`

存原文章节或章节切片。

建议字段：

- `id`
- `project_id`
- `chapter_id`
- `scene_id`
- `anchor`
- `text`
- `metadata`
- `created_at`
- `updated_at`

用途：

- 提供真正可读的原文窗口

### 5.2 `knowledge_entities`

存人物 / 物品 / 地点 / 势力 / 线索等实体。

建议字段：

- `id`
- `project_id`
- `entity_type`
- `name`
- `aliases`
- `summary`
- `canonical_status`
- `metadata`

用途：

- 统一实体入口

### 5.3 `story_events`

存事件节点。

建议字段：

- `id`
- `project_id`
- `chapter_id`
- `scene_id`
- `title`
- `summary`
- `event_type`
- `source_anchor`
- `metadata`

用途：

- 把“实体出现”挂到明确事件上

### 5.4 `entity_appearances`

这是最关键的一张表。

建议字段：

- `id`
- `project_id`
- `entity_id`
- `event_id`
- `chapter_id`
- `source_anchor`
- `appearance_role`
- `summary`
- `confidence`
- `metadata`

用途：

- 物品 / 人物 / 地点在什么事件里、以什么角色出现

### 5.5 `fact_cards`

建议字段：

- `id`
- `project_id`
- `query`
- `answer`
- `confidence`
- `allowed_usage`
- `forbidden_usage`
- `unresolved`
- `metadata`

### 5.6 `fact_card_evidence`

不要把证据糊成一坨 JSON，最好独立一张表。

建议字段：

- `id`
- `fact_card_id`
- `source_type`
- `chapter_id`
- `source_anchor`
- `summary`
- `evidence_rank`
- `metadata`

用途：

- 明确 fact card 证据链

## 6. 为什么不建议把这些都继续塞到 `project_documents`

因为 `project_documents` 当前的语义是：

```text
一个 project 下，
每个 kind 基本是一份长期维护的大文档
```

如果你把：

- 每个物品
- 每个事件
- 每个 fact card

都塞成一个 `kind`

很快会出问题：

- 无法自然表达一对多 appearance
- 无法稳定 join
- 查询条件会退化成文本过滤
- “查某物品最近一次出现且有 raw_text_verified 的事件”会很难写
- 最后又会回到“SQL 里存 markdown，再用程序二次解析”

这等于既没吃到 SQL 的结构化好处，也把 markdown 的直观性弄丢了。

## 7. 最务实的推进顺序

不要一口气把所有人物、地点、伏笔、账本全上库。

按这个顺序做：

### Phase 1

只补最小证据链：

- `chapter_sources`
- `knowledge_entities`
- `story_events`
- `entity_appearances`

先把：

```text
物品 -> 出现事件 -> 章节 anchor -> 原文窗口
```

打通。

### Phase 2

再补：

- `fact_cards`
- `fact_card_evidence`

把“查证结果可缓存、可复用、可审计”做起来。

### Phase 3

再考虑更复杂的：

- 时间线关系
- 冲突检测
- canon 漂移告警
- 自动回查失效

## 8. 给你一个非常直接的实现判断标准

以后你每设计一张 SQL 表，就问自己两个问题：

### 问题 1

这张表存的是：

- 人要长期直接阅读和维护的大文档

还是：

- 程序要高频过滤、排序、关联、定位的结构化索引

如果是前者，优先 `project_documents`。  
如果是后者，应该单独建表。

### 问题 2

这条记录能不能单独作为“最终事实”被信任？

如果不能，而必须回原文确认，  
那它就只是索引，不是 source of truth。

## 9. 最后的落地结论

你现在最该做的不是“把以前的 markdown 思路全改成 SQL 思路”，而是：

```text
保留原来的事实分层
只把 SQL 放到它最擅长的位置
```

最合理的角色分工是：

```text
project_documents = 项目级 canon 文档
knowledge_entities / story_events / entity_appearances = 检索导航图
chapter_sources = 原文证据源
fact_cards / fact_card_evidence = 查证结果
```

如果只用一句话总结：

```text
SQL 不是取代“物品 -> 事件 -> 原文”的链路，
而是把这条链路做成稳定、可查、可复用的结构化导航系统。
```
