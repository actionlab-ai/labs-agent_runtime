# 固定 Workflow 到模型请求的真实组装

这份文档只讲两条固定链：

1. `POST /v1/workflows/project-kickoff`
2. `POST /v1/workflows/project-kernel`

## 1. 代码入口

固定 workflow 的 HTTP 入口在：

- [workflows.go](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/httpapi/workflows.go)

workflow 编排和持久化在：

- [workflow_service.go](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/httpapi/workflow_service.go)
- [workflow_plan.go](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/httpapi/workflow_plan.go)
- [workflow_persistence.go](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/httpapi/workflow_persistence.go)

skill 执行和模型请求组装在：

- [runtime.go](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/runtime/runtime.go)
- [file_tools.go](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/runtime/file_tools.go)
- [model.go](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/model/model.go)

## 2. 真实流程

这里不是把 `skill_id` 直接传给模型。

真实流程是：

```text
HTTP 请求
-> internal/httpapi 选中固定 workflow
-> workflow 选中固定 skill
-> runtime 读取 SKILL.md 和 frontmatter
-> 组装 compiled prompt
-> 组装 tools
-> POST 到 <base_url>/chat/completions
-> 收到模型输出
-> 服务端判断这是“正式文档”还是“澄清问题”
-> 只有正式文档才写回项目文档 provider
```

## 3. 模型真正拿到的内容

发给模型的不是裸 `skill_id`，而是一份编译后的 prompt。

结构是：

1. `# Skill Metadata`
2. `# Skill Tooling`
3. `# Base Directory`
4. `# Activated Tool Arguments`
5. `# Skill Instruction`
6. `# Original User Request`
7. `# Current Skill Task`
8. `# Output Rule`

所以模型看到的是：

- 当前 skill 是谁
- skill 自己的硬规则
- 允许调用哪些工具
- 本轮实参是什么
- 用户原始请求是什么
- 当前只需要完成哪一项任务

## 4. `project-kernel` 新增的澄清分支

`project-kernel` 现在固定传入：

```json
{
  "task": "<用户输入>",
  "question_mode": "clarify_first",
  "document_kind": "novel_core"
}
```

`novel-emotional-core` 必须只输出两种结果之一：

### 正式文档模式

第一行标题必须是：

```markdown
# 小说情感内核
```

此时服务端会：

- 判定 `response_mode=document`
- 判定 `needs_input=false`
- 把全文写成 `novel_core`

### 澄清模式

第一行标题必须是：

```markdown
# 需要补充的信息
```

此时服务端会：

- 判定 `response_mode=clarification`
- 判定 `needs_input=true`
- 不写入 `novel_core`

## 5. 工具也会一起发给模型

如果当前 skill 允许本地工具，或者项目模式已启用，模型请求里还会带这些 tool schema：

- `Read`
- `Write`
- `Edit`
- `Glob`
- `ListProjectDocuments`
- `ReadProjectDocument`
- `WriteProjectDocument`
- `Bash`
- `PowerShell`

当前机制是：

- prompt 负责说明目标与规则
- tools 负责暴露可以执行的能力

## 6. 直接调用你自己的 API

### kickoff

```bash
curl -X POST http://127.0.0.1:8080/v1/workflows/project-kickoff \
  -H "Content-Type: application/json" \
  -d '{
    "project": "urban-safe-growth",
    "model": "deepseek-flash",
    "input": "男频、番茄、都市重生、安全感、持续变强。主角前期弱，但必须让读者从第一阶段就感觉他有自己的绝对安全区。",
    "debug": true
  }'
```

### kernel

```bash
curl -X POST http://127.0.0.1:8080/v1/workflows/project-kernel \
  -H "Content-Type: application/json" \
  -d '{
    "project": "urban-safe-growth",
    "model": "deepseek-flash",
    "input": "围绕安全感、尊严和持续变强，设计这本书真正的情感内核。如果关键信息不足，先反问我，不要直接定稿。",
    "debug": true
  }'
```

## 7. 最底层模型 curl

如果你想直接看到服务端到底向模型发了什么：

1. 先跑一次 workflow
2. 再看：
   - `skill-calls/<skill-id>/compiled-prompt.md`
   - `skill-calls/<skill-id>/round-01-chat-request.json`

最小模型请求模板是：

```bash
curl -X POST https://api.deepseek.com/chat/completions \
  -H "Authorization: Bearer <YOUR_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-flash",
    "messages": [
      {
        "role": "user",
        "content": "<compiled-prompt.md 的全文>"
      }
    ],
    "tools": [
      "<round-01-chat-request.json 里的 tools 数组>"
    ],
    "temperature": 0.7,
    "max_tokens": 32768
  }'
```

关键点只有两个：

- 不是传 `skill_id`
- 而是传“skill 编译后的 prompt + tools”
