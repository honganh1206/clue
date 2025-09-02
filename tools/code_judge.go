package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/honganh1206/clue/schema"
)

type CodeJudgeInput struct {
	Code       string   `json:"code" jsonschema_description:"The code to be judged and evaluated"`
	Assertions []string `json:"assertions" jsonschema_description:"List of constraints/assertions to check against the code. Use [MUST] prefix for required constraints."`
}

var CodeJudgeInputSchema = schema.Generate[CodeJudgeInput]()

// TODO: Calling this is very costly :)
// 2 requests already cost 2 mil tokens
var CodeJudgeDefinition = ToolDefinition{
	Name: "code_judge",
	Description: `Do NOT invoke this tool unless being told explicitly to.
	Evaluate the code against the constraints and provide:

1. Brief code analysis
2. Constraints met
3. Constraints not met
4. Final score (1-3)

### Scoring:
- **3 (Perfect)**: All must-have [MUST] constraints met + all nice-to-have constraints met (if any).
- **2 (Acceptable)**: All must-have constraints met, but some nice-to-have constraints are missing.
- **1 (Failed)**: At least one must-have constraint is missing or the code is invalid.

### Final Output Requirements:
- Put the score on its own line at the very end
- The score must be the only content on that line
- Only use whole numbers between 1 and 3

Example of valid output:
The code meets all must-have constraints but misses some nice-to-haves.
2

Example of INVALID output:
Score: 2/3
Final rating: 2.5
`,
	InputSchema: CodeJudgeInputSchema,
	Function:    CodeJudge,
}

type JudgementResult struct {
	Score   int    `json:"score"`
	Message string `json:"message"`
}

func CodeJudge(input json.RawMessage) (string, error) {
	var judgeInput CodeJudgeInput
	// TODO: Fail to unmarshal input here
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
