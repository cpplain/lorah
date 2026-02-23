package lorah

import (
	"testing"
)

func TestGenerateSchema(t *testing.T) {
	s := GenerateSchema()

	// Top-level fields that must be present
	requiredTopLevel := []string{
		"model",
		"max_turns",
		"max_iterations",
		"auto_continue_delay",
		"tools",
		"security",
		"error_recovery",
		"post_run_instructions",
	}

	for _, field := range requiredTopLevel {
		if _, ok := s[field]; !ok {
			t.Errorf("GenerateSchema() missing top-level field %q", field)
		}
	}
}

func TestSchemaModel(t *testing.T) {
	s := GenerateSchema()

	model, ok := s["model"]
	if !ok {
		t.Fatal("schema missing 'model' field")
	}
	if model.Type != "string" {
		t.Errorf("model.Type = %q, want string", model.Type)
	}
	if model.Description == "" {
		t.Error("model.Description is empty")
	}
	if model.Default == nil {
		t.Error("model.Default is nil")
	}
	if len(model.Options) == 0 {
		t.Error("model.Options is empty")
	}
}

func TestSchemaTools(t *testing.T) {
	s := GenerateSchema()

	tools, ok := s["tools"]
	if !ok {
		t.Fatal("schema missing 'tools' field")
	}
	if tools.Type != "object" {
		t.Errorf("tools.Type = %q, want object", tools.Type)
	}
	if len(tools.Fields) == 0 {
		t.Error("tools.Fields is empty")
	}

	builtin, ok := tools.Fields["builtin"]
	if !ok {
		t.Fatal("tools.Fields missing 'builtin'")
	}
	if builtin.Type != "array" {
		t.Errorf("tools.builtin.Type = %q, want array", builtin.Type)
	}
	if len(builtin.Options) == 0 {
		t.Error("tools.builtin.Options is empty")
	}

	mcpServers, ok := tools.Fields["mcp_servers"]
	if !ok {
		t.Fatal("tools.Fields missing 'mcp_servers'")
	}
	if mcpServers.Type != "object" {
		t.Errorf("tools.mcp_servers.Type = %q, want object", mcpServers.Type)
	}
}

func TestSchemaSecurity(t *testing.T) {
	s := GenerateSchema()

	security, ok := s["security"]
	if !ok {
		t.Fatal("schema missing 'security' field")
	}

	permMode, ok := security.Fields["permission_mode"]
	if !ok {
		t.Fatal("security missing 'permission_mode'")
	}
	if len(permMode.Enum) == 0 {
		t.Error("permission_mode.Enum is empty")
	}
	// Verify expected values
	expected := map[string]bool{
		"default":           false,
		"acceptEdits":       false,
		"bypassPermissions": false,
		"plan":              false,
	}
	for _, v := range permMode.Enum {
		if _, ok := expected[v]; !ok {
			t.Errorf("unexpected permission_mode enum value: %q", v)
		}
		expected[v] = true
	}
	for v, found := range expected {
		if !found {
			t.Errorf("permission_mode.Enum missing %q", v)
		}
	}

	sandbox, ok := security.Fields["sandbox"]
	if !ok {
		t.Fatal("security missing 'sandbox'")
	}
	if _, ok := sandbox.Fields["enabled"]; !ok {
		t.Error("sandbox missing 'enabled'")
	}
	if _, ok := sandbox.Fields["network"]; !ok {
		t.Error("sandbox missing 'network'")
	}
}

func TestSchemaErrorRecovery(t *testing.T) {
	s := GenerateSchema()

	er, ok := s["error_recovery"]
	if !ok {
		t.Fatal("schema missing 'error_recovery' field")
	}

	requiredFields := []string{
		"max_consecutive_errors",
		"initial_backoff_seconds",
		"max_backoff_seconds",
		"backoff_multiplier",
	}
	for _, f := range requiredFields {
		if _, ok := er.Fields[f]; !ok {
			t.Errorf("error_recovery missing field %q", f)
		}
	}
}
