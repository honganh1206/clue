// Package tools provides tool definitions for the Clue CLI agent system.
package tools

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/honganh1206/clue/schema"
)

//go:embed bash.md
var bashPrompt string

type BashInput struct {
	Command string `json:"command" jsonschema_description:"The bash command to execute."`
}

var BashInputSchema = schema.Generate[BashInput]()

var BashDefinition = ToolDefinition{
	Name:        "bash",
	Description: bashPrompt,
	InputSchema: BashInputSchema, // Machine-readable description of the tool's input
	Function:    Bash,
}

func Bash(input json.RawMessage) (string, error) {
	// Parse the JSON input into a BashInput struct
	bashInput := BashInput{}
	err := json.Unmarshal(input, &bashInput)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("bash", "-c", bashInput.Command)

	// TODO: Add a way to stop the execution.
	// Maybe an interactive bash interface?
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Command failed with error: %s\nOutput: %s", err.Error(), string(output)), nil
	}

	return strings.TrimSpace(string(output)), err
}
