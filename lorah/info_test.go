package lorah

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetTemplate(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{name: "config.json", wantNil: false},
		{name: "spec.md", wantNil: false},
		{name: "initialization.md", wantNil: false},
		{name: "implementation.md", wantNil: false},
		{name: "nonexistent.txt", wantNil: true},
		{name: "", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTemplate(tt.name)
			if tt.wantNil {
				if got != nil {
					t.Errorf("GetTemplate(%q) = %v, want nil", tt.name, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("GetTemplate(%q) = nil, want non-nil", tt.name)
			}
			if got.Name != tt.name {
				t.Errorf("GetTemplate(%q).Name = %q, want %q", tt.name, got.Name, tt.name)
			}
			if got.Description == "" {
				t.Errorf("GetTemplate(%q).Description is empty", tt.name)
			}
			if got.TargetPath == "" {
				t.Errorf("GetTemplate(%q).TargetPath is empty", tt.name)
			}
			if got.Content == "" {
				t.Errorf("GetTemplate(%q).Content is empty", tt.name)
			}
		})
	}
}

func TestListTemplates(t *testing.T) {
	summaries := ListTemplates()

	if len(summaries) == 0 {
		t.Fatal("ListTemplates() returned empty list")
	}

	// Verify all expected templates are present
	expected := map[string]bool{
		"config.json":       false,
		"spec.md":           false,
		"initialization.md": false,
		"implementation.md": false,
	}
	for _, s := range summaries {
		if _, ok := expected[s.Name]; ok {
			expected[s.Name] = true
		}
		if s.Description == "" {
			t.Errorf("template %q has empty description", s.Name)
		}
		if s.TargetPath == "" {
			t.Errorf("template %q has empty target path", s.Name)
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("ListTemplates() missing expected template %q", name)
		}
	}
}

func TestTemplateContent(t *testing.T) {
	// config.json should be valid JSON
	configTpl := GetTemplate("config.json")
	if configTpl == nil {
		t.Fatal("config.json template not found")
	}
	if !strings.Contains(configTpl.Content, "model") {
		t.Error("config.json template should contain 'model'")
	}

	// spec.md should be markdown
	specTpl := GetTemplate("spec.md")
	if specTpl == nil {
		t.Fatal("spec.md template not found")
	}
	if !strings.Contains(specTpl.Content, "# Project Specification") {
		t.Error("spec.md should contain '# Project Specification'")
	}

	// initialization.md should have INIT PHASE content
	initTpl := GetTemplate("initialization.md")
	if initTpl == nil {
		t.Fatal("initialization.md template not found")
	}
	if !strings.Contains(initTpl.Content, "INIT PHASE") {
		t.Error("initialization.md should contain 'INIT PHASE'")
	}

	// implementation.md should have BUILD PHASE content
	buildTpl := GetTemplate("implementation.md")
	if buildTpl == nil {
		t.Fatal("implementation.md template not found")
	}
	if !strings.Contains(buildTpl.Content, "BUILD PHASE") {
		t.Error("implementation.md should contain 'BUILD PHASE'")
	}
}

func TestGetGuide(t *testing.T) {
	guide := GetGuide()

	if guide.Title == "" {
		t.Error("guide.Title is empty")
	}
	if guide.Content == "" {
		t.Error("guide.Content is empty")
	}
	if len(guide.Sections) == 0 {
		t.Error("guide.Sections is empty")
	}

	// Check for expected sections
	sectionTitles := make(map[string]bool)
	for _, s := range guide.Sections {
		sectionTitles[s.Title] = true
		if s.Content == "" {
			t.Errorf("section %q has empty content", s.Title)
		}
	}

	// Should have at least "Getting Started"
	if !sectionTitles["Getting Started"] {
		t.Error("guide missing 'Getting Started' section")
	}
}

func TestInitProject(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "lorah-init-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Run init
	if err := InitProject(tmpDir); err != nil {
		t.Fatalf("InitProject() returned error: %v", err)
	}

	// Verify expected files were created
	harnessDir := filepath.Join(tmpDir, ".lorah")
	expectedFiles := []string{
		filepath.Join(harnessDir, "config.json"),
		filepath.Join(harnessDir, "spec.md"),
		filepath.Join(harnessDir, TaskListFile),
		filepath.Join(harnessDir, AgentProgressFile),
		filepath.Join(harnessDir, "prompts", "initialization.md"),
		filepath.Join(harnessDir, "prompts", "implementation.md"),
	}

	for _, f := range expectedFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("InitProject() did not create expected file: %s", f)
		}
	}

	// Verify config.json has content
	configContent, err := os.ReadFile(filepath.Join(harnessDir, "config.json"))
	if err != nil {
		t.Fatal("failed to read config.json:", err)
	}
	if len(configContent) == 0 {
		t.Error("config.json is empty")
	}
}

func TestInitProjectAlreadyExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lorah-init-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize once
	if err := InitProject(tmpDir); err != nil {
		t.Fatalf("first InitProject() returned error: %v", err)
	}

	// Try to initialize again — should return error
	err = InitProject(tmpDir)
	if err == nil {
		t.Error("second InitProject() should return error when config already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error message should contain 'already exists', got: %v", err)
	}
}

func TestCmdInfoTemplateList(t *testing.T) {
	// Capture stdout by temporarily redirecting (not easily done without pipe)
	// Instead, just test that it doesn't error
	if err := CmdInfoTemplate("", true, false, false); err != nil {
		t.Errorf("CmdInfoTemplate list returned error: %v", err)
	}
}

func TestCmdInfoTemplateJSON(t *testing.T) {
	if err := CmdInfoTemplate("", true, false, true); err != nil {
		t.Errorf("CmdInfoTemplate list JSON returned error: %v", err)
	}
}

func TestCmdInfoTemplateByName(t *testing.T) {
	if err := CmdInfoTemplate("config.json", false, false, false); err != nil {
		t.Errorf("CmdInfoTemplate by name returned error: %v", err)
	}
}

func TestCmdInfoSchemaHuman(t *testing.T) {
	if err := CmdInfoSchema(false); err != nil {
		t.Errorf("CmdInfoSchema human returned error: %v", err)
	}
}

func TestCmdInfoSchemaJSON(t *testing.T) {
	if err := CmdInfoSchema(true); err != nil {
		t.Errorf("CmdInfoSchema JSON returned error: %v", err)
	}
}

func TestCmdInfoPresetList(t *testing.T) {
	if err := CmdInfoPreset("", true, false); err != nil {
		t.Errorf("CmdInfoPreset list returned error: %v", err)
	}
}

func TestCmdInfoPresetByName(t *testing.T) {
	if err := CmdInfoPreset("python", false, false); err != nil {
		t.Errorf("CmdInfoPreset by name returned error: %v", err)
	}
}

func TestCmdInfoPresetJSON(t *testing.T) {
	if err := CmdInfoPreset("go", false, true); err != nil {
		t.Errorf("CmdInfoPreset JSON returned error: %v", err)
	}
}

func TestCmdInfoGuideHuman(t *testing.T) {
	if err := CmdInfoGuide(false); err != nil {
		t.Errorf("CmdInfoGuide human returned error: %v", err)
	}
}

func TestCmdInfoGuideJSON(t *testing.T) {
	if err := CmdInfoGuide(true); err != nil {
		t.Errorf("CmdInfoGuide JSON returned error: %v", err)
	}
}
