# Lorah

## Project Overview

Lorah is a simple infinite-loop harness for long-running autonomous coding agents. It runs Claude Code CLI in a continuous loop, parsing stream-JSON output and formatting it nicely. Includes a task management system for structured agent workflow coordination. The agent manages its own workflow — Lorah just provides the loop, error recovery, output formatting, and task tracking. Follows the Ralph pattern. Distributed as a single self-contained binary with no external runtime dependencies.

## Commands

```bash
make build                         # Build binary
go run . run PROMPT.md             # Development run
lorah run PROMPT.md [flags...]     # Run loop (all flags after prompt passed to claude CLI)
lorah task <subcommand> [args...]  # Task management
```

Use TDD — write tests before implementation. Use `make fmt` and `make lint`.

## Architecture

```
main.go                   CLI router: subcommand dispatch, help text, version
internal/loop/
  loop.go                 Run() entry point, signal handling, infinite loop
  claude.go               Subprocess execution
  output.go               Stream-JSON parsing and formatted output
  constants.go            ANSI colors, buffer size, retry delay
internal/task/
  task.go                 Core types: Phase, Section, Task, TaskStatus, TaskList, Filter
  storage.go              Storage interface
  json_storage.go         JSONStorage implementation (tasks.json)
  format.go               Output formatters: json, markdown
  cmd.go                  CLI subcommand handlers
docs/design/              Design specifications (authoritative reference)
```

## Design Principles

**Ralph Philosophy**: The agent is smart enough to manage its own workflow. Don't orchestrate — provide a simple loop and trust the model.

**Radical Simplicity**: Every line of code is overhead. The simplest solution that works is the best solution. Prefer deleting code over adding it.

**Agent is in Control**: The harness provides the loop and nice output. The agent reads the codebase, decides what to do, and makes progress. No phase management needed.

**No Ceremony**: No config files, session state, lock files, or scaffolding commands. Just a prompt file and a loop.

**Filesystem as State**: No session files. Git commits show progress. Agent reads files to understand context.

**Design Specifications**: Authoritative design docs live in `docs/design/`. When in doubt about intended behavior, consult the specs: `cli.md`, `run.md`, `output.md`, `task.md`.

## Dependencies

No external runtime dependencies. All functionality uses the Go standard library. The `claude` CLI (separate install) is the only runtime requirement.
