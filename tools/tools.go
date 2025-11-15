package tools

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

const (
	ToolNameBash                = "bash"
	ToolNameReadFile            = "read_file"
	ToolNameEditFile            = "edit_file"
	ToolNameGrepSearch          = "grep_search"
	ToolNameListFiles           = "list_files"
	ToolNamePlanRead            = "plan_read"
	ToolNamePlanWrite           = "plan_write"
	ToolNameCodebaseSearchAgent = "codebase_search_agent"
)

type ToolBox struct {
	Tools []*ToolDefinition
}

type ToolDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	InputSchema *jsonschema.Schema `json:"input_schema"`
	Function    func(input json.RawMessage, meta ToolMetadata) (string, error)
	IsSubTool   bool `json:"-"`
}

type ToolMetadata struct {
	ConversationID string `json:"conversation_id"`
}

