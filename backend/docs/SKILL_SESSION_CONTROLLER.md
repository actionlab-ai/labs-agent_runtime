# Skill Session Controller

本文档解释了共享的交互式技能执行流程。

## Goal

某些技能无法仅凭用户的首次提示安全地完成执行。它们需要向人类询问缺失的事实、偏好或决策，然后在相同的对话上下文中继续执行。

这不应在每个业务工作流中单独实现。共享边界如下：


- 运行时（Runtime）向每个技能执行暴露一个内置的 AskHuman 工具。
- skillsession.Manager 负责管理会话状态、暂停、恢复和转录记录。
- HTTP 层暴露一个通用的技能会话控制器，供 Web UI 使用。

## Flow

```text
POST /v1/skill-sessions
  -> 从 PostgreSQL/缓存中解析项目
  -> 从 PostgreSQL/缓存或默认模型设置中解析模型
  -> 构建带有项目文档提供者的运行时环境
  -> 运行指定的 skill_id
  -> 模型可以调用 AskHuman
  -> 响应状态为 completed（已完成）或 needs_input（需要输入）

POST /v1/skill-sessions/{id}/turns
  -> 接收人类回答
  -> 将回答注入为待处理的 AskHuman 工具结果
  -> 恢复相同的技能对话
  -> 响应状态再次变为 completed（已完成）或 needs_input（需要输入）
```

## API

Start:

```http
POST /v1/skill-sessions
Content-Type: application/json

{
  "project": "urban-rebirth",
  "model": "deepseek-flash",
  "skill_id": "novel-emotional-core",
  "input": "首先构建情感核心。如果缺少必要事实，请向我提问。",
  "arguments": {
    "document_kind": "novel_core"
  }
}
```

Continue:

```http
POST /v1/skill-sessions/{id}/turns
Content-Type: application/json

{
  "input": "选择尊严和认可作为核心回报。",
  "answers": {
    "payoff": "dignity and recognition"
  },
  "notes": "避免简单的突然致富情节。"
}
```

## 边界

`AskHuman` 仅用于请求缺失的人类输入。它不写入文件、项目文档、PostgreSQL 行、Redis 键或提供者对象。

持久化的项目工件仍应通过由提供者支持的运行时工具（如 WriteProjectDocument）进行处理。这样可以确保文件系统、S3、PostgreSQL 和 Redis 的同步保持在一个提供者边界之内。

首个实现版本将技能会话存储在内存中。这对于单个运行的 HTTP 进程和本地开发已经足够。如果需要多副本支持或重启后可恢复的功能，则需将 SkillExecutionState 持久化存储到 PostgreSQL 或 Redis 中。

## Claude Code Reference

This follows the useful part of Claude Code's `AskUserQuestion` pattern:

- the model emits a user-interaction tool call;
- the runtime pauses instead of guessing;
- the UI collects human input;
- the runtime resumes by injecting the answer as a tool result.

The implementation does not copy Claude Code's global session state. This project
keeps that responsibility in `internal/skillsession`.
