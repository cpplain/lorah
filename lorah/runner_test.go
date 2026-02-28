package lorah

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

// makeCfg creates a minimal HarnessConfig pointing at a temp directory.
func makeCfg(t *testing.T) *HarnessConfig {
	t.Helper()
	tmpDir := t.TempDir()
	harnessDir := filepath.Join(tmpDir, ".lorah")
	if err := os.MkdirAll(harnessDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return &HarnessConfig{
		ProjectDir: tmpDir,
		HarnessDir: harnessDir,
		Harness: HarnessSettings{
			AutoContinueDelay: 0,
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
				"--max-turns": float64(100),
			},
			Settings: map[string]any{
				"model": "claude-test",
			},
		},
	}
}

// ─── SessionState load/save ───────────────────────────────────────────────────

func TestLoadSession_NoFile(t *testing.T) {
	cfg := makeCfg(t)

	state, err := LoadSession(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.SessionNumber != 0 {
		t.Errorf("SessionNumber: got %d, want 0", state.SessionNumber)
	}
	if len(state.CompletedPhases) != 0 {
		t.Errorf("CompletedPhases: got %v, want []", state.CompletedPhases)
	}
}

func TestLoadSession_ValidFile(t *testing.T) {
	cfg := makeCfg(t)

	initial := SessionState{SessionNumber: 3, CompletedPhases: []string{"init", "build"}}
	data, _ := json.MarshalIndent(initial, "", "  ")
	stateFile := filepath.Join(cfg.HarnessDir, "session.json")
	if err := os.WriteFile(stateFile, data, 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := LoadSession(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.SessionNumber != 3 {
		t.Errorf("SessionNumber: got %d, want 3", state.SessionNumber)
	}
	if len(state.CompletedPhases) != 2 {
		t.Errorf("CompletedPhases length: got %d, want 2", len(state.CompletedPhases))
	}
}

func TestLoadSession_CorruptFile(t *testing.T) {
	cfg := makeCfg(t)

	stateFile := filepath.Join(cfg.HarnessDir, "session.json")
	if err := os.WriteFile(stateFile, []byte("not-valid-json{{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := LoadSession(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return fresh state
	if state.SessionNumber != 0 {
		t.Errorf("SessionNumber: got %d, want 0", state.SessionNumber)
	}
}

func TestSaveSession_RoundTrip(t *testing.T) {
	cfg := makeCfg(t)

	state := SessionState{SessionNumber: 7, CompletedPhases: []string{"phase-a"}}
	SaveSession(cfg, state)

	loaded, err := LoadSession(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.SessionNumber != 7 {
		t.Errorf("SessionNumber: got %d, want 7", loaded.SessionNumber)
	}
	if len(loaded.CompletedPhases) != 1 || loaded.CompletedPhases[0] != "phase-a" {
		t.Errorf("CompletedPhases: got %v, want [phase-a]", loaded.CompletedPhases)
	}
}

func TestSaveSession_CreatesFileInHarnessDir(t *testing.T) {
	cfg := makeCfg(t)

	state := SessionState{SessionNumber: 1, CompletedPhases: []string{}}
	SaveSession(cfg, state)

	stateFile := filepath.Join(cfg.HarnessDir, "session.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("session.json was not created")
	}
}

// ─── EvaluateCondition ────────────────────────────────────────────────────────

func TestEvaluateCondition_Empty(t *testing.T) {
	dir := t.TempDir()
	ok, err := EvaluateCondition("", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("empty condition should be true")
	}
}

func TestEvaluateCondition_ExistsTrue(t *testing.T) {
	dir := t.TempDir()
	// Create a file inside dir
	f := filepath.Join(dir, "found.txt")
	if err := os.WriteFile(f, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	ok, err := EvaluateCondition("exists:found.txt", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("exists: should be true when file exists")
	}
}

func TestEvaluateCondition_ExistsFalse(t *testing.T) {
	dir := t.TempDir()

	ok, err := EvaluateCondition("exists:missing.txt", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("exists: should be false when file is missing")
	}
}

func TestEvaluateCondition_NotExistsTrue(t *testing.T) {
	dir := t.TempDir()

	ok, err := EvaluateCondition("not_exists:missing.txt", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("not_exists: should be true when file is missing")
	}
}

func TestEvaluateCondition_NotExistsFalse(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "present.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	ok, err := EvaluateCondition("not_exists:present.txt", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("not_exists: should be false when file exists")
	}
}

func TestEvaluateCondition_UnknownPrefix(t *testing.T) {
	dir := t.TempDir()
	_, err := EvaluateCondition("invalid:foo", dir)
	if err == nil {
		t.Error("expected error for unknown prefix")
	}
}

func TestEvaluateCondition_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	_, err := EvaluateCondition("exists:../../etc/passwd", dir)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

// ─── SelectPhase ─────────────────────────────────────────────────────────────

// mockTracker implements ProgressTracker for testing.
type mockTracker struct {
	initialized bool
	complete    bool
}

func (m *mockTracker) GetSummary() (int, int) { return 0, 0 }
func (m *mockTracker) IsInitialized() bool    { return m.initialized }
func (m *mockTracker) IsComplete() bool       { return m.complete }
func (m *mockTracker) Summary() string        { return "" }

func TestSelectPhase_InitializationWhenNotInitialized(t *testing.T) {
	tracker := &mockTracker{initialized: false}
	state := SessionState{CompletedPhases: []string{}}

	phaseName, promptFile := SelectPhase(tracker, state)

	if phaseName != "initialization" {
		t.Errorf("phaseName = %q, want initialization", phaseName)
	}
	if promptFile != InitializationPromptFile {
		t.Errorf("promptFile = %q, want %q", promptFile, InitializationPromptFile)
	}
}

func TestSelectPhase_ImplementationWhenInitialized(t *testing.T) {
	tracker := &mockTracker{initialized: true}
	state := SessionState{CompletedPhases: []string{}}

	phaseName, promptFile := SelectPhase(tracker, state)

	if phaseName != "implementation" {
		t.Errorf("phaseName = %q, want implementation", phaseName)
	}
	if promptFile != ImplementationPromptFile {
		t.Errorf("promptFile = %q, want %q", promptFile, ImplementationPromptFile)
	}
}

func TestSelectPhase_ImplementationWhenInitCompleted(t *testing.T) {
	tracker := &mockTracker{initialized: false}
	state := SessionState{CompletedPhases: []string{"initialization"}}

	phaseName, promptFile := SelectPhase(tracker, state)

	if phaseName != "implementation" {
		t.Errorf("phaseName = %q, want implementation", phaseName)
	}
	if promptFile != ImplementationPromptFile {
		t.Errorf("promptFile = %q, want %q", promptFile, ImplementationPromptFile)
	}
}

// ─── BackoffDuration ──────────────────────────────────────────────────────────

func TestBackoffDuration_FirstError(t *testing.T) {
	cfg := ErrorRecoveryConfig{
		InitialBackoffSeconds: 5.0,
		MaxBackoffSeconds:     120.0,
		BackoffMultiplier:     2.0,
	}
	d := BackoffDuration(1, cfg)
	expected := 5 * time.Second
	if d != expected {
		t.Errorf("got %v, want %v", d, expected)
	}
}

func TestBackoffDuration_SecondError(t *testing.T) {
	cfg := ErrorRecoveryConfig{
		InitialBackoffSeconds: 5.0,
		MaxBackoffSeconds:     120.0,
		BackoffMultiplier:     2.0,
	}
	d := BackoffDuration(2, cfg)
	expected := 10 * time.Second
	if d != expected {
		t.Errorf("got %v, want %v", d, expected)
	}
}

func TestBackoffDuration_Capped(t *testing.T) {
	cfg := ErrorRecoveryConfig{
		InitialBackoffSeconds: 5.0,
		MaxBackoffSeconds:     30.0,
		BackoffMultiplier:     2.0,
	}
	// 5 * 2^5 = 160, capped at 30
	d := BackoffDuration(6, cfg)
	expected := 30 * time.Second
	if d != expected {
		t.Errorf("got %v, want %v", d, expected)
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello world", 5, "hello"},
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"", 5, ""},
		// Multi-byte UTF-8 characters
		{"日本語テスト", 3, "日本語"},
		{"emoji: 🎉🎊🎈", 9, "emoji: 🎉🎊"},
		{"mixed αβγ test", 10, "mixed αβγ "},
	}
	for _, tt := range tests {
		got := truncateString(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}
