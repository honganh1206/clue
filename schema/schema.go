package schema

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema" // Generate JSON schema from Go types
	"google.golang.org/genai"
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

// TODO: Use when migrating from jsonschema to json.RawMessage?
func ConvertStructToJSONRawMessage[T any]() json.RawMessage {
	var v T
	b, err := json.Marshal(v)

	if err != nil {
		return nil
	}

	return json.RawMessage(b)
}

func ConvertToGeminiSchema(inputSchema any) (*genai.Schema, error) {
	schemaBytes, err := json.Marshal(inputSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input schema: %w", err)
	}

	var rawSchema map[string]any
	if err := json.Unmarshal(schemaBytes, &rawSchema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to generic map: %w", err)
	}

	return buildGeminiSchema(rawSchema), nil
}

// Convert generic schema to gemini schema recursively
func buildGeminiSchema(schema map[string]any) *genai.Schema {
	result := &genai.Schema{}

	// Set schema type
	if schemaType, ok := schema["type"].(string); ok {
		switch schemaType {
		case "object":
			result.Type = genai.TypeObject
		case "string":
			result.Type = genai.TypeString
		case "integer":
			result.Type = genai.TypeInteger
		case "number":
			result.Type = genai.TypeNumber
		case "boolean":
			result.Type = genai.TypeBoolean
		case "array":
			result.Type = genai.TypeArray
		default:
			result.Type = genai.TypeString // fallback
		}
	}

	if desc, ok := schema["description"].(string); ok {
		result.Description = desc
	}

	if props, ok := schema["properties"].(map[string]any); ok {
		result.Properties = make(map[string]*genai.Schema)
		for name, prop := range props {
			if propMap, ok := prop.(map[string]any); ok {
				result.Properties[name] = buildGeminiSchema(propMap)
			}
		}
	}

	if items, ok := schema["items"].(map[string]any); ok {
		result.Items = buildGeminiSchema(items)
	}

	if required, ok := schema["required"].([]any); ok {
		result.Required = make([]string, 0, len(required))
		for _, req := range required {
			if reqStr, ok := req.(string); ok {
				result.Required = append(result.Required, reqStr)
			}
		}
	}

	if enumVals, ok := schema["enum"].([]any); ok {
		result.Enum = make([]string, 0, len(enumVals))
		for _, val := range enumVals {
			if valStr, ok := val.(string); ok {
				result.Enum = append(result.Enum, valStr)
			}
		}
	}

	// Skip problematic fields like exclusiveMaximum, exclusiveMinimum, default,
	// minLength, maxLength, format, $schema, $id, title, etc.
	// These are not supported by genai.Schema or cause marshaling issues

	return result
}
