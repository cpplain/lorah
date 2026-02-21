## YOUR ROLE - INITIALIZATION PHASE

You are cataloging all issues in this codebase. This runs once. Do NOT make any fixes — only discover and document.

### STEP 1: Orient Yourself

```bash
pwd
git log --oneline -10
git diff HEAD~1 --name-only 2>/dev/null || git ls-files
```

Understand what files are present and what has changed recently.

### STEP 2: Read All Source Files

Systematically read every source file. Use Glob to find all relevant files, then Read each one carefully.

**Examples — adapt to the actual language/project:**

- Go: `**/*.go` (exclude vendor and test files as needed)
- Python: `**/*.py` (exclude `__pycache__` and `.venv` directories)
- JavaScript/TypeScript: `**/*.{js,ts,jsx,tsx}` (exclude `node_modules`)
- Rust: `**/*.rs`

Use the Glob tool to discover files, then use the Read tool for each file. Do not skip files. Shallow coverage produces an incomplete inventory.

### STEP 3: Catalog Issues

For each issue found, categorize it as one of:

- **bug** — incorrect behavior, logic errors, off-by-one errors, wrong return values
- **security** — injection, improper auth, data exposure, unsafe deserialization, insecure defaults
- **logic** — design flaws, incorrect assumptions, missing edge cases, race conditions
- **performance** — unnecessary allocations, N+1 queries, blocking operations where async is needed
- **idiom** — non-idiomatic code for the language (non-idiomatic Go, non-idiomatic Python, etc.)
- **consistency** — patterns that differ from the established conventions in this codebase

**Focus on:**

- Issues that affect correctness, security, or reliability
- Language-idiomatic patterns that differ from community standards
- Inconsistencies with conventions already established in this codebase

**Do NOT flag:**

- Style preferences not established in the codebase
- Premature abstractions or over-engineering suggestions
- "Clean Code" dogma (excessive interfaces, design patterns for their own sake)
- Subjective naming preferences unless naming is actively misleading

### STEP 4: Write tasks.json

Create `.lorah/tasks.json` with all discovered issues **ordered by priority**:

```json
[
  {
    "name": "brief description of the issue",
    "category": "security",
    "file": "path/to/file.go",
    "line": 42,
    "passes": false,
    "notes": "optional detail about why this is an issue"
  }
]
```

**Priority ordering (highest to lowest):**

1. security
2. bug
3. logic
4. performance
5. idiom
6. consistency

**Important rules:**

- Every issue gets its own entry
- Issues must be ordered by priority (all security first, then bugs, then logic, etc.)
- If no issues are found in a category, skip that category
- Be thorough — it is better to over-report than under-report at this stage
- The fix phase will work through this list from top to bottom

### STEP 5: Create Progress File

Create `.lorah/progress.md` to track session handoffs:

```text
# Review Progress

## Inventory Complete
- Total issues: [count]
  - Security: [count]
  - Bugs: [count]
  - Logic: [count]
  - Performance: [count]
  - Idiom: [count]
  - Consistency: [count]

## Fix Session Log
(Each fix session will append progress here)
```

This file gives future sessions the context they need without reading the entire issue list.

### STEP 6: Commit

```bash
git add .lorah/tasks.json .lorah/progress.md
git commit -m "review: inventory complete"
```

Print a summary of how many issues were found per category.
