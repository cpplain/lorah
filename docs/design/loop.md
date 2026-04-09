# Loop Specification

---

## 1. Overview

### Purpose

`lorah` executes Claude Code CLI in an infinite loop, piping a prompt file to each
invocation and displaying formatted stream-JSON output in real-time. The loop runs
until interrupted; the agent manages its own workflow.

### Goals

- **Subcommand-free**: `lorah <prompt-file> [claude-flags...]`, unambiguous routing
- **Thin router**: `main.go` is stdlib `switch` only, no business logic
- **Infinite loop**: runs until Ctrl+C or SIGTERM, no iteration limit
- **Error recovery**: failed iterations sleep and retry automatically
- **Signal handling**: first Ctrl+C/SIGTERM stops after current loop; second triggers immediate exit
- **Flag passthrough**: all arguments after the prompt file passed to `claude` unchanged
- **Real-time output**: stream-JSON parsed and displayed as it arrives
- **Helpful on error**: missing arguments explain correct usage

### Non-Goals

- Subcommands (the binary does exactly one thing)
- External flag parsing library (no cobra, no pflag)
- Shell completion or typo suggestion
- Configurable retry strategy (exponential backoff, max retries, jitter)
- Session persistence across process restarts
- Multiple concurrent Claude invocations
- Iteration limits or timeout

---

## 2. Interface

### CLI

```
lorah <prompt-file> [claude-flags...]
```

### Top-Level Flags

| Flag        | Short | Description                     |
| ----------- | ----- | ------------------------------- |
| `--version` | `-V`  | Print version and exit 0        |
| `--help`    | `-h`  | Show top-level usage and exit 0 |

`--version` and `--help` are only recognized as `os.Args[1]`. They are not parsed anywhere else.

### Go Function

```go
// Run starts the infinite Claude Code CLI execution loop.
// It handles signal interrupts and retries on error.
// Run does not return under normal operation.
func Run(promptFile string, claudeFlags []string)
```

`Run` is the only exported symbol from `internal/loop`.

---

## 3. Routing

### Routing Rules

1. No arguments → print top-level usage, exit 1
2. `args[0]` is `--version`, `-version`, or `-V` → print `lorah <version>`, exit 0
3. `args[0]` is `--help`, `-help`, or `-h` → print top-level usage, exit 0
4. `args[0]` begins with `-` and did not match rules 2 or 3 → print `Unknown flag: <arg>` followed by top-level usage, exit 1
5. Otherwise → `loop.Run(args[0], args[1:])`

Any `args[0]` that does not begin with `-` is treated as a prompt file path. If the
file cannot be opened, `loop.runClaude` returns an error and the loop's normal error
recovery applies (see §4).

### Version Output

```
lorah <version>
```

`Version` is `"dev"` by default; injected at build time via `-ldflags '-X main.Version=...'`.

### Top-Level Usage (`lorah` or `lorah --help`)

```
Usage: lorah <prompt-file> [claude-flags...]

Simple infinite-loop harness for Claude Code.

Runs Claude Code CLI in a continuous loop with formatted output.
Retries automatically on error with a 5-second delay.

Arguments:
  <prompt-file>      Path to prompt file (required)
  [claude-flags...]  Flags passed directly to claude CLI

Examples:
  lorah prompt.md
  lorah prompt.md --settings .lorah/settings.json
  lorah prompt.md --model claude-opus-4-6 --max-turns 50

Flags:
  -V, --version    Print version and exit
  -h, --help       Show this help message
```

### `main.go` Structure

```go
var Version = "dev"

func main()         // os.Exit(route(os.Args[1:], Version, loop.Run))
func route(...)     // the 5 routing rules above
func printUsage()   // top-level help text
```

Nothing else. No signal handling, no loop logic, no output formatting — those live
in `internal/loop`.

### Flag Parsing

Top-level routing uses a stdlib `switch` on `args[0]`. No external flag parsing
library is used anywhere in the codebase.

---

## 4. Loop Lifecycle

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
   2. `printSection("Lorah", colorBlue, fmt.Sprintf("Starting loop %d...", iteration))`
   3. Call `runClaude(ctx, promptFile, claudeFlags)`
   4. On error: print error to stderr, sleep `retryDelay` (5s), continue
   5. On success: `printSection("Lorah", colorBlue, fmt.Sprintf("Loop %d completed successfully", iteration))`, continue immediately

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

## 5. Claude Code CLI Execution

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

## 6. Package Structure

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

## 7. Exit Codes

| Code | Meaning                                   |
| ---- | ----------------------------------------- |
| 0    | Success (including `--version`, `--help`) |
| 1    | Runtime error or usage error              |

---

## 8. Examples

Each example is an invocation or event followed by the observable outcome.

### 1. No arguments

Input: `lorah`

Outcome: Top-level usage printed to stderr. Process exits 1.

### 2. Version flag

Input: `lorah --version` (also `-V`, `-version`)

Outcome: Prints `lorah <version>` to stdout. Process exits 0.

### 3. Help flag

Input: `lorah --help` (also `-h`, `-help`)

Outcome: Top-level usage printed to stdout. Process exits 0.

### 4. Basic loop

Input: `lorah prompt.md`

Outcome: `loop.Run("prompt.md", [])` begins iterating. Each iteration prints `Starting loop N...`, pipes `prompt.md` to `claude -p --output-format stream-json --verbose`, prints `Loop N completed successfully` on success, and proceeds immediately to the next iteration.

### 5. Flag passthrough

Input: `lorah prompt.md --model claude-opus-4-6 --max-turns 50`

Outcome: `loop.Run("prompt.md", ["--model", "claude-opus-4-6", "--max-turns", "50"])`. The subprocess argv is `claude -p --output-format stream-json --verbose --model claude-opus-4-6 --max-turns 50`.

### 6. Missing prompt file

Input: `lorah missing.md`

Outcome: `runClaude` returns `opening prompt file: open missing.md: no such file or directory`. An error section is printed to stderr, followed by `Retrying in 5s...`. The loop sleeps 5s and retries — the same error repeats until the file appears or the process is interrupted.

### 7. Subprocess non-zero exit

Input: `claude` subprocess exits with code 2 mid-iteration.

Outcome: `cmd.Wait()` returns an error. The error section `Claude Code CLI exited with error: exit status 2` is printed to stderr. The loop sleeps 5s and retries.

### 8. First interrupt during iteration

Input: Ctrl+C while a `claude` subprocess is running.

Outcome: Prints blank line, then `Received interrupt, stopping after current loop...`. The subprocess continues to completion (context is not cancelled). After the iteration finishes, the loop checks the `stopping` flag and calls `os.Exit(0)`.

### 9. Second interrupt during iteration

Input: Ctrl+C a second time while the subprocess is still running from example 8.

Outcome: Prints blank line, then `Received second interrupt, shutting down...`. `cancel()` is called to signal any running subprocess, and `os.Exit(0)` is called immediately.

### 10. Unknown flag as first argument

Input: `lorah --nope`

Outcome: `args[0]` begins with `-` but matches neither `--version` nor `--help`. Prints `Unknown flag: --nope` followed by top-level usage to stderr. Process exits 1.

### 11. Version injected at build time

Input: Binary built with `-ldflags '-X main.Version=0.6.0'`, then `lorah --version`.

Outcome: Prints `lorah 0.6.0` to stdout. Process exits 0.

### 12. Successful iteration

Input: `lorah prompt.md` with a valid prompt and a `claude` subprocess that completes normally.

Outcome: Iteration N prints `Starting loop N...`, streams formatted output from `printMessages`, prints `Loop N completed successfully`, and begins iteration N+1 with no delay.

---

## 9. Related Specifications

- [output.md](output.md) — `printMessages` and `printSection` behavior
