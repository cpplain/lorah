// Package lorah provides the harness for long-running autonomous coding agents.
//
// This file (client.go) builds the exec.Cmd for invoking the claude CLI subprocess.
// The Go port runs the claude CLI directly as a subprocess using os/exec,
// rather than using the Python claude-agent-sdk wrapper. This file
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
	"strings"
	"sync"
	"time"
)

const (
	colorReset = "\033[0m"
	colorGreen = "\033[32m"
	colorBlue  = "\033[34m"
	colorBold  = "\033[1m"
)

// isTerminal returns true if the file is a terminal (TTY).
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// outputManager manages output source tagging and synchronization.
type outputManager struct {
	mu         sync.Mutex
	lastSource string
	writer     io.Writer
}

// printLorah prints harness output with Lorah tag.
func (om *outputManager) printLorah(format string, args ...any) {
	om.mu.Lock()
	defer om.mu.Unlock()

	fmt.Fprint(om.writer, colorBlue+"==> "+colorReset+colorBold+"Lorah"+colorReset+"\n")
	fmt.Fprintf(om.writer, format, args...)
	om.lastSource = "lorah"
}

// printClaude prints Claude output with Claude tag.
func (om *outputManager) printClaude(text string) {
	om.mu.Lock()
	defer om.mu.Unlock()

	fmt.Fprint(om.writer, colorBlue+"==> "+colorReset+colorBold+"Claude"+colorReset+"\n")
	fmt.Fprintf(om.writer, "%s\n", text)
	om.lastSource = "claude"
}

// printThinking prints Claude's extended thinking with tag.
func (om *outputManager) printThinking(text string) {
	om.mu.Lock()
	defer om.mu.Unlock()

	fmt.Fprint(om.writer, colorBlue+"==> "+colorReset+colorBold+"Claude (thinking)"+colorReset+"\n")
	fmt.Fprintf(om.writer, "%s\n", text)
	om.lastSource = "thinking"
}

// printTool prints tool invocation output.
func (om *outputManager) printTool(toolName string, content string) {
	om.mu.Lock()
	defer om.mu.Unlock()

	if content == "" {
		fmt.Fprintf(om.writer, colorGreen+"==> "+colorReset+colorBold+"%s"+colorReset+"\n", toolName)
	} else {
		fmt.Fprintf(om.writer, colorGreen+"==> "+colorReset+colorBold+"%s"+colorReset+"\n%s\n", toolName, content)
	}
	om.lastSource = "tool"
}

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
	cmd.Env = os.Environ()

	return cmd, nil
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

// spinner displays a rotating activity indicator on terminals.
// A nil spinner is safe to use - all methods are no-ops.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type spinner struct {
	writer  io.Writer
	stopCh  chan struct{}
	doneCh  chan struct{}
	running bool
}

// newSpinner creates a spinner for the given writer.
// Returns nil if the writer is not a terminal, making the spinner a no-op.
func newSpinner(w io.Writer) *spinner {
	f, ok := w.(*os.File)
	if !ok || !isTerminal(f) {
		return nil
	}
	return &spinner{writer: w}
}

// start begins the spinning animation with hidden cursor.
// No-op if spinner is nil or already running.
func (s *spinner) start() {
	if s == nil || s.running {
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})

	fmt.Fprint(s.writer, "\033[?25l")          // hide cursor
	fmt.Fprint(s.writer, spinnerFrames[0]+" ") // initial frame

	go s.animate()
}

// animate runs the spinner animation loop.
func (s *spinner) animate() {
	defer close(s.doneCh)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	frame := 0
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			frame = (frame + 1) % len(spinnerFrames)
			fmt.Fprint(s.writer, "\r"+spinnerFrames[frame]+" ")
		}
	}
}

// stop halts the spinner, clears the line, and restores cursor visibility.
// No-op if spinner is nil or not running.
func (s *spinner) stop() {
	if s == nil || !s.running {
		return
	}
	close(s.stopCh)
	<-s.doneCh
	s.running = false

	fmt.Fprint(s.writer, "\033[2K\r") // clear line
	fmt.Fprint(s.writer, "\033[?25h") // show cursor
}

// formatToolUse extracts the key parameter for a tool invocation.
func formatToolUse(name string, input map[string]any) string {
	// Extract the most relevant parameter for each tool type
	switch name {
	case "Bash":
		if cmd, ok := input["command"].(string); ok {
			return cmd
		}
	case "Read", "Edit", "Write":
		if path, ok := input["file_path"].(string); ok {
			return path
		}
	case "Grep", "Glob":
		if pattern, ok := input["pattern"].(string); ok {
			return pattern
		}
	case "WebFetch":
		if url, ok := input["url"].(string); ok {
			return url
		}
	case "Task":
		if desc, ok := input["description"].(string); ok {
			return desc
		}
	}
	return ""
}

// RunSession executes one Claude CLI session with the given prompt.
//
// It builds the command with stream-JSON output, captures stdout via
// StdoutPipe, and parses messages using the messages package. Text content
// is printed to stdout; the ResultMessage fields are captured into the
// returned SessionResult.
func RunSession(ctx context.Context, cfg *HarnessConfig, prompt string, om *outputManager) (SessionResult, error) {
	om.printLorah("Sending prompt to Claude...\n")

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
		sp := newSpinner(os.Stdout)
		defer sp.stop() // Ensure cleanup on all exit paths
		sp.start()
		for {
			// Check for context cancellation before reading next message
			if ctx.Err() != nil {
				break
			}

			msg, err := parser.Next()
			if err != nil {
				sp.stop()
				break
			}

			switch m := msg.(type) {
			case *AssistantMessage:
				for _, block := range m.Content {
					sp.stop()
					switch b := block.(type) {
					case *TextBlock:
						om.printClaude(b.Text)
					case *ThinkingBlock:
						om.printThinking(b.Thinking)
					case *ToolUseBlock:
						toolName := b.Name
						if len(toolName) > 0 {
							toolName = strings.ToUpper(toolName[:1]) + strings.ToLower(toolName[1:])
						}
						content := formatToolUse(b.Name, b.Input)
						if lines := strings.Split(content, "\n"); len(lines) > 3 {
							content = strings.Join(lines[:3], "\n") + fmt.Sprintf("\n... +%d lines", len(lines)-3)
						}
						om.printTool(toolName, content)
					}
					sp.start()
				}
			case *UserMessage:
				// Tool results suppressed
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
