# Guide: Task Files

Each task gets its own file in `.lorah/tasks/`. The planning agent creates one task file per iteration; the testing and implementation agents update it as they work. This keeps context small — an agent only reads the task it is working on. At most one task is non-completed at any time.

Task files use sequential numbering as a prefix (e.g., `01-parse-cli-args.md`) to preserve ordering.

```markdown
---
status: in_progress
---

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

- `in_progress` — actively being worked by the current or most recent iteration.
- `completed` — done. Tests pass, code is committed.
- `blocked` — cannot proceed. See notes in Log for details.

## Blocked task handling

When the planning agent encounters a blocked task, it revises the existing task file — updating the Behavior, Acceptance Criteria, or Context as needed to address the issue noted in the Log. It sets status back to `in_progress` and adds notes to the Planning log explaining the revision. No new task file is created.
