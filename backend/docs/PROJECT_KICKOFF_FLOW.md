# 项目初始化流程

现在推荐把新书起盘拆成两步，不要混成一个大流程：

1. `project-kickoff`
2. `project-kernel`

## 第一步：项目定调

接口：
`POST /v1/workflows/project-kickoff`

固定执行的 skill：
- `novel-project-kickoff`

它负责：

- 平台感
- 目标读者
- 核心卖点
- 风格边界
- 禁区

它正常会回写 4 份项目文档：

- `project_brief`
- `reader_contract`
- `style_guide`
- `taboo`

如果信息不足，它也可以先反问；这时通常不会产生这些正式文档。

## 第二步：情感内核

接口：
`POST /v1/workflows/project-kernel`

固定执行的 skill：
- `novel-emotional-core`

它负责：

- 读者现实里的情绪缺口
- 主角最深情绪缺口
- 压迫系统
- 爽感兑现机制
- 长线情绪承诺

它对应的正式项目文档是：

- `novel_core`

## `project-kernel` 现在的两种返回模式

### 1. 正式定稿

当信息足够时，API 会返回：

```json
{
  "workflow_id": "project-kernel",
  "stage": "kernel",
  "response_mode": "document",
  "needs_input": false,
  "updated_documents": [
    {
      "kind": "novel_core"
    }
  ]
}
```

这表示：

- 模型输出的是正式 `# 小说情感内核`
- 服务端认定它可以作为 canon
- 结果已落为 `novel_core`

### 2. 反问澄清

当信息不足时，API 会返回：

```json
{
  "workflow_id": "project-kernel",
  "stage": "kernel",
  "response_mode": "clarification",
  "needs_input": true,
  "updated_documents": [],
  "final_text": "# 需要补充的信息\n..."
}
```

这表示：

- 模型没有被允许靠猜测定稿
- 这轮只是收集缺失信息
- 不会误写 `novel_core`

## `novel-emotional-core` 的关键信息判定

它现在先检查 6 个锚点：

1. 题材、平台感或目标读者
2. 主角当前处境
3. 主角最深情绪缺口
4. 外部压迫系统
5. 读者想得到的代偿或爽感
6. 禁区或价值观边界

满足任意一条就会先反问：

- 6 个锚点里明确项少于 4 项
- 第 3、4、5 项里缺任意 2 项
- 只有题材感觉，没有主角当前处境
- 只能靠猜测才能补全核心动机、压迫来源或代偿方式

## 这条链的工程化约束

- workflow 固定传入 `question_mode=clarify_first`
- skill 明确区分“澄清模式”和“定稿模式”
- HTTP 服务端不会把澄清输出误持久化成 `novel_core`
- `response_mode` 和 `needs_input` 是前端判断 UI 分支的直接依据

## 一句话关系

- `kickoff` 决定这本书卖什么
- `kernel` 决定这本书为什么打动人
