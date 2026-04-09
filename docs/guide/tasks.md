# Guide: Task Files

Each task gets its own file in `.lorah/tasks/`. The planning agent creates one task file per iteration; the testing and implementation agents update it as they work. This keeps context small — an agent only reads the task it is working on. At most one task is non-completed at any time.

Task files use sequential numbering as a prefix (e.g., `01-parse-cli-args.md`) to preserve ordering.

```markdown
---
status: test
---

<!-- Valid statuses: test | implement | blocked | completed -->

# Task: <title>

## Behavior

What this task implements — reference the relevant spec section(s).

## Acceptance Criteria

- Concrete, testable conditions.

## Context

Relevant files, prior task decisions, or anything the next agent
needs.

## Log

### Planning

- ...

### Testing

- ...

### Implementation

- ...
```

## Status values

- `test` — has testable behavior; next iteration writes tests.
- `implement` — no testable behavior (pure config/scaffolding), or tests are written and next iteration writes production code.
- `blocked` — cannot proceed. See notes in Log for details.
- `completed` — done. Tests pass (if any), code is committed.

## Blocked task handling

When the planning agent encounters a blocked task, it revises the existing task file — updating the Behavior, Acceptance Criteria, or Context as needed to address the issue noted in the Log. It sets status to `test` or `implement` based on whether the revised task has testable behavior, and adds notes to the Planning log explaining the revision. No new task file is created.
