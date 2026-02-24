package lorah

import (
	"testing"
)

func TestGetPreset(t *testing.T) {
	tests := []struct {
		name     string
		wantNil  bool
		wantName string
	}{
		{name: "python", wantNil: false, wantName: "python"},
		{name: "go", wantNil: false, wantName: "go"},
		{name: "rust", wantNil: false, wantName: "rust"},
		{name: "web-nodejs", wantNil: false, wantName: "web-nodejs"},
		{name: "read-only", wantNil: false, wantName: "read-only"},
		{name: "nonexistent", wantNil: true},
		{name: "", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPreset(tt.name)
			if tt.wantNil {
				if got != nil {
					t.Errorf("GetPreset(%q) = %v, want nil", tt.name, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("GetPreset(%q) = nil, want non-nil", tt.name)
			}
			if got.Name != tt.wantName {
				t.Errorf("GetPreset(%q).Name = %q, want %q", tt.name, got.Name, tt.wantName)
			}
			if got.Description == "" {
				t.Errorf("GetPreset(%q).Description is empty", tt.name)
			}
			if len(got.Config) == 0 {
				t.Errorf("GetPreset(%q).Config is empty", tt.name)
			}
		})
	}
}

func TestListPresets(t *testing.T) {
	summaries := ListPresets()

	if len(summaries) == 0 {
		t.Fatal("ListPresets() returned empty list")
	}

	// Verify all expected presets are present
	expected := []string{"python", "go", "rust", "web-nodejs", "read-only"}
	nameMap := make(map[string]bool)
	for _, s := range summaries {
		nameMap[s.Name] = true
	}
	for _, name := range expected {
		if !nameMap[name] {
			t.Errorf("ListPresets() missing expected preset %q", name)
		}
	}

	// Verify canonical order
	for i, name := range expected {
		if i < len(summaries) && summaries[i].Name != name {
			t.Errorf("ListPresets()[%d].Name = %q, want %q", i, summaries[i].Name, name)
		}
	}

	// Verify each summary has non-empty description
	for _, s := range summaries {
		if s.Description == "" {
			t.Errorf("ListPresets() preset %q has empty description", s.Name)
		}
	}
}

func TestPresetConfigs(t *testing.T) {
	// Python should have PyPI domains
	python := GetPreset("python")
	if python == nil {
		t.Fatal("python preset not found")
	}
	claude, ok := python.Config["claude"].(map[string]any)
	if !ok {
		t.Fatal("python preset missing claude config")
	}
	settings, ok := claude["settings"].(map[string]any)
	if !ok {
		t.Fatal("python preset missing settings")
	}
	sandbox, ok := settings["sandbox"].(map[string]any)
	if !ok {
		t.Fatal("python preset missing sandbox")
	}
	network, ok := sandbox["network"].(map[string]any)
	if !ok {
		t.Fatal("python preset missing network")
	}
	domains, ok := network["allowedDomains"].([]string)
	if !ok {
		t.Fatal("python preset missing allowedDomains")
	}
	found := false
	for _, d := range domains {
		if d == "pypi.org" {
			found = true
			break
		}
	}
	if !found {
		t.Error("python preset missing pypi.org in allowedDomains")
	}

	// Go should have golang.org proxy
	goPreset := GetPreset("go")
	if goPreset == nil {
		t.Fatal("go preset not found")
	}
	goClaude, ok := goPreset.Config["claude"].(map[string]any)
	if !ok {
		t.Fatal("go preset missing claude config")
	}
	goSettings, ok := goClaude["settings"].(map[string]any)
	if !ok {
		t.Fatal("go preset missing settings")
	}
	goSandbox, ok := goSettings["sandbox"].(map[string]any)
	if !ok {
		t.Fatal("go preset missing sandbox")
	}
	goNetwork, ok := goSandbox["network"].(map[string]any)
	if !ok {
		t.Fatal("go preset missing network")
	}
	goDomains, ok := goNetwork["allowedDomains"].([]string)
	if !ok {
		t.Fatal("go preset missing allowedDomains")
	}
	found = false
	for _, d := range goDomains {
		if d == "proxy.golang.org" {
			found = true
			break
		}
	}
	if !found {
		t.Error("go preset missing proxy.golang.org in allowedDomains")
	}

	// web-nodejs should have allowLocalBinding = true
	webNodeJS := GetPreset("web-nodejs")
	if webNodeJS == nil {
		t.Fatal("web-nodejs preset not found")
	}
	webClaude, ok := webNodeJS.Config["claude"].(map[string]any)
	if !ok {
		t.Fatal("web-nodejs preset missing claude config")
	}
	webSettings, ok := webClaude["settings"].(map[string]any)
	if !ok {
		t.Fatal("web-nodejs preset missing settings")
	}
	webSandbox, ok := webSettings["sandbox"].(map[string]any)
	if !ok {
		t.Fatal("web-nodejs preset missing sandbox")
	}
	webNetwork, ok := webSandbox["network"].(map[string]any)
	if !ok {
		t.Fatal("web-nodejs preset missing network")
	}
	allowBinding, ok := webNetwork["allowLocalBinding"].(bool)
	if !ok || !allowBinding {
		t.Error("web-nodejs preset should have allowLocalBinding = true")
	}

	// read-only should have bypassPermissions
	readOnly := GetPreset("read-only")
	if readOnly == nil {
		t.Fatal("read-only preset not found")
	}
	roClaude, ok := readOnly.Config["claude"].(map[string]any)
	if !ok {
		t.Fatal("read-only preset missing claude config")
	}
	roSettings, ok := roClaude["settings"].(map[string]any)
	if !ok {
		t.Fatal("read-only preset missing settings")
	}
	roPerms, ok := roSettings["permissions"].(map[string]any)
	if !ok {
		t.Fatal("read-only preset missing permissions")
	}
	permMode, ok := roPerms["defaultMode"].(string)
	if !ok || permMode != "bypassPermissions" {
		t.Errorf("read-only preset defaultMode = %q, want bypassPermissions", permMode)
	}
}
