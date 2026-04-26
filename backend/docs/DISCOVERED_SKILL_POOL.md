# Discovered Skill Pool 说明

这份文档专门解释当前 runtime 里最重要的一层状态：

- fresh activation window
- retained discovered-skill pool

它们决定了“search 是不是只是查推荐”，还是“search 真的改变了后续可调用能力”。

## 1. 这个机制解决什么问题

如果 search 只是返回排序列表，模型会陷入这种循环：

```text
search
-> 看候选
-> 调通用执行器
-> 再 search
-> 忘掉刚才已经发现过什么
```

这样做的坏处是：

- 模型没有稳定的工作集
- 每次 search 都像重新开始
- 稍微换个查询词，前一轮发现的能力就丢了

## 2. 当前实现怎么做

现在的流程是：

1. `tool_search` 先返回排序结果
2. 其中最相关的小簇命中进入 `activation window`
3. `activation window` 再和历史 retained pool 合并
4. 下一轮 router 只看到：
   - `tool_search`
   - retained skill tools
   - `skill_call` 兼容兜底

这时 search 就不再只是“推荐一下”，而是“真的把下一轮可调用能力重建出来”。

## 3. 两层状态分别是什么

### activation window

本轮 search 新激活的少量 skill。

它强调的是：

- 这次 query 刚刚带出来什么
- 当前最相关的那一小簇 callable skills 是谁

### retained discovered-skill pool

之前已经发现、并且仍然保留在运行时桌面上的 skill tools。

它强调的是：

- 之前搜出来的东西不要立刻丢
- 多轮 tool 使用要有连续性

## 4. 一个直观类比

把 runtime 想成一张写作桌：

- activation window
  - 这次刚从书架上拿到桌面的几本书
- retained pool
  - 桌上已经摊开、暂时还不想收回去的那一小摞书

每次新 search 可以再拿 1 到 3 本新书上桌。
但不会因为这次拿了新书，就立刻把前面摊开的书全扔回去。

只有桌面容量不够时，旧的尾部项目才会被清掉。

这就是 `max_retained_skills` 的意义。

## 5. 当前边界

当前这套 retained pool 只在单次 `Run` 内有效。

也就是说：

- 同一次 router run 内，它已经能形成“已发现 skill 工具池”
- 但它还不是 `novelcode` 那种“从消息历史恢复 discovered tools”的跨轮记忆机制

这个差距我们已经明确记成 TODO，等 mem 系统稳定后再推进。

## 6. 为什么这一步值得先做

你当前的方向是先把 tool 层打磨到很稳，再开始扩小说 skill。

这个顺序是对的，因为：

- 如果 runtime 老是忘掉自己发现过什么
- skill 写得再漂亮，也会被调用层浪费掉

只有当 runtime 能稳定做到：

- 搜得准
- 激活得窄
- 保留得住
- 输出差异解释清楚

后面的小说 skills 才能真正变成可组合、可迭代、可维护的积木。
