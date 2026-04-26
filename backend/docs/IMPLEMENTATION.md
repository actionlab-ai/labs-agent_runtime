# 实现说明

这份文档描述当前 `Novel Agent Runtime` 的实际运行结构，不再站在旧版视角描述。

## 1. 当前到底实现到哪一步

当前 runtime 已经实现了：

1. 启动时只加载 skill metadata
2. 首轮只暴露 `tool_search`
3. `tool_search` 返回：
   - query 解析
   - ranking hits
   - `tool_reference_like` 激活对象
   - activation window
   - retained discovered-skill pool
4. 下一轮动态重建 tools 列表
5. 激活后的 skill tool 可暴露 skill 自己声明的 `tool_input_schema`
6. skill executor 内部支持：
   - `Read`
   - `Write`
   - `Edit`
   - `Glob`
   - `Bash`
   - `PowerShell`
7. 小说类 durable output 可以直接落成 markdown 文档
8. skill 执行成功后，默认直接返回 skill 输出或文档摘要
9. request assembly 会把预处理后的 messages / tools / prompt / final chat request 落盘
10. response analysis 会把每轮是否发生 tool call 结构化落盘

## 2. 关键调用链

当前主链可以概括为：

```text
用户输入
-> Router 首轮只看到 tool_search
-> 模型调用 tool_search
-> runtime 返回 hits + tool_reference_like + activation plan
-> runtime 重建下一轮 tools
-> 模型调用 activated skill tool
-> runtime 按需加载完整 SKILL.md
-> skill executor 发起第二次模型调用
-> skill executor 如有需要继续调用 Read / Write / Edit / Glob / Bash / PowerShell
-> 成功后返回 skill 输出或文档路径摘要
```

## 3. 现在的请求装配器长什么样

当前 runtime 不再是“拿着 messages 和 tools 直接发模型”。

现在每轮请求前，都会先显式 build assembly。

### Router assembly

每轮 router 会先组装：

1. `system prompt blocks`
2. `<available-deferred-skills>` reminder
3. `<retained-skill-tools>` reminder
4. live conversation
5. final tool specs
6. final chat request

然后落盘到：

- `router/round-XX-assembly.json`
- `router/round-XX-assembly.md`

### Skill assembly

每轮 skill executor 会先组装：

1. executor system prompt
2. compiled skill prompt
3. local tool pack
4. final chat request

然后落盘到：

- `skill-calls/<skill>/round-XX-assembly.json`
- `skill-calls/<skill>/round-XX-assembly.md`

## 4. 为什么这层很重要

这层是当前 Go runtime 向 `novelcode` 靠近时最关键的一步之一。

因为 `novelcode` 的强点并不只是：

- 有 `ToolSearch`
- 有 `Read / Write / Edit / Glob`

而是它在真正发 API 请求前，会先做一遍：

- messages 预处理
- deferred tools 处理
- final tool schema 装配
- final system prompt 组装

现在 Go 版虽然还没有 `novelcode/messages.ts` 那么重的 transcript repair / compaction 体系，但已经有了清晰的 request assembly 层。

## 5. 现在比旧版强在哪

旧版更像这样：

```text
tool_search
-> 返回 skill metadata
-> 模型挑一个 skill_id
-> 调 generic skill_call
```

新版更像这样：

```text
tool_search
-> 返回激活引用和可调用面变化
-> 下一轮可用 tools 真被重建
-> 模型优先调用 activated skill tool
```

也就是说，旧版 search 更像“推荐”。
新版 search 已经是“动态改变后续能力面”。

## 6. skill-specific schema 是怎么接进来的

如果 skill frontmatter 里声明了：

- `tool_description`
- `tool_contract`
- `tool_output_contract`
- `tool_input_schema`

那么激活后的 skill tool 就不再只是：

```json
{"task":"..."}
```

而是可以直接暴露：

- `premise`
- `protagonist`
- `power`
- `setting`
- `must_use`
- `constraints`

这种结构化输入。

这让当前 Go runtime 比之前更接近“真实工具 schema”的感觉。

## 7. 为什么还保留 `skill_call`

`skill_call` 还在，但现在是兼容兜底层，不是主路径。

保留它的原因是：

1. 旧 prompt / 旧习惯仍然能过渡
2. 未来某些 skill-specific tool 还没完全稳定时，有一个统一 fallback
3. 出问题时更容易做回退

但从 router prompt 到 tool spec 设计，当前都在明确鼓励模型优先调用 activated skill tool。

## 8. 文件工具层现在怎么接入

当前本地工具不在 router 顶层直接暴露，而是在 skill executor 内部按需暴露。

也就是说：

- router 负责发现和激活 skill
- skill executor 负责在被激活后，决定是否读写文档，或是否执行终端命令

这很适合小说工作流，因为：

- 搜索和激活逻辑保持干净
- 文档操作更贴近 skill 自己的业务上下文
- shell 操作也被限制在 skill 局部上下文里，不会污染 router 层
- 产物能直接落到 markdown 文件，而不是只能回到终端

## 9. 响应分析现在怎么落盘

当前 router 和 skill executor 每轮响应后，都会额外生成：

- `router/round-XX-response-analysis.json`
- `skill-calls/<skill>/round-XX-response-analysis.json`

这里面会直接告诉你：

- 这一轮有没有 `tool_calls`
- tool name 是什么
- finish reason 是什么
- 文本内容是什么

这让“看模型到底是不是在调工具”变得简单很多。

## 10. 当前已知边界

### 边界 1：还没有原生 `tool_reference`

我们现在的 `tool_reference_like` 是 Go runtime 里的等价物，不是 wire-level 的原生 `tool_reference`。

### 边界 2：retained pool 还只是单次 Run 内记忆

它还没有变成 transcript / mem 驱动的跨轮恢复机制。

### 边界 3：skill 依然会进入一次 skill executor 模型调用

也就是说，我们现在更像：

```text
发现 skill 工具
-> 把结构化参数交给 skill executor
```

而不是：

```text
发现真实底层工具
-> 直接在同一模型回合里完成一切
```

但对“小说 skill runtime”这个产品目标来说，这条路是合理的。

## 11. 当前最值得继续打磨的方向

按优先级看，下一步最值钱的是：

1. 增加“未 discover 先误调用”的自修复提示
2. 等 mem 系统稳定后，再做跨轮 discovered memory
3. 在 tool 层稳定后，再继续扩 `opening / outline / rewrite` 这类 skill contract
