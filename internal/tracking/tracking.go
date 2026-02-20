// Package tracking provides progress tracker implementation for monitoring agent progress.
// Tracks progress via a JSON array with a boolean "passes" field in tasks.json.
package tracking

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// TaskListFile is the fixed name of the tracking checklist file.
	TaskListFile = "tasks.json"
	// PassingField is the JSON field name used to indicate a task is passing.
	PassingField = "passes"
	// AgentProgressFile is the fixed name of the agent progress notes file.
	AgentProgressFile = "progress.md"
)

// Default file contents for tracking files
const (
	defaultTaskListContent      = "[]"
	defaultAgentProgressContent = `# Review Progress

## Inventory Complete
- Total issues: 0
  - Security: 0
  - Bugs: 0
  - Logic: 0
  - Performance: 0
  - Idiom: 0
  - Consistency: 0

## Summary

(The initialization phase will populate this section)

## Fix Session Log

(Each fix session will append progress here)
`
)

// ProgressTracker is the interface for progress trackers.
type ProgressTracker interface {
	// GetSummary returns (passing_count, total_count).
	GetSummary() (int, int)

	// IsInitialized returns true if the tracking file exists and is valid.
	IsInitialized() bool

	// IsComplete returns true when all items are passing and there is at least one item.
	IsComplete() bool

	// DisplaySummary prints a progress summary to stdout.
	DisplaySummary()
}

// JsonChecklistTracker tracks progress via a JSON array with a boolean passing field.
type JsonChecklistTracker struct {
	filePath     string
	passingField string
	// Cached data for performance
	cachedPassing int
	cachedTotal   int
	cachedModTime int64 // Unix nanoseconds
	cachedSize    int64 // File size in bytes
}

// NewTracker creates a ProgressTracker for the given harness directory.
// Currently returns a JsonChecklistTracker. Future implementations
// (NotesFileTracker, NoneTracker) can be selected via configuration.
func NewTracker(harnessDir string) ProgressTracker {
	return NewJsonChecklistTracker(harnessDir)
}

// NewJsonChecklistTracker creates a ProgressTracker for the given harness directory.
// The tracker monitors .lorah/tasks.json with the "passes" field.
func NewJsonChecklistTracker(harnessDir string) ProgressTracker {
	filePath := filepath.Join(harnessDir, TaskListFile)
	return &JsonChecklistTracker{
		filePath:     filePath,
		passingField: PassingField,
	}
}

// EnsureTrackingFiles creates tracking files if they don't exist.
// This is a safety net to ensure tasks.json and progress.md
// are present before the agent runs, using consistent default content.
// Returns error only if file creation fails (not if files already exist).
func EnsureTrackingFiles(harnessDir string) error {
	taskListPath := filepath.Join(harnessDir, TaskListFile)
	if _, err := os.Stat(taskListPath); os.IsNotExist(err) {
		if err := os.WriteFile(taskListPath, []byte(defaultTaskListContent), 0o644); err != nil {
			return fmt.Errorf("failed to create %s: %w", TaskListFile, err)
		}
	}

	agentProgressPath := filepath.Join(harnessDir, AgentProgressFile)
	if _, err := os.Stat(agentProgressPath); os.IsNotExist(err) {
		if err := os.WriteFile(agentProgressPath, []byte(defaultAgentProgressContent), 0o644); err != nil {
			return fmt.Errorf("failed to create %s: %w", AgentProgressFile, err)
		}
	}

	return nil
}

// GetSummary returns (passing_count, total_count) by reading and parsing the JSON file.
// Results are cached and only re-parsed when the file modification time changes.
func (t *JsonChecklistTracker) GetSummary() (int, int) {
	// Check file modification time for cache invalidation
	info, err := os.Stat(t.filePath)
	if err != nil {
		// File doesn't exist or can't be accessed
		t.cachedPassing = 0
		t.cachedTotal = 0
		t.cachedModTime = 0
		t.cachedSize = 0
		return 0, 0
	}

	modTime := info.ModTime().UnixNano()
	fileSize := info.Size()

	// Return cached values if file hasn't been modified
	// Check both modTime and size to handle edge case where file is modified within same nanosecond
	if modTime == t.cachedModTime && fileSize == t.cachedSize && t.cachedModTime != 0 {
		return t.cachedPassing, t.cachedTotal
	}

	// File has been modified or cache is empty - re-read and parse
	data, err := os.ReadFile(t.filePath)
	if err != nil {
		t.cachedPassing = 0
		t.cachedTotal = 0
		t.cachedModTime = 0
		t.cachedSize = 0
		return 0, 0
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(data, &items); err != nil {
		t.cachedPassing = 0
		t.cachedTotal = 0
		t.cachedModTime = 0
		t.cachedSize = 0
		return 0, 0
	}

	passing := 0
	for _, item := range items {
		if val, ok := item[t.passingField]; ok {
			if b, ok := val.(bool); ok && b {
				passing++
			}
		}
	}

	// Update cache
	t.cachedPassing = passing
	t.cachedTotal = len(items)
	t.cachedModTime = modTime
	t.cachedSize = fileSize

	return passing, len(items)
}

// IsInitialized returns true if the JSON file exists and contains at least one item.
func (t *JsonChecklistTracker) IsInitialized() bool {
	_, total := t.GetSummary()
	return total > 0
}

// IsComplete returns true when all items are passing and there is at least one item.
func (t *JsonChecklistTracker) IsComplete() bool {
	passing, total := t.GetSummary()
	return passing == total && total > 0
}

// DisplaySummary prints a progress summary to stdout.
func (t *JsonChecklistTracker) DisplaySummary() {
	passing, total := t.GetSummary()
	if total > 0 {
		percentage := float64(passing) / float64(total) * 100
		fmt.Printf("\nProgress: %d/%d tests passing (%.1f%%)\n", passing, total, percentage)
	} else {
		fmt.Printf("\nProgress: %s not yet created\n", filepath.Base(t.filePath))
	}
}
