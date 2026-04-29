# Skill Session Controller

This document explains the shared interactive skill execution flow.

## Goal

Some skills cannot finish safely from the first user prompt. They need to ask the
human for missing facts, preferences, or decisions, then continue with the same
conversation context.

This should not be implemented separately inside every business workflow. The shared
boundary is:

- Runtime exposes a built-in `AskHuman` tool to every skill execution.
- `skillsession.Manager` owns session state, pause, resume, and transcript.
- HTTP exposes a generic skill session controller for web UI usage.

## Flow

```text
POST /v1/skill-sessions
  -> resolve project from PostgreSQL/cache
  -> resolve model from PostgreSQL/cache or default model setting
  -> build runtime with project document provider
  -> run explicit skill_id
  -> model can call AskHuman
  -> response status is completed or needs_input

POST /v1/skill-sessions/{id}/turns
  -> receive human answers
  -> inject answers as the pending AskHuman tool result
  -> resume the same skill conversation
  -> response status is completed or needs_input again
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
  "input": "Build the emotional core first. Ask me if required facts are missing.",
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
  "input": "Choose dignity and recognition as the core payoff.",
  "answers": {
    "payoff": "dignity and recognition"
  },
  "notes": "Avoid simple sudden wealth."
}
```

## Boundaries

`AskHuman` only asks for missing human input. It does not write files, project
documents, PostgreSQL rows, Redis keys, or provider objects.

Durable project artifacts should still go through provider-backed runtime tools such
as `WriteProjectDocument`. That keeps filesystem, S3, PostgreSQL, and Redis sync
behind one provider boundary.

The first implementation stores skill sessions in memory. It is enough for one
running HTTP process and local development. If multiple replicas or restart-safe
resume are required, persist `SkillExecutionState` in PostgreSQL or Redis.

## Claude Code Reference

This follows the useful part of Claude Code's `AskUserQuestion` pattern:

- the model emits a user-interaction tool call;
- the runtime pauses instead of guessing;
- the UI collects human input;
- the runtime resumes by injecting the answer as a tool result.

The implementation does not copy Claude Code's global session state. This project
keeps that responsibility in `internal/skillsession`.
