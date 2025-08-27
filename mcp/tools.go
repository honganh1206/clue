package mcp

import (
	"github.com/invopop/jsonschema"
)

// ToolResultContent defines the structure for content returned by a tool call.
type ToolResultContent struct {
	Type     string `json:"type"`               // "text" or "image"
	Text     string `json:"text,omitempty"`     // non-empty when type == "text"
	Data     string `json:"data,omitempty"`     // non-empty base64 encoded data when type == "image"
	MimeType string `json:"mimeType,omitempty"` // non-empty mime type for type == "image"
}

// Tool defines the structure for a tool's metadata.
type Tool struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	InputSchema *jsonschema.Schema `json:"inputSchema"`
}

// Tools is a collection of Tool.
type Tools []Tool

// Defines the parameters for the "tools/call" request.
type ToolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// Defines the result for the "tools/call" response.
type ToolsCallResult struct {
	Content []ToolResultContent `json:"content"`
	IsError bool                `json:"isError"`
}

// Defines the parameters for the "tools/list" request.
type ToolsListParams struct {
	// Used for pagination when listing tools.
	// If Cursor is empty, we are requesting the first page
	Cursor string `json:"cursor,omitempty"`
}

// Defines the result for the "tools/list" response.
type ToolsListResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// ByName finds a tool by its name from a list of tools.
func (t Tools) ByName(name string) (Tool, bool) {
	for _, tool := range t {
		if tool.Name == name {
			return tool, true
		}
	}
	return Tool{}, false
}

type ToolDetails struct {
	Server *Server
	Name   string
}
