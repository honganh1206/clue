package tools

import (
	"encoding/json"

	"github.com/honganh1206/clue/server/api"
	"github.com/invopop/jsonschema"
)

const (
	ToolNameBash       = "bash"
	ToolNameReadFile   = "read_file"
	ToolNameEditFile   = "edit_file"
	ToolNameGrepSearch = "grep_search"
	ToolNameListFiles  = "list_files"
	ToolNamePlanRead   = "plan_read"
	ToolNamePlanWrite  = "plan_write"
	ToolNameFinder     = "finder"
)

type ToolBox struct {
	Tools []*ToolDefinition
}

type ToolDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	InputSchema *jsonschema.Schema `json:"input_schema"`
	Function    func(data *ToolData) (string, error)
	IsSubTool   bool `json:"-"`
}

type (
	ToolInput  json.RawMessage
	ToolOutput string
)

type ToolData struct {
	// Not the best design, but it's better than creating new instances of client
	// Can I have a Output field of type string and the Function of ToolDefinition should return it?
	*api.Client
	Input          json.RawMessage
	ConversationID string
}
