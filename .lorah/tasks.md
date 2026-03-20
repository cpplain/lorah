# Lorah

Implementation plan for lorah, a CLI harness for autonomous coding agents.

## 1. Infrastructure

Prerequisites for TDD: test tooling and build system setup.

- [completed] Add `make test` target to Makefile

  ```notes
  - Run `go test ./...`
  - Add `test` to the `.PHONY` declaration
  - Update the help text
  - Prerequisite for all TDD work that follows
  - Added `test` target running `go test ./...`, added to `.PHONY` and help text
  ```

## 2. CLI Router

Refactor `main.go` from a monolithic program into a thin subcommand router per `cli.md`. Introduces `lorah run <prompt-file>` and `lorah task` subcommands while keeping existing loop logic in place for subsequent features to migrate into `internal/loop/`.

### Routing

- [completed] Write tests for CLI routing

  ```notes
  - Extract a testable `route(args []string, version string, runFn func(string, []string) error, taskFn func([]string) error) int` function
  - Test: `--version`/`-V` prints version to stdout and returns 0
  - Test: `--help`/`-h` prints usage to stderr and returns 0
  - Test: no args prints usage to stderr and returns 1 (per cli.md, distinct from `--help`: `--help` exits 0, no-args exits 1)
  - Test: `run <file>` calls runFn with prompt file and remaining flags
  - Test: `task <subcommand>` calls taskFn
  - Test: unknown subcommand prints error to stderr and returns 1
  - Test in `main_test.go` — test routing logic directly, no subprocess overhead
  - Tests written in `main_test.go`; minimal `route()` stub added to `main.go` for compilation
  - Uses `captureOutput()` helper with `os.Pipe()` to capture stdout/stderr
  - All tests fail as expected (stub returns 0, never calls handlers)
  ```

- [completed] Implement CLI routing refactor

  ```notes
  - Extract `route()` from `main()` matching the signature used in tests
  - Add `run` subcommand wrapping existing loop logic inline
  - Add `task` stub (prints "not yet implemented" to stderr, returns 1)
  - Update help text for `lorah run <prompt-file> [claude-flags...]` and `lorah task <subcommand>` per `cli.md`
  - Keep all existing loop and output logic in `main.go` for now — subsequent features migrate to `internal/loop/`
  - Implemented in `main.go`: `route()`, `runCmd()`, `taskCmd()`, `printUsage()`, `printRunUsage()`, `printTaskUsage()`
  - `main()` now calls `os.Exit(route(os.Args[1:], Version, runCmd, taskCmd))`
  - Fixed tool name formatting bug: `strings.ToUpper(name[:1]) + name[1:]` (not `ToLower` on remainder)
  - All 6 routing tests pass
  ```

## 3. Run Loop

Migrate the infinite loop, subprocess execution, and constants from `main.go` into `internal/loop/` per `run.md`.

### Loop Lifecycle and Signal Handling

- [completed] Write tests for loop lifecycle and signal handling

  ```notes
  - Test in `internal/loop/loop_test.go`
  - Test `Run()` by injecting a fake `runFn` to verify:
    - Iteration counter increments
    - `printSection` called for start and success
    - Loop continues on success
  - Test error handling: on `runFn` error, error printed to stderr, `retryDelay` sleep occurs, loop continues
  - Test stopping flag: loop exits after current iteration when `stopping` is set
  - Test two-signal shutdown:
    - First signal sets `stopping` flag and prints interrupt message
    - Second signal calls `cancel()` and exits immediately
  - Use `os.Signal` channels to simulate `SIGINT` in tests
  - `constants.go` values are trivial — no dedicated test needed
  - Tests written in `internal/loop/loop_test.go` with `captureOutput()` helper
  - Minimal stubs in `internal/loop/loop.go` and `internal/loop/constants.go` for compilation
  - Internal API: `run(ctx, cancel, promptFile, flags, runFn, stopping, delay)` and `handleSignals(sigCh, stopping, cancel, exitFn)`
  - `run()` returns when stopping is set (no os.Exit); `Run()` will call os.Exit after run() returns
  - `handleSignals()` uses `stopping.Swap(true)` to distinguish first vs second signal; takes `exitFn func(int)` instead of `os.Exit` directly for testability
  - All 7 tests fail as expected (stubs are no-ops); no panics or compile errors
  ```

- [completed] Implement loop lifecycle and signal handling

  ```notes
  - Create `internal/loop/constants.go`:
    - ANSI color constants: `colorReset`, `colorGreen`, `colorBlue`, `colorBold`, `colorRed`
    - `maxBufferSize` (1MB)
    - `retryDelay` (5s)
  - Create `internal/loop/loop.go` with `Run(promptFile, claudeFlags)`:
    - Cancellable context
    - Signal handler goroutine: buffered channel, `atomic.Bool stopping`
    - First signal: sets `stopping`, prints interrupt message
    - Second signal: calls `cancel()` and `os.Exit(0)`
    - Infinite loop with iteration counter
    - `printSection` for start and success
    - Error display to stderr with `retryDelay` sleep
    - Stop-after-iteration check per run.md §3
  - Update `main.go` `runCmd` to call `loop.Run()` instead of inlining loop logic
  - Added stub `internal/loop/claude.go` with placeholder `runClaude` (returns "not yet implemented") — needed for `Run()` to compile; full implementation in subsequent task
  - Added stub `internal/loop/output.go` with minimal `printSection` — needed for `run()` and `handleSignals()` to compile; tests and full implementation in subsequent tasks
  - Removed `os/signal` and `syscall` imports from `main.go` (no longer used after inline loop removal)
  - `runClaude` and `printMessages` remain in `main.go` as dead code — will be removed when stream-JSON output task migrates them to `internal/loop/output.go` and `internal/loop/claude.go`
  - All 7 loop tests pass
  ```

### Subprocess Execution

- [completed] Write tests for subprocess execution

  ```notes
  - Test in `internal/loop/claude_test.go`
  - Test `runClaude()` using the `TestHelperProcess` pattern (`os.Args[0]` re-exec):
    - Correct args passed to `claude`: `-p`, `--output-format`, `stream-json`, `--verbose`, plus passthrough flags
    - Prompt file piped to stdin
    - Error returned on non-zero exit
  - Test error cases:
    - Missing prompt file returns error prefixed `"opening prompt file:"`
    - Failed subprocess returns error prefixed `"Claude Code CLI exited with error:"`
  - Added `var execCommandContext = exec.CommandContext` stub to `claude.go` for test overriding
  - `TestHelperProcess` supports env vars: `GO_TEST_HELPER_PROCESS`, `GO_TEST_ARGS_FILE`, `GO_TEST_STDIN_FILE`, `GO_TEST_EXIT_CODE`
  - `helperClaudeFunc(extraEnv...)` is a reusable helper that builds the override function
  - All 4 tests fail as expected (stub returns "not yet implemented"); no panics or compile errors
  ```

- [completed] Implement subprocess execution

  ```notes
  - Create `internal/loop/claude.go` with `runClaude(ctx, promptFile, flags)`:
    - Opens prompt file (error prefix: `"opening prompt file:"`)
    - Builds args: `-p`, `--output-format`, `stream-json`, `--verbose`, `flags...`
    - `exec.CommandContext` with `StdoutPipe`
    - `cmd.Stdin = file`, `cmd.Stderr = os.Stderr`, `cmd.Env = os.Environ()`
    - Calls `printMessages(stdout)`
    - Returns `cmd.Wait()` error prefixed per run.md §4 (`"Claude Code CLI exited with error:"`)
  - Do NOT set `cmd.Env = os.Environ()` — in Go, nil Env inherits parent environment (same behavior),
    and setting it explicitly would overwrite env vars injected by the test helper on the returned cmd.
  - Added stub `printMessages(r io.Reader)` to `output.go` (reads and discards) — needed for
    compilation; full implementation in the subsequent stream-JSON output task.
  - All 4 subprocess tests pass.
  ```

### Stream-JSON Parsing and Output

- [completed] Write tests for stream-JSON parsing and formatted output

  ```notes
  - Test `printSection(label, color, content string)`:
    - ANSI output: colored icon, bold label, reset, trimmed content, trailing blank line
    - Empty content omits content line but still prints trailing blank line
  - Test `printMessages(r io.Reader)` — feed newline-delimited JSON via `strings.NewReader`:
    - `assistant`/`text` → `printSection("Claude", "", text)`
    - `assistant`/`thinking` → `printSection("Claude (thinking)", "", thinking)`
    - `assistant`/`tool_use` for each known tool name → correct label and extracted input key
    - Unknown tool → header only, no content
    - `result` with `is_error=true` → `printSection("Result (error)", colorRed, result)`
    - Non-error `result` → silently skipped
    - Unknown message type → silently skipped
    - Malformed JSON line → silently skipped
    - Multi-line tool input → first line + `"... +N lines"` truncation
  - Capture stdout in tests using `os.Pipe()` or a `bytes.Buffer` passed via writer injection
  - Tests written in `internal/loop/output_test.go`; uses existing `captureOutput()` helper from `loop_test.go`
  - `printSection` tests PASS (stub already has correct implementation)
  - `printMessages` tests FAIL as expected (stub discards all input); no panics or compile errors
  - JSON format for assistant messages: `{"type":"assistant","message":{"content":[...]}}`
  - JSON format for result messages: `{"type":"result","is_error":true,"result":"..."}`
  - Tool input is nested under `"input"` key as a JSON object: `{"command":"ls"}`, `{"file_path":"/foo"}`, etc.
  ```

- [completed] Implement stream-JSON parsing and formatted output

  ```notes
  - Create `internal/loop/output.go`:
    - `printSection(label, color, content string)`: writes `color+icon+reset+" "+bold+label+reset+newline`, then trimmed content (if non-empty), then blank line
    - `printMessages(r io.Reader)`: `bufio.Scanner` with buffer up to `maxBufferSize`; skips empty lines and JSON parse failures
    - Dispatch on `msg["type"]`:
      - `"assistant"` → iterate content blocks, dispatch on `block["type"]` (`text`/`thinking`/`tool_use`)
      - `"result"` → if `is_error`, print error section; otherwise silently skip
      - Others → silently skip
    - Tool display: title-case via `strings.ToUpper(name[:1])+name[1:]` — do NOT use `ToLower` on the remainder (current `main.go` bug breaks `"WebFetch"` → `"Webfetch"`)
    - Extract input key via lookup table: `Bash→command`, `Read/Edit/Write→file_path`, `Grep/Glob→pattern`, `WebFetch→url`, `Task→description`, `Agent→prompt`
    - Truncate multi-line content: first line + `"... +N lines"`
  - Removed equivalent dead code from `main.go`: `printSection`, `runClaude`, `printMessages` functions and their associated constants and imports
  - All 10 output tests pass (2 printSection + 8 printMessages)
  ```

---

## 4. Task Management

Implement the `lorah task` subcommand system per `task.md`. Provides CRUD operations for agent task management with JSON storage backend and multiple output formats. All code lives in `internal/task/` (unexported package). Design spec: `docs/design/task.md`.

### Core Types

- [completed] Write tests for core types (task.go)

  ```notes
  - Test in `internal/task/task_test.go`
  - Test TaskStatus constants: `StatusPending` = `"pending"`, `StatusInProgress` = `"in_progress"`, `StatusCompleted` = `"completed"`
  - Test Phase JSON serialization: `ID` always present; `Name` and `Description` use `omitempty` (absent when empty)
  - Test Section JSON serialization: `ID` and `PhaseID` always present; `Name` and `Description` use `omitempty`
  - Test Task JSON serialization: `ID`, `Subject`, `Status`, `LastUpdated` always present; `PhaseID`, `SectionID`, `Notes` use `omitempty` (absent when empty)
  - Test Task JSON deserialization round-trip: marshal → unmarshal with all fields populated, verify equality
  - Test TaskList JSON serialization: `Tasks`, `Version`, `LastUpdated` always present; `Name`, `Description`, `Phases`, `Sections` use `omitempty`
  - Test generateID(): returns 8-character string, all characters are lowercase hex (`[0-9a-f]`)
  - Test generateID() uniqueness: call multiple times, verify all results are distinct
  - Add minimal type stubs in `internal/task/task.go` for compilation (types with zero-value fields, `generateID()` returning `""`)
  - Expect: type/serialization tests pass against stubs (correct JSON tags); generateID tests fail (stub returns `""`)
  - NOTE: Overzealous prior agent committed full implementations for all Phase 4 files. Added missing
    TestPhaseJSONOmitEmpty and TestSectionJSONOmitEmpty tests; all other tests already existed. The
    full task.go implementation (not stubs) was already present and correct per spec — all tests pass.
  - Next task (Implement core types): implementation already exists and is correct; just verify tests
    pass and mark completed.
  ```

- [completed] Implement core types (task.go)

  ```notes
  - Create `internal/task/task.go`
  - Phase struct: `ID string "json:\"id\""`, `Name string "json:\"name,omitempty\""`, `Description string "json:\"description,omitempty\""`
  - Section struct: `ID string "json:\"id\""`, `PhaseID string "json:\"phaseId\""`, `Name string "json:\"name,omitempty\""`, `Description string "json:\"description,omitempty\""`
  - Task struct: `ID`, `Subject`, `Status` (TaskStatus), `PhaseID` (omitempty), `SectionID` (omitempty), `Notes` (omitempty), `LastUpdated` (time.Time) — JSON field names are camelCase
  - TaskStatus type: `type TaskStatus string` with three constants
  - TaskList struct: `Name` (omitempty), `Description` (omitempty), `Phases []Phase` (omitempty), `Sections []Section` (omitempty), `Tasks []Task`, `Version string`, `LastUpdated time.Time`
  - Filter struct: `Status []TaskStatus`, `PhaseID string`, `SectionID string`, `Limit int` — used by Storage.List, not serialized to JSON
  - generateID(): use `crypto/rand.Read(4 bytes)` → `hex.EncodeToString` → 8-char lowercase hex string
  - All tests should pass
  - Implementation was already present from prior overzealous agent; verified all tests pass against spec.
  ```

### Storage Core

- [completed] Write tests for storage core (json_storage.go)

  ```notes
  - Test in `internal/task/json_storage_test.go`
  - Use `t.TempDir()` for isolated `tasks.json` file in each test
  - Test Load — non-existent file: returns empty `TaskList` with `Version: "1.0"` and empty `Tasks` slice (not an error)
  - Test Load — existing file: write a JSON file manually, verify `Load()` deserializes all fields correctly (name, description, phases, sections, tasks, version, lastUpdated)
  - Test Save: sets `TaskList.LastUpdated` to current time before writing; file is indented JSON (`json.MarshalIndent`); round-trips with `Load()`
  - Test Save: file written with 0644 permissions
  - Add Storage interface in `internal/task/storage.go` and JSONStorage struct stub in `internal/task/json_storage.go` for compilation
  - JSONStorage stub: `Load()` returns `&TaskList{}` (not nil); all other methods return errors
  - All tests should fail as expected (stubs are no-ops); no panics
  - NOTE: Prior overzealous agent already wrote full implementation in json_storage.go and all test cases
    for CRUD, list/filter in json_storage_test.go. Added missing test coverage: expanded
    TestLoadExistingFile to verify all fields (description, phases, sections, version, lastUpdated),
    and added TestSaveFilePermissions (0644). Full implementation was already present and correct.
    All tests pass.
  ```

- [completed] Implement storage core (storage.go + json_storage.go)

  ```notes
  - Create `internal/task/storage.go` with Storage interface:
    - `Load() (*TaskList, error)`
    - `Save(list *TaskList) error`
    - `Get(id string) (*Task, error)`
    - `List(filter Filter) ([]Task, error)`
    - `Create(task *Task) error`
    - `Update(task *Task) error`
    - `Delete(id string) error`
  - Create `internal/task/json_storage.go` with JSONStorage struct:
    - Fields: `mu sync.RWMutex`, `path string`
    - Constructor: `NewJSONStorage(path string) *JSONStorage`
  - Load: acquire read lock; read file; if `os.IsNotExist`, return `&TaskList{Version: "1.0", Tasks: []Task{}}` (not an error); unmarshal JSON
  - Save: acquire write lock; set `list.LastUpdated = time.Now()`; `json.MarshalIndent(list, "", "  ")`; write to file with 0644 permissions
  - Stub remaining methods (Get/List/Create/Update/Delete return errors) to satisfy the interface
  - All tests should pass
  - NOTE: Prior overzealous agent already implemented everything in storage.go and json_storage.go
    including full CRUD and List. Verified all tests pass, marked completed.
  ```

### Storage Create, Get, Update, Delete

- [completed] Write tests for storage create, get, update, delete (json_storage.go)

  ```notes
  - Continue in `internal/task/json_storage_test.go`
  - Use `t.TempDir()` for isolated `tasks.json` file in each test
  - Test Create: auto-sets `task.LastUpdated` to current time; task appears in subsequent `Load()`
  - Test Create — duplicate ID: returns an error (does not overwrite)
  - Test Get — found: returns correct task by ID
  - Test Get — not found: returns an error
  - Test Update — found: modifies fields; sets `LastUpdated` to current time; preserves unmodified fields
  - Test Update — not found: returns an error
  - Test Delete — found: removes task from list; subsequent `Get()` returns not found
  - Test Delete — not found: returns an error
  - Stubs from previous section already exist; all tests should fail as expected (stubs return errors); no panics
  - NOTE: Overzealous prior agent already wrote all tests (TestCreate, TestCreateDuplicateID, TestGet,
    TestGetNotFound, TestUpdate, TestUpdateNotFound, TestDelete, TestDeleteNotFound) and the full
    implementation. All tests pass. Same pattern as previous tasks.
  ```

- [completed] Implement storage create, get, update, delete (json_storage.go)

  ```notes
  - Add to `internal/task/json_storage.go`
  - Create: call Load; scan for duplicate ID (return error if found); set `task.LastUpdated = time.Now()`; append to `list.Tasks`; call Save
  - Get: call Load; linear scan by ID; return error `"task not found: <id>"` if not found
  - Update: call Load; find task by `task.ID`; replace in slice; set `task.LastUpdated = time.Now()`; call Save; error if not found
  - Delete: call Load; find by ID; remove from slice; call Save; error if not found
  - Note: Create/Update/Delete call Load then Save — no transaction; acceptable per spec (no multi-agent coordination required)
  - Stub `List` (returns error) to satisfy the interface; implemented in subsequent task
  - All tests should pass
  - NOTE: Overzealous prior agent already wrote the full implementation. All tests pass. Verified and marked completed.
  ```

### Storage List & Filter

- [completed] Write tests for storage list and filter (json_storage.go)

  ```notes
  - Continue in `internal/task/json_storage_test.go`
  - Use `t.TempDir()` for isolated `tasks.json` file in each test
  - Test List — no filter: returns all tasks
  - Test List — status filter: single status returns only matching tasks
  - Test List — status filter: multiple statuses (OR within the status list)
  - Test List — PhaseID filter: only tasks matching the phase
  - Test List — SectionID filter: only tasks matching the section
  - Test List — combined filters: status + PhaseID (AND-combined)
  - Test List — Limit: `Limit=0` means no limit; `Limit=N` returns at most N tasks
  - Stub from previous section already exists (returns error); all tests should fail as expected; no panics
  - Prior agent had already written tests for single status, PhaseID, SectionID, Limit, and NoLimit.
    Added the three missing cases: TestListNoFilter, TestListFilterByMultipleStatuses, TestListFilterCombined.
  - Full List implementation was already present; all tests pass.
  ```

- [completed] Implement storage list and filter (json_storage.go)

  ```notes
  - Add to `internal/task/json_storage.go`
  - List: call Load; iterate tasks; apply filters (AND-combined): status matches any in `filter.Status` (OR within the status list); PhaseID exact match; SectionID exact match; empty filter fields are ignored; apply Limit after filtering (0 = no limit)
  - All tests should pass
  - Prior overzealous agent had already written the full List implementation. Verified all tests pass and marked completed.
  ```

### Single-Task Formatters

- [completed] Write tests for single-task formatters (format.go)

  ```notes
  - Test in `internal/task/format_test.go`
  - Use fixed `time.Time` values for deterministic output (e.g. `time.Date(2026, 3, 10, 14, 22, 0, 0, time.UTC)`)
  - Create a helper TaskList fixture with phases and sections for name resolution tests

  **FormatTaskMarkdown:**
  - Test H1 heading is the task subject
  - Test status is always shown: `**Status:** <status>`
  - Test lastUpdated is always shown: `**Updated:** <ISO8601>`
  - Test phase line: when phaseId is set and phase has a name, render `**Phase:** <name>`; when no name in phases list, render `**Phase:** <hex-id>`; when phaseId is empty, omit line entirely
  - Test section line: same name-resolution rules as phase; omit when sectionId is empty
  - Test notes non-empty: rendered as-is below field list, separated by blank line
  - Test notes empty: render `**Notes:** (none)`

  **FormatTaskJSON:**
  - Test outputs single task object (not wrapped in `{"tasks": [...]}` envelope)
  - Test all fields present including omitempty fields when populated
  - Test valid JSON via `json.Unmarshal` round-trip

  - Add function stubs `FormatTaskMarkdown` and `FormatTaskJSON` in `internal/task/format.go` returning zero values for compilation
  - All tests should fail as expected (stubs return empty string/nil)
  - NOTE: Overzealous prior agent already wrote tests for all formatters (single-task, list, export)
    and the full format.go implementation. Verified all tests match spec requirements and pass.
    Tests in `internal/task/format_test.go`: TestFormatTaskMarkdown (4 subtests covering all spec
    cases), TestFormatTaskJSON (valid JSON + no envelope). All tests pass.
  ```

- [completed] Implement single-task formatters (format.go)

  ```notes
  - Create `internal/task/format.go`
  - Function signatures:
    - `FormatTaskMarkdown(task *Task, list *TaskList) string`
    - `FormatTaskJSON(task *Task) (string, error)`

  **FormatTaskMarkdown:**
  - H1 = subject; always show `**Status:**` and `**Updated:**` (ISO8601)
  - Phase/section: look up name in `list.Phases`/`list.Sections` by ID; use name if found, hex ID fallback; omit line if phaseId/sectionId empty
  - Notes: if non-empty, blank line + notes content; if empty, `**Notes:** (none)`

  **FormatTaskJSON:**
  - `json.MarshalIndent(task, "", "  ")` — single object, no envelope

  - All tests should pass
  - NOTE: Full implementation already exists in `internal/task/format.go` from overzealous prior agent,
    including FormatTaskMarkdown, FormatTaskJSON, FormatListMarkdown, FormatListJSON,
    FormatExportMarkdown, and shared helpers. All tests pass. Verified all tests pass; marked completed.
  ```

### List Grouped Formatter

- [completed] Write tests for list grouped formatter (format.go)

  ```notes
  - Continue in `internal/task/format_test.go`

  **FormatListMarkdown (grouped mode):**
  - Test grouped output: phase H2 headings, section H3 headings, task bullets
  - Test bullet format: `- \`<id>\` [<status>] <subject>`
  - Test notes rendering: non-empty notes → blank line after bullet, then fenced code block with `notes` info string, indented 2 spaces (opening fence, content lines, closing fence)
  - Test tasks without notes render as bare bullets (no extra lines)
  - Test phases/sections with zero matching tasks are omitted
  - Test phase name resolution: name if available, hex ID fallback
  - Test section name resolution: same rules
  - Test tasks with no phase collected under `## (none)`
  - Test tasks with no section appear directly under phase heading (no H3)
  - Test NO project name/description H1 in list output (only in export)
  - Test ordering: tasks ordered by phase position (order in `phases` array), then section position (order in `sections` array), then task ID

  - Add stubs `FormatListMarkdown` and `FormatListJSON` to `internal/task/format.go` for compilation (both return empty string/nil)
  - All tests should fail as expected
  - NOTE: Overzealous prior agent already wrote TestFormatListMarkdown (2 subtests), TestFormatListMarkdownFlat, TestFormatListJSON, TestFormatExportMarkdown, and full implementations. Added 6 missing subtests to TestFormatListMarkdown: phase name hex fallback, section name hex fallback, tasks with no phase under (none), tasks without section directly under phase heading, tasks without notes as bare bullets, and ordering by phase/section position. All tests pass.
  ```

- [completed] Implement list grouped formatter (format.go)

  ```notes
  - Add to `internal/task/format.go`
  - Function signature: `FormatListMarkdown(tasks []Task, list *TaskList, flat bool) string`

  **Shared grouping helper (e.g. `renderGrouped`):**
  - Group tasks by phaseId then sectionId
  - Order phases by position in `list.Phases` array; orphan phases (tasks referencing phaseIds not in list) appended at end
  - Order sections by position in `list.Sections` array within each phase
  - Tasks with no phase → group key `""` → heading `## (none)`
  - Tasks with no section → appear directly under phase heading, no H3
  - Phase heading: `## {name}` if name exists, else `## {id}`
  - Section heading: `### {name}` if name exists, else `### {id}`
  - Task bullet: `` - `{id}` [{status}] {subject} ``
  - Notes rendering: blank line after bullet, then fenced block with `notes` info string indented 2 spaces (keeps code block inside list item)
  - `includeDescriptions bool` parameter: when false (list), skip phase/section description paragraphs; when true (export), render them

  **FormatListMarkdown:**
  - When flat=true: return `""` (stub; implemented in next section)
  - When flat=false: call shared grouping helper with includeDescriptions=false (no project H1, no phase/section descriptions)

  - All tests should pass
  - NOTE: Implementation was already present from overzealous prior agent in `internal/task/format.go`.
    Full `renderGrouped` helper, `renderFlatList`, `FormatListMarkdown`, `FormatListJSON`,
    `FormatExportMarkdown` all implemented and passing all tests. Verified and marked completed.
  ```

### List Flat and JSON Formatters

- [completed] Write tests for list flat and JSON formatters (format.go)

  ```notes
  - Continue in `internal/task/format_test.go`

  **FormatListMarkdown flat mode (`--flat`):**
  - Test suppresses phase and section headings
  - Test omits notes
  - Test output is flat bullet list: `- \`<id>\` [<status>] <subject>`

  **FormatListJSON:**
  - Test `{"tasks": [...]}` envelope wrapping task objects
  - Test valid JSON round-trip

  - Stubs already exist from previous section; all tests should fail as expected
  - NOTE: Overzealous prior agent already wrote both TestFormatListMarkdownFlat and
    TestFormatListJSON tests, and the full implementations. All spec requirements are
    covered. All tests pass.
  ```

- [completed] Implement list flat and JSON formatters (format.go)

  ```notes
  - Add to `internal/task/format.go`
  - Function signature: `FormatListJSON(tasks []Task) (string, error)`

  **FormatListMarkdown flat path:**
  - When flat=true: iterate tasks, emit bare bullets only (no headings, no notes)
  - Replace the `""` stub from the previous section

  **FormatListJSON:**
  - Wrap tasks in `{"tasks": [...]}` envelope; `json.MarshalIndent`

  - All tests should pass
  - NOTE: Overzealous prior agent already wrote full implementations of `renderFlatList` and
    `FormatListJSON` in `internal/task/format.go`. All tests pass. Verified and marked completed.
  ```

### Export Formatter

- [completed] Write tests for export formatter (format.go)

  ```notes
  - Continue in `internal/task/format_test.go`

  **FormatExportMarkdown:**
  - Test project name renders as `# {name}` when set
  - Test project description renders as paragraph after H1 when both name and description set
  - Test no H1 when name is not set
  - Test description skipped when name is not set (even if description is set)
  - Test phase descriptions render as paragraphs below phase headings
  - Test section descriptions render as paragraphs below section headings
  - Test no description paragraph when description is empty (no blank placeholder)
  - Test same task bullet and notes format as list markdown (shared rendering logic)

  - Add stub `FormatExportMarkdown` to `internal/task/format.go` for compilation
  - All tests should fail as expected
  - NOTE: Overzealous prior agent already wrote most tests and the full FormatExportMarkdown
    implementation. Added two missing subtests: "section description renders below section heading"
    (FAILS — renderGrouped does not render section descriptions when includePhaseDesc=true; the
    implementation task must fix this) and "no description paragraph when description is empty"
    (PASSES — implementation correctly skips empty phase descriptions and never adds placeholders).
  ```

- [completed] Implement export formatter (format.go)

  ```notes
  - Add to `internal/task/format.go`
  - Function signature: `FormatExportMarkdown(tasks []Task, list *TaskList) string`

  **FormatExportMarkdown:**
  - If `list.Name` is set, render `# {name}` as first line
  - If `list.Description` is set AND name is set, render as paragraph after H1 followed by blank line
  - If name is not set: skip both (no H1, no description)
  - Call shared grouping helper (from list grouped formatter) with includeDescriptions=true to render phase/section descriptions

  - Fixed: added section description rendering in `renderGrouped` after `### {section heading}` line
    when `includePhaseDesc && sec.Description != ""`. The prior overzealous agent had only implemented
    phase description rendering. One-line fix in `renderGrouped` in `format.go`.
  - All tests pass.
  ```

### Dispatch

- [completed] Write tests for dispatch (cmd.go)

  ```notes
  - Test in `internal/task/cmd_test.go`
  - Define a `mockStorage` struct implementing the `Storage` interface for test isolation (no file I/O)
  - mockStorage: in-memory `TaskList` with configurable phases/sections/tasks; records calls for assertion
  - Function under test: `HandleTask(args []string, w io.Writer, storage Storage) int` — returns exit code
  - Capture stdout via the `w io.Writer` parameter; capture stderr separately where needed

  **Dispatch:**
  - Test unknown subcommand: prints error to stderr, returns 1
  - Test no subcommand: prints task usage to stderr, returns 1
  - Test `--help`/`-h`: prints task usage to stderr, returns 0
  - Test routes to correct handler for list, get, create, update, delete, export

  **multiFlag type:**
  - Test `String()` returns comma-joined values
  - Test `Set()` appends to the slice (supports repeatable invocation)

  - Add `HandleTask` stub returning 1 and `multiFlag` type stub in `internal/task/cmd.go` for compilation
  - All tests should fail as expected (stub returns 1); no panics
  - NOTE: Overzealous prior agent had already written the mockStorage, multiFlag tests, dispatch tests
    (unknown/no-args), and full subcommand tests for list/get/create/update/export + full cmd.go
    implementation. Added missing tests: TestHandleTaskHelp (--help/-h returns 0, FAILS — no
    implementation), TestDeleteByID/TestDeleteCmdNotFound/TestDeleteMissingID (FAIL/PASS/PASS — stub
    returns 1). Added deleteCmd stub and "delete" case to switch in cmd.go for compilation.
  - Next task (Implement dispatch): must add --help/-h handling to HandleTask and implement deleteCmd
  ```

- [completed] Implement dispatch (cmd.go)

  ```notes
  - Create `internal/task/cmd.go`
  - `HandleTask(args []string, w io.Writer, storage Storage) int` — main entry point
  - `multiFlag` type: `type multiFlag []string` implementing `flag.Value`:
    - `String()`: `strings.Join(f, ",")`
    - `Set(value string)`: append to slice

  **Dispatch logic:**
  - No args: print task usage to stderr, return 1
  - `--help`/`-help`/`-h` as first arg: print task usage to stderr, return 0
  - Switch on `args[0]`: dispatch to list/get/create/update/delete/export handlers
  - Unknown subcommand: print error + task usage to stderr, return 1

  - Add stub handlers for list/get/create/update/delete/export (return 1) to satisfy the switch; implement in subsequent tasks
  - All tests should pass
  - Prior overzealous agent had implemented all handlers except deleteCmd (stub) and --help handling.
  - Added --help/-help/-h case to HandleTask switch (returns 0).
  - Implemented deleteCmd: extracts ID from args[0], returns 1 with usage if missing, calls storage.Delete(id), returns 1 with error if not found, returns 0.
  - All tests pass.
  ```

### List Subcommand

- [completed] Write tests for list handler (cmd.go)

  ```notes
  - Continue in `internal/task/cmd_test.go`

  **list subcommand:**
  - Test default format is markdown
  - Test `--status=pending` filters to matching tasks
  - Test `--status` is repeatable: `--status=pending --status=in_progress` passes both statuses
  - Test `--phase=<hex-id>` passes PhaseID filter to storage
  - Test `--section=<hex-id>` passes SectionID filter to storage
  - Test `--limit=5` passes Limit filter to storage
  - Test `--flat` passes flat=true to FormatListMarkdown
  - Test `--format=json` outputs JSON envelope
  - Test `--flat` with `--format=json`: flat flag is ignored, output is JSON envelope (--flat only applies to markdown)
  - Test invalid `--status` value: returns 1 with error message

  - Stub for list handler already exists (returns 1); all tests should fail as expected
  - NOTE: Overzealous prior agent had already written most list tests and the full listCmd implementation.
    Added 3 missing tests: TestListSectionFilter (--section filter), TestListFlatWithJSON (--flat ignored
    with --format=json), TestListInvalidStatus (invalid status returns 1). All tests pass.
  - Next task (Implement list handler): implementation already exists and is correct; just verify tests
    pass and mark completed.
  ```

- [completed] Implement list handler (cmd.go)

  ```notes
  - Add to `internal/task/cmd.go`

  **list handler:**
  - `flag.NewFlagSet("lorah task list", flag.ContinueOnError)`
  - Flags: `--status` (multiFlag), `--phase` (string), `--section` (string), `--limit` (int, default 0), `--flat` (bool), `--format` (string, default `"markdown"`)
  - Validate each status value against TaskStatus constants; return 1 on invalid
  - Build Filter; call `storage.List(filter)`
  - Call `storage.Load()` to get full TaskList for name resolution in markdown output
  - Format: `--format=json` → FormatListJSON; default → FormatListMarkdown(tasks, list, flat)
  - Write to `w`; return 0

  - All tests should pass
  - NOTE: Full implementation already existed from overzealous prior agent in cmd.go. Verified all tests pass; marked completed.
  ```

### Get Subcommand

- [completed] Write tests for get handler (cmd.go)

  ```notes
  - Continue in `internal/task/cmd_test.go`

  **get subcommand:**
  - Test retrieves task by ID (first positional arg)
  - Test default format is markdown; output includes subject as H1
  - Test `--format=json` outputs single task JSON object (no envelope)
  - Test task not found: returns 1 with error message
  - Test no ID argument: returns 1 with usage

  - Stub for get handler already exists (returns 1); all tests should fail as expected
  - NOTE: Overzealous prior agent had already written all 4 get tests (TestGetByID,
    TestGetCmdNotFound, TestGetMissingID, TestGetJSONFormat) and the full getCmd
    implementation. Strengthened TestGetByID to verify subject appears as `# Subject`
    H1 (was only checking `strings.Contains(out, "Test get")`). All tests pass.
  - Next task (Implement get handler): implementation already exists and is correct;
    just verify tests pass and mark completed.
  ```

- [completed] Implement get handler (cmd.go)

  ```notes
  - Add to `internal/task/cmd.go`

  **get handler:**
  - `flag.NewFlagSet("lorah task get", flag.ContinueOnError)`; extract ID from `fs.Arg(0)`
  - Return 1 with usage if no ID provided
  - Call `storage.Get(id)` — return 1 with error message if not found
  - Call `storage.Load()` for name resolution
  - Format: `--format=json` → FormatTaskJSON; default → FormatTaskMarkdown(task, list)
  - Write to `w`; return 0

  - All tests should pass
  - NOTE: Implementation was already present from the overzealous prior agent in cmd.go. Verified all
    tests pass; marked completed.
  ```

### Create Subcommand

- [completed] Write tests for create handler (cmd.go)

  ```notes
  - Continue in `internal/task/cmd_test.go`

  **create subcommand — basic creation:**
  - Test `--subject` is required: returns 1 if missing
  - Test creates task with auto-generated ID; prints `task <hex-id>` to stdout
  - Test default status is `pending`
  - Test `--status=in_progress` sets status on created task
  - Test invalid `--status` value: returns 1 with error message
  - Test `--phase=<hex-id>` assigns task to existing phase (no new phase created, no "phase" line printed)
  - Test `--section=<hex-id>` assigns task to existing section (no new section created)
  - Test `--section=<hex-id>` without phase context (no `--phase` or `--phase-name`): returns 1 with error

  - Stub for create handler already exists (returns 1); all tests should fail as expected
  - NOTE: Most create tests already existed from the overzealous prior agent (TestCreateRequiresSubject,
    TestCreateBasic, TestCreateWithExistingPhase, TestCreateInvalidStatus). Added 3 missing tests:
    TestCreateWithStatus (PASSES — implementation already validates status), TestCreateWithExistingSection
    (PASSES — implementation sets sectionID correctly), TestCreateSectionWithoutPhaseContext (FAILS —
    implementation doesn't validate that --section requires a phase context per spec).
  - Next task (Implement create handler): must add validation that --section without phase context returns 1.
  ```

- [completed] Implement create handler (cmd.go)

  ```notes
  - Add to `internal/task/cmd.go`

  **create handler (basic creation):**
  - `flag.NewFlagSet("lorah task create", flag.ContinueOnError)`
  - Flags: `--subject` (string), `--status` (string, default `"pending"`), `--phase`, `--phase-name`, `--phase-description`, `--section`, `--section-name`, `--section-description`, `--project-name`, `--project-description`
  - Validate: `--subject` non-empty (return 1); `--status` valid TaskStatus (return 1)
  - `--phase` provided: set phaseId directly
  - `--section` provided: validate phase context exists (`--phase` or `--phase-name`), return 1 with error if missing; set sectionId directly
  - Build Task with `generateID()`, subject, status (as TaskStatus), phaseId, sectionId
  - Call `storage.Create(&task)`
  - Print `task <id>` to stdout; return 0
  - Auto-generation flags (`--phase-name`, `--phase-description`, `--section-name`, `--section-description`, `--project-name`, `--project-description`) parsed but not yet handled (implemented in next section)

  - All tests should pass
  - Prior overzealous agent had already written most of the implementation (auto-generation included).
    Only missing validation: `--section` without phase context (`--phase` or `--phase-name`) must return 1.
    Added guard before `sectionID = *section` in createCmd. All tests pass.
  ```

### Create Auto-Generation

- [completed] Write tests for create auto-generation (cmd.go)

  ```notes
  - Continue in `internal/task/cmd_test.go`

  **create subcommand — auto-generation:**
  - Test `--phase-name="Phase 1"` without `--phase`: auto-generates new phase ID, creates Phase entry, prints `phase <hex-id>` before `task <hex-id>`
  - Test `--phase-description="..."`: sets description on auto-generated phase
  - Test `--section-name="1.1 Foo"` without `--section`: auto-generates section ID, creates Section entry with correct PhaseID, prints `section <hex-id>` between phase and task lines
  - Test `--section-description="..."`: sets description on auto-generated section
  - Test `--project-name` sets TaskList.Name
  - Test `--project-description` sets TaskList.Description
  - Test output ordering: `phase <id>` before `section <id>` before `task <id>` (only newly created entities)
  - Test `--section-name` without phase context (no `--phase` or `--phase-name`): returns 1 with error

  - Several tests already existed from prior iterations (TestCreateWithPhaseName, TestCreateWithPhaseAndSectionName, TestCreateWithProjectNameAndDescription).
  - Added 5 new tests: TestCreatePhaseDescription, TestCreateSectionDescription, TestCreateSectionPhaseIDAssigned, TestCreateOutputOrdering, TestCreateSectionNameWithoutPhaseContext.
  - TestCreateSectionNameWithoutPhaseContext FAILS as expected — implementation guards `--section` without phase context but not `--section-name` without phase context.
  - All other new tests PASS against the existing implementation.
  ```

- [completed] Implement create auto-generation (cmd.go)

  ```notes
  - Add to the create handler in `internal/task/cmd.go`

  **Phase auto-generation:**
  - `--phase-name` or `--phase-description` without `--phase`: auto-generate phase ID, create Phase entry, append to list.Phases, record as "newly created" → print `phase <id>`
  - `--phase` provided with `--phase-name` or `--phase-description`: upsert on the existing phase in list.Phases

  **Section auto-generation:**
  - `--section-name` or `--section-description` without `--section`: auto-generate section ID, create Section with PhaseID, append to list.Sections, record as "newly created" → print `section <id>`
  - `--section-name` without any phase context: return 1 with error

  **Project metadata:**
  - Apply `--project-name`/`--project-description` to list

  **Storage:**
  - Call `storage.Save(list)` to persist phase/section/project metadata before `storage.Create(&task)`
  - Print: `phase <id>` (if new), `section <id>` (if new), `task <id>` (always)

  - Added guard in `else if *sectionName != "" || *sectionDesc != ""` branch: checks `phaseID == ""`
    and returns 1 with error "—section-name requires a phase context (--phase or --phase-name)".
  - All other auto-generation logic was already implemented by prior overzealous agent.
  - All tests pass.
  ```

### Update Basic Fields

- [completed] Write tests for update basic fields (cmd.go)

  ```notes
  - Continue in `internal/task/cmd_test.go`

  **update subcommand — basic fields:**
  - Test partial update via `fs.Visit`: only explicitly provided flags modify fields (omitted flags leave fields unchanged)
  - Test `--status=completed` updates status
  - Test `--subject="new"` updates subject; `--subject=""` clears subject (not treated as "not provided")
  - Test `--notes="..."` replaces notes field
  - Test task not found: returns 1 with error message
  - Test no ID argument: returns 1 with usage
  - Test invalid `--status` value: returns 1 with error message

  - Stub for update handler already exists (returns 1); all tests should fail as expected
  - NOTE: Overzealous prior agent had already written most update tests (TestUpdateStatus,
    TestUpdateNotes, TestUpdateCmdNotFound, TestUpdateMissingID, TestUpdatePartialPreservesFields,
    TestUpdateInvalidStatus, TestUpdateSetsLastUpdated) and the full updateCmd implementation.
    Added 2 missing tests: TestUpdateSubject (--subject updates subject) and TestUpdateSubjectClear
    (--subject="" clears subject, not treated as "not provided"). All tests pass.
  - Next task (Implement update basic fields): implementation already exists and is correct;
    just verify tests pass and mark completed.
  ```

- [completed] Implement update basic fields (cmd.go)

  ```notes
  - Add to `internal/task/cmd.go`

  **update handler (basic fields):**
  - `flag.NewFlagSet("lorah task update", flag.ContinueOnError)`
  - Extract ID from `fs.Arg(0)`; return 1 with usage if missing
  - Flags: `--status`, `--subject`, `--phase`, `--phase-name`, `--phase-description`, `--section`, `--section-name`, `--section-description`, `--notes`, `--project-name`, `--project-description`
  - Use `fs.Visit` after `fs.Parse` to build `provided map[string]bool` — only update fields where `provided["flag-name"]` is true
  - Load task via `storage.Get(id)` — return 1 if not found
  - Apply only provided basic fields to task: status (validate), subject, phaseId, sectionId, notes
  - Call `storage.Update(&task)`; return 0
  - Metadata flags (`--phase-name`, `--phase-description`, `--section-name`, `--section-description`, `--project-name`, `--project-description`) parsed but not yet handled (implemented in next section)

  - All tests should pass
  - NOTE: Prior overzealous agent already wrote the full updateCmd implementation including metadata
    handling (phase-name, phase-description, section-name, section-description, project-name,
    project-description). All tests pass. Verified and marked completed.
  - NOTE for next task (Write tests for update metadata): implementation already exists and handles
    all metadata mutations; tests must match existing behavior. See updateCmd in cmd.go lines 306-430.
  ```

### Update Metadata

- [completed] Write tests for update metadata (cmd.go)

  ```notes
  - Continue in `internal/task/cmd_test.go`

  **update subcommand — metadata:**
  - Test `--phase=<hex-id>` reassigns phase on task
  - Test `--phase-name="..."` upserts name on phase referenced by `--phase` (requires `--phase`)
  - Test `--phase-description="..."` upserts description on phase (requires `--phase`)
  - Test `--section=<hex-id>` reassigns section (requires `--phase`)
  - Test `--section-name="..."` upserts name on section referenced by `--section` (requires `--section`)
  - Test `--section-description="..."` upserts description on section (requires `--section`)
  - Test `--project-name` / `--project-description` set TaskList metadata
  - Test `--phase-name` without `--phase`: returns 1 with error
  - Test `--phase-description` without `--phase`: returns 1 with error
  - Test `--section` without `--phase`: returns 1 with error
  - Test `--section-name` without `--section`: returns 1 with error
  - Test `--section-description` without `--section`: returns 1 with error

  - Stubs from previous section already exist; all tests should fail as expected
  - Added 12 tests: TestUpdatePhaseReassigns, TestUpdatePhaseNameUpserts, TestUpdatePhaseDescriptionUpserts,
    TestUpdateSectionReassigns, TestUpdateSectionNameUpserts, TestUpdateSectionDescriptionUpserts,
    TestUpdateProjectMetadata (all PASS), and TestUpdatePhaseNameRequiresPhase,
    TestUpdatePhaseDescriptionRequiresPhase, TestUpdateSectionRequiresPhase,
    TestUpdateSectionNameRequiresSection, TestUpdateSectionDescriptionRequiresSection (all FAIL — impl
    silently skips instead of returning 1).
  - Next task (Implement update metadata): must add validation guards before the metadata upsert logic
    in updateCmd so that --phase-name/--phase-description without --phase, --section without --phase,
    and --section-name/--section-description without --section all return 1 with an error message.
  ```

- [pending] Implement update metadata (cmd.go)

  ```notes
  - Add to the update handler in `internal/task/cmd.go`

  **Metadata handling:**
  - Load TaskList via `storage.Load()` for metadata operations
  - `--phase-name`/`--phase-description`: require `--phase` (return 1 if not provided); upsert on list.Phases entry matching the phase ID
  - `--section`/`--section-name`/`--section-description`: require `--phase` (return 1 if not provided); `--section-name`/`--section-description` additionally require `--section`; upsert on list.Sections entry
  - Apply project metadata (`--project-name`, `--project-description`) to list if provided
  - Call `storage.Save(list)` after applying metadata changes

  - All tests should pass
  ```

### Delete Subcommand

- [pending] Write tests for delete handler (cmd.go)

  ```notes
  - Continue in `internal/task/cmd_test.go`

  **delete subcommand:**
  - Test deletes task by ID; returns 0 with no output
  - Test task not found: returns 1 with error message
  - Test no ID argument: returns 1 with usage

  - Stub for delete handler already exists (returns 1); all tests should fail as expected
  ```

- [pending] Implement delete handler (cmd.go)

  ```notes
  - Add to `internal/task/cmd.go`

  **delete handler:**
  - `flag.NewFlagSet("lorah task delete", flag.ContinueOnError)`
  - Extract ID from `fs.Arg(0)`; return 1 with usage if missing
  - Call `storage.Delete(id)` — return 1 with error message if not found
  - Return 0

  - All tests should pass
  ```

### Export Subcommand

- [pending] Write tests for export handler (cmd.go)

  ```notes
  - Continue in `internal/task/cmd_test.go`

  **export subcommand:**
  - Test default: outputs export markdown to `w` (stdout)
  - Test `--output=<file>`: writes to specified file path instead of `w`
  - Test `--status=pending` filter: only matching tasks included
  - Test `--status` is repeatable
  - Test output is export markdown format (includes project H1 if name is set, phase/section descriptions)

  - Stub for export handler already exists (returns 1); all tests should fail as expected
  ```

- [pending] Implement export handler (cmd.go)

  ```notes
  - Add to `internal/task/cmd.go`

  **export handler:**
  - `flag.NewFlagSet("lorah task export", flag.ContinueOnError)`
  - Flags: `--output` (string), `--status` (multiFlag)
  - Build Filter from status flags; call `storage.List(filter)` and `storage.Load()`
  - Call `FormatExportMarkdown(tasks, list)`
  - If `--output` provided: write to file; else: write to `w`
  - Return 0

  - All tests should pass
  ```

### Wire-Up

- [pending] Wire up task command to main.go

  ```notes
  - Update `taskCmd` in `main.go`:
    - Create `task.NewJSONStorage("tasks.json")` pointing to tasks.json in the working directory
    - Call `task.HandleTask(args, os.Stdout, storage)` and propagate non-zero exit code
    - Import `internal/task` package
  - No new tests needed — existing routing tests in `main_test.go` cover the dispatch path
  - Verify: `go build` succeeds; `lorah task --help` prints usage; `lorah task list` runs against empty state without error
  ```
