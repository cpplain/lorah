# Review Workflow

A structured code review harness that separates issue discovery from fixing and applies consistent criteria across the entire codebase using Anthropic's proven two-phase pattern.

## Why a Harness for Code Review?

Interactive code review tends to degrade over iterations. Each review pass catches some issues, fixes introduce others, and attention drifts as the session lengthens. The result is a whack-a-mole cycle: fix one thing, discover another, repeat until "good enough."

A harness-based review solves this by enforcing structure:

- **Discovery before fixing** — The initialization phase catalogs all issues before any changes are made. Nothing gets missed because a fix drew attention elsewhere.
- **Consistent criteria** — Each fix session applies the same standards. There is no drift between early and late review.
- **Priority ordering** — Issues are ordered by priority (security → bugs → logic → performance → idiom → consistency) and fixed systematically.
- **Built-in verification** — Each fix is verified before being marked complete. Regression checks run before new work starts.

## Review Philosophy

This workflow focuses on **practical quality** — issues that affect correctness, reliability, and maintainability by practitioners of the language.

**What it reviews:**

- Actual bugs and runtime errors
- Security vulnerabilities (injection, improper auth, data exposure, unsafe defaults)
- Logic errors, incorrect assumptions, missing edge cases
- Race conditions and concurrency issues
- Resource leaks and performance problems that have real impact
- Non-idiomatic code for the language (idiomatic Go, idiomatic Python, idiomatic Rust, etc.)
- Inconsistencies with conventions already established in the codebase

**What it does not review:**

- Arbitrary "Clean Code" rules
- Premature abstraction or generalization
- Style preferences not already established in the codebase
- Academic design pattern exercises
- Changes that do not improve correctness or maintainability

**On idiomatic code:** Language communities develop idioms because they work — they improve readability for practitioners, reduce cognitive load, and align with how the ecosystem is designed. Idiomatic Go (explicit error handling, minimal interfaces, receiver naming), idiomatic Python (comprehensions, context managers, duck typing), and idiomatic Rust (ownership patterns, Result/Option chaining) are practical standards, not academic exercises.

## Phases

This workflow follows Anthropic's proven two-phase pattern for autonomous agents:

### 1. Initialization (runs once)

Reads every source file and catalogs all issues into `.lorah/tasks.json` **ordered by priority**:

1. Security issues (highest priority)
2. Bugs
3. Logic errors
4. Performance problems
5. Idiomatic issues
6. Consistency issues (lowest priority)

No fixes are made during initialization. This separation prevents the common failure mode where early fixes distract from later discovery.

### 2. Implementation (iterative)

Works through the issue list from top to bottom, one issue per session. Each session:

1. Orients (checks progress, git log, last session's work)
2. Verifies previously fixed issues still work (regression check)
3. Picks the first incomplete issue
4. Fixes and verifies it
5. Updates tracking and commits
6. Documents progress
7. Exits cleanly

The harness auto-continues with fresh sessions until all issues pass, then exits with "ALL COMPLETE".

## Usage

```bash
cp -r workflows/review/.lorah /path/to/your/project/
cd /path/to/your/project
lorah run
```

Note: This workflow does not include `config.json` — lorah uses reasonable defaults.

## Output

After a successful run, the project directory will contain:

| File                 | Description                                     |
| -------------------- | ----------------------------------------------- |
| `.lorah/tasks.json`  | Complete issue inventory with resolution status |
| `.lorah/progress.md` | Session-by-session progress log                 |

Each fix is committed individually with a descriptive message, giving you a clean git history of every change made during the review.

## Security

This workflow uses `bypassPermissions` with sandboxing enabled. The sandbox confines all operations to the project directory — the agent cannot read or modify files outside the project. Version control provides the recovery mechanism: any unwanted change can be reverted with `git revert` or `git checkout`.

This follows [Anthropic's guidance](https://www.anthropic.com/engineering/claude-code-sandboxing) that sandboxing is the preferred security mechanism for autonomous agents, replacing approval prompts (which cause approval fatigue) with pre-defined filesystem boundaries.
