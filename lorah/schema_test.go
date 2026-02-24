package lorah

import (
	"testing"
)

func TestGenerateSchema(t *testing.T) {
	s := GenerateSchema()

	// Top-level sections that must be present
	requiredSections := []string{
		"claude",
		"harness",
	}

	for _, section := range requiredSections {
		if _, ok := s[section]; !ok {
			t.Errorf("GenerateSchema() missing top-level section %q", section)
		}
	}

	// Fields that should NOT be present at top level
	removedFields := []string{
		"model",
		"tools",
		"security",
		"post_run_instructions",
	}

	for _, field := range removedFields {
		if _, ok := s[field]; ok {
			t.Errorf("GenerateSchema() should not have field %q at top level", field)
		}
	}
}

func TestSchemaMaxTurns(t *testing.T) {
	s := GenerateSchema()

	claude, ok := s["claude"]
	if !ok {
		t.Fatal("schema missing 'claude' section")
	}

	flags, ok := claude.Fields["flags"]
	if !ok {
		t.Fatal("claude section missing 'flags' field")
	}

	maxTurns, ok := flags.Fields["--max-turns"]
	if !ok {
		t.Fatal("claude.flags section missing '--max-turns' field")
	}
	if maxTurns.Type != "integer" {
		t.Errorf("--max-turns.Type = %q, want integer", maxTurns.Type)
	}
	if maxTurns.Description == "" {
		t.Error("--max-turns.Description is empty")
	}
	if maxTurns.Default == nil {
		t.Error("--max-turns.Default is nil")
	}
}

func TestSchemaMaxIterations(t *testing.T) {
	s := GenerateSchema()

	harness, ok := s["harness"]
	if !ok {
		t.Fatal("schema missing 'harness' section")
	}

	maxIter, ok := harness.Fields["max-iterations"]
	if !ok {
		t.Fatal("harness section missing 'max-iterations' field")
	}
	if maxIter.Type != "integer" {
		t.Errorf("max-iterations.Type = %q, want integer", maxIter.Type)
	}
	if maxIter.Description == "" {
		t.Error("max-iterations.Description is empty")
	}
}

func TestSchemaAutoContinueDelay(t *testing.T) {
	s := GenerateSchema()

	harness, ok := s["harness"]
	if !ok {
		t.Fatal("schema missing 'harness' section")
	}

	delay, ok := harness.Fields["auto-continue-delay"]
	if !ok {
		t.Fatal("harness section missing 'auto-continue-delay' field")
	}
	if delay.Type != "integer" {
		t.Errorf("auto-continue-delay.Type = %q, want integer", delay.Type)
	}
	if delay.Description == "" {
		t.Error("auto-continue-delay.Description is empty")
	}
	if delay.Default == nil {
		t.Error("auto-continue-delay.Default is nil")
	}
}

func TestSchemaErrorRecovery(t *testing.T) {
	s := GenerateSchema()

	harness, ok := s["harness"]
	if !ok {
		t.Fatal("schema missing 'harness' section")
	}

	er, ok := harness.Fields["error-recovery"]
	if !ok {
		t.Fatal("harness section missing 'error-recovery' field")
	}

	if er.Type != "object" {
		t.Errorf("error-recovery.Type = %q, want object", er.Type)
	}

	requiredFields := []string{
		"max-consecutive-errors",
		"initial-backoff-seconds",
		"max-backoff-seconds",
		"backoff-multiplier",
		"max-error-message-length",
	}
	for _, f := range requiredFields {
		if _, ok := er.Fields[f]; !ok {
			t.Errorf("error-recovery missing field %q", f)
		}
	}

	// Check that each field has defaults set
	for _, f := range requiredFields {
		field := er.Fields[f]
		if field.Default == nil {
			t.Errorf("error-recovery.%s missing default value", f)
		}
		if field.Description == "" {
			t.Errorf("error-recovery.%s missing description", f)
		}
	}
}
