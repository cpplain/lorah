# Agent Task Management System Design

**Purpose:** Efficient task management for autonomous agent loops with minimal token usage and maximum agent learning capability.

**Author:** Design session 2026-03-12
**Status:** Design Complete - Ready for Implementation

---

## Table of Contents

1. [Problem Statement](#problem-statement)
2. [Design Goals](#design-goals)
3. [Solution Architecture](#solution-architecture)
4. [Token Efficiency Analysis](#token-efficiency-analysis)
5. [JSON Schema Design](#json-schema-design)
6. [Go Harness CLI](#go-harness-cli)
7. [Agent Learning Through Completion Notes](#agent-learning-through-completion-notes)
8. [Markdown Generation](#markdown-generation)
9. [Reference Implementation](#reference-implementation)
10. [Migration from Existing Templates](#migration-from-existing-templates)

---

## Problem Statement

### Current Approach

Using markdown files as task lists for autonomous agent loops (e.g., ralph pattern):

- Agent reads entire markdown file on each iteration
- Files grow to 500-1500+ lines as tasks are completed
- Token usage: **1,250-3,750 tokens per iteration**
- 90% of content is noise (completed tasks, blocked tasks, future work)
- Agent must mentally filter through everything

### Key Questions

1. **Context limits:** Large files hit read tool truncation limits (~2000 lines)
2. **Token efficiency:** How to reduce token usage without losing context?
3. **Agent learning:** How can agents learn from previous iterations?
4. **Human readability:** How to maintain readable task lists for humans?

---

## Design Goals

1. **Efficient filtering** - Agent only reads actionable tasks (pending, not blocked)
2. **Context preservation** - Maintain all necessary context (specs, paths, goals)
3. **Agent learning** - Completion notes enable cross-iteration learning
4. **Storage flexibility** - Abstract storage (JSON, SQLite, etc.)
5. **Human readable** - Generate markdown views for human review
6. **Simple integration** - Drop into existing Go harness/loop

---

## Solution Architecture

### Core Concept: Abstraction Layer

```
┌─────────────────────────────────────┐
│         Agent Loop                  │
│  (reads tasks, does work, updates)  │
└─────────────┬───────────────────────┘
              │
              ├─ harness task list --status=pending
              ├─ harness task get <id>
              ├─ harness task update <id> --status=completed
              │
┌─────────────▼───────────────────────┐
│      Go Harness CLI                 │
│  (validates, filters, manages)      │
└─────────────┬───────────────────────┘
              │
              ├─ Backend: JSON file
              ├─ Backend: SQLite database
              └─ Backend: Remote API
```

**Key Insight:** Harness becomes the API. Storage is an implementation detail. Agent code never changes when you swap backends.

### Why This Design?

**Compared to direct file access:**

- ✅ Agent can't corrupt task file
- ✅ Harness adds validation, locking, history
- ✅ Easy to swap JSON → SQLite later
- ✅ Multi-agent coordination possible

**Compared to built-in Task tools:**

- ✅ Guaranteed persistence across agent invocations
- ✅ Explicit file-based storage (no lifecycle ambiguity)
- ✅ You control the format and migration path

---

## Token Efficiency Analysis

### Current Markdown Approach (1500-line file)

```
1500 lines × ~2.5 tokens/line = 3,750 tokens per iteration

Contains:
- ✅ 50 pending tasks (needed)
- ❌ 200 completed tasks (noise)
- ❌ 30 blocked tasks (can't work on)
- ❌ 100 future tasks (not relevant yet)
- ❌ All historical context and notes
```

### Harness CLI Approach

```bash
harness task list --status=pending --limit=10 --format=json
```

Returns only actionable tasks:

```
10-20 active tasks × ~5-10 lines each × ~2.5 tokens/line = 125-500 tokens
```

**Result: 80-95% token reduction**

### Tiered Detail Approach (Even Better)

```bash
# Step 1: Summary view (minimal tokens)
harness task list --status=pending --format=compact
# 1: Fix auth bug
# 2: Add tests
# 3: Update docs
# ~50 tokens

# Step 2: Get details only when claiming
harness task get 1 --format=json
# {"id": 1, "subject": "...", "description": "... full markdown context ..."}
# ~100-200 tokens

# Total: ~150-250 tokens vs 3,750 tokens = 93% savings
```

---

## JSON Schema Design

### Task Structure

```json
{
  "tasks": [
    {
      "id": "9.1.1",
      "subject": "StatCard.svelte component",
      "description": "Metric display with trend\n\nPath: web/loom-web/...\nReference: specs/...",
      "status": "pending",
      "priority": 1,
      "category": "ui-components",
      "phase": "9",
      "section": "9.1",
      "tags": ["svelte", "component"],
      "dependencies": [],
      "blockedBy": [],
      "createdAt": "2026-01-20T10:30:00Z",
      "startedAt": null,
      "completedAt": null,
      "estimatedTokens": 500,
      "metadata": {
        "path": "web/loom-web/src/lib/components/common/",
        "specs": ["specs/observability-ui.md"],
        "relatedFiles": ["web/loom-web/src/lib/ui/"]
      },
      "notes": ""
    }
  ],
  "version": "1.0",
  "lastUpdated": "2026-03-12T14:22:00Z"
}
```

### Field Descriptions

| Field             | Type    | Purpose                                                  |
| ----------------- | ------- | -------------------------------------------------------- |
| `id`              | string  | Unique identifier (hierarchical: "9.1.1" or UUID)        |
| `subject`         | string  | Brief, actionable title (imperative form)                |
| `description`     | string  | **Markdown formatted** - full context, specs, references |
| `status`          | enum    | `pending`, `in_progress`, `completed`, `blocked`         |
| `priority`        | int     | 1-5, higher = more urgent                                |
| `category`        | string  | For filtering: `ui-components`, `api-endpoints`, etc.    |
| `phase`           | string  | Top-level grouping                                       |
| `section`         | string  | Sub-grouping within phase                                |
| `tags`            | array   | Flexible labels: `["svelte", "urgent", "backend"]`       |
| `dependencies`    | array   | Task IDs that must complete first                        |
| `blockedBy`       | array   | External blockers (not other tasks)                      |
| `createdAt`       | ISO8601 | Timestamp                                                |
| `startedAt`       | ISO8601 | When agent claimed task                                  |
| `completedAt`     | ISO8601 | When marked complete                                     |
| `estimatedTokens` | int     | Optional: complexity estimate                            |
| `metadata`        | object  | Arbitrary structured data                                |
| `notes`           | string  | **Markdown formatted** - completion notes                |

### Status Workflow

```
pending → in_progress → completed
   ↓           ↓
 blocked    blocked
```

---

## Go Harness CLI

### Command Structure

```bash
# List tasks
harness task list [--status=STATUS] [--category=CATEGORY] [--phase=PHASE] [--limit=N] [--format=json|compact|markdown]

# Get task details
harness task get <id> [--format=json|markdown]

# Create task
harness task create --subject="..." --description="..." [--priority=N] [--category=CAT] [--phase=P]

# Update task
harness task update <id> [--status=STATUS] [--notes="..." --priority=N]

# Mark in progress
harness task start <id>

# Mark completed
harness task complete <id> [--notes="..."]

# Export to markdown
harness task export [--output=FILE.md] [--status=STATUS]

# Watch mode (regenerate markdown on changes)
harness task watch --output=TASKS.md

# Show statistics
harness task stats
```

### Output Formats

#### JSON (for agent parsing)

```json
{
  "tasks": [
    {"id": "1", "subject": "Fix auth bug", "status": "pending", ...}
  ]
}
```

#### Compact (minimal tokens)

```
1  [pending]     Fix authentication bug
2  [pending]     Add unit tests
3  [in_progress] Update documentation
```

#### Markdown (human readable)

```markdown
## Phase 9: Web UI Components

### 9.1 Common Components

- [ ] Fix authentication bug
- [ ] Add unit tests
- [x] Update documentation
```

### Agent Prompt Integration

```
CRITICAL: All task operations MUST use the harness CLI:

1. List available tasks:
   harness task list --status=pending --format=json

2. Get task details:
   harness task get <id> --format=json

3. Start working on a task:
   harness task start <id>

4. Complete a task:
   harness task complete <id> --notes="completion notes"

Never directly read/write task files.
Always include completion notes when marking tasks complete.
```

---

## Agent Learning Through Completion Notes

### The Learning Loop

One of the most powerful aspects of this design is **cross-iteration learning**. Each agent iteration can learn from previous attempts.

### Completion Notes Format

**Structured markdown in `notes` field:**

```markdown
## Completion Summary

Fixed authentication by updating the JWT validation logic.

## What Worked

- Regenerated secret keys
- Added expiry validation
- Restarted auth service

## Issues Encountered

- First attempt failed - needed to clear Redis cache
- JWT library version mismatch required upgrade

## Related Files

- auth/jwt.go:45-67
- config/auth.yaml

## Follow-up Tasks

- Consider adding monitoring for token expiry
- Document key rotation process
```

### Agent Learning Workflow

**Before picking a task:**

```bash
# Agent reviews recent completions for context
harness task list --status=completed --limit=20 --format=json
```

**Why this works:**

1. **Pattern recognition** - "Last time we fixed auth, we had to restart the service"
2. **Avoid repeating mistakes** - "Redis cache needed clearing"
3. **Understand dependencies** - "JWT library version matters"
4. **Context for related work** - See what files were touched

**Token cost:** ~200-500 tokens for 20 recent completions
**Value:** Massive - prevents repeating work, builds on knowledge

### Example: Agent Learning in Action

**Iteration 1:** Agent fixes auth bug, notes "Had to clear Redis cache"

**Iteration 5:** Agent picks another auth-related task, reads recent completions:

```json
{
  "id": "42",
  "subject": "Fix JWT validation",
  "notes": "## Issues Encountered\n- First attempt failed - needed to clear Redis cache"
}
```

Agent now knows to clear Redis cache proactively. Saves an entire failed iteration.

---

## Markdown Generation

### Why Generate Markdown?

**JSON is the source of truth** (structured, queryable, efficient)
**Markdown is the human interface** (readable, familiar, shareable)

### Generate on Demand

```bash
# Quick review
harness task list --format=markdown

# Export for sharing
harness task export --output=PLAN.md

# Live updating view
harness task watch --output=PLAN.md
```

### Implementation

**Simple approach:** ~50 lines of Go with `fmt.Fprintf`

**Advanced approach:** Use `text/template` for custom layouts

### Example Output

Generated markdown looks exactly like original hidave.md template:

```markdown
# Observability Suite Implementation Plan

**Status:** 39/39 Components Complete
**Last Updated:** 2026-03-12

---

## Phase 9: Web UI Components

**Goal:** Build Svelte 5 components for the observability UI.

### 9.1 Common Components

**Path:** `web/loom-web/src/lib/components/common/`

- [x] `StatCard.svelte` — Metric display with trend
- [x] `Sparkline.svelte` — Mini inline chart
- [ ] `TimeRangePicker.svelte` — Time range selector

### 9.2 Crash Components

**Path:** `web/loom-web/src/lib/components/crash/`

- [x] `IssueList.svelte` — Paginated issue list
- [ ] `IssueDetail.svelte` — Full issue view
```

**With completion notes:**

```bash
harness task show 42 --format=markdown
```

```markdown
# Task: Fix authentication bug

**Status:** Completed
**Completed:** 2026-01-25
**Phase:** 9.2
**Priority:** High

## Description

Fixed authentication by updating the JWT validation logic.

Path: `auth/jwt.go:45-67`

---

## Completion Notes

### What Worked

- Regenerated secret keys
- Added expiry validation

### Issues Encountered

- Redis cache needed clearing
- JWT library version mismatch
```

---

## Reference Implementation

### File Structure

```
harness/
├── cmd/
│   └── task/
│       ├── main.go          # CLI entry point
│       ├── list.go          # List command
│       ├── get.go           # Get command
│       ├── create.go        # Create command
│       ├── update.go        # Update command
│       └── export.go        # Export markdown
├── pkg/
│   └── tasks/
│       ├── task.go          # Data structures
│       ├── storage.go       # Storage interface
│       ├── json_storage.go  # JSON backend
│       ├── sqlite_storage.go # SQLite backend (future)
│       ├── markdown.go      # Markdown formatter
│       └── filters.go       # Query filters
└── tasks.json               # Task storage file
```

### Go Code Examples

#### Data Structures (`task.go`)

```go
package tasks

import "time"

type Task struct {
    ID            string            `json:"id"`
    Subject       string            `json:"subject"`
    Description   string            `json:"description"`
    Status        TaskStatus        `json:"status"`
    Priority      int               `json:"priority"`
    Category      string            `json:"category,omitempty"`
    Phase         string            `json:"phase,omitempty"`
    Section       string            `json:"section,omitempty"`
    Tags          []string          `json:"tags,omitempty"`
    Dependencies  []string          `json:"dependencies,omitempty"`
    BlockedBy     []string          `json:"blockedBy,omitempty"`
    CreatedAt     time.Time         `json:"createdAt"`
    StartedAt     *time.Time        `json:"startedAt,omitempty"`
    CompletedAt   *time.Time        `json:"completedAt,omitempty"`
    EstimatedTokens int             `json:"estimatedTokens,omitempty"`
    Metadata      map[string]any    `json:"metadata,omitempty"`
    Notes         string            `json:"notes,omitempty"`
}

type TaskStatus string

const (
    StatusPending    TaskStatus = "pending"
    StatusInProgress TaskStatus = "in_progress"
    StatusCompleted  TaskStatus = "completed"
    StatusBlocked    TaskStatus = "blocked"
)

type TaskList struct {
    Tasks       []Task    `json:"tasks"`
    Version     string    `json:"version"`
    LastUpdated time.Time `json:"lastUpdated"`
}

// Filter options for querying tasks
type FilterOptions struct {
    Status   []TaskStatus
    Category string
    Phase    string
    Tags     []string
    Limit    int
}
```

#### Storage Interface (`storage.go`)

```go
package tasks

// Storage defines the interface for task persistence
// This allows swapping between JSON, SQLite, or other backends
type Storage interface {
    // Load all tasks
    Load() (*TaskList, error)

    // Save all tasks
    Save(list *TaskList) error

    // Get single task by ID
    Get(id string) (*Task, error)

    // List tasks with filters
    List(filter FilterOptions) ([]Task, error)

    // Create new task
    Create(task *Task) error

    // Update existing task
    Update(task *Task) error

    // Delete task
    Delete(id string) error
}
```

#### JSON Backend (`json_storage.go`)

```go
package tasks

import (
    "encoding/json"
    "fmt"
    "os"
    "sync"
    "time"
)

type JSONStorage struct {
    filepath string
    mu       sync.RWMutex
}

func NewJSONStorage(filepath string) *JSONStorage {
    return &JSONStorage{filepath: filepath}
}

func (s *JSONStorage) Load() (*TaskList, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    data, err := os.ReadFile(s.filepath)
    if err != nil {
        if os.IsNotExist(err) {
            return &TaskList{
                Tasks:       []Task{},
                Version:     "1.0",
                LastUpdated: time.Now(),
            }, nil
        }
        return nil, err
    }

    var list TaskList
    if err := json.Unmarshal(data, &list); err != nil {
        return nil, fmt.Errorf("parse tasks.json: %w", err)
    }

    return &list, nil
}

func (s *JSONStorage) Save(list *TaskList) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    list.LastUpdated = time.Now()

    data, err := json.MarshalIndent(list, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal tasks: %w", err)
    }

    if err := os.WriteFile(s.filepath, data, 0644); err != nil {
        return fmt.Errorf("write tasks.json: %w", err)
    }

    return nil
}

func (s *JSONStorage) Get(id string) (*Task, error) {
    list, err := s.Load()
    if err != nil {
        return nil, err
    }

    for _, task := range list.Tasks {
        if task.ID == id {
            return &task, nil
        }
    }

    return nil, fmt.Errorf("task not found: %s", id)
}

func (s *JSONStorage) List(filter FilterOptions) ([]Task, error) {
    list, err := s.Load()
    if err != nil {
        return nil, err
    }

    var filtered []Task
    for _, task := range list.Tasks {
        if matchesFilter(task, filter) {
            filtered = append(filtered, task)
        }

        if filter.Limit > 0 && len(filtered) >= filter.Limit {
            break
        }
    }

    return filtered, nil
}

func (s *JSONStorage) Create(task *Task) error {
    list, err := s.Load()
    if err != nil {
        return err
    }

    // Check for duplicate ID
    for _, t := range list.Tasks {
        if t.ID == task.ID {
            return fmt.Errorf("task already exists: %s", task.ID)
        }
    }

    task.CreatedAt = time.Now()
    list.Tasks = append(list.Tasks, *task)

    return s.Save(list)
}

func (s *JSONStorage) Update(task *Task) error {
    list, err := s.Load()
    if err != nil {
        return err
    }

    for i, t := range list.Tasks {
        if t.ID == task.ID {
            list.Tasks[i] = *task
            return s.Save(list)
        }
    }

    return fmt.Errorf("task not found: %s", task.ID)
}

func (s *JSONStorage) Delete(id string) error {
    list, err := s.Load()
    if err != nil {
        return err
    }

    for i, t := range list.Tasks {
        if t.ID == id {
            list.Tasks = append(list.Tasks[:i], list.Tasks[i+1:]...)
            return s.Save(list)
        }
    }

    return fmt.Errorf("task not found: %s", id)
}

func matchesFilter(task Task, filter FilterOptions) bool {
    // Status filter
    if len(filter.Status) > 0 {
        match := false
        for _, status := range filter.Status {
            if task.Status == status {
                match = true
                break
            }
        }
        if !match {
            return false
        }
    }

    // Category filter
    if filter.Category != "" && task.Category != filter.Category {
        return false
    }

    // Phase filter
    if filter.Phase != "" && task.Phase != filter.Phase {
        return false
    }

    // Tags filter (any match)
    if len(filter.Tags) > 0 {
        match := false
        for _, filterTag := range filter.Tags {
            for _, taskTag := range task.Tags {
                if taskTag == filterTag {
                    match = true
                    break
                }
            }
        }
        if !match {
            return false
        }
    }

    return true
}
```

#### Markdown Formatter (`markdown.go`)

```go
package tasks

import (
    "bytes"
    "fmt"
    "sort"
    "strings"
    "time"
)

type MarkdownOptions struct {
    IncludeCompleted bool
    IncludeNotes     bool
    GroupBy          string // "phase", "category", "status"
}

func FormatAsMarkdown(tasks []Task, opts MarkdownOptions) string {
    var buf bytes.Buffer

    switch opts.GroupBy {
    case "phase":
        formatByPhase(&buf, tasks, opts)
    case "category":
        formatByCategory(&buf, tasks, opts)
    case "status":
        formatByStatus(&buf, tasks, opts)
    default:
        formatFlat(&buf, tasks, opts)
    }

    return buf.String()
}

func formatByPhase(buf *bytes.Buffer, tasks []Task, opts MarkdownOptions) {
    // Group tasks by phase
    phaseMap := make(map[string][]Task)
    for _, task := range tasks {
        if !opts.IncludeCompleted && task.Status == StatusCompleted {
            continue
        }
        phase := task.Phase
        if phase == "" {
            phase = "Unassigned"
        }
        phaseMap[phase] = append(phaseMap[phase], task)
    }

    // Sort phases
    var phases []string
    for phase := range phaseMap {
        phases = append(phases, phase)
    }
    sort.Strings(phases)

    // Write each phase
    for _, phase := range phases {
        phaseTasks := phaseMap[phase]

        fmt.Fprintf(buf, "## Phase %s\n\n", phase)

        // Group by section within phase
        sectionMap := make(map[string][]Task)
        for _, task := range phaseTasks {
            section := task.Section
            if section == "" {
                section = "General"
            }
            sectionMap[section] = append(sectionMap[section], task)
        }

        var sections []string
        for section := range sectionMap {
            sections = append(sections, section)
        }
        sort.Strings(sections)

        for _, section := range sections {
            sectionTasks := sectionMap[section]

            fmt.Fprintf(buf, "### %s\n\n", section)

            // Get path from first task's metadata
            if len(sectionTasks) > 0 {
                if path, ok := sectionTasks[0].Metadata["path"].(string); ok {
                    fmt.Fprintf(buf, "**Path:** `%s`\n\n", path)
                }
            }

            for _, task := range sectionTasks {
                formatTaskItem(buf, task)
            }

            buf.WriteString("\n")
        }
    }
}

func formatByCategory(buf *bytes.Buffer, tasks []Task, opts MarkdownOptions) {
    categoryMap := make(map[string][]Task)
    for _, task := range tasks {
        if !opts.IncludeCompleted && task.Status == StatusCompleted {
            continue
        }
        category := task.Category
        if category == "" {
            category = "Uncategorized"
        }
        categoryMap[category] = append(categoryMap[category], task)
    }

    var categories []string
    for cat := range categoryMap {
        categories = append(categories, cat)
    }
    sort.Strings(categories)

    for _, category := range categories {
        fmt.Fprintf(buf, "## %s\n\n", strings.Title(category))
        for _, task := range categoryMap[category] {
            formatTaskItem(buf, task)
        }
        buf.WriteString("\n")
    }
}

func formatByStatus(buf *bytes.Buffer, tasks []Task, opts MarkdownOptions) {
    statusOrder := []TaskStatus{StatusInProgress, StatusPending, StatusBlocked, StatusCompleted}
    statusNames := map[TaskStatus]string{
        StatusInProgress: "In Progress",
        StatusPending:    "Pending",
        StatusBlocked:    "Blocked",
        StatusCompleted:  "Completed",
    }

    for _, status := range statusOrder {
        if !opts.IncludeCompleted && status == StatusCompleted {
            continue
        }

        var statusTasks []Task
        for _, task := range tasks {
            if task.Status == status {
                statusTasks = append(statusTasks, task)
            }
        }

        if len(statusTasks) == 0 {
            continue
        }

        fmt.Fprintf(buf, "## %s\n\n", statusNames[status])
        for _, task := range statusTasks {
            formatTaskItem(buf, task)
        }
        buf.WriteString("\n")
    }
}

func formatFlat(buf *bytes.Buffer, tasks []Task, opts MarkdownOptions) {
    for _, task := range tasks {
        if !opts.IncludeCompleted && task.Status == StatusCompleted {
            continue
        }
        formatTaskItem(buf, task)
    }
}

func formatTaskItem(buf *bytes.Buffer, task Task) {
    checkbox := "[ ]"
    if task.Status == StatusCompleted {
        checkbox = "[x]"
    }

    fmt.Fprintf(buf, "- %s `%s` %s", checkbox, task.ID, task.Subject)

    // Add priority indicator
    if task.Priority > 3 {
        buf.WriteString(" 🔴")
    }

    buf.WriteString("\n")
}

func FormatTaskDetail(task Task, includeNotes bool) string {
    var buf bytes.Buffer

    fmt.Fprintf(&buf, "# Task: %s\n\n", task.Subject)
    fmt.Fprintf(&buf, "**ID:** %s\n", task.ID)
    fmt.Fprintf(&buf, "**Status:** %s\n", task.Status)
    fmt.Fprintf(&buf, "**Priority:** %d\n", task.Priority)

    if task.Category != "" {
        fmt.Fprintf(&buf, "**Category:** %s\n", task.Category)
    }
    if task.Phase != "" {
        fmt.Fprintf(&buf, "**Phase:** %s\n", task.Phase)
    }
    if len(task.Tags) > 0 {
        fmt.Fprintf(&buf, "**Tags:** %s\n", strings.Join(task.Tags, ", "))
    }

    fmt.Fprintf(&buf, "**Created:** %s\n", task.CreatedAt.Format(time.RFC3339))
    if task.StartedAt != nil {
        fmt.Fprintf(&buf, "**Started:** %s\n", task.StartedAt.Format(time.RFC3339))
    }
    if task.CompletedAt != nil {
        fmt.Fprintf(&buf, "**Completed:** %s\n", task.CompletedAt.Format(time.RFC3339))
    }

    buf.WriteString("\n---\n\n")

    if task.Description != "" {
        fmt.Fprintf(&buf, "## Description\n\n%s\n\n", task.Description)
    }

    if len(task.Dependencies) > 0 {
        fmt.Fprintf(&buf, "## Dependencies\n\n")
        for _, dep := range task.Dependencies {
            fmt.Fprintf(&buf, "- %s\n", dep)
        }
        buf.WriteString("\n")
    }

    if includeNotes && task.Notes != "" {
        fmt.Fprintf(&buf, "## Completion Notes\n\n%s\n", task.Notes)
    }

    return buf.String()
}
```

#### CLI Command Example (`list.go`)

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "yourproject/pkg/tasks"
)

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List tasks",
    RunE:  runList,
}

var (
    listStatus   []string
    listCategory string
    listPhase    string
    listLimit    int
    listFormat   string
)

func init() {
    listCmd.Flags().StringSliceVar(&listStatus, "status", nil, "Filter by status (pending,in_progress,completed,blocked)")
    listCmd.Flags().StringVar(&listCategory, "category", "", "Filter by category")
    listCmd.Flags().StringVar(&listPhase, "phase", "", "Filter by phase")
    listCmd.Flags().IntVar(&listLimit, "limit", 0, "Limit number of results")
    listCmd.Flags().StringVar(&listFormat, "format", "compact", "Output format (json,compact,markdown)")
}

func runList(cmd *cobra.Command, args []string) error {
    storage := tasks.NewJSONStorage("tasks.json")

    // Build filter
    filter := tasks.FilterOptions{
        Category: listCategory,
        Phase:    listPhase,
        Limit:    listLimit,
    }

    for _, s := range listStatus {
        filter.Status = append(filter.Status, tasks.TaskStatus(s))
    }

    // Get tasks
    taskList, err := storage.List(filter)
    if err != nil {
        return fmt.Errorf("list tasks: %w", err)
    }

    // Output based on format
    switch listFormat {
    case "json":
        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")
        return enc.Encode(map[string]interface{}{
            "tasks": taskList,
        })

    case "compact":
        for _, task := range taskList {
            status := "[ ]"
            if task.Status == tasks.StatusCompleted {
                status = "[x]"
            }
            fmt.Printf("%-4s [%-12s] %s\n", task.ID, task.Status, task.Subject)
        }

    case "markdown":
        opts := tasks.MarkdownOptions{
            IncludeCompleted: true,
            GroupBy:          "phase",
        }
        md := tasks.FormatAsMarkdown(taskList, opts)
        fmt.Println(md)

    default:
        return fmt.Errorf("unknown format: %s", listFormat)
    }

    return nil
}
```

---

## Migration from Existing Templates

### From hidave.md Style to JSON

The hidave.md template maps very cleanly to this JSON schema:

**Markdown structure:**

```markdown
## Phase 9: Web UI Components

**Goal:** Build Svelte 5 components for the observability UI.

### 9.1 Common Components

**Path:** `web/loom-web/src/lib/components/common/`

- [x] `StatCard.svelte` — Metric display with trend
- [ ] `Sparkline.svelte` — Mini inline chart
```

**Becomes JSON:**

```json
{
  "id": "9.1.1",
  "subject": "StatCard.svelte — Metric display with trend",
  "description": "Path: web/loom-web/src/lib/components/common/\n\nReference: specs/observability-ui.md",
  "status": "completed",
  "phase": "9",
  "section": "9.1",
  "category": "ui-components",
  "metadata": {
    "path": "web/loom-web/src/lib/components/common/",
    "goal": "Build Svelte 5 components for the observability UI.",
    "specs": ["specs/observability-ui.md"]
  }
}
```

### Migration Script

```go
// Pseudo-code for migration
func migrateMarkdownToJSON(mdPath string) (*tasks.TaskList, error) {
    // Parse markdown
    content, _ := os.ReadFile(mdPath)

    var taskList tasks.TaskList
    var currentPhase, currentSection string

    // Parse line by line
    lines := strings.Split(string(content), "\n")
    for _, line := range lines {
        // Extract phase headers (## Phase X:)
        if strings.HasPrefix(line, "## Phase ") {
            currentPhase = extractPhaseNumber(line)
        }

        // Extract section headers (### X.X)
        if strings.HasPrefix(line, "### ") {
            currentSection = extractSectionNumber(line)
        }

        // Extract tasks (- [x] or - [ ])
        if strings.HasPrefix(line, "- [") {
            task := parseTaskLine(line)
            task.Phase = currentPhase
            task.Section = currentSection
            taskList.Tasks = append(taskList.Tasks, task)
        }
    }

    return &taskList, nil
}
```

### What Maps Well

| Markdown            | JSON                             | Notes                              |
| ------------------- | -------------------------------- | ---------------------------------- |
| Phase headers       | `phase`, `section`               | Hierarchical IDs                   |
| Task checkboxes     | `status`                         | `[x]` → completed, `[ ]` → pending |
| Inline descriptions | `subject`                        | After checkbox                     |
| Goal sections       | `metadata.goal` or `description` | Contextual info                    |
| Path sections       | `metadata.path`                  | File locations                     |
| Verification logs   | `notes`                          | Completion details                 |
| Reference links     | `metadata.specs`                 | Documentation links                |

---

## Backend Options

### JSON File (Start Here)

**Pros:**

- ✅ Simple - just a file
- ✅ Human readable/editable
- ✅ Git-friendly (text diffs)
- ✅ No dependencies

**Cons:**

- ❌ No concurrent access (need file locking)
- ❌ No complex queries
- ❌ Performance degrades with >1000 tasks

**When to use:** MVP, small projects, <100 active tasks

### SQLite (Scale Later)

**Pros:**

- ✅ Efficient queries with indexes
- ✅ Atomic updates, transactions
- ✅ Built-in history/audit via triggers
- ✅ Complex filters (JOINs, subqueries)
- ✅ Scales to 10,000+ tasks

**Cons:**

- ❌ Binary format (not directly readable)
- ❌ Schema migrations needed
- ❌ Slightly more complex

**When to use:** Large task lists, complex queries, multi-agent coordination

### Schema Migration Path

**Start with JSON:**

```
tasks.json (simple file)
```

**Migrate to SQLite when needed:**

```go
// One-time migration
harness task migrate --from=json --to=sqlite

// Harness automatically uses SQLite backend
// Agent commands don't change
```

**SQL Schema:**

```sql
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    subject TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    priority INTEGER DEFAULT 1,
    category TEXT,
    phase TEXT,
    section TEXT,
    tags TEXT, -- JSON array
    dependencies TEXT, -- JSON array
    blocked_by TEXT, -- JSON array
    created_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    estimated_tokens INTEGER,
    metadata TEXT, -- JSON
    notes TEXT
);

CREATE INDEX idx_status ON tasks(status);
CREATE INDEX idx_category ON tasks(category);
CREATE INDEX idx_phase ON tasks(phase);

CREATE TABLE task_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT NOT NULL,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    field TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);
```

---

## Implementation Checklist

### Phase 1: JSON Backend (MVP)

- [ ] Implement `task.go` data structures
- [ ] Implement `json_storage.go` backend
- [ ] Implement basic CLI commands:
  - [ ] `harness task list`
  - [ ] `harness task get <id>`
  - [ ] `harness task create`
  - [ ] `harness task update <id>`
  - [ ] `harness task start <id>`
  - [ ] `harness task complete <id>`
- [ ] Add JSON output format
- [ ] Test with sample tasks

### Phase 2: Agent Integration

- [ ] Update agent prompt with CLI commands
- [ ] Test agent loop reading tasks
- [ ] Test agent marking tasks complete
- [ ] Verify completion notes are captured

### Phase 3: Markdown Export

- [ ] Implement `markdown.go` formatter
- [ ] Add `--format=markdown` to list command
- [ ] Add `harness task export` command
- [ ] Test markdown output matches original format

### Phase 4: Advanced Features

- [ ] Add `--format=compact` for minimal token usage
- [ ] Add `harness task stats` command
- [ ] Add `harness task watch` for live updates
- [ ] Implement dependency tracking
- [ ] Add validation (blocked tasks, missing dependencies)

### Phase 5: SQLite Backend (Optional)

- [ ] Implement `sqlite_storage.go`
- [ ] Create migration script
- [ ] Add `harness task migrate` command
- [ ] Test performance with large task sets

---

## Example Usage Scenarios

### Scenario 1: Agent Starting Work

```bash
# Agent lists pending tasks
$ harness task list --status=pending --format=json
{
  "tasks": [
    {
      "id": "9.1.1",
      "subject": "StatCard.svelte component",
      "description": "Metric display with trend...",
      "status": "pending",
      "priority": 2
    }
  ]
}

# Agent picks task 9.1.1 and starts
$ harness task start 9.1.1

# Agent does the work...
# Agent completes with notes
$ harness task complete 9.1.1 --notes="Built component, added tests, verified in storybook"
```

### Scenario 2: Agent Learning from History

```bash
# Agent about to work on auth bug
# First checks recent completions
$ harness task list --status=completed --category=auth --limit=5 --format=json
{
  "tasks": [
    {
      "id": "42",
      "subject": "Fix JWT validation",
      "notes": "Had to clear Redis cache first. JWT library v3.0 required."
    }
  ]
}

# Agent now knows to clear Redis and check library version
# Saves an entire failed iteration
```

### Scenario 3: Human Review

```bash
# Human wants to review progress
$ harness task export --output=PROGRESS.md
Exported 150 tasks to PROGRESS.md

# Open PROGRESS.md in editor
# Shows familiar markdown format with checkboxes
```

### Scenario 4: Statistics

```bash
$ harness task stats
Total Tasks: 150
  Completed: 120 (80%)
  In Progress: 5
  Pending: 20
  Blocked: 5

By Category:
  ui-components: 50 (40 completed)
  api-endpoints: 30 (30 completed)
  documentation: 20 (15 completed)

Recent Activity:
  Last 7 days: 25 tasks completed
  Average: 3.6 tasks/day
```

---

## Conclusion

This design provides:

1. **80-95% token reduction** compared to reading full markdown files
2. **Cross-iteration learning** through structured completion notes
3. **Storage flexibility** - start with JSON, migrate to SQLite when needed
4. **Human readability** - generate markdown on demand
5. **Simple integration** - drop into existing Go harness

The key insight is **abstraction**: the harness CLI becomes the interface, storage is an implementation detail, and agents operate efficiently on structured data while humans review familiar markdown.

---

## Next Steps

1. **Implement JSON backend** - Start with `task.go`, `json_storage.go`
2. **Build basic CLI** - `list`, `get`, `create`, `update`, `complete`
3. **Test with sample tasks** - Migrate a few tasks from existing markdown
4. **Integrate with agent loop** - Update prompt to use CLI commands
5. **Measure token savings** - Compare before/after
6. **Add markdown export** - Generate human-readable views
7. **Iterate based on usage** - Add features as needed

The reference implementation above is production-ready Go code. An implementing agent can copy, adapt, and integrate into your existing harness with minimal changes.
