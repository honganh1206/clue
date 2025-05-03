package schema

import (
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/invopop/jsonschema" // Generate JSON schema from Go types
)

func GenerateAnthropicSchema[T any]() anthropic.ToolInputSchemaParam {
	schema := generateRawSchema[T]()
	// Define the shape of input that the tools accept
	return anthropic.ToolInputSchemaParam{
		Properties: schema.Properties,
	}
}

// Translate Go structs into JSON schema during runtime
// Thus producing a standard format usable outside Go
func generateRawSchema[T any]() *jsonschema.Schema {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	var v T

	rawSchema := reflector.Reflect(v)

	return rawSchema

}
