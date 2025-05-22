package messages

import (
	"encoding/json"
)

type ContentBlock interface {
	isContentBlock()
}

const (
	UserRole      = "user"
	AssistantRole = "assistant"
)

const (
	TextType       = "text"
	ToolUseType    = "tool_use"
	ToolResultType = "tool_result"
)

type MessageParam struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

type MessageRequest struct {
	MessageParam
}

type MessageResponse struct {
	MessageParam
	// Optional fields for tool responses
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Input    json.RawMessage `json:"input"`
	IsError  bool            `json:"is_error"`
	ToolCall bool            `json:"tool_call"`
	Model    string          `json:"model"`
}

type TextContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func NewTextContentBlock(text string) ContentBlock {
	return TextContentBlock{
		Type: TextType,
		Text: text,
	}
}

func (t TextContentBlock) isContentBlock() {}

type ToolUseContentBlock struct {
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Input    json.RawMessage `json:"input"`
	IsError  bool            `json:"is_error"`
	ToolCall bool            `json:"tool_call"`
}

func (t ToolUseContentBlock) isContentBlock() {}

func NewToolUseContentBlock(id, name string, input json.RawMessage) ContentBlock {
	return ToolUseContentBlock{
		Type:  ToolUseType,
		ID:    id,
		Name:  name,
		Input: input,
	}
}

type ToolResultContentBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   any    `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// NewToolResultContentBlock creates a new tool result content block with the given parameters.
func NewToolResultContentBlock(toolUseID string, content any, isError bool) ContentBlock {
	return ToolResultContentBlock{
		Type:      "tool_result",
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
	}
}

func (t ToolResultContentBlock) isContentBlock() {}
