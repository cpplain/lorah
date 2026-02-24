// Package test contains integration tests for lorah.
package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cpplain/lorah/lorah"
)

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

// makeInitedProject runs lorah.InitProject in a temp dir and returns the dir.
func makeInitedProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := lorah.InitProject(dir); err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	return dir
}

// loadSessionState reads and unmarshals .lorah/session.json.
func loadSessionState(t *testing.T, projectDir string) lorah.SessionState {
	t.Helper()
	path := filepath.Join(projectDir, ".lorah", "session.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read session.json: %v", err)
	}
	var state lorah.SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("unmarshal session.json: %v", err)
	}
	return state
}
