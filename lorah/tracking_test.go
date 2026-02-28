package lorah

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile creates a file with the given content in the temp directory.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	return path
}

// --- JsonChecklistTracker tests ---

func TestJsonChecklistTracker_GetSummary_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	tr := NewJsonChecklistTracker(dir)
	passing, total := tr.GetSummary()
	if passing != 0 || total != 0 {
		t.Errorf("GetSummary() = (%d, %d), want (0, 0)", passing, total)
	}
}

func TestJsonChecklistTracker_GetSummary_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, TaskListFile, "not valid json")
	tr := NewJsonChecklistTracker(dir)
	passing, total := tr.GetSummary()
	if passing != 0 || total != 0 {
		t.Errorf("GetSummary() = (%d, %d), want (0, 0)", passing, total)
	}
}

func TestJsonChecklistTracker_GetSummary_Empty(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, TaskListFile, "[]")
	tr := NewJsonChecklistTracker(dir)
	passing, total := tr.GetSummary()
	if passing != 0 || total != 0 {
		t.Errorf("GetSummary() = (%d, %d), want (0, 0)", passing, total)
	}
}

func TestJsonChecklistTracker_GetSummary_Mixed(t *testing.T) {
	dir := t.TempDir()
	content := `[
		{"name": "a", "passes": true},
		{"name": "b", "passes": false},
		{"name": "c", "passes": true},
		{"name": "d"}
	]`
	writeFile(t, dir, TaskListFile, content)
	tr := NewJsonChecklistTracker(dir)
	passing, total := tr.GetSummary()
	if passing != 2 || total != 4 {
		t.Errorf("GetSummary() = (%d, %d), want (2, 4)", passing, total)
	}
}

func TestJsonChecklistTracker_GetSummary_AllPassing(t *testing.T) {
	dir := t.TempDir()
	content := `[{"passes": true}, {"passes": true}]`
	writeFile(t, dir, TaskListFile, content)
	tr := NewJsonChecklistTracker(dir)
	passing, total := tr.GetSummary()
	if passing != 2 || total != 2 {
		t.Errorf("GetSummary() = (%d, %d), want (2, 2)", passing, total)
	}
}

func TestJsonChecklistTracker_IsInitialized(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"file not found", "", false},
		{"empty array", "[]", false},
		{"has items", `[{"passes": false}]`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.content != "" {
				writeFile(t, dir, TaskListFile, tt.content)
			}
			tr := NewJsonChecklistTracker(dir)
			got := tr.IsInitialized()
			if got != tt.want {
				t.Errorf("IsInitialized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJsonChecklistTracker_IsComplete(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"file not found", "", false},
		{"empty array", "[]", false},
		{"some failing", `[{"passes": true}, {"passes": false}]`, false},
		{"all passing", `[{"passes": true}, {"passes": true}]`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.content != "" {
				writeFile(t, dir, TaskListFile, tt.content)
			}
			tr := NewJsonChecklistTracker(dir)
			got := tr.IsComplete()
			if got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJsonChecklistTracker_Summary_WithItems(t *testing.T) {
	dir := t.TempDir()
	content := `[{"passes": true}, {"passes": false}, {"passes": true}]`
	writeFile(t, dir, TaskListFile, content)
	tr := NewJsonChecklistTracker(dir)
	output := tr.Summary()
	if !strings.Contains(output, "2/3") {
		t.Errorf("Summary() output %q missing '2/3'", output)
	}
	if !strings.Contains(output, "66.7%") {
		t.Errorf("Summary() output %q missing '66.7%%'", output)
	}
}

func TestJsonChecklistTracker_Summary_NoFile(t *testing.T) {
	dir := t.TempDir()
	tr := NewJsonChecklistTracker(dir)
	output := tr.Summary()
	if !strings.Contains(output, "not yet created") {
		t.Errorf("Summary() output %q missing 'not yet created'", output)
	}
}
