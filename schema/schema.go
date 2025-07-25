package schema

import (
	"github.com/invopop/jsonschema" // Generate JSON schema from Go types
)

// Translate Go structs into JSON schema during runtime
// Thus producing a standard format usable outside Go
func Generate[T any]() *jsonschema.Schema {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	var v T

	rawSchema := reflector.Reflect(v)

	return rawSchema
}

// Ensure compatibility with genai.Schema's custom UnmarshalJSON method
// func DeserializeToolSchema(jsonBytes []byte) (*genai.Schema, error) {
// 	if len(jsonBytes) == 0 || string(jsonBytes) == "null" {
// 		return &genai.Schema{Type: genai.TypeObject, Properties: map[string]*genai.Schema{}}, nil
// 	}

// }

// func convertNumericString
