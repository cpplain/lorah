package client

import (
	"context"
	"strings"
	"testing"

	"github.com/cpplain/lorah/internal/config"
)

// minimalConfig returns a HarnessConfig with minimal required fields set.
func minimalConfig() *config.HarnessConfig {
	return &config.HarnessConfig{
		Model:      "claude-sonnet-4-5",
		MaxTurns:   10,
		ProjectDir: "/tmp/test-project",
		HarnessDir: "/tmp/test-project/.lorah",
		Tools: config.ToolsConfig{
			Builtin:    []string{"Read", "Write", "Bash"},
			McpServers: map[string]config.McpServerConfig{},
		},
		Security: config.SecurityConfig{
			PermissionMode: "acceptEdits",
			Sandbox: config.SandboxConfig{
				Enabled:                  true,
				AutoAllowBashIfSandboxed: true,
				AllowUnsandboxedCommands: false,
				ExcludedCommands:         []string{},
				Network: config.SandboxNetworkConfig{
					AllowedDomains:    []string{},
					AllowLocalBinding: false,
					AllowUnixSockets:  []string{},
				},
			},
			Permissions: config.PermissionRulesConfig{
				Allow: []string{},
				Deny:  []string{},
			},
		},
	}
}

func TestBuildCommand_BasicFlags(t *testing.T) {
	cfg := minimalConfig()
	cmd, err := BuildCommand(context.Background(), cfg, "claude-sonnet-4-5", "Do something")
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
		{"--model", true},
		{"claude-sonnet-4-5", true},
		{"--output-format", true},
		{"stream-json", true},
		{"--verbose", true},
		{"--permission-mode", true},
		{"acceptEdits", true},
		{"--add-dir", true},
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
	cfg := minimalConfig()
	cfg.ProjectDir = "/tmp/my-project"

	cmd, err := BuildCommand(context.Background(), cfg, cfg.Model, "test prompt")
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	if cmd.Dir != "/tmp/my-project" {
		t.Errorf("cmd.Dir = %q, want %q", cmd.Dir, "/tmp/my-project")
	}
}

func TestBuildCommand_AllowedTools(t *testing.T) {
	cfg := minimalConfig()
	cfg.Tools.Builtin = []string{"Read", "Write", "Bash", "Glob"}

	cmd, err := BuildCommand(context.Background(), cfg, cfg.Model, "test")
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	args := cmd.Args
	argsStr := strings.Join(args, " ")

	for _, tool := range cfg.Tools.Builtin {
		if !strings.Contains(argsStr, tool) {
			t.Errorf("expected tool %q in args: %q", tool, argsStr)
		}
	}
}

func TestBuildCommand_MCPServers(t *testing.T) {
	cfg := minimalConfig()
	cfg.Tools.McpServers = map[string]config.McpServerConfig{
		"filesystem": {
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
			Env:     map[string]string{},
		},
	}

	cmd, err := BuildCommand(context.Background(), cfg, cfg.Model, "test")
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	argsStr := strings.Join(cmd.Args, " ")

	// Should include MCP tool pattern
	if !strings.Contains(argsStr, "mcp__filesystem__*") {
		t.Errorf("expected MCP tool pattern in args: %q", argsStr)
	}

	// Should include --mcp-config
	if !strings.Contains(argsStr, "--mcp-config") {
		t.Errorf("expected --mcp-config in args: %q", argsStr)
	}
}

func TestBuildCommand_SettingsFlag(t *testing.T) {
	cfg := minimalConfig()
	cfg.Security.Sandbox = config.SandboxConfig{
		Enabled:                  true,
		AutoAllowBashIfSandboxed: true,
		AllowUnsandboxedCommands: false,
		ExcludedCommands:         []string{"curl", "wget"},
		Network: config.SandboxNetworkConfig{
			AllowedDomains:    []string{"api.anthropic.com"},
			AllowLocalBinding: true,
			AllowUnixSockets:  []string{"/tmp/my.sock"},
		},
	}

	cmd, err := BuildCommand(context.Background(), cfg, cfg.Model, "test")
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	// Should use --settings flag (not individual --sandbox flags)
	args := cmd.Args
	var settingsValue string
	for i, arg := range args {
		if arg == "--settings" && i+1 < len(args) {
			settingsValue = args[i+1]
			break
		}
	}

	if settingsValue == "" {
		t.Fatalf("expected --settings flag in args: %v", args)
	}

	// Settings value should be valid JSON containing sandbox config
	if !strings.Contains(settingsValue, `"sandbox"`) {
		t.Errorf("expected sandbox key in settings JSON: %q", settingsValue)
	}
	if !strings.Contains(settingsValue, `"enabled":true`) {
		t.Errorf("expected enabled:true in settings JSON: %q", settingsValue)
	}

	// Should NOT have old-style --sandbox flags
	argsStr := strings.Join(args, " ")
	for _, badFlag := range []string{"--sandbox ", "--no-sandbox", "--sandbox-auto-allow-bash", "--sandbox-exclude-command"} {
		if strings.Contains(argsStr, badFlag) {
			t.Errorf("unexpected legacy sandbox flag %q in args: %q", badFlag, argsStr)
		}
	}
}

func TestBuildCommand_PermissionRules(t *testing.T) {
	cfg := minimalConfig()
	cfg.Security.Permissions = config.PermissionRulesConfig{
		Allow: []string{"Bash(git:*)", "Read(/tmp/*)"},
		Deny:  []string{"Bash(rm:*)"},
	}

	cmd, err := BuildCommand(context.Background(), cfg, cfg.Model, "test")
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	args := cmd.Args

	// Find --allowedTools and --disallowedTools flags
	var allowedTools []string
	var disallowedTools []string
	for i, arg := range args {
		if arg == "--allowedTools" && i+1 < len(args) {
			allowedTools = append(allowedTools, args[i+1])
		}
		if arg == "--disallowedTools" && i+1 < len(args) {
			disallowedTools = append(disallowedTools, args[i+1])
		}
	}

	// Check allow rules are in allowedTools
	for _, rule := range cfg.Security.Permissions.Allow {
		found := false
		for _, tool := range allowedTools {
			if tool == rule {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("allow rule %q not found in --allowedTools: %v", rule, allowedTools)
		}
	}

	// Check deny rules are in disallowedTools
	for _, rule := range cfg.Security.Permissions.Deny {
		found := false
		for _, tool := range disallowedTools {
			if tool == rule {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("deny rule %q not found in --disallowedTools: %v", rule, disallowedTools)
		}
	}
}

func TestBuildCommand_MaxTurns(t *testing.T) {
	cfg := minimalConfig()
	cfg.MaxTurns = 500

	cmd, err := BuildCommand(context.Background(), cfg, cfg.Model, "test")
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

func TestBuildCommand_DifferentModel(t *testing.T) {
	cfg := minimalConfig()
	cfg.Model = "claude-sonnet-4-5"

	// Pass a different model (per-phase override)
	cmd, err := BuildCommand(context.Background(), cfg, "claude-opus-4-5", "test")
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	// Find the model argument
	args := cmd.Args
	var modelValue string
	for i, arg := range args {
		if arg == "--model" && i+1 < len(args) {
			modelValue = args[i+1]
			break
		}
	}

	if modelValue != "claude-opus-4-5" {
		t.Errorf("expected model claude-opus-4-5, got %q", modelValue)
	}
}

func TestBuildCommand_PromptIsLastArg(t *testing.T) {
	cfg := minimalConfig()
	prompt := "This is my test prompt"

	cmd, err := BuildCommand(context.Background(), cfg, cfg.Model, prompt)
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

func TestBuildSettingsJSON_SandboxEnabled(t *testing.T) {
	cfg := minimalConfig()
	cfg.Security.Sandbox = config.SandboxConfig{
		Enabled:                  true,
		AutoAllowBashIfSandboxed: false,
		AllowUnsandboxedCommands: true,
		ExcludedCommands:         []string{"curl"},
		Network: config.SandboxNetworkConfig{
			AllowedDomains:    []string{"example.com"},
			AllowLocalBinding: false,
			AllowUnixSockets:  []string{},
		},
	}

	result, err := buildSettingsJSON(cfg)
	if err != nil {
		t.Fatalf("buildSettingsJSON() error = %v", err)
	}

	tests := []struct {
		name    string
		substr  string
		present bool
	}{
		{"sandbox key", `"sandbox"`, true},
		{"enabled true", `"enabled":true`, true},
		{"autoAllowBash false", `"autoAllowBashIfSandboxed":false`, true},
		{"allowUnsandboxed true", `"allowUnsandboxedCommands":true`, true},
		{"excludedCommands curl", `"curl"`, true},
		{"allowedDomains example.com", `"example.com"`, true},
		{"allowLocalBinding false", `"allowLocalBinding":false`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contains := strings.Contains(result, tt.substr)
			if contains != tt.present {
				t.Errorf("JSON %q: contains %q = %v, want %v", result, tt.substr, contains, tt.present)
			}
		})
	}
}

func TestBuildSettingsJSON_SandboxDisabled(t *testing.T) {
	cfg := minimalConfig()
	cfg.Security.Sandbox = config.SandboxConfig{
		Enabled: false,
	}

	result, err := buildSettingsJSON(cfg)
	if err != nil {
		t.Fatalf("buildSettingsJSON() error = %v", err)
	}

	if !strings.Contains(result, `"enabled":false`) {
		t.Errorf("expected enabled:false in JSON: %q", result)
	}
}

func TestBuildMCPServerArg(t *testing.T) {
	t.Run("simple server no env", func(t *testing.T) {
		serverCfg := config.McpServerConfig{
			Command: "npx",
			Args:    []string{"-y", "@mcp/server"},
			Env:     map[string]string{},
		}

		result, err := buildMCPServerArg("myserver", serverCfg)
		if err != nil {
			t.Fatalf("buildMCPServerArg() error = %v", err)
		}

		// Should contain the server name
		if !strings.Contains(result, "myserver") {
			t.Errorf("expected server name in result: %q", result)
		}
		// Should contain command
		if !strings.Contains(result, "npx") {
			t.Errorf("expected command 'npx' in result: %q", result)
		}
		// Should contain args
		if !strings.Contains(result, "@mcp/server") {
			t.Errorf("expected arg @mcp/server in result: %q", result)
		}
		// Should NOT have env when empty
		if strings.Contains(result, `"env"`) {
			t.Errorf("did not expect 'env' key in result: %q", result)
		}
	})

	t.Run("server with env", func(t *testing.T) {
		serverCfg := config.McpServerConfig{
			Command: "node",
			Args:    []string{"server.js"},
			Env:     map[string]string{"API_KEY": "secret123"},
		}

		result, err := buildMCPServerArg("myserver", serverCfg)
		if err != nil {
			t.Fatalf("buildMCPServerArg() error = %v", err)
		}

		// Should contain env
		if !strings.Contains(result, `"env"`) {
			t.Errorf("expected 'env' key in result: %q", result)
		}
		if !strings.Contains(result, "API_KEY") {
			t.Errorf("expected API_KEY in result: %q", result)
		}
	})
}

func TestBuildCommand_EnvSet(t *testing.T) {
	cfg := minimalConfig()

	cmd, err := BuildCommand(context.Background(), cfg, cfg.Model, "test")
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
