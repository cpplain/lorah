# Implementation Phase

## Workflow

1. Read `.lorah/plan.md` for scope and acceptance criteria.
2. Read the relevant design specs indexed in `docs/design/README.md`
   for behavioral details.
3. Review git history and current task file in `.lorah/tasks/`.
4. If tests were written in the testing phase, read them. Otherwise
   skip to step 6.
5. Read the relevant design spec section(s) referenced in the task.
6. Write production code to satisfy the acceptance criteria (and make
   tests pass, if they exist). Do not write new tests.
7. Verify: if tests exist, run the full test suite — all tests must
   pass. Otherwise, verify acceptance criteria directly (e.g., run
   commands, check file contents).
8. Update the task file: set status to `completed`, add
   implementation notes to the Log.
9. If acceptance criteria has been met, update `.lorah/plan.md`.
10. Commit.

## Blocked workflow

If the existing tests conflict with the design spec:

1. Discard uncommitted changes.
2. Set the task status to `blocked` with notes explaining the
   conflict.
3. Exit without committing.

The next iteration will route to the planning phase to reassess the
task.
