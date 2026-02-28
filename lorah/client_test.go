package lorah

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// minimalHarnessConfig returns a HarnessConfig with minimal required fields set.
func minimalHarnessConfig() *HarnessConfig {
	return &HarnessConfig{
		Harness: HarnessSettings{
			ErrorRecovery: ErrorRecoveryConfig{
				MaxConsecutiveErrors:  5,
				InitialBackoffSeconds: 5.0,
				MaxBackoffSeconds:     120.0,
				BackoffMultiplier:     2.0,
				MaxErrorMessageLength: 2000,
			},
		},
		Claude: ClaudeSection{
			Flags: map[string]any{
				"--max-turns": float64(10),
			},
			Settings: map[string]any{
				"model": "claude-sonnet-4-5",
				"permissions": map[string]any{
					"defaultMode": "bypassPermissions",
				},
				"sandbox": map[string]any{
					"enabled": true,
				},
			},
		},
		ProjectDir: "/tmp/test-project",
		HarnessDir: "/tmp/test-project/.lorah",
	}
}

func TestBuildCommand_BasicFlags(t *testing.T) {
	cfg := minimalHarnessConfig()
	cmd, err := BuildCommand(context.Background(), cfg, "Do something")
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	args := cmd.Args // Args[0] is the binary name
	argsStr := strings.Join(args, " ")

	// Check required flags
	tests := []struct {
		flag string
		want bool
	}{
		{"--output-format", true},
		{"stream-json", true},
		{"--verbose", true},
		{"--add-dir", true},
		{"--max-turns", true},
		{"--settings", true},
		{"Do something", true},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			contains := strings.Contains(argsStr, tt.flag)
			if contains != tt.want {
				t.Errorf("args %q: contains %q = %v, want %v", argsStr, tt.flag, contains, tt.want)
			}
		})
	}
}

func TestBuildCommand_WorkingDirectory(t *testing.T) {
	cfg := minimalHarnessConfig()
	cfg.ProjectDir = "/tmp/my-project"

	cmd, err := BuildCommand(context.Background(), cfg, "test prompt")
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	if cmd.Dir != "/tmp/my-project" {
		t.Errorf("cmd.Dir = %q, want %q", cmd.Dir, "/tmp/my-project")
	}
}

func TestBuildCommand_SettingsFlag(t *testing.T) {
	cfg := minimalHarnessConfig()
	cfg.Claude.Settings["model"] = "claude-opus-4-6"
	if sandbox, ok := cfg.Claude.Settings["sandbox"].(map[string]any); ok {
		sandbox["enabled"] = true
	}

	cmd, err := BuildCommand(context.Background(), cfg, "test")
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	// Find --settings flag and its value
	args := cmd.Args
	var settingsValue string
	for i, arg := range args {
		if arg == "--settings" && i+1 < len(args) {
			settingsValue = args[i+1]
			break
		}
	}

	// Verify the settings JSON contains the model
	if !strings.Contains(settingsValue, "claude-opus-4-6") {
		t.Errorf("expected settings to contain 'claude-opus-4-6', got %q", settingsValue)
	}
}

func TestBuildCommand_MaxTurns(t *testing.T) {
	cfg := minimalHarnessConfig()
	cfg.Claude.Flags["--max-turns"] = float64(500)

	cmd, err := BuildCommand(context.Background(), cfg, "test")
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	argsStr := strings.Join(cmd.Args, " ")
	if !strings.Contains(argsStr, "--max-turns") {
		t.Errorf("expected --max-turns in args: %q", argsStr)
	}
	if !strings.Contains(argsStr, "500") {
		t.Errorf("expected 500 in args: %q", argsStr)
	}
}

func TestBuildCommand_PromptIsLastArg(t *testing.T) {
	cfg := minimalHarnessConfig()
	prompt := "This is my test prompt"

	cmd, err := BuildCommand(context.Background(), cfg, prompt)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	args := cmd.Args
	if len(args) == 0 {
		t.Fatal("expected non-empty args")
	}

	lastArg := args[len(args)-1]
	if lastArg != prompt {
		t.Errorf("expected prompt %q as last arg, got %q", prompt, lastArg)
	}
}

func TestBuildCommand_EnvSet(t *testing.T) {
	cfg := minimalHarnessConfig()

	cmd, err := BuildCommand(context.Background(), cfg, "test")
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	// Env should be set (not nil)
	if cmd.Env == nil {
		t.Error("expected cmd.Env to be non-nil")
	}

	// Should have at least some environment variables
	if len(cmd.Env) == 0 {
		t.Error("expected cmd.Env to be non-empty")
	}
}

// ─── outputManager ───────────────────────────────────────────────────────────

func TestOutputManager_TagSwitching(t *testing.T) {
	var buf bytes.Buffer
	om := &outputManager{writer: &buf}

	// First lorah message should print header
	om.printLorah("first lorah message\n")
	output := buf.String()
	if !strings.Contains(output, "==> LORAH\n") {
		t.Error("First lorah message should print LORAH header")
	}
	if !strings.Contains(output, "first lorah message") {
		t.Error("First lorah message should contain message text")
	}

	buf.Reset()

	// Second lorah message should NOT print header
	om.printLorah("second lorah message\n")
	output = buf.String()
	if strings.Contains(output, "==> LORAH\n") {
		t.Error("Second consecutive lorah message should not print LORAH header")
	}
	if !strings.Contains(output, "second lorah message") {
		t.Error("Second lorah message should contain message text")
	}

	buf.Reset()

	// Claude message should print CLAUDE header
	om.printClaude("claude response")
	output = buf.String()
	if !strings.Contains(output, "==> CLAUDE\n") {
		t.Error("First claude message should print CLAUDE header")
	}
	if !strings.Contains(output, "claude response") {
		t.Error("First claude message should contain message text")
	}

	buf.Reset()

	// Back to lorah should print header again
	om.printLorah("back to lorah\n")
	output = buf.String()
	if !strings.Contains(output, "==> LORAH\n") {
		t.Error("Switching back to lorah should print LORAH header")
	}
	if !strings.Contains(output, "back to lorah") {
		t.Error("Message should contain text")
	}

	buf.Reset()

	// Thinking message should print CLAUDE (thinking) header
	om.printThinking("considering the approach...")
	output = buf.String()
	if !strings.Contains(output, "==> CLAUDE (thinking)\n") {
		t.Error("First thinking message should print CLAUDE (thinking) header")
	}
	if !strings.Contains(output, "considering the approach...") {
		t.Error("First thinking message should contain message text")
	}

	buf.Reset()

	// Second thinking message should NOT print header
	om.printThinking("more thinking...")
	output = buf.String()
	if strings.Contains(output, "==> CLAUDE (thinking)\n") {
		t.Error("Second consecutive thinking message should not print CLAUDE (thinking) header")
	}
	if !strings.Contains(output, "more thinking...") {
		t.Error("Second thinking message should contain message text")
	}

	buf.Reset()

	// After claude response, tool should print its own header
	om.printTool("BASH", "ls -la")
	output = buf.String()
	if !strings.Contains(output, "BASH") {
		t.Error("Tool should print tool name header")
	}
	if !strings.Contains(output, "ls -la") {
		t.Error("Tool should print content")
	}
	if om.lastSource != "tool" {
		t.Errorf("lastSource should be 'tool', got %q", om.lastSource)
	}

	buf.Reset()

	// Tool with empty content should still print header (no content line)
	om.printTool("READ", "")
	output = buf.String()
	if !strings.Contains(output, "READ") {
		t.Error("Tool with empty content should print tool name header")
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("Tool with empty content should print only header line, got %d lines", len(lines))
	}
}

func TestOutputManager_InitialState(t *testing.T) {
	var buf bytes.Buffer
	om := &outputManager{writer: &buf}

	// Very first message should print tag
	om.printClaude("first message ever")
	output := buf.String()
	if !strings.Contains(output, "==> CLAUDE\n") {
		t.Error("Very first message should print tag header")
	}
}

// ─── formatToolUse ───────────────────────────────────────────────────────────

func TestFormatToolUse(t *testing.T) {
	tests := []struct {
		name  string
		tool  string
		input map[string]any
		want  string
	}{
		{
			name:  "Bash command",
			tool:  "Bash",
			input: map[string]any{"command": "ls -la"},
			want:  "ls -la",
		},
		{
			name:  "Read file_path",
			tool:  "Read",
			input: map[string]any{"file_path": "/tmp/foo.txt"},
			want:  "/tmp/foo.txt",
		},
		{
			name:  "Edit file_path",
			tool:  "Edit",
			input: map[string]any{"file_path": "/tmp/bar.go", "old_string": "foo", "new_string": "bar"},
			want:  "/tmp/bar.go",
		},
		{
			name:  "Write file_path",
			tool:  "Write",
			input: map[string]any{"file_path": "/tmp/new.txt", "content": "hello"},
			want:  "/tmp/new.txt",
		},
		{
			name:  "Grep pattern",
			tool:  "Grep",
			input: map[string]any{"pattern": "TODO", "path": "/src"},
			want:  "TODO",
		},
		{
			name:  "Glob pattern",
			tool:  "Glob",
			input: map[string]any{"pattern": "**/*.go"},
			want:  "**/*.go",
		},
		{
			name:  "WebFetch url",
			tool:  "WebFetch",
			input: map[string]any{"url": "https://example.com", "prompt": "Extract title"},
			want:  "https://example.com",
		},
		{
			name:  "Task description",
			tool:  "Task",
			input: map[string]any{"description": "Run tests", "subagent_type": "test-runner"},
			want:  "Run tests",
		},
		{
			name:  "Unknown tool",
			tool:  "UnknownTool",
			input: map[string]any{"foo": "bar"},
			want:  "",
		},
		{
			name:  "Missing parameter",
			tool:  "Bash",
			input: map[string]any{"other": "value"},
			want:  "",
		},
		{
			name:  "Wrong type for parameter",
			tool:  "Read",
			input: map[string]any{"file_path": 123},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatToolUse(tt.tool, tt.input)
			if got != tt.want {
				t.Errorf("formatToolUse(%q, %v) = %q, want %q", tt.tool, tt.input, got, tt.want)
			}
		})
	}
}

// ─── spinner ─────────────────────────────────────────────────────────────────

func TestSpinner_TerminalDetection(t *testing.T) {
	// Test with bytes.Buffer (not a terminal)
	var buf bytes.Buffer
	s := newSpinner(&buf)
	if s != nil {
		t.Error("Spinner should be nil for non-terminal writer (bytes.Buffer)")
	}
}

func TestSpinner_NilSafe(t *testing.T) {
	var buf bytes.Buffer
	s := newSpinner(&buf) // Not a terminal, will be nil

	// These operations should be no-ops and not panic
	s.start()
	s.stop()

	// Buffer should be empty (no escape codes written)
	if buf.Len() > 0 {
		t.Errorf("Nil spinner should not write to buffer, got: %q", buf.String())
	}
}
