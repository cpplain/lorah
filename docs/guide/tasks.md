# Guide: Task Files

Each task gets its own file in `.lorah/tasks/`. The planning agent creates one task file per iteration; the testing and implementation agents update it as they work. This keeps context small — an agent only reads the task it is working on.

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
