# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Lorah is a configurable harness for long-running autonomous coding agents. It enables multi-phase agent execution by invoking the `claude` CLI subprocess with SDK-native sandbox isolation, error recovery, progress tracking, and MCP server integration. Configuration is driven entirely by `.lorah/config.json`. Distributed as a single self-contained binary with no external runtime dependencies.

## Commands

```bash
# Build binary
go build -o ./bin/lorah .

# Install globally
go install .

# Run commands
lorah init --project-dir ./my-project
lorah verify --project-dir ./my-project
lorah run --project-dir ./my-project
```

For development without installing:

```bash
go run . <command> --project-dir ./my-project
```

For testing, this project uses the standard Go toolchain:

```bash
# Run all tests
go test ./...

# Run tests for the lorah package
go test ./lorah/

# Run a single test
go test ./lorah/ -run TestDefaultValues -v

# Run with race detector
go test -race ./...
```

No linter or formatter beyond `gofmt` is configured. Tests use the standard `testing` package (no external test framework).

## Architecture

The system runs a loop that executes **phases** sequentially. Each phase invokes the `claude` CLI subprocess (no context carryover) with a configured prompt. Phases can be `run_once` (skipped after first completion) and have path-based conditions (`exists:`, `not_exists:`). Session state (completed phases) persists in `.lorah/session.json`.

### Package Structure

```
main.go              CLI entry point and subcommand wiring
test/                End-to-end integration tests
lorah/               Main package - all functionality in one place
  config.go          Load and validate .lorah/config.json
  runner.go          Main agent loop: phase selection, state, error recovery
  client.go          Build and execute claude CLI subprocess
  verify.go          Pre-run setup checks
  tracking.go        Progress monitoring (JSON checklist, notes file, none)
  messages.go        Parse stream-JSON output from claude CLI
  messages_types.go  Message type definitions
  lock.go            PID-based instance locking
  lock_unix.go       Unix-specific lock implementation
  lock_windows.go    Windows-specific lock implementation
  schema.go          Generate configuration JSON schema
  presets.go         Built-in preset configurations
  info.go            Templates, guides, and init scaffolding
  getting-started.md Embedded setup documentation
  templates/         Embedded template files
  *_test.go          Test files alongside implementation
```

### Main Components

**main.go** — CLI entry point. Parses flags, routes to subcommands (run, verify, init, info), and calls functions in the `lorah` package.

**lorah package** — Single package containing all functionality, organized by file:

- **config.go** — Loads `.lorah/config.json` into Go structs (optional - uses reasonable defaults if missing). Resolves `file:prompts/foo.md` references to file contents. Performs minimal validation - Claude CLI validates its own flags and settings.

- **runner.go** — Main agent loop. Manages phase selection, session state persistence, error tracking with exponential backoff, and auto-continue between sessions. Calls `RunSession()` for each session.

- **client.go** — Builds and executes the `claude` CLI subprocess with flags and settings passed through from config. Parses stream-JSON output and returns a `SessionResult`.

- **tracking.go** — Progress monitoring with 3 implementations: `JsonChecklistTracker` (JSON array with boolean `passes` field), `NotesFileTracker` (plain text), `NoneTracker`.

- **verify.go** — Runs setup checks: Go version, CLI available, API connectivity, config valid, file references, directory permissions.

- **messages.go** — Parses newline-delimited JSON from `claude` CLI stdout into typed message structs (system, assistant, result, user).

- **lock.go** — PID-based lock file at `<harnessDir>/harness.lock` to prevent concurrent runs. Detects and clears stale locks from crashed processes.

- **schema.go** — Generates documentation-oriented JSON schema for the config format. Used by `info schema`.

- **presets.go** — Built-in preset network configurations (python, go, rust, web-nodejs, etc.).

- **info.go** — Embeds starter templates (config.json, spec.md, initialization.md, implementation.md). Handles `init` scaffolding and `info` subcommands.

## Key Patterns

- **Flat package structure** — All code lives in a single `lorah` package (~3000 lines), organized into focused files. This follows Go idioms ("start flat") and matches similar-sized CLI tools.
- **Optional config with defaults** — Configuration is optional. Reasonable defaults are used if `.lorah/config.json` is missing. User config is deep-merged over defaults, allowing partial overrides. Only configure what you need to change.
- **Passthrough model** — The `harness` section contains lorah-specific settings. The `claude` section is passed directly to Claude CLI without lorah validation. Claude CLI validates its own flags and settings.
- **Factory functions** — `client.BuildCommand()` encapsulates all CLI flag construction; `tracking.NewTracker()` selects the right implementation.
- **CLI-native security** — Security is enforced through `claude` CLI via the `--settings` flag (sandbox, permissions). See [Claude Code Settings](https://code.claude.com/docs/en/settings) for configuration options.
- **`file:` resolution** — Prompt strings starting with `file:` are resolved relative to the `.lorah/` directory and replaced with file contents during config loading.
- **Atomic file writes** — Temp file + rename pattern for session state and audit log to prevent corruption.
- **Stream processing** — `claude` CLI stdout piped directly to terminal while messages are parsed line-by-line; unknown message types are skipped gracefully.

## Design Principles

**Understand Anthropic's Guidance First**: Before designing any feature, read and understand Anthropic's documentation. They have already solved most agent problems and documented both WHAT to do and WHY. Do not design solutions without first understanding their recommended approach.

Required reading:

- [Agent SDK Overview](https://platform.claude.com/docs/en/agent-sdk/overview)
- [Claude Code Sandboxing](https://www.anthropic.com/engineering/claude-code-sandboxing)
- [Effective Harnesses](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)
- [Building Effective Agents](https://www.anthropic.com/research/building-effective-agents)

## Dependencies

No external runtime dependencies. All functionality uses the Go standard library. The `claude` CLI (separate install) is the only runtime requirement for executing agent sessions.
