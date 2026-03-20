# CLI Specification

---

## 1. Overview

### Purpose

`lorah` uses a subcommand-based CLI. The first argument selects the operation.
All routing is a stdlib `switch` statement with no external dependencies.
`main.go` contains only routing logic — no business logic lives there.

### Goals

- **Subcommand-based**: `lorah <command> [args...]` with unambiguous routing
- **Thin router**: `main.go` is ~60 lines of routing only
- **Flag passthrough**: `run` passes all trailing args to `claude` CLI unchanged
- **Helpful on error**: unknown commands print error + usage; missing args explain correct usage

### Non-Goals

- External flag parsing library (no cobra, no pflag)
- Shell completion
- Typo suggestion for unknown commands

---

## 2. Interface

### Usage

```
lorah <command> [arguments]
```

### Commands

| Command | Arguments                         | Description                             |
| ------- | --------------------------------- | --------------------------------------- |
| `run`   | `<prompt-file> [claude-flags...]` | Run Claude Code CLI in an infinite loop |
| `task`  | `<subcommand> [args...]`          | Manage tasks                            |

### Top-Level Flags

| Flag        | Short            | Description                     |
| ----------- | ---------------- | ------------------------------- |
| `--version` | `-V`, `-version` | Print version and exit 0        |
| `--help`    | `-h`, `-help`    | Show top-level usage and exit 0 |

Top-level flags are only recognized as `os.Args[1]`. They are not parsed anywhere else.

---

## 3. Behavior

### Routing Rules

1. No arguments → print top-level usage, exit 1
2. `--version`, `-version`, `-V` → print `lorah <version>`, exit 0
3. `--help`, `-help`, `-h` → print top-level usage, exit 0
4. `run` → dispatch to `runCmd` with `os.Args[2:]`
5. `task` → dispatch to `taskCmd` with `os.Args[2:]`
6. Anything else → print `Unknown command: <input>` + top-level usage to stderr, exit 1

### Version Output

```
lorah <version>
```

`Version` is `"dev"` by default; injected at build time via `-ldflags '-X main.Version=...'`.

### Top-Level Usage (`lorah` or `lorah --help`)

```
Usage: lorah <command> [arguments]

Simple infinite-loop harness for Claude Code.

Commands:
  run     Run Claude Code CLI in an infinite loop
  task    Manage tasks

Flags:
  -V, --version    Print version and exit
  -h, --help       Show this help message

Run 'lorah <command> --help' for command-specific help.
```

### Run Command Usage (`lorah run --help`)

```
Usage: lorah run <prompt-file> [claude-flags...]

Run Claude Code CLI in a continuous loop with formatted output.
Retries automatically on error with a 5-second delay.

Arguments:
  <prompt-file>      Path to prompt file (required)
  [claude-flags...]  Flags passed directly to claude CLI

Examples:
  lorah run prompt.md
  lorah run task.txt --settings .lorah/settings.json
  lorah run instructions.md --model claude-opus-4-6 --max-turns 50

Flags:
  -h, --help    Show this help message
```

### Task Command Usage (`lorah task --help`)

```
Usage: lorah task <subcommand> [args...] [flags...]

Manage tasks stored in tasks.json.

Subcommands:
  list        List tasks
  get         Get task details
  create      Create a new task
  update      Update a task
  delete      Delete a task
  export      Export tasks to markdown

Flags:
  -h, --help    Show this help message

Run 'lorah task <subcommand> --help' for subcommand-specific help.
```

---

## 4. Router Implementation

### `main.go` Structure

```go
var Version = "dev"

func main()          // subcommand switch on os.Args[1]
func runCmd(args []string)   // extract prompt file + claude flags, call loop.Run()
func taskCmd(args []string)  // dispatch to task subcommand handler
func printUsage()            // top-level help text
func printRunUsage()         // run-specific help text
func printTaskUsage()        // task-specific help text
```

### What Lives in `main.go`

- `var Version` — ldflags target must be in package `main`
- `main()` — the subcommand switch only
- `runCmd()`, `taskCmd()` — argument validation and dispatch
- `printUsage()`, `printRunUsage()`, `printTaskUsage()` — help text

Nothing else. No signal handling, no loop logic, no output formatting, no task logic.

### Flag Parsing

Top-level routing uses a stdlib `switch` on `os.Args[1]`. Subcommand flag parsing
(e.g., `--status=pending`, `--format=markdown`) uses `flag.NewFlagSet` — also stdlib.
No external flag parsing library is used anywhere in the codebase.

### `runCmd` Behavior

```go
func runCmd(args []string)
```

1. If `len(args) == 0` → print run usage, exit 1
2. If `args[0]` is `--help`, `-help`, or `-h` → print run usage, exit 0
3. `promptFile = args[0]`, `claudeFlags = args[1:]`
4. Call `loop.Run(promptFile, claudeFlags)`

### `taskCmd` Behavior

```go
func taskCmd(args []string)
```

1. If `len(args) == 0` → print task usage, exit 1
2. If `args[0]` is `--help`, `-help`, or `-h` → print task usage, exit 0
3. Dispatch on `args[0]` (subcommand name) to the appropriate handler in `internal/task`
4. Unknown subcommand → print `Unknown subcommand: <input>` + task usage to stderr, exit 1

---

## 5. Exit Codes

| Code | Meaning                                   |
| ---- | ----------------------------------------- |
| 0    | Success (including `--version`, `--help`) |
| 1    | Runtime error or usage error              |

---

## 6. Examples

```sh
# Version and help
lorah --version
lorah --help

# Run command
lorah run prompt.md
lorah run prompt.md --settings .lorah/settings.json
lorah run prompt.md --model claude-opus-4-6 --max-turns 50
lorah run --help

# Task command
lorah task list
lorah task list --status=pending
lorah task get 1
lorah task update 1 --status=completed --notes="Done"
lorah task --help

# Error cases
lorah unknown              # Unknown command: unknown
lorah run                  # shows run usage, exits 1
lorah task                 # shows task usage, exits 1
```

---

## 7. Related Specifications

- [run.md](run.md) — `run` command behavior and loop lifecycle
- [task.md](task.md) — `task` subcommand behavior and storage
- [output.md](output.md) — output formatting used by `run`
