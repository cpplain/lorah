# Lorah Agent Harness - Implementation Review

**Repository**: `cpplain/lorah`
**Language**: Go
**Purpose**: Long-running autonomous coding agent orchestration for Claude Code CLI

---

## Executive Summary

Lorah is an implementation of Anthropic's recommended patterns for long-running agent harnesses, as documented in their blog post [Effective Harnesses for Long-Running Agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents). This review maps Lorah's implementation to the patterns described in Anthropic's research, identifying areas of full conformance, partial alignment, and intentional deviations.

**Key characteristics**:

- Clean separation between harness logic (orchestration) and agent behavior (prompts)
- Robust error recovery with exponential backoff
- Convention-based defaults with minimal configuration burden
- JSON task tracking preventing progress corruption
- Single binary deployment with no external dependencies

**Key strength**: Simplicity. Lorah does one thing well - orchestrate Claude Code CLI sessions with reliable state management.

---

## Architecture

### Core Components

```
main.go              → CLI entry point (run, verify, init, info)
lorah/
  runner.go          → Agent loop, phase selection, error recovery
  client.go          → Claude CLI subprocess execution
  config.go          → Configuration loading with deep merge
  tracking.go        → JSON checklist progress monitoring (JsonChecklistTracker)
  verify.go          → Pre-run environment validation
  messages.go        → Stream-JSON parser
  messages_types.go  → Message type definitions
  info.go            → Template embedding, scaffolding
  presets.go         → Built-in project type configs
  lock.go            → PID-based instance locking
  schema.go          → Configuration schema generation
```

### Control Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Acquire PID lock (harness.lock)                          │
│ 2. Load config (.lorah/config.json)                         │
│ 3. Initialize tracker (tasks.json)                          │
│ 4. Load session state (session.json)                        │
│ 5. Ensure tracking files exist                              │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     AGENT LOOP                               │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Select Phase:                                       │    │
│  │  - initialization (if not done)                     │    │
│  │  - implementation (iterative)                       │    │
│  └────────────────────────────────────────────────────┘    │
│                         │                                    │
│                         ▼                                    │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Load prompt from .lorah/prompts/{phase}.md         │    │
│  │ Prepend error context if previous session failed   │    │
│  └────────────────────────────────────────────────────┘    │
│                         │                                    │
│                         ▼                                    │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Run Claude CLI session (subprocess)                │    │
│  │  - Stream output to terminal                        │    │
│  │  - Parse JSON messages                              │    │
│  │  - Capture result                                   │    │
│  └────────────────────────────────────────────────────┘    │
│                         │                                    │
│                         ▼                                    │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Success? → Reset errors, mark phase, continue       │    │
│  │ Error?   → Increment errors, backoff, retry         │    │
│  │ Complete? → Exit loop                               │    │
│  └────────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Implementation Overview

### Two-Phase Workflow

Lorah hardcodes two phases:

| Phase              | Prompt File                        | Purpose            | Runs           |
| ------------------ | ---------------------------------- | ------------------ | -------------- |
| **initialization** | `.lorah/prompts/initialization.md` | One-time setup     | Once           |
| **implementation** | `.lorah/prompts/implementation.md` | Iterative building | Until complete |

**Phase selection logic** (from `runner.go`):

```go
if !tracker.IsInitialized() && !initCompleted {
    return initializationPhase, InitializationPromptFile
}
return implementationPhase, ImplementationPromptFile
```

**Initialization detection**: `tracker.IsInitialized()` returns true when `tasks.json` contains at least one item.

**Completion detection**: `tracker.IsComplete()` returns true when all items in `tasks.json` have `"passes": true` and count > 0.

### State Files

Lorah maintains four state files with clear ownership boundaries:

#### 1. `tasks.json` (Agent-written)

Equivalent to Anthropic's `feature_list.json`. Progress checklist with task schema:

```json
[
  { "name": "Task name", "description": "What to build", "passes": false },
  { "name": "Another task", "description": "Description", "passes": true }
]
```

**Tracker operations** (`tracking.go`):

- `IsInitialized()` → file exists and has items (total > 0)
- `IsComplete()` → all items have `passes: true` and count > 0
- `GetSummary()` → (passing_count, total_count)

#### 2. `progress.md` (Agent-written)

Equivalent to Anthropic's `claude-progress.txt`. Handoff notes between sessions for human-readable context.

#### 3. `session.json` (Harness-written)

Session state tracking:

```json
{
  "session_number": 5,
  "completed_phases": ["initialization"]
}
```

**Updated by**: Harness only (not the agent)
**Atomic writes**: Uses temp file + rename pattern

#### 4. `harness.lock` (Harness-written)

Contains PID of running harness instance. Prevents concurrent runs; detects and clears stale locks.

**Clear ownership boundaries**:

| File           | Owner   | Purpose                   |
| -------------- | ------- | ------------------------- |
| `tasks.json`   | Agent   | Task completion tracking  |
| `progress.md`  | Agent   | Session handoff notes     |
| `session.json` | Harness | Phase and session state   |
| `harness.lock` | Harness | Concurrent run prevention |
| `config.json`  | User    | Configuration overrides   |

### Configuration

**Convention-based paths** (fixed):

```
.lorah/
  config.json              # Optional overrides
  spec.md                  # Project specification
  tasks.json               # Created by init phase
  progress.md              # Created by init phase
  session.json             # Created by harness
  harness.lock             # Created by harness
  prompts/
    initialization.md      # Init phase prompt
    implementation.md      # Build phase prompt
```

**Loading strategy**:

1. Load embedded defaults
2. Deep-merge user `config.json` over defaults
3. Apply CLI flag overrides
4. Validate harness section only (Claude section passed through)

**No per-phase config** - single global config applies to all phases.

---

## Mapping Implementation to Blog Post Patterns

This section maps Lorah's design to the six key problems identified in Anthropic's [Effective Harnesses for Long-Running Agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents) blog post.

### Problem 1: Premature Completion

**Blog finding**: Agents declare "I'm done!" when the project is only partially complete, lacking objective completion criteria.

**Lorah's approach**:

The initialization phase creates `tasks.json` with a boolean `passes` field for each task. Completion is only reached when **all** tasks have `"passes": true`. The `tracker.IsComplete()` function enforces this objective criterion:

```go
func (t *JsonChecklistTracker) IsComplete() bool {
    passing, total := t.GetSummary()
    return passing == total && total > 0
}
```

**Conformance**: ✓ FULL - Uses same JSON boolean tracking pattern as Anthropic's reference implementation

**Deviation**: Simpler schema - Lorah uses `{name, description, passes}` vs Anthropic's `{category, description, steps[], passes}`. Trade-off: less structure but lower corruption risk.

### Problem 2: Context Loss Between Sessions

**Blog finding**: Agents waste time re-understanding project state when resuming, especially across context window boundaries.

**Lorah's approach**:

Three-part context restoration pattern:

1. **`progress.md`**: Agent-written session notes explaining what was accomplished, issues found, and next steps
2. **Git log**: Implementation history provides concrete record of changes
3. **Orientation commands**: Implementation prompt explicitly requires reading state files

From `.lorah/prompts/implementation.md`:

```markdown
### STEP 1: Get Your Bearings

pwd && ls -la
cat .lorah/spec.md
cat .lorah/tasks.json
cat .lorah/progress.md
git log --oneline -10
```

**Conformance**: ✓ FULL - Three-part context restoration matches Anthropic's pattern

**Deviation**: No `init.sh` script execution by harness. Lorah relies on agent-written setup instructions rather than a harness-executed environment setup script.

### Problem 3: Silent Regression

**Blog finding**: Agents introduce bugs when implementing new features, breaking previously working functionality without detecting it.

**Lorah's approach**:

Default prompts say "Implement & Test" and "verify it works" but do NOT include an explicit regression testing step.

From `.lorah/prompts/implementation.md`:

```markdown
### STEP 3: Implement & Test

Implement the task and verify it works.
```

**Conformance**: ✗ MISSING - Structure supports regression testing but prompts don't mandate it

**Anthropic's approach**: Step 3 of coding workflow explicitly requires testing 1-2 features marked `"passes": true` to verify they still work before implementing new features.

**Gap**: Lorah's prompts lack "BEFORE implementing new features, test 1-2 previously passing tasks to verify no regressions" instruction.

### Problem 4: Insufficient Verification

**Blog finding**: Agents mark features complete without thorough end-to-end testing. Anthropic recommends browser automation (Puppeteer MCP) to test as human users would.

**Lorah's approach**:

Prompts mention testing but don't emphasize end-to-end verification or require specific testing methods:

```markdown
### STEP 3: Implement & Test

Implement the task and verify it works.
```

**Conformance**: ✗ MISSING - Testing mentioned but not emphasized with critical language or browser automation requirements

**Anthropic's approach**: Explicit Puppeteer MCP requirement with strong language:

- "CRITICAL: You MUST verify features through the actual UI"
- "DON'T: Only test with curl (backend testing alone is insufficient)"
- "DON'T: Use JavaScript evaluation to bypass UI"

**Gap**: Lorah's prompts lack browser/screenshot verification emphasis and don't mandate end-to-end testing through the UI.

### Problem 5: Feature List Corruption

**Blog finding**: "The model is less likely to inappropriately change or overwrite JSON files compared to Markdown files." Agents may remove or modify test cases inappropriately.

**Lorah's approach**:

Uses JSON format (`tasks.json`) with minimal schema to reduce corruption risk. The schema itself (no nested arrays) makes accidental corruption during edits less likely.

```json
[{ "name": "Task name", "description": "What to build", "passes": false }]
```

**Conformance**: ✓ FULL - JSON format choice matches Anthropic's recommendation

**Deviation**: Prompts lack emphatic "CATASTROPHIC" language. Anthropic's prompts include:

- "IT IS CATASTROPHIC TO REMOVE OR EDIT FEATURES IN FUTURE SESSIONS"
- "YOU CAN ONLY MODIFY ONE FIELD: 'passes' — NEVER: Remove tests, edit descriptions..."

Lorah's prompts say "Mark task as `\"passes\": true`" but don't explicitly forbid other modifications with strong language.

### Problem 6: Inadequate Test Coverage

**Blog finding**: Vague high-level requirements lead to incomplete implementations. Anthropic recommends expanding specs into "200+ discrete, testable features" with detailed verification steps.

**Lorah's approach**:

Tasks are user-defined during initialization phase. No minimum count requirement. No structured verification steps per task.

From `.lorah/prompts/initialization.md`:

```markdown
### STEP 2: Create Task List

Create `.lorah/tasks.json` with testable requirements:

[{ "name": "Task name", "description": "What to build", "passes": false }]
```

**Conformance**: ~ PARTIAL - Structure supports comprehensive task lists, but doesn't enforce scale or detail

**Anthropic's approach**:

- Minimum 200 features
- Mix of narrow tests (2-5 steps) and comprehensive tests (10+ steps)
- At least 25 tests must have 10+ steps each
- Both "functional" and "style" categories

**Gap**: Lorah doesn't enforce minimum task count, doesn't require step arrays per task, and doesn't mandate comprehensive coverage.

---

## Conformance Summary

Quick-reference table showing alignment with Anthropic's patterns:

| Anthropic Pattern                 | Lorah Implementation                        | Status    |
| --------------------------------- | ------------------------------------------- | --------- |
| Two-agent pattern (init + build)  | Fixed init + implementation phases          | ✓ FULL    |
| JSON feature tracking             | `tasks.json` with `passes` boolean          | ✓ FULL    |
| Session orientation               | Prompts mandate pwd/git log/progress review | ✓ FULL    |
| Single feature per session        | Implicit in implementation prompt           | ✓ FULL    |
| Production-ready between sessions | "Commit your changes" in prompt             | ✓ FULL    |
| Clear handoff artifacts           | `progress.md` + git commits                 | ✓ FULL    |
| Error recovery with backoff       | Exponential backoff implemented             | ✓ FULL    |
| Sandbox isolation                 | Delegated to Claude CLI                     | ✓ FULL    |
| Regression testing before work    | Not in default prompts                      | ✗ MISSING |
| Browser/E2E verification          | Not in default prompts                      | ✗ MISSING |
| init.sh script execution          | Not implemented                             | ✗ MISSING |
| "CATASTROPHIC" immutability       | Not in default prompts                      | ✗ MISSING |
| 200+ features requirement         | Not enforced                                | ✗ MISSING |

**Summary**: 8/13 patterns fully implemented, 5/13 intentionally simplified or omitted.

---

## Key Design Decisions

### Security Delegation vs Defense-in-Depth

**Decision**: Delegate all security to Claude CLI rather than implement custom validation layer.

**Trade-off**:

- **Anthropic's reference implementation**: Custom bash allowlist (18 commands), specialized validators for risky commands (pkill, chmod), fail-safe design
- **Lorah**: Pure passthrough to Claude CLI `--settings` flag for sandbox configuration

From `config.go`:

```go
// Claude config is not validated - Claude CLI handles its own validation
```

**Rationale**: Claude CLI already provides robust sandboxing with OS-level isolation, filesystem restrictions, and configurable permissions. Duplicating this logic would increase maintenance burden and create two sources of truth for security policy. By delegating, Lorah remains simple and leverages Claude CLI's battle-tested security model.

### Convention-Based Configuration

**Decision**: Fixed file paths (`.lorah/tasks.json`, `.lorah/progress.md`) and two-phase model rather than flexible configuration.

**Trade-off**:

- **Pros**: Simpler mental model, less configuration surface, works out-of-box with zero config
- **Cons**: Can't add phases without code changes, can't customize file paths, can't have per-phase configuration

**Rationale**: Convention over configuration reduces cognitive load. Users get a working harness immediately with `lorah init`, and only configure when they need to override defaults. The deep-merge config system allows partial overrides without exposing every internal detail.

### Minimal Task Schema

**Decision**: Simple `{name, description, passes}` schema vs Anthropic's `{category, description, steps[], passes}`.

**Trade-off**:

- **Anthropic schema**: More structure, explicit verification steps, categorization
- **Lorah schema**: Minimal fields, lower corruption risk, simpler for agents to work with

**Rationale**: Each additional field increases the surface area for agent mistakes. The simpler schema reduces the chance of accidental corruption while still providing enough structure for completion tracking. Verification steps can be written in the `description` field if needed.

---

## Strengths

### 1. Simplicity

- Single binary (no runtime dependencies except Claude CLI)
- Minimal configuration surface
- Fixed two-phase model (easy to understand)
- Flat package structure (~3000 lines in one package)

### 2. Robustness

- PID-based locking prevents concurrent runs
- Exponential backoff on errors with circuit breaker
- Atomic file writes (temp file + rename)
- Stream-JSON parsing handles unknown message types gracefully

### 3. Clear Ownership Boundaries

Harness and agent have clearly separated state files with no overlap. Each file has a single owner (harness, agent, or user), reducing confusion and preventing race conditions.

### 4. Workflow Flexibility

Same harness works for feature development, code review, bug fixing, refactoring - just swap prompt files. See `workflows/review/` for an example of code review mode.

---

## Limitations

### 1. Fixed Phase Model

Only two phases (init + implementation) hardcoded in `runner.go`.

Can't easily:

- Add research phase before init
- Add review/QA phase after implementation
- Run deployment phase after features complete

**Workaround**: Modify prompts to include multi-step workflows within existing phases.

### 2. No Per-Phase Configuration

Single `config.json` for entire harness. Can't override settings (like `max-turns` or `model`) per phase.

### 3. Minimal Task Schema

Tasks have `name`, `description`, and `passes`. Can't express:

- Task dependencies (blocking relationships)
- Priority levels
- Structured verification steps (step arrays)
- Categories (functional vs style)

**Trade-off**: Simpler schema = less corruption risk.

### 4. No Init Script Execution

Environment setup via prompts only, not executable script. The harness doesn't run `init.sh` like Anthropic's reference - it relies on the agent to set up the environment as instructed.

### 5. Prompt Gaps Compared to Anthropic

Default prompts lack:

- Explicit regression testing step before new work
- Browser/screenshot verification emphasis
- "CRITICAL" or "CATASTROPHIC" language for immutability rules
- Minimum feature count requirements (200+)
- Structured verification steps per feature

---

## Lorah-Specific Features

Features Lorah provides beyond the minimal reference implementation:

### 1. Presets System

Built-in configurations for common project types:

| Preset       | Configuration                                            |
| ------------ | -------------------------------------------------------- |
| `python`     | Python-specific settings (pip, uv, PyPI access)          |
| `go`         | Go-specific settings (module proxy access)               |
| `rust`       | Rust-specific settings (crates.io access)                |
| `web-nodejs` | Node.js web app settings (npm, local dev server binding) |
| `read-only`  | Analysis-only mode (restricts tools to Read, Glob, Grep) |

**Usage**: `lorah init --preset python`

### 2. Workflow Flexibility via Prompts

Same harness supports different workflows by swapping prompt files:

**Build workflow** (default):

- Init: Create task list
- Implementation: Build tasks one at a time

**Review workflow** (`workflows/review/`):

- Init: Catalog ALL issues in codebase (no fixes)
- Implementation: Fix ONE issue per session, with explicit regression verification

**Same harness, different behavior** - only prompts change.

### 3. Error Recovery with Exponential Backoff

Configurable error recovery with circuit breaker:

```json
"error-recovery": {
  "max-consecutive-errors": 5,
  "initial-backoff-seconds": 5.0,
  "max-backoff-seconds": 120.0,
  "backoff-multiplier": 2.0,
  "max-error-message-length": 2000
}
```

**Backoff formula**: `delay = min(initial * multiplier^(n-1), max)`

**Backoff schedule**:

- Error 1: 5 seconds
- Error 2: 10 seconds
- Error 3: 20 seconds
- Error 4: 40 seconds
- Error 5: 80 seconds
- Error 6+: 120 seconds (capped)

**Error context injection**: On retry, previous error message is prepended to prompt:

```
Note: The previous session encountered an error: {error_message}
Please continue with your work.

{original_prompt}
```

### 4. Optional Configuration with Defaults

Works out-of-box without config file. Embedded defaults + deep-merge user overrides:

1. Load embedded template defaults
2. Deep-merge user `config.json` over defaults (if exists)
3. Apply CLI flag overrides
4. Validate harness section only

**Philosophy**: Only configure what you need to change.

### 5. Single Binary Distribution

Go binary with no external runtime dependencies (except Claude CLI). Simpler deployment than Python-based reference implementation:

```bash
# Build
go build -o ./bin/lorah .

# Install
go install .

# No pip, no virtualenv, no package.json
```

---

## Usage Example

```bash
# Initialize project with preset
lorah init --project-dir ./my-app --preset go

# Verify setup
lorah verify --project-dir ./my-app

# Run agent loop
lorah run --project-dir ./my-app

# Continue after interruption (same command)
lorah run --project-dir ./my-app

# Limit iterations for testing
lorah run --project-dir ./my-app --max-iterations 5
```

**Expected timeline**:

- Session 1 (init): Several minutes
- Sessions 2+: 5-15 minutes each
- Full application: Many hours/days (depends on task count)

**Session resumability**:

- Press Ctrl+C to pause
- Run same command to resume
- Progress persists via git and `tasks.json`

---

## Conclusion

Lorah is a functional agent harness that implements the core structural patterns from Anthropic's guidance while making pragmatic trade-offs favoring simplicity.

**Full conformance** (8/13 Anthropic patterns):

- Two-agent architecture (init + implementation)
- JSON progress tracking with boolean completion field
- Session orientation via mandatory state file reads
- Single-feature-per-session workflow
- Clean handoffs via git commits and progress notes
- Error recovery with exponential backoff
- Sandbox isolation (delegated to Claude CLI)
- Production-ready code between sessions

**Intentional simplifications** (5/13 patterns):

- Regression testing not mandated in prompts
- Browser/E2E verification not emphasized
- No init.sh harness execution
- No emphatic "CATASTROPHIC" immutability language
- No minimum feature count enforcement (200+)

These gaps reflect design choices favoring minimal prompt complexity and simpler mental models over comprehensive failure-mode coverage. The harness provides robust orchestration structure; prompts can be enhanced to add missing verification patterns when needed.

**Best suited for**:

- Projects with straightforward init → build workflows
- Teams wanting minimal configuration overhead
- Use cases where two phases suffice
- Developers comfortable writing custom prompts for specialized workflows

The codebase is well-structured, idiomatic Go, and provides a solid foundation for autonomous coding workflows.
