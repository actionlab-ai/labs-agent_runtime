# Skill 输出 Contract

这份文档解释“skill 输出 contract 分层”到底是什么意思，以及它为什么和 `tool_search` 是两条相关但不同的线。

## 1. 一句话解释

`tool_search` 负责回答：

- 该调用哪个 skill
- skill 什么时候变成可调用
- skill 在后续几轮里是否继续保留

输出 contract 负责回答：

- 这个 skill 跑完后，应该稳定返回什么形状

所以它们的关系是：

```text
tool layer 解决“调谁”
output contract 解决“返回什么”
```

## 2. 为什么这层很重要

当前 runtime 已经有一层输出规整逻辑，比如：

- `return_skill_output_direct`
- `normalizeSkillOutput()`

它现在能把 opening skill 的输出收得比较干净，但如果以后 skill 多了：

- opening
- outline
- rewrite
- worldbuilding
- scene continuation

而每个 skill 都各自输出一套格式，runtime 很快就会变成：

- 这里剥标题
- 那里删分析
- 另一处提取“正文段”

这会让 runtime 越来越脏。

## 3. contract 分层想解决什么

它的目标是：

- 不让每个新 skill 都逼 runtime 新增一段字符串清洗逻辑
- 让 skill 类型本身决定输出格式

## 4. 当前推荐的三类 contract

### `opening_v1`

用途：

- 生成直接给读者看的开篇正文

输出要求：

- 默认只给成品正文
- 不加标题
- 不加分析
- 不加自检
- 不加“下一轮还需要什么信息”

### `outline_v1`

用途：

- 生成结构化故事规划，而不是成品 prose

输出要求：

- 稳定分段
- 稳定标题
- 重点是 premise / hook / progression / chapter arc / suspense

### `rewrite_v1`

用途：

- 对已有段落进行改写

输出要求：

- 先给改写结果
- 再给极短改动说明
- 不要讲课式长解释

## 5. 为什么这和 `novelcode` 不完全是一回事

`novelcode` 主要解决的是：

- deferred tool discovery
- discovered tool persistence
- 动态 callable surface

它并不直接等于“skill 输出 contract 分层”。

所以更准确的说法是：

```text
输出 contract 和 novelcode 方向兼容
但它不是 novelcode 的同一子系统
```

## 6. 当前项目里已经落到哪一步

目前已经开始落地的是：

- `webnovel-opening-sniper`
  - `tool_contract: opening_v1`
  - `tool_output_contract: opening_prose_v1`

这意味着：

- tool 层已经开始知道“这个 skill 是哪种 contract”
- runtime 以后可以按 contract 类型做更稳定的输出处理

## 7. 推荐顺序

当前最合理的推进顺序仍然是：

1. 先把 tool layer 打磨到很稳
2. 再把 `opening / outline / rewrite` 三类 contract 定死
3. 最后才开始大规模扩 skill 库

这个顺序的好处是：

- runtime 不会越来越多临时补丁
- 每个新 skill 都能挂在稳定 contract 上
- 后续小说技能扩展会轻松很多
