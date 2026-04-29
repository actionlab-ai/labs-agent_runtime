# Novel Agent Runtime Spec

This directory is the implementation contract for AI-assisted development.

The working pattern is:

1. Describe the product and architecture rule in `spec/`.
2. Ask AI to implement only against the relevant spec files.
3. Ask AI to produce verification scripts or tests for the listed invariants.
4. Review the design principle and test result first, then review code only where risk is high.

This is a spec-driven, invariant-driven workflow. The goal is not to document code after the fact;
the goal is to make design intent explicit enough that code generation cannot easily drift.

## Directory Layout

- `global/`: rules shared by every module.
- `modules/`: executable module specs. Each file defines data ownership, flow order, API behavior, failure policy, and verification points.
- `composed/`: application-level composition of modules.
- `resolved/`: flattened current-state architecture for quick AI context.
- `generation/`: rules for AI code generation and verification.
- `rules.yaml`: the top-level execution contract.

## How To Use With AI

Give AI a task in this shape:

```text
Read spec/rules.yaml, spec/global/*.yaml, and spec/modules/<module>.yaml.
Implement only the requested change.
Do not violate listed invariants.
After implementation, add or update verification that proves each changed flow still follows the spec.
```

For example:

```text
Read spec/modules/project.yaml and update project delete behavior.
The flow must be db_delete first, then async cache delete, and filesystem metadata sync must not become source of truth.
Add a verification script or test that proves PG deletion succeeds even if Redis sync fails.
```

## Current Implemented Capabilities

- HTTP-only service entrypoint through Gin.
- PostgreSQL-backed project, model, default model, run, and project document management.
- Redis cache for project, model profile, and default model setting.
- Filesystem project provider that mirrors project metadata and documents after PostgreSQL writes.
- Runtime router with `tool_search`, retained skill tools, and provider-backed project document tools.
- Fixed workflows for project kickoff and emotional core creation.
- Interactive skill sessions with built-in `AskHuman` pause/resume support.
- Zap structured logging across HTTP, database, cache, project filesystem sync, runs, workflows, and skill sessions.
