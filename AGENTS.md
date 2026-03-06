# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Lorah is a simple infinite-loop harness for long-running autonomous coding agents. It runs Claude Code CLI in a continuous loop, parsing stream-JSON output and formatting it nicely. The agent manages its own workflow - Lorah just provides the loop, error recovery, and output formatting. Follows the [Ralph pattern](https://ghuntley.com/ralph/). Distributed as a single self-contained binary with no external runtime dependencies.

## Commands

```bash
# Build binary
go build -o ./bin/lorah .

# Install globally
go install .

# Run Lorah
lorah PROMPT.md [claude-flags...]
```

All arguments after `PROMPT.md` are passed directly to Claude CLI:

```bash
# With settings file
lorah PROMPT.md --settings .lorah/settings.json

# With specific model
lorah PROMPT.md --model claude-opus-4-6

# With multiple flags
lorah PROMPT.md --settings settings.json --max-turns 50 --verbose
```

For development:

```bash
go run . PROMPT.md [claude-flags...]
```

No tests currently - the implementation is simple enough to verify manually. No linter or formatter beyond `gofmt`.

## Architecture

Lorah is a single-file Go program that implements a simple infinite loop:

```
1. Execute: claude -p --output-format=stream-json --verbose [user-flags...] PROMPT.md
2. Parse stream-JSON line-by-line from stdout
3. Format output with color-coded sections and tool activity
4. On error: print error, sleep 5s, retry
5. On success: continue immediately to next iteration
6. Repeat forever until Ctrl+C
```

### File Structure

```
main.go              Single file containing everything
  - Constants        ANSI color codes, retry delay
  - Message Types    stream-JSON message/content block structs
  - Main Loop        Infinite execution loop with signal handling
  - Claude Exec      Build command, pipe I/O, run subprocess
  - Message Parsing  Parse newline-delimited JSON into typed structs
  - Output Format    Color-coded section headers, tool activity display
go.mod               Go module definition
go.sum               Dependency checksums (none - stdlib only)
```

### Main Sections

**Constants** — ANSI color codes for terminal output and retry delay configuration.

**Helper Functions** — `printSection()` outputs labeled sections with color formatting.

**Main Loop** — Entry point:

- Parses CLI args: `PROMPT.md` and optional Claude CLI flags
- Handles `--version` and `--help` flags
- Sets up signal handler for graceful Ctrl+C shutdown
- Runs infinite loop calling `runClaude()` with error recovery

**Claude Execution** — `runClaude()` function:

- Opens prompt file and pipes it to stdin
- Builds `claude` command with `-p`, `--output-format=stream-json`, `--verbose`, and user flags
- Streams stdout to `printMessages()` for real-time output
- Returns error if command fails

**Output Formatting** — `printMessages()` function:

- Scans stdout line-by-line (newline-delimited JSON)
- Parses each line as `map[string]any` for flexibility
- Gracefully skips malformed/unknown message types for forward compatibility
- Displays color-coded section headers: `==> Claude`, `==> Lorah`, `==> Error`
- Shows tool activity: `==> Bash`, `==> Read`, etc. with relevant input displayed
- Extracts relevant input params (command, file_path, pattern, url, etc.) based on tool name
- Truncates long tool inputs to 3 lines for readability

## Key Patterns

- **Single file** — Entire program in one `main.go` file. No packages, no navigation overhead. Readable in one sitting.
- **No configuration** — Zero config files. Prompts are plain markdown. Claude CLI flags passed through directly.
- **Ralph pattern** — Simple infinite loop. Agent decides workflow. Harness provides loop + nice output. Inspired by [Geoffrey Huntley's Ralph](https://ghuntley.com/ralph/).
- **Agent-driven** — No phase orchestration. Agent reads codebase, decides what to do next, makes progress iteratively.
- **Filesystem as state** — No session files. Git commits show progress. Agent reads files to understand context.
- **Flag passthrough** — All args after `PROMPT.md` passed directly to `claude` CLI. User has full control.
- **CLI-native security** — Security enforced through Claude CLI `--settings` flag (sandbox, permissions). See [Claude Code Settings](https://code.claude.com/docs/en/settings).
- **Stream processing** — stdout parsed line-by-line as newline-delimited JSON; unknown types skipped gracefully for forward compatibility.
- **Simple error handling** — Fixed 5-second retry delay on errors. No exponential backoff complexity.
- **Color-coded output** — Section headers and tool activity formatted with ANSI colors for readability.

## Design Principles

**Ralph Philosophy**: The agent is smart enough to manage its own workflow. Don't orchestrate - provide a simple loop and trust the model.

**Radical Simplicity**: Every line of code is overhead. The simplest solution that works is the best solution. Prefer deleting code over adding it.

**Agent is in Control**: The harness provides the loop and nice output. The agent reads the codebase, decides what to do, and makes progress. No phase management needed.

**No Ceremony**: No config files, session state, lock files, or scaffolding commands. Just a prompt file and a loop.

Required reading:

- [Ralph by Geoffrey Huntley](https://ghuntley.com/ralph/) - The pattern this implementation follows
- [Building Effective Agents](https://www.anthropic.com/research/building-effective-agents) - Agent design principles from Anthropic
- [Agent SDK Overview](https://platform.claude.com/docs/en/agent-sdk/overview) - Understanding agent capabilities
- [Claude Code Sandboxing](https://www.anthropic.com/engineering/claude-code-sandboxing) - Security model

## Dependencies

No external runtime dependencies. All functionality uses the Go standard library. The `claude` CLI (separate install) is the only runtime requirement for executing agent sessions.
