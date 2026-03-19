package task

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// FormatTaskMarkdown returns a single-task markdown string for the get subcommand.
func FormatTaskMarkdown(task *Task, list *TaskList) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n\n", task.Subject)
	fmt.Fprintf(&sb, "**Status:** %s\n", task.Status)
	if task.PhaseID != "" {
		fmt.Fprintf(&sb, "**Phase:** %s\n", resolvePhaseName(task.PhaseID, list))
	}
	if task.SectionID != "" {
		fmt.Fprintf(&sb, "**Section:** %s\n", resolveSectionName(task.SectionID, list))
	}
	fmt.Fprintf(&sb, "**Updated:** %s\n", task.LastUpdated.UTC().Format("2006-01-02T15:04:05Z"))
	sb.WriteString("\n")
	if task.Notes != "" {
		fmt.Fprintf(&sb, "%s\n", task.Notes)
	} else {
		sb.WriteString("**Notes:** (none)\n")
	}
	return sb.String()
}

// FormatTaskJSON returns a single task as indented JSON (not wrapped in an envelope).
func FormatTaskJSON(task *Task) (string, error) {
	b, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FormatListMarkdown returns tasks grouped by phase/section as markdown.
// When flat is true, headings and notes are suppressed.
func FormatListMarkdown(tasks []Task, list *TaskList, flat bool) string {
	if flat {
		return renderFlatList(tasks)
	}
	return renderGrouped(tasks, list, false)
}

// FormatListJSON returns tasks wrapped in a {"tasks": [...]} JSON envelope.
func FormatListJSON(tasks []Task) (string, error) {
	envelope := struct {
		Tasks []Task `json:"tasks"`
	}{Tasks: tasks}
	b, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FormatExportMarkdown returns tasks as markdown with project name H1 and description.
func FormatExportMarkdown(tasks []Task, list *TaskList) string {
	var sb strings.Builder
	if list.Name != "" {
		fmt.Fprintf(&sb, "# %s\n\n", list.Name)
		if list.Description != "" {
			fmt.Fprintf(&sb, "%s\n\n", list.Description)
		}
	}
	sb.WriteString(renderGrouped(tasks, list, true))
	return sb.String()
}

// resolvePhaseName returns the phase name for the given ID, or the ID itself as fallback.
func resolvePhaseName(phaseID string, list *TaskList) string {
	for _, p := range list.Phases {
		if p.ID == phaseID {
			if p.Name != "" {
				return p.Name
			}
			return p.ID
		}
	}
	return phaseID
}

// resolveSectionName returns the section name for the given ID, or the ID itself as fallback.
func resolveSectionName(sectionID string, list *TaskList) string {
	for _, s := range list.Sections {
		if s.ID == sectionID {
			if s.Name != "" {
				return s.Name
			}
			return s.ID
		}
	}
	return sectionID
}

// taskBullet returns the markdown bullet line for a task.
func taskBullet(t Task) string {
	return fmt.Sprintf("- `%s` [%s] %s", t.ID, t.Status, t.Subject)
}

// notesBlock returns a 2-space-indented fenced notes block.
func notesBlock(notes string) string {
	var sb strings.Builder
	sb.WriteString("\n  ```notes\n")
	for _, line := range strings.Split(notes, "\n") {
		fmt.Fprintf(&sb, "  %s\n", line)
	}
	sb.WriteString("  ```\n")
	return sb.String()
}

// renderFlatList renders tasks as a flat bullet list with no headings or notes.
func renderFlatList(tasks []Task) string {
	var sb strings.Builder
	for _, t := range tasks {
		fmt.Fprintf(&sb, "%s\n", taskBullet(t))
	}
	return sb.String()
}

// renderGrouped renders tasks grouped by phase then section.
// includePhaseDesc controls whether phase descriptions are printed (used by export).
func renderGrouped(tasks []Task, list *TaskList, includePhaseDesc bool) string {
	var sb strings.Builder

	// Build a lookup: phaseID -> []Task, sectionID -> []Task
	byPhase := make(map[string][]Task)
	bySection := make(map[string][]Task)
	for _, t := range tasks {
		if t.PhaseID == "" {
			byPhase[""] = append(byPhase[""], t)
		} else if t.SectionID == "" {
			byPhase[t.PhaseID] = append(byPhase[t.PhaseID], t)
		} else {
			bySection[t.SectionID] = append(bySection[t.SectionID], t)
		}
	}

	// Track which phase IDs are covered by list.Phases.
	knownPhaseIDs := make(map[string]bool)
	for _, p := range list.Phases {
		knownPhaseIDs[p.ID] = true
	}

	// Render phases in TaskList order
	for _, phase := range list.Phases {
		// Determine if this phase has any tasks (direct or via sections)
		hasAny := len(byPhase[phase.ID]) > 0
		if !hasAny {
			for _, sec := range list.Sections {
				if sec.PhaseID == phase.ID && len(bySection[sec.ID]) > 0 {
					hasAny = true
					break
				}
			}
		}
		if !hasAny {
			continue
		}

		fmt.Fprintf(&sb, "## %s\n\n", resolvePhaseName(phase.ID, list))
		if includePhaseDesc && phase.Description != "" {
			fmt.Fprintf(&sb, "%s\n\n", phase.Description)
		}

		// Direct phase tasks (no section)
		for _, t := range byPhase[phase.ID] {
			sb.WriteString(taskBullet(t))
			sb.WriteString("\n")
			if t.Notes != "" {
				sb.WriteString(notesBlock(t.Notes))
			}
		}

		// Sections in TaskList order
		for _, sec := range list.Sections {
			if sec.PhaseID != phase.ID {
				continue
			}
			secTasks := bySection[sec.ID]
			if len(secTasks) == 0 {
				continue
			}
			fmt.Fprintf(&sb, "### %s\n\n", resolveSectionName(sec.ID, list))
			for _, t := range secTasks {
				sb.WriteString(taskBullet(t))
				sb.WriteString("\n")
				if t.Notes != "" {
					sb.WriteString(notesBlock(t.Notes))
				}
			}
		}
	}

	// Render orphan phases: tasks with a phaseID not in list.Phases.
	var orphanIDs []string
	for id := range byPhase {
		if id != "" && !knownPhaseIDs[id] {
			orphanIDs = append(orphanIDs, id)
		}
	}
	sort.Strings(orphanIDs)
	for _, id := range orphanIDs {
		fmt.Fprintf(&sb, "## %s\n\n", id)
		for _, t := range byPhase[id] {
			sb.WriteString(taskBullet(t))
			sb.WriteString("\n")
			if t.Notes != "" {
				sb.WriteString(notesBlock(t.Notes))
			}
		}
	}

	// Tasks with no phase
	if noPhase := byPhase[""]; len(noPhase) > 0 {
		sb.WriteString("## (none)\n\n")
		for _, t := range noPhase {
			sb.WriteString(taskBullet(t))
			sb.WriteString("\n")
			if t.Notes != "" {
				sb.WriteString(notesBlock(t.Notes))
			}
		}
	}

	return sb.String()
}
