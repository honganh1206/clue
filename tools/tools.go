package tools

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

type ToolDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	InputSchema *jsonschema.Schema `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error)
}
