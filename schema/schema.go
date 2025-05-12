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
