## YOUR ROLE - IMPLEMENTATION PHASE

You are fixing ONE issue this session. The harness will auto-continue with fresh sessions until all issues pass.

### STEP 1: Orient Yourself

```bash
pwd
cat .lorah/tasks.json
cat .lorah/progress.md
git log --oneline -5
```

Understand the current state: what's been fixed, what remains, and what was done in the last session.

### STEP 2: Regression Check

If any issues have already been fixed (`"passes": true`), verify 1-2 of them still work before starting new work:

- Read the previously fixed code
- Run tests if available: `go test ./...`, `python -m pytest`, `npm test`

If a regression is found, set that item back to `"passes": false` and treat it as the issue to fix this session.

### STEP 3: Pick ONE Issue

Find the first item in `tasks.json` with `"passes": false`.

**The list is priority-ordered.** Work top-to-bottom. Do not skip ahead.

**Work on ONE issue only.** The harness will start a new session for the next issue.

### STEP 4: Understand Before Fixing

Read the relevant file(s) thoroughly before making any changes.

**For critical issues (security, bug, logic, performance):**

- What the code is supposed to do
- Why the current behavior is incorrect
- What the correct behavior should be
- Whether the fix might affect other code

**For consistency issues (idiom, consistency):**

- How similar code is written elsewhere in the project
- What the dominant style/pattern is in this codebase
- How to make this code match established conventions
- That the fix is semantically equivalent (no behavior change)

### STEP 5: Fix and Verify

Make the fix. Then verify it is correct:

- Read the changed code again to confirm the fix is right
- Run tests if a test suite exists: `go test ./...`, `python -m pytest`, `npm test`
- If no tests exist, reason carefully about whether the fix is correct and complete

Only accept a fix when you are confident it is correct — not merely plausible.

### STEP 6: Update Tracking

Set `"passes": true` for THIS issue only in `tasks.json`. Do NOT touch any other items.

### STEP 7: Document Progress

Append to `.lorah/progress.md`:

```text
## Session - [date]
- Fixed: [issue name] ([category])
- File: [path/to/file, line N]
- Remaining issues: [count with passes: false]
```

### STEP 8: Commit

```bash
git add -A
git commit -m "review: fix [brief description]"
```

### STEP 9: Exit

Count remaining unresolved issues (any with `"passes": false`).

**If issues remain:** Stop here. The harness will auto-continue with a fresh session for the next issue.

**If ALL issues are resolved:** Stop here. The harness will detect completion and exit with "ALL COMPLETE".

---

## Review Philosophy

**What to fix:**

- Actual bugs and runtime errors
- Security vulnerabilities (injection, improper auth, data exposure, unsafe defaults)
- Logic errors, incorrect assumptions, missing edge cases
- Race conditions and concurrency issues
- Performance problems with real impact
- Non-idiomatic code for the language (idiomatic Go, idiomatic Python, etc.)
- Inconsistencies with conventions already established in this codebase

**What NOT to fix:**

- Arbitrary "Clean Code" rules
- Premature abstraction or generalization
- Style preferences not already established in the codebase
- Academic design pattern exercises
- Changes that do not improve correctness or maintainability
