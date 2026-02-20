package messages

import (
	"io"
	"strings"
	"testing"
)

var parseMessageTests = []struct {
	name     string
	input    string
	wantType MessageType
	wantErr  bool
	check    func(t *testing.T, msg Message)
}{
	{
		name:     "system init",
		input:    `{"type":"system","subtype":"init","session_id":"abc-123","tools":[]}`,
		wantType: MessageTypeSystem,
		check: func(t *testing.T, msg Message) {
			t.Helper()
			sys := msg.(*SystemMessage)
			if sys.Subtype != "init" {
				t.Errorf("Subtype = %q, want %q", sys.Subtype, "init")
			}
			if got := sys.SessionID(); got != "abc-123" {
				t.Errorf("SessionID() = %q, want %q", got, "abc-123")
			}
		},
	},
	{
		name:     "system non-init",
		input:    `{"type":"system","subtype":"compact"}`,
		wantType: MessageTypeSystem,
		check: func(t *testing.T, msg Message) {
			t.Helper()
			sys := msg.(*SystemMessage)
			if sys.Subtype != "compact" {
				t.Errorf("Subtype = %q, want %q", sys.Subtype, "compact")
			}
			// Non-init messages have no session_id.
			if got := sys.SessionID(); got != "" {
				t.Errorf("SessionID() = %q, want empty", got)
			}
		},
	},
	{
		name:     "assistant text",
		input:    `{"type":"assistant","message":{"model":"claude-3","content":[{"type":"text","text":"Hello!"}]}}`,
		wantType: MessageTypeAssistant,
		check: func(t *testing.T, msg Message) {
			t.Helper()
			ast := msg.(*AssistantMessage)
			if ast.Model != "claude-3" {
				t.Errorf("Model = %q, want %q", ast.Model, "claude-3")
			}
			if len(ast.Content) != 1 {
				t.Fatalf("Content len = %d, want 1", len(ast.Content))
			}
			tb, ok := ast.Content[0].(*TextBlock)
			if !ok {
				t.Fatalf("Content[0] type = %T, want *TextBlock", ast.Content[0])
			}
			if tb.Text != "Hello!" {
				t.Errorf("Text = %q, want %q", tb.Text, "Hello!")
			}
		},
	},
	{
		name:     "assistant thinking block",
		input:    `{"type":"assistant","message":{"model":"claude-3","content":[{"type":"thinking","thinking":"let me think","signature":"sig"}]}}`,
		wantType: MessageTypeAssistant,
		check: func(t *testing.T, msg Message) {
			t.Helper()
			ast := msg.(*AssistantMessage)
			if len(ast.Content) != 1 {
				t.Fatalf("Content len = %d, want 1", len(ast.Content))
			}
			tb, ok := ast.Content[0].(*ThinkingBlock)
			if !ok {
				t.Fatalf("Content[0] type = %T, want *ThinkingBlock", ast.Content[0])
			}
			if tb.Thinking != "let me think" {
				t.Errorf("Thinking = %q, want %q", tb.Thinking, "let me think")
			}
		},
	},
	{
		name:     "assistant tool use block",
		input:    `{"type":"assistant","message":{"model":"claude-3","content":[{"type":"tool_use","id":"tool-1","name":"bash","input":{"command":"ls"}}]}}`,
		wantType: MessageTypeAssistant,
		check: func(t *testing.T, msg Message) {
			t.Helper()
			ast := msg.(*AssistantMessage)
			if len(ast.Content) != 1 {
				t.Fatalf("Content len = %d, want 1", len(ast.Content))
			}
			tb, ok := ast.Content[0].(*ToolUseBlock)
			if !ok {
				t.Fatalf("Content[0] type = %T, want *ToolUseBlock", ast.Content[0])
			}
			if tb.Name != "bash" {
				t.Errorf("Name = %q, want %q", tb.Name, "bash")
			}
			if tb.Input["command"] != "ls" {
				t.Errorf("Input[command] = %v, want %q", tb.Input["command"], "ls")
			}
		},
	},
	{
		name:     "assistant tool result block",
		input:    `{"type":"assistant","message":{"model":"claude-3","content":[{"type":"tool_result","tool_use_id":"tool-1","content":"output","is_error":false}]}}`,
		wantType: MessageTypeAssistant,
		check: func(t *testing.T, msg Message) {
			t.Helper()
			ast := msg.(*AssistantMessage)
			if len(ast.Content) != 1 {
				t.Fatalf("Content len = %d, want 1", len(ast.Content))
			}
			tb, ok := ast.Content[0].(*ToolResultBlock)
			if !ok {
				t.Fatalf("Content[0] type = %T, want *ToolResultBlock", ast.Content[0])
			}
			if tb.ToolUseID != "tool-1" {
				t.Errorf("ToolUseID = %q, want %q", tb.ToolUseID, "tool-1")
			}
			if tb.IsError {
				t.Error("IsError = true, want false")
			}
		},
	},
	{
		name:     "result success",
		input:    `{"type":"result","subtype":"success","session_id":"abc-123","is_error":false,"result":"done","duration_ms":1000,"duration_api_ms":500,"num_turns":2}`,
		wantType: MessageTypeResult,
		check: func(t *testing.T, msg Message) {
			t.Helper()
			res := msg.(*ResultMessage)
			if res.Subtype != "success" {
				t.Errorf("Subtype = %q, want %q", res.Subtype, "success")
			}
			if res.SessionID != "abc-123" {
				t.Errorf("SessionID = %q, want %q", res.SessionID, "abc-123")
			}
			if res.IsError {
				t.Error("IsError = true, want false")
			}
			if res.Result != "done" {
				t.Errorf("Result = %q, want %q", res.Result, "done")
			}
			if res.DurationMs != 1000 {
				t.Errorf("DurationMs = %d, want 1000", res.DurationMs)
			}
			if res.NumTurns != 2 {
				t.Errorf("NumTurns = %d, want 2", res.NumTurns)
			}
		},
	},
	{
		name:     "result error",
		input:    `{"type":"result","subtype":"error_during_execution","session_id":"abc-123","is_error":true}`,
		wantType: MessageTypeResult,
		check: func(t *testing.T, msg Message) {
			t.Helper()
			res := msg.(*ResultMessage)
			if !res.IsError {
				t.Error("IsError = false, want true")
			}
			if res.Subtype != "error_during_execution" {
				t.Errorf("Subtype = %q, want %q", res.Subtype, "error_during_execution")
			}
		},
	},
	{
		name:     "user message",
		input:    `{"type":"user","message":{"role":"user","content":"hello"}}`,
		wantType: MessageTypeUser,
	},
	{
		name:     "unknown type wrapped as system",
		input:    `{"type":"status","status":"compacting"}`,
		wantType: MessageTypeSystem,
		check: func(t *testing.T, msg Message) {
			t.Helper()
			sys := msg.(*SystemMessage)
			if sys.Subtype != "unknown" {
				t.Errorf("Subtype = %q, want %q", sys.Subtype, "unknown")
			}
		},
	},
	{
		name:    "invalid json",
		input:   `{not valid json}`,
		wantErr: true,
	},
	{
		name:     "empty json object missing type",
		input:    `{}`,
		wantType: MessageTypeSystem, // falls through to unknown handler
	},
}

func TestParseMessage(t *testing.T) {
	for _, tt := range parseMessageTests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseMessage([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Error("ParseMessage() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMessage() error = %v", err)
			}
			if msg.Type() != tt.wantType {
				t.Errorf("Type() = %q, want %q", msg.Type(), tt.wantType)
			}
			if tt.check != nil {
				tt.check(t, msg)
			}
		})
	}
}

func TestParserNext(t *testing.T) {
	input := strings.Join([]string{
		`{"type":"system","subtype":"init","session_id":"xyz"}`,
		``,
		`{"type":"assistant","message":{"model":"claude-3","content":[{"type":"text","text":"Hi"}]}}`,
		`{"type":"result","subtype":"success","session_id":"xyz","is_error":false}`,
	}, "\n")

	p := NewParser(strings.NewReader(input))

	msg1, err := p.Next()
	if err != nil {
		t.Fatalf("Next() 1 error = %v", err)
	}
	if msg1.Type() != MessageTypeSystem {
		t.Errorf("msg1 Type = %q, want %q", msg1.Type(), MessageTypeSystem)
	}

	msg2, err := p.Next()
	if err != nil {
		t.Fatalf("Next() 2 error = %v", err)
	}
	if msg2.Type() != MessageTypeAssistant {
		t.Errorf("msg2 Type = %q, want %q", msg2.Type(), MessageTypeAssistant)
	}

	msg3, err := p.Next()
	if err != nil {
		t.Fatalf("Next() 3 error = %v", err)
	}
	if msg3.Type() != MessageTypeResult {
		t.Errorf("msg3 Type = %q, want %q", msg3.Type(), MessageTypeResult)
	}

	_, err = p.Next()
	if err != io.EOF {
		t.Errorf("Next() 4 error = %v, want io.EOF", err)
	}
}

func TestParserSkipsEmptyLines(t *testing.T) {
	input := "\n\n" + `{"type":"result","subtype":"success","session_id":"s","is_error":false}` + "\n\n"
	p := NewParser(strings.NewReader(input))

	msg, err := p.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if msg.Type() != MessageTypeResult {
		t.Errorf("Type = %q, want %q", msg.Type(), MessageTypeResult)
	}
}

func TestJSONDecodeErrorMessage(t *testing.T) {
	_, err := ParseMessage([]byte(`{bad}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	p := NewParser(strings.NewReader(`{bad}`))
	_, parserErr := p.Next()
	if parserErr == nil {
		t.Fatal("expected parser error, got nil")
	}
	jde, ok := parserErr.(*JSONDecodeError)
	if !ok {
		t.Fatalf("error type = %T, want *JSONDecodeError", parserErr)
	}
	if jde.Unwrap() == nil {
		t.Error("Unwrap() = nil, want non-nil cause")
	}
}
