# Guide: Prompt Files

The prompt file is a markdown file piped to Claude Code on each loop iteration. Rather than a single monolithic prompt, the structure splits into a router prompt and phase-specific prompts. This keeps each agent's context small and focused.

## Router prompt

The main `prompt.md` orients the agent and routes it to the correct phase prompt. It is the only file piped to Claude Code — phase prompts are read by the agent during execution.

```markdown
# <Project Name>

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
```

## Phase prompts

Each phase prompt lives in `.lorah/prompts/` and defines the workflow for a single phase. The agent reads exactly one per iteration.

**`prompts/plan.md`** — Select the next task from the design specs and current state, then create a task file.

```markdown
# Planning Phase

## Workflow

1. Read `.lorah/plan.md` for scope and acceptance criteria.
2. Read the design specs in `docs/design/` for behavioral details.
3. Review git history and completed tasks in `.lorah/tasks/` to
   understand what has been built.
4. Check for a blocked task in `.lorah/tasks/`. If one exists, read
   its Log and revise the task to address the issue. Set status to
   `test` or `implement` (same criteria as step 7), add notes to the
   Log, and skip to step 8.
5. Check the plan file's acceptance criteria against current git
   state and test results. If all criteria are met, exit — the work
   is complete.
6. Identify the single next task — the smallest unit of work that
   moves toward acceptance criteria.
7. Create a new task file in `.lorah/tasks/` using the task file
   format. Set the task status based on whether it has testable
   behavior: `test` if it implements logic or behavior that benefits
   from test-first development; `implement` if it is pure
   configuration or scaffolding with no behavioral logic to test. Add
   planning notes to the Log.
8. Commit.
```

**`prompts/test.md`** — Write tests for the current task.

```markdown
# Testing Phase

## Workflow

1. Read the current task file in `.lorah/tasks/`.
2. Read the relevant design spec section(s) referenced in the task.
3. Write tests that verify the behavior described in the task's
   acceptance criteria. Do not write any production code. Add stubs
   or interface definitions only if required to make tests
   compilable.
4. Verify: run the test suite. Failures are expected (no
   implementation yet), but panics and compilation errors must be
   fixed.
5. Update the Testing section of the task file's Log with files
   created and edge cases covered.
6. Update the task status from `test` to `implement`.
7. Commit.

## Blocked workflow

If the design spec is ambiguous or contradicts the task file, add a
note to the task file explaining the issue, set status to `blocked`,
and exit without committing test code.
```

**`prompts/implement.md`** — Write production code to satisfy the task.

```markdown
# Implementation Phase

## Workflow

1. Read the current task file in `.lorah/tasks/`.
2. Read the relevant design spec section(s) referenced in the task.
3. If tests were written in the testing phase, read them.
4. Write production code to satisfy the acceptance criteria (and
   make tests pass, if they exist). Do not write new tests.
5. Verify: if tests exist, run the full test suite — all tests must
   pass. Otherwise, verify acceptance criteria directly (e.g., run
   commands, check file contents).
6. Update the task file: set status to `completed`, add
   implementation notes to the Log.
7. Commit.

## Blocked workflow

If the existing tests conflict with the design spec:

1. Discard uncommitted changes.
2. Set the task status to `blocked` with notes explaining the
   conflict.
3. Exit without committing.

The next iteration will route to the planning phase to reassess the
task.
```

The prompts above are starting points. Adapt the rules and workflow steps to match your project. Focus on boundaries and invariants rather than detailed instructions for every scenario.
