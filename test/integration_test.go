// Package test contains integration tests that verify lorah workflows by
// calling library functions directly (no binary building).
package test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cpplain/lorah/lorah"
)

// ─── Integration: init workflow ──────────────────────────────────────────────

func TestIntegration_InitWorkflow(t *testing.T) {
	t.Run("creates_expected_files", func(t *testing.T) {
		dir := t.TempDir()
		if err := lorah.InitProject(dir); err != nil {
			t.Fatalf("InitProject: %v", err)
		}

		harnessDir := filepath.Join(dir, ".lorah")
		expected := []string{
			filepath.Join(harnessDir, "spec.md"),
			filepath.Join(harnessDir, lorah.TaskListFile),
			filepath.Join(harnessDir, lorah.AgentProgressFile),
			filepath.Join(harnessDir, "prompts", "initialization.md"),
			filepath.Join(harnessDir, "prompts", "implementation.md"),
		}
		for _, f := range expected {
			if _, err := os.Stat(f); os.IsNotExist(err) {
				t.Errorf("expected file missing: %s", f)
			}
		}
	})

	t.Run("defaults_loaded_after_init", func(t *testing.T) {
		dir := makeInitedProject(t)
		// lorah.LoadConfig should succeed using defaults (no config.json created by init)
		cfg, err := lorah.LoadConfig(dir, nil)
		if err != nil {
			t.Fatalf("LoadConfig on freshly initialized project: %v", err)
		}
		// Verify defaults are applied
		maxTurns, ok := cfg.Claude.Flags["--max-turns"].(float64)
		if !ok || maxTurns == 0 {
			t.Error("expected default --max-turns to be set")
		}
	})

	t.Run("refuses_overwrite", func(t *testing.T) {
		dir := makeInitedProject(t)
		// Second init should fail
		err := lorah.InitProject(dir)
		if err == nil {
			t.Error("expected error on second InitProject call, got nil")
		}
	})

	t.Run("verify_passes_after_init", func(t *testing.T) {
		dir := makeInitedProject(t)
		results := lorah.RunVerify(dir)

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
		if err := os.WriteFile(filepath.Join(harnessDir, lorah.TaskListFile), []byte("[]"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Write a fake claude script to PATH
		binDir := writeFakeClaude(t, dir)
		origPath := os.Getenv("PATH")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

		// Build a minimal config with max-iterations=1 and one phase
		maxIter := 1
		cfg := &lorah.HarnessConfig{
			ProjectDir: dir,
			HarnessDir: harnessDir,
			Harness: lorah.HarnessSettings{
				MaxIterations:     &maxIter,
				AutoContinueDelay: 0,
				ErrorRecovery: lorah.ErrorRecoveryConfig{
					MaxConsecutiveErrors:  3,
					InitialBackoffSeconds: 1.0,
					MaxBackoffSeconds:     10.0,
					BackoffMultiplier:     2.0,
					MaxErrorMessageLength: 2000,
				},
			},
			Claude: lorah.ClaudeSection{
				Flags: map[string]any{
					"--max-turns": float64(100),
				},
				Settings: map[string]any{
					"model": "claude-test",
					"permissions": map[string]any{
						"defaultMode": "default",
					},
				},
			},
		}

		if err := lorah.RunAgent(context.Background(), cfg); err != nil {
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
		if err := os.WriteFile(filepath.Join(harnessDir, lorah.TaskListFile), []byte("[]"), 0o644); err != nil {
			t.Fatal(err)
		}

		binDir := writeFakeClaude(t, dir)
		origPath := os.Getenv("PATH")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

		maxIter := 1
		cfg := &lorah.HarnessConfig{
			ProjectDir: dir,
			HarnessDir: harnessDir,
			Harness: lorah.HarnessSettings{
				MaxIterations:     &maxIter,
				AutoContinueDelay: 0,
				ErrorRecovery: lorah.ErrorRecoveryConfig{
					MaxConsecutiveErrors:  3,
					InitialBackoffSeconds: 1.0,
					MaxBackoffSeconds:     10.0,
					BackoffMultiplier:     2.0,
					MaxErrorMessageLength: 2000,
				},
			},
			Claude: lorah.ClaudeSection{
				Flags: map[string]any{
					"--max-turns": float64(100),
				},
				Settings: map[string]any{
					"model": "claude-test",
					"permissions": map[string]any{
						"defaultMode": "default",
					},
				},
			},
		}

		if err := lorah.RunAgent(context.Background(), cfg); err != nil {
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
		if err := os.WriteFile(filepath.Join(harnessDir, lorah.TaskListFile), []byte("[]"), 0o644); err != nil {
			t.Fatal(err)
		}

		binDir := writeFakeClaude(t, dir)
		origPath := os.Getenv("PATH")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

		maxIter := 3
		cfg := &lorah.HarnessConfig{
			ProjectDir: dir,
			HarnessDir: harnessDir,
			Harness: lorah.HarnessSettings{
				MaxIterations:     &maxIter,
				AutoContinueDelay: 0,
				ErrorRecovery: lorah.ErrorRecoveryConfig{
					MaxConsecutiveErrors:  3,
					InitialBackoffSeconds: 1.0,
					MaxBackoffSeconds:     10.0,
					BackoffMultiplier:     2.0,
					MaxErrorMessageLength: 2000,
				},
			},
			Claude: lorah.ClaudeSection{
				Flags: map[string]any{
					"--max-turns": float64(100),
				},
				Settings: map[string]any{
					"model": "claude-test",
					"permissions": map[string]any{
						"defaultMode": "default",
					},
				},
			},
		}

		if err := lorah.RunAgent(context.Background(), cfg); err != nil {
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
		if err := os.WriteFile(filepath.Join(harnessDir, lorah.TaskListFile), []byte(taskList), 0o644); err != nil {
			t.Fatal(err)
		}

		binDir := writeFakeClaude(t, dir)
		origPath := os.Getenv("PATH")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

		cfg := &lorah.HarnessConfig{
			ProjectDir: dir,
			HarnessDir: harnessDir,
			Harness: lorah.HarnessSettings{
				AutoContinueDelay: 0,
				ErrorRecovery: lorah.ErrorRecoveryConfig{
					MaxConsecutiveErrors:  3,
					InitialBackoffSeconds: 1.0,
					MaxBackoffSeconds:     10.0,
					BackoffMultiplier:     2.0,
					MaxErrorMessageLength: 2000,
				},
			},
			Claude: lorah.ClaudeSection{
				Flags: map[string]any{
					"--max-turns": float64(100),
				},
				Settings: map[string]any{
					"model": "claude-test",
					"permissions": map[string]any{
						"defaultMode": "default",
					},
				},
			},
		}

		// RunAgent should exit cleanly when tracker.IsComplete() returns true
		if err := lorah.RunAgent(context.Background(), cfg); err != nil {
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

	t.Run("uses_defaults_when_no_config", func(t *testing.T) {
		dir := t.TempDir()
		// Create .lorah directory but no config.json
		harnessDir := filepath.Join(dir, ".lorah")
		if err := os.MkdirAll(harnessDir, 0o755); err != nil {
			t.Fatal(err)
		}
		// LoadConfig should succeed using defaults
		cfg, err := lorah.LoadConfig(dir, nil)
		if err != nil {
			t.Errorf("LoadConfig with no config file: %v", err)
		}
		// Verify defaults are applied
		maxTurns, ok := cfg.Claude.Flags["--max-turns"].(float64)
		if !ok || maxTurns == 0 {
			t.Error("expected default Claude.Flags[--max-turns] to be set")
		}
	})
}
