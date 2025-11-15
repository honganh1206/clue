package tools

import (
	_ "embed"

	"github.com/honganh1206/clue/schema"
)

//go:embed codebase_search_agent.md
var codebaseSearchAgentPrompt string

var CodebaseSearchAgentDefinition = ToolDefinition{
	Name:        ToolNameCodebaseSearchAgent,
	Description: codebaseSearchAgentPrompt,
	InputSchema: CodebaseSearchAgentInputSchema,
	IsSubTool:   true,
}

type CodebaseSearchAgentInput struct {
	Query string `json:"query" jsonschema_description:"The search query describing what you're looking for in the codebase. Be specific and include context."`
}

var CodebaseSearchAgentInputSchema = schema.Generate[CodebaseSearchAgentInput]()
