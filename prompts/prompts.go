package prompts

import (
	_ "embed"
)

//go:embed claude.txt
var claudeSystemPrompt string

func ClaudeSystemPrompt() string {
	return claudeSystemPrompt
}
