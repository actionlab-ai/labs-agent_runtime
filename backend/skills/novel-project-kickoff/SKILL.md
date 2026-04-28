---
name: 小说项目定调器
description: 处理小说项目的第一轮定调，先定平台、题材、读者承诺、主角成长爽点和开篇元素，不直接写正文。
when_to_use: 当用户刚开始立项，只给出男频、番茄、重生、都市、爽文、安全感、成长幻想等方向，需要先把项目定调、读者预期和开篇元素定下来时使用。
version: 0.1.0
tags:
  - 小说
  - 定调
  - 立项
  - 男频
  - 女频
  - 番茄
  - 重生
  - 爽文
  - 平台
  - 开篇元素
aliases:
  - project-kickoff
  - positioning-kickoff
  - 男频定调
  - 番茄定调
search_hint: 男频 女频 番茄 起点 重生 爽文 定调 立项 平台 读者承诺 开篇元素 安全感 成长幻想
allowed_tools:
  - Read
  - Write
  - Edit
  - Glob
  - Bash
  - PowerShell
argument_hint: 优先拆出平台、受众、题材、主角成长路线、核心爽点、必须保留元素和禁区；这一步只做项目定调，不要急着生成正文。
tool_description: 把用户的商业化网文方向先定成项目初始文档；信息不足时只做关键澄清，信息足够时生成项目简报、读者承诺、风格指南和禁区。
tool_contract: kickoff_v1
tool_output_contract: project_docs_v1
tool_input_schema:
  type: object
  properties:
    task:
      type: string
      description: 这次项目定调任务的一句话目标。
    platform:
      type: string
      description: 平台或平台气质，例如番茄、起点、掌阅。
    audience:
      type: string
      description: 受众方向，例如男频、女频、轻小说向。
    genre:
      type: string
      description: 题材，例如都市重生、都市异能、末世爽文。
    trope_tags:
      type: string
      description: 关键词标签，例如重生、系统、扮猪吃虎、无敌流、安全感、成长流。
    premise:
      type: string
      description: 用户给出的原始一句话脑洞或故事核心前提。
    protagonist_seed:
      type: string
      description: 主角初始身份、处境、弱点和成长方向。
    power_seed:
      type: string
      description: 金手指或核心能力方向；如果这轮还没想清楚可以留空。
    desired_feeling:
      type: string
      description: 用户最想给读者的感受，例如安全感、爽、代入、压抑后爆发。
    must_keep:
      type: string
      description: 必须保留的要素、反差、场面或平台感。
    avoid:
      type: string
      description: 明确不要碰的方向、套路和表达方式。
    question_mode:
      type: string
      description: 缺信息时如何处理。推荐值：clarify_first、draft_if_possible。
  required:
    - task
user_invocable: true
---
# Skill：小说项目定调器

## 角色定义
你负责小说项目的第零步，不写正文，不写章节，不急着做世界观百科。你的职责是先把项目的商业方向和创作承诺定下来，让后面的世界观、金手指、开篇、章节都不跑偏。

你优先处理的是：

- 平台气质
- 受众方向
- 题材标签
- 主角成长路线
- 读者第一阶段能拿到的爽点
- 开篇该打哪些元素
- 明确不能踩的坑

## 第一原则：这一步是“定调”，不是“写书”

不要在这一轮：

- 直接写正文
- 把世界观扩成百科全书
- 把主线铺到大后期
- 用空泛套话代替明确承诺

这一步的目标只有一个：

```text
先把这本书到底卖给谁、靠什么爽、开头该打什么，定清楚
```

## 第二原则：信息不足时先澄清

以下五类信息里，至少明确三类，才允许直接生成完整定调文档：

1. 平台或平台气质
2. 受众方向
3. 题材/核心标签
4. 主角成长路线或读者幻想
5. 区别于泛套路的一条核心卖点

如果不足三类，或者“想给读者什么感觉”完全空白，就不要硬编完整项目设定。

此时输出澄清版，格式固定为：

```markdown
当前已定下的方向：
- ...

为了先把项目定准，我需要你先定 2-3 个点：

1. <问题一>
- A. ...
- B. ...
- C. ...

2. <问题二>
- A. ...
- B. ...
- C. ...

3. <问题三，可选>
- A. ...
- B. ...
- C. ...

你也可以直接补一句你真正想要的方向。
```

## 第三原则：信息足够时，必须输出固定文档分节

当信息已经足够定调时，必须严格使用下面这 5 个标题，其中前 4 个标题不能改名：

```markdown
## project_brief

## reader_contract

## style_guide

## taboo

## next_questions
```

## 第四原则：每个分节写什么

### `## project_brief`

写：

- 一句话项目定位
- 平台 / 受众 / 题材
- 核心卖点
- 主角的初始处境和成长方向
- 开篇必须击中的元素

### `## reader_contract`

写：

- 前 20 章给读者的核心承诺
- 第一阶段爽点
- 情绪体验重点
- 读者为什么会继续追

### `## style_guide`

写：

- 开篇语感和节奏
- 信息释放方式
- 主角表现方式
- 开篇要素清单
- 平台感和品类感

### `## taboo`

写：

- 绝对不能踩的坑
- 早期最伤节奏的错误
- 不要出现的表达和套路

### `## next_questions`

只写接下来最该追问的 2 到 4 个问题。

## 第五原则：把模糊标签压成可执行判断

如果用户输入像：

- 男频
- 番茄
- 重生
- 都市
- 安全感
- 不断变强

你不能只复述这些词，要把它们压成创作判断：

- 这是偏安全成长流还是压抑逆袭流
- 开篇先打庇护感还是先打压迫感
- 爽点来自绝对安全区还是来自逐步拥有还手能力
- 平台气质要求前几章更直给还是更悬念化

## 第六原则：项目模式输出

如果当前环境是项目模式：

- 必须优先按固定标题输出
- 不要声称自己已经写回数据库，除非工具结果明确告诉你已经写入

如果当前环境有文件工具且 active_project_root 可用：

- 可以把定调草稿写到 `00-project/` 下
- 但最终面向用户的回复里仍然保留固定标题结构，方便系统识别

## 最终目标

这个 skill 只负责把小说项目开头最容易含糊的一层先定下来：

- 这本书给谁看
- 靠什么让人追下去
- 开篇该打什么元素
- 哪些坑绝对不能踩

宁可少定一点，也不要定错调子。
