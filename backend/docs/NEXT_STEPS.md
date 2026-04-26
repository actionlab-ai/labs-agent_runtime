# 下一步

本文档只记录“从当前状态往前走”的真实待办，不再按旧版 `v0.2 / v0.3 / v0.4` 这种历史阶段名来写。

## 当前完成度

### 已完成

- metadata-only skill boot
- search-first runtime
- activation window
- retained discovered-skill pool
- `tool_reference_like` 激活对象
- skill-specific `tool_input_schema`
- skill executor 内部 `Read / Write / Edit / Glob`
- skill executor 内部 `Bash / PowerShell`
- router / skill executor request assembly 落盘
- router / skill executor response analysis 落盘
- 默认文档落地目录
- DeepSeek 实测调通
- `return_skill_output_direct`
- `opening_v1` 的初步输出 contract
- `bootstrap_v1` 的起盘与澄清 contract

### 进行中

- 把 tool 层继续往 `novelcode` 的稳定性和可恢复性靠拢

### 暂缓

- 跨轮 discovered memory
  - 等 mem 系统稳定后再做

## 当前优先级

### P0

1. 增加“未 discover 先误调用”的自修复提示
   - 如果模型试图调用尚未 discover 的 skill tool，runtime 明确提示：
   - 先 `tool_search("select:<skill_id>")`
   - 再 retry

2. 继续减少 `skill_call` 在主路径中的存在感
   - 保留兼容
   - 但让 router 更坚定地优先使用 activated skill tool

### P1

1. 等 mem 系统稳定后，把 discovered skill pool 从单次 `Run` 内存推进到 transcript / memory 驱动的跨轮恢复机制
2. 评估是否需要轻量 compaction state carry-over

### P2

1. 定义 `outline_v1`
2. 定义 `rewrite_v1`
3. 给不同小说 skill 建立稳定 output contract
4. 继续打磨 `bootstrap_v1`
   - 用户澄清后的二轮合并
   - 世界观 / 金手指 / 主线任务的分段拆 skill

### P3

1. 更强的本地 search
   - BM25
   - embedding rerank
2. `references` 自动加载策略
3. 更细的项目文件读取与知识片段装配能力

## 原则

接下来仍然坚持这个顺序：

```text
先把 tool layer 磨稳
-> 再扩小说 skill 库
```

因为当前阶段最怕的不是“skill 内容不够多”，而是“调用层还不够稳，导致 skill 设计价值被浪费”。
