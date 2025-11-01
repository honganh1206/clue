package tools

import (
	"encoding/json"

	"github.com/honganh1206/clue/api"
	"github.com/invopop/jsonschema"
)

type ToolBox struct {
	Tools []*ToolDefinition
}

type ToolDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	InputSchema *jsonschema.Schema `json:"input_schema"`
	// TODO: Making the client the 2nd param feels wonky
	// but is there a better way?
	Function  func(input json.RawMessage, client *api.Client) (string, error)
	IsSubTool bool `json:"-"`
}

type ToolInput struct {
	Path string `json:"path"`
}
