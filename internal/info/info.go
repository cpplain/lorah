// Package info provides template embedding, guide retrieval, and preset
// information for the lorah tool. It handles the info subcommand
// and the init scaffolding workflow.
package info

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "embed"

	"github.com/cpplain/lorah/internal/config"
	"github.com/cpplain/lorah/internal/presets"
	"github.com/cpplain/lorah/internal/schema"
	"github.com/cpplain/lorah/internal/tracking"
)

//go:embed setup-guide.md
var setupGuideContent []byte

// templateFS holds the embedded template files.
// Because embed.FS is initialized at package level, we use go:embed directives.
// We embed each file individually due to the nested structure.
//
//go:embed templates/config.json
var templateConfigJSON []byte

//go:embed templates/spec.md
var templateSpecMD []byte

//go:embed templates/prompts/initialization.md
var templateInitializationMD []byte

//go:embed templates/prompts/implementation.md
var templateImplementationMD []byte

// Template describes a single template file.
type Template struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TargetPath  string `json:"target_path"`
	Content     string `json:"content"`
}

// TemplateSummary holds just the metadata for listing.
type TemplateSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TargetPath  string `json:"target_path"`
}

// templateMeta defines the metadata for each embedded template.
var templateMeta = []TemplateSummary{
	{
		Name:        "config.json",
		Description: "Main harness configuration",
		TargetPath:  ".lorah/config.json",
	},
	{
		Name:        "spec.md",
		Description: "Project specification template",
		TargetPath:  ".lorah/spec.md",
	},
	{
		Name:        "initialization.md",
		Description: "Initialization phase prompt",
		TargetPath:  ".lorah/prompts/initialization.md",
	},
	{
		Name:        "implementation.md",
		Description: "Implementation phase prompt",
		TargetPath:  ".lorah/prompts/implementation.md",
	},
}

// templateContent maps template name to embedded content.
var templateContent = map[string][]byte{
	"config.json":       templateConfigJSON,
	"spec.md":           templateSpecMD,
	"initialization.md": templateInitializationMD,
	"implementation.md": templateImplementationMD,
}

// GetTemplate returns a Template by name, or nil if not found.
func GetTemplate(name string) *Template {
	content, ok := templateContent[name]
	if !ok {
		return nil
	}
	// Find metadata
	for _, meta := range templateMeta {
		if meta.Name == name {
			return &Template{
				Name:        meta.Name,
				Description: meta.Description,
				TargetPath:  meta.TargetPath,
				Content:     string(content),
			}
		}
	}
	return nil
}

// ListTemplates returns summary metadata for all templates in canonical order.
func ListTemplates() []TemplateSummary {
	return templateMeta
}

// Guide holds the setup guide content.
type Guide struct {
	Title    string    `json:"title"`
	Sections []Section `json:"sections"`
	Content  string    `json:"content"`
}

// Section holds a guide section.
type Section struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// GetGuide returns the setup guide parsed into sections.
func GetGuide() Guide {
	content := string(setupGuideContent)

	// Parse markdown sections by ## headers
	var sections []Section
	var currentSection string
	var currentLines []string

	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "## ") {
			// Save previous section
			if currentSection != "" {
				sections = append(sections, Section{
					Title:   currentSection,
					Content: strings.TrimSpace(strings.Join(currentLines, "\n")),
				})
			}
			// Start new section
			currentSection = strings.TrimPrefix(line, "## ")
			currentLines = nil
		} else if currentSection != "" {
			currentLines = append(currentLines, line)
		}
	}

	// Save last section
	if currentSection != "" {
		sections = append(sections, Section{
			Title:   currentSection,
			Content: strings.TrimSpace(strings.Join(currentLines, "\n")),
		})
	}

	return Guide{
		Title:    "Harness Setup Guide",
		Sections: sections,
		Content:  content,
	}
}

// InitProject scaffolds the .lorah/ directory in the given project dir.
// It mirrors the Python cmd_init() behavior.
func InitProject(projectDir string) error {
	harnessDir := filepath.Join(projectDir, config.ConfigDirName)
	configFile := filepath.Join(harnessDir, "config.json")

	// Check if already initialized
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("config already exists: %s\nRemove it first if you want to reinitialize", configFile)
	}

	// Create directory structure
	promptsDir := filepath.Join(harnessDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create prompts directory: %w", err)
	}

	// Write template files
	type fileTarget struct {
		name    string
		dest    string
		content []byte
	}
	files := []fileTarget{
		{"config.json", configFile, templateConfigJSON},
		{"spec.md", filepath.Join(harnessDir, "spec.md"), templateSpecMD},
		{"initialization.md", filepath.Join(promptsDir, "initialization.md"), templateInitializationMD},
		{"implementation.md", filepath.Join(promptsDir, "implementation.md"), templateImplementationMD},
	}

	for _, f := range files {
		if err := os.WriteFile(f.dest, f.content, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", f.dest, err)
		}
	}

	// Create tracking files using shared logic to prevent drift
	if err := tracking.EnsureTrackingFiles(harnessDir); err != nil {
		return err
	}

	fmt.Printf("Created %s/\n", harnessDir)
	fmt.Printf("  - config.json (edit this to configure your project)\n")
	fmt.Printf("  - spec.md (describe what you're building)\n")
	fmt.Printf("  - %s (task tracking checklist)\n", tracking.TaskListFile)
	fmt.Printf("  - %s (progress notes)\n", tracking.AgentProgressFile)
	fmt.Printf("  - prompts/initialization.md (setup phase prompt)\n")
	fmt.Printf("  - prompts/implementation.md (main work phase prompt)\n")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Edit %s with your project specification\n", filepath.Join(harnessDir, "spec.md"))
	fmt.Printf("  2. Edit prompts to guide the agent's behavior\n")
	fmt.Printf("  3. Run: lorah verify --project-dir %s\n", projectDir)
	fmt.Printf("  4. Run: lorah run --project-dir %s\n", projectDir)

	return nil
}

// formatTemplateHuman formats template info for human reading.
func formatTemplateHuman(t Template) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Template: %s\n", t.Name)
	fmt.Fprintf(&sb, "Description: %s\n", t.Description)
	fmt.Fprintf(&sb, "Target Path: %s\n", t.TargetPath)
	sb.WriteString("\n")
	sb.WriteString("Content:\n")
	sb.WriteString(strings.Repeat("-", 60))
	sb.WriteString("\n")
	sb.WriteString(t.Content)
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", 60))
	return sb.String()
}

// formatPresetHuman formats preset info for human reading.
func formatPresetHuman(p presets.Preset) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Preset: %s\n", p.Name)
	fmt.Fprintf(&sb, "Description: %s\n", p.Description)
	sb.WriteString("\n")
	sb.WriteString("Configuration:\n")

	// Sort keys for deterministic output
	keys := make([]string, 0, len(p.Config))
	for k := range p.Config {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := p.Config[key]
		switch v := value.(type) {
		case []string:
			fmt.Fprintf(&sb, "  %s:\n", key)
			for _, item := range v {
				fmt.Fprintf(&sb, "    - %s\n", item)
			}
		case bool:
			fmt.Fprintf(&sb, "  %s = %v\n", key, v)
		case string:
			fmt.Fprintf(&sb, "  %s = %s\n", key, v)
		default:
			fmt.Fprintf(&sb, "  %s = %v\n", key, v)
		}
	}
	return sb.String()
}

// formatSchemaHuman formats schema information for human reading.
func formatSchemaHuman(s schema.Schema) string {
	var sb strings.Builder
	sb.WriteString("Configuration Schema\n")
	sb.WriteString(strings.Repeat("=", 60))
	sb.WriteString("\n\n")

	var formatField func(name string, info schema.FieldInfo, indent int)
	formatField = func(name string, info schema.FieldInfo, indent int) {
		prefix := strings.Repeat("  ", indent)
		fmt.Fprintf(&sb, "%s%s:\n", prefix, name)

		if info.Description != "" {
			fmt.Fprintf(&sb, "%s  Description: %s\n", prefix, info.Description)
		}
		if info.Type != "" {
			fmt.Fprintf(&sb, "%s  Type: %s\n", prefix, info.Type)
		}
		if len(info.Enum) > 0 {
			fmt.Fprintf(&sb, "%s  Options: %s\n", prefix, strings.Join(info.Enum, ", "))
		}
		if len(info.Options) > 0 {
			fmt.Fprintf(&sb, "%s  Options: %s\n", prefix, strings.Join(info.Options, ", "))
		}
		if info.Default != nil {
			switch d := info.Default.(type) {
			case string:
				fmt.Fprintf(&sb, "%s  Default: %q\n", prefix, d)
			default:
				fmt.Fprintf(&sb, "%s  Default: %v\n", prefix, d)
			}
		}

		// Recurse into nested fields
		if len(info.Fields) > 0 {
			fmt.Fprintf(&sb, "%s  Fields:\n", prefix)
			// Sort for deterministic output
			subNames := make([]string, 0, len(info.Fields))
			for n := range info.Fields {
				subNames = append(subNames, n)
			}
			sort.Strings(subNames)
			for _, subName := range subNames {
				formatField(subName, info.Fields[subName], indent+2)
			}
		}
		sb.WriteString("\n")
	}

	// Sort top-level keys for deterministic output
	names := make([]string, 0, len(s))
	for n := range s {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		formatField(name, s[name], 0)
	}

	return sb.String()
}

// CmdInfoTemplate handles `lorah info template`.
func CmdInfoTemplate(name string, listFlag bool, allFlag bool, jsonOutput bool) error {
	if allFlag {
		// Get all templates with full content
		var templates []Template
		for _, meta := range templateMeta {
			t := GetTemplate(meta.Name)
			if t != nil {
				templates = append(templates, *t)
			}
		}

		if jsonOutput {
			data, err := json.MarshalIndent(templates, "", "  ")
			if err != nil {
				return fmt.Errorf("JSON marshal failed: %w", err)
			}
			fmt.Println(string(data))
		} else {
			for i, t := range templates {
				if i > 0 {
					fmt.Println("\n" + strings.Repeat("=", 60) + "\n")
				}
				fmt.Println(formatTemplateHuman(t))
			}
		}
		return nil
	}

	if listFlag {
		summaries := ListTemplates()
		if jsonOutput {
			data, err := json.MarshalIndent(summaries, "", "  ")
			if err != nil {
				return fmt.Errorf("JSON marshal failed: %w", err)
			}
			fmt.Println(string(data))
		} else {
			fmt.Println("Available Templates:")
			fmt.Println(strings.Repeat("-", 60))
			for _, t := range summaries {
				fmt.Printf("  %-20s - %s\n", t.Name, t.Description)
				fmt.Printf("  %-20s   Target: %s\n", "", t.TargetPath)
				fmt.Println()
			}
		}
		return nil
	}

	if name == "" {
		fmt.Println("Error: --name, --list, or --all required")
		fmt.Println("Usage: lorah info template [--name NAME] [--list] [--all]")
		return nil
	}

	t := GetTemplate(name)
	if t == nil {
		fmt.Printf("Error: Template not found: %s\n", name)
		fmt.Println("\nAvailable templates:")
		for _, meta := range templateMeta {
			fmt.Printf("  - %s\n", meta.Name)
		}
		return nil
	}

	if jsonOutput {
		data, err := json.MarshalIndent(t, "", "  ")
		if err != nil {
			return fmt.Errorf("JSON marshal failed: %w", err)
		}
		fmt.Println(string(data))
	} else {
		fmt.Println(formatTemplateHuman(*t))
	}
	return nil
}

// CmdInfoSchema handles `lorah info schema`.
func CmdInfoSchema(jsonOutput bool) error {
	s := schema.GenerateSchema()

	if jsonOutput {
		data, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return fmt.Errorf("JSON marshal failed: %w", err)
		}
		fmt.Println(string(data))
	} else {
		fmt.Println(formatSchemaHuman(s))
	}
	return nil
}

// CmdInfoPreset handles `lorah info preset`.
func CmdInfoPreset(name string, listFlag bool, jsonOutput bool) error {
	if listFlag {
		summaries := presets.ListPresets()
		if jsonOutput {
			data, err := json.MarshalIndent(summaries, "", "  ")
			if err != nil {
				return fmt.Errorf("JSON marshal failed: %w", err)
			}
			fmt.Println(string(data))
		} else {
			fmt.Println("Available Presets:")
			fmt.Println(strings.Repeat("-", 60))
			for _, p := range summaries {
				fmt.Printf("  %-20s - %s\n", p.Name, p.Description)
			}
			fmt.Println()
		}
		return nil
	}

	if name == "" {
		fmt.Println("Error: --name or --list required")
		fmt.Println("Usage: lorah info preset [--name NAME] [--list]")
		return nil
	}

	p := presets.GetPreset(name)
	if p == nil {
		fmt.Printf("Error: Preset not found: %s\n", name)
		fmt.Println("\nAvailable presets:")
		for _, s := range presets.ListPresets() {
			fmt.Printf("  - %s\n", s.Name)
		}
		return nil
	}

	if jsonOutput {
		data, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			return fmt.Errorf("JSON marshal failed: %w", err)
		}
		fmt.Println(string(data))
	} else {
		fmt.Println(formatPresetHuman(*p))
	}
	return nil
}

// CmdInfoGuide handles `lorah info guide`.
func CmdInfoGuide(jsonOutput bool) error {
	guide := GetGuide()

	if jsonOutput {
		data, err := json.MarshalIndent(guide, "", "  ")
		if err != nil {
			return fmt.Errorf("JSON marshal failed: %w", err)
		}
		fmt.Println(string(data))
	} else {
		fmt.Print(guide.Content)
	}
	return nil
}
