# Run Command Specification

---

## 1. Overview

### Purpose

The `run` command executes Claude Code CLI in an infinite loop, piping a prompt file
to each invocation and displaying formatted stream-JSON output in real-time.
The loop runs until interrupted; the agent manages its own workflow.

### Goals

- **Infinite loop**: runs until Ctrl+C or SIGTERM, no iteration limit
- **Error recovery**: failed iterations sleep and retry automatically
- **Signal handling**: first Ctrl+C/SIGTERM stops after current loop; second triggers immediate exit
- **Flag passthrough**: all arguments after the prompt file passed to `claude` unchanged
- **Real-time output**: stream-JSON parsed and displayed as it arrives

### Non-Goals

- Configurable retry strategy (exponential backoff, max retries, jitter)
- Session persistence across process restarts
- Multiple concurrent Claude invocations
- Iteration limits or timeout

---

## 2. Interface

### CLI

```
lorah run <prompt-file> [claude-flags...]
```

### Go Function

```go
// Run starts the infinite Claude Code CLI execution loop.
// It handles signal interrupts and retries on error.
// Run does not return under normal operation.
func Run(promptFile string, claudeFlags []string)
```

`Run` is the only exported symbol from `internal/loop`.

### Argument Handling (in `main.go`)

Argument validation for `run` is handled by `runCmd` in `main.go`.
See [cli.md section 4](cli.md#4-router-implementation) for the full behavior.
`runCmd` calls `loop.Run(promptFile, claudeFlags)` after validation.

---

## 3. Loop Lifecycle

### Constants

```go
retryDelay = 5 * time.Second
```

Defined in `internal/loop/constants.go` alongside the output constants.

### Iteration Flow

```go
func Run(promptFile string, claudeFlags []string)
```

1. Create cancellable context and set up signal handler
2. Initialize `iteration = 0`
3. Loop:
   1. Increment `iteration`
   2. `printSection("Lorah", colorBlue, "Starting loop...")`
   3. Call `runClaude(ctx, promptFile, claudeFlags)`
   4. On error: print error to stderr, sleep `retryDelay` (5s), continue
   5. On success: `printSection("Lorah", colorBlue, "Loop completed successfully")`, continue immediately

### Signal Handling

- A goroutine listens on a buffered channel for `os.Interrupt` and `syscall.SIGTERM`
- On **first signal** (either `SIGINT` or `SIGTERM`):
  1. Set a `stopping` flag (e.g. `atomic.Bool`)
  2. Print blank line
  3. `printSection("Lorah", colorBlue, "Received interrupt, stopping after current loop...")`
  4. Do **not** cancel the context — let the current subprocess finish naturally
- On **second signal** (any, while `stopping` is set):
  1. Print blank line
  2. `printSection("Lorah", colorBlue, "Received second interrupt, shutting down...")`
  3. Call `cancel()` to propagate cancellation to any running subprocess
  4. `os.Exit(0)`
- After each successful or failed iteration, the loop checks `stopping`; if set, `os.Exit(0)`
- Signal handling lives in `loop.go`, not `main.go`, because it is part of the loop lifecycle

### Error Display

Printed to stderr on a failed iteration:

```
<red>⏺ <bold>Error<reset>
<error message>

Retrying in 5s...

```

---

## 4. Claude Code CLI Execution

### Signature

```go
func runClaude(ctx context.Context, promptFile string, flags []string) error
```

### Subprocess Configuration

1. Open `promptFile` for reading; return error if it cannot be opened
2. Build argument list: `-p`, `--output-format`, `stream-json`, `--verbose`, then `flags...`
3. `exec.CommandContext(ctx, "claude", args...)`
4. `cmd.Stdin = file` — prompt file contents piped to stdin
5. `cmd.Stderr = os.Stderr` — claude stderr passes through directly
6. `cmd.Env = os.Environ()` — inherit full environment

### Execution Steps

1. Create stdout pipe via `cmd.StdoutPipe()`
2. `cmd.Start()`
3. `printMessages(stdout)` — blocking call that reads and formats stream-JSON in real-time
4. `cmd.Wait()` — waits for subprocess to exit
5. Return any error from `cmd.Wait()`

### Error Sources

| Source                        | Error Prefix                            |
| ----------------------------- | --------------------------------------- |
| Prompt file not readable      | `"opening prompt file: "`               |
| stdout pipe creation failure  | `"creating stdout pipe: "`              |
| `claude` not found in PATH    | `"starting Claude Code CLI: "`          |
| Claude Code CLI non-zero exit | `"Claude Code CLI exited with error: "` |

---

## 5. Package Structure

```
internal/loop/
  loop.go       -- Run() exported entry point, signal handling, infinite loop
  claude.go     -- runClaude() subprocess execution
  output.go     -- printMessages(), printSection() formatting
  constants.go  -- ANSI colors, maxBufferSize, retryDelay
```

`internal/` ensures the package is not importable outside the module.
A single `loop` package (not multiple sub-packages) because `printMessages`
is only called from `runClaude`, and `runClaude` is only called from `Run`.
Splitting further would require exporting symbols with no reason to be public.

---

## 6. Examples

```sh
# Basic usage
lorah run prompt.md

# With Claude settings file
lorah run prompt.md --settings .lorah/settings.json

# With specific model and turn limit
lorah run prompt.md --model claude-opus-4-6 --max-turns 50

# With multiple flags
lorah run prompt.md --settings settings.json --model claude-opus-4-6 --verbose

# Help
lorah run --help

# First Ctrl+C → prints "Received interrupt, stopping after current loop...", exits 0 after iteration
# Second Ctrl+C → prints "Received second interrupt, shutting down...", exits 0 immediately
```

---

## 7. Related Specifications

- [cli.md](cli.md) — CLI routing and argument extraction for the `run` command
- [output.md](output.md) — `printMessages` and `printSection` behavior
- [task.md](task.md) — task management the agent uses within the run loop
