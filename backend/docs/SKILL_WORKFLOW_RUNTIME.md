# Skill / Workflow Runtime 边界

这份文档回答两个问题：

1. skill 是怎么被包装成真正模型请求的
2. 为什么 `project-kickoff` 和 `project-kernel` 要做成固定 workflow

## 结论

现在有两种执行面：

- `Runtime.Run`
  - 通用 router
  - 会走 `tool_search`

- `WorkflowRunner + RuntimeSkillRunner`
  - 固定点名 skill
  - 不走 `tool_search`

`project-kickoff` 和 `project-kernel` 都属于第二类。

## skill 到模型请求的真实包装

一个固定 workflow 最终不是把“skill 名字”直接发给模型。

真正过程是：

```text
HTTP workflow
  -> 选定 skill_id
  -> Runtime.ExecuteSkill
  -> 读取 skill frontmatter + SKILL.md 正文
  -> 拼出 compiled prompt
  -> 注入项目上下文和工具提示
  -> 组装 tools
  -> 发 chat/completions
```

对应代码入口：

- [Runtime.ExecuteSkill](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/runtime/runtime.go:343)
- [ComposeSkillPrompt](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/runtime/runtime.go:411)
- [skillDocumentHint](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/runtime/file_tools.go:768)

## compiled prompt 里到底有什么

当前 skill executor 发给模型的主消息，不是简单一句“请执行某 skill”，而是一个完整拼装后的 prompt。

内容结构是：

1. `# Skill Metadata`
2. `# Skill Tooling`
3. `# Base Directory`
4. `# Activated Tool Arguments`
5. `# Skill Instruction`
6. `# Original User Request`
7. `# Current Skill Task`
8. `# Output Rule`

也就是：

- 告诉模型我们想干什么
- 告诉模型现在给了什么参数
- 告诉模型当前有哪些工具能用
- 把完整 skill 说明文贴进去
- 再把原始用户请求带进去

所以这里不是“只传 skill 名”，而是“skill 被编译成完整上下文后再发模型”。

## 为什么固定 workflow 比 router 更适合前两步

### `/v1/workflows/project-kickoff`

固定执行：

- `novel-project-kickoff`

固定目标：

- `project_brief`
- `reader_contract`
- `style_guide`
- `taboo`

### `/v1/workflows/project-kernel`

固定执行：

- `novel-emotional-core`

固定目标：

- `novel_core`

既然这两步的职责和目标文档都固定，就不应该让 router 先 search 再猜。

## workflow 和 skill 的关系

不是一对一等价关系。

- skill：能力定义
- workflow：产品流程

在现在这版里：

- `project-kickoff` workflow 直接点名调用 `novel-project-kickoff`
- `project-kernel` workflow 直接点名调用 `novel-emotional-core`
- `project-bootstrap` workflow 只是一个组合入口，允许继续往后扩

## 服务端后处理

固定 workflow 跑完 skill 后，服务端还会再做一层持久化兜底。

### kickoff

从输出里提取固定分节：

- `## project_brief`
- `## reader_contract`
- `## style_guide`
- `## taboo`

然后写回项目文档。

### kernel

不再做分节提取，而是把最终输出整体写成：

- `novel_core`

这是因为 `novel-emotional-core` 的输出本来就是一个完整内核文档，不是四分节型定调文档。
