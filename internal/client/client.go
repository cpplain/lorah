// Package client builds the exec.Cmd for invoking the claude CLI subprocess.
//
// The Go port runs the claude CLI directly as a subprocess using os/exec,
// rather than using the Python claude-agent-sdk wrapper. This package
// constructs the correct command with all necessary flags derived from
// HarnessConfig.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/cpplain/lorah/internal/config"
	"github.com/cpplain/lorah/internal/messages"
)

// BuildCommand constructs an exec.Cmd for invoking the claude CLI.
//
// The command is built from the HarnessConfig and an optional phase-specific
// model. It sets:
//   - --model: the Claude model to use
//   - --output-format stream-json --verbose: stream-JSON output mode
//   - --permission-mode: how to handle tool permission prompts
//   - --allowedTools / --disallowedTools: builtin tool allow/deny rules
//   - --add-dir: working directory for claude CLI
//   - MCP server configuration
//   - Sandbox flags
//
// The returned Cmd does not set Stdout so that RunSession can capture it
// via StdoutPipe.
func BuildCommand(ctx context.Context, cfg *config.HarnessConfig, model string, prompt string) (*exec.Cmd, error) {
	args := []string{}

	// Model
	args = append(args, "--model", model)

	// Stream-JSON output mode
	args = append(args, "--output-format", "stream-json", "--verbose")

	// Permission mode
	args = append(args, "--permission-mode", cfg.Security.PermissionMode)

	// Working directory for claude CLI
	args = append(args, "--add-dir", cfg.ProjectDir)

	// Allowed tools (builtin)
	for _, tool := range cfg.Tools.Builtin {
		args = append(args, "--allowedTools", tool)
	}

	// MCP servers: add allowed tool patterns and server configs
	for serverName := range cfg.Tools.McpServers {
		args = append(args, "--allowedTools", fmt.Sprintf("mcp__%s__*", serverName))
	}

	// Permission allow rules
	for _, rule := range cfg.Security.Permissions.Allow {
		args = append(args, "--allowedTools", rule)
	}

	// Permission deny rules
	for _, rule := range cfg.Security.Permissions.Deny {
		args = append(args, "--disallowedTools", rule)
	}

	// MCP server configuration
	for serverName, serverCfg := range cfg.Tools.McpServers {
		mcpArg, err := buildMCPServerArg(serverName, serverCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to build MCP server arg for %q: %w", serverName, err)
		}
		args = append(args, "--mcp-config", mcpArg)
	}

	// Settings JSON (includes sandbox configuration)
	settingsJSON, err := buildSettingsJSON(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build settings JSON: %w", err)
	}
	args = append(args, "--settings", settingsJSON)

	// Max turns
	if cfg.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", cfg.MaxTurns))
	}

	// The prompt is the final positional argument
	args = append(args, prompt)

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

// buildMCPServerArg builds the JSON string for a single MCP server config
// passed to claude CLI via --mcp-config.
//
// The claude CLI expects a JSON object mapping server names to their configs.
func buildMCPServerArg(name string, serverCfg config.McpServerConfig) (string, error) {
	// Define the structure for the MCP server configuration
	type mcpServerJSON struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env,omitempty"`
	}

	// Build the full structure: {"<name>": {...}}
	// Note: json.Marshal sorts map keys lexicographically, providing deterministic output.
	fullConfig := map[string]mcpServerJSON{
		name: {
			Command: serverCfg.Command,
			Args:    serverCfg.Args,
			Env:     serverCfg.Env,
		},
	}

	result, err := json.Marshal(fullConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP server config: %w", err)
	}

	return string(result), nil
}

// buildSettingsJSON creates the JSON string for the --settings flag.
//
// The Claude CLI accepts sandbox configuration through a settings JSON object,
// not through individual CLI flags. This function translates the SandboxConfig
// into the expected JSON format documented at:
// https://code.claude.com/docs/en/settings#sandbox-settings
func buildSettingsJSON(cfg *config.HarnessConfig) (string, error) {
	settings := map[string]interface{}{
		"sandbox": map[string]interface{}{
			"enabled":                  cfg.Security.Sandbox.Enabled,
			"autoAllowBashIfSandboxed": cfg.Security.Sandbox.AutoAllowBashIfSandboxed,
			"allowUnsandboxedCommands": cfg.Security.Sandbox.AllowUnsandboxedCommands,
			"excludedCommands":         cfg.Security.Sandbox.ExcludedCommands,
			"network": map[string]interface{}{
				"allowedDomains":    cfg.Security.Sandbox.Network.AllowedDomains,
				"allowLocalBinding": cfg.Security.Sandbox.Network.AllowLocalBinding,
				"allowUnixSockets":  cfg.Security.Sandbox.Network.AllowUnixSockets,
			},
		},
	}

	jsonBytes, err := json.Marshal(settings)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
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
func RunSession(ctx context.Context, cfg *config.HarnessConfig, model, prompt string) (SessionResult, error) {
	fmt.Printf("Sending prompt to Claude...\n\n")

	cmd, err := BuildCommand(ctx, cfg, model, prompt)
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

	// Watch for context cancellation and close stdout to immediately unblock parser.
	// Without this, parser.Next() can remain blocked on I/O even after ctx is cancelled.
	go func() {
		<-ctx.Done()
		stdout.Close()
	}()

	var result SessionResult
	done := make(chan struct{})

	go func() {
		defer close(done)
		parser := messages.NewParser(stdout)
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
				if _, ok := err.(*messages.JSONDecodeError); ok {
					continue
				}
				break
			}
			switch m := msg.(type) {
			case *messages.AssistantMessage:
				for _, block := range m.Content {
					if tb, ok := block.(*messages.TextBlock); ok {
						fmt.Print(tb.Text)
					}
				}
			case *messages.ResultMessage:
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
