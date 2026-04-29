# Spec-Driven AI Workflow

你描述的工作方式可以叫：

- Spec-Driven Development
- Invariant-Driven Development
- Architecture Decision Records
- Contract-First Implementation

在这个项目里，更准确的说法是：**用 spec 固化设计原理，用 invariants 约束 AI 生成，用 verification 脚本验证执行结果**。

## 为什么这样适合 AI 写代码

AI 很擅长根据上下文补全代码，但如果上下文只是一句需求，它会倾向于“能跑就行”的局部实现。项目复杂后，最容易变形的不是语法，而是架构边界：

- 谁是数据源真相？
- 写入顺序是什么？
- 缓存失败是否影响主流程？
- 文件系统是主存储还是投影？
- runtime 能不能维护业务 session？
- skill 应该直接写文件，还是调用 provider？

这些问题如果只靠临时聊天解释，很容易在下一轮改动里丢失。spec 的作用就是把这些决定固化成可重复读取的工程输入。

## 正确使用方式

每次让 AI 改代码前，先指定相关 spec：

```text
先读 spec/rules.yaml、spec/global/*.yaml、spec/modules/project.yaml。
这次只改 project 删除流程。
必须满足：
1. PostgreSQL 先删除；
2. Redis 删除只能异步发生；
3. Redis 失败不能影响 HTTP 删除结果；
4. 文件系统只是 provider projection，不能当源数据。
改完补验证脚本或测试。
```

这样 AI 的任务就不是“帮我实现删除”，而是“按这个状态机实现删除，并证明状态机没变形”。

## 推荐任务模板

```text
目标：
<要实现的功能>

必须读取：
- spec/rules.yaml
- spec/global/data_flow.yaml
- spec/global/invariants.yaml
- spec/modules/<module>.yaml

实现约束：
- 不要改变 spec 没允许改变的边界
- 不要引入新的存储路径
- 不要绕过 provider/service/repository 分层
- 失败策略按 spec 执行

验收：
- 说明每个 flow step 对应的代码位置
- 说明每个 invariant 如何验证
- 增加或更新测试/脚本
- 给出测试命令和结果
```

## 评审重点

你不需要每次读完所有代码。优先看这四个东西：

1. Flow order 是否符合 spec。
2. Source of truth 是否仍然是 PostgreSQL。
3. Boundary 是否没被绕开，比如 skill 是否还通过 provider 写文档。
4. Verification 是否真的覆盖了 spec 里的 invariant。

代码细节只在高风险点展开看，比如 migration、SQL、缓存同步、provider 写入、runtime tool execution、session resume。

## 当前项目的核心原则

- PostgreSQL 是业务数据源真相。
- Redis 是性能缓存，不是源数据。
- Filesystem provider 是项目文档投影，未来可以替换成 S3。
- Runtime 负责模型调用、工具暴露、工具执行和 debug artifacts。
- SkillSession 负责 AskHuman 的通用暂停/恢复。
- Workflow 负责产品级固定流程，不负责通用 session 状态机。
- Model profile 和 default model 都在数据库，不在 config.yaml。
- config.yaml 只保留基础设施连接、runtime 目录、日志等系统配置。
