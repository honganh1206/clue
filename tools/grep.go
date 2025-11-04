package tools

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/honganh1206/clue/schema"
)

//go:embed grep.md
var grepSearchPrompt string

var GrepSearchDefinition = ToolDefinition{
	Name:        "grep_search",
	Description: grepSearchPrompt,
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
