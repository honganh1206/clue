package tools

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

type ToolBox struct {
	Tools []*ToolDefinition
}

type ToolDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	InputSchema *jsonschema.Schema `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error)
	IsSubTool   bool `json:"-"`
}

type ToolInput struct {
	Path string `json:"path"`
}
