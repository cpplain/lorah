# Future Work: Bidirectional Transport Interface

## What It Is

A bidirectional `Transport` interface replaces the current `client.RunSession()` request-response model with a persistent, long-lived subprocess connection. Instead of spawning a new Claude CLI process per session, the harness maintains a single subprocess and exchanges stream-JSON messages over stdin/stdout for the duration of a session.

The interface is modeled after the implementation in [lorah](https://github.com/cpplain/lorah/tree/main/internal/transport):

```go
// Transport defines the interface for communicating with the Claude CLI.
type Transport interface {
    // Connect starts the transport. For subprocess transport, this spawns the CLI.
    Connect(ctx context.Context) error

    // Send writes a user message to the CLI.
    Send(ctx context.Context, input string) error

    // Messages returns a channel of parsed messages from the CLI.
    // The channel is closed when the CLI exits or the transport is closed.
    Messages() <-chan messages.Message

    // Close shuts down the transport and releases resources.
    // Safe to call multiple times.
    Close() error
}
```

## Why It Wasn't Done Now

The current `client.RunSession()` function satisfies every current use case: it spawns the CLI with a single prompt, streams output, captures the `ResultMessage`, and returns. Adding a full bidirectional transport would require:

1. Significant structural changes to how prompts flow through the runner
2. A new stdin-writing code path (the CLI must receive `--input-format stream-json`)
3. Careful goroutine lifecycle management (two goroutines, coordinated shutdown)
4. A `sync.Once`-guarded `Close()` to prevent double-close panics

The risk and complexity are not justified by any current requirement. The future work is documented here so the design is preserved and the implementation path is clear when the need arises.

## What It Enables

- **Multi-turn sessions without subprocess overhead** — the CLI process stays alive between turns, avoiding the startup cost of re-spawning for each interaction.
- **Session resumption** — the transport can send multiple user messages and receive multiple assistant responses within a single CLI process, which is required for a true agentic conversation loop.
- **Streaming backpressure control** — callers consume `Messages()` at their own pace; the 100-buffer channel absorbs bursts without blocking the parser goroutine.
- **Bidirectional stdin injection** — the harness can inject new instructions mid-session without killing and restarting the process.

## Interface Shape

The full interface is shown above. The primary concrete implementation is `Subprocess`, which wraps an `*exec.Cmd` and exposes the above methods. A `SubprocessOptions` struct configures the subprocess:

```go
type Options struct {
    CLIPath            string  // Override CLI auto-discovery
    WorkDir            string  // Working directory for the CLI process
    SessionID          string  // Resume a specific session via --resume
    Continue           bool    // Resume the most recent session via --continue
    PermissionMode     string  // "default", "acceptEdits", "plan", "bypassPermissions"
    MaxTurns           int     // Limit agentic turns via --max-turns
    MaxBudgetUSD       float64 // Limit spending via --max-budget-usd (0 = unlimited)
    Model              string  // Override model via --model
    SystemPrompt       string  // Replace the default system prompt
    AppendSystemPrompt string  // Append to the default system prompt
    Env                map[string]string // Additional environment variables
}
```

## Implementation Notes

### Two goroutines

The `Subprocess` implementation uses exactly two goroutines after `Connect()`:

1. **`readStdout`** — reads and parses stream-JSON lines from the CLI's stdout using `messages.NewParser`. Sends parsed `messages.Message` values to `msgChan`. When the CLI exits, calls `cmd.Wait()` and reports a non-zero exit code to `errChan`.
2. **`drainStderr`** — reads and discards lines from the CLI's stderr. This prevents the subprocess from blocking if stderr fills up. (The `--verbose` flag causes the CLI to write debug output to stderr.)

### 100-buffer msgChan

`msgChan` is created with a buffer of 100:

```go
msgChan: make(chan messages.Message, 100),
```

This absorbs bursts of messages without blocking `readStdout`. Callers should consume `Messages()` continuously; if the buffer fills, `readStdout` blocks and eventually the CLI's stdout pipe fills, causing the CLI to stall.

### sync.Once Close

`Close()` uses `sync.Once` to ensure the shutdown sequence runs at most once, regardless of how many goroutines call `Close()` concurrently:

```go
func (s *Subprocess) Close() error {
    s.closeOnce.Do(func() {
        s.closeMu.Lock()
        s.closed = true
        s.closeMu.Unlock()

        if s.stdin != nil {
            s.stdin.Close() // signals EOF to the CLI
        }

        done := make(chan struct{})
        go func() {
            s.wg.Wait()
            close(done)
        }()

        select {
        case <-done:
        case <-time.After(DefaultStreamCloseTimeout):
            if s.cmd != nil && s.cmd.Process != nil {
                s.cmd.Process.Kill() // force-kill after timeout
            }
        }

        close(s.errChan)
    })
    return nil
}
```

### 60-second force-kill

After closing stdin, `Close()` waits up to `DefaultStreamCloseTimeout` (60 seconds) for the goroutines to finish. If the CLI does not exit within that window, the process is force-killed with `cmd.Process.Kill()`. This prevents the harness from hanging indefinitely if the CLI becomes unresponsive.

### CLAUDE_CODE_ENTRYPOINT env var

The subprocess sets `CLAUDE_CODE_ENTRYPOINT=harness` in the CLI environment. This env var allows the Claude CLI to identify that it is being driven by a harness (rather than an interactive terminal), which may affect its behavior (e.g., disabling interactive prompts). The exact string should match whatever the Claude CLI team defines for harness identification.

### CLI args for bidirectional mode

The subprocess must pass `--input-format stream-json` in addition to `--output-format stream-json --verbose`. This tells the CLI to read user messages from stdin as stream-JSON objects rather than raw text:

```go
args := []string{
    "--output-format", "stream-json",
    "--verbose",
    "--input-format", "stream-json",
}
```

## Migration Path from RunSession()

The current `client.RunSession()` function in `internal/client/client.go` is the natural predecessor:

1. Extract `BuildCommand` flags into an `Options` struct (already partially done via `HarnessConfig`).
2. Implement `Subprocess` with `Connect()`, `Send()`, and `Close()` in a new `internal/transport/` package.
3. In `internal/runner/runner.go`, replace the `client.RunSession(ctx, cfg, model, prompt)` call with:
   ```go
   t := transport.NewSubprocess(opts)
   if err := t.Connect(ctx); err != nil { ... }
   defer t.Close()
   if err := t.Send(ctx, prompt); err != nil { ... }
   for msg := range t.Messages() {
       // handle messages
   }
   ```
4. Update `cmd/lorah/integration_test.go` to have the fake claude stub read from stdin (in `--input-format stream-json` mode) and write to stdout.
5. Remove `client.RunSession()` once the transport is in place.

The `internal/messages/` package (types and parser) is already implemented and is compatible with both `RunSession()` and the bidirectional transport — no changes needed there.
