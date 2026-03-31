# Guide: Prompt Files

The prompt file is a markdown file piped to Claude Code on each loop iteration. Rather than a single monolithic prompt, the structure splits into a router prompt and phase-specific prompts. This keeps each agent's context small and focused.

## Router prompt

The main `prompt.md` orients the agent and routes it to the correct phase prompt. It is the only file piped to Claude Code — phase prompts are read by the agent during execution.

```markdown
# <Role Title>

You are a <role> for the <project> project. Your job is to complete
exactly one task per invocation.

---

## Workflow

1. **Orient** — Run `git log --oneline -10` to understand what was
   done in prior iterations.

2. **Route** — Scan `.lorah/tasks/` for task files.
   - If no task file has `status: in_progress`, read and follow
     `.lorah/prompts/plan.md`.
   - If a task has `status: in_progress` and no tests exist for it,
     read and follow `.lorah/prompts/test.md`.
   - If a task has `status: in_progress` and tests exist, read and
     follow `.lorah/prompts/implement.md`.

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
4. Identify the single next task — the smallest unit of work that
   moves toward acceptance criteria.
5. Create a new task file in `.lorah/tasks/` using the task file
   format. Set status to `in_progress`. Add planning notes to the
   Log.
6. Commit the new task file.
```

**`prompts/test.md`** — Write tests for the in-progress task.

```markdown
# Testing Phase

## Workflow

1. Read the in-progress task file in `.lorah/tasks/`.
2. Read the relevant design spec section(s) referenced in the task.
3. Write tests that verify the behavior described in the task's
   acceptance criteria. Do not write any production code. Add stubs
   or interface definitions only if required to make tests
   compilable.
4. Verify: run the test suite. Failures are expected (no
   implementation yet), but panics and compilation errors must be
   fixed.
5. Update the task file's Testing log with files created and edge
   cases covered.
6. Commit.

## Blocked workflow

If the design spec is ambiguous or contradicts the task file, add a
note to the task file explaining the issue, set status to `blocked`,
and exit without committing test code.
```

**`prompts/implement.md`** — Make the tests pass.

```markdown
# Implementation Phase

## Workflow

1. Read the in-progress task file in `.lorah/tasks/`.
2. Read the tests written in the testing phase.
3. Write production code to make the tests pass. Do not write new
   tests.
4. Verify: run the full test suite. All tests must pass.
5. Update the task file: set status to `completed`, add
   implementation notes to the Log.
6. Commit.

## Blocked workflow

If the existing tests conflict with the design spec:

1. Discard uncommitted changes.
2. Set the task status to `blocked` with notes explaining the
   conflict.
3. Exit without committing.

The next iteration will route back to the testing phase to fix the
tests.
```

The prompts above are starting points. Adapt the role, rules, and workflow steps to match your project. Focus on boundaries and invariants rather than detailed instructions for every scenario.
