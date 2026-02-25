package lorah

import (
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
