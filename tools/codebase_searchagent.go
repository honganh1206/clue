package tools

import (
	_ "embed"

	"github.com/honganh1206/clue/schema"
)

//go:embed codebase_searchagent.md
var codebaseSearchPrompt string

var CodebaseSearchDefinition = ToolDefinition{
	Name:        "codebase_searchagent",
	Description: codebaseSearchPrompt,
	InputSchema: CodebaseSearchInputSchema,
	IsSubTool:   true,
}

type CodebaseSearchInput struct {
	Query string `json:"query" jsonschema_description:"The search query describing what you're looking for in the codebase. Be specific and include context."`
}

var CodebaseSearchInputSchema = schema.Generate[CodebaseSearchInput]()
