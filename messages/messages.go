package messages

import "encoding/json"

// TODO: use json annotation `json:"id"`
type MessageParam struct {
	Role    string
	Content []ContentBlock
}

type MessageRequest struct {
	MessageParam
}

type MessageResponse struct {
	MessageParam
	// Optional fields for tool responses
	ID       string
	Name     string
	Input    json.RawMessage
	IsError  bool
	ToolCall bool
	Model    string // Which LLM was used
}

// Block of content within a message
type ContentBlock struct {
	Type     string // Different categories like text, code, tools
	Text     string
	ID       string
	Name     string
	Input    json.RawMessage
	IsError  bool
	ToolCall bool
}

const (
	UserRole      = "user"
	AssistantRole = "assistant"
)
