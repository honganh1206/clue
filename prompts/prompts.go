package prompts

import (
	_ "embed"
)

//go:embed system.txt
var systemPrompt string

func System() string {
	return systemPrompt
}
