# Planning API — Local Dev Scaffold

Complete exactly one task per invocation.

---

## Workflow

1. **Orient** — Run `git log --oneline -10` to understand what was
   done in prior iterations.

2. **Route** — Scan `.lorah/tasks/` for task files. At most one
   task is non-completed at any time.
   - If a task has `status: blocked`, read its Log to understand the
     issue, then read and follow `.lorah/prompts/plan.md`.
   - Else if a task has `status: test`, read and follow
     `.lorah/prompts/test.md`.
   - Else if a task has `status: implement`, read and follow
     `.lorah/prompts/implement.md`.
   - Else, read and follow `.lorah/prompts/plan.md`.

3. **Exit** — Stop. Do not proceed to the next task.

---

## Rules

- One task per invocation: complete one task, commit, exit.
- Design specs are authoritative: `docs/design/` defines the target
  behavior.
- Task files use incrementing numeric prefix with kebab-case name
  (e.g., `01-docker-compose.md`, `02-knexfile.md`).
- Each invocation must start in a clean git state. If uncommitted
  changes exist, discard them before proceeding.
