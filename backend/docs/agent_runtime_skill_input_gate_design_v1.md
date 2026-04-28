# Agent Runtime 精准多轮需求收集与 Skill 调用设计文档 v1

> 适用项目：`actionlab-ai/labs-agent_runtime/backend`  
> 设计目标：在 Go HTTP Runtime + Filesystem/Project Context Provider 的基础上，实现一套**不依赖模型猜测**的精准多轮需求收集机制，让每个 Skill 只处理“已结构化、已校验、已补齐”的输入。

---

## 0. 一句话结论

把每个 Skill 看成一个 Form；  
把 Skill 需要的输入看成 Slot；  
把 Thread State 看成 Session；  
把 SkillInputGate 看成 Form Runner；  
把缺字段返回看成 Interrupt；  
把用户下一轮补充看成 Resume；  
字段齐全后再调用真正的 SkillExecutor。

```text
用户
  ↓
HTTP Chat API
  ↓
ConversationRuntime
  ↓
ThreadState Repository
  ↓
SkillInputGate
  ├─ 字段不足 → clarification_required
  └─ 字段齐全 → SkillExecutor → 模型执行目标 Skill
```

---

## 1. 背景问题

当前 Agent Runtime 已经有一次请求内的执行逻辑，例如：

```text
用户输入
  → router
  → tool_search
  → skill 激活
  → skill executor
  → 模型返回
```

但还缺一个关键能力：

```text
跨 HTTP 请求的多轮需求收集。
```

例如用户说：

```text
帮我写一个都市异能开头
```

此时信息明显不够。系统不能直接调用 `opening_v1` 让模型瞎写，而应该问：

```text
1. 主角是谁？
2. 开篇钩子是什么？
3. 第一章直接冲突是什么？
```

用户第二轮补充：

```text
主角是外卖员，能看到死者最后三分钟执念。
```

用户不应该传上一轮历史。系统应该根据 `thread_id` 自己恢复状态，知道：

```text
已知：genre = 都市异能
新增：protagonist = 外卖员，能看到死者最后三分钟执念
仍缺：hook, first_conflict
```

这就是本文档要解决的问题。

---

## 2. 设计目标

### 2.1 功能目标

实现一套 deterministic SkillInputGate，具备：

```text
1. 根据 Skill 的 input_contract 判断需要哪些字段。
2. 从用户输入、结构化 answers、项目上下文中填充字段。
3. 记录每个字段的来源、状态、更新时间。
4. 校验 required / enum / min_len / max_len / pattern。
5. 字段缺失时返回 clarification_required。
6. 字段齐全后生成 invocation_args。
7. runtime 调用真正的 SkillExecutor。
8. 用户多轮补充时，只需要传 thread_id + 当前消息。
```

### 2.2 非目标

第一版不要做：

```text
1. 不依赖模型理解用户自由文本。
2. 不让每个 Skill 自己处理多轮追问。
3. 不把多轮状态只存在 Redis。
4. 不让前端每次传完整 history。
5. 不把 chat history 原样全部塞给 skill。
```

### 2.3 工程目标

```text
1. 可测试。
2. 可恢复。
3. 可回放。
4. 可追踪字段来源。
5. 可扩展到多个 Skill。
6. 可扩展到 Workflow。
```

---

## 3. 核心原则

### 3.1 Skill 无状态

错误设计：

```text
opening_v1 skill 自己记住用户上轮说了什么。
chapter_v1 skill 自己判断缺哪些字段。
每个 skill 都写一套追问逻辑。
```

正确设计：

```text
ConversationRuntime 负责 thread_state。
SkillInputGate 负责 slot filling。
SkillExecutor 负责目标 skill 执行。
```

### 3.2 Runtime 记忆，不是 Skill 记忆

模型和 Skill 都应该尽量无状态。

用户第二轮只传：

```json
{
  "message": "主角是外卖员，能看到死者最后三分钟执念。"
}
```

路径中携带 thread_id：

```http
POST /v1/chat/threads/{thread_id}/messages
```

Runtime 根据 thread_id 恢复：

```text
1. active_skill_id
2. slot_state
3. pending_questions
4. project_context
5. recent messages
```

### 3.3 PostgreSQL 主存储，Redis 只缓存

```text
PostgreSQL:
  agent_threads
  agent_messages
  agent_runs
  slot_state
  pending_questions

Redis:
  active thread cache
  thread lock
  recent state snapshot
```

Redis 丢失后，系统必须能从 PostgreSQL 恢复。

### 3.4 Gate 尽量不用模型

Gate 的职责不是“智能猜测”，而是“按字段契约收口”。

Gate 只做：

```text
字段匹配
别名识别
结构化 answers 合并
pending question 对齐
默认值填充
规则校验
缺失追问
```

### 3.5 Skill 只接收干净参数

SkillExecutor 最终收到：

```json
{
  "skill_id": "opening_v1",
  "invocation_args": {
    "genre": "都市异能",
    "protagonist": "外卖员，能看到死者最后三分钟执念",
    "hook": "接到三天前死人的订单",
    "first_conflict": "送餐到凶宅时被真正凶手发现"
  }
}
```

而不是一坨混乱聊天历史。

---

## 4. 概念模型

| 概念 | 含义 | 类比 |
|---|---|---|
| Thread | 一段用户和系统的多轮会话 | Session |
| Message | 用户/助手/工具/Skill 消息 | Chat History |
| Skill | 具体能力，例如开篇生成 | Action |
| Input Contract | Skill 的输入字段契约 | Form Schema |
| Slot | 某个字段当前值 | Slot / Parameter |
| Slot State | 当前 Skill 已收集字段状态 | Session Parameters |
| SkillInputGate | 字段收集、校验、追问器 | Form Runner |
| Clarification | 缺字段时返回给用户的问题 | Prompt |
| Invocation Args | 字段齐全后传给 Skill 的参数 | Filled Form Result |

---

## 5. 总体架构

```text
/v1/chat/threads
/v1/chat/threads/{thread_id}/messages
/v1/chat/threads/{thread_id}
        |
        v
ConversationRuntime
        |
        |-- load thread_state
        |-- save user message
        |-- determine active_skill_id
        |-- load skill input_contract
        |-- load project_context
        |
        v
SkillInputGate
        |
        |-- merge structured_answers
        |-- extract from latest_user_message
        |-- fill from project_context
        |-- apply defaults
        |-- validate fields
        |-- build questions or invocation_args
        |
        v
ConversationRuntime
        |
        |-- save updated slot_state
        |
        |-- if not ready:
        |       save assistant clarification message
        |       return clarification_required
        |
        |-- if ready:
                call SkillExecutor
                save assistant skill_result message
                return skill_result
```

---

## 6. 目录结构建议

新增目录：

```text
backend/internal/conversation/
  runtime.go
  service.go
  types.go

backend/internal/gate/
  contract.go
  slot_state.go
  extractor.go
  validator.go
  question_builder.go
  runner.go
  errors.go

backend/internal/repository/
  thread_repo.go
  message_repo.go

backend/internal/httpapi/
  chat_service.go

backend/migrations/
  00xx_agent_threads.sql
  00xx_agent_messages.sql
  00xx_agent_runs_thread_id.sql
```

---

## 7. Skill Input Contract 设计

每个 Skill 必须声明自己的输入契约。

示例：`skills/opening_v1/SKILL.md`

```yaml
---
id: opening_v1
name: 网文开篇狙击手
description: 生成黄金600字强钩子开篇
user_invocable: true

input_contract:
  version: 1
  mode: form

  fields:
    genre:
      label: 小说类型
      type: enum
      required: true
      options:
        - 都市异能
        - 玄幻
        - 仙侠
        - 科幻
        - 悬疑
        - 都市日常
        - 系统流
      aliases:
        - 类型
        - 题材
        - 分类
        - genre
      examples:
        - 都市异能
        - 玄幻系统流
      ask: 这本小说是什么类型？比如都市异能、玄幻、仙侠、悬疑。
      source_priority:
        - structured_answers
        - latest_user_message
        - project_context
        - default

    protagonist:
      label: 主角设定
      type: text
      required: true
      min_len: 8
      max_len: 800
      aliases:
        - 主角
        - 男主
        - 女主
        - protagonist
        - 主人公
      ask: 主角是谁？请给出身份、核心能力或当前困境。
      examples:
        - 外卖员，能看到死者最后三分钟的执念
        - 废柴宗门弟子，意外绑定残缺系统

    hook:
      label: 开篇钩子
      type: text
      required: true
      min_len: 8
      max_len: 800
      aliases:
        - 钩子
        - 爆点
        - 开篇事件
        - 异常事件
        - 开头事件
      ask: 开篇第一钩子是什么？也就是读者前30秒看到的异常事件。
      examples:
        - 主角接到三天前死者的外卖订单
        - 主角醒来发现自己被全城通缉

    first_conflict:
      label: 第一幕冲突
      type: text
      required: true
      min_len: 8
      max_len: 1200
      aliases:
        - 冲突
        - 第一冲突
        - 第一章矛盾
        - 直接冲突
        - 第一幕冲突
      ask: 第一章里主角马上遇到的直接冲突是什么？谁逼他、什么危险、代价是什么？
      examples:
        - 主角送餐到凶宅，被真正凶手发现，必须十分钟内逃出去

    style:
      label: 文风要求
      type: text
      required: false
      default: 无AI味、短句推进、强冲突、少解释、多行动
      aliases:
        - 文风
        - 风格
        - 口吻
        - style
      ask: 有特殊文风要求吗？没有则使用默认网文强钩子风格。

  ask_policy:
    max_questions_per_turn: 3
    include_known_fields: true
    include_examples: true
    prefer_structured_form: true

  execution_policy:
    execute_when_required_filled: true
    allow_default_for_optional: true
    reject_unknown_required: true
---
```

---

## 8. Slot State 设计

`slot_state` 是 thread 当前正在收集的结构化字段。

示例：

```json
{
  "active_skill_id": "opening_v1",
  "status": "collecting_fields",
  "fields": {
    "genre": {
      "key": "genre",
      "value": "都市异能",
      "status": "filled",
      "source_type": "user_message",
      "source_id": "m_001",
      "confidence": "explicit",
      "updated_at": "2026-04-29T10:00:00Z"
    },
    "protagonist": {
      "key": "protagonist",
      "value": null,
      "status": "missing"
    },
    "hook": {
      "key": "hook",
      "value": null,
      "status": "missing"
    },
    "first_conflict": {
      "key": "first_conflict",
      "value": null,
      "status": "missing"
    },
    "style": {
      "key": "style",
      "value": "无AI味、短句推进、强冲突、少解释、多行动",
      "status": "defaulted",
      "source_type": "contract_default"
    }
  },
  "pending_fields": [
    "protagonist",
    "hook",
    "first_conflict"
  ],
  "last_questions": [
    {
      "field": "protagonist",
      "question": "主角是谁？请给出身份、核心能力或当前困境。"
    }
  ]
}
```

字段状态枚举：

```text
missing      缺失
filled       用户明确提供
defaulted    使用默认值
invalid      有值但校验失败
conflicted   与已有项目上下文冲突
confirmed    用户确认后的值
```

来源类型：

```text
structured_answers
latest_user_message
user_message
project_context
contract_default
system_inference
```

第一版建议不要使用 `system_inference`，避免规则或模型乱猜。

---

## 9. 数据库设计

### 9.1 agent_threads

```sql
CREATE TABLE IF NOT EXISTS agent_threads (
    id UUID PRIMARY KEY,
    project_id UUID NULL,

    title TEXT NULL,
    status TEXT NOT NULL DEFAULT 'idle',

    active_skill_id TEXT NULL,
    active_workflow TEXT NULL,

    slot_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    pending_questions JSONB NOT NULL DEFAULT '[]'::jsonb,

    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_threads_project_id
ON agent_threads(project_id);

CREATE INDEX IF NOT EXISTS idx_agent_threads_status
ON agent_threads(status);
```

### 9.2 agent_messages

```sql
CREATE TABLE IF NOT EXISTS agent_messages (
    id UUID PRIMARY KEY,
    thread_id UUID NOT NULL REFERENCES agent_threads(id) ON DELETE CASCADE,

    role TEXT NOT NULL,
    content TEXT NOT NULL,

    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_messages_thread_id_created_at
ON agent_messages(thread_id, created_at);
```

`role` 取值：

```text
user
assistant
system
tool
skill
```

### 9.3 agent_runs 增加 thread_id

```sql
ALTER TABLE agent_runs
ADD COLUMN IF NOT EXISTS thread_id UUID NULL;

CREATE INDEX IF NOT EXISTS idx_agent_runs_thread_id
ON agent_runs(thread_id);
```

如果当前项目里 run 表名字不同，按实际表名调整。

---

## 10. Redis 设计

Redis 只做缓存和锁。

### 10.1 active thread cache

key：

```text
agent:thread:{thread_id}:state
```

value：

```json
{
  "thread_id": "t_001",
  "project_id": "p_001",
  "status": "collecting_fields",
  "active_skill_id": "opening_v1",
  "slot_state": {},
  "updated_at": "2026-04-29T10:00:00Z"
}
```

TTL：

```text
24h 到 7d
```

策略：

```text
读：先 Redis，miss 后读 PostgreSQL。
写：先 PostgreSQL，成功后更新 Redis。
恢复：Redis 丢失后从 PostgreSQL 重建。
```

### 10.2 thread lock

key：

```text
agent:thread:{thread_id}:lock
```

用途：

```text
防止用户连续快速发送多条消息导致 slot_state 并发覆盖。
```

建议：

```text
SET lock value NX EX 15
```

---

## 11. HTTP API 设计

### 11.1 创建对话线程

```http
POST /v1/chat/threads
```

请求：

```json
{
  "project_id": "p_001",
  "initial_message": "帮我写一个都市异能开头",
  "target_skill_id": "opening_v1"
}
```

`target_skill_id` 可选。  
如果不传，runtime 可以先走现有 router 或简单规则识别。

响应一：需要补字段

```json
{
  "type": "clarification_required",
  "thread_id": "t_001",
  "target_skill_id": "opening_v1",
  "known_fields": {
    "genre": "都市异能"
  },
  "missing_fields": [
    "protagonist",
    "hook",
    "first_conflict"
  ],
  "questions": [
    {
      "field": "protagonist",
      "label": "主角设定",
      "question": "主角是谁？请给出身份、核心能力或当前困境。"
    }
  ]
}
```

响应二：字段齐全，直接执行成功

```json
{
  "type": "skill_result",
  "thread_id": "t_001",
  "target_skill_id": "opening_v1",
  "run_id": "r_001",
  "content": "生成结果..."
}
```

### 11.2 向已有线程发送消息

```http
POST /v1/chat/threads/{thread_id}/messages
```

请求：

```json
{
  "message": "主角是外卖员，能看到死者最后三分钟执念。"
}
```

也支持结构化答案：

```json
{
  "answers": {
    "protagonist": "外卖员，能看到死者最后三分钟执念",
    "hook": "接到三天前死人的外卖订单"
  }
}
```

优先级：

```text
answers > message
```

响应格式同上。

### 11.3 获取线程详情

```http
GET /v1/chat/threads/{thread_id}
```

响应：

```json
{
  "thread_id": "t_001",
  "project_id": "p_001",
  "status": "collecting_fields",
  "active_skill_id": "opening_v1",
  "slot_state": {},
  "messages": [
    {
      "role": "user",
      "content": "帮我写一个都市异能开头"
    },
    {
      "role": "assistant",
      "content": "还缺主角设定、开篇钩子、第一幕冲突。"
    }
  ]
}
```

---

## 12. SkillInputGate 输入输出

### 12.1 GateInput

```go
type GateInput struct {
    ThreadID          string                 `json:"thread_id"`
    ProjectID         string                 `json:"project_id,omitempty"`
    TargetSkillID     string                 `json:"target_skill_id"`

    LatestUserMessage string                 `json:"latest_user_message,omitempty"`
    StructuredAnswers map[string]any         `json:"structured_answers,omitempty"`

    SlotState         SlotState              `json:"slot_state"`
    PendingQuestions  []PendingQuestion      `json:"pending_questions,omitempty"`

    SkillContract     SkillInputContract     `json:"skill_contract"`

    ProjectContext    map[string]any         `json:"project_context,omitempty"`
}
```

### 12.2 GateOutput

```go
type GateOutput struct {
    Ready            bool                    `json:"ready"`
    TargetSkillID    string                  `json:"target_skill_id"`

    UpdatedSlotState SlotState               `json:"updated_slot_state"`

    KnownFields      map[string]any          `json:"known_fields,omitempty"`
    MissingFields    []string                `json:"missing_fields,omitempty"`
    Violations       []FieldViolation        `json:"violations,omitempty"`
    Questions        []ClarificationQuestion `json:"questions,omitempty"`

    InvocationArgs   map[string]any          `json:"invocation_args,omitempty"`
}
```

---

## 13. Gate 内部处理流程

```text
1. Normalize Input
   - 清理空白字符
   - 统一中文冒号/英文冒号
   - 统一序号格式

2. Merge Structured Answers
   - 如果请求里有 answers，直接按 field key 合并
   - 校验 field 是否存在于 contract

3. Extract From Latest Message
   - 根据 aliases 做字段识别
   - 根据 pending_fields 做顺序填充
   - 根据 enum options 做枚举识别

4. Fill From Project Context
   - 如果 contract 允许从 project_context 填字段
   - 只填明确字段，不做猜测

5. Apply Defaults
   - optional 字段可以使用 default
   - required 字段默认不允许用 default，除非 contract 明确允许

6. Validate
   - required
   - type
   - enum
   - min_len
   - max_len
   - pattern
   - custom validators

7. Build Questions
   - 对 missing 字段生成 ask
   - 对 invalid 字段生成修正问题
   - 一轮最多 max_questions_per_turn 个

8. Build Invocation Args
   - 如果 required 字段全部通过
   - 只输出 skill 需要的字段
```

---

## 14. Extractor 规则设计

第一版不使用模型，只做规则解析。

### 14.1 结构化答案优先

用户传：

```json
{
  "answers": {
    "protagonist": "外卖员",
    "hook": "死人订单"
  }
}
```

直接填充。

### 14.2 字段别名匹配

配置：

```yaml
protagonist:
  aliases:
    - 主角
    - 男主
    - 女主
```

用户：

```text
主角是外卖员，能看到死者执念。
```

提取：

```json
{
  "protagonist": "外卖员，能看到死者执念"
}
```

### 14.3 冒号格式匹配

支持：

```text
主角：外卖员
主角: 外卖员
【主角】外卖员
1. 主角设定：外卖员
```

正则思路：

```text
(alias)\s*[:：]\s*(value)
```

### 14.4 pending field 顺序填充

如果上一轮只问了一个字段：

```json
[
  {
    "field": "first_conflict",
    "question": "第一章里主角马上遇到的直接冲突是什么？"
  }
]
```

用户无论怎么答，都优先填给 `first_conflict`。

如果上一轮问了多个字段，且用户没有明确字段名，则：

```text
1. 尝试按序号匹配
2. 尝试按换行拆分
3. 仍无法判断时，不强填，返回结构化问题
```

### 14.5 enum 识别

字段：

```yaml
genre:
  type: enum
  options:
    - 都市异能
    - 玄幻
    - 仙侠
```

用户：

```text
来个都市异能的
```

提取：

```json
{
  "genre": "都市异能"
}
```

### 14.6 不要危险猜测

用户：

```text
来个爽文
```

不要直接猜成：

```json
{
  "genre": "都市异能"
}
```

应该返回：

```json
{
  "missing_fields": ["genre"],
  "questions": [
    {
      "field": "genre",
      "question": "你说的爽文更偏哪种类型？都市异能、玄幻、系统流、还是悬疑？"
    }
  ]
}
```

---

## 15. Validator 设计

### 15.1 required

```go
if field.Required && IsEmpty(value) {
    missing = append(missing, field.Key)
}
```

### 15.2 enum

```go
if field.Type == "enum" && !Contains(field.Options, value) {
    violations = append(violations, FieldViolation{
        Field: field.Key,
        Code: "invalid_enum",
    })
}
```

### 15.3 min_len / max_len

中文按 rune 数计算。

```go
length := utf8.RuneCountInString(value)
```

### 15.4 pattern

适合：

```text
端口
URL
模型 ID
邮箱
K8s namespace
```

### 15.5 custom validator

预留接口：

```go
type FieldValidator interface {
    Validate(ctx context.Context, field FieldSpec, value any) *FieldViolation
}
```

第一版可以先不做复杂自定义。

---

## 16. Question Builder 设计

输出应该既适合聊天，也适合前端表单。

### 16.1 聊天文本

```text
还缺 2 项信息：

1. 主角设定：主角是谁？请给出身份、能力或困境。
2. 第一幕冲突：第一章里主角马上遇到的直接冲突是什么？

你可以直接按下面格式回复：

主角设定：
第一幕冲突：
```

### 16.2 结构化 form

```json
{
  "form": [
    {
      "field": "protagonist",
      "label": "主角设定",
      "type": "textarea",
      "required": true,
      "placeholder": "例如：外卖员，能看到死者最后三分钟执念"
    },
    {
      "field": "first_conflict",
      "label": "第一幕冲突",
      "type": "textarea",
      "required": true,
      "placeholder": "例如：送餐到凶宅，被真正凶手发现"
    }
  ]
}
```

前端能展示表单，用户填完后提交 `answers`，这是最精准的方式。

---

## 17. ConversationRuntime 设计

核心结构：

```go
type ConversationRuntime struct {
    ThreadRepo   ThreadRepository
    MessageRepo  MessageRepository
    SkillRepo    SkillRepository
    ProjectCtx   ProjectContextProvider
    Gate         SkillInputGate
    SkillRuntime SkillRuntime
    Cache        ThreadCache
    Locker       ThreadLocker
}
```

核心流程：

```go
func (r *ConversationRuntime) HandleMessage(ctx context.Context, req ChatMessageRequest) (*ChatResponse, error) {
    unlock, err := r.Locker.Lock(ctx, req.ThreadID)
    if err != nil {
        return nil, err
    }
    defer unlock()

    thread, err := r.ThreadRepo.Get(ctx, req.ThreadID)
    if err != nil {
        return nil, err
    }

    msg, err := r.MessageRepo.Insert(ctx, AgentMessage{
        ThreadID: req.ThreadID,
        Role:     "user",
        Content:  req.Message,
        Metadata: map[string]any{
            "answers": req.Answers,
        },
    })
    if err != nil {
        return nil, err
    }

    _ = msg

    skillID := thread.ActiveSkillID
    if skillID == "" {
        skillID = r.SelectSkill(ctx, req.Message)
        thread.ActiveSkillID = skillID
    }

    contract, err := r.SkillRepo.LoadInputContract(ctx, skillID)
    if err != nil {
        return nil, err
    }

    projectContext, err := r.ProjectCtx.Load(ctx, thread.ProjectID)
    if err != nil {
        return nil, err
    }

    gateOut, err := r.Gate.Process(ctx, GateInput{
        ThreadID:          thread.ID,
        ProjectID:         thread.ProjectID,
        TargetSkillID:     skillID,
        LatestUserMessage: req.Message,
        StructuredAnswers: req.Answers,
        SlotState:         thread.SlotState,
        PendingQuestions:  thread.PendingQuestions,
        SkillContract:     contract,
        ProjectContext:    projectContext,
    })
    if err != nil {
        return nil, err
    }

    thread.SlotState = gateOut.UpdatedSlotState
    thread.PendingQuestions = gateOut.Questions

    if gateOut.Ready {
        thread.Status = "ready_to_execute"
    } else {
        thread.Status = "collecting_fields"
    }

    if err := r.ThreadRepo.UpdateState(ctx, thread); err != nil {
        return nil, err
    }

    if !gateOut.Ready {
        text := RenderClarificationText(gateOut)

        _, _ = r.MessageRepo.Insert(ctx, AgentMessage{
            ThreadID: thread.ID,
            Role:     "assistant",
            Content:  text,
            Metadata: map[string]any{
                "type":           "clarification_required",
                "missing_fields": gateOut.MissingFields,
                "known_fields":   gateOut.KnownFields,
                "questions":      gateOut.Questions,
            },
        })

        return &ChatResponse{
            Type:          "clarification_required",
            ThreadID:      thread.ID,
            TargetSkillID: skillID,
            KnownFields:   gateOut.KnownFields,
            MissingFields: gateOut.MissingFields,
            Questions:     gateOut.Questions,
        }, nil
    }

    result, err := r.SkillRuntime.ExecuteSkill(ctx, SkillExecuteRequest{
        ThreadID:       thread.ID,
        ProjectID:      thread.ProjectID,
        SkillID:        skillID,
        InvocationArgs: gateOut.InvocationArgs,
    })
    if err != nil {
        return nil, err
    }

    _, _ = r.MessageRepo.Insert(ctx, AgentMessage{
        ThreadID: thread.ID,
        Role:     "assistant",
        Content:  result.FinalText,
        Metadata: map[string]any{
            "type":     "skill_result",
            "skill_id": skillID,
            "run_id":   result.RunID,
        },
    })

    _ = r.ThreadRepo.UpdateStatus(ctx, thread.ID, "completed")

    return &ChatResponse{
        Type:          "skill_result",
        ThreadID:      thread.ID,
        TargetSkillID: skillID,
        RunID:         result.RunID,
        Content:       result.FinalText,
    }, nil
}
```

---

## 18. 状态机设计

```text
idle
  ↓
skill_selected
  ↓
collecting_fields
  ├─ 用户补充后仍缺字段 → collecting_fields
  ├─ 用户补充后字段非法 → collecting_fields
  └─ 字段齐全 → ready_to_execute
                      ↓
                executing_skill
                      ↓
                  completed
```

异常状态：

```text
failed
cancelled
expired
```

状态说明：

| 状态 | 说明 |
|---|---|
| idle | thread 刚创建，还没有目标 skill |
| skill_selected | 已确定目标 skill |
| collecting_fields | 正在收集字段 |
| ready_to_execute | 字段齐全，可以执行 |
| executing_skill | skill 执行中 |
| completed | 执行完成 |
| failed | 执行失败 |
| cancelled | 用户取消 |
| expired | 会话过期 |

---

## 19. SkillExecutor 接收格式

SkillExecutor 不接收原始多轮聊天历史，而接收清洗后的执行包。

```json
{
  "thread_id": "t_001",
  "project_id": "p_001",
  "skill_id": "opening_v1",
  "task": "生成黄金600字强钩子开篇",
  "invocation_args": {
    "genre": "都市异能",
    "protagonist": "外卖员，能看到死者最后三分钟执念",
    "hook": "接到三天前死人的订单",
    "first_conflict": "送餐到凶宅时被真正凶手发现，必须在十分钟内逃出去",
    "style": "无AI味、短句推进、强冲突、少解释、多行动"
  },
  "constraints": {
    "do_not_invent_core_facts": true,
    "use_only_invocation_args_as_hard_facts": true,
    "project_context_has_higher_priority": true
  }
}
```

Skill prompt 中应该明确：

```text
你收到的是已经通过 SkillInputGate 校验的结构化参数。
invocation_args 中的字段是本次任务硬约束。
不得擅自改写核心事实。
如果需要补充细节，只能补充非核心描写，不得改变主角、钩子、冲突、世界规则。
```

---

## 20. P0 开发任务清单

### P0-1：新增数据库表

完成：

```text
agent_threads
agent_messages
agent_runs.thread_id
```

验收：

```text
可以创建 thread。
可以保存 message。
可以读取 thread 的 slot_state。
```

### P0-2：实现 SkillInputContract 解析

完成：

```text
从 SKILL.md frontmatter 读取 input_contract。
解析 fields / aliases / validators / defaults。
```

验收：

```text
opening_v1 的 contract 可以被 Go 正确加载。
```

### P0-3：实现 SlotState

完成：

```text
SlotState 结构体
MergeStructuredAnswers
ApplyDefaults
KnownFields
MissingFields
```

验收：

```text
能把 answers 合并进 slot_state。
能识别缺失字段。
```

### P0-4：实现 Rule Extractor

完成：

```text
alias 冒号匹配
enum options 匹配
pending single field 填充
pending 多字段弱解析
```

验收：

```text
“主角是外卖员”能填 protagonist。
“钩子：死人订单”能填 hook。
“来个都市异能”能填 genre。
```

### P0-5：实现 Validator

完成：

```text
required
enum
min_len
max_len
```

验收：

```text
缺 required 返回 missing。
enum 不合法返回 invalid_enum。
文本太短返回 too_short。
```

### P0-6：实现 QuestionBuilder

完成：

```text
根据 missing fields 生成问题。
一轮最多 max_questions_per_turn。
返回 text + form。
```

验收：

```text
缺 protagonist/hook/conflict 时返回三个问题。
```

### P0-7：实现 ConversationRuntime

完成：

```text
创建 thread。
发送 message。
读取 thread_state。
调用 Gate。
保存更新后的 slot_state。
ready 后调用 SkillExecutor。
```

验收：

```text
用户多轮补字段，不需要传历史。
字段齐全后自动执行 skill。
```

### P0-8：HTTP API

完成：

```text
POST /v1/chat/threads
POST /v1/chat/threads/{thread_id}/messages
GET /v1/chat/threads/{thread_id}
```

验收：

```text
curl 可以完成完整多轮。
```

---

## 21. P1 增强任务

```text
1. Redis active thread cache。
2. Redis thread lock。
3. 前端结构化 form 渲染。
4. project_context 字段预填。
5. 字段冲突检测。
6. thread 过期策略。
7. 用户取消 / 重置当前 skill。
8. 支持切换 active_skill_id。
9. 支持 nested object 字段。
10. 支持 array 字段。
```

---

## 22. P2 高级能力

```text
1. 多 skill workflow。
2. 一个 thread 里串联多个 form。
3. skill 执行完成后把结果写入 project_documents。
4. 对 slot_state 做版本化。
5. 用户确认机制。
6. 非确定性自然语言理解可选接入模型，但默认关闭。
7. 根据项目状态自动选择 skill。
8. 支持人工审批。
```

---

## 23. 示例 curl 流程

### 23.1 创建 thread

```bash
curl -X POST http://localhost:8080/v1/chat/threads \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "p_001",
    "target_skill_id": "opening_v1",
    "initial_message": "帮我写一个都市异能开头"
  }'
```

期望返回：

```json
{
  "type": "clarification_required",
  "thread_id": "t_001",
  "missing_fields": [
    "protagonist",
    "hook",
    "first_conflict"
  ]
}
```

### 23.2 第二轮补充

```bash
curl -X POST http://localhost:8080/v1/chat/threads/t_001/messages \
  -H "Content-Type: application/json" \
  -d '{
    "message": "主角是外卖员，能看到死者最后三分钟执念。钩子是接到三天前死人的订单。"
  }'
```

期望返回：

```json
{
  "type": "clarification_required",
  "missing_fields": [
    "first_conflict"
  ]
}
```

### 23.3 第三轮补充

```bash
curl -X POST http://localhost:8080/v1/chat/threads/t_001/messages \
  -H "Content-Type: application/json" \
  -d '{
    "message": "冲突是他送餐到凶宅时被真正凶手发现，必须在十分钟内逃出去。"
  }'
```

期望返回：

```json
{
  "type": "skill_result",
  "run_id": "r_001",
  "content": "..."
}
```

---

## 24. 验收标准

### 24.1 功能验收

必须满足：

```text
1. 用户首轮信息不足时，不调用 skill。
2. 返回明确 known_fields / missing_fields / questions。
3. 用户第二轮只传 thread_id + message。
4. 系统能恢复上一轮 slot_state。
5. 系统能继续填充字段。
6. 字段齐全后自动调用目标 skill。
7. skill 接收到的是 invocation_args，不是混乱历史。
8. Redis 丢失不影响恢复。
9. PostgreSQL 中能回放完整对话。
10. 每次 run 能关联 thread_id。
```

### 24.2 准确性验收

测试输入：

```text
帮我写一个都市异能开头。
```

系统必须识别：

```json
{
  "genre": "都市异能"
}
```

系统不得编造：

```json
{
  "protagonist": "..."
}
```

测试输入：

```text
主角是外卖员。
```

如果 pending field 是 `protagonist`，必须填入 protagonist。

测试输入：

```text
来个爽文。
```

系统不得强行填 genre，必须追问具体类型。

### 24.3 工程验收

```text
1. Gate 单元测试覆盖率高于核心逻辑 80%。
2. ConversationRuntime 有集成测试。
3. 所有状态变更写 PostgreSQL。
4. Redis 只是缓存，删除 Redis 后流程可继续。
5. 同一 thread 并发请求有锁保护。
```

---

## 25. AI 编程助手执行提示词

可以把下面这段直接交给 AI 写代码：

```text
你要在 labs-agent_runtime/backend 中实现一套 deterministic SkillInputGate 多轮字段收集机制。

要求：
1. 不依赖模型能力做字段收集。
2. skill 不保存多轮状态。
3. ConversationRuntime 使用 thread_id 从 PostgreSQL 恢复 thread_state。
4. SkillInputGate 根据 skill 的 input_contract 做 slot filling、validation、question building。
5. 字段不全时返回 clarification_required。
6. 字段齐全时生成 invocation_args 并调用现有 SkillExecutor。
7. PostgreSQL 是主存储，Redis 只作为可选缓存。
8. 用户第二轮只需要传 thread_id + message 或 answers，不需要传历史。
9. 需要新增 /v1/chat/threads、/v1/chat/threads/{id}/messages、/v1/chat/threads/{id}。
10. 需要新增 agent_threads、agent_messages，并给 agent_runs 增加 thread_id。
11. 需要实现 gate/contract.go、gate/slot_state.go、gate/extractor.go、gate/validator.go、gate/question_builder.go、gate/runner.go。
12. 需要为 opening_v1 示例 skill 增加 input_contract YAML。
13. 需要提供 curl 测试用例和单元测试。

核心流程：
- 创建 thread。
- 保存用户消息。
- 加载 active_skill_id。
- 加载 skill input_contract。
- 调用 SkillInputGate。
- 保存 updated_slot_state。
- 如果 not ready，保存 assistant clarification message 并返回 questions。
- 如果 ready，调用 SkillExecutor.ExecuteSkill，并把 invocation_args 传入。
```

---

## 26. 第一版最小实现边界

为了避免一开始做复杂，第一版只实现：

```text
1. 一个 thread 同一时间只处理一个 active_skill。
2. input_contract 只支持 text / enum 两种字段类型。
3. extractor 只支持：
   - structured answers
   - alias 冒号匹配
   - enum option 包含匹配
   - 单 pending field 直接填充
4. validator 只支持：
   - required
   - enum
   - min_len
   - max_len
5. 不做模型理解。
6. 不做复杂 workflow。
7. 不做向量检索。
8. 不做自动推断。
```

第一版跑通后，再扩展 nested object、array、workflow、project_context 冲突检测。

---

## 27. 最终收益

这套方案可以带来：

```text
1. 准确：字段不齐不执行，避免模型乱补。
2. 稳定：多轮状态在 PostgreSQL，可恢复。
3. 清晰：每个 Skill 只关注自己的业务能力。
4. 可测：Gate 是规则系统，容易写单元测试。
5. 可扩展：后续可以接前端表单、Workflow、人工审批。
6. 可维护：所有缺失字段和已知字段都有结构化记录。
7. 可复用：所有 Skill 共享同一套多轮需求收集机制。
```

---

## 28. 推荐开发顺序

```text
第 1 步：新增数据库表。
第 2 步：实现 input_contract 解析。
第 3 步：实现 SlotState。
第 4 步：实现 Gate validator。
第 5 步：实现 Gate extractor。
第 6 步：实现 QuestionBuilder。
第 7 步：实现 ConversationRuntime。
第 8 步：实现 HTTP API。
第 9 步：接入现有 SkillExecutor。
第 10 步：为 opening_v1 写端到端测试。
第 11 步：补 Redis lock/cache。
第 12 步：接前端结构化 form。
```

---

## 29. 端到端例子

### 用户第一轮

```text
帮我写一个都市异能开头
```

Gate 输出：

```json
{
  "ready": false,
  "known_fields": {
    "genre": "都市异能"
  },
  "missing_fields": [
    "protagonist",
    "hook",
    "first_conflict"
  ]
}
```

### 系统追问

```text
还缺 3 项信息：

1. 主角设定：主角是谁？请给出身份、能力或困境。
2. 开篇钩子：开篇第一钩子是什么？
3. 第一幕冲突：第一章里主角马上遇到的直接冲突是什么？

你可以按下面格式回复：

主角设定：
开篇钩子：
第一幕冲突：
```

### 用户第二轮

```text
主角是外卖员，能看到死者最后三分钟执念。钩子是接到三天前死人的订单。
```

Gate 输出：

```json
{
  "ready": false,
  "known_fields": {
    "genre": "都市异能",
    "protagonist": "外卖员，能看到死者最后三分钟执念",
    "hook": "接到三天前死人的订单"
  },
  "missing_fields": [
    "first_conflict"
  ]
}
```

### 用户第三轮

```text
冲突是他送餐到凶宅时被真正凶手发现，必须在十分钟内逃出去。
```

Gate 输出：

```json
{
  "ready": true,
  "invocation_args": {
    "genre": "都市异能",
    "protagonist": "外卖员，能看到死者最后三分钟执念",
    "hook": "接到三天前死人的订单",
    "first_conflict": "他送餐到凶宅时被真正凶手发现，必须在十分钟内逃出去",
    "style": "无AI味、短句推进、强冲突、少解释、多行动"
  }
}
```

Runtime 调用：

```text
SkillExecutor.ExecuteSkill(opening_v1, invocation_args)
```

---

## 30. 设计定论

本方案不是给每个 Skill 增加“多轮对话能力”，而是给 Runtime 增加一套统一的：

```text
Form Runner / Slot Filling / Requirement Gate
```

这样所有 Skill 都能复用同一套：

```text
缺字段判断
追问生成
字段填充
状态恢复
参数组装
```

最终架构：

```text
ConversationRuntime 管状态；
SkillInputGate 管字段；
ContextProvider 管项目上下文；
SkillExecutor 管能力执行；
PostgreSQL 管持久化；
Redis 管缓存和锁。
```

这就是精准、工程化、可维护的多轮 Skill 调用方案。
