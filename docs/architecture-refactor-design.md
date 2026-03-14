# Architecture Refactor Design: Multi-Package Structure with Subcommands

**Purpose:** Restructure Lorah from a single-file program to a multi-package architecture with required subcommands, enabling clean integration of new features (task management, context kill, etc.) without compromising the loop harness's simplicity.

**Author:** Design session 2026-03-13
**Status:** Design Complete - Ready for Implementation

---

## Table of Contents

1. [Problem Statement](#problem-statement)
2. [Design Goals](#design-goals)
3. [Subcommand Design](#subcommand-design)
4. [Package Structure](#package-structure)
5. [CLI Router (`main.go`)](#cli-router-maingo)
6. [Loop Package (`internal/loop/`)](#loop-package-internalloop)
7. [Documentation Updates](#documentation-updates)
8. [Implementation Checklist](#implementation-checklist)
9. [Verification](#verification)

---

## Problem Statement

### Current State

Lorah is a single file (`main.go`, 252 lines) with four functions:

```
main()           CLI argument parsing, signal handling, infinite loop
runClaude()      Claude CLI subprocess execution
printMessages()  Stream-JSON parsing and output formatting
printSection()   Color-coded section header helper
```

This works well for its current scope. As new features are added, however, all logic lives in the same file with no clear separation of concerns:

- Adding task management commands would mix task CRUD logic into a loop harness file
- Adding context kill monitoring would entangle subprocess management with output formatting
- Testing any subsystem in isolation is impossible without touching everything
- `main.go` becomes harder to read in one sitting — violating its core design principle

### Key Questions

1. **Extensibility:** How do we add subcommands (task, future features) without polluting the loop harness?
2. **Simplicity:** How do we maintain the "readable in one sitting" quality for each concern?
3. **Compatibility:** What is the right time to make a breaking CLI change?
4. **Routing:** How do we implement subcommand routing without external dependencies?

---

## Design Goals

1. **Required subcommands** — Clean CLI routing with no ambiguity in argument parsing
2. **Thin `main.go`** — CLI routing only; no business logic in the entry point
3. **Isolated loop package** — Loop logic remains simple and self-contained in its own package
4. **No new dependencies** — Subcommand routing via stdlib `switch` statement
5. **Breaking change now** — Pre-1.0 is the right time; costs are lowest before wider adoption
6. **Future-ready** — New subcommands (`task`, `stats`, etc.) add a `case` and a handler; nothing else changes

---

## Subcommand Design

### Before (current)

```bash
lorah PROMPT.md [claude-flags...]
```

### After

```bash
lorah run PROMPT.md [claude-flags...]
lorah --version
lorah --help
```

### Why Required Subcommands?

**Simpler routing:** No ambiguity detection (is this a subcommand or a filename?). The first argument is always a subcommand or a flag.

**Cleaner help:** `lorah --help` shows available commands. `lorah run --help` shows run-specific usage.

**Extensible:** `lorah task list`, `lorah task get`, etc. can coexist without any routing changes.

**Right time:** Lorah is pre-1.0 and intentionally held there until the CLI API is stable. This is the lowest-cost moment for a breaking change.

### Routing Rules

1. No arguments → print usage, exit 1
2. `--version`, `-version`, `-V` → print version, exit 0
3. `--help`, `-help`, `-h` → print top-level usage, exit 0
4. `run` → dispatch to run handler with remaining args
5. Anything else → print "Unknown command" + usage, exit 1

---

## Package Structure

### Target Layout

```
main.go                          # CLI router only (~60 lines)
internal/
  loop/
    loop.go                      # Run() — exported entry point
    claude.go                    # runClaude() — subprocess execution
    output.go                    # printMessages(), printSection() — formatting
    constants.go                 # ANSI colors, buffer size, retry delay
```

`internal/` ensures loop packages are not importable outside the module. Future packages (`internal/tasks/`) are created when those features are built — no empty placeholders.

### Why a Single `internal/loop` Package?

`output.go` and `claude.go` are tightly coupled to the loop — `printMessages` is only called from `runClaude`, and `runClaude` is only called from the loop. Splitting them into separate packages would require exporting symbols that have no reason to be public. One package, multiple files for readability.

### Why `constants.go` Not `output_constants.go`?

`retryDelay` is used by the loop (`loop.go`), not output. Constants are shared across the package; a single file without a narrow qualifier is cleaner.

---

## CLI Router (`main.go`)

### Structure

```go
package main

import (
    "fmt"
    "os"

    "github.com/cpplain/lorah/internal/loop"
)

// Version is set via ldflags during build. Default is "dev" for local builds.
var Version = "dev"

func main() {
    if len(os.Args) < 2 {
        printUsage()
        os.Exit(1)
    }

    switch os.Args[1] {
    case "--version", "-version", "-V":
        fmt.Printf("lorah %s\n", Version)
        os.Exit(0)
    case "--help", "-help", "-h":
        printUsage()
        os.Exit(0)
    case "run":
        runCmd(os.Args[2:])
    default:
        fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
        printUsage()
        os.Exit(1)
    }
}

func runCmd(args []string) {
    if len(args) == 0 {
        printRunUsage()
        os.Exit(1)
    }
    if args[0] == "--help" || args[0] == "-help" || args[0] == "-h" {
        printRunUsage()
        os.Exit(0)
    }
    promptFile := args[0]
    claudeFlags := args[1:]
    loop.Run(promptFile, claudeFlags)
}
```

### Help Text

**Top-level (`lorah --help`):**

```
Usage: lorah <command> [arguments]

Simple infinite-loop harness for Claude Code.

Commands:
  run    Run Claude CLI in an infinite loop

Flags:
  -V, --version    Print version and exit
  -h, --help       Show this help message

Run 'lorah <command> --help' for command-specific help.
```

**Run command (`lorah run --help`):**

```
Usage: lorah run <prompt-file> [claude-flags...]

Run Claude CLI in a continuous loop with formatted output.
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

### What Stays in `main.go`

- `var Version = "dev"` — ldflags target is `main.Version`; must stay in package main
- `main()` — subcommand switch only
- `runCmd()` — argument extraction, delegates to `loop.Run()`
- `printUsage()` — top-level help
- `printRunUsage()` — run-specific help

**Nothing else.** No signal handling, no loop logic, no output formatting.

---

## Loop Package (`internal/loop/`)

### Public API

The package exports exactly one function:

```go
// Run starts the infinite Claude CLI execution loop.
// It handles signal interrupts and retries on error.
// Run does not return under normal operation.
func Run(promptFile string, claudeFlags []string)
```

Everything else is unexported.

### `loop.go` — Entry Point

Contains the exported `Run()` function. Responsibilities:

- Signal handler setup (Ctrl+C → graceful shutdown)
- Infinite loop with iteration tracking
- Error handling: print error, sleep `retryDelay`, continue
- Success: print completion, continue immediately

Signal handling lives here (not in `main.go`) because it is part of the loop lifecycle. `main.go` dispatches to `Run()` and does not get control back.

```go
func Run(promptFile string, claudeFlags []string) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigCh
        fmt.Println()
        printSection("Lorah", colorBlue, "Received interrupt, shutting down...")
        cancel()
        os.Exit(0)
    }()

    iteration := 0
    for {
        iteration++
        printSection("Lorah", colorBlue, "Starting loop...")

        if err := runClaude(ctx, promptFile, claudeFlags); err != nil {
            fmt.Fprintf(os.Stderr, "\n%s⏺ %sError%s\n", colorRed, colorBold, colorReset)
            fmt.Fprintf(os.Stderr, "%v\n\n", err)
            fmt.Fprintf(os.Stderr, "Retrying in %v...\n\n", retryDelay)
            time.Sleep(retryDelay)
            continue
        }

        printSection("Lorah", colorBlue, "Loop completed successfully")
    }
}
```

### `claude.go` — Subprocess Execution

Contains `runClaude()`. Moves verbatim from `main.go:114-152`. Builds the `claude -p --output-format=stream-json --verbose [flags...]` command, pipes the prompt file to stdin, streams stdout to `printMessages()`.

### `output.go` — Formatting

Contains `printSection()` and `printMessages()`. Moves verbatim from `main.go:38-45` and `main.go:155-251`. Parses newline-delimited stream-JSON, displays color-coded section headers and tool activity.

### `constants.go` — Package Constants

```go
const (
    colorReset = "\033[0m"
    colorGreen = "\033[32m"
    colorBlue  = "\033[34m"
    colorBold  = "\033[1m"
    colorRed   = "\033[31m"

    maxBufferSize = 1024 * 1024 // 1MB buffer for JSON parsing
    retryDelay    = 5 * time.Second
)
```

---

## Documentation Updates

### CHANGELOG.md

Add under `[Unreleased]`:

```markdown
### Changed

- **BREAKING**: CLI now requires subcommands. Use `lorah run PROMPT.md` instead of `lorah PROMPT.md`.
- Refactored from single-file to multi-package architecture (`internal/loop/`).
```

### AGENTS.md (and CLAUDE.md symlink)

- Update all usage examples to `lorah run PROMPT.md [claude-flags...]`
- Update file structure section to show `main.go` + `internal/loop/` layout
- Update "Main Sections" to describe new package responsibilities
- Update example commands in the "Commands" section

### README.md

- Update usage examples from `lorah PROMPT.md` to `lorah run PROMPT.md`

### Makefile

No changes required. The ldflags target (`-X 'main.Version=...'`) references `main.Version` which remains in `main.go`.

---

## Implementation Checklist

### Phase 1: Package Extraction

- [ ] Create `internal/loop/constants.go` — move constants from `main.go:26-35`
- [ ] Create `internal/loop/output.go` — move `printSection()` and `printMessages()` from `main.go`
- [ ] Create `internal/loop/claude.go` — move `runClaude()` from `main.go`
- [ ] Create `internal/loop/loop.go` — create exported `Run()` with signal handling and infinite loop
- [ ] Verify `go build` succeeds

### Phase 2: CLI Router

- [ ] Rewrite `main.go` as thin CLI router
- [ ] Implement `printUsage()` with subcommand listing
- [ ] Implement `printRunUsage()` with run-specific help
- [ ] Implement `runCmd()` dispatching to `loop.Run()`
- [ ] Verify `go build` succeeds

### Phase 3: Documentation

- [ ] Update `CHANGELOG.md`
- [ ] Update `AGENTS.md` (also updates `CLAUDE.md` via symlink)
- [ ] Update `README.md`

---

## Verification

After implementation, verify the following:

```bash
# Build succeeds
go build -o ./bin/lorah .

# Static analysis clean
go vet ./...

# Top-level flags
lorah --version          # prints version, exits 0
lorah --help             # shows subcommand list, exits 0
lorah                    # shows usage, exits 1

# Run subcommand
lorah run --help                           # shows run-specific help, exits 0
lorah run PROMPT.md                        # runs loop
lorah run PROMPT.md --settings file.json   # flag passthrough works
lorah run PROMPT.md --model claude-opus-4-6 --max-turns 50

# Error cases
lorah unknown            # prints "Unknown command: unknown" + usage, exits 1
lorah run                # shows run usage, exits 1

# Ctrl+C during run
# → prints "Received interrupt, shutting down...", exits 0
```
