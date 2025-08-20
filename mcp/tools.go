package mcp

import "encoding/json"

// ToolResultContent defines the structure for content returned by a tool call.
type MCPToolResultContent struct {
	Type     string `json:"type"`               // "text" or "image"
	Text     string `json:"text,omitempty"`     // non-empty when type == "text"
	Data     string `json:"data,omitempty"`     // non-empty base64 encoded data when type == "image"
	MimeType string `json:"mimeType,omitempty"` // non-empty mime type for type == "image"
}

// Tool defines the structure for a tool's metadata.
type MCPTool struct {
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	RawInputSchema json.RawMessage `json:"inputSchema"` // raw json bytes of the input schema
}

// Tools is a collection of Tool.
type MCPTools []MCPTool

// Defines the parameters for the "tools/call" request.
type MCPToolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// Defines the result for the "tools/call" response.
type MCPToolsCallResult struct {
	Content []MCPToolResultContent `json:"content"`
	IsError bool                   `json:"isError"`
}

// Defines the parameters for the "tools/list" request.
type MCPToolsListParams struct {
	// Used for pagination when listing tools.
	// If Cursor is empty, we are requesting the first page
	Cursor string `json:"cursor,omitempty"`
}

// Defines the result for the "tools/list" response.
type MCPToolsListResult struct {
	Tools      []MCPTool `json:"tools"`
	NextCursor string    `json:"nextCursor,omitempty"`
}

// ByName finds a tool by its name from a list of tools.
func (t MCPTools) ByName(name string) (MCPTool, bool) {
	for _, tool := range t {
		if tool.Name == name {
			return tool, true
		}
	}
	return MCPTool{}, false
}
