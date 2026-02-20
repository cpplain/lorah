package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeTempConfig writes a config JSON file to a temp harness directory and
// returns the project directory path.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	harnessDir := filepath.Join(dir, ConfigDirName)
	if err := os.MkdirAll(harnessDir, 0o755); err != nil {
		t.Fatalf("failed to create harness dir: %v", err)
	}
	configFile := filepath.Join(harnessDir, "config.json")
	if err := os.WriteFile(configFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	return dir
}

// minimalConfig returns a minimal valid config JSON.
func minimalConfig() string {
	return `{}`
}

// TestDefaultValues verifies that default values are applied when no overrides
// are present in the config.
func TestDefaultValues(t *testing.T) {
	dir := writeTempConfig(t, minimalConfig())

	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.Model != DefaultModel {
		t.Errorf("Model = %q, want %q", cfg.Model, DefaultModel)
	}
	if cfg.MaxTurns != 1000 {
		t.Errorf("MaxTurns = %d, want 1000", cfg.MaxTurns)
	}
	if cfg.MaxIterations != nil {
		t.Errorf("MaxIterations = %v, want nil", cfg.MaxIterations)
	}
	if cfg.AutoContinueDelay != 3 {
		t.Errorf("AutoContinueDelay = %d, want 3", cfg.AutoContinueDelay)
	}
	if cfg.Security.PermissionMode != string(PermissionModeAcceptEdits) {
		t.Errorf("PermissionMode = %q, want %q", cfg.Security.PermissionMode, PermissionModeAcceptEdits)
	}
	if !cfg.Security.Sandbox.Enabled {
		t.Error("Sandbox.Enabled = false, want true")
	}
	if !cfg.Security.Sandbox.AutoAllowBashIfSandboxed {
		t.Error("AutoAllowBashIfSandboxed = false, want true")
	}
	if cfg.ErrorRecovery.MaxConsecutiveErrors != 5 {
		t.Errorf("MaxConsecutiveErrors = %d, want 5", cfg.ErrorRecovery.MaxConsecutiveErrors)
	}
	if cfg.ErrorRecovery.InitialBackoffSeconds != 5.0 {
		t.Errorf("InitialBackoffSeconds = %f, want 5.0", cfg.ErrorRecovery.InitialBackoffSeconds)
	}
	if cfg.ErrorRecovery.MaxBackoffSeconds != 120.0 {
		t.Errorf("MaxBackoffSeconds = %f, want 120.0", cfg.ErrorRecovery.MaxBackoffSeconds)
	}
	if cfg.ErrorRecovery.BackoffMultiplier != 2.0 {
		t.Errorf("BackoffMultiplier = %f, want 2.0", cfg.ErrorRecovery.BackoffMultiplier)
	}
}

// TestDefaultBuiltinTools verifies the default built-in tools list.
func TestDefaultBuiltinTools(t *testing.T) {
	dir := writeTempConfig(t, minimalConfig())
	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	expected := DefaultBuiltinTools
	if len(cfg.Tools.Builtin) != len(expected) {
		t.Fatalf("Builtin tools count = %d, want %d", len(cfg.Tools.Builtin), len(expected))
	}
	for i, tool := range expected {
		if cfg.Tools.Builtin[i] != tool {
			t.Errorf("Builtin[%d] = %q, want %q", i, cfg.Tools.Builtin[i], tool)
		}
	}
}

// TestConfigFileNotFound verifies that a missing config file produces a ConfigError.
func TestConfigFileNotFound(t *testing.T) {
	dir := t.TempDir()
	// Don't create any config file
	_, err := LoadConfig(dir, nil)
	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}
	if _, ok := err.(*ConfigError); !ok {
		t.Errorf("expected *ConfigError, got %T: %v", err, err)
	}
}

// TestInvalidJSON verifies that malformed JSON produces a ConfigError.
func TestInvalidJSON(t *testing.T) {
	dir := writeTempConfig(t, `{ invalid json }`)
	_, err := LoadConfig(dir, nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestJSONLoading verifies that all config fields are loaded from JSON.
func TestJSONLoading(t *testing.T) {
	configJSON := `{
		"model": "claude-opus-4-5",
		"max_turns": 500,
		"max_iterations": 10,
		"auto_continue_delay": 5,
		"post_run_instructions": ["check logs", "run tests"],
		"error_recovery": {
			"max_consecutive_errors": 3,
			"initial_backoff_seconds": 2.0,
			"max_backoff_seconds": 60.0,
			"backoff_multiplier": 1.5
		}
	}`
	dir := writeTempConfig(t, configJSON)
	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Model != "claude-opus-4-5" {
		t.Errorf("Model = %q, want claude-opus-4-5", cfg.Model)
	}
	if cfg.MaxTurns != 500 {
		t.Errorf("MaxTurns = %d, want 500", cfg.MaxTurns)
	}
	if cfg.MaxIterations == nil || *cfg.MaxIterations != 10 {
		t.Errorf("MaxIterations = %v, want 10", cfg.MaxIterations)
	}
	if cfg.AutoContinueDelay != 5 {
		t.Errorf("AutoContinueDelay = %d, want 5", cfg.AutoContinueDelay)
	}
	if len(cfg.PostRunInstructions) != 2 {
		t.Errorf("PostRunInstructions count = %d, want 2", len(cfg.PostRunInstructions))
	}
	if cfg.ErrorRecovery.MaxConsecutiveErrors != 3 {
		t.Errorf("MaxConsecutiveErrors = %d", cfg.ErrorRecovery.MaxConsecutiveErrors)
	}
	if cfg.ErrorRecovery.InitialBackoffSeconds != 2.0 {
		t.Errorf("InitialBackoffSeconds = %f", cfg.ErrorRecovery.InitialBackoffSeconds)
	}
}

// TestSecurityConfig verifies that security config is loaded correctly.
func TestSecurityConfig(t *testing.T) {
	configJSON := `{
		"security": {
			"permission_mode": "bypassPermissions",
			"sandbox": {
				"enabled": false,
				"auto_allow_bash_if_sandboxed": false,
				"allow_unsandboxed_commands": true,
				"excluded_commands": ["docker", "kubectl"],
				"network": {
					"allowed_domains": ["github.com"],
					"allow_local_binding": true,
					"allow_unix_sockets": ["/var/run/docker.sock"]
				}
			},
			"permissions": {
				"allow": ["Bash(npm:*)"],
				"deny": ["Bash(rm:-rf*)"]
			}
		}
	}`
	dir := writeTempConfig(t, configJSON)
	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Security.PermissionMode != "bypassPermissions" {
		t.Errorf("PermissionMode = %q", cfg.Security.PermissionMode)
	}
	if cfg.Security.Sandbox.Enabled {
		t.Error("Sandbox.Enabled = true, want false")
	}
	if cfg.Security.Sandbox.AutoAllowBashIfSandboxed {
		t.Error("AutoAllowBashIfSandboxed = true, want false")
	}
	if !cfg.Security.Sandbox.AllowUnsandboxedCommands {
		t.Error("AllowUnsandboxedCommands = false, want true")
	}
	if len(cfg.Security.Sandbox.ExcludedCommands) != 2 {
		t.Errorf("ExcludedCommands count = %d, want 2", len(cfg.Security.Sandbox.ExcludedCommands))
	}
	if len(cfg.Security.Sandbox.Network.AllowedDomains) != 1 ||
		cfg.Security.Sandbox.Network.AllowedDomains[0] != "github.com" {
		t.Errorf("AllowedDomains = %v", cfg.Security.Sandbox.Network.AllowedDomains)
	}
	if !cfg.Security.Sandbox.Network.AllowLocalBinding {
		t.Error("AllowLocalBinding = false, want true")
	}
	if len(cfg.Security.Permissions.Allow) != 1 ||
		cfg.Security.Permissions.Allow[0] != "Bash(npm:*)" {
		t.Errorf("Permissions.Allow = %v", cfg.Security.Permissions.Allow)
	}
	if len(cfg.Security.Permissions.Deny) != 1 {
		t.Errorf("Permissions.Deny = %v", cfg.Security.Permissions.Deny)
	}
}

// TestMcpServerConfig verifies that MCP server configs are loaded correctly.
func TestMcpServerConfig(t *testing.T) {
	configJSON := `{
		"tools": {
			"mcp_servers": {
				"puppeteer": {
					"command": "npx",
					"args": ["-y", "@modelcontextprotocol/server-puppeteer"],
					"env": {"PUPPETEER_HEADLESS": "true"}
				}
			}
		}
	}`
	dir := writeTempConfig(t, configJSON)
	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	srv, ok := cfg.Tools.McpServers["puppeteer"]
	if !ok {
		t.Fatal("MCP server 'puppeteer' not found")
	}
	if srv.Command != "npx" {
		t.Errorf("Command = %q, want npx", srv.Command)
	}
	if len(srv.Args) != 2 {
		t.Errorf("Args count = %d, want 2", len(srv.Args))
	}
	if srv.Env["PUPPETEER_HEADLESS"] != "true" {
		t.Errorf("Env PUPPETEER_HEADLESS = %q", srv.Env["PUPPETEER_HEADLESS"])
	}
}

// TestEnvVarExpansion verifies that environment variables are expanded in MCP
// server env values.
func TestEnvVarExpansion(t *testing.T) {
	// Set a test env var
	t.Setenv("TEST_API_KEY_GO_PORT", "secret123")

	configJSON := `{
		"tools": {
			"mcp_servers": {
				"myserver": {
					"command": "myserver",
					"env": {"API_KEY": "${TEST_API_KEY_GO_PORT}"}
				}
			}
		}
	}`
	dir := writeTempConfig(t, configJSON)
	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	srv := cfg.Tools.McpServers["myserver"]
	if srv.Env["API_KEY"] != "secret123" {
		t.Errorf("API_KEY = %q, want secret123", srv.Env["API_KEY"])
	}
}

// TestCLIOverrideModel verifies that --model CLI override replaces config model.
func TestCLIOverrideModel(t *testing.T) {
	dir := writeTempConfig(t, minimalConfig())
	overrides := &CLIOverrides{Model: "claude-opus-4-5"}
	cfg, err := LoadConfig(dir, overrides)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Model != "claude-opus-4-5" {
		t.Errorf("Model = %q, want claude-opus-4-5", cfg.Model)
	}
}

// TestCLIOverrideMaxIterations verifies that --max-iterations CLI override works.
func TestCLIOverrideMaxIterations(t *testing.T) {
	dir := writeTempConfig(t, minimalConfig())
	n := 42
	overrides := &CLIOverrides{MaxIterations: &n}
	cfg, err := LoadConfig(dir, overrides)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.MaxIterations == nil || *cfg.MaxIterations != 42 {
		t.Errorf("MaxIterations = %v, want 42", cfg.MaxIterations)
	}
}

// TestCLIOverrideEmpty verifies that empty overrides don't change config values.
func TestCLIOverrideEmpty(t *testing.T) {
	dir := writeTempConfig(t, minimalConfig())
	overrides := &CLIOverrides{} // empty — should not override
	cfg, err := LoadConfig(dir, overrides)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Model != DefaultModel {
		t.Errorf("Model = %q, want %q", cfg.Model, DefaultModel)
	}
}

// TestValidationErrors covers various validation failures.
func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		wantErrFrag string
	}{
		{
			name: "empty model",
			configJSON: `{
				"model": ""
			}`,
			wantErrFrag: "model must be a non-empty string",
		},
		{
			name: "invalid permission mode",
			configJSON: `{
				"security": {"permission_mode": "invalid_mode"}
			}`,
			wantErrFrag: "security.permission_mode",
		},
		{
			name: "max_turns zero",
			configJSON: `{
				"max_turns": 0
			}`,
			wantErrFrag: "max_turns must be positive",
		},
		{
			name: "negative auto_continue_delay",
			configJSON: `{
				"auto_continue_delay": -1
			}`,
			wantErrFrag: "auto_continue_delay must be non-negative",
		},
		{
			name: "max_iterations zero",
			configJSON: `{
				"max_iterations": 0
			}`,
			wantErrFrag: "max_iterations must be positive",
		},
		{
			name: "mcp server empty command",
			configJSON: `{
				"tools": {
					"mcp_servers": {
						"myserver": {"command": ""}
					}
				}
			}`,
			wantErrFrag: "tools.mcp_servers.myserver.command must be a non-empty string",
		},
		{
			name: "error_recovery max_consecutive_errors zero",
			configJSON: `{
				"error_recovery": {"max_consecutive_errors": 0}
			}`,
			wantErrFrag: "error_recovery.max_consecutive_errors must be positive",
		},
		{
			name: "error_recovery backoff_multiplier too low",
			configJSON: `{
				"error_recovery": {"backoff_multiplier": 0.5}
			}`,
			wantErrFrag: "error_recovery.backoff_multiplier must be >= 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeTempConfig(t, tt.configJSON)
			_, err := LoadConfig(dir, nil)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErrFrag)
			}
			if !containsString(err.Error(), tt.wantErrFrag) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErrFrag)
			}
		})
	}
}

// TestResolvedPaths verifies that ProjectDir and HarnessDir are set correctly.
func TestResolvedPaths(t *testing.T) {
	dir := writeTempConfig(t, minimalConfig())
	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	absDir, _ := filepath.Abs(dir)
	if cfg.ProjectDir != absDir {
		t.Errorf("ProjectDir = %q, want %q", cfg.ProjectDir, absDir)
	}
	expectedHarnessDir := filepath.Join(absDir, ConfigDirName)
	if cfg.HarnessDir != expectedHarnessDir {
		t.Errorf("HarnessDir = %q, want %q", cfg.HarnessDir, expectedHarnessDir)
	}
}

// TestConfigJSONSerialization verifies that a config can be serialized to JSON
// and deserialized back.
func TestConfigJSONSerialization(t *testing.T) {
	dir := writeTempConfig(t, minimalConfig())
	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	// Serialize
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// Deserialize
	var cfg2 HarnessConfig
	if err := json.Unmarshal(data, &cfg2); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if cfg2.Model != cfg.Model {
		t.Errorf("Model mismatch: %q vs %q", cfg2.Model, cfg.Model)
	}
}

// TestBuiltinToolsOverride verifies that config can override the builtin tools list.
func TestBuiltinToolsOverride(t *testing.T) {
	configJSON := `{
		"tools": {"builtin": ["Read", "Glob"]}
	}`
	dir := writeTempConfig(t, configJSON)
	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Tools.Builtin) != 2 {
		t.Errorf("Builtin count = %d, want 2", len(cfg.Tools.Builtin))
	}
}

// containsString returns true if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
