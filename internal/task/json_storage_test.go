package task

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newJSONStorage(t *testing.T) *JSONStorage {
	t.Helper()
	dir := t.TempDir()
	return &JSONStorage{path: filepath.Join(dir, "tasks.json")}
}

func TestLoadNonExistent(t *testing.T) {
	s := newJSONStorage(t)
	list, err := s.Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if list.Version != "1.0" {
		t.Errorf("expected Version %q, got %q", "1.0", list.Version)
	}
	if len(list.Tasks) != 0 {
		t.Errorf("expected empty tasks, got %d", len(list.Tasks))
	}
}

func TestLoadExistingFile(t *testing.T) {
	s := newJSONStorage(t)
	original := &TaskList{
		Name:    "Test Project",
		Version: "1.0",
		Tasks: []Task{
			{ID: "a1b2c3d4", Subject: "first task", Status: StatusPending, LastUpdated: time.Now().UTC().Truncate(time.Second)},
		},
	}
	if err := s.Save(original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	list, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if list.Name != "Test Project" {
		t.Errorf("expected Name %q, got %q", "Test Project", list.Name)
	}
	if len(list.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(list.Tasks))
	}
	if list.Tasks[0].ID != "a1b2c3d4" {
		t.Errorf("expected task ID %q, got %q", "a1b2c3d4", list.Tasks[0].ID)
	}
}

func TestSaveUpdatesLastUpdatedAndRoundTrips(t *testing.T) {
	s := newJSONStorage(t)
	before := time.Now().UTC().Add(-time.Second)
	list := &TaskList{Version: "1.0", Tasks: []Task{}}
	if err := s.Save(list); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if list.LastUpdated.Before(before) {
		t.Errorf("LastUpdated not updated: got %v", list.LastUpdated)
	}

	// Verify the file contains indented JSON (check for at least one newline + spaces)
	data, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Fatal("expected non-empty file")
	}
	// Indented JSON has newlines
	hasNewline := false
	for _, c := range content {
		if c == '\n' {
			hasNewline = true
			break
		}
	}
	if !hasNewline {
		t.Error("expected indented JSON with newlines")
	}

	// Round-trip
	loaded, err := s.Load()
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if !loaded.LastUpdated.Equal(list.LastUpdated) {
		t.Errorf("LastUpdated round-trip mismatch: saved %v, loaded %v", list.LastUpdated, loaded.LastUpdated)
	}
}

func TestCreate(t *testing.T) {
	s := newJSONStorage(t)
	task := &Task{ID: "aaaabbbb", Subject: "new task", Status: StatusPending}
	before := time.Now().UTC().Add(-time.Second)
	if err := s.Create(task); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if task.LastUpdated.Before(before) {
		t.Errorf("Create did not set LastUpdated: got %v", task.LastUpdated)
	}

	// Verify persisted
	list, err := s.Load()
	if err != nil {
		t.Fatalf("Load after Create: %v", err)
	}
	if len(list.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(list.Tasks))
	}
	if list.Tasks[0].ID != "aaaabbbb" {
		t.Errorf("expected task ID %q, got %q", "aaaabbbb", list.Tasks[0].ID)
	}
}

func TestCreateDuplicateID(t *testing.T) {
	s := newJSONStorage(t)
	task := &Task{ID: "aaaabbbb", Subject: "task one", Status: StatusPending}
	if err := s.Create(task); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	dup := &Task{ID: "aaaabbbb", Subject: "task two", Status: StatusPending}
	if err := s.Create(dup); err == nil {
		t.Fatal("expected error for duplicate ID, got nil")
	}
}

func TestGet(t *testing.T) {
	s := newJSONStorage(t)
	task := &Task{ID: "ccccdddd", Subject: "findable", Status: StatusInProgress}
	if err := s.Create(task); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.Get("ccccdddd")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Subject != "findable" {
		t.Errorf("expected Subject %q, got %q", "findable", got.Subject)
	}
}

func TestGetNotFound(t *testing.T) {
	s := newJSONStorage(t)
	_, err := s.Get("notexist")
	if err == nil {
		t.Fatal("expected error for missing task, got nil")
	}
}

func TestListFilterByStatus(t *testing.T) {
	s := newJSONStorage(t)
	tasks := []Task{
		{ID: "id000001", Subject: "pending task", Status: StatusPending},
		{ID: "id000002", Subject: "in-progress task", Status: StatusInProgress},
		{ID: "id000003", Subject: "completed task", Status: StatusCompleted},
	}
	for i := range tasks {
		if err := s.Create(&tasks[i]); err != nil {
			t.Fatalf("Create task %d: %v", i, err)
		}
	}

	results, err := s.List(Filter{Status: []TaskStatus{StatusPending}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "id000001" {
		t.Errorf("expected ID %q, got %q", "id000001", results[0].ID)
	}
}

func TestListFilterByPhaseID(t *testing.T) {
	s := newJSONStorage(t)
	tasks := []Task{
		{ID: "id000001", Subject: "in phase", Status: StatusPending, PhaseID: "phase001"},
		{ID: "id000002", Subject: "no phase", Status: StatusPending},
	}
	for i := range tasks {
		if err := s.Create(&tasks[i]); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	results, err := s.List(Filter{PhaseID: "phase001"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 1 || results[0].ID != "id000001" {
		t.Errorf("unexpected results: %v", results)
	}
}

func TestListFilterBySectionID(t *testing.T) {
	s := newJSONStorage(t)
	tasks := []Task{
		{ID: "id000001", Subject: "in section", Status: StatusPending, SectionID: "sect0001"},
		{ID: "id000002", Subject: "no section", Status: StatusPending},
	}
	for i := range tasks {
		if err := s.Create(&tasks[i]); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	results, err := s.List(Filter{SectionID: "sect0001"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 1 || results[0].ID != "id000001" {
		t.Errorf("unexpected results: %v", results)
	}
}

func TestListFilterLimit(t *testing.T) {
	s := newJSONStorage(t)
	for i := 0; i < 5; i++ {
		task := &Task{ID: fmt.Sprintf("id00000%d", i), Subject: fmt.Sprintf("task %d", i), Status: StatusPending}
		if err := s.Create(task); err != nil {
			t.Fatalf("Create task %d: %v", i, err)
		}
	}

	results, err := s.List(Filter{Limit: 3})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results (limit), got %d", len(results))
	}
}

func TestListNoLimit(t *testing.T) {
	s := newJSONStorage(t)
	for i := 0; i < 5; i++ {
		task := &Task{ID: fmt.Sprintf("id00000%d", i), Subject: fmt.Sprintf("task %d", i), Status: StatusPending}
		if err := s.Create(task); err != nil {
			t.Fatalf("Create task %d: %v", i, err)
		}
	}

	results, err := s.List(Filter{Limit: 0})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results (no limit), got %d", len(results))
	}
}

func TestUpdate(t *testing.T) {
	s := newJSONStorage(t)
	task := &Task{ID: "updateid", Subject: "original", Status: StatusPending}
	if err := s.Create(task); err != nil {
		t.Fatalf("Create: %v", err)
	}

	before := time.Now().UTC().Add(-time.Second)
	task.Subject = "updated"
	task.Status = StatusCompleted
	if err := s.Update(task); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if task.LastUpdated.Before(before) {
		t.Errorf("Update did not set LastUpdated: got %v", task.LastUpdated)
	}

	got, err := s.Get("updateid")
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.Subject != "updated" {
		t.Errorf("expected Subject %q, got %q", "updated", got.Subject)
	}
	if got.Status != StatusCompleted {
		t.Errorf("expected Status %q, got %q", StatusCompleted, got.Status)
	}
}

func TestUpdateNotFound(t *testing.T) {
	s := newJSONStorage(t)
	task := &Task{ID: "missing0", Subject: "ghost", Status: StatusPending}
	if err := s.Update(task); err == nil {
		t.Fatal("expected error for missing task, got nil")
	}
}

func TestDelete(t *testing.T) {
	s := newJSONStorage(t)
	task := &Task{ID: "deleteid", Subject: "to delete", Status: StatusPending}
	if err := s.Create(task); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.Delete("deleteid"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := s.Get("deleteid")
	if err == nil {
		t.Fatal("expected error after Delete, got nil")
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := newJSONStorage(t)
	if err := s.Delete("notfound"); err == nil {
		t.Fatal("expected error for missing task, got nil")
	}
}
