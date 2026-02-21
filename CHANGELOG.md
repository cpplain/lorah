# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

- Updated README.md to document two-phase execution model with fixed file names
- Renamed review workflow prompts from `inventory.md`/`fix.md` to `initialization.md`/`implementation.md`
- Updated review workflow README to use initialization/implementation terminology
- Removed deprecated config options from review workflow example (`system_prompt`, `tracking`, `phases`)

### Deprecated

### Removed

### Fixed

- Corrected broken relative link in setup-guide.md (../README.md → ../../README.md)

### Security

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

[unreleased]: https://github.com/cpplain/lorah/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/cpplain/lorah/releases/tag/v0.1.0
