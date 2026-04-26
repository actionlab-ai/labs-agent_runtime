# 事实检索策略矩阵

## 核心原则

检索策略不能每次靠模型临场发挥。必须根据问题类型选择固定策略。

## 策略总览

| 问题类型 | 策略 | 原因 |
|---|---|---|
| 当前状态 | recent_first | 状态会变化，最近更权威 |
| 起源设定 | origin_first | 要看最早定义 |
| 物品细节 | origin_first + recent_confirmation | 外观常在首次出现最精确 |
| 伏笔回收 | origin_recent_bridge | 要看埋点、最近暗示、可否回收 |
| 信息差 | recent_first + raw_evidence | 谁知道什么会变化 |
| 关系变化 | timeline_sweep | 关系是逐步变化链 |
| 角色口吻 | representative_sample | 需要典型样本，不只是最近 |
| 能力规则 | origin_rule + latest_usage | 看规则和突破情况 |
| 爽点兑现 | promise_ledger + recent_window | 看欠账多久、是否已兑现 |
| 矛盾仲裁 | conflict_resolution | 多来源冲突需仲裁 |

## recent_first

适用：

```text
现在 / 当前 / 还在不在 / 知不知道 / 有没有拿着 / 最新状态
```

流程：

```text
查当前 ledger → 查最近 snapshot → 从当前章往前找最近出现 → 读最近原文窗口 → 若不确定继续扩展
```

## origin_first

适用：

```text
第一次 / 最早 / 起初 / 原本 / 当时怎么写 / 物品外观 / 规则定义
```

流程：

```text
查 first_seen → 读首次出现原文窗口 → 提取原始事实 → 查最近确认
```

## origin_recent_bridge

适用：

```text
伏笔能不能回收 / 这个线索现在能不能揭露 / 设定有没有变化
```

流程：

```text
找首次埋点 → 找最近一次出现 → 找中间关键转折 → 判断当前 reveal level
```

## timeline_sweep

适用：

```text
关系怎么变化 / 仇恨怎么积累 / 支线怎么推进
```

流程：

```text
找实体 pair → 按章节顺序列事件 → 抽取变化链 → 得出当前关系状态
```

## representative_sample

适用：

```text
口吻 / 说话方式 / 行为风格 / 人物像不像
```

流程：

```text
找典型场景 → 找高压场景 → 找私下场景 → 找最近状态 → 生成 voice profile
```

## conflict_resolution

适用：

```text
前后矛盾 / 到底哪个为准 / snapshot 与原文不一致
```

流程：

```text
收集冲突 claim → 标注来源类型 → 按证据优先级仲裁 → 输出 canonical fact → 生成修正建议
```
