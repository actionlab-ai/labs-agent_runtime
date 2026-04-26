# 完整请求流程

这份文档专门描述当前 Go runtime 里一条完整请求，是如何从“用户说第一句话”一步步走到“本地组装 skill / tools / prompt 并发给模型”的。

它对应的是你本地 `novelcode` 里这几层能力的 Go 版落地：

- `claude.ts` 的请求装配
- `toolSearch.ts` 的 deferred tools 发现
- `messages.ts` 的消息预处理思路
- `toolToAPISchema()` 的最终 tool schema 注入

当前 Go 版没有照搬它的全部历史复杂度，但已经把主骨架补出来了。

## 1. 总流程

当前完整链路可以概括成：

```text
用户输入
-> Router 组装 round assembly
-> 注入 system prompt blocks
-> 注入 available deferred skills 提醒
-> 注入 retained skill tools 提醒
-> 拼出最终 router chat request
-> 发给模型
-> 分析模型回复是否带 tool_calls
-> 若调用 tool_search，更新 activation / retained pool
-> 若调用 activated skill tool，进入 skill executor
-> skill executor 组装 skill assembly
-> 注入 executor system prompt
-> 注入 compiled skill prompt
-> 注入 skill 允许的本地工具 schema
-> 发给模型
-> 分析 skill 回复是否继续调本地工具
-> 如有需要执行 Read / Write / Edit / Glob / Bash / PowerShell
-> 把 tool result 回填给 skill executor
-> skill 完成后返回正文或文档摘要
```

## 2. Router 这一层现在做了什么

### 2.1 输入

Router 现在内部维护的是“会话 conversation”，而不是把 system prompt 固定写死在全局消息里。

每一轮真正发请求前，runtime 会重新组装：

- `system prompt blocks`
- `deferred skills reminder`
- `retained skill tools reminder`
- `live conversation`
- `tool specs`

### 2.2 预处理结果

当前每一轮都会生成：

- `router/round-XX-assembly.json`
- `router/round-XX-assembly.md`

这里面会看到：

- 当前 round
- 当前 retained pool
- 当前有哪些 deferred skills 还没 discover
- 当前最终给模型的 messages
- 当前最终给模型的 tools
- 当前最终 chat request

这层设计就是为了让 Go runtime 更像 `novelcode` 的“请求装配器”。

### 2.3 novelcode 风格提醒

当前 Router 会在模型真正看到用户原始输入前，先插入两个 meta reminder：

1. `<available-deferred-skills>`
   - 告诉模型本地还有哪些 skill 可以通过 `tool_search` 发现
   - 同时明确提示 `select:<skill_id>` 这种 query form
2. `<retained-skill-tools>`
   - 告诉模型哪些 activated skill tools 当前已经是可调用的
   - 明确鼓励优先用它们，而不是退回 `skill_call`

这对应的是 `novelcode` 里：

- `<available-deferred-tools>`
- `tool_reference` discover 之后的可调用面变化

## 3. 模型回复后，runtime 怎么判断下一步

每轮 router 请求返回后，runtime 还会额外生成：

- `router/round-XX-response-analysis.json`

这里会总结：

- `finish_reason`
- 是否有 `tool_calls`
- tool call names
- 文本内容

也就是说，现在不仅能看 raw response，还能直接看“这一轮模型到底有没有调工具、调了哪个工具”。

## 4. skill executor 这一层现在做了什么

当 Router 命中某个 activated skill tool 后，会进入 skill executor。

这时候 runtime 会：

1. 延迟加载完整 `SKILL.md`
2. 根据 frontmatter 生成 compiled skill prompt
3. 根据 `allowed_tools` 决定要不要暴露：
   - `Read`
   - `Write`
   - `Edit`
   - `Glob`
   - `Bash`
   - `PowerShell`
4. 把这些本地工具 schema 一起注入 skill 模型请求

同样，每轮 skill 请求前也会落盘：

- `skill-calls/<skill>/round-XX-assembly.json`
- `skill-calls/<skill>/round-XX-assembly.md`

这里能看到：

- 当前 skill id
- 当前 compiled prompt
- 当前本地工具包
- 当前最终 skill chat request

## 5. skill 模型回复后怎么继续

skill executor 每轮也会生成：

- `skill-calls/<skill>/round-XX-response-analysis.json`

如果模型回复里带了本地工具调用：

- runtime 会执行工具
- 把 tool result 追加回 conversation
- 再发下一轮 skill request

如果不带 tool call：

- 就认为这次 skill 已完成
- 输出正文或文档摘要

## 6. 现在和 novelcode 最像的地方

已经比较接近的部分：

1. 请求不是裸发，而是先 assembly
2. deferred pool 会在请求前被显式注入提醒
3. retained callable surface 会在请求前被显式注入提醒
4. 请求前和响应后都有结构化落盘
5. tool / skill 调用链已经能被逐轮回放

## 7. 还没完全一比一的地方

还没做满的部分：

1. 还没有 wire-level 原生 `tool_reference`
2. 还没有 `novelcode/messages.ts` 那种完整的 transcript repair / pairing / compaction 体系
3. 还没有跨轮 transcript-derived discovered memory
4. 还没有 `toolToAPISchema()` 那么重的 provider-aware beta / cache / defer_loading 细节

但对你当前这个“小说 skill runtime”目标来说，主链已经足够完整，而且终于可以被人类看清楚了。
