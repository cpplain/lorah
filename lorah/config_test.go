package lorah

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	configFile := filepath.Join(harnessDir, ConfigFileName)
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

	// Harness defaults
	if cfg.Harness.MaxIterations != nil {
		t.Errorf("MaxIterations = %v, want nil", cfg.Harness.MaxIterations)
	}
	if cfg.Harness.AutoContinueDelay != 3 {
		t.Errorf("AutoContinueDelay = %d, want 3", cfg.Harness.AutoContinueDelay)
	}
	if cfg.Harness.ErrorRecovery.MaxConsecutiveErrors != 5 {
		t.Errorf("MaxConsecutiveErrors = %d, want 5", cfg.Harness.ErrorRecovery.MaxConsecutiveErrors)
	}
	if cfg.Harness.ErrorRecovery.InitialBackoffSeconds != 5.0 {
		t.Errorf("InitialBackoffSeconds = %f, want 5.0", cfg.Harness.ErrorRecovery.InitialBackoffSeconds)
	}
	if cfg.Harness.ErrorRecovery.MaxBackoffSeconds != 120.0 {
		t.Errorf("MaxBackoffSeconds = %f, want 120.0", cfg.Harness.ErrorRecovery.MaxBackoffSeconds)
	}
	if cfg.Harness.ErrorRecovery.BackoffMultiplier != 2.0 {
		t.Errorf("BackoffMultiplier = %f, want 2.0", cfg.Harness.ErrorRecovery.BackoffMultiplier)
	}
	if cfg.Harness.ErrorRecovery.MaxErrorMessageLength != 2000 {
		t.Errorf("MaxErrorMessageLength = %d, want 2000", cfg.Harness.ErrorRecovery.MaxErrorMessageLength)
	}

	// Claude defaults - access via map with explicit flag names
	maxTurns, ok := cfg.Claude.Flags["--max-turns"].(float64)
	if !ok || int(maxTurns) != 1000 {
		t.Errorf("Claude.Flags[--max-turns] = %v, want 1000", cfg.Claude.Flags["--max-turns"])
	}
	model, ok := cfg.Claude.Settings["model"].(string)
	if !ok || model != "claude-sonnet-4-5" {
		t.Errorf("Claude.Settings[model] = %v, want 'claude-sonnet-4-5'", cfg.Claude.Settings["model"])
	}
	permissions, ok := cfg.Claude.Settings["permissions"].(map[string]any)
	if !ok {
		t.Error("Claude.Settings[permissions] is not a map")
	} else {
		defaultMode, ok := permissions["defaultMode"].(string)
		if !ok || defaultMode != "bypassPermissions" {
			t.Errorf("permissions[defaultMode] = %v, want 'bypassPermissions'", permissions["defaultMode"])
		}
	}
	sandbox, ok := cfg.Claude.Settings["sandbox"].(map[string]any)
	if !ok {
		t.Error("Claude.Settings[sandbox] is not a map")
	} else {
		enabled, ok := sandbox["enabled"].(bool)
		if !ok || !enabled {
			t.Errorf("sandbox[enabled] = %v, want true", sandbox["enabled"])
		}
	}
}

// TestConfigFileNotFound verifies that a missing config file uses defaults.
func TestConfigFileNotFound(t *testing.T) {
	dir := t.TempDir()
	// Create harness dir but no config file
	harnessDir := filepath.Join(dir, ConfigDirName)
	if err := os.MkdirAll(harnessDir, 0o755); err != nil {
		t.Fatalf("failed to create harness dir: %v", err)
	}

	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig failed for missing config (should use defaults): %v", err)
	}

	// Verify defaults are used
	maxTurns, ok := cfg.Claude.Flags["--max-turns"].(float64)
	if !ok || int(maxTurns) != 1000 {
		t.Errorf("Claude.Flags[--max-turns] = %v, want 1000 (default)", cfg.Claude.Flags["--max-turns"])
	}
}

// TestInvalidJSON verifies that malformed JSON produces an error.
func TestInvalidJSON(t *testing.T) {
	dir := writeTempConfig(t, `{invalid json`)
	_, err := LoadConfig(dir, nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestJSONLoading verifies that JSON fields are properly loaded and merged.
func TestJSONLoading(t *testing.T) {
	configJSON := `{
		"harness": {
			"max-iterations": 10,
			"auto-continue-delay": 5,
			"error-recovery": {
				"max-consecutive-errors": 3,
				"initial-backoff-seconds": 10.0,
				"max-backoff-seconds": 60.0,
				"backoff-multiplier": 1.5,
				"max-error-message-length": 1000
			}
		},
		"claude": {
			"flags": {
				"--max-turns": 500
			},
			"settings": {
				"model": "claude-opus-4-6",
				"permissions": {
					"defaultMode": "requireApproval"
				}
			}
		}
	}`
	dir := writeTempConfig(t, configJSON)

	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	maxTurns, ok := cfg.Claude.Flags["--max-turns"].(float64)
	if !ok || int(maxTurns) != 500 {
		t.Errorf("Claude.Flags[--max-turns] = %v, want 500", cfg.Claude.Flags["--max-turns"])
	}
	model, ok := cfg.Claude.Settings["model"].(string)
	if !ok || model != "claude-opus-4-6" {
		t.Errorf("Claude.Settings[model] = %v, want 'claude-opus-4-6'", cfg.Claude.Settings["model"])
	}
	permissions, ok := cfg.Claude.Settings["permissions"].(map[string]any)
	if !ok {
		t.Error("Claude.Settings[permissions] is not a map")
	} else {
		defaultMode, ok := permissions["defaultMode"].(string)
		if !ok || defaultMode != "requireApproval" {
			t.Errorf("permissions[defaultMode] = %v, want 'requireApproval'", permissions["defaultMode"])
		}
	}
	if cfg.Harness.MaxIterations == nil || *cfg.Harness.MaxIterations != 10 {
		t.Errorf("MaxIterations = %v, want 10", cfg.Harness.MaxIterations)
	}
	if cfg.Harness.AutoContinueDelay != 5 {
		t.Errorf("AutoContinueDelay = %d, want 5", cfg.Harness.AutoContinueDelay)
	}
	if cfg.Harness.ErrorRecovery.MaxConsecutiveErrors != 3 {
		t.Errorf("MaxConsecutiveErrors = %d, want 3", cfg.Harness.ErrorRecovery.MaxConsecutiveErrors)
	}
	if cfg.Harness.ErrorRecovery.InitialBackoffSeconds != 10.0 {
		t.Errorf("InitialBackoffSeconds = %f, want 10.0", cfg.Harness.ErrorRecovery.InitialBackoffSeconds)
	}
	if cfg.Harness.ErrorRecovery.MaxBackoffSeconds != 60.0 {
		t.Errorf("MaxBackoffSeconds = %f, want 60.0", cfg.Harness.ErrorRecovery.MaxBackoffSeconds)
	}
	if cfg.Harness.ErrorRecovery.BackoffMultiplier != 1.5 {
		t.Errorf("BackoffMultiplier = %f, want 1.5", cfg.Harness.ErrorRecovery.BackoffMultiplier)
	}
	if cfg.Harness.ErrorRecovery.MaxErrorMessageLength != 1000 {
		t.Errorf("MaxErrorMessageLength = %d, want 1000", cfg.Harness.ErrorRecovery.MaxErrorMessageLength)
	}
}

// TestCLIOverrideMaxIterations verifies that CLI overrides work.
func TestCLIOverrideMaxIterations(t *testing.T) {
	dir := writeTempConfig(t, minimalConfig())
	maxIter := 42
	cfg, err := LoadConfig(dir, &CLIOverrides{
		MaxIterations: &maxIter,
	})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Harness.MaxIterations == nil || *cfg.Harness.MaxIterations != 42 {
		t.Errorf("MaxIterations = %v, want 42", cfg.Harness.MaxIterations)
	}
}

// TestCLIOverrideEmpty verifies that nil overrides don't affect config.
func TestCLIOverrideEmpty(t *testing.T) {
	dir := writeTempConfig(t, minimalConfig())
	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	maxTurns, ok := cfg.Claude.Flags["--max-turns"].(float64)
	if !ok || int(maxTurns) != 1000 {
		t.Errorf("Claude.Flags[--max-turns] = %v, want 1000 (default)", cfg.Claude.Flags["--max-turns"])
	}
}

// TestValidationErrors verifies that invalid configurations are rejected.
func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		configJSON string
		wantErr    string
	}{
		// Claude section validation removed - Claude CLI handles its own validation
		{
			name:       "negative auto-continue-delay",
			configJSON: `{"harness": {"auto-continue-delay": -1}}`,
			wantErr:    "harness.auto-continue-delay must be non-negative",
		},
		{
			name:       "zero max-iterations",
			configJSON: `{"harness": {"max-iterations": 0}}`,
			wantErr:    "harness.max-iterations must be positive when set",
		},
		{
			name:       "negative max-iterations",
			configJSON: `{"harness": {"max-iterations": -1}}`,
			wantErr:    "harness.max-iterations must be positive when set",
		},
		{
			name:       "zero max-consecutive-errors",
			configJSON: `{"harness": {"error-recovery": {"max-consecutive-errors": 0}}}`,
			wantErr:    "harness.error-recovery.max-consecutive-errors must be positive",
		},
		{
			name:       "negative initial-backoff-seconds",
			configJSON: `{"harness": {"error-recovery": {"initial-backoff-seconds": -1}}}`,
			wantErr:    "harness.error-recovery.initial-backoff-seconds must be positive",
		},
		{
			name:       "max-backoff less than initial",
			configJSON: `{"harness": {"error-recovery": {"initial-backoff-seconds": 10, "max-backoff-seconds": 5}}}`,
			wantErr:    "harness.error-recovery.max-backoff-seconds must be >= initial-backoff-seconds",
		},
		{
			name:       "backoff-multiplier less than 1",
			configJSON: `{"harness": {"error-recovery": {"backoff-multiplier": 0.5}}}`,
			wantErr:    "harness.error-recovery.backoff-multiplier must be >= 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeTempConfig(t, tt.configJSON)
			_, err := LoadConfig(dir, nil)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			errMsg := err.Error()
			if tt.wantErr != "" && !strings.Contains(errMsg, tt.wantErr) {
				t.Errorf("error = %q, want substring %q", errMsg, tt.wantErr)
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

	if cfg.ProjectDir == "" {
		t.Error("ProjectDir should not be empty")
	}
	if cfg.HarnessDir == "" {
		t.Error("HarnessDir should not be empty")
	}
	if cfg.HarnessDir != filepath.Join(cfg.ProjectDir, ConfigDirName) {
		t.Errorf("HarnessDir = %q, want %q", cfg.HarnessDir, filepath.Join(cfg.ProjectDir, ConfigDirName))
	}
}

// TestConfigJSONSerialization verifies that config can round-trip through JSON.
func TestConfigJSONSerialization(t *testing.T) {
	dir := writeTempConfig(t, minimalConfig())
	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	// Serialize and deserialize
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var cfg2 HarnessConfig
	if err := json.Unmarshal(data, &cfg2); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	maxTurns1, ok1 := cfg.Claude.Flags["--max-turns"].(float64)
	maxTurns2, ok2 := cfg2.Claude.Flags["--max-turns"].(float64)
	if !ok1 || !ok2 || maxTurns1 != maxTurns2 {
		t.Errorf("Claude.Flags[--max-turns] mismatch: %v vs %v", cfg2.Claude.Flags["--max-turns"], cfg.Claude.Flags["--max-turns"])
	}
}

// TestAllowUnsandboxedCommandsDefaultFalse verifies that allowUnsandboxedCommands defaults to false.
func TestAllowUnsandboxedCommandsDefaultFalse(t *testing.T) {
	dir := writeTempConfig(t, minimalConfig())
	cfg, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	sandbox, ok := cfg.Claude.Settings["sandbox"].(map[string]any)
	if !ok {
		t.Fatal("sandbox is not a map")
	}
	allowUnsandboxed, ok := sandbox["allowUnsandboxedCommands"].(bool)
	if !ok {
		t.Error("allowUnsandboxedCommands not found or not a bool")
	} else if allowUnsandboxed != false {
		t.Errorf("AllowUnsandboxedCommands = %v, want false", allowUnsandboxed)
	}
}
