# Novel Knowledge Assets v0.1

这是一组“小说工程知识资产”文件，用来作为后续演化 Skill / PromptPack / Agent Runtime 的起点。

核心判断：

```text
小说项目也要像代码项目一样，有可检索、可追踪、可验证的结构。
```

本包不是普通 prompt 合集，而是把我们讨论过的思想沉淀成可维护的 Markdown 规则资产：

- 128K 上下文工程
- Knowledge Shard：人物、物品、伏笔、爽点、信息差
- Fact Card / Evidence Card：带证据的事实回查
- Retrieval Strategy：起源优先、最近优先、双端验证、时间线扫描
- Master / SubAgent 协作
- 读者体验、情绪、节奏、支线回归
- NovelCode Runtime：小说版 Codex / ClaudeCode 的工程思想

推荐演化路线：

```text
Markdown 知识资产
  ↓
PromptPack / Skill
  ↓
工具 Schema
  ↓
Go Runtime
  ↓
Workflow 节点
  ↓
真实章节生产与审稿
```

核心分层：

```text
原文是 source of truth
shard 是索引
snapshot 是状态缓存
ledger 是长期状态
tool-search 是导航
subagent 是局部调查员
fact card 是证据结果
context compiler 是 128K 压缩器
master writer 是最终创作决策者
```
