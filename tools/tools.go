package tools

import (
	"encoding/json"

	"github.com/honganh1206/tinker/server/data"
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
	Function    func(input ToolInput) (string, error)
	IsSubTool   bool `json:"-"`
}

type ToolObject struct {
	Plan *data.Plan
}

type ToolInput struct {
	RawInput json.RawMessage
	*ToolObject
}
