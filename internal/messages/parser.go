package messages

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// DefaultMaxBufferSize is the maximum size for a single JSON line (1MB).
const DefaultMaxBufferSize = 1024 * 1024

// JSONDecodeError is returned when a line cannot be parsed as a message.
type JSONDecodeError struct {
	Line  string
	Cause error
}

func (e *JSONDecodeError) Error() string {
	line := e.Line
	if len(line) > 100 {
		line = line[:100] + "..."
	}
	return fmt.Sprintf("failed to decode JSON message: %s: %v", line, e.Cause)
}

func (e *JSONDecodeError) Unwrap() error { return e.Cause }

// Parser reads newline-delimited JSON from an io.Reader and parses messages.
type Parser struct {
	scanner *bufio.Scanner
}

// NewParser creates a Parser that reads stream-JSON from r.
func NewParser(r io.Reader) *Parser {
	scanner := bufio.NewScanner(r)
	// Start with default 4KB buffer, grow automatically up to 1MB max.
	// More memory-efficient than pre-allocating 256KB for every parser.
	scanner.Buffer(make([]byte, 0, 4096), DefaultMaxBufferSize)
	return &Parser{scanner: scanner}
}

// Next returns the next message from the stream.
// It returns io.EOF when the stream is exhausted.
// Parse errors are returned as *JSONDecodeError; the caller can choose to skip them.
func (p *Parser) Next() (Message, error) {
	for p.scanner.Scan() {
		line := p.scanner.Text()
		if line == "" {
			continue
		}
		msg, err := ParseMessage([]byte(line))
		if err != nil {
			return nil, &JSONDecodeError{Line: line, Cause: err}
		}
		return msg, nil
	}
	if err := p.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// ParseMessage parses a single JSON line into a Message.
func ParseMessage(data []byte) (Message, error) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}

	switch MessageType(envelope.Type) {
	case MessageTypeSystem:
		return parseSystemMessage(data)
	case MessageTypeAssistant:
		return parseAssistantMessage(data)
	case MessageTypeResult:
		return parseResultMessage(data)
	case MessageTypeUser:
		return parseUserMessage(data)
	default:
		// Forward-compatible: unknown types are wrapped as SystemMessage with raw data.
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		return &SystemMessage{Subtype: "unknown", Data: raw}, nil
	}
}

func parseSystemMessage(data []byte) (*SystemMessage, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	subtype, _ := raw["subtype"].(string)
	return &SystemMessage{Subtype: subtype, Data: raw}, nil
}

func parseAssistantMessage(data []byte) (*AssistantMessage, error) {
	var outer struct {
		Message struct {
			Model   string            `json:"model"`
			Content []json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(data, &outer); err != nil {
		return nil, err
	}

	content := make([]ContentBlock, 0, len(outer.Message.Content))
	for _, raw := range outer.Message.Content {
		block, err := parseContentBlock(raw)
		if err != nil {
			return nil, err
		}
		content = append(content, block)
	}

	return &AssistantMessage{
		Model:   outer.Message.Model,
		Content: content,
	}, nil
}

func parseContentBlock(data json.RawMessage) (ContentBlock, error) {
	var typed struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typed); err != nil {
		return nil, err
	}

	switch ContentBlockType(typed.Type) {
	case ContentBlockText:
		var block TextBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		return &block, nil
	case ContentBlockThinking:
		var block ThinkingBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		return &block, nil
	case ContentBlockToolUse:
		var block ToolUseBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		return &block, nil
	case ContentBlockToolResult:
		var block ToolResultBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		return &block, nil
	default:
		// Unknown block type: store raw JSON as text for forward compatibility.
		return &TextBlock{Text: string(data)}, nil
	}
}

func parseResultMessage(data []byte) (*ResultMessage, error) {
	var raw struct {
		Subtype       string   `json:"subtype"`
		SessionID     string   `json:"session_id"`
		IsError       bool     `json:"is_error"`
		Result        string   `json:"result"`
		DurationMs    int      `json:"duration_ms"`
		DurationAPIMs int      `json:"duration_api_ms"`
		NumTurns      int      `json:"num_turns"`
		TotalCostUSD  *float64 `json:"total_cost_usd"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &ResultMessage{
		Subtype:       raw.Subtype,
		SessionID:     raw.SessionID,
		IsError:       raw.IsError,
		Result:        raw.Result,
		DurationMs:    raw.DurationMs,
		DurationAPIMs: raw.DurationAPIMs,
		NumTurns:      raw.NumTurns,
		TotalCostUSD:  raw.TotalCostUSD,
	}, nil
}

func parseUserMessage(data []byte) (*UserMessage, error) {
	var outer struct {
		Message struct {
			Content any `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(data, &outer); err != nil {
		return nil, err
	}
	return &UserMessage{Content: outer.Message.Content}, nil
}
