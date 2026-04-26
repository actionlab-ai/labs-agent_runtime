# Context Compiler 与 128K 上下文工程

## 1. Context Compiler 是什么

Context Compiler 不写小说。  
它只负责把无限项目资料压缩成当前任务所需的 128K 高价值上下文。

输入：

```yaml
inputs:
  task_type: draft | audit | revise | snapshot | fact_resolve
  target_chapter:
  user_request:
  selected_promptpack:
  focus_entities:
  focus_plot_threads:
```

输出：

```yaml
outputs:
  context_md:
  context_yaml:
  included_sources:
  excluded_sources:
  token_budget_report:
  uncertainty:
```

## 2. Context Pack 标准结构

```markdown
# Task
# Reader Contract
# Chapter Intent
# Scene Cards
# Must Carry Forward
# Must Not Contradict
# Active Characters
# Info Gap
# Plot Threads
# Reader Promise / Payoff Ledger
# Relevant Evidence Cards
# Relevant Raw Text Windows
# Style Constraints
# Output Contract
# Included Sources
# Uncertainty
```

## 3. 优先级

```text
P0 当前任务和输出契约
P1 读者契约
P2 本章 brief / scene cards
P3 上一章 snapshot
P4 活跃角色状态
P5 信息差
P6 活跃伏笔 / 支线
P7 读者承诺 / 爽点账本
P8 事实证据卡
P9 必要前文原文窗口
P10 风格约束
```

## 4. 写正文的预算建议

```text
Runtime/system 规则：2K - 4K
PromptPack：5K - 10K
读者契约：2K - 4K
本章 brief：3K - 6K
scene cards：5K - 10K
上一章 snapshot：3K - 5K
最近 3 章摘要：5K - 10K
活跃角色状态：5K - 10K
信息差/伏笔/爽点账本：8K - 15K
相关历史原文片段：10K - 25K
当前已写草稿：20K - 50K
输出预留：10K - 20K
```

## 5. 质量检查

一个 context pack 应该回答：

```text
模型是否知道本章爽点？
模型是否知道主角当前目标？
模型是否知道主角不能知道什么？
模型是否知道反派不知道什么？
模型是否知道上一章钩子？
模型是否知道本章兑现什么、延迟什么？
模型是否知道本章不能写什么？
模型是否知道哪些设定只能暗示？
模型是否知道每个场景的冲突和转折？
模型是否知道输出格式？
```

如果不能，context pack 不合格。
