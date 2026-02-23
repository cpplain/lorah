package lorah

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckResultString tests the String() formatting of CheckResult.
func TestCheckResultString(t *testing.T) {
	tests := []struct {
		name     string
		result   CheckResult
		contains []string
	}{
		{
			name: "PASS with message",
			result: CheckResult{
				Name:    "Go version",
				Status:  "PASS",
				Message: "1.21.0",
			},
			contains: []string{"[PASS]", "Go version", "1.21.0"},
		},
		{
			name: "FAIL without message",
			result: CheckResult{
				Name:   "Config file",
				Status: "FAIL",
			},
			contains: []string{"[FAIL]", "Config file"},
		},
		{
			name: "WARN with message",
			result: CheckResult{
				Name:    "MCP servers",
				Status:  "WARN",
				Message: "npx not found on PATH",
			},
			contains: []string{"[WARN]", "MCP servers", "npx not found on PATH"},
		},
		{
			name: "result includes separator",
			result: CheckResult{
				Name:    "Authentication",
				Status:  "PASS",
				Message: "ANTHROPIC_API_KEY set",
			},
			contains: []string{" - "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.String()
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("String() = %q, missing %q", got, want)
				}
			}
		})
	}
}

// TestCheckResultStringNoMessage verifies that no " - " separator appears when there's no message.
func TestCheckResultStringNoMessage(t *testing.T) {
	result := CheckResult{Name: "Test", Status: "PASS"}
	got := result.String()
	if strings.Contains(got, " - ") {
		t.Errorf("String() = %q; should not contain ' - ' when message is empty", got)
	}
}

// TestCheckGoVersion verifies that CheckGoVersion always returns a PASS.
func TestCheckGoVersion(t *testing.T) {
	result := CheckGoVersion()
	if result.Status != "PASS" {
		t.Errorf("CheckGoVersion().Status = %q; want PASS", result.Status)
	}
	if result.Name == "" {
		t.Error("CheckGoVersion().Name should not be empty")
	}
	if result.Message == "" {
		t.Error("CheckGoVersion().Message should contain version string")
	}
}

// TestCheckConfigExists tests config file existence checking.
func TestCheckConfigExists(t *testing.T) {
	t.Run("config file exists", func(t *testing.T) {
		dir := t.TempDir()
		configFile := filepath.Join(dir, "config.json")
		if err := os.WriteFile(configFile, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}

		result := CheckConfigExists(dir)
		if result.Status != "PASS" {
			t.Errorf("CheckConfigExists().Status = %q; want PASS", result.Status)
		}
		if !strings.Contains(result.Message, configFile) {
			t.Errorf("CheckConfigExists().Message = %q; want to contain %q", result.Message, configFile)
		}
	})

	t.Run("config file missing", func(t *testing.T) {
		dir := t.TempDir()

		result := CheckConfigExists(dir)
		if result.Status != "FAIL" {
			t.Errorf("CheckConfigExists().Status = %q; want FAIL", result.Status)
		}
		if !strings.Contains(result.Message, "Not found") {
			t.Errorf("CheckConfigExists().Message = %q; want to contain 'Not found'", result.Message)
		}
	})
}

// TestCheckConfigValid tests config validation.
func TestCheckConfigValid(t *testing.T) {
	t.Run("valid minimal config", func(t *testing.T) {
		dir := t.TempDir()
		harnessDir := filepath.Join(dir, ".lorah")
		if err := os.MkdirAll(harnessDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Write minimal valid config
		cfg := map[string]any{}
		data, _ := json.Marshal(cfg)
		if err := os.WriteFile(filepath.Join(harnessDir, "config.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}

		result, loadedCfg := CheckConfigValid(dir)
		if result.Status != "PASS" {
			t.Errorf("CheckConfigValid().Status = %q; want PASS (message: %s)", result.Status, result.Message)
		}
		if loadedCfg == nil {
			t.Error("CheckConfigValid() should return non-nil config on success")
		}
	})

	t.Run("missing config dir", func(t *testing.T) {
		dir := t.TempDir()

		result, cfg := CheckConfigValid(dir)
		if result.Status != "FAIL" {
			t.Errorf("CheckConfigValid().Status = %q; want FAIL", result.Status)
		}
		if cfg != nil {
			t.Error("CheckConfigValid() should return nil config on failure")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		dir := t.TempDir()
		harnessDir := filepath.Join(dir, ".lorah")
		if err := os.MkdirAll(harnessDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Write invalid JSON
		if err := os.WriteFile(filepath.Join(harnessDir, "config.json"), []byte("{invalid json}"), 0o644); err != nil {
			t.Fatal(err)
		}

		result, cfg := CheckConfigValid(dir)
		if result.Status != "FAIL" {
			t.Errorf("CheckConfigValid().Status = %q; want FAIL", result.Status)
		}
		if cfg != nil {
			t.Error("CheckConfigValid() should return nil config on failure")
		}
	})
}

// TestCheckRequiredFiles tests that required files are checked.
func TestCheckRequiredFiles(t *testing.T) {
	t.Run("all files exist", func(t *testing.T) {
		dir := t.TempDir()
		harnessDir := filepath.Join(dir, ".lorah")
		if err := os.MkdirAll(filepath.Join(harnessDir, "prompts"), 0o755); err != nil {
			t.Fatal(err)
		}
		// Create all required files
		os.WriteFile(filepath.Join(harnessDir, TaskListFile), []byte("[]"), 0o644)
		os.WriteFile(filepath.Join(harnessDir, AgentProgressFile), []byte(""), 0o644)
		os.WriteFile(filepath.Join(harnessDir, "prompts", "initialization.md"), []byte(""), 0o644)
		os.WriteFile(filepath.Join(harnessDir, "prompts", "implementation.md"), []byte(""), 0o644)

		result := CheckRequiredFiles(harnessDir)
		if result.Status != "PASS" {
			t.Errorf("CheckRequiredFiles().Status = %q; want PASS", result.Status)
		}
	})

	t.Run("files missing", func(t *testing.T) {
		dir := t.TempDir()
		harnessDir := filepath.Join(dir, ".lorah")
		os.MkdirAll(harnessDir, 0o755)

		result := CheckRequiredFiles(harnessDir)
		if result.Status != "FAIL" {
			t.Errorf("CheckRequiredFiles().Status = %q; want FAIL", result.Status)
		}
		if result.Message == "" {
			t.Error("CheckRequiredFiles().Message should contain missing files")
		}
	})
}

// TestCheckProjectDir tests project directory writability checking.
func TestCheckProjectDir(t *testing.T) {
	t.Run("existing writable directory", func(t *testing.T) {
		dir := t.TempDir()

		result := CheckProjectDir(dir)
		if result.Status != "PASS" {
			t.Errorf("CheckProjectDir(%q).Status = %q; want PASS", dir, result.Status)
		}
		if !strings.Contains(result.Message, dir) {
			t.Errorf("CheckProjectDir().Message = %q; want to contain %q", result.Message, dir)
		}
	})

	t.Run("nonexistent directory with writable parent", func(t *testing.T) {
		parentDir := t.TempDir()
		nonExistent := filepath.Join(parentDir, "new-project")

		result := CheckProjectDir(nonExistent)
		if result.Status != "PASS" {
			t.Errorf("CheckProjectDir(%q).Status = %q; want PASS", nonExistent, result.Status)
		}
		if !strings.Contains(result.Message, "Will be created") {
			t.Errorf("CheckProjectDir().Message = %q; want to contain 'Will be created'", result.Message)
		}
	})
}

// TestRunVerify tests the full verification run.
func TestRunVerify(t *testing.T) {
	t.Run("returns multiple results", func(t *testing.T) {
		dir := t.TempDir()

		results := RunVerify(dir)
		if len(results) == 0 {
			t.Error("RunVerify() should return at least one result")
		}

		// Go version check should always be first and PASS
		if results[0].Name != "Go version" {
			t.Errorf("results[0].Name = %q; want 'Go version'", results[0].Name)
		}
		if results[0].Status != "PASS" {
			t.Errorf("results[0].Status = %q; want PASS", results[0].Status)
		}
	})

	t.Run("config-dependent checks skipped when no config", func(t *testing.T) {
		dir := t.TempDir()

		results := RunVerify(dir)

		// Find config validation result
		var configValidResult *CheckResult
		for i := range results {
			if results[i].Name == "Config validation" {
				configValidResult = &results[i]
				break
			}
		}

		if configValidResult == nil {
			t.Fatal("Expected Config validation result not found")
		}
		if configValidResult.Status != "FAIL" {
			t.Errorf("Config validation should fail when no config exists, got %q", configValidResult.Status)
		}

		// Required files, MCP servers, and Project directory should not appear
		for _, r := range results {
			switch r.Name {
			case "Required files", "MCP servers", "Project directory":
				t.Errorf("Config-dependent check %q should not run when config fails", r.Name)
			}
		}
	})

	t.Run("config-dependent checks run with valid config", func(t *testing.T) {
		dir := t.TempDir()
		harnessDir := filepath.Join(dir, ".lorah")
		if err := os.MkdirAll(harnessDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Write minimal valid config
		cfg := map[string]any{}
		data, _ := json.Marshal(cfg)
		if err := os.WriteFile(filepath.Join(harnessDir, "config.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}

		results := RunVerify(dir)

		// Should include config-dependent checks
		checkNames := make(map[string]bool)
		for _, r := range results {
			checkNames[r.Name] = true
		}

		expectedChecks := []string{"Required files", "MCP servers", "Project directory"}
		for _, name := range expectedChecks {
			if !checkNames[name] {
				t.Errorf("Expected check %q not found in results", name)
			}
		}
	})
}

// TestCheckMCPCommandsNoServers tests MCP check with no configured servers.
func TestCheckMCPCommandsNoServers(t *testing.T) {
	dir := t.TempDir()
	harnessDir := filepath.Join(dir, ".lorah")
	if err := os.MkdirAll(harnessDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := map[string]any{}
	data, _ := json.Marshal(cfg)
	os.WriteFile(filepath.Join(harnessDir, "config.json"), data, 0o644)

	_, loadedCfg := CheckConfigValid(dir)
	if loadedCfg == nil {
		t.Fatal("config should load successfully")
	}

	result := CheckMCPCommands(loadedCfg)
	if result.Status != "PASS" {
		t.Errorf("CheckMCPCommands() with no servers: Status = %q; want PASS", result.Status)
	}
	if !strings.Contains(result.Message, "None configured") {
		t.Errorf("CheckMCPCommands() with no servers: Message = %q; want 'None configured'", result.Message)
	}
}
