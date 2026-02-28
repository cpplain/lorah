# Effective Harnesses for Long-Running Agents

## Article Summary

Source: [Anthropic Engineering Blog](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)

### The Problem

Long-running agents face a fundamental challenge: maintaining coherent state and context across multiple sessions. Each new session begins without memory of prior work—like an engineering shift change where incoming workers lack context. Without structure, agents may:

- Lose track of completed work
- Re-implement already-finished features
- Leave code in half-finished, unmergeable states
- Fail to recover context efficiently when resuming
- Declare victory prematurely without proper verification

### The Solution: Structured Tracking Files

The article advocates for a two-phase approach using an **Initializer Agent** followed by incremental **Coding Agent** sessions, connected by persistent tracking files.

---

## Key Files

### 1. feature_list.json

A structured JSON file listing all required features with status tracking.

```json
{
  "category": "functional",
  "description": "New chat button creates fresh conversation",
  "steps": ["Step-by-step verification instructions"],
  "passes": false
}
```

**Critical constraint**: Agents may **only modify the `passes` field**. The article states:

> "It is unacceptable to remove or edit tests because this could lead to missing or buggy functionality."

**Format choice**: JSON was chosen over Markdown for the feature list:

> "We landed on using JSON for this, as the model is less likely to inappropriately change or overwrite JSON files compared to Markdown files."

**Scale example**: In the article's claude.ai clone demo, this meant over 200 features covering both functional requirements ("a user can open a new chat, type in a query, press enter, and see an AI response") and style requirements (UI polish, responsive layout).

### 2. claude-progress.txt

A chronological log of agent activities enabling subsequent sessions to quickly understand completed work without re-reading entire conversation context. Updated at the end of each session with:

- Summary of work completed
- Commits made
- Blockers encountered
- Suggested next steps

### 3. init.sh

An executable script that starts the development environment. Agents run this at session start to ensure consistent application state before implementing new features.

### 4. Git Repository

Used for version control with descriptive commit messages. Allows agents to:

- Revert problematic changes
- Maintain working codebase states between sessions
- Provide audit trail of all changes

---

## Two-Phase Workflow

### Phase 1: Initializer Agent

Used once at project start to establish the foundational tracking files:

1. **Create feature_list.json** - Analyze project requirements and create comprehensive feature list with:
   - Clear descriptions
   - Step-by-step verification procedures for each feature
   - All features start with `passes: false`

2. **Create init.sh** - Write startup script that:
   - Sets up the development environment
   - Installs dependencies
   - Starts development server
   - Is idempotent (safe to run multiple times)

3. **Initialize git repository** - Create initial commit with added files

4. **Create claude-progress.txt** - Document initial setup and suggested starting point

### Phase 2: Coding Agent

Used for each incremental work session. Each session follows this protocol:

#### Session Start

1. Run `pwd` to confirm working directory
2. Read `claude-progress.txt` for context on previous sessions
3. Run `git log` to see recent commits
4. Read `feature_list.json` to identify incomplete features
5. Start development server via `init.sh`

This "warm-up" sequence saves tokens by quickly recovering context instead of re-exploring the codebase.

#### Baseline Testing (Critical)

**Before implementing new features**, run baseline functionality tests:

- Select 1-2 features where `passes: true`
- Run their verification steps
- If any baseline tests fail, fix regressions BEFORE new work

This catches bugs introduced in previous sessions.

#### Feature Selection

Select the **highest-priority incomplete feature** where `passes` is `false`.

Work on **ONE feature per session**. This constraint ensures:

- Focused, completable work units
- Clean state at session end
- Easy rollback if needed

#### Implementation

1. Implement the feature incrementally
2. Commit frequently with descriptive messages
3. After implementation, run verification steps

#### End-to-End Verification

The article emphasizes that agents initially failed to verify end-to-end functionality. The solution:

> "Explicit prompting to use browser automation tools (Puppeteer MCP) for human-like testing rather than unit tests alone."

**Verification requirements:**

- Test through actual UI interactions (not just unit tests)
- Use browser automation (e.g., Puppeteer MCP) when applicable
- Verify as an end-user would experience the feature
- Only mark `passes: true` after full verification

#### Session End

**Before any session termination**:

1. Commit all changes with descriptive message
2. Update `feature_list.json`: set `passes: true` for verified features
3. Append to `claude-progress.txt`:
   - Session summary
   - Features completed
   - Commits made
   - Blockers encountered
   - Suggested next steps
4. Ensure code is in mergeable state

---

## Common Failure Modes and Solutions

| Problem                                  | Initializer Solution                             | Coding Agent Solution                              |
| ---------------------------------------- | ------------------------------------------------ | -------------------------------------------------- |
| Agent declares victory prematurely       | Create feature list with structured verification | Read feature file; work on single feature          |
| Buggy/undocumented code states           | Write git repo + progress file initially         | Read progress/git logs; test baseline; commit work |
| Features marked complete without testing | Establish feature list with verification steps   | Self-verify all features before marking complete   |
| Time wasted understanding app setup      | Write `init.sh` startup script                   | Read and execute `init.sh`                         |

---

## Core Principles

The patterns described here are derived from observing effective human engineers—the harness essentially codifies practices that experienced developers already follow during handoffs.

### Clean State Outputs

The system emphasizes producing orderly, documented code suitable for merging to main branches. This prevents subsequent agents from inheriting undocumented, half-finished features.

### Incremental Progress

- Work on one feature per session
- Use git commits with descriptive messages
- Write progress summaries for continuity
- Allows reverting failed changes

### Explicit Verification

Agents must self-verify features through end-to-end testing before marking them complete. Unit tests alone are insufficient—browser automation and user-perspective testing are required.

---

## Future Directions

The article notes open questions:

> "It's still unclear whether a single, general-purpose coding agent performs best across contexts, or if better performance can be achieved through a multi-agent architecture."

Potential specialized agents mentioned:

- Testing agent
- Quality assurance agent
- Code cleanup agent

Generalization beyond web development to:

- Scientific research
- Financial modeling

---

## Limitations

The article acknowledges current constraints:

- **Vision limitations**: Claude's visual processing has gaps that affect verification accuracy
- **Browser automation gaps**: Tools cannot detect all UI states (e.g., browser-native alert modals are invisible to Puppeteer)

These limitations inform which verification approaches are most reliable.
