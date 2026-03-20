package task

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var (
	formatTestTime1 = time.Date(2026, 3, 10, 14, 22, 0, 0, time.UTC)
	formatTestTime2 = time.Date(2026, 3, 10, 13, 0, 0, 0, time.UTC)
)

func TestFormatTaskMarkdown(t *testing.T) {
	list := &TaskList{
		Phases: []Phase{
			{ID: "d4e5f6a7", Name: "Phase 1: Run Loop"},
		},
		Sections: []Section{
			{ID: "b8c9d0e1", PhaseID: "d4e5f6a7", Name: "1.1 Output Formatting"},
		},
	}

	t.Run("all fields present", func(t *testing.T) {
		task := &Task{
			ID:          "a3f7b2c1",
			Subject:     "Implement stream-JSON output parsing",
			Status:      StatusCompleted,
			PhaseID:     "d4e5f6a7",
			SectionID:   "b8c9d0e1",
			Notes:       "Scans stdout line-by-line as newline-delimited JSON. Skips empty lines and parse failures gracefully.",
			LastUpdated: formatTestTime1,
		}
		want := "# Implement stream-JSON output parsing\n\n**Status:** completed\n**Phase:** Phase 1: Run Loop\n**Section:** 1.1 Output Formatting\n**Updated:** 2026-03-10T14:22:00Z\n\nScans stdout line-by-line as newline-delimited JSON. Skips empty lines and parse failures gracefully.\n"
		got := FormatTaskMarkdown(task, list)
		if got != want {
			t.Errorf("got:\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("no phase or section", func(t *testing.T) {
		task := &Task{
			ID:          "c2a9e5b3",
			Subject:     "Add usage examples to README",
			Status:      StatusPending,
			LastUpdated: formatTestTime2,
		}
		want := "# Add usage examples to README\n\n**Status:** pending\n**Updated:** 2026-03-10T13:00:00Z\n\n**Notes:** (none)\n"
		got := FormatTaskMarkdown(task, list)
		if got != want {
			t.Errorf("got:\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("phase and section IDs not in list use hex fallback", func(t *testing.T) {
		task := &Task{
			ID:          "a1b2c3d4",
			Subject:     "Orphan task",
			Status:      StatusPending,
			PhaseID:     "deadbeef",
			SectionID:   "cafebabe",
			LastUpdated: formatTestTime1,
		}
		got := FormatTaskMarkdown(task, list)
		if !strings.Contains(got, "**Phase:** deadbeef") {
			t.Errorf("expected hex phase ID fallback, got:\n%s", got)
		}
		if !strings.Contains(got, "**Section:** cafebabe") {
			t.Errorf("expected hex section ID fallback, got:\n%s", got)
		}
	})

	t.Run("empty notes shows none", func(t *testing.T) {
		task := &Task{
			ID:          "a3f7b2c1",
			Subject:     "Implement stream-JSON output parsing",
			Status:      StatusCompleted,
			PhaseID:     "d4e5f6a7",
			SectionID:   "b8c9d0e1",
			Notes:       "",
			LastUpdated: formatTestTime1,
		}
		got := FormatTaskMarkdown(task, list)
		if !strings.Contains(got, "**Notes:** (none)") {
			t.Errorf("expected '**Notes:** (none)' for empty notes, got:\n%s", got)
		}
	})
}

func TestFormatTaskJSON(t *testing.T) {
	task := &Task{
		ID:          "a3f7b2c1",
		Subject:     "Implement stream-JSON output parsing",
		Status:      StatusCompleted,
		PhaseID:     "d4e5f6a7",
		SectionID:   "b8c9d0e1",
		Notes:       "Scans stdout.",
		LastUpdated: formatTestTime1,
	}
	got, err := FormatTaskJSON(task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must not be wrapped in a {"tasks": ...} envelope
	if strings.Contains(got, `"tasks"`) {
		t.Errorf("expected direct task JSON, not envelope: %s", got)
	}

	// Valid JSON that round-trips to the task
	var decoded Task
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, got)
	}
	if decoded.ID != task.ID {
		t.Errorf("ID: got %q, want %q", decoded.ID, task.ID)
	}
	if decoded.Subject != task.Subject {
		t.Errorf("Subject: got %q, want %q", decoded.Subject, task.Subject)
	}
	if decoded.Status != task.Status {
		t.Errorf("Status: got %q, want %q", decoded.Status, task.Status)
	}
}

func TestFormatListJSON(t *testing.T) {
	tasks := []Task{
		{ID: "a3f7b2c1", Subject: "Task A", Status: StatusCompleted, LastUpdated: formatTestTime1},
		{ID: "b8e4d1f0", Subject: "Task B", Status: StatusPending, LastUpdated: formatTestTime2},
	}
	got, err := FormatListJSON(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must be wrapped in {"tasks": [...]}
	var envelope struct {
		Tasks []Task `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(got), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, got)
	}
	if len(envelope.Tasks) != 2 {
		t.Errorf("expected 2 tasks in envelope, got %d", len(envelope.Tasks))
	}
	if envelope.Tasks[0].ID != "a3f7b2c1" {
		t.Errorf("first task ID: got %q, want %q", envelope.Tasks[0].ID, "a3f7b2c1")
	}
	if envelope.Tasks[1].ID != "b8e4d1f0" {
		t.Errorf("second task ID: got %q, want %q", envelope.Tasks[1].ID, "b8e4d1f0")
	}
}

func TestFormatListMarkdown(t *testing.T) {
	list := &TaskList{
		Name:        "My Project",
		Description: "Project description.",
		Phases: []Phase{
			{ID: "d4e5f6a7", Name: "Phase 1: Run Loop"},
		},
		Sections: []Section{
			{ID: "b8c9d0e1", PhaseID: "d4e5f6a7", Name: "1.1 Output Formatting"},
			{ID: "f2a3b4c5", PhaseID: "d4e5f6a7", Name: "1.2 Signal Handling"},
		},
	}

	tasks := []Task{
		{
			ID: "d6e7f8a9", Subject: "Add tool input truncation", Status: StatusPending,
			PhaseID: "d4e5f6a7", SectionID: "b8c9d0e1",
			Notes:       "Tool inputs currently show full content.",
			LastUpdated: formatTestTime1,
		},
		{
			ID: "c1d2e3f4", Subject: "Implement two-signal graceful shutdown", Status: StatusPending,
			PhaseID: "d4e5f6a7", SectionID: "f2a3b4c5",
			LastUpdated: formatTestTime2,
		},
	}

	t.Run("grouped by phase and section", func(t *testing.T) {
		got := FormatListMarkdown(tasks, list, false)

		// No project H1 in list output
		if strings.Contains(got, "# My Project") {
			t.Errorf("list should not contain project H1, got:\n%s", got)
		}
		// Phase heading present
		if !strings.Contains(got, "## Phase 1: Run Loop") {
			t.Errorf("expected phase heading, got:\n%s", got)
		}
		// Section headings present
		if !strings.Contains(got, "### 1.1 Output Formatting") {
			t.Errorf("expected section 1.1 heading, got:\n%s", got)
		}
		if !strings.Contains(got, "### 1.2 Signal Handling") {
			t.Errorf("expected section 1.2 heading, got:\n%s", got)
		}
		// Task bullets present
		if !strings.Contains(got, "- `d6e7f8a9` [pending] Add tool input truncation") {
			t.Errorf("expected task bullet for d6e7f8a9, got:\n%s", got)
		}
		if !strings.Contains(got, "- `c1d2e3f4` [pending] Implement two-signal graceful shutdown") {
			t.Errorf("expected task bullet for c1d2e3f4, got:\n%s", got)
		}
		// Notes block with 2-space indented fence
		if !strings.Contains(got, "  ```notes") {
			t.Errorf("expected indented notes opening fence, got:\n%s", got)
		}
		if !strings.Contains(got, "  Tool inputs currently show full content.") {
			t.Errorf("expected indented notes content, got:\n%s", got)
		}
	})

	t.Run("sections with zero matching tasks omitted", func(t *testing.T) {
		// Only tasks in section b8c9d0e1 (1.1 Output Formatting)
		got := FormatListMarkdown(tasks[:1], list, false)
		if !strings.Contains(got, "### 1.1 Output Formatting") {
			t.Errorf("expected 1.1 section heading, got:\n%s", got)
		}
		// 1.2 should be omitted since no tasks match
		if strings.Contains(got, "### 1.2 Signal Handling") {
			t.Errorf("1.2 section should be omitted (no tasks), got:\n%s", got)
		}
	})

	t.Run("phase name hex fallback", func(t *testing.T) {
		listNoName := &TaskList{
			Phases:   []Phase{{ID: "a1b2c3d4"}}, // no name
			Sections: []Section{},
		}
		noNameTasks := []Task{
			{ID: "e5f6a7b8", Subject: "Task in unnamed phase", Status: StatusPending, PhaseID: "a1b2c3d4", LastUpdated: formatTestTime1},
		}
		got := FormatListMarkdown(noNameTasks, listNoName, false)
		if !strings.Contains(got, "## a1b2c3d4") {
			t.Errorf("expected hex phase ID heading as fallback, got:\n%s", got)
		}
	})

	t.Run("section name hex fallback", func(t *testing.T) {
		listNoName := &TaskList{
			Phases:   []Phase{{ID: "d4e5f6a7", Name: "Phase 1"}},
			Sections: []Section{{ID: "a1b2c3d4", PhaseID: "d4e5f6a7"}}, // no name
		}
		noNameTasks := []Task{
			{ID: "e5f6a7b8", Subject: "Task in unnamed section", Status: StatusPending, PhaseID: "d4e5f6a7", SectionID: "a1b2c3d4", LastUpdated: formatTestTime1},
		}
		got := FormatListMarkdown(noNameTasks, listNoName, false)
		if !strings.Contains(got, "### a1b2c3d4") {
			t.Errorf("expected hex section ID heading as fallback, got:\n%s", got)
		}
	})

	t.Run("tasks with no phase collected under (none)", func(t *testing.T) {
		noPhaseTask := Task{ID: "aaaabbbb", Subject: "Orphan task", Status: StatusPending, LastUpdated: formatTestTime1}
		got := FormatListMarkdown([]Task{noPhaseTask}, list, false)
		if !strings.Contains(got, "## (none)") {
			t.Errorf("expected '## (none)' heading for tasks without phase, got:\n%s", got)
		}
		if !strings.Contains(got, "- `aaaabbbb` [pending] Orphan task") {
			t.Errorf("expected task bullet under (none), got:\n%s", got)
		}
	})

	t.Run("tasks without section appear directly under phase heading", func(t *testing.T) {
		noSectionTask := Task{
			ID: "e5f6a7b8", Subject: "Phase-only task", Status: StatusPending,
			PhaseID:     "d4e5f6a7",
			LastUpdated: formatTestTime1,
		}
		got := FormatListMarkdown([]Task{noSectionTask}, list, false)
		if !strings.Contains(got, "## Phase 1: Run Loop") {
			t.Errorf("expected phase heading, got:\n%s", got)
		}
		// No section heading (no H3)
		if strings.Contains(got, "### ") {
			t.Errorf("no section heading expected for task with no section, got:\n%s", got)
		}
		if !strings.Contains(got, "- `e5f6a7b8` [pending] Phase-only task") {
			t.Errorf("expected task bullet, got:\n%s", got)
		}
	})

	t.Run("tasks without notes render as bare bullets", func(t *testing.T) {
		noNotesTasks := []Task{
			{
				ID: "c1d2e3f4", Subject: "No notes task", Status: StatusPending,
				PhaseID: "d4e5f6a7", SectionID: "f2a3b4c5",
				LastUpdated: formatTestTime2,
			},
		}
		got := FormatListMarkdown(noNotesTasks, list, false)
		if !strings.Contains(got, "- `c1d2e3f4` [pending] No notes task") {
			t.Errorf("expected task bullet, got:\n%s", got)
		}
		if strings.Contains(got, "```notes") {
			t.Errorf("expected no notes block for task without notes, got:\n%s", got)
		}
	})

	t.Run("ordering by phase position then section position", func(t *testing.T) {
		orderedList := &TaskList{
			Phases: []Phase{
				{ID: "phase001", Name: "Phase A"},
				{ID: "phase002", Name: "Phase B"},
			},
			Sections: []Section{
				{ID: "sect0001", PhaseID: "phase001", Name: "Section A1"},
				{ID: "sect0002", PhaseID: "phase001", Name: "Section A2"},
			},
		}
		orderedTasks := []Task{
			{ID: "task0003", Subject: "Task in B", Status: StatusPending, PhaseID: "phase002", LastUpdated: formatTestTime1},
			{ID: "task0002", Subject: "Task in A2", Status: StatusPending, PhaseID: "phase001", SectionID: "sect0002", LastUpdated: formatTestTime1},
			{ID: "task0001", Subject: "Task in A1", Status: StatusPending, PhaseID: "phase001", SectionID: "sect0001", LastUpdated: formatTestTime1},
		}
		got := FormatListMarkdown(orderedTasks, orderedList, false)
		posPhaseA := strings.Index(got, "## Phase A")
		posPhaseB := strings.Index(got, "## Phase B")
		posSectA1 := strings.Index(got, "### Section A1")
		posSectA2 := strings.Index(got, "### Section A2")
		if posPhaseA < 0 || posPhaseB < 0 {
			t.Fatalf("expected Phase A and Phase B headings, got:\n%s", got)
		}
		if posPhaseA >= posPhaseB {
			t.Errorf("Phase A should appear before Phase B")
		}
		if posSectA1 < 0 || posSectA2 < 0 {
			t.Fatalf("expected Section A1 and Section A2 headings, got:\n%s", got)
		}
		if posSectA1 >= posSectA2 {
			t.Errorf("Section A1 should appear before Section A2")
		}
	})
}

func TestFormatListMarkdownFlat(t *testing.T) {
	list := &TaskList{
		Phases:   []Phase{{ID: "d4e5f6a7", Name: "Phase 1: Run Loop"}},
		Sections: []Section{{ID: "b8c9d0e1", PhaseID: "d4e5f6a7", Name: "1.1 Output Formatting"}},
	}
	tasks := []Task{
		{
			ID: "d6e7f8a9", Subject: "Add tool input truncation", Status: StatusPending,
			PhaseID: "d4e5f6a7", SectionID: "b8c9d0e1",
			Notes:       "Some notes.",
			LastUpdated: formatTestTime1,
		},
		{
			ID: "c1d2e3f4", Subject: "Implement two-signal graceful shutdown", Status: StatusPending,
			PhaseID: "d4e5f6a7", SectionID: "b8c9d0e1",
			LastUpdated: formatTestTime2,
		},
	}

	got := FormatListMarkdown(tasks, list, true)

	// No phase or section headings
	if strings.Contains(got, "## ") || strings.Contains(got, "### ") {
		t.Errorf("flat mode should not contain headings, got:\n%s", got)
	}
	// No notes blocks
	if strings.Contains(got, "```notes") {
		t.Errorf("flat mode should not contain notes, got:\n%s", got)
	}
	// Task bullets present
	if !strings.Contains(got, "- `d6e7f8a9` [pending] Add tool input truncation") {
		t.Errorf("expected task bullet for d6e7f8a9, got:\n%s", got)
	}
	if !strings.Contains(got, "- `c1d2e3f4` [pending] Implement two-signal graceful shutdown") {
		t.Errorf("expected task bullet for c1d2e3f4, got:\n%s", got)
	}
}

func TestFormatExportMarkdown(t *testing.T) {
	list := &TaskList{
		Name:        "Lorah Development Plan",
		Description: "Track progress on the Lorah infinite-loop harness implementation.",
		Phases: []Phase{
			{ID: "d4e5f6a7", Name: "Phase 1: Run Loop", Description: "Implement the infinite loop."},
		},
		Sections: []Section{
			{ID: "b8c9d0e1", PhaseID: "d4e5f6a7", Name: "1.1 Output Formatting"},
		},
	}
	tasks := []Task{
		{
			ID: "a3f7b2c1", Subject: "Implement stream-JSON output parsing", Status: StatusCompleted,
			PhaseID: "d4e5f6a7", SectionID: "b8c9d0e1",
			Notes:       "Scans stdout line-by-line.",
			LastUpdated: formatTestTime1,
		},
	}

	t.Run("project name H1 and description present", func(t *testing.T) {
		got := FormatExportMarkdown(tasks, list)
		if !strings.Contains(got, "# Lorah Development Plan") {
			t.Errorf("expected project H1, got:\n%s", got)
		}
		if !strings.Contains(got, "Track progress on the Lorah infinite-loop harness implementation.") {
			t.Errorf("expected project description, got:\n%s", got)
		}
	})

	t.Run("phase heading and description present", func(t *testing.T) {
		got := FormatExportMarkdown(tasks, list)
		if !strings.Contains(got, "## Phase 1: Run Loop") {
			t.Errorf("expected phase heading, got:\n%s", got)
		}
		if !strings.Contains(got, "Implement the infinite loop.") {
			t.Errorf("expected phase description, got:\n%s", got)
		}
	})

	t.Run("section heading and task bullet with notes", func(t *testing.T) {
		got := FormatExportMarkdown(tasks, list)
		if !strings.Contains(got, "### 1.1 Output Formatting") {
			t.Errorf("expected section heading, got:\n%s", got)
		}
		if !strings.Contains(got, "- `a3f7b2c1` [completed] Implement stream-JSON output parsing") {
			t.Errorf("expected task bullet, got:\n%s", got)
		}
		if !strings.Contains(got, "  ```notes") {
			t.Errorf("expected indented notes fence, got:\n%s", got)
		}
		if !strings.Contains(got, "  Scans stdout line-by-line.") {
			t.Errorf("expected indented notes content, got:\n%s", got)
		}
	})

	t.Run("no name skips H1 and description", func(t *testing.T) {
		listNoName := &TaskList{
			Description: "Some description.",
			Phases:      list.Phases,
			Sections:    list.Sections,
		}
		got := FormatExportMarkdown(tasks, listNoName)
		if strings.HasPrefix(got, "# ") {
			t.Errorf("expected no H1 when name is empty, got:\n%s", got)
		}
		// Description skipped when name is absent
		if strings.Contains(got, "Some description.") {
			t.Errorf("description should be skipped when name is absent, got:\n%s", got)
		}
	})

	t.Run("tasks with no phase collected under (none)", func(t *testing.T) {
		noPhaseTask := Task{
			ID: "aaaabbbb", Subject: "Orphan task", Status: StatusPending,
			LastUpdated: formatTestTime1,
		}
		got := FormatExportMarkdown([]Task{noPhaseTask}, list)
		if !strings.Contains(got, "## (none)") {
			t.Errorf("expected '## (none)' heading for tasks without phase, got:\n%s", got)
		}
	})

	t.Run("section description renders below section heading", func(t *testing.T) {
		listWithSectionDesc := &TaskList{
			Name: "My Project",
			Phases: []Phase{
				{ID: "d4e5f6a7", Name: "Phase 1: Run Loop"},
			},
			Sections: []Section{
				{ID: "b8c9d0e1", PhaseID: "d4e5f6a7", Name: "1.1 Output Formatting", Description: "Handles stream-JSON parsing."},
			},
		}
		got := FormatExportMarkdown(tasks, listWithSectionDesc)
		if !strings.Contains(got, "### 1.1 Output Formatting") {
			t.Errorf("expected section heading, got:\n%s", got)
		}
		if !strings.Contains(got, "Handles stream-JSON parsing.") {
			t.Errorf("expected section description in export output, got:\n%s", got)
		}
	})

	t.Run("no description paragraph when description is empty", func(t *testing.T) {
		listNoDesc := &TaskList{
			Name: "My Project",
			Phases: []Phase{
				{ID: "d4e5f6a7", Name: "Phase 1: Run Loop"}, // no description
			},
			Sections: []Section{
				{ID: "b8c9d0e1", PhaseID: "d4e5f6a7", Name: "1.1 Output Formatting"}, // no description
			},
		}
		got := FormatExportMarkdown(tasks, listNoDesc)
		// Phase heading must be present
		if !strings.Contains(got, "## Phase 1: Run Loop") {
			t.Errorf("expected phase heading, got:\n%s", got)
		}
		// No "(none)" placeholder for missing descriptions
		if strings.Contains(got, "(none)") {
			t.Errorf("expected no placeholder text for empty descriptions, got:\n%s", got)
		}
		// No consecutive blank lines beyond heading separators (no extra blank line from missing description)
		if strings.Contains(got, "\n\n\n") {
			t.Errorf("unexpected triple newline (extra blank line) from empty description, got:\n%q", got)
		}
	})
}
