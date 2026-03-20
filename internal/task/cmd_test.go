package task

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

// mockStorage implements Storage for test isolation.
type mockStorage struct {
	list    *TaskList
	tasks   map[string]*Task
	loadErr error
	saveErr error
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		list:  &TaskList{Version: "1.0"},
		tasks: make(map[string]*Task),
	}
}

func (m *mockStorage) Load() (*TaskList, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return m.list, nil
}

func (m *mockStorage) Save(list *TaskList) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.list = list
	return nil
}

func (m *mockStorage) Get(id string) (*Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, errors.New("task not found: " + id)
	}
	return t, nil
}

func (m *mockStorage) List(filter Filter) ([]Task, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	var result []Task
	for _, t := range m.list.Tasks {
		if len(filter.Status) > 0 {
			match := false
			for _, s := range filter.Status {
				if t.Status == s {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if filter.PhaseID != "" && t.PhaseID != filter.PhaseID {
			continue
		}
		if filter.SectionID != "" && t.SectionID != filter.SectionID {
			continue
		}
		result = append(result, t)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}
	return result, nil
}

func (m *mockStorage) Create(task *Task) error {
	if _, ok := m.tasks[task.ID]; ok {
		return errors.New("duplicate id: " + task.ID)
	}
	m.tasks[task.ID] = task
	m.list.Tasks = append(m.list.Tasks, *task)
	return nil
}

func (m *mockStorage) Update(task *Task) error {
	if _, ok := m.tasks[task.ID]; !ok {
		return errors.New("task not found: " + task.ID)
	}
	m.tasks[task.ID] = task
	for i, t := range m.list.Tasks {
		if t.ID == task.ID {
			m.list.Tasks[i] = *task
			break
		}
	}
	return nil
}

func (m *mockStorage) Delete(id string) error {
	if _, ok := m.tasks[id]; !ok {
		return errors.New("task not found: " + id)
	}
	delete(m.tasks, id)
	for i, t := range m.list.Tasks {
		if t.ID == id {
			m.list.Tasks = append(m.list.Tasks[:i], m.list.Tasks[i+1:]...)
			break
		}
	}
	return nil
}

// --- multiFlag tests ---

func TestMultiFlagString(t *testing.T) {
	var f multiFlag
	if f.String() != "" {
		t.Errorf("expected empty string, got %q", f.String())
	}
	f = multiFlag{"a", "b"}
	got := f.String()
	if got != "a,b" {
		t.Errorf("expected %q, got %q", "a,b", got)
	}
}

func TestMultiFlagSet(t *testing.T) {
	var f multiFlag
	if err := f.Set("foo"); err != nil {
		t.Fatal(err)
	}
	if err := f.Set("bar"); err != nil {
		t.Fatal(err)
	}
	if len(f) != 2 || f[0] != "foo" || f[1] != "bar" {
		t.Errorf("unexpected values: %v", f)
	}
}

// --- HandleTask dispatch tests ---

func TestHandleTaskHelp(t *testing.T) {
	for _, arg := range []string{"--help", "-h"} {
		var buf bytes.Buffer
		code := HandleTask([]string{arg}, &buf, newMockStorage())
		if code != 0 {
			t.Errorf("HandleTask(%q): expected exit 0, got %d", arg, code)
		}
	}
}

func TestHandleTaskUnknownSubcommand(t *testing.T) {
	var buf bytes.Buffer
	code := HandleTask([]string{"bogus"}, &buf, newMockStorage())
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestHandleTaskNoArgs(t *testing.T) {
	var buf bytes.Buffer
	code := HandleTask([]string{}, &buf, newMockStorage())
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

// --- list tests ---

func TestListBasic(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	task := &Task{ID: "aabbccdd", Subject: "Do something", Status: StatusPending, LastUpdated: now}
	store.list.Tasks = []Task{*task}
	store.tasks["aabbccdd"] = task

	var buf bytes.Buffer
	code := HandleTask([]string{"list"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "aabbccdd") {
		t.Errorf("expected task ID in output, got:\n%s", out)
	}
}

func TestListStatusFilter(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	store.list.Tasks = []Task{
		{ID: "aaaaaa01", Subject: "Pending task", Status: StatusPending, LastUpdated: now},
		{ID: "aaaaaa02", Subject: "Completed task", Status: StatusCompleted, LastUpdated: now},
	}

	var buf bytes.Buffer
	code := HandleTask([]string{"list", "--status=pending"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "aaaaaa01") {
		t.Errorf("expected pending task in output")
	}
	if strings.Contains(out, "aaaaaa02") {
		t.Errorf("expected completed task to be filtered out")
	}
}

func TestListMultipleStatusFlags(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	store.list.Tasks = []Task{
		{ID: "aaaaaa01", Subject: "Pending", Status: StatusPending, LastUpdated: now},
		{ID: "aaaaaa02", Subject: "In progress", Status: StatusInProgress, LastUpdated: now},
		{ID: "aaaaaa03", Subject: "Completed", Status: StatusCompleted, LastUpdated: now},
	}

	var buf bytes.Buffer
	code := HandleTask([]string{"list", "--status=pending", "--status=in_progress"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "aaaaaa01") || !strings.Contains(out, "aaaaaa02") {
		t.Errorf("expected pending and in_progress tasks in output")
	}
	if strings.Contains(out, "aaaaaa03") {
		t.Errorf("expected completed task to be filtered out")
	}
}

func TestListFlatFormat(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	store.list.Tasks = []Task{
		{ID: "aaaaaa01", Subject: "Task A", Status: StatusPending, LastUpdated: now, Notes: "some notes"},
	}

	var buf bytes.Buffer
	code := HandleTask([]string{"list", "--flat"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if strings.Contains(out, "notes") {
		t.Errorf("expected notes to be suppressed in flat mode, got:\n%s", out)
	}
	if strings.Contains(out, "##") {
		t.Errorf("expected no headings in flat mode, got:\n%s", out)
	}
}

func TestListJSONFormat(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	store.list.Tasks = []Task{
		{ID: "aaaaaa01", Subject: "Task A", Status: StatusPending, LastUpdated: now},
	}

	var buf bytes.Buffer
	code := HandleTask([]string{"list", "--format=json"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, `"tasks"`) {
		t.Errorf("expected JSON envelope with tasks key, got:\n%s", out)
	}
}

func TestListPhaseFilter(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	store.list.Tasks = []Task{
		{ID: "aaaaaa01", Subject: "Phase1 task", Status: StatusPending, PhaseID: "phase111", LastUpdated: now},
		{ID: "aaaaaa02", Subject: "Phase2 task", Status: StatusPending, PhaseID: "phase222", LastUpdated: now},
	}

	var buf bytes.Buffer
	code := HandleTask([]string{"list", "--phase=phase111"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "aaaaaa01") {
		t.Errorf("expected phase1 task in output")
	}
	if strings.Contains(out, "aaaaaa02") {
		t.Errorf("expected phase2 task to be filtered out")
	}
}

func TestListLimit(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	store.list.Tasks = []Task{
		{ID: "aaaaaa01", Subject: "Task 1", Status: StatusPending, LastUpdated: now},
		{ID: "aaaaaa02", Subject: "Task 2", Status: StatusPending, LastUpdated: now},
		{ID: "aaaaaa03", Subject: "Task 3", Status: StatusPending, LastUpdated: now},
	}

	var buf bytes.Buffer
	code := HandleTask([]string{"list", "--limit=2"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	count := strings.Count(out, "aaaaaa")
	if count != 2 {
		t.Errorf("expected 2 tasks with limit=2, got %d in:\n%s", count, out)
	}
}

func TestListSectionFilter(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	store.list.Tasks = []Task{
		{ID: "aaaaaa01", Subject: "Section1 task", Status: StatusPending, SectionID: "sect1111", LastUpdated: now},
		{ID: "aaaaaa02", Subject: "Section2 task", Status: StatusPending, SectionID: "sect2222", LastUpdated: now},
	}

	var buf bytes.Buffer
	code := HandleTask([]string{"list", "--section=sect1111"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "aaaaaa01") {
		t.Errorf("expected section1 task in output")
	}
	if strings.Contains(out, "aaaaaa02") {
		t.Errorf("expected section2 task to be filtered out")
	}
}

func TestListFlatWithJSON(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	store.list.Tasks = []Task{
		{ID: "aaaaaa01", Subject: "Task A", Status: StatusPending, LastUpdated: now, Notes: "some notes"},
	}

	var buf bytes.Buffer
	code := HandleTask([]string{"list", "--flat", "--format=json"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	// --flat is ignored when --format=json; output should be JSON envelope
	if !strings.Contains(out, `"tasks"`) {
		t.Errorf("expected JSON envelope with tasks key even with --flat, got:\n%s", out)
	}
}

func TestListInvalidStatus(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"list", "--status=bogus"}, &buf, store)
	if code != 1 {
		t.Errorf("expected exit 1 for invalid status, got %d", code)
	}
}

// --- get tests ---

func TestGetByID(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	task := &Task{ID: "deadbeef", Subject: "Test get", Status: StatusPending, LastUpdated: now}
	store.tasks["deadbeef"] = task
	store.list = &TaskList{Version: "1.0", Tasks: []Task{*task}}

	var buf bytes.Buffer
	code := HandleTask([]string{"get", "deadbeef"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	// Default format is markdown; subject should appear as H1.
	if !strings.Contains(out, "# Test get") {
		t.Errorf("expected subject as H1 in markdown output, got:\n%s", out)
	}
}

func TestGetCmdNotFound(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"get", "notexist"}, &buf, store)
	if code != 1 {
		t.Errorf("expected exit 1 for missing task, got %d", code)
	}
}

func TestGetMissingID(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"get"}, &buf, store)
	if code != 1 {
		t.Errorf("expected exit 1 when ID not provided, got %d", code)
	}
}

func TestGetJSONFormat(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	task := &Task{ID: "deadbeef", Subject: "Test get JSON", Status: StatusPending, LastUpdated: now}
	store.tasks["deadbeef"] = task
	store.list = &TaskList{Version: "1.0", Tasks: []Task{*task}}

	var buf bytes.Buffer
	code := HandleTask([]string{"get", "deadbeef", "--format=json"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, `"id"`) || !strings.Contains(out, "deadbeef") {
		t.Errorf("expected JSON with task fields, got:\n%s", out)
	}
	// Should NOT be wrapped in {"tasks": [...]}
	if strings.Contains(out, `"tasks"`) {
		t.Errorf("get JSON should not be wrapped in tasks envelope, got:\n%s", out)
	}
}

// --- create tests ---

func TestCreateRequiresSubject(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"create"}, &buf, store)
	if code != 1 {
		t.Errorf("expected exit 1 when --subject missing, got %d", code)
	}
}

func TestCreateBasic(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"create", "--subject=New task"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "task ") {
		t.Errorf("expected 'task <id>' in output, got:\n%s", out)
	}
	// Verify task was stored
	if len(store.list.Tasks) != 1 {
		t.Fatalf("expected 1 task created, got %d", len(store.list.Tasks))
	}
	if store.list.Tasks[0].Subject != "New task" {
		t.Errorf("expected subject 'New task', got %q", store.list.Tasks[0].Subject)
	}
	if store.list.Tasks[0].Status != StatusPending {
		t.Errorf("expected default status pending, got %q", store.list.Tasks[0].Status)
	}
}

func TestCreateWithPhaseName(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"create", "--subject=Task", "--phase-name=My Phase"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	// Should output phase <id> and task <id>
	if !strings.Contains(out, "phase ") {
		t.Errorf("expected 'phase <id>' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "task ") {
		t.Errorf("expected 'task <id>' in output, got:\n%s", out)
	}
	// Verify phase was created
	if len(store.list.Phases) != 1 {
		t.Fatalf("expected 1 phase created, got %d", len(store.list.Phases))
	}
	if store.list.Phases[0].Name != "My Phase" {
		t.Errorf("expected phase name 'My Phase', got %q", store.list.Phases[0].Name)
	}
}

func TestCreateWithPhaseAndSectionName(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"create", "--subject=Task", "--phase-name=Phase 1", "--section-name=Section 1.1"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "phase ") {
		t.Errorf("expected 'phase <id>' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "section ") {
		t.Errorf("expected 'section <id>' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "task ") {
		t.Errorf("expected 'task <id>' in output, got:\n%s", out)
	}
}

func TestCreateWithExistingPhase(t *testing.T) {
	store := newMockStorage()
	store.list.Phases = []Phase{{ID: "phase001", Name: "Existing Phase"}}

	var buf bytes.Buffer
	code := HandleTask([]string{"create", "--subject=Task", "--phase=phase001"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	// Should NOT output phase <id> since we didn't create a new one
	if strings.Contains(out, "phase ") {
		t.Errorf("expected no 'phase <id>' when using existing phase, got:\n%s", out)
	}
	if !strings.Contains(out, "task ") {
		t.Errorf("expected 'task <id>' in output, got:\n%s", out)
	}
	if len(store.list.Tasks) > 0 && store.list.Tasks[0].PhaseID != "phase001" {
		t.Errorf("expected task phaseID 'phase001', got %q", store.list.Tasks[0].PhaseID)
	}
}

func TestCreateInvalidStatus(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"create", "--subject=Task", "--status=invalid"}, &buf, store)
	if code != 1 {
		t.Errorf("expected exit 1 for invalid status, got %d", code)
	}
}

func TestCreateWithProjectNameAndDescription(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"create", "--subject=Task", "--project-name=My Project", "--project-description=A great project"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if store.list.Name != "My Project" {
		t.Errorf("expected project name 'My Project', got %q", store.list.Name)
	}
	if store.list.Description != "A great project" {
		t.Errorf("expected project description 'A great project', got %q", store.list.Description)
	}
}

func TestCreateWithStatus(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"create", "--subject=Task", "--status=in_progress"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if len(store.list.Tasks) != 1 {
		t.Fatalf("expected 1 task created, got %d", len(store.list.Tasks))
	}
	if store.list.Tasks[0].Status != StatusInProgress {
		t.Errorf("expected status in_progress, got %q", store.list.Tasks[0].Status)
	}
}

func TestCreateWithExistingSection(t *testing.T) {
	store := newMockStorage()
	store.list.Phases = []Phase{{ID: "phase001", Name: "Phase 1"}}
	store.list.Sections = []Section{{ID: "sect0001", PhaseID: "phase001", Name: "Section 1.1"}}

	var buf bytes.Buffer
	code := HandleTask([]string{"create", "--subject=Task", "--phase=phase001", "--section=sect0001"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	// Should NOT output "section <id>" since we used an existing section
	if strings.Contains(out, "section ") {
		t.Errorf("expected no 'section <id>' when using existing section, got:\n%s", out)
	}
	if !strings.Contains(out, "task ") {
		t.Errorf("expected 'task <id>' in output, got:\n%s", out)
	}
	if len(store.list.Tasks) != 1 || store.list.Tasks[0].SectionID != "sect0001" {
		t.Errorf("expected task sectionID 'sect0001', tasks: %+v", store.list.Tasks)
	}
	// No new section should have been created
	if len(store.list.Sections) != 1 {
		t.Errorf("expected existing section count to remain 1, got %d", len(store.list.Sections))
	}
}

func TestCreatePhaseDescription(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"create", "--subject=Task", "--phase-name=My Phase", "--phase-description=Phase desc"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if len(store.list.Phases) != 1 {
		t.Fatalf("expected 1 phase created, got %d", len(store.list.Phases))
	}
	if store.list.Phases[0].Description != "Phase desc" {
		t.Errorf("expected phase description 'Phase desc', got %q", store.list.Phases[0].Description)
	}
}

func TestCreateSectionDescription(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"create", "--subject=Task", "--phase-name=Phase 1", "--section-name=Section 1.1", "--section-description=Section desc"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if len(store.list.Sections) != 1 {
		t.Fatalf("expected 1 section created, got %d", len(store.list.Sections))
	}
	if store.list.Sections[0].Description != "Section desc" {
		t.Errorf("expected section description 'Section desc', got %q", store.list.Sections[0].Description)
	}
}

func TestCreateSectionPhaseIDAssigned(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"create", "--subject=Task", "--phase-name=Phase 1", "--section-name=Section 1.1"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if len(store.list.Phases) != 1 {
		t.Fatalf("expected 1 phase, got %d", len(store.list.Phases))
	}
	if len(store.list.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(store.list.Sections))
	}
	// Section must reference the auto-generated phase's ID.
	if store.list.Sections[0].PhaseID != store.list.Phases[0].ID {
		t.Errorf("expected section PhaseID %q, got %q", store.list.Phases[0].ID, store.list.Sections[0].PhaseID)
	}
}

func TestCreateOutputOrdering(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"create", "--subject=Task", "--phase-name=Phase 1", "--section-name=Section 1.1"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 output lines (phase, section, task), got %d: %q", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "phase ") {
		t.Errorf("expected first line to start with 'phase ', got %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "section ") {
		t.Errorf("expected second line to start with 'section ', got %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "task ") {
		t.Errorf("expected third line to start with 'task ', got %q", lines[2])
	}
}

func TestCreateSectionNameWithoutPhaseContext(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	// --section-name without --phase or --phase-name should return 1
	code := HandleTask([]string{"create", "--subject=Task", "--section-name=Section 1.1"}, &buf, store)
	if code != 1 {
		t.Errorf("expected exit 1 when --section-name used without phase context, got %d", code)
	}
}

func TestCreateSectionWithoutPhaseContext(t *testing.T) {
	store := newMockStorage()
	store.list.Sections = []Section{{ID: "sect0001", PhaseID: "phase001", Name: "Section 1.1"}}

	var buf bytes.Buffer
	// --section without --phase or --phase-name should return 1
	code := HandleTask([]string{"create", "--subject=Task", "--section=sect0001"}, &buf, store)
	if code != 1 {
		t.Errorf("expected exit 1 when --section used without phase context, got %d", code)
	}
}

// --- update tests ---

func TestUpdateStatus(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	task := &Task{ID: "upd00001", Subject: "Update me", Status: StatusPending, LastUpdated: now}
	store.tasks["upd00001"] = task
	store.list.Tasks = []Task{*task}

	var buf bytes.Buffer
	code := HandleTask([]string{"update", "upd00001", "--status=in_progress"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if store.tasks["upd00001"].Status != StatusInProgress {
		t.Errorf("expected status in_progress, got %q", store.tasks["upd00001"].Status)
	}
}

func TestUpdateNotes(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	task := &Task{ID: "upd00002", Subject: "Notes task", Status: StatusPending, LastUpdated: now}
	store.tasks["upd00002"] = task
	store.list.Tasks = []Task{*task}

	var buf bytes.Buffer
	code := HandleTask([]string{"update", "upd00002", "--notes=Some notes here"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if store.tasks["upd00002"].Notes != "Some notes here" {
		t.Errorf("expected notes 'Some notes here', got %q", store.tasks["upd00002"].Notes)
	}
}

func TestUpdateCmdNotFound(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"update", "notexist", "--status=completed"}, &buf, store)
	if code != 1 {
		t.Errorf("expected exit 1 for missing task, got %d", code)
	}
}

func TestUpdateMissingID(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"update"}, &buf, store)
	if code != 1 {
		t.Errorf("expected exit 1 when ID not provided, got %d", code)
	}
}

func TestUpdatePartialPreservesFields(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	task := &Task{ID: "upd00003", Subject: "Keep me", Status: StatusPending, Notes: "keep notes", LastUpdated: now}
	store.tasks["upd00003"] = task
	store.list.Tasks = []Task{*task}

	var buf bytes.Buffer
	code := HandleTask([]string{"update", "upd00003", "--status=completed"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	updated := store.tasks["upd00003"]
	if updated.Subject != "Keep me" {
		t.Errorf("expected subject to be preserved, got %q", updated.Subject)
	}
	if updated.Notes != "keep notes" {
		t.Errorf("expected notes to be preserved, got %q", updated.Notes)
	}
	if updated.Status != StatusCompleted {
		t.Errorf("expected status completed, got %q", updated.Status)
	}
}

func TestUpdateInvalidStatus(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	task := &Task{ID: "upd00004", Subject: "Task", Status: StatusPending, LastUpdated: now}
	store.tasks["upd00004"] = task
	store.list.Tasks = []Task{*task}

	var buf bytes.Buffer
	code := HandleTask([]string{"update", "upd00004", "--status=bad"}, &buf, store)
	if code != 1 {
		t.Errorf("expected exit 1 for invalid status, got %d", code)
	}
}

func TestUpdateSetsLastUpdated(t *testing.T) {
	store := newMockStorage()
	past := time.Now().Add(-1 * time.Hour)
	task := &Task{ID: "upd00005", Subject: "Time task", Status: StatusPending, LastUpdated: past}
	store.tasks["upd00005"] = task
	store.list.Tasks = []Task{*task}

	var buf bytes.Buffer
	code := HandleTask([]string{"update", "upd00005", "--status=completed"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	updated := store.tasks["upd00005"]
	if !updated.LastUpdated.After(past) {
		t.Errorf("expected LastUpdated to be updated, old=%v new=%v", past, updated.LastUpdated)
	}
}

func TestUpdateSubject(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	task := &Task{ID: "upd00006", Subject: "Old subject", Status: StatusPending, LastUpdated: now}
	store.tasks["upd00006"] = task
	store.list.Tasks = []Task{*task}

	var buf bytes.Buffer
	code := HandleTask([]string{"update", "upd00006", "--subject=New subject"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if store.tasks["upd00006"].Subject != "New subject" {
		t.Errorf("expected subject 'New subject', got %q", store.tasks["upd00006"].Subject)
	}
}

func TestUpdateSubjectClear(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	task := &Task{ID: "upd00007", Subject: "Has subject", Status: StatusPending, LastUpdated: now}
	store.tasks["upd00007"] = task
	store.list.Tasks = []Task{*task}

	var buf bytes.Buffer
	// --subject="" should clear the subject (not treated as "not provided")
	code := HandleTask([]string{"update", "upd00007", "--subject="}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if store.tasks["upd00007"].Subject != "" {
		t.Errorf("expected subject to be cleared, got %q", store.tasks["upd00007"].Subject)
	}
}

// --- delete tests ---

func TestDeleteByID(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	task := &Task{ID: "del00001", Subject: "Delete me", Status: StatusPending, LastUpdated: now}
	store.tasks["del00001"] = task
	store.list.Tasks = []Task{*task}

	var buf bytes.Buffer
	code := HandleTask([]string{"delete", "del00001"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if buf.Len() > 0 {
		t.Errorf("expected no output on successful delete, got:\n%s", buf.String())
	}
	if _, err := store.Get("del00001"); err == nil {
		t.Errorf("expected task to be deleted, but it still exists")
	}
}

func TestDeleteCmdNotFound(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"delete", "notexist"}, &buf, store)
	if code != 1 {
		t.Errorf("expected exit 1 for missing task, got %d", code)
	}
}

func TestDeleteMissingID(t *testing.T) {
	store := newMockStorage()

	var buf bytes.Buffer
	code := HandleTask([]string{"delete"}, &buf, store)
	if code != 1 {
		t.Errorf("expected exit 1 when ID not provided, got %d", code)
	}
}

// --- export tests ---

func TestExportBasic(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	store.list.Name = "My Project"
	store.list.Tasks = []Task{
		{ID: "exp00001", Subject: "Export task", Status: StatusPending, LastUpdated: now},
	}

	var buf bytes.Buffer
	code := HandleTask([]string{"export"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "My Project") {
		t.Errorf("expected project name in export output, got:\n%s", out)
	}
	if !strings.Contains(out, "exp00001") {
		t.Errorf("expected task ID in export output, got:\n%s", out)
	}
}

func TestExportStatusFilter(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	store.list.Tasks = []Task{
		{ID: "exp00001", Subject: "Pending task", Status: StatusPending, LastUpdated: now},
		{ID: "exp00002", Subject: "Completed task", Status: StatusCompleted, LastUpdated: now},
	}

	var buf bytes.Buffer
	code := HandleTask([]string{"export", "--status=pending"}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "exp00001") {
		t.Errorf("expected pending task in export")
	}
	if strings.Contains(out, "exp00002") {
		t.Errorf("expected completed task to be filtered out of export")
	}
}

func TestExportToFile(t *testing.T) {
	store := newMockStorage()
	now := time.Now()
	store.list.Tasks = []Task{
		{ID: "exp00001", Subject: "File export task", Status: StatusPending, LastUpdated: now},
	}

	outFile := t.TempDir() + "/export.md"
	var buf bytes.Buffer
	code := HandleTask([]string{"export", "--output=" + outFile}, &buf, store)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	// buf should be empty (written to file instead)
	if buf.Len() > 0 {
		t.Errorf("expected nothing written to stdout when --output given, got:\n%s", buf.String())
	}
}
