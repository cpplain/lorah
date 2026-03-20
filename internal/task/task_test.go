package task

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"
)

// TestTaskStatusConstants verifies the string values of TaskStatus constants.
func TestTaskStatusConstants(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   string
	}{
		{StatusPending, "pending"},
		{StatusInProgress, "in_progress"},
		{StatusCompleted, "completed"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("status %q = %q, want %q", tt.status, string(tt.status), tt.want)
		}
	}
}

// TestPhaseJSONOmitEmpty verifies that optional Phase fields are absent from JSON when empty.
func TestPhaseJSONOmitEmpty(t *testing.T) {
	phase := Phase{ID: "d4e5f6a7"}

	data, err := json.Marshal(phase)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// ID must always be present.
	if _, ok := m["id"]; !ok {
		t.Errorf("expected key %q in JSON, but missing", "id")
	}

	// Optional fields must be absent when empty.
	for _, key := range []string{"name", "description"} {
		if _, ok := m[key]; ok {
			t.Errorf("unexpected key %q in JSON when field is empty", key)
		}
	}
}

// TestSectionJSONOmitEmpty verifies that optional Section fields are absent from JSON when empty.
func TestSectionJSONOmitEmpty(t *testing.T) {
	section := Section{ID: "b8c9d0e1", PhaseID: "d4e5f6a7"}

	data, err := json.Marshal(section)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// ID and PhaseID must always be present.
	for _, key := range []string{"id", "phaseId"} {
		if _, ok := m[key]; !ok {
			t.Errorf("expected key %q in JSON, but missing", key)
		}
	}

	// Optional fields must be absent when empty.
	for _, key := range []string{"name", "description"} {
		if _, ok := m[key]; ok {
			t.Errorf("unexpected key %q in JSON when field is empty", key)
		}
	}
}

// TestTaskJSONOmitEmpty verifies that optional Task fields are absent from JSON when empty.
func TestTaskJSONOmitEmpty(t *testing.T) {
	task := Task{
		ID:          "a3f7b2c1",
		Subject:     "Do something",
		Status:      StatusPending,
		LastUpdated: time.Date(2026, 3, 10, 14, 22, 0, 0, time.UTC),
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// Required fields must be present.
	for _, key := range []string{"id", "subject", "status", "lastUpdated"} {
		if _, ok := m[key]; !ok {
			t.Errorf("expected key %q in JSON, but missing", key)
		}
	}

	// Optional fields must be absent when empty.
	for _, key := range []string{"phaseId", "sectionId", "notes"} {
		if _, ok := m[key]; ok {
			t.Errorf("unexpected key %q in JSON when field is empty", key)
		}
	}
}

// TestTaskJSONRoundTrip verifies JSON serialization and deserialization of a Task with all fields.
func TestTaskJSONRoundTrip(t *testing.T) {
	original := Task{
		ID:          "a3f7b2c1",
		Subject:     "Implement stream-JSON output parsing",
		Status:      StatusCompleted,
		PhaseID:     "d4e5f6a7",
		SectionID:   "b8c9d0e1",
		Notes:       "Scans stdout line-by-line.",
		LastUpdated: time.Date(2026, 3, 10, 14, 22, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var got Task
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if got.ID != original.ID {
		t.Errorf("ID: got %q, want %q", got.ID, original.ID)
	}
	if got.Subject != original.Subject {
		t.Errorf("Subject: got %q, want %q", got.Subject, original.Subject)
	}
	if got.Status != original.Status {
		t.Errorf("Status: got %q, want %q", got.Status, original.Status)
	}
	if got.PhaseID != original.PhaseID {
		t.Errorf("PhaseID: got %q, want %q", got.PhaseID, original.PhaseID)
	}
	if got.SectionID != original.SectionID {
		t.Errorf("SectionID: got %q, want %q", got.SectionID, original.SectionID)
	}
	if got.Notes != original.Notes {
		t.Errorf("Notes: got %q, want %q", got.Notes, original.Notes)
	}
	if !got.LastUpdated.Equal(original.LastUpdated) {
		t.Errorf("LastUpdated: got %v, want %v", got.LastUpdated, original.LastUpdated)
	}
}

// TestTaskListJSONOmitEmpty verifies that optional TaskList fields are absent from JSON when empty.
func TestTaskListJSONOmitEmpty(t *testing.T) {
	list := TaskList{
		Tasks:       []Task{},
		Version:     "1.0",
		LastUpdated: time.Date(2026, 3, 10, 15, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// Required fields must be present.
	for _, key := range []string{"tasks", "version", "lastUpdated"} {
		if _, ok := m[key]; !ok {
			t.Errorf("expected key %q in JSON, but missing", key)
		}
	}

	// Optional fields must be absent when empty.
	for _, key := range []string{"name", "description", "phases", "sections"} {
		if _, ok := m[key]; ok {
			t.Errorf("unexpected key %q in JSON when field is empty", key)
		}
	}
}

// TestGenerateID verifies that generateID returns an 8-character lowercase hex string.
func TestGenerateID(t *testing.T) {
	hexPattern := regexp.MustCompile(`^[0-9a-f]{8}$`)

	id := generateID()
	if !hexPattern.MatchString(id) {
		t.Errorf("generateID() = %q, want 8-char lowercase hex", id)
	}
}

// TestGenerateIDUnique verifies that generateID produces unique values.
func TestGenerateIDUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := range 100 {
		id := generateID()
		if seen[id] {
			t.Errorf("generateID() returned duplicate value %q on iteration %d", id, i)
		}
		seen[id] = true
	}
}
