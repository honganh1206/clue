package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/honganh1206/clue/schema"
)

var GrepSearchDefinition = ToolDefinition{
	Name: "grep_search",
	Description: `Search for exact text patterns in files using ripgrep, a fast keyword search tool.

WHEN TO USE THIS TOOL:
- When you need to find exact text matches like variable names, function calls, or specific strings
- When you know the precise pattern you're looking for (including regex patterns)
- When you want to quickly locate all occurrences of a specific term across multiple files
- When you need to search for code patterns with exact syntax

WHEN NOT TO USE THIS TOOL:
- For semantic or conceptual searches (e.g., "how does authentication work")
- For finding code that implements a certain functionality without knowing the exact terms
- When you already have read the entire file`,
	InputSchema: GrepSearchInputSchema,
	Function:    GrepSearch,
}

type GrepSearchInput struct {
	Pattern   string `json:"pattern" jsonschema_description:"The regexp pattern to search for."`
	Directory string `json:"directory,omitempty" jsonschema_description:"Optional directory to scope the search."`
}

var GrepSearchInputSchema = schema.Generate[GrepSearchInput]()

func GrepSearch(input json.RawMessage) (string, error) {
	searchInput := GrepSearchInput{}
	err := json.Unmarshal(input, &searchInput)
	if err != nil {
		return "", err
	}

	if searchInput.Pattern == "" {
		return "", fmt.Errorf("invalid pattern parameter")
	}

	args := []string{"rg", "--json", searchInput.Pattern}

	if searchInput.Directory != "" {
		args = append(args, searchInput.Directory)
	}

	cmd := exec.Command(args[0], args[1:]...)
	if output, err := cmd.CombinedOutput(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok && exitErr.ExitCode() == 1 {
			// Empty result
			return "[]", nil
		}
		return "", fmt.Errorf("failed to run command '%s': %w (output: %s)", strings.Join(args, " "), err, output)
	} else {
		outputStr := strings.TrimSpace(string(output))
		lines := strings.Split(outputStr, "\n")
		arr := "[" + strings.Join(lines, ",") + "]"

		return arr, nil
	}

}
