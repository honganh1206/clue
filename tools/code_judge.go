package tools

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/honganh1206/clue/schema"
)

type CodeJudgeInput struct {
	Code       string   `json:"code" jsonschema_description:"The code to be judged and evaluated"`
	Assertions []string `json:"assertions" jsonschema_description:"List of constraints/assertions to check against the code. Use [MUST] prefix for required constraints."`
}

//go:embed code_judge.md
var codeJudgePrompt string

var CodeJudgeInputSchema = schema.Generate[CodeJudgeInput]()

var CodeJudgeDefinition = ToolDefinition{
	Name:        "code_judge",
	Description: codeJudgePrompt,
	InputSchema: CodeJudgeInputSchema,
	Function:    CodeJudge,
}

type JudgementResult struct {
	Score   int    `json:"score"`
	Message string `json:"message"`
}

func CodeJudge(input json.RawMessage) (string, error) {
	var judgeInput CodeJudgeInput
	err := json.Unmarshal(input, &judgeInput)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal input: %w", err)
	}

	if judgeInput.Code == "" {
		return "", fmt.Errorf("code cannot be empty")
	}

	if len(judgeInput.Assertions) == 0 {
		return "", fmt.Errorf("assertions cannot be empty")
	}

	// Create the prompt for LLM evaluation
	prompt := buildJudgePrompt(judgeInput.Code, judgeInput.Assertions)

	// Return the formatted prompt that will be sent to the LLM
	// The actual LLM communication is handled by the calling code
	return prompt, nil
}

func buildJudgePrompt(code string, assertions []string) string {
	// Use the judge prompt template
	promptTemplate := `## Task

You are an expert code judger.
### Code:
` + "```" + `
` + code + `
` + "```" + `

### Constraints:
` + formatAssertions(assertions) + `
`

	return promptTemplate
}

func formatAssertions(assertions []string) string {
	var formatted []string
	for _, assertion := range assertions {
		formatted = append(formatted, "- "+assertion)
	}
	return strings.Join(formatted, "\n")
}
