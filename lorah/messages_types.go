package lorah

// MessageType identifies the kind of message emitted by the CLI.
type MessageType string

const (
	MessageTypeSystem    MessageType = "system"
	MessageTypeAssistant MessageType = "assistant"
	MessageTypeResult    MessageType = "result"
	MessageTypeUser      MessageType = "user"
)

// Message is implemented by all stream-JSON message types.
type Message interface {
	Type() MessageType
}

// ContentBlockType identifies the kind of content block in an assistant message.
type ContentBlockType string

const (
	ContentBlockText       ContentBlockType = "text"
	ContentBlockThinking   ContentBlockType = "thinking"
	ContentBlockToolUse    ContentBlockType = "tool_use"
	ContentBlockToolResult ContentBlockType = "tool_result"
)

// ContentBlock is implemented by all content block types.
type ContentBlock interface {
	BlockType() ContentBlockType
}

// TextBlock contains plain text content.
type TextBlock struct {
	Text string `json:"text"`
}

func (b *TextBlock) BlockType() ContentBlockType { return ContentBlockText }

// ThinkingBlock contains extended thinking content.
type ThinkingBlock struct {
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

func (b *ThinkingBlock) BlockType() ContentBlockType { return ContentBlockThinking }

// ToolUseBlock represents a tool invocation by the assistant.
type ToolUseBlock struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

func (b *ToolUseBlock) BlockType() ContentBlockType { return ContentBlockToolUse }

// ToolResultBlock contains the result of a tool invocation.
type ToolResultBlock struct {
	ToolUseID string `json:"tool_use_id"`
	Content   any    `json:"content"`
	IsError   bool   `json:"is_error"`
}

func (b *ToolResultBlock) BlockType() ContentBlockType { return ContentBlockToolResult }

// SystemMessage is sent once at the start of a session with subtype "init".
// The init message carries the session ID and other session metadata.
type SystemMessage struct {
	Subtype string
	// Data holds the full parsed message for access to any field.
	Data map[string]any
}

func (m *SystemMessage) Type() MessageType { return MessageTypeSystem }

// SessionID returns the session_id from an init system message.
func (m *SystemMessage) SessionID() string {
	sid, _ := m.Data["session_id"].(string)
	return sid
}

// AssistantMessage contains the assistant's response with parsed content blocks.
type AssistantMessage struct {
	Model   string
	Content []ContentBlock
}

func (m *AssistantMessage) Type() MessageType { return MessageTypeAssistant }

// ResultMessage is the final message sent after the CLI finishes processing.
type ResultMessage struct {
	Subtype       string
	SessionID     string
	IsError       bool
	Result        string
	DurationMs    int
	DurationAPIMs int
	NumTurns      int
	TotalCostUSD  *float64
}

func (m *ResultMessage) Type() MessageType { return MessageTypeResult }

// UserMessage is emitted when the CLI echoes back user input (for logging/replay).
type UserMessage struct {
	Content any
}

func (m *UserMessage) Type() MessageType { return MessageTypeUser }
