package messages

// Generic chat message
type Message struct {
	Role    string
	Content []ContentBlock
	Source  string // Which LLM was used
}

// Block of content within a message
type ContentBlock struct {
	Type     string // Different categories like text, code, tools
	Text     string
	ID       string
	Name     string
	Input    []byte
	IsError  bool
	ToolCall bool
}

const (
	TextType       = "text"
	ToolUseType    = "tool_use"
	ToolResultType = "tool_result"
)

const (
	UserRole      = "user"
	AssistantRole = "assistant"
)
