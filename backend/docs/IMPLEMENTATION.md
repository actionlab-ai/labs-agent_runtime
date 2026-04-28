# 实现说明

这份文档描述当前真实代码结构，不站在旧版本视角。

## 1. 目录职责

### 进程入口

- [main.go](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/cmd/novelrt/main.go)

只做三件事：

1. 解析命令行参数
2. 加载配置
3. 调用 `httpapi.Serve(...)`

### HTTP API 层

- [internal/httpapi](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/httpapi)

这层负责：

- 构建 Gin router
- 注册各资源路由
- 装配 store / cache / filesystem 适配器
- 固定 workflow 调度
- 通用 `/v1/runs` 调度
- service 层编排

当前子文件分工：

- `app.go`: 服务启动与 router 装配
- `api_types.go`: HTTP request DTO 和 `appStore` 接口
- `service_errors.go`: service 层状态错误
- `runtime_factory.go`: 共享 runtime/session 装配
- `run_service.go`: `/v1/runs` service
- `workflow_service.go`: 固定 workflow service
- `routes_*.go`: 各资源路由
- `workflow_*.go`: 固定 workflow 路由、计划、持久化
- `project_document_provider.go`: runtime 项目文档 provider
- `project_store.go`: project store adapter
- `model_store.go`: model store adapter
- `stores.go`: store adapter 基础类型
- `http_helpers.go`: 公共 helper

### 运行时与业务层

- `internal/runtime`
- `internal/workflow`
- `internal/skill`
- `internal/project`
- `internal/store`

这里才是真正的业务能力实现。

## 2. 两条主要执行链

### 通用运行链

```text
/v1/runs
-> 选择模型
-> 读取项目上下文
-> Runtime.Run(...)
-> router 可调用 tool_search
-> 动态激活 skill tool
-> skill executor 执行 skill
```

### 固定 workflow 链

```text
/v1/workflows/project-kickoff
或
/v1/workflows/project-kernel
-> 固定选中 skill
-> SequentialWorkflowRunner
-> Runtime.ExecuteSkill(...)
-> 结果写回项目文档
```

## 3. 当前固定 workflow 映射

### `project-kickoff`

- 固定 skill: `novel-project-kickoff`
- 固定写回:
  - `project_brief`
  - `reader_contract`
  - `style_guide`
  - `taboo`

### `project-kernel`

- 固定 skill: `novel-emotional-core`
- 固定写回:
  - `novel_core`

## 4. skill 到模型请求的真实封装

服务端不会把 `skill_id` 裸发给模型。

实际过程在：

- [ExecuteSkill](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/runtime/runtime.go:343)
- [ComposeSkillPrompt](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/runtime/runtime.go:411)
- [skillDocumentHint](/C:/Users/admin/Desktop/novel-knowledge-assets-v0.1/backend/internal/runtime/file_tools.go:768)

它会组装出一份完整 prompt，包含：

1. Skill Metadata
2. Skill Tooling
3. Base Directory
4. Activated Tool Arguments
5. Skill Instruction
6. Original User Request
7. Current Skill Task
8. Output Rule

然后再把 `tools` 一起挂到模型请求里。

## 5. 为什么这次要把 cmd 层继续下沉

之前的问题不是“文件多”，而是“HTTP 细节、store 适配、workflow 编排、入口装配都堆在 cmd 目录里”。

这会带来三个维护问题：

1. 读入口时看不出真正服务边界
2. 加新接口时容易继续堆大文件
3. 测试和实现都绑在 `package main`

所以现在把 HTTP 层整体下沉到 `internal/httpapi`，目的是：

1. `cmd` 只保留可执行入口
2. `httpapi` 成为可独立测试的服务层
3. 后续继续拆子模块时，有稳定的承载位置

## 6. 当前边界

现在已经完成的是“服务层结构整理”，不是业务协议重写。

也就是说：

- API 路径没变
- workflow 语义没变
- runtime 行为没变

变化的是：

- 目录更清楚
- 依赖更集中
- route 层变薄
- 复杂编排开始下沉到 service 层
