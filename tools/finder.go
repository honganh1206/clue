package tools

import (
	_ "embed"

	"github.com/honganh1206/clue/schema"
)

//go:embed finder.md
var finderPrompt string

var FinderDefinition = ToolDefinition{
	Name:        ToolNameFinder,
	Description: finderPrompt,
	InputSchema: FinderInputSchema,
	IsSubTool:   true,
}

type FinderInput struct {
	Query string `json:"query" jsonschema_description:"The search query describing what you're looking for in the codebase. Be specific and include context."`
}

var FinderInputSchema = schema.Generate[FinderInput]()
