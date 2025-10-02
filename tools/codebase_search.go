package tools

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/honganh1206/clue/schema"
)

//go:embed codebase_search.md
var codebaseSearchPrompt string

var CodebaseSearchDefinition = ToolDefinition{
	Name:        "codebase_search",
	Description: codebaseSearchPrompt,
	InputSchema: CodebaseSearchInputSchema,
	Function:    CodebaseSearch,
}

type CodebaseSearchInput struct {
	Query string `json:"query" jsonschema_description:"The search query describing what you're looking for in the codebase. Be specific and include context."`
}

var CodebaseSearchInputSchema = schema.Generate[CodebaseSearchInput]()

// SubAgentRunner is a function type that can run a sub-agent
// This will be injected from the agent package to avoid import cycles
var SubAgentRunner func(systemPrompt, userQuery string) (string, error)

func CodebaseSearch(input json.RawMessage) (string, error) {
	if SubAgentRunner == nil {
		return "", fmt.Errorf("codebase_search not initialized: SubAgentRunner is nil")
	}

	var searchInput CodebaseSearchInput
	err := json.Unmarshal(input, &searchInput)
	if err != nil {
		return "", err
	}

	if searchInput.Query == "" {
		return "", fmt.Errorf("query parameter is required")
	}

	// Delegate to the sub-agent runner (injected by agent package)
	result, err := SubAgentRunner(codebaseSearchPrompt, searchInput.Query)
	if err != nil {
		return "", fmt.Errorf("sub-agent search failed: %w", err)
	}

	return result, nil
}
