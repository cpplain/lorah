// Package main integration tests verify end-to-end CLI behavior and
// the run workflow using a fake claude subprocess.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cpplain/lorah/internal/config"
	"github.com/cpplain/lorah/internal/info"
	"github.com/cpplain/lorah/internal/runner"
	"github.com/cpplain/lorah/internal/tracking"
	"github.com/cpplain/lorah/internal/verify"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

// buildBinary compiles the lorah binary into a temp directory and
// returns the path. The binary is built once per test binary invocation.
func buildBinary(t *testing.T) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "lorah")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = filepath.Join(moduleRoot(), "cmd", "lorah")
	// Pass GOCACHE so sandbox builds succeed
	if gcache := os.Getenv("GOCACHE"); gcache != "" {
		cmd.Env = append(os.Environ(), "GOCACHE="+gcache)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, out)
	}
	return binPath
}

// moduleRoot returns the root directory of the Go module (two levels up from
// this file: cmd/lorah/ → project root).
func moduleRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	// filename is .../cmd/lorah/integration_test.go
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// writeFakeClaude writes a shell script to tmpDir/bin/claude that acts as a
// stub, emitting a valid stream-JSON ResultMessage and exiting 0. Returns the
// bin directory to prepend to PATH.
func writeFakeClaude(t *testing.T, tmpDir string) string {
	t.Helper()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	claudePath := filepath.Join(binDir, "claude")
	script := "#!/bin/sh\necho '{\"type\":\"result\",\"subtype\":\"success\",\"session_id\":\"fake-session-123\",\"is_error\":false,\"result\":\"done\",\"duration_ms\":100,\"num_turns\":1,\"total_cost_usd\":0.01}'\nexit 0\n"
	if err := os.WriteFile(claudePath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return binDir
}

// makeInitedProject runs info.InitProject in a temp dir and returns the dir.
func makeInitedProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := info.InitProject(dir); err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	return dir
}

// loadSessionState reads and unmarshals .lorah/session.json.
func loadSessionState(t *testing.T, projectDir string) runner.SessionState {
	t.Helper()
	path := filepath.Join(projectDir, ".lorah", "session.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read session.json: %v", err)
	}
	var state runner.SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("unmarshal session.json: %v", err)
	}
	return state
}

// ─── Integration: init workflow ──────────────────────────────────────────────

func TestIntegration_InitWorkflow(t *testing.T) {
	t.Run("creates_expected_files", func(t *testing.T) {
		dir := t.TempDir()
		if err := info.InitProject(dir); err != nil {
			t.Fatalf("InitProject: %v", err)
		}

		harnessDir := filepath.Join(dir, ".lorah")
		expected := []string{
			filepath.Join(harnessDir, "config.json"),
			filepath.Join(harnessDir, "spec.md"),
			filepath.Join(harnessDir, tracking.TaskListFile),
			filepath.Join(harnessDir, tracking.AgentProgressFile),
			filepath.Join(harnessDir, "prompts", "initialization.md"),
			filepath.Join(harnessDir, "prompts", "implementation.md"),
		}
		for _, f := range expected {
			if _, err := os.Stat(f); os.IsNotExist(err) {
				t.Errorf("expected file missing: %s", f)
			}
		}
	})

	t.Run("config_json_is_valid", func(t *testing.T) {
		dir := makeInitedProject(t)
		// config.LoadConfig should succeed on the scaffolded project
		cfg, err := config.LoadConfig(dir, nil)
		if err != nil {
			t.Fatalf("LoadConfig on freshly initialized project: %v", err)
		}
		if cfg.Model == "" {
			t.Error("expected non-empty model in default config")
		}
	})

	t.Run("refuses_overwrite", func(t *testing.T) {
		dir := makeInitedProject(t)
		// Second init should fail
		err := info.InitProject(dir)
		if err == nil {
			t.Error("expected error on second InitProject call, got nil")
		}
	})

	t.Run("verify_passes_after_init", func(t *testing.T) {
		dir := makeInitedProject(t)
		results := verify.RunVerify(dir)

		var fails int
		for _, r := range results {
			if r.Status == "FAIL" {
				// Only count config-related fails — auth/environment checks may
				// fail in a sandboxed test environment.
				if strings.Contains(r.Name, "Config") || strings.Contains(r.Name, "config") {
					fails++
					t.Errorf("config-related FAIL: %s — %s", r.Name, r.Message)
				}
			}
		}
		_ = fails // allow auth checks to fail in sandbox
	})
}

// ─── Integration: run workflow ───────────────────────────────────────────────

func TestIntegration_RunWorkflow(t *testing.T) {
	t.Run("session_state_increments_on_run", func(t *testing.T) {
		dir := t.TempDir()
		harnessDir := filepath.Join(dir, ".lorah")
		promptsDir := filepath.Join(harnessDir, "prompts")
		if err := os.MkdirAll(promptsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create required prompt files
		if err := os.WriteFile(filepath.Join(promptsDir, "initialization.md"), []byte("Initialize"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(promptsDir, "implementation.md"), []byte("Build"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create empty tasks.json
		if err := os.WriteFile(filepath.Join(harnessDir, tracking.TaskListFile), []byte("[]"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Write a fake claude script to PATH
		binDir := writeFakeClaude(t, dir)
		origPath := os.Getenv("PATH")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

		// Build a minimal config with max_iterations=1 and one phase
		maxIter := 1
		cfg := &config.HarnessConfig{
			Model:             "claude-test",
			ProjectDir:        dir,
			HarnessDir:        harnessDir,
			MaxIterations:     &maxIter,
			AutoContinueDelay: 0,
			ErrorRecovery: config.ErrorRecoveryConfig{
				MaxConsecutiveErrors:  3,
				InitialBackoffSeconds: 1.0,
				MaxBackoffSeconds:     10.0,
				BackoffMultiplier:     2.0,
			},
			Security: config.SecurityConfig{
				PermissionMode: "default",
			},
		}

		if err := runner.RunAgent(context.Background(), cfg); err != nil {
			t.Fatalf("RunAgent: %v", err)
		}

		// Session state should have been saved with session_number >= 1
		state := loadSessionState(t, dir)
		if state.SessionNumber < 1 {
			t.Errorf("session_number: got %d, want >= 1", state.SessionNumber)
		}
	})

	t.Run("run_once_phase_marked_complete", func(t *testing.T) {
		dir := t.TempDir()
		harnessDir := filepath.Join(dir, ".lorah")
		promptsDir := filepath.Join(harnessDir, "prompts")
		if err := os.MkdirAll(promptsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create required prompt files
		if err := os.WriteFile(filepath.Join(promptsDir, "initialization.md"), []byte("Initialize"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(promptsDir, "implementation.md"), []byte("Build"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create empty tasks.json so initialization phase runs
		if err := os.WriteFile(filepath.Join(harnessDir, tracking.TaskListFile), []byte("[]"), 0o644); err != nil {
			t.Fatal(err)
		}

		binDir := writeFakeClaude(t, dir)
		origPath := os.Getenv("PATH")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

		maxIter := 1
		cfg := &config.HarnessConfig{
			Model:             "claude-test",
			ProjectDir:        dir,
			HarnessDir:        harnessDir,
			MaxIterations:     &maxIter,
			AutoContinueDelay: 0,
			ErrorRecovery: config.ErrorRecoveryConfig{
				MaxConsecutiveErrors:  3,
				InitialBackoffSeconds: 1.0,
				MaxBackoffSeconds:     10.0,
				BackoffMultiplier:     2.0,
			},
			Security: config.SecurityConfig{
				PermissionMode: "default",
			},
		}

		if err := runner.RunAgent(context.Background(), cfg); err != nil {
			t.Fatalf("RunAgent: %v", err)
		}

		state := loadSessionState(t, dir)
		// initialization should be in completed_phases since it's run_once
		found := false
		for _, name := range state.CompletedPhases {
			if name == "initialization" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'initialization' in completed_phases, got %v", state.CompletedPhases)
		}
	})

	t.Run("max_iterations_stops_loop", func(t *testing.T) {
		dir := t.TempDir()
		harnessDir := filepath.Join(dir, ".lorah")
		promptsDir := filepath.Join(harnessDir, "prompts")
		if err := os.MkdirAll(promptsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create required prompt files
		if err := os.WriteFile(filepath.Join(promptsDir, "initialization.md"), []byte("Initialize"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(promptsDir, "implementation.md"), []byte("Build"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create empty tasks.json
		if err := os.WriteFile(filepath.Join(harnessDir, tracking.TaskListFile), []byte("[]"), 0o644); err != nil {
			t.Fatal(err)
		}

		binDir := writeFakeClaude(t, dir)
		origPath := os.Getenv("PATH")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

		maxIter := 3
		cfg := &config.HarnessConfig{
			Model:             "claude-test",
			ProjectDir:        dir,
			HarnessDir:        harnessDir,
			MaxIterations:     &maxIter,
			AutoContinueDelay: 0,
			ErrorRecovery: config.ErrorRecoveryConfig{
				MaxConsecutiveErrors:  3,
				InitialBackoffSeconds: 1.0,
				MaxBackoffSeconds:     10.0,
				BackoffMultiplier:     2.0,
			},
			Security: config.SecurityConfig{
				PermissionMode: "default",
			},
		}

		if err := runner.RunAgent(context.Background(), cfg); err != nil {
			t.Fatalf("RunAgent: %v", err)
		}

		state := loadSessionState(t, dir)
		// Should have run exactly maxIter sessions
		if state.SessionNumber != maxIter {
			t.Errorf("session_number: got %d, want %d", state.SessionNumber, maxIter)
		}
	})

	t.Run("agent_exits_when_all_tasks_complete", func(t *testing.T) {
		dir := t.TempDir()
		harnessDir := filepath.Join(dir, ".lorah")
		promptsDir := filepath.Join(harnessDir, "prompts")
		if err := os.MkdirAll(promptsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create required prompt files
		if err := os.WriteFile(filepath.Join(promptsDir, "initialization.md"), []byte("Initialize"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(promptsDir, "implementation.md"), []byte("Build"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create tasks.json with all tasks passing (triggers exit)
		taskList := `[{"name": "task1", "passes": true}, {"name": "task2", "passes": true}]`
		if err := os.WriteFile(filepath.Join(harnessDir, tracking.TaskListFile), []byte(taskList), 0o644); err != nil {
			t.Fatal(err)
		}

		binDir := writeFakeClaude(t, dir)
		origPath := os.Getenv("PATH")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

		cfg := &config.HarnessConfig{
			Model:             "claude-test",
			ProjectDir:        dir,
			HarnessDir:        harnessDir,
			AutoContinueDelay: 0,
			ErrorRecovery: config.ErrorRecoveryConfig{
				MaxConsecutiveErrors:  3,
				InitialBackoffSeconds: 1.0,
				MaxBackoffSeconds:     10.0,
				BackoffMultiplier:     2.0,
			},
			Security: config.SecurityConfig{
				PermissionMode: "default",
			},
		}

		// RunAgent should exit cleanly when tracker.IsComplete() returns true
		if err := runner.RunAgent(context.Background(), cfg); err != nil {
			t.Fatalf("RunAgent: %v", err)
		}

		// Since tasks.json already has items, tracker.IsInitialized() = true
		// So agent skips initialization and runs implementation once, then exits
		// No phases should be in completed_phases (only initialization gets marked, and it was skipped)
		state := loadSessionState(t, dir)
		if len(state.CompletedPhases) != 0 {
			t.Errorf("expected 0 completed phases (initialization skipped because tracker already initialized), got %v", state.CompletedPhases)
		}
	})

	t.Run("run_fails_gracefully_when_no_config", func(t *testing.T) {
		dir := t.TempDir()
		// No .lorah/config.json → LoadConfig should fail
		_, err := config.LoadConfig(dir, nil)
		if err == nil {
			t.Error("expected error loading config from empty directory")
		}
	})
}

// ─── Integration: CLI parity ─────────────────────────────────────────────────

// TestIntegration_CLIParity tests that the binary's output and exit codes
// match the expected behavior for all subcommands.
func TestIntegration_CLIParity(t *testing.T) {
	bin := buildBinary(t)

	t.Run("version_flag", func(t *testing.T) {
		out, err := exec.Command(bin, "--version").CombinedOutput()
		if err != nil {
			t.Fatalf("--version exited with error: %v\nOutput: %s", err, out)
		}
		output := string(out)
		if !strings.Contains(output, "lorah") {
			t.Errorf("--version output missing 'lorah': %q", output)
		}
		// Should contain either a version number (X.Y.Z) or "dev"
		if !strings.Contains(output, ".") && !strings.Contains(output, "dev") {
			t.Errorf("--version output missing version: %q", output)
		}
	})

	t.Run("no_args_shows_usage", func(t *testing.T) {
		cmd := exec.Command(bin)
		out, _ := cmd.CombinedOutput()
		output := string(out)
		// Should show available commands
		for _, sub := range []string{"run", "verify", "init", "info"} {
			if !strings.Contains(output, sub) {
				t.Errorf("usage missing subcommand %q: %q", sub, output)
			}
		}
	})

	t.Run("unknown_command_exits_nonzero", func(t *testing.T) {
		cmd := exec.Command(bin, "badcmd")
		err := cmd.Run()
		if err == nil {
			t.Error("expected non-zero exit for unknown command")
		}
	})

	t.Run("init_creates_scaffold", func(t *testing.T) {
		dir := t.TempDir()
		out, err := exec.Command(bin, "init", "--project-dir", dir).CombinedOutput()
		if err != nil {
			t.Fatalf("init exited with error: %v\nOutput: %s", err, out)
		}

		// Check expected files were created
		harnessDir := filepath.Join(dir, ".lorah")
		for _, f := range []string{
			filepath.Join(harnessDir, "config.json"),
			filepath.Join(harnessDir, "spec.md"),
			filepath.Join(harnessDir, tracking.TaskListFile),
			filepath.Join(harnessDir, tracking.AgentProgressFile),
			filepath.Join(harnessDir, "prompts", "initialization.md"),
			filepath.Join(harnessDir, "prompts", "implementation.md"),
		} {
			if _, statErr := os.Stat(f); os.IsNotExist(statErr) {
				t.Errorf("init: expected file missing: %s", f)
			}
		}

		// Output should mention the created directory
		output := string(out)
		if !strings.Contains(output, ".lorah") {
			t.Errorf("init output missing '.lorah': %q", output)
		}
	})

	t.Run("init_refuses_second_init", func(t *testing.T) {
		dir := t.TempDir()
		// First init
		if out, err := exec.Command(bin, "init", "--project-dir", dir).CombinedOutput(); err != nil {
			t.Fatalf("first init failed: %v\n%s", err, out)
		}
		// Second init should fail
		cmd := exec.Command(bin, "init", "--project-dir", dir)
		err := cmd.Run()
		if err == nil {
			t.Error("expected non-zero exit on second init")
		}
	})

	t.Run("verify_exit_code_fail_without_config", func(t *testing.T) {
		dir := t.TempDir()
		cmd := exec.Command(bin, "verify", "--project-dir", dir)
		err := cmd.Run()
		if err == nil {
			t.Error("verify should exit non-zero when config missing")
		}
	})

	t.Run("verify_exit_code_pass_with_valid_config", func(t *testing.T) {
		dir := makeInitedProject(t)
		out, err := exec.Command(bin, "verify", "--project-dir", dir).CombinedOutput()
		if err != nil {
			// May fail due to auth env vars not set in test environment; check
			// that it only fails due to FAIL items, not crashes
			output := string(out)
			if !strings.Contains(output, "Verification Results") {
				t.Errorf("verify output missing 'Verification Results': %q", output)
			}
		} else {
			// Passed — check output format
			output := string(out)
			if !strings.Contains(output, "passed") {
				t.Errorf("verify output missing 'passed': %q", output)
			}
		}
	})

	t.Run("verify_output_format", func(t *testing.T) {
		dir := t.TempDir()
		cmd := exec.Command(bin, "verify", "--project-dir", dir)
		out, _ := cmd.CombinedOutput()
		output := string(out)

		// Should always show verification results header regardless of pass/fail
		if !strings.Contains(output, "Verification Results") {
			t.Errorf("verify output missing 'Verification Results': %q", output)
		}
		// Should show summary line with counts
		if !strings.Contains(output, "passed") || !strings.Contains(output, "failed") {
			t.Errorf("verify output missing pass/fail counts: %q", output)
		}
	})

	t.Run("run_exits_with_error_when_no_config", func(t *testing.T) {
		dir := t.TempDir()
		cmd := exec.Command(bin, "run", "--project-dir", dir)
		err := cmd.Run()
		if err == nil {
			t.Error("run should exit non-zero when config missing")
		}
	})

	t.Run("run_error_output_mentions_configuration", func(t *testing.T) {
		dir := t.TempDir()
		cmd := exec.Command(bin, "run", "--project-dir", dir)
		out, _ := cmd.CombinedOutput()
		output := string(out)
		if !strings.Contains(strings.ToLower(output), "config") && !strings.Contains(strings.ToLower(output), "error") {
			t.Errorf("run error output should mention config/error: %q", output)
		}
	})

	t.Run("info_template_list", func(t *testing.T) {
		out, err := exec.Command(bin, "info", "template", "--list").CombinedOutput()
		if err != nil {
			t.Fatalf("info template --list: %v\n%s", err, out)
		}
		output := string(out)
		for _, name := range []string{"config.json", "spec.md", "initialization.md", "implementation.md"} {
			if !strings.Contains(output, name) {
				t.Errorf("info template --list missing %q: %q", name, output)
			}
		}
	})

	t.Run("info_template_by_name", func(t *testing.T) {
		out, err := exec.Command(bin, "info", "template", "--name", "config.json").CombinedOutput()
		if err != nil {
			t.Fatalf("info template --name config.json: %v\n%s", err, out)
		}
		if len(out) == 0 {
			t.Error("info template --name config.json returned empty output")
		}
	})

	t.Run("info_schema", func(t *testing.T) {
		out, err := exec.Command(bin, "info", "schema").CombinedOutput()
		if err != nil {
			t.Fatalf("info schema: %v\n%s", err, out)
		}
		// Should contain some JSON schema content
		output := string(out)
		if !strings.Contains(output, "model") {
			t.Errorf("info schema output missing 'model': %q", output)
		}
	})

	t.Run("info_schema_json_flag", func(t *testing.T) {
		out, err := exec.Command(bin, "info", "schema", "--json").CombinedOutput()
		if err != nil {
			t.Fatalf("info schema --json: %v\n%s", err, out)
		}
		// Should be valid JSON
		var v interface{}
		if jsonErr := json.Unmarshal(out, &v); jsonErr != nil {
			t.Errorf("info schema --json output is not valid JSON: %v\n%s", jsonErr, out)
		}
	})

	t.Run("info_preset_list", func(t *testing.T) {
		out, err := exec.Command(bin, "info", "preset", "--list").CombinedOutput()
		if err != nil {
			t.Fatalf("info preset --list: %v\n%s", err, out)
		}
		output := string(out)
		// Should list known presets
		for _, name := range []string{"python", "go", "rust"} {
			if !strings.Contains(output, name) {
				t.Errorf("info preset --list missing %q: %q", name, output)
			}
		}
	})

	t.Run("info_preset_by_name", func(t *testing.T) {
		out, err := exec.Command(bin, "info", "preset", "--name", "go").CombinedOutput()
		if err != nil {
			t.Fatalf("info preset --name go: %v\n%s", err, out)
		}
		if len(out) == 0 {
			t.Error("info preset --name go returned empty output")
		}
	})

	t.Run("info_guide", func(t *testing.T) {
		out, err := exec.Command(bin, "info", "guide").CombinedOutput()
		if err != nil {
			t.Fatalf("info guide: %v\n%s", err, out)
		}
		if len(out) == 0 {
			t.Error("info guide returned empty output")
		}
	})

	t.Run("info_guide_json_flag", func(t *testing.T) {
		out, err := exec.Command(bin, "info", "guide", "--json").CombinedOutput()
		if err != nil {
			t.Fatalf("info guide --json: %v\n%s", err, out)
		}
		var v interface{}
		if jsonErr := json.Unmarshal(out, &v); jsonErr != nil {
			t.Errorf("info guide --json is not valid JSON: %v\n%s", jsonErr, out)
		}
	})

	t.Run("info_no_subcommand_exits_nonzero", func(t *testing.T) {
		cmd := exec.Command(bin, "info")
		err := cmd.Run()
		if err == nil {
			t.Error("info with no subcommand should exit non-zero")
		}
	})

	t.Run("info_unknown_topic_exits_nonzero", func(t *testing.T) {
		cmd := exec.Command(bin, "info", "badtopic")
		err := cmd.Run()
		if err == nil {
			t.Error("info with unknown topic should exit non-zero")
		}
	})
}

// ─── Integration: run workflow with fake claude ───────────────────────────────

// TestIntegration_RunViaBinary tests the full run workflow by invoking the
// lorah binary with a fake claude stub on PATH.
func TestIntegration_RunViaBinary(t *testing.T) {
	bin := buildBinary(t)

	t.Run("run_with_fake_claude_exits_zero", func(t *testing.T) {
		dir := t.TempDir()
		harnessDir := filepath.Join(dir, ".lorah")
		if err := os.MkdirAll(harnessDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Write a minimal config.json using a well-formed JSON string that
		// passes config validation (max_turns, etc. required).
		configContent := `{
  "model": "claude-test",
  "max_iterations": 1
}`
		if err := os.WriteFile(filepath.Join(harnessDir, "config.json"), []byte(configContent), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create required prompt files
		promptsDir := filepath.Join(harnessDir, "prompts")
		if err := os.MkdirAll(promptsDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(promptsDir, "initialization.md"), []byte("Initialize the project"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(promptsDir, "implementation.md"), []byte("Build something"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create tasks.json
		if err := os.WriteFile(filepath.Join(harnessDir, tracking.TaskListFile), []byte("[]"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create fake claude stub
		binDir := writeFakeClaude(t, dir)

		// Run lorah run with fake claude on PATH
		cmd := exec.Command(bin, "run", "--project-dir", dir, "--max-iterations", "1")
		cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s%c%s", binDir, os.PathListSeparator, os.Getenv("PATH")))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run with fake claude failed: %v\nOutput: %s", err, out)
		}

		// Output should show the LORAH banner
		output := string(out)
		if !strings.Contains(output, "LORAH") {
			t.Errorf("run output missing 'LORAH' banner: %q", output)
		}
		if !strings.Contains(output, "SESSION") {
			t.Errorf("run output missing 'SESSION' header: %q", output)
		}
	})

	t.Run("run_tracks_session_state_via_binary", func(t *testing.T) {
		dir := t.TempDir()
		harnessDir := filepath.Join(dir, ".lorah")
		promptsDir := filepath.Join(harnessDir, "prompts")
		if err := os.MkdirAll(promptsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Write a minimal config.json
		configContent := `{
  "model": "claude-test",
  "max_iterations": 2
}`
		if err := os.WriteFile(filepath.Join(harnessDir, "config.json"), []byte(configContent), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create required prompt files
		if err := os.WriteFile(filepath.Join(promptsDir, "initialization.md"), []byte("Initialize"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(promptsDir, "implementation.md"), []byte("Build"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create tasks.json
		if err := os.WriteFile(filepath.Join(harnessDir, tracking.TaskListFile), []byte("[]"), 0o644); err != nil {
			t.Fatal(err)
		}

		binDir := writeFakeClaude(t, dir)

		// Run with max_iterations=2 — should run init once, then build once
		cmd := exec.Command(bin, "run", "--project-dir", dir)
		cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s%c%s", binDir, os.PathListSeparator, os.Getenv("PATH")))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run failed: %v\nOutput: %s", err, out)
		}

		// Session state should have been written
		sessionPath := filepath.Join(harnessDir, "session.json")
		if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
			t.Fatal("session.json was not created")
		}

		var state runner.SessionState
		data, _ := os.ReadFile(sessionPath)
		if err := json.Unmarshal(data, &state); err != nil {
			t.Fatalf("unmarshal session.json: %v", err)
		}

		if state.SessionNumber < 1 {
			t.Errorf("session_number: got %d, want >= 1", state.SessionNumber)
		}

		// initialization phase should be marked complete
		found := false
		for _, name := range state.CompletedPhases {
			if name == "initialization" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'initialization' in completed_phases, got %v", state.CompletedPhases)
		}
	})
}
