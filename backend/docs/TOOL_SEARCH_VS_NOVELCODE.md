# 当前 `tool_search` 与 `novelcode` 的差距

这份文档基于你本地桌面的 `novelcode` 源码做对照：

- `C:\Users\admin\Desktop\novelcode\src\utils\toolSearch.ts`
- `C:\Users\admin\Desktop\novelcode\src\tools\ToolSearchTool\ToolSearchTool.ts`
- `C:\Users\admin\Desktop\novelcode\src\services\api\claude.ts`
- `C:\Users\admin\Desktop\novelcode\src\services\compact\compact.ts`

## 1. 先说结论

当前 Go runtime 和 `novelcode` 已经不是“完全两套思路”了。

现在更准确的说法是：

```text
novelcode：deferred tool discovery
当前 Go runtime：deferred skill discovery
```

也就是说：

- 大方向已经靠近
- 但抽象层级和记忆机制还没完全对齐

## 2. 已经对齐到哪

### 已经很接近的部分

1. 首轮先 search
2. search 不再只是返回推荐列表，而是会改变下一轮可调用面
3. 支持：
   - bare exact query
   - `select:...`
   - `+required`
4. skill body 只在执行时延迟加载
5. 激活后的 skill tool 可以保留在 retained pool 里

### 新近补上的部分

1. `tool_search` 返回 `tool_reference_like`
   - 它是当前 Go runtime 对 `tool_reference` 的等价模拟
2. activated skill tool 可以暴露 `tool_input_schema`
   - 不再只能是 `task:string`
3. request assembly 已经显式落盘
   - router 和 skill executor 在发模型前都会先 build assembly
4. response analysis 已经显式落盘
   - 现在可以逐轮看到模型有没有调工具、调了哪个工具

## 2.5 新近补上的“更像 novelcode 的请求装配层”

你本地 `novelcode` 在 `claude.ts` 里真正强的，不只是 `ToolSearch` 本身，而是：

- 先决定 deferred tools 怎么处理
- 再做 messages 预处理
- 再 build tool schemas
- 再 build 最终 API request

当前 Go runtime 现在也补上了一个更轻量但同方向的 assembly 层：

- `router/round-XX-assembly.json`
- `skill-calls/<skill>/round-XX-assembly.json`

这意味着现在不是只能看：

- raw request
- raw response

而是还能看：

- 这一轮到底有哪些 deferred skills 被提醒给模型
- 这一轮 retained skill tools 是怎么被提醒给模型的
- 这一轮最终 messages / tools / prompt 是怎么拼出来的

## 3. 仍然存在的核心差距

### 差距一：没有原生 `tool_reference`

`novelcode` 在 wire 上返回的是真正的 `tool_reference` block。

当前 Go runtime 做的是：

```text
tool_search
-> 返回 tool_reference_like
-> runtime 在下一轮重建 tools 列表
```

这已经很像，但 still 不是同一层。

### 差距二：没有跨轮 discovered memory

`novelcode` 会从消息历史里恢复 discovered tools。

当前 Go runtime 还只是：

```text
runState.RetainedSkills
```

也就是：

- 单次 `Run` 内没问题
- 还不是 transcript / mem 驱动的跨轮恢复

这一项目前已明确暂缓，等 mem 系统稳定后再做。

### 差距三：发现的是 skill，不是最终底层真实工具

`novelcode` 更接近：

```text
发现真实工具 schema
-> 模型直接调用真实工具
```

当前 Go runtime 更接近：

```text
发现 skill
-> 暴露 skill-specific schema
-> skill executor 再做一次受控模型调用
```

这不是错，而是当前产品形态的抽象选择。

### 差距四：消息预处理体系还远比 `novelcode` 轻

`novelcode` 的 `messages.ts` 里有一整套重处理逻辑：

- 合并消息
- 清洗 tool_reference blocks
- pairing repair
- transcript compaction 兼容
- tool_result 修复

当前 Go runtime 现在有了 request assembly，但还没有那套超重的 transcript repair 体系。

### 差距五：还没有“先 discover 再 retry”的自修复提示

`novelcode` 还有一个很实用的防呆层：

- 如果模型直接调用了还没 discover 的 deferred tool
- 系统会提示它：
  - 先 `ToolSearch("select:xxx")`
  - 再 retry

当前 Go runtime 还没把这层补上。

## 4. 当前真实定位

所以今天的系统不应该再被描述成：

```text
一个普通的 metadata search + generic dispatcher
```

而应该描述成：

```text
一个 novelcode 风格的 Go 版 deferred skill runtime
```

只是它当前：

- 还没有原生 `tool_reference`
- 还没有跨轮 discovered memory
- 还没有完全退掉 skill executor 这一层

## 5. 现在最值钱的继续方向

按优先级看：

1. 先补“未 discover 先误调用”的自修复提示
2. 再等 mem 系统稳定后做跨轮 discovered memory
3. 然后才是继续扩小说 skill contracts 和 skill 库

这条顺序能保证：

- 先把操作系统打稳
- 再把应用层技能库做大
