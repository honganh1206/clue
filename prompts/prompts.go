package prompts

import (
	_ "embed"
	"strings"
)

//go:embed claude.txt
var claudeSystemPrompt string

// TODO: Parameterize this with LLM provider name
func ClaudeSystemPrompt() string {
	trimmedPrompt := strings.TrimSpace(string(claudeSystemPrompt))
	if len(trimmedPrompt) == 0 {
		return claudeSystemPrompt
	}

	return trimmedPrompt
}

//go:embed gemini.md
var geminiSystemPrompt string

func GeminiSystemPrompt() string {
	trimmedPrompt := strings.TrimSpace(string(geminiSystemPrompt))
	if len(trimmedPrompt) == 0 {
		return geminiSystemPrompt
	}

	return trimmedPrompt
}
