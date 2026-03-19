# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Subcommand-based CLI: `lorah <command> [arguments]` with `run` and `task` commands
- Task management system (`lorah task`) with 8 subcommands: list, get, create, update, start, complete, export, stats
- JSON-based task storage in `tasks.json` with Storage interface for future backend swaps
- Two task output formats: json, markdown (default); `--flat` flag on `list` for flat bullet output

### Changed

- **BREAKING**: CLI changed to `lorah run <prompt-file> [claude-flags...]` (was `lorah <prompt-file> [claude-flags...]`)
- **BREAKING**: Multi-package architecture replaces single-file structure (`main.go` router + `internal/loop/` + `internal/task/`)

## [0.4.0] - 2026-03-09

### Changed

- Output sections now use `⏺` symbol instead of `==>`
- Added blank line spacing between output sections for improved readability
- Tool inputs now truncate to 1 line instead of 3 lines to prevent trailing blank lines

### Fixed

- Truncated tool inputs now display line count (e.g., "... +5 lines") instead of just "..."
- Removed excessive blank lines caused by trailing whitespace in Claude output

## [0.3.0] - 2026-03-05

### Added

- Infinite loop execution following [Ralph pattern](https://ghuntley.com/ralph/)
- Direct flag passthrough to Claude Code CLI

### Changed

- **BREAKING**: CLI interface changed to `lorah PROMPT.md [claude-flags...]` (was `lorah run --project-dir`)
- **BREAKING**: Now follows Ralph technique instead of Anthropic's [effective harnesses for long-running agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)
- **BREAKING**: Agent manages its own workflow autonomously (removed multi-phase orchestration, config system, session state, progress tracking)
- Error retry uses fixed 5-second delay (was exponential backoff)

## [0.2.0] - 2026-02-24

### Added

- Configuration is now optional with sensible defaults — only configure what you need to change

### Changed

- **BREAKING**: Configuration format changed to a split structure with `harness` (lorah settings) and `claude` (passthrough to Claude Code CLI) sections. Existing configs require migration—run `lorah info template` to see the new format.
- Updated documentation to document two-phase execution model with fixed file names (initialization and implementation phases)
- Renamed `docs/setup-guide.md` to `docs/getting-started.md`
- Renamed review workflow prompts from `inventory.md`/`fix.md` to `initialization.md`/`implementation.md`
- Migrated examples to new config format; review workflow now uses defaults

### Fixed

- Fixed broken link in getting started guide

## [0.1.0] - 2026-02-20

### Added

- Initial release of Lorah, a configurable harness for long-running autonomous coding agents
- CLI commands: `run`, `verify`, `init`, and `info` (with `template`, `schema`, `preset`, `guide` subcommands)
- Multi-phase agent execution with initialization (run-once) and implementation (iterative) phases
- Phase conditions using `exists:` and `not_exists:` path-based rules
- Session state persistence in `.lorah/session.json` for resume capability
- Graceful interrupt handling with automatic session resume
- Error recovery with configurable exponential backoff and circuit breaker
- JSON checklist progress tracking via `tasks.json` with automatic completion detection
- Progress notes file (`progress.md`) for session handoff documentation
- SDK-native sandbox isolation with network, filesystem, and command restrictions
- Permission modes: `default`, `acceptEdits`, `bypassPermissions`, `plan`
- Fine-grained tool allow/deny rules configuration
- MCP (Model Context Protocol) server integration with environment variable expansion
- Built-in network presets: `python`, `go`, `rust`, `web-nodejs`, `read-only`
- PID-based instance locking to prevent concurrent runs
- Project scaffolding via `lorah init` with embedded starter templates
- Pre-flight verification via `lorah verify` (CLI, API, config, files, permissions)
- Configuration JSON schema accessible via `lorah info schema`
- Single self-contained binary with no external runtime dependencies

[unreleased]: https://github.com/cpplain/lorah/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/cpplain/lorah/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/cpplain/lorah/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/cpplain/lorah/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cpplain/lorah/releases/tag/v0.1.0
