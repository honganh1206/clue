package conversation

import (
	"encoding/json"
	"fmt"
	"time"
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
	Role    string             `json:"role"`
	Content []ContentBlockJSON `json:"content"`
	// Optional as metadata
	ID        string    `json:"id,omitempty" db:"id"`
	CreatedAt time.Time `json:"created_at,omitempty" db:"created_at"`
	Sequence  int       `json:"sequence,omitempty" db:"sequence_number"`
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

// ContentBlockJSON is a wrapper for JSON marshaling/unmarshaling of ContentBlock
type ContentBlockJSON struct {
	ContentBlock
}

func (cbj *ContentBlockJSON) UnmarshalJSON(data []byte) error {
	// First unmarshal to determine the type
	var temp struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Create the appropriate concrete type based on the type field
	switch temp.Type {
	case TextType:
		var textBlock TextContentBlock
		if err := json.Unmarshal(data, &textBlock); err != nil {
			return err
		}
		cbj.ContentBlock = textBlock
	case ToolUseType:
		var toolUseBlock ToolUseContentBlock
		if err := json.Unmarshal(data, &toolUseBlock); err != nil {
			return err
		}
		cbj.ContentBlock = toolUseBlock
	case ToolResultType:
		var toolResultBlock ToolResultContentBlock
		if err := json.Unmarshal(data, &toolResultBlock); err != nil {
			return err
		}
		cbj.ContentBlock = toolResultBlock
	default:
		return fmt.Errorf("unknown content block type: %s", temp.Type)
	}

	return nil
}

func (cbj ContentBlockJSON) MarshalJSON() ([]byte, error) {
	// Marshal the concrete type directly
	return json.Marshal(cbj.ContentBlock)
}

// Helper functions to convert between ContentBlock and ContentBlockJSON
func ToContentBlockJSON(cb ContentBlock) ContentBlockJSON {
	return ContentBlockJSON{ContentBlock: cb}
}

func ToContentBlockJSONSlice(cbs []ContentBlock) []ContentBlockJSON {
	result := make([]ContentBlockJSON, len(cbs))
	for i, cb := range cbs {
		result[i] = ToContentBlockJSON(cb)
	}
	return result
}

func FromContentBlockJSON(cbj ContentBlockJSON) ContentBlock {
	return cbj.ContentBlock
}

func FromContentBlockJSONSlice(cbjs []ContentBlockJSON) []ContentBlock {
	result := make([]ContentBlock, len(cbjs))
	for i, cbj := range cbjs {
		result[i] = FromContentBlockJSON(cbj)
	}
	return result
}
