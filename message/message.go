package message

import (
	"encoding/json"
	"time"
)

// TODO: Rename this struct to Payload?
// and create Message struct in conversation package?
type Message struct {
	Role string `json:"role"`
	// Cannot unmarshal interface as not concrete type, so we use tagged union
	Content []ContentBlockUnion `json:"content"`
	// Optional as metadata
	ID        string    `json:"id,omitempty" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Sequence  int       `json:"sequence,omitempty" db:"sequence_number"`
}

const (
	UserRole      = "user"
	AssistantRole = "assistant"
	// Gemini uses model instead of assistant
	// TODO: 2-way conversion from and to assistant
	ModelRole = "model"
)

const (
	TextType       = "text"
	ToolUseType    = "tool_use"
	ToolResultType = "tool_result"
)

type TextContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func NewTextContentBlock(text string) ContentBlockUnion {
	return ContentBlockUnion{
		Type: TextType,
		OfTextBlock: &TextContentBlock{
			Text: text,
		}}
}

type ToolUseContentBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

func NewToolUseContentBlock(id, name string, input json.RawMessage) ContentBlockUnion {
	return ContentBlockUnion{
		Type: ToolUseType,
		OfToolUseBlock: &ToolUseContentBlock{
			ID:    id,
			Name:  name,
			Input: input,
		}}
}

type ToolResultContentBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	ToolName  string `json:"tool_name"`
	Content   any    `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

func NewToolResultContentBlock(toolUseID, toolName string, content any, isError bool) ContentBlockUnion {
	return ContentBlockUnion{
		Type: ToolResultType,
		OfToolResultBlock: &ToolResultContentBlock{
			ToolUseID: toolUseID,
			ToolName:  toolName,
			Content:   content,
			IsError:   isError,
		}}

}

// Tagged union taking on different fixed types
type ContentBlockUnion struct {
	Type string `json:"type"`
	// Tag field ensures the correct variant is selected at runtime
	// Only one can be used at a time, and a tag indicates which type is used
	OfTextBlock       *TextContentBlock       `json:"-"`
	OfToolUseBlock    *ToolUseContentBlock    `json:"-"`
	OfToolResultBlock *ToolResultContentBlock `json:"-"`
}

// TODO: Research UnmarshalRoot of anthropic go sdk
// There should be a neater way of doing this?
func (c *ContentBlockUnion) MarshalJSON() ([]byte, error) {
	switch c.Type {
	case TextType:
		if c.OfTextBlock != nil {
			return json.Marshal(struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				Type: TextType,
				Text: c.OfTextBlock.Text,
			})
		}
	case ToolUseType:
		if c.OfToolUseBlock != nil {
			return json.Marshal(struct {
				Type  string          `json:"type"`
				ID    string          `json:"id"`
				Name  string          `json:"name"`
				Input json.RawMessage `json:"input"`
			}{
				Type:  ToolUseType,
				ID:    c.OfToolUseBlock.ID,
				Name:  c.OfToolUseBlock.Name,
				Input: c.OfToolUseBlock.Input,
			})
		}
	case ToolResultType:
		if c.OfToolResultBlock != nil {
			return json.Marshal(struct {
				Type      string `json:"type"`
				ToolUseID string `json:"tool_use_id"`
				Content   any    `json:"content"`
				IsError   bool   `json:"is_error,omitempty"`
			}{
				Type:      ToolResultType,
				ToolUseID: c.OfToolResultBlock.ToolUseID,
				Content:   c.OfToolResultBlock.Content,
				IsError:   c.OfToolResultBlock.IsError,
			})
		}
	}
	return json.Marshal(struct {
		Type string `json:"type"`
	}{Type: c.Type})
}

func (c *ContentBlockUnion) UnmarshalJSON(data []byte) error {
	var typeOnly struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeOnly); err != nil {
		return err
	}

	c.Type = typeOnly.Type

	switch c.Type {
	case TextType:
		var textBlock struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(data, &textBlock); err != nil {
			return err
		}
		c.OfTextBlock = &TextContentBlock{
			Type: textBlock.Type,
			Text: textBlock.Text,
		}
	case ToolUseType:
		var toolUseBlock struct {
			Type  string          `json:"type"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		}
		if err := json.Unmarshal(data, &toolUseBlock); err != nil {
			return err
		}
		c.OfToolUseBlock = &ToolUseContentBlock{
			Type:  toolUseBlock.Type,
			ID:    toolUseBlock.ID,
			Name:  toolUseBlock.Name,
			Input: toolUseBlock.Input,
		}
	case ToolResultType:
		var toolResultBlock struct {
			Type      string `json:"type"`
			ToolUseID string `json:"tool_use_id"`
			Content   any    `json:"content"`
			IsError   bool   `json:"is_error"`
		}
		if err := json.Unmarshal(data, &toolResultBlock); err != nil {
			return err
		}
		c.OfToolResultBlock = &ToolResultContentBlock{
			Type:      toolResultBlock.Type,
			ToolUseID: toolResultBlock.ToolUseID,
			Content:   toolResultBlock.Content,
			IsError:   toolResultBlock.IsError,
		}
	}

	return nil
}
