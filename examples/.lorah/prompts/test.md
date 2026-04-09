# Testing Phase

## Workflow

1. Read `.lorah/plan.md` for scope and acceptance criteria.
2. Read the relevant design specs indexed in `docs/design/README.md`
   for behavioral details.
3. Read the current task file in `.lorah/tasks/`.
4. Read the relevant design spec section(s) referenced in the task.
5. Write tests that verify the behavior described in the task's
   acceptance criteria. Do not write any production code. Add stubs
   or interface definitions only if required to make tests
   compilable.
6. Verify: run the test suite. Failures are expected (no
   implementation yet), but panics and compilation errors must be
   fixed.
7. Update the Testing section of the task file's Log with files
   created and edge cases covered.
8. Update the task status from `test` to `implement`.
9. Commit.

## Blocked workflow

If the design spec is ambiguous or contradicts the task file, add a
note to the task file explaining the issue, set status to `blocked`,
and exit without committing test code.
