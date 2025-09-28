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

// bashPrompt embeds the bash tool documentation from bash.md file
//go:embed bash.md
var bashPrompt string

// BashInput represents the input structure for the bash tool
// It contains a single command field that specifies the bash command to execute
type BashInput struct {
	Command string `json:"command" jsonschema_description:"The bash command to execute."`
}

// BashInputSchema generates the JSON schema for BashInput using reflection
var BashInputSchema = schema.Generate[BashInput]()

// BashDefinition provides the tool definition for the bash command executor
// This includes the tool name, description (from embedded markdown), input schema, and execution function
var BashDefinition = ToolDefinition{
	Name:        "bash",
	Description: bashPrompt,
	InputSchema: BashInputSchema, // Machine-readable description of the tool's input
	Function:    Bash,
}

// Bash executes a bash command and returns its output
// It takes a JSON-encoded BashInput as input and returns the command output as a string
// If the command fails, it returns both the error message and the output for debugging
func Bash(input json.RawMessage) (string, error) {
	// Parse the JSON input into a BashInput struct
	bashInput := BashInput{}
	err := json.Unmarshal(input, &bashInput)
	if err != nil {
		return "", err
	}

	// Execute the bash command using exec.Command
	// The "-c" flag tells bash to execute the command string
	cmd := exec.Command("bash", "-c", bashInput.Command)
	
	// CombinedOutput captures both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If command execution fails, return error details with output for debugging
		// We don't return the error directly to allow the caller to handle it gracefully
		return fmt.Sprintf("Command failed with error: %s\nOutput: %s", err.Error(), string(output)), nil
	}

	// Return the trimmed output (removing trailing whitespace)
	return strings.TrimSpace(string(output)), err
}
