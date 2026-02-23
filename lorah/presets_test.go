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
	domains, ok := python.Config["security.sandbox.network.allowed_domains"].([]string)
	if !ok {
		t.Fatal("python preset missing allowed_domains")
	}
	found := false
	for _, d := range domains {
		if d == "pypi.org" {
			found = true
			break
		}
	}
	if !found {
		t.Error("python preset missing pypi.org in allowed_domains")
	}

	// Go should have golang.org proxy
	goPreset := GetPreset("go")
	if goPreset == nil {
		t.Fatal("go preset not found")
	}
	goDomains, ok := goPreset.Config["security.sandbox.network.allowed_domains"].([]string)
	if !ok {
		t.Fatal("go preset missing allowed_domains")
	}
	found = false
	for _, d := range goDomains {
		if d == "proxy.golang.org" {
			found = true
			break
		}
	}
	if !found {
		t.Error("go preset missing proxy.golang.org in allowed_domains")
	}

	// web-nodejs should have allow_local_binding = true
	webNodeJS := GetPreset("web-nodejs")
	if webNodeJS == nil {
		t.Fatal("web-nodejs preset not found")
	}
	allowBinding, ok := webNodeJS.Config["security.sandbox.network.allow_local_binding"].(bool)
	if !ok || !allowBinding {
		t.Error("web-nodejs preset should have allow_local_binding = true")
	}

	// read-only should have bypassPermissions
	readOnly := GetPreset("read-only")
	if readOnly == nil {
		t.Fatal("read-only preset not found")
	}
	permMode, ok := readOnly.Config["security.permission_mode"].(string)
	if !ok || permMode != "bypassPermissions" {
		t.Errorf("read-only preset permission_mode = %q, want bypassPermissions", permMode)
	}
}
