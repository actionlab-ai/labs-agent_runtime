# NovelCode Runtime 架构

## 目标

构建小说版 Codex / ClaudeCode：

```text
用户不用手动整理 128K。
Runtime 自动检索、查证、编译上下文、调用模型、保存结果。
```

## 核心组件

```text
Master Writer Agent
Context Compiler
Retrieval Strategy Skill
Canon Resolver SubAgent
Promise Probe SubAgent
Pacing Probe SubAgent
Voice Probe SubAgent
Tool Search
Knowledge Store
Run Store
Changeset Manager
Model Client
```

## 工具

```text
knowledge.search
knowledge.read_shard
chapter.search
chapter.read_window
snapshot.search
ledger.query
fact.resolve
context.build
evidence.create_card
audit.reader_experience
audit.continuity
```

## 执行模式

```text
用户任务
  ↓
Master 规划
  ↓
Tool Search 找能力和资料
  ↓
Context Compiler 编译上下文
  ↓
模型执行当前节点
  ↓
审稿和查证
  ↓
结果入账
```

## 和 ClaudeCode 的相似点

```text
不把所有能力都塞给模型
用 search 找相关能力 / 文件
模型通过工具读取资料
每次执行有 run record
输出不直接覆盖正式文件
支持 changeset / approval
支持局部重跑
```

## 小说领域改造点

```text
函数定义 -> 人物/物品/伏笔 shard
调用链 -> 剧情线/关系线
测试 -> 审稿和连续性检查
编译错误 -> 设定矛盾
git diff -> state diff / changeset
```

## 关键原则

模型可以请求工具，Runtime 执行工具。模型不能直接访问文件系统。写正式稿必须 changeset + approval。
