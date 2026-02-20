# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Lorah is a configurable harness for long-running autonomous coding agents. It enables multi-phase agent execution by invoking the `claude` CLI subprocess with SDK-native sandbox isolation, error recovery, progress tracking, and MCP server integration. Configuration is driven entirely by `.lorah/config.json`. Distributed as a single self-contained binary with no external runtime dependencies.

## Commands

```bash
# Build binary
go build ./cmd/lorah

# Install globally
go install ./cmd/lorah

# Run commands
lorah init --project-dir ./my-project
lorah verify --project-dir ./my-project
lorah run --project-dir ./my-project
```

For development without installing:

```bash
go run ./cmd/lorah <command> --project-dir ./my-project
```

For testing, this project uses the standard Go toolchain:

```bash
# Run all tests
go test ./...

# Run tests for a single package
go test ./internal/config/

# Run a single test
go test ./internal/config/ -run TestDefaultValues -v

# Run with race detector
go test -race ./...
```

No linter or formatter beyond `gofmt` is configured. Tests use the standard `testing` package (no external test framework).

## Architecture

The system runs a loop that executes **phases** sequentially. Each phase invokes the `claude` CLI subprocess (no context carryover) with a configured prompt. Phases can be `run_once` (skipped after first completion) and have path-based conditions (`exists:`, `not_exists:`). Session state (completed phases) persists in `.lorah/session.json`.

### Package Structure

```
cmd/lorah/    CLI entry point and subcommand wiring
internal/
  config/             Load and validate .lorah/config.json
  runner/             Main agent loop: phase selection, state, error recovery
  client/             Build and execute claude CLI subprocess
  verify/             Pre-run setup checks
  tracking/           Progress monitoring (JSON checklist, notes file, none)
  messages/           Parse stream-JSON output from claude CLI
  lock/               PID-based instance locking
  schema/             Generate configuration JSON schema
  presets/            Built-in preset configurations
  info/               Templates, guides, and init scaffolding
```

### Package Dependency Flow

**cmd/lorah/main.go** orchestrates all packages:

- **config** — loads and validates configuration (foundational, no internal imports)
- **runner** — executes the agent loop (uses config, client, tracking, lock)
- **verify** — runs setup checks (uses config)
- **info** — scaffolding and documentation commands (uses config, schema, presets)

**config** — Loads `.lorah/config.json` into Go structs. Resolves `file:prompts/foo.md` references to file contents. Validates permission modes, tracking types, phase names, and file paths.

**runner** — Main agent loop. Manages phase selection, session state persistence, error tracking with exponential backoff, and auto-continue between sessions. Invokes `client.RunSession()` for each session.

**client** — Builds and executes the `claude` CLI subprocess with flags for model, permission mode, tools, MCP servers, sandbox settings, and prompt. Parses stream-JSON output and returns a `SessionResult`.

**tracking** — Progress monitoring with 3 implementations: `JsonChecklistTracker` (JSON array with boolean `passes` field), `NotesFileTracker` (plain text), `NoneTracker`.

**verify** — Runs setup checks: Go version, CLI available, API connectivity, config exists, config valid, file references, MCP commands, directory permissions.

**messages** — Parses newline-delimited JSON from `claude` CLI stdout into typed message structs (system, assistant, result, user).

**lock** — PID-based lock file at `<harnessDir>/harness.lock` to prevent concurrent runs. Detects and clears stale locks from crashed processes.

**schema** — Generates documentation-oriented JSON schema for the config format. Used by `info schema`.

**presets** — Built-in preset network configurations (python, go, rust, web-nodejs, etc.).

**info** — Embeds starter templates (config.json, spec.md, initialization.md, implementation.md). Handles `init` scaffolding and `info` subcommands.

## Key Patterns

- **Struct-based config** — All configuration is modeled as nested Go structs with defaults applied via merge-over-defaults pattern and explicit validation functions in `config`.
- **Factory functions** — `client.BuildCommand()` encapsulates all CLI flag construction; `tracking.NewTracker()` selects the right implementation.
- **CLI-native security** — Security is enforced through `claude` CLI flags (`--permission-mode`, `--allowedTools`, `--disallowedTools`, `--settings` for sandbox), not application-layer hooks.
- **`file:` resolution** — Prompt strings starting with `file:` are resolved relative to the `.lorah/` directory and replaced with file contents during config loading.
- **MCP environment variables** — MCP server `env` values support `${VAR}` syntax for environment variable expansion.
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
