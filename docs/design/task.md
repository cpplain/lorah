# Task Command Specification

---

## 1. Overview

### Purpose

The `task` command provides CRUD operations for agent task management. Tasks are
stored in a JSON file and queryable via CLI subcommands. Agents use task commands
to read, claim, and complete work items efficiently without reading large markdown files.

### Goals

- **Token-efficient**: agents read only actionable tasks, not entire files
- **Structured storage**: JSON backend with a defined schema
- **Agent learning**: completion notes enable cross-iteration knowledge transfer
- **Multiple output formats**: `markdown` (default) for agents and humans, `json` for scripting and non-LLM tooling
- **Storage abstraction**: interface-based backend — JSON now, SQLite in the future

### Non-Goals

- SQLite backend (future work, after JSON proves out in practice)
- Watch mode or live-updating markdown files
- Multi-agent coordination or file locking
- Migration tooling

---

## 2. Interface

### CLI

```
lorah task <subcommand> [args...] [flags...]
```

### Subcommands

| Subcommand | Arguments | Description                       |
| ---------- | --------- | --------------------------------- |
| `list`     |           | List tasks with optional filters  |
| `get`      | `<id>`    | Get full details for a task       |
| `create`   |           | Create a new task                 |
| `update`   | `<id>`    | Update fields on an existing task |
| `delete`   | `<id>`    | Delete a task                     |
| `export`   |           | Export tasks to markdown          |

---

## 3. Task Schema

### Phase and Section Types

```go
type Phase struct {
    ID          string `json:"id"`
    Name        string `json:"name,omitempty"`
    Description string `json:"description,omitempty"`
}

type Section struct {
    ID          string `json:"id"`
    PhaseID     string `json:"phaseId"`
    Name        string `json:"name,omitempty"`
    Description string `json:"description,omitempty"`
}
```

`ID` on `Phase` and `Section` is an auto-generated 8-char lowercase hex string (same format as task IDs). `Name` carries the human-readable label and may include ordinal prefixes for sorting and prioritization (e.g. `"Phase 1: Run Loop"`, `"1.1 Output Formatting"`). `Section.PhaseID` references the parent phase's hex ID.

### Task Type

```go
type Task struct {
    ID          string     `json:"id"`
    Subject     string     `json:"subject"`
    Status      TaskStatus `json:"status"`
    PhaseID     string     `json:"phaseId,omitempty"`
    SectionID   string     `json:"sectionId,omitempty"`
    Notes       string     `json:"notes,omitempty"`
    LastUpdated time.Time  `json:"lastUpdated"`
}
```

### Status Type

```go
type TaskStatus string

const (
    StatusPending    TaskStatus = "pending"
    StatusInProgress TaskStatus = "in_progress"
    StatusCompleted  TaskStatus = "completed"
)
```

### TaskList (File Root)

```go
type TaskList struct {
    Name        string        `json:"name,omitempty"`
    Description string        `json:"description,omitempty"`
    Phases      []Phase       `json:"phases,omitempty"`
    Sections    []Section     `json:"sections,omitempty"`
    Tasks       []Task        `json:"tasks"`
    Version     string        `json:"version"`
    LastUpdated time.Time     `json:"lastUpdated"`
}
```

### Field Descriptions

| Field         | Type    | Purpose                                                   |
| ------------- | ------- | --------------------------------------------------------- |
| `name`        | string  | Project name; rendered as H1 in markdown export           |
| `description` | string  | Project description; rendered below H1 in markdown export |
| `id`          | string  | Unique identifier; auto-generated 8-char lowercase hex    |
| `subject`     | string  | Brief, actionable title in imperative form                |
| `status`      | enum    | `pending`, `in_progress`, `completed`                     |
| `phaseId`     | string  | Hex ID of the parent `Phase` entry                        |
| `sectionId`   | string  | Hex ID of the parent `Section` entry                      |
| `notes`       | string  | Markdown-formatted notes for agent learning               |
| `lastUpdated` | ISO8601 | Timestamp updated automatically on every create or update |

---

## 4. Storage

### Interface

```go
type Storage interface {
    Load() (*TaskList, error)
    Save(list *TaskList) error
    Get(id string) (*Task, error)
    List(filter Filter) ([]Task, error)
    Create(task *Task) error
    Update(task *Task) error
    Delete(id string) error
}
```

### Filter

```go
type Filter struct {
    Status  []TaskStatus
    PhaseID   string
    SectionID string
    Limit   int
}
```

Filters are AND-combined. `Limit = 0` means no limit.

### JSON Backend Behavior

- File: `tasks.json` in the working directory
- Thread safety: `sync.RWMutex` (read lock on `Load`, write lock on `Save`)
- Non-existent file on `Load`: returns an empty `TaskList` with `Version: "1.0"` (not an error)
- Duplicate `id` on `Create`: returns an error
- `Save` updates `LastUpdated` to the current time before writing
- Writes with `json.MarshalIndent` for human readability

### Example File

```json
{
  "name": "Lorah Development Plan",
  "description": "Track progress on the Lorah infinite-loop harness implementation.",
  "phases": [
    {
      "id": "d4e5f6a7",
      "name": "Phase 1: Run Loop",
      "description": "Implement the infinite loop, subprocess execution, and output formatting."
    },
    {
      "id": "e8f9a0b1",
      "name": "Phase 2: Task Management",
      "description": "Implement the task management system for agent workflow coordination."
    }
  ],
  "sections": [
    {
      "id": "b8c9d0e1",
      "phaseId": "d4e5f6a7",
      "name": "1.1 Output Formatting",
      "description": "Stream-JSON parsing and color-coded terminal output."
    },
    {
      "id": "f2a3b4c5",
      "phaseId": "d4e5f6a7",
      "name": "1.2 Signal Handling"
    }
  ],
  "tasks": [
    {
      "id": "a3f7b2c1",
      "subject": "Implement stream-JSON output parsing",
      "status": "completed",
      "phaseId": "d4e5f6a7",
      "sectionId": "b8c9d0e1",
      "notes": "Scans stdout line-by-line as newline-delimited JSON. Skips empty lines and parse failures gracefully.",
      "lastUpdated": "2026-03-10T14:22:00Z"
    },
    {
      "id": "b8e4d1f0",
      "subject": "Add color-coded section headers",
      "status": "in_progress",
      "phaseId": "d4e5f6a7",
      "sectionId": "b8c9d0e1",
      "lastUpdated": "2026-03-10T13:00:00Z"
    },
    {
      "id": "c2a9e5b3",
      "subject": "Add usage examples to README",
      "status": "pending",
      "lastUpdated": "2026-03-10T12:00:00Z"
    }
  ],
  "version": "1.0",
  "lastUpdated": "2026-03-10T15:00:00Z"
}
```

Notes:

- `name` and `description` are optional `omitempty` fields on the `TaskList`; they are absent when empty
- `phases` and `sections` are lookup tables for display names and descriptions used in markdown export headings; they are populated via `--phase-name`/`--phase-description`/`--section-name`/`--section-description` on `create` and `update`; phase and section IDs are auto-generated 8-char hex
- `description` on phases and sections is optional; it captures goals or context (e.g. what the phase aims to accomplish, what packages it covers) and is rendered below the heading in markdown export
- A task's `phaseId`/`sectionId` fields are bare ID strings; there is no referential integrity — a task can reference a phase not in the `phases` array and vice versa
- `omitempty` fields (`phaseId`, `sectionId`, `notes`) are absent when empty (see `c2a9e5b3`)
- `status` must be one of the defined `TaskStatus` values: `pending`, `in_progress`, `completed`
- `lastUpdated` on tasks is set automatically on every `Create` or `Update`; it is always present
- `version` is set to `"1.0"` on first write and never changed; it is a forward-looking schema marker for future migration tooling
- `lastUpdated` on the `TaskList` root is set to the current time on every `Save()`; it is not consumed by any subcommand

---

## 5. Flag Parsing

Each subcommand parses its own flags using `flag.NewFlagSet` (stdlib):

```go
fs := flag.NewFlagSet("lorah task list", flag.ContinueOnError)
var statuses multiFlag
fs.Var(&statuses, "status", "filter by status (repeatable)")
format := fs.String("format", "markdown", "output format")
// ...
fs.Parse(args)
```

`multiFlag` is a `[]string` type implementing `flag.Value` (`String()` and `Set()` methods).
It is used for any flag that can be specified more than once. `flag.String` cannot support
repeatable flags — each call to `Set` overwrites the previous value.

`flag.ContinueOnError` is used so the handler can print custom usage on error.
Unknown flags exit 1 with a usage message. This is stdlib — no external library.

For `update`'s partial-update semantics, use `fs.Visit` after `fs.Parse` to enumerate
only the flags that were explicitly provided:

```go
provided := map[string]bool{}
fs.Visit(func(f *flag.Flag) { provided[f.Name] = true })
```

Only update a field on the task if `provided["<flag-name>"]` is true. This is the correct
approach because it allows `--subject=""` to clear the subject field. Do not use an empty
string as a sentinel for "not provided" — that makes it impossible to clear string fields.

---

## 6. Subcommand Behavior

### `list`

```
lorah task list [--status=STATUS] [--phase=P] [--section=S] [--limit=N] [--flat] [--format=json|markdown]
```

- Default format: `markdown`
- `--status` is repeatable to filter by multiple statuses
- Results ordered by phase, then section, then id (ordered by position in the `phases`/`sections` arrays, then task ID)
- `--flat`: suppresses phase and section headings; outputs a flat bullet list of tasks. Only applies to markdown format; ignored when `--format=json`

### `get`

```
lorah task get <id> [--format=json|markdown]
```

- Default format: `markdown`
- Exits 1 with an error message if the task ID is not found

### `create`

```
lorah task create --subject="..." [--status=STATUS] [--phase=P] [--phase-name="..."] [--phase-description="..."] [--section=S] [--section-name="..."] [--section-description="..."] [--project-name="..."] [--project-description="..."]
```

- `--subject` is required
- Auto-generates a unique 8-character lowercase hex ID (e.g. `"a3f7b2c1"`)
- Default status: `pending`
- Exits 1 if `--status` is not a valid `TaskStatus` (`pending`, `in_progress`, `completed`)
- Sets `lastUpdated` to the current time
- Prints the new task's ID to stdout on success (one line per created entity; see output format below)
- `--phase` assigns the task to an existing phase by hex ID
- `--phase-name` sets the phase name; if `--phase` is omitted, auto-generates a new phase hex ID and creates it
- `--phase-description` sets the phase description; follows the same `--phase`/auto-generate rules as `--phase-name`
- `--section` assigns the task to an existing section by hex ID; requires a phase context (`--phase` or auto-generated via `--phase-name`)
- `--section-name` sets the section name; if `--section` is omitted, auto-generates a new section hex ID and creates it; requires a phase context (`--phase` or auto-generated via `--phase-name`)
- `--section-description` sets the section description; follows the same `--section`/auto-generate rules as `--section-name`
- `--project-name` sets the `name` field on the `TaskList`
- `--project-description` sets the `description` field on the `TaskList`
- Output on success (one line per created entity, only lines for newly created entities are printed):

  ```
  phase <hex-id>
  section <hex-id>
  task <hex-id>
  ```

### `update`

```
lorah task update <id> [--status=STATUS] [--subject="..."] [--phase=P] [--phase-name="..."] [--phase-description="..."] [--section=S] [--section-name="..."] [--section-description="..."] [--notes="..."] [--project-name="..."] [--project-description="..."]
```

- Partial update: only provided flags are changed; omitted fields are unchanged
- Exits 1 if task not found
- Exits 1 if `--status` is not a valid `TaskStatus` (`pending`, `in_progress`, `completed`)
- Always sets `lastUpdated` to the current time
- `--notes` sets (replaces) the `notes` field; the caller is responsible for appending if needed
- `--phase` reassigns the task to an existing phase by hex ID
- `--phase-name` upserts the name on the phase referenced by `--phase`; requires `--phase`
- `--phase-description` upserts the description on the phase referenced by `--phase`; requires `--phase`
- `--section` reassigns the task to an existing section by hex ID; requires `--phase`
- `--section-name` upserts the name on the section referenced by `--section`; requires `--section`
- `--section-description` upserts the description on the section referenced by `--section`; requires `--section`
- `--project-name` sets the `name` field on the `TaskList`
- `--project-description` sets the `description` field on the `TaskList`

### `delete`

```
lorah task delete <id>
```

- Exits 1 with error message if task not found
- No output on success
- Returns 0

### `export`

```
lorah task export [--output=FILE] [--status=STATUS]
```

- Outputs to stdout unless `--output` specifies a file path
- `--status` is repeatable; when provided, only tasks matching any of the specified statuses are included
- Renders markdown (see Output Formats section)

---

## 7. Output Formats

### Single-Task Markdown (for `get`)

```markdown
# Implement stream-JSON output parsing

**Status:** completed
**Phase:** Phase 1: Run Loop
**Section:** 1.1 Output Formatting
**Updated:** 2026-03-10T14:22:00Z

Scans stdout line-by-line as newline-delimited JSON. Skips empty lines and parse failures gracefully.
```

Rules:

- H1 heading is the task `subject`
- `status` is always shown
- `lastUpdated` is always shown (it is always set)
- Optional fields (`phaseId`, `sectionId`) are omitted when empty; when present, render the phase/section name if available, otherwise the hex ID
- `notes` printed as-is below the field list, separated by a blank line; if `notes` is empty, print `**Notes:** (none)`

### Single-Task JSON (for `get`)

```json
{
  "id": "a3f7b2c1",
  "subject": "Implement stream-JSON output parsing",
  "status": "completed",
  "phaseId": "d4e5f6a7",
  "sectionId": "b8c9d0e1",
  "notes": "Scans stdout line-by-line as newline-delimited JSON. Skips empty lines and parse failures gracefully.",
  "lastUpdated": "2026-03-10T14:22:00Z"
}
```

A single task object with all fields present. Not wrapped in a `{"tasks": [...]}` envelope — `get` returns the task directly, not a list.

### List Markdown (for `list`)

The `list` subcommand renders markdown using the same grouped structure as `export`: phase H2 headings, section H3 headings, and bullet list items. Output is scoped to the filtered results — phases and sections with zero matching tasks are omitted. The list and export markdown formatters share the same rendering logic.

Project name/description H1 is **not** rendered in `list` output (only in `export`).

Tasks with a non-empty `notes` field render the notes in a fenced code block with `notes` info string, indented 2 spaces, on the line after the task bullet. Tasks without notes render as bare bullets.

Example (`lorah task list --status=pending --phase=d4e5f6a7`):

````markdown
## Phase 1: Run Loop

### 1.1 Output Formatting

- `d6e7f8a9` [pending] Add tool input truncation

  ```notes
  Tool inputs currently show full content. Need to truncate to 1 line
  with `... +N lines` indicator. See output.md spec for details.
  ```

### 1.2 Signal Handling

- `c1d2e3f4` [pending] Implement two-signal graceful shutdown
````

#### Flat Mode (`--flat`)

When `--flat` is passed, phase and section headings are suppressed and notes are omitted. Output is a flat bullet list:

```markdown
- `d6e7f8a9` [pending] Add tool input truncation
- `c1d2e3f4` [pending] Implement two-signal graceful shutdown
```

Useful when agents want minimal tokens and the grouping context is unnecessary.

### JSON (for scripting)

```json
{
  "tasks": [
    {
      "id": "a3f7b2c1",
      "subject": "Implement stream-JSON output parsing",
      "status": "completed",
      "phaseId": "d4e5f6a7",
      "sectionId": "b8c9d0e1",
      "notes": "Scans stdout line-by-line as newline-delimited JSON. Skips empty lines and parse failures gracefully.",
      "lastUpdated": "2026-03-10T14:22:00Z"
    }
  ]
}
```

Full task objects, suitable for programmatic parsing.

### Markdown Export (for `export`)

````markdown
# Lorah Development Plan

Track progress on the Lorah infinite-loop harness implementation.

## Phase 1: Run Loop

Implement the infinite loop, subprocess execution, and output formatting.

### 1.1 Output Formatting

Stream-JSON parsing and color-coded terminal output.

- `a3f7b2c1` [completed] Implement stream-JSON output parsing

  ```notes
  Scans stdout line-by-line as newline-delimited JSON. Skips empty lines
  and parse failures gracefully.
  ```

- `b8e4d1f0` [completed] Add color-coded section headers
- `d6e7f8a9` [pending] Add tool input truncation

### 1.2 Signal Handling

- `c1d2e3f4` [pending] Implement two-signal graceful shutdown
````

Grouped by phase (H2) then section (H3). Status is rendered inline in brackets after the task ID.

Project heading rules:

- If `name` is set: render `# {name}` as the first line
- If `description` is set (and `name` is set): render as a plain paragraph after the H1, followed by a blank line before the first phase heading
- If `name` is not set: no H1 heading (export starts directly at phase headings)
- If `description` is set but `name` is not: skip

Phase and section heading rules:

- Phase with name: `## {name}`; without name: `## {id}` (hex fallback)
- Section with name: `### {name}`; without name: `### {id}` (hex fallback)
- If a phase or section has a `description`, render it as a plain paragraph on the line after the heading, followed by a blank line before the task list
- If there is no `description`, omit the paragraph entirely (no blank placeholder)

Tasks with no `phase` are collected under `## (none)`.
Tasks with no `section` appear directly under the phase heading with no H3 sub-heading.

Task note rules:

- If a task has a non-empty `notes` field, render a blank line after the task bullet, then a fenced code block with `notes` info string indented 2 spaces (opening fence, all content lines, closing fence)
- The notes content is rendered verbatim inside the fence
- The 2-space indent keeps the code block part of the list item, preserving list structure
- Tasks without notes render as bare bullets with no extra lines

---

## 8. Agent Integration

### Recommended Workflow

1. `lorah task list --status=pending` — scan available tasks
2. `lorah task get <id>` — read full details for the chosen task
3. `lorah task update <id> --status=in_progress` — claim the task
4. _(agent does the work)_
5. `lorah task update <id> --status=completed --notes="..."` — record what happened

### Learning from Prior Work

Before starting work on a related task, review recent completions:

```

lorah task list --status=completed --phase=<phase-hex-id> --limit=5

```

Completion notes should include: what worked, issues encountered, related files changed,
and any follow-up observations. This enables future iterations to avoid repeating mistakes
and build on prior context without re-reading the codebase from scratch.

---

## 9. Package Structure

```

internal/task/
task.go -- Phase, Section, Task, TaskStatus, TaskList, Filter types
storage.go -- Storage interface
json_storage.go -- JSONStorage implementation
format.go -- Output formatters: json, markdown
cmd.go -- CLI subcommand dispatch and handlers

```

`internal/` ensures the package is not importable outside the module.

---

## 10. Examples

```sh
# List pending tasks (agent workflow)
lorah task list --status=pending

# List pending tasks (flat, for minimal tokens)
lorah task list --status=pending --flat

# Get full task details
lorah task get a3f7b2c1

# Get full task details as JSON (for scripting)
lorah task get a3f7b2c1 --format=json

# Create first task in a new phase and section (auto-generates phase/section IDs)
# Output:
#   phase d4e5f6a7
#   section b8c9d0e1
#   task a3f7b2c1
lorah task create --subject="Implement stream-JSON output parsing" --project-name="Lorah Development Plan" --project-description="Track progress on the Lorah infinite-loop harness implementation." --phase-name="Phase 1: Run Loop" --phase-description="Implement the infinite loop, subprocess execution, and output formatting." --section-name="1.1 Output Formatting" --section-description="Stream-JSON parsing and color-coded terminal output."

# Create subsequent tasks in the same phase/section (IDs from previous output)
lorah task create --subject="Add color-coded section headers" --phase=d4e5f6a7 --section=b8c9d0e1

# Claim and complete a task
lorah task update a3f7b2c1 --status=in_progress
lorah task update a3f7b2c1 --status=completed --notes="Scans stdout line-by-line as newline-delimited JSON. Skips empty lines and parse failures gracefully."

# Human review (markdown export)
lorah task export --output=PROGRESS.md

# Filter by phase
lorah task list --phase=d4e5f6a7 --status=pending --limit=10
```

---

## 11. Related Specifications

- [cli.md](cli.md) — top-level routing to the `task` subcommand
- [run.md](run.md) — the run loop that agents use to process tasks
- [output.md](output.md) — output system (`task` uses its own formatters, not `printMessages`)
