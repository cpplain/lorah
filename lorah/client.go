// Package client builds the exec.Cmd for invoking the claude CLI subprocess.
//
// The Go port runs the claude CLI directly as a subprocess using os/exec,
// rather than using the Python claude-agent-sdk wrapper. This package
// constructs the correct command with all necessary flags derived from
// HarnessConfig.
package lorah

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
)

// BuildCommand constructs an exec.Cmd for invoking the claude CLI.
//
// The command is built from the HarnessConfig. It sets:
//   - --output-format stream-json --verbose: stream-JSON output mode (required for parsing)
//   - --add-dir: working directory for claude CLI
//   - config flags: all flags from cfg.Claude.Flags (passthrough to CLI)
//   - --settings: claude settings.json content
//
// The returned Cmd does not set Stdout so that RunSession can capture it
// via StdoutPipe.
func BuildCommand(ctx context.Context, cfg *HarnessConfig, prompt string) (*exec.Cmd, error) {
	settingsJSON, err := cfg.SettingsJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize settings: %w", err)
	}

	args := []string{
		"--output-format", "stream-json",
		"--verbose",
		"--add-dir", cfg.ProjectDir,
	}

	// Add flags from config (passthrough to Claude CLI)
	// Keys are explicit flag names (e.g., "--max-turns"), values are serialized.
	// null values mean include the flag without a value (boolean flags).
	// Sort keys for deterministic ordering.
	keys := make([]string, 0, len(cfg.Claude.Flags))
	for key := range cfg.Claude.Flags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := cfg.Claude.Flags[key]
		if value == nil {
			args = append(args, key)
		} else {
			args = append(args, key, fmt.Sprintf("%v", value))
		}
	}

	// Add settings and prompt
	args = append(args, "--settings", settingsJSON, prompt)

	cmd := exec.CommandContext(ctx, "claude", args...)

	// Set working directory
	cmd.Dir = cfg.ProjectDir

	// Stderr streams to terminal; Stdout is left unset so RunSession can
	// capture it via StdoutPipe.
	cmd.Stderr = os.Stderr

	// Pass through environment, including auth credentials
	cmd.Env = buildEnv()

	return cmd, nil
}

// buildEnv returns the environment variables to pass to the claude subprocess.
//
// It includes all current environment variables plus auth credentials if set.
func buildEnv() []string {
	return os.Environ()
}

// SessionResult holds the parsed outcome of a single Claude CLI session.
type SessionResult struct {
	SessionID    string
	NumTurns     int
	TotalCostUSD *float64
	DurationMs   int
	IsError      bool
	ErrorMsg     string
}

// RunSession executes one Claude CLI session with the given prompt.
//
// It builds the command with stream-JSON output, captures stdout via
// StdoutPipe, and parses messages using the messages package. Text content
// is printed to stdout; the ResultMessage fields are captured into the
// returned SessionResult.
func RunSession(ctx context.Context, cfg *HarnessConfig, prompt string) (SessionResult, error) {
	fmt.Printf("Sending prompt to Claude...\n\n")

	cmd, err := BuildCommand(ctx, cfg, prompt)
	if err != nil {
		return SessionResult{}, fmt.Errorf("failed to build command: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return SessionResult{}, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return SessionResult{}, fmt.Errorf("failed to start claude CLI: %w", err)
	}

	var result SessionResult
	done := make(chan struct{})

	// Watch for context cancellation and close stdout to immediately unblock parser.
	// Without this, parser.Next() can remain blocked on I/O even after ctx is cancelled.
	// Also exit when work completes to avoid goroutine leak when using context.Background().
	go func() {
		select {
		case <-ctx.Done():
			stdout.Close()
		case <-done:
			// Work complete, subprocess has exited
		}
	}()

	go func() {
		defer close(done)
		parser := NewParser(stdout)
		for {
			// Check for context cancellation before reading next message
			if ctx.Err() != nil {
				break
			}

			msg, err := parser.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				// *JSONDecodeError: skip for forward-compatibility
				if _, ok := err.(*JSONDecodeError); ok {
					continue
				}
				break
			}
			switch m := msg.(type) {
			case *AssistantMessage:
				for _, block := range m.Content {
					if tb, ok := block.(*TextBlock); ok {
						fmt.Print(tb.Text)
					}
				}
			case *ResultMessage:
				result.SessionID = m.SessionID
				result.NumTurns = m.NumTurns
				result.TotalCostUSD = m.TotalCostUSD
				result.DurationMs = m.DurationMs
				result.IsError = m.IsError
				if m.IsError {
					result.ErrorMsg = m.Result
				}
			}
		}
	}()

	<-done

	if err := cmd.Wait(); err != nil {
		// If context was cancelled, mark the session as incomplete.
		if ctx.Err() != nil {
			result.IsError = true
			result.ErrorMsg = fmt.Sprintf("session cancelled: %v", ctx.Err())
			return result, nil
		}
		if !result.IsError {
			result.IsError = true
			result.ErrorMsg = fmt.Sprintf("claude CLI exited with error: %v", err)
		}
	}

	return result, nil
}
