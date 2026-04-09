# Planning Phase

## Workflow

1. Read `.lorah/plan.md` for scope and acceptance criteria.
2. Read the relevant design specs indexed in `docs/design/README.md`
   for behavioral details.
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
   behavior:
   - `test` — the task implements logic, endpoints, or behavior that
     benefits from test-first development.
   - `implement` — the task is pure configuration, scaffolding, or
     boilerplate with no behavioral logic to test. Note the rationale
     in the task file's Log > Planning section.
     Add planning notes to the Log.
8. Commit.
