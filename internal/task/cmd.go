package task

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// multiFlag is a []string implementing flag.Value for repeatable flags.
type multiFlag []string

func (f *multiFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *multiFlag) Set(s string) error {
	*f = append(*f, s)
	return nil
}

// validStatus returns true if s is a valid TaskStatus.
func validStatus(s string) bool {
	switch TaskStatus(s) {
	case StatusPending, StatusInProgress, StatusCompleted:
		return true
	}
	return false
}

// HandleTask dispatches lorah task subcommands.
// It returns an exit code: 0 for success, 1 for error.
func HandleTask(args []string, w io.Writer, storage Storage) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: lorah task <subcommand> [args...]")
		return 1
	}
	switch args[0] {
	case "--help", "-help", "-h":
		fmt.Fprintln(os.Stderr, "usage: lorah task <subcommand> [args...]")
		return 0
	case "list":
		return listCmd(args[1:], w, storage)
	case "get":
		return getCmd(args[1:], w, storage)
	case "create":
		return createCmd(args[1:], w, storage)
	case "update":
		return updateCmd(args[1:], w, storage)
	case "delete":
		return deleteCmd(args[1:], w, storage)
	case "export":
		return exportCmd(args[1:], w, storage)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", args[0])
		return 1
	}
}

func listCmd(args []string, w io.Writer, storage Storage) int {
	fs := flag.NewFlagSet("lorah task list", flag.ContinueOnError)
	var statuses multiFlag
	fs.Var(&statuses, "status", "filter by status (repeatable)")
	phase := fs.String("phase", "", "filter by phase ID")
	section := fs.String("section", "", "filter by section ID")
	limit := fs.Int("limit", 0, "maximum number of results (0 = no limit)")
	flat := fs.Bool("flat", false, "flat bullet list, no headings or notes")
	format := fs.String("format", "markdown", "output format (markdown|json)")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	var statusFilter []TaskStatus
	for _, s := range statuses {
		if !validStatus(s) {
			fmt.Fprintf(os.Stderr, "invalid status: %s\n", s)
			return 1
		}
		statusFilter = append(statusFilter, TaskStatus(s))
	}

	filter := Filter{
		Status:    statusFilter,
		PhaseID:   *phase,
		SectionID: *section,
		Limit:     *limit,
	}

	tasks, err := storage.List(filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing tasks: %v\n", err)
		return 1
	}

	list, err := storage.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading task list: %v\n", err)
		return 1
	}

	var out string
	switch *format {
	case "json":
		out, err = FormatListJSON(tasks)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error formatting JSON: %v\n", err)
			return 1
		}
	default:
		out = FormatListMarkdown(tasks, list, *flat)
	}

	fmt.Fprint(w, out)
	return 0
}

func getCmd(args []string, w io.Writer, storage Storage) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "usage: lorah task get <id> [--format=json|markdown]")
		return 1
	}
	id := args[0]

	fs := flag.NewFlagSet("lorah task get", flag.ContinueOnError)
	format := fs.String("format", "markdown", "output format (markdown|json)")
	if err := fs.Parse(args[1:]); err != nil {
		return 1
	}

	task, err := storage.Get(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	list, err := storage.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading task list: %v\n", err)
		return 1
	}

	var out string
	switch *format {
	case "json":
		out, err = FormatTaskJSON(task)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error formatting JSON: %v\n", err)
			return 1
		}
	default:
		out = FormatTaskMarkdown(task, list)
	}

	fmt.Fprint(w, out)
	return 0
}

func createCmd(args []string, w io.Writer, storage Storage) int {
	fs := flag.NewFlagSet("lorah task create", flag.ContinueOnError)
	subject := fs.String("subject", "", "task subject (required)")
	status := fs.String("status", "pending", "task status")
	phase := fs.String("phase", "", "existing phase ID")
	phaseName := fs.String("phase-name", "", "phase name (creates new phase if --phase omitted)")
	phaseDesc := fs.String("phase-description", "", "phase description")
	section := fs.String("section", "", "existing section ID")
	sectionName := fs.String("section-name", "", "section name (creates new section if --section omitted)")
	sectionDesc := fs.String("section-description", "", "section description")
	projectName := fs.String("project-name", "", "project name")
	projectDesc := fs.String("project-description", "", "project description")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *subject == "" {
		fmt.Fprintln(os.Stderr, "--subject is required")
		return 1
	}
	if !validStatus(*status) {
		fmt.Fprintf(os.Stderr, "invalid status: %s\n", *status)
		return 1
	}

	list, err := storage.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading task list: %v\n", err)
		return 1
	}

	listModified := false

	if *projectName != "" {
		list.Name = *projectName
		listModified = true
	}
	if *projectDesc != "" {
		list.Description = *projectDesc
		listModified = true
	}

	// Resolve or create phase.
	var phaseID string
	var newPhaseID string
	if *phase != "" {
		phaseID = *phase
		if *phaseName != "" || *phaseDesc != "" {
			for i, p := range list.Phases {
				if p.ID == phaseID {
					if *phaseName != "" {
						list.Phases[i].Name = *phaseName
					}
					if *phaseDesc != "" {
						list.Phases[i].Description = *phaseDesc
					}
					listModified = true
					break
				}
			}
		}
	} else if *phaseName != "" || *phaseDesc != "" {
		newPhaseID = generateID()
		phaseID = newPhaseID
		p := Phase{ID: newPhaseID}
		if *phaseName != "" {
			p.Name = *phaseName
		}
		if *phaseDesc != "" {
			p.Description = *phaseDesc
		}
		list.Phases = append(list.Phases, p)
		listModified = true
	}

	// Resolve or create section.
	var sectionID string
	var newSectionID string
	if *section != "" {
		if phaseID == "" {
			fmt.Fprintln(os.Stderr, "--section requires a phase context (--phase or --phase-name)")
			return 1
		}
		sectionID = *section
		if *sectionName != "" || *sectionDesc != "" {
			for i, s := range list.Sections {
				if s.ID == sectionID {
					if *sectionName != "" {
						list.Sections[i].Name = *sectionName
					}
					if *sectionDesc != "" {
						list.Sections[i].Description = *sectionDesc
					}
					listModified = true
					break
				}
			}
		}
	} else if *sectionName != "" || *sectionDesc != "" {
		if phaseID == "" {
			fmt.Fprintln(os.Stderr, "--section-name requires a phase context (--phase or --phase-name)")
			return 1
		}
		newSectionID = generateID()
		sectionID = newSectionID
		s := Section{ID: newSectionID, PhaseID: phaseID}
		if *sectionName != "" {
			s.Name = *sectionName
		}
		if *sectionDesc != "" {
			s.Description = *sectionDesc
		}
		list.Sections = append(list.Sections, s)
		listModified = true
	}

	if listModified {
		if err := storage.Save(list); err != nil {
			fmt.Fprintf(os.Stderr, "error saving task list: %v\n", err)
			return 1
		}
	}

	task := &Task{
		ID:          generateID(),
		Subject:     *subject,
		Status:      TaskStatus(*status),
		PhaseID:     phaseID,
		SectionID:   sectionID,
		LastUpdated: time.Now(),
	}
	if err := storage.Create(task); err != nil {
		fmt.Fprintf(os.Stderr, "error creating task: %v\n", err)
		return 1
	}

	if newPhaseID != "" {
		fmt.Fprintf(w, "phase %s\n", newPhaseID)
	}
	if newSectionID != "" {
		fmt.Fprintf(w, "section %s\n", newSectionID)
	}
	fmt.Fprintf(w, "task %s\n", task.ID)
	return 0
}

func updateCmd(args []string, w io.Writer, storage Storage) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "usage: lorah task update <id> [flags...]")
		return 1
	}
	id := args[0]

	fs := flag.NewFlagSet("lorah task update", flag.ContinueOnError)
	statusFlag := fs.String("status", "", "task status")
	subjectFlag := fs.String("subject", "", "task subject")
	notesFlag := fs.String("notes", "", "task notes")
	phaseFlag := fs.String("phase", "", "existing phase ID")
	phaseNameFlag := fs.String("phase-name", "", "phase name")
	phaseDescFlag := fs.String("phase-description", "", "phase description")
	sectionFlag := fs.String("section", "", "existing section ID")
	sectionNameFlag := fs.String("section-name", "", "section name")
	sectionDescFlag := fs.String("section-description", "", "section description")
	projectNameFlag := fs.String("project-name", "", "project name")
	projectDescFlag := fs.String("project-description", "", "project description")
	if err := fs.Parse(args[1:]); err != nil {
		return 1
	}

	provided := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { provided[f.Name] = true })

	if provided["status"] && !validStatus(*statusFlag) {
		fmt.Fprintf(os.Stderr, "invalid status: %s\n", *statusFlag)
		return 1
	}
	if provided["phase-name"] && !provided["phase"] {
		fmt.Fprintln(os.Stderr, "--phase-name requires --phase")
		return 1
	}
	if provided["phase-description"] && !provided["phase"] {
		fmt.Fprintln(os.Stderr, "--phase-description requires --phase")
		return 1
	}
	if provided["section"] && !provided["phase"] {
		fmt.Fprintln(os.Stderr, "--section requires --phase")
		return 1
	}
	if provided["section-name"] && !provided["section"] {
		fmt.Fprintln(os.Stderr, "--section-name requires --section")
		return 1
	}
	if provided["section-description"] && !provided["section"] {
		fmt.Fprintln(os.Stderr, "--section-description requires --section")
		return 1
	}

	task, err := storage.Get(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	if provided["status"] {
		task.Status = TaskStatus(*statusFlag)
	}
	if provided["subject"] {
		task.Subject = *subjectFlag
	}
	if provided["notes"] {
		task.Notes = *notesFlag
	}
	if provided["phase"] {
		task.PhaseID = *phaseFlag
	}
	if provided["section"] {
		task.SectionID = *sectionFlag
	}
	task.LastUpdated = time.Now()

	if err := storage.Update(task); err != nil {
		fmt.Fprintf(os.Stderr, "error updating task: %v\n", err)
		return 1
	}

	// Handle list-level mutations.
	needsList := provided["project-name"] || provided["project-description"] ||
		provided["phase-name"] || provided["phase-description"] ||
		provided["section-name"] || provided["section-description"]
	if needsList {
		list, err := storage.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading task list: %v\n", err)
			return 1
		}
		listModified := false
		if provided["project-name"] {
			list.Name = *projectNameFlag
			listModified = true
		}
		if provided["project-description"] {
			list.Description = *projectDescFlag
			listModified = true
		}
		if provided["phase-name"] && provided["phase"] {
			for i, p := range list.Phases {
				if p.ID == *phaseFlag {
					list.Phases[i].Name = *phaseNameFlag
					listModified = true
					break
				}
			}
		}
		if provided["phase-description"] && provided["phase"] {
			for i, p := range list.Phases {
				if p.ID == *phaseFlag {
					list.Phases[i].Description = *phaseDescFlag
					listModified = true
					break
				}
			}
		}
		if provided["section-name"] && provided["section"] {
			for i, s := range list.Sections {
				if s.ID == *sectionFlag {
					list.Sections[i].Name = *sectionNameFlag
					listModified = true
					break
				}
			}
		}
		if provided["section-description"] && provided["section"] {
			for i, s := range list.Sections {
				if s.ID == *sectionFlag {
					list.Sections[i].Description = *sectionDescFlag
					listModified = true
					break
				}
			}
		}
		if listModified {
			if err := storage.Save(list); err != nil {
				fmt.Fprintf(os.Stderr, "error saving task list: %v\n", err)
				return 1
			}
		}
	}

	_ = w
	return 0
}

func deleteCmd(args []string, w io.Writer, storage Storage) int {
	_ = w
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "usage: lorah task delete <id>")
		return 1
	}
	id := args[0]
	if err := storage.Delete(id); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func exportCmd(args []string, w io.Writer, storage Storage) int {
	fs := flag.NewFlagSet("lorah task export", flag.ContinueOnError)
	var statuses multiFlag
	fs.Var(&statuses, "status", "filter by status (repeatable)")
	output := fs.String("output", "", "output file path (default stdout)")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	var statusFilter []TaskStatus
	for _, s := range statuses {
		if !validStatus(s) {
			fmt.Fprintf(os.Stderr, "invalid status: %s\n", s)
			return 1
		}
		statusFilter = append(statusFilter, TaskStatus(s))
	}

	tasks, err := storage.List(Filter{Status: statusFilter})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing tasks: %v\n", err)
		return 1
	}

	list, err := storage.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading task list: %v\n", err)
		return 1
	}

	out := FormatExportMarkdown(tasks, list)

	if *output != "" {
		if err := os.WriteFile(*output, []byte(out), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output file: %v\n", err)
			return 1
		}
		return 0
	}

	fmt.Fprint(w, out)
	return 0
}
