package message

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// TODO: Rename this struct to Payload?
// and create Message struct in conversation package?
type Message struct {
	Role string `json:"role"`
	// Cannot unmarshal interface as not concrete type, so we use tagged union
	Content []ContentBlock `json:"content"`
	// Optional as metadata
	ID        string    `json:"id,omitempty" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Sequence  int       `json:"-" db:"sequence_number"`
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

type ContentBlock interface {
	Type() string
	ToAnthropic() anthropic.ContentBlockParamUnion
}

type TextBlock struct {
	Text string `json:"text"`
}

func (t TextBlock) Type() string { return "text" }
func (t TextBlock) ToAnthropic() anthropic.ContentBlockParamUnion {
	return anthropic.NewTextBlock(t.Text)
}

func NewTextBlock(text string) ContentBlock {
	return TextBlock{
		Text: text,
	}
}

type ToolUseBlock struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

func (t ToolUseBlock) Type() string { return "tool_use" }
func (t ToolUseBlock) ToAnthropic() anthropic.ContentBlockParamUnion {
	return anthropic.NewToolUseBlock(t.ID, t.Input, t.Name)
}

func NewToolUseBlock(id, name string, input json.RawMessage) ContentBlock {
	return ToolUseBlock{
		ID:    id,
		Name:  name,
		Input: input,
	}
}

type ToolResultBlock struct {
	ToolUseID string `json:"tool_use_id"`
	ToolName  string `json:"tool_name"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

func (t ToolResultBlock) Type() string { return "tool_result" }
func (t ToolResultBlock) ToAnthropic() anthropic.ContentBlockParamUnion {
	return anthropic.NewToolResultBlock(t.ToolUseID, t.Content, t.IsError)
}

func NewToolResultBlock(toolUseID, toolName, content string, isError bool) ContentBlock {
	return ToolResultBlock{
		ToolUseID: toolUseID,
		ToolName:  toolName,
		Content:   content,
		IsError:   isError,
	}
}

// Custom JSON marshaling for Message to handle ContentBlock interface
func (m *Message) MarshalJSON() ([]byte, error) {
	type MessageAlias Message
	type contentWithType struct {
		Type      string          `json:"type"`
		Text      string          `json:"text,omitempty"`
		ID        string          `json:"id,omitempty"`
		Name      string          `json:"name,omitempty"`
		Input     json.RawMessage `json:"input,omitempty"`
		ToolUseID string          `json:"tool_use_id,omitempty"`
		ToolName  string          `json:"tool_name,omitempty"`
		Content   string          `json:"content,omitempty"`
		IsError   bool            `json:"is_error,omitempty"`
	}

	temp := struct {
		*MessageAlias
		Content []contentWithType `json:"content"`
	}{
		MessageAlias: (*MessageAlias)(m),
		Content:      make([]contentWithType, len(m.Content)),
	}

	for i, block := range m.Content {
		switch b := block.(type) {
		case TextBlock:
			temp.Content[i] = contentWithType{Type: TextType, Text: b.Text}
		case ToolUseBlock:
			temp.Content[i] = contentWithType{Type: ToolUseType, ID: b.ID, Name: b.Name, Input: b.Input}
		case ToolResultBlock:
			temp.Content[i] = contentWithType{Type: ToolResultType, ToolUseID: b.ToolUseID, ToolName: b.ToolName, Content: b.Content, IsError: b.IsError}
		default:
			return nil, fmt.Errorf("unknown content block type: %T", block)
		}
	}

	return json.Marshal(temp)
}

// Custom JSON unmarshaling for Message to handle ContentBlock interface
func (m *Message) UnmarshalJSON(data []byte) error {
	type MessageAlias Message
	type contentWithType struct {
		Type      string          `json:"type"`
		Text      string          `json:"text,omitempty"`
		ID        string          `json:"id,omitempty"`
		Name      string          `json:"name,omitempty"`
		Input     json.RawMessage `json:"input,omitempty"`
		ToolUseID string          `json:"tool_use_id,omitempty"`
		ToolName  string          `json:"tool_name,omitempty"`
		Content   string          `json:"content,omitempty"`
		IsError   bool            `json:"is_error,omitempty"`
	}

	temp := struct {
		*MessageAlias
		Content []contentWithType `json:"content"`
	}{
		MessageAlias: (*MessageAlias)(m),
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	m.Content = make([]ContentBlock, len(temp.Content))
	for i, c := range temp.Content {
		switch c.Type {
		case TextType:
			m.Content[i] = TextBlock{Text: c.Text}
		case ToolUseType:
			m.Content[i] = ToolUseBlock{ID: c.ID, Name: c.Name, Input: c.Input}
		case ToolResultType:
			m.Content[i] = ToolResultBlock{ToolUseID: c.ToolUseID, ToolName: c.ToolName, Content: c.Content, IsError: c.IsError}
		default:
			return fmt.Errorf("unknown content block type: %s", c.Type)
		}
	}

	return nil
}
