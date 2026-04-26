# 文件工具与文档落地

这份文档专门说明当前 runtime 新接入的文件操作层，以及为什么它对小说 skill 很关键。

## 1. 为什么现在要做这层

你现在的目标已经很明确了：

```text
小说 skill 的最终产物，不应该只是吐给终端
而应该尽量落成文档
```

所以这次不是只补了几个工具名，而是把 skill executor 升级成了可以像 `novelcode` 一样，在 skill 内部继续走文件工具回合。

## 2. 当前接入了哪些工具

当前 skill executor 内部支持四个文件工具和两个 shell 工具：

- `Read`
  - 读取文本文件，可带 `offset / limit`
- `Write`
  - 创建或整文件覆盖
- `Edit`
  - 基于 `old_string -> new_string` 做定位替换
- `Glob`
  - 按 glob 模式匹配文件
- `Bash`
  - 执行 bash 终端命令
- `PowerShell`
  - 执行 powershell 终端命令

当前实现说明：

- `PowerShell` 会优先尝试 `pwsh`，再回退到 `powershell`
- `Bash` 会优先尝试系统里的 `bash`，再尝试常见 Git Bash 路径
- 如果 `Bash` 的可执行文件不在常规位置，可以通过环境变量 `NOVEL_BASH_PATH` 指定

这些名字刻意贴近 `novelcode`：

- `Read`
- `Write`
- `Edit`
- `Glob`
- `Bash`
- `PowerShell`

## 3. 行为上尽量对齐了什么

### 已对齐

1. `Write` 覆盖已有文件前，要求先 `Read`
2. `Edit` 修改已有文件前，要求先 `Read`
3. 如果文件在 read 之后又被改过，会阻止继续写
4. `Glob` 返回工作区相对路径
5. skill 现在可以自己把文档写到 workspace，而不是只能吐文本
6. `Bash / PowerShell` 的 tool description 会明确要求模型优先使用 `Glob / Read / Edit / Write` 处理文件操作

### 当前简化版

和 `novelcode` 相比，当前 Go 版还是轻量一些：

- 只处理文本文件
- 没做图片 / PDF / notebook 读取
- 没接权限系统和 LSP 通知
- 没做复杂 diff 展示
- shell 工具目前是轻量版，没有做 novelcode 那么重的权限和安全策略

但对“小说内容落 markdown 文档”这个目标来说，已经够用，而且路径是对的。

## 4. 默认文档落地目录

当前默认 workspace root 是：

```text
仓库根目录
```

当前默认文档输出目录是：

[docs/08-generated-drafts](../../docs/08-generated-drafts/README.md)

也就是说，skill 在没收到显式 `document_path` 时，应该优先把小说产物写到这里。

## 5. opening skill 现在怎么用这层

`webnovel-opening-sniper` 现在已经：

1. 允许使用 `Read / Write / Edit / Glob`
2. 允许使用 `Bash / PowerShell`
3. 在 schema 里支持可选 `document_path`
4. 在 skill 说明里写明：
   - 有文件工具时，默认要落 markdown
   - 写完后只返回简短路径摘要
   - 除非用户明确要求 chat-only output，否则不要把整篇正文再重复回终端
   - shell 只用于终端任务，不替代文件工具

## 6. 这层能力的真正价值

这让当前 runtime 从：

```text
skill 输出一段文本
```

推进成：

```text
skill 生成内容
-> skill 自己把内容写入文档
-> runtime 返回路径摘要
```

这一步很重要，因为后面无论是：

- 开篇
- 大纲
- 改写
- 设定卡
- 角色卡

都更适合变成可持续编辑的 markdown 文档，而不是一次性聊天回复。
