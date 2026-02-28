# Lorah Enhancement Recommendations

**Goal**: Make Lorah strictly enforce Anthropic's recommended patterns for long-running autonomous coding agents.

**Philosophy**: Pre-v1, Lorah should be opinionated. Enforce proven patterns; relax constraints later based on feedback.

---

## Context

This document synthesizes:

1. [Anthropic Blog Post](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents) - Core concepts
2. [Anthropic Reference Implementation](https://github.com/anthropics/claude-quickstarts/tree/main/autonomous-coding) - Python quickstart
3. Lorah Implementation - Current Go-based harness

### What Lorah Does Well

| Strength             | Details                                                      |
| -------------------- | ------------------------------------------------------------ |
| Single binary        | No runtime dependencies except Claude CLI                    |
| Config system        | JSON config with deep merge over defaults                    |
| Error recovery       | Exponential backoff with circuit breaker                     |
| State separation     | `tasks.json` (agent) vs `session.json` (harness)             |
| Presets              | Built-in configs for python, go, rust, web-nodejs, read-only |
| Workflow flexibility | Alternative prompt sets via `workflows/`                     |
| PID locking          | Prevents concurrent runs                                     |
| Atomic writes        | Temp file + rename prevents corruption                       |

### Gaps vs. Anthropic's Reference

| Feature              | Anthropic Has                                 | Lorah Missing     |
| -------------------- | --------------------------------------------- | ----------------- |
| Regression testing   | Step 3: Test passing features before new work | Not in prompts    |
| Browser verification | Puppeteer MCP, screenshots, E2E testing       | Not in prompts    |
| init.sh execution    | Harness runs setup script each session        | Prompt-based only |
| Strong immutability  | "IT IS CATASTROPHIC to edit features"         | Softer language   |
| Quality standards    | Zero console errors, polished UI checklist    | Not explicit      |
| Custom security      | Bash command allowlist                        | Delegates to CLI  |

---

## Naming Alignment

Adopt Anthropic's taxonomy for cognitive consistency:

| Concept         | Anthropic             | Lorah Current | Adopt                |
| --------------- | --------------------- | ------------- | -------------------- |
| Checklist file  | `feature_list.json`   | `tasks.json`  | `feature_list.json`  |
| Checklist items | features              | tasks         | features             |
| Progress log    | `claude-progress.txt` | `progress.md` | `claude-progress.md` |
| Specification   | `app_spec.txt`        | `spec.md`     | `app_spec.md`        |
| Setup script    | `init.sh`             | (none)        | `init.sh`            |

**Keep `.md` extensions** (better than `.txt` for formatting/rendering).

**Keep phase names**: "initialization/implementation" clearer than "Initializer Agent/Coding Agent".

---

## Prompt Enforcement Architecture

### Problem

Users providing custom prompts can skip critical steps (regression testing, verification).

### Solution: Wrapper + Slot Injection

Harness controls **structure and requirements**; user controls **project-specific details**.

```
┌─────────────────────────────────────────┐
│ [HARNESS: orientation]                  │  ← Cannot remove
│ [HARNESS: regression testing]           │  ← Cannot remove
├─────────────────────────────────────────┤
│ [USER: implementation.md]               │  ← User controls
├─────────────────────────────────────────┤
│ [HARNESS: verification wrapper]         │  ← Cannot remove
│ [USER: verification.md]                 │  ← User controls method
│ [HARNESS: verification footer]          │  ← Cannot remove
├─────────────────────────────────────────┤
│ [USER: quality-standards.md]            │  ← User controls criteria
├─────────────────────────────────────────┤
│ [HARNESS: exit protocol]                │  ← Cannot remove
│ [HARNESS: immutability warning]         │  ← Cannot remove
└─────────────────────────────────────────┘
```

### What's Embedded vs User-Provided

| Content                        | Source                        | User Can Modify?    |
| ------------------------------ | ----------------------------- | ------------------- |
| Orientation protocol           | Harness embedded              | No                  |
| Regression testing requirement | Harness embedded              | No                  |
| "You MUST verify" framing      | Harness embedded              | No                  |
| _How_ to verify                | User's `verification.md`      | Yes (file required) |
| Quality criteria               | User's `quality-standards.md` | Yes (file required) |
| Exit protocol                  | Harness embedded              | No                  |
| Feature immutability warning   | Harness embedded              | No                  |

### Prompt Assembly

```go
func assembleImplementationPrompt(userDir string) string {
    return fmt.Sprintf(`# Implementation Phase

%s

%s

---

## Implementation

%s

---

## Verification (REQUIRED)

⚠️ You MUST verify the feature works before marking it complete.
Do not skip this step. Do not rely on code inspection alone.

### How to Verify This Project

%s

### After Verification

- If verification passes → update feature_list.json, set passes: true
- If verification fails → fix the issue, re-verify

---

## Quality Standards

%s

---

%s
`,
        embedded("orientation"),
        embedded("regression"),
        readUserFile("implementation.md"),
        readUserFile("verification.md"),
        readUserFile("quality-standards.md"),
        embedded("exit"),
    )
}
```

---

## Initialization Enforcement

### What Can Go Wrong

| Problem                         | Consequence                                |
| ------------------------------- | ------------------------------------------ |
| Vague features ("make it work") | Agent can't verify, marks done prematurely |
| Features too large              | Can't finish in one session, broken state  |
| Missing init.sh                 | No reliable environment setup              |
| No git init                     | No rollback capability                     |

### Post-Init Validation

Harness validates before allowing implementation phase:

```go
func validateInitialization(projectDir string) error {
    checks := []struct {
        name  string
        check func() error
    }{
        {"feature_list.json exists", checkFeatureListExists},
        {"feature_list.json has 10-200 items", checkFeatureCount},
        {"feature_list.json schema valid", checkFeatureSchema},
        {"no vague feature names", checkFeatureQuality},
        {"all features start passes: false", checkInitialState},
        {"init.sh exists", checkInitShExists},
        {"init.sh is executable", checkInitShExecutable},
        {"init.sh runs successfully", runInitSh},
        {"git initialized", checkGitRepo},
        {"initial commit exists", checkGitCommits},
        {"claude-progress.md exists", checkProgressExists},
    }

    for _, c := range checks {
        if err := c.check(); err != nil {
            return fmt.Errorf("init validation [%s]: %w", c.name, err)
        }
    }
    return nil
}
```

### Feature Quality Validation

Detect vague feature names:

```go
vaguePatterns := []string{
    "^make .* work$",
    "^add \\w+$",        // "add login" too vague
    "^fix \\w+$",        // "fix bugs" too vague
    "^implement \\w+$",  // "implement auth" too vague
}
```

**Implementation phase cannot start until initialization passes all checks.**

---

## Directory Structure

### Configured Project

```
my-project/
├── .lorah/
│   ├── config.json                    # User: harness + claude settings
│   ├── app_spec.md                    # User: project specification
│   │
│   ├── prompts/
│   │   ├── initialization/
│   │   │   ├── project-description.md # User: what to build
│   │   │   └── feature-guidance.md    # User: how to structure features
│   │   │
│   │   └── implementation/
│   │       ├── implementation.md      # User: project-specific guidance
│   │       ├── verification.md        # User: how to verify (preset default)
│   │       └── quality-standards.md   # User: quality criteria (preset default)
│   │
│   ├── feature_list.json              # Agent-created, harness-validated
│   ├── claude-progress.md             # Agent-created
│   ├── session.json                   # Harness-created
│   │
│   └── init.sh                        # Agent-created, harness-executed
│
└── [project source code...]
```

### Embedded in Harness Binary

```
embedded/
├── wrappers/
│   ├── orientation.md          # pwd, ls, spec, features, git log
│   ├── regression.md           # Test passing features first
│   ├── verification-wrapper.md # "You MUST verify..."
│   ├── exit.md                 # Commit all, leave working
│   └── immutability.md         # "CATASTROPHIC to edit features"
│
└── presets/
    ├── web-frontend/
    │   ├── verification.md
    │   └── quality-standards.md
    ├── cli-tool/
    │   ├── verification.md
    │   └── quality-standards.md
    ├── library/
    ├── backend-api/
    └── data-pipeline/
```

---

## Project-Type Presets

Verification and quality standards vary by project type:

### web-frontend

**verification.md:**

```markdown
For each feature:

1. Test through browser UI (not just code inspection)
2. Use actual browser interactions (clicks, keyboard)
3. Take screenshots of key states
4. Check browser console for errors (must be zero)
5. Verify responsive behavior
```

**quality-standards.md:**

```markdown
- Zero console errors
- Polished UI design
- Responsive layout
- Accessible (keyboard navigation, contrast)
```

### cli-tool

**verification.md:**

```markdown
For each feature:

1. Run the command with expected inputs
2. Capture stdout/stderr output
3. Verify exit code is 0 for success
4. Test error cases (invalid input, missing flags)
5. Confirm help text is accurate
```

**quality-standards.md:**

```markdown
- Clear, helpful error messages
- Consistent flag naming (--verbose, -v)
- Exit code 0 on success, non-zero on failure
- Works in pipeline (stdin/stdout friendly)
```

### library

**verification.md:**

```markdown
For each feature:

1. Run test suite (pytest/go test/cargo test)
2. Verify public API matches documentation
3. Check for breaking changes
4. Test with example usage code
```

**quality-standards.md:**

```markdown
- Test coverage on public API
- Documented public functions
- No breaking changes without version bump
- Consistent error handling
```

### backend-api

**verification.md:**

```markdown
For each feature:

1. Test endpoints with curl/httpie
2. Verify response status codes
3. Check response body structure
4. Test error cases (400, 404, 500)
5. Verify database state changes
```

**quality-standards.md:**

```markdown
- Consistent response format
- Proper HTTP status codes
- Clear error messages
- Input validation
```

---

## Validation Flow

```
lorah run
    │
    ▼
┌─────────────────────────────────────┐
│ Phase: Initialization               │
│                                     │
│ 1. Assemble init prompt             │
│    (embedded + user slots)          │
│ 2. Run Claude session               │
│ 3. Validate outputs:                │
│    ├─ feature_list.json valid       │
│    ├─ init.sh exists + runs         │
│    ├─ git initialized               │
│    └─ claude-progress.md exists     │
│ 4. If validation fails → retry      │
│    with error context               │
└─────────────────────────────────────┘
    │ (validation passes)
    ▼
┌─────────────────────────────────────┐
│ Phase: Implementation (loop)        │
│                                     │
│ 1. Run init.sh                      │
│ 2. Assemble impl prompt             │
│    (embedded + user slots)          │
│ 3. Run Claude session               │
│ 4. Check completion                 │
│    └─ All features pass? → Done     │
│    └─ Features remain? → Continue   │
└─────────────────────────────────────┘
```

---

## Implementation Roadmap

### Phase 1: Prompt Improvements (No Harness Code)

Update default prompts to include:

- Regression testing step
- Browser/E2E verification emphasis
- Quality standards section
- Strong immutability language

**Effort**: Low
**Breaking changes**: None

### Phase 2: Naming Alignment

- Rename `tasks.json` → `feature_list.json`
- Rename `progress.md` → `claude-progress.md`
- Rename `spec.md` → `app_spec.md`
- Update all prompts to use "features" not "tasks"

**Effort**: Low
**Breaking changes**: Migration command needed

### Phase 3: init.sh Support

- Check for `.lorah/init.sh` before implementation phase
- Execute if present
- Handle errors gracefully
- Update init prompt to require init.sh creation

**Effort**: Low-Medium
**Breaking changes**: None (only runs if file exists)

### Phase 4: Wrapper + Slot Prompt Assembly

- Embed required sections in binary
- Require user files (verification.md, quality-standards.md)
- Assemble prompts with wrapper + user content
- Validate user files exist and aren't empty

**Effort**: Medium
**Breaking changes**: Directory structure change

### Phase 5: Post-Init Validation

- Implement validation checks
- Gate implementation phase on validation pass
- Retry init with error context on failure

**Effort**: Medium
**Breaking changes**: None (stricter behavior)

### Phase 6: Preset Enhancements

- Add preset-specific verification.md templates
- Add preset-specific quality-standards.md templates
- Update `lorah init --preset` to create new structure

**Effort**: Medium
**Breaking changes**: None

---

## What Lorah Enforces (Non-Negotiable)

1. **Orientation protocol** - pwd, ls, spec, features, git log
2. **Regression testing** - Test passing features before new work
3. **Verification requirement** - Must verify before marking complete
4. **Exit protocol** - Commit all, leave working state
5. **Feature immutability** - Only `passes` field can change
6. **init.sh execution** - Runs before each implementation session
7. **Post-init validation** - Gate to implementation phase

## What Users Control

1. **Project description** - What to build
2. **Verification method** - How to test (browser/CLI/API/etc.)
3. **Quality standards** - Criteria appropriate to project
4. **Implementation guidance** - Project-specific instructions
5. **Feature guidance** - How to structure features for this domain

## What Presets Provide

1. **Default verification.md** - Sensible verification for project type
2. **Default quality-standards.md** - Sensible criteria for project type
