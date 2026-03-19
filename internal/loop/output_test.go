package loop

import (
	"fmt"
	"strings"
	"testing"
)

// TestPrintSection_WithContent verifies ANSI-colored icon, bold label, trimmed content, and trailing blank line.
func TestPrintSection_WithContent(t *testing.T) {
	stdout, _ := captureOutput(func() {
		printSection("MyLabel", colorBlue, "  hello world  ")
	})

	if !strings.Contains(stdout, colorBlue+"⏺"+colorReset) {
		t.Errorf("expected colored icon in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, colorBold+"MyLabel"+colorReset) {
		t.Errorf("expected bold label in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "hello world") {
		t.Errorf("expected content in output, got: %q", stdout)
	}
	if strings.Contains(stdout, "  hello world  ") {
		t.Errorf("expected content to be trimmed, got: %q", stdout)
	}
	if !strings.HasSuffix(stdout, "\n\n") {
		t.Errorf("expected trailing blank line, got: %q", stdout)
	}
}

// TestPrintSection_EmptyContent verifies that empty content omits the content line but still prints trailing blank line.
func TestPrintSection_EmptyContent(t *testing.T) {
	stdout, _ := captureOutput(func() {
		printSection("MyLabel", colorGreen, "")
	})

	if !strings.HasSuffix(stdout, "\n\n") {
		t.Errorf("expected trailing blank line, got: %q", stdout)
	}
	// Output: "header\n\n" → split by \n gives ["header", "", ""] → 3 parts
	parts := strings.Split(stdout, "\n")
	if len(parts) != 3 {
		t.Errorf("expected 3 newline-delimited parts for empty content (header + blank + trailing), got %d: %v", len(parts), parts)
	}
}

// TestPrintMessages_AssistantText verifies assistant text blocks display with "Claude" label.
func TestPrintMessages_AssistantText(t *testing.T) {
	input := `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello, world!"}]}}` + "\n"

	stdout, _ := captureOutput(func() {
		printMessages(strings.NewReader(input))
	})

	if !strings.Contains(stdout, "Claude") {
		t.Errorf("expected 'Claude' in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Hello, world!") {
		t.Errorf("expected text content in output, got: %q", stdout)
	}
}

// TestPrintMessages_AssistantThinking verifies thinking blocks display with "Claude (thinking)" label.
func TestPrintMessages_AssistantThinking(t *testing.T) {
	input := `{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"Deep thoughts..."}]}}` + "\n"

	stdout, _ := captureOutput(func() {
		printMessages(strings.NewReader(input))
	})

	if !strings.Contains(stdout, "Claude (thinking)") {
		t.Errorf("expected 'Claude (thinking)' in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Deep thoughts...") {
		t.Errorf("expected thinking content in output, got: %q", stdout)
	}
}

// TestPrintMessages_ToolUse verifies tool_use blocks display with correct label, input content, and green color.
func TestPrintMessages_ToolUse(t *testing.T) {
	tests := []struct {
		toolName   string
		inputKey   string
		inputValue string
	}{
		{"Bash", "command", "ls -la"},
		{"Read", "file_path", "/path/to/file.go"},
		{"Edit", "file_path", "/path/to/edit.go"},
		{"Write", "file_path", "/path/to/write.go"},
		{"Grep", "pattern", "func.*main"},
		{"Glob", "pattern", "**/*.go"},
		{"WebFetch", "url", "https://example.com"},
		{"Task", "description", "do something important"},
		{"Agent", "prompt", "run this agent now"},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			input := fmt.Sprintf(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":%q,"input":{%q:%q}}]}}`,
				tt.toolName, tt.inputKey, tt.inputValue) + "\n"

			stdout, _ := captureOutput(func() {
				printMessages(strings.NewReader(input))
			})

			if !strings.Contains(stdout, tt.toolName) {
				t.Errorf("expected tool name %q in output, got: %q", tt.toolName, stdout)
			}
			if !strings.Contains(stdout, tt.inputValue) {
				t.Errorf("expected input value %q in output, got: %q", tt.inputValue, stdout)
			}
			if !strings.Contains(stdout, colorGreen) {
				t.Errorf("expected green color for tool section, got: %q", stdout)
			}
		})
	}
}

// TestPrintMessages_UnknownTool verifies that unknown tools display header only with no content.
func TestPrintMessages_UnknownTool(t *testing.T) {
	input := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"UnknownTool","input":{"foo":"bar"}}]}}` + "\n"

	stdout, _ := captureOutput(func() {
		printMessages(strings.NewReader(input))
	})

	if !strings.Contains(stdout, "UnknownTool") {
		t.Errorf("expected tool name 'UnknownTool' in output, got: %q", stdout)
	}
	if strings.Contains(stdout, "bar") {
		t.Errorf("expected no content for unknown tool, got: %q", stdout)
	}
}

// TestPrintMessages_ErrorResult verifies error result messages display with "Result (error)" label and red color.
func TestPrintMessages_ErrorResult(t *testing.T) {
	input := `{"type":"result","is_error":true,"result":"something went wrong"}` + "\n"

	stdout, _ := captureOutput(func() {
		printMessages(strings.NewReader(input))
	})

	if !strings.Contains(stdout, "Result (error)") {
		t.Errorf("expected 'Result (error)' in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "something went wrong") {
		t.Errorf("expected error text in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, colorRed) {
		t.Errorf("expected red color for error section, got: %q", stdout)
	}
}

// TestPrintMessages_NonErrorResult verifies non-error result messages are silently skipped.
func TestPrintMessages_NonErrorResult(t *testing.T) {
	input := `{"type":"result","is_error":false,"result":"done"}` + "\n"

	stdout, _ := captureOutput(func() {
		printMessages(strings.NewReader(input))
	})

	if stdout != "" {
		t.Errorf("expected no output for non-error result, got: %q", stdout)
	}
}

// TestPrintMessages_UnknownType verifies unknown message types are silently skipped.
func TestPrintMessages_UnknownType(t *testing.T) {
	input := `{"type":"system","content":"some system message"}` + "\n"

	stdout, _ := captureOutput(func() {
		printMessages(strings.NewReader(input))
	})

	if stdout != "" {
		t.Errorf("expected no output for unknown message type, got: %q", stdout)
	}
}

// TestPrintMessages_MalformedJSON verifies malformed JSON lines are silently skipped.
func TestPrintMessages_MalformedJSON(t *testing.T) {
	input := "not valid json\n"

	stdout, _ := captureOutput(func() {
		printMessages(strings.NewReader(input))
	})

	if stdout != "" {
		t.Errorf("expected no output for malformed JSON, got: %q", stdout)
	}
}

// TestPrintMessages_MultiLineTruncation verifies multi-line tool input is truncated to first line plus count.
func TestPrintMessages_MultiLineTruncation(t *testing.T) {
	multiLineCmd := "line1\nline2\nline3"
	input := fmt.Sprintf(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":%q}}]}}`, multiLineCmd) + "\n"

	stdout, _ := captureOutput(func() {
		printMessages(strings.NewReader(input))
	})

	if !strings.Contains(stdout, "line1") {
		t.Errorf("expected first line in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "... +2 lines") {
		t.Errorf("expected truncation indicator '... +2 lines', got: %q", stdout)
	}
	if strings.Contains(stdout, "line2") || strings.Contains(stdout, "line3") {
		t.Errorf("expected subsequent lines to be truncated, got: %q", stdout)
	}
}
