package schema

import (
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/invopop/jsonschema" // Generate JSON schema from Go types
)

// Translate Go structs into JSON schema during runtime
// Thus producing a standard format usable outside Go
func GenerateSchema[T any]() anthropic.ToolInputSchemaParam {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	var v T

	schema := reflector.Reflect(v)

	return anthropic.ToolInputSchemaParam{
		Properties: schema.Properties,
	}
}
