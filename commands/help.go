package commands

import (
	"flag"
	"fmt"

	"github.com/honganh1206/clue/inference"
)

// WIP
func showHelp() {
	providers := []inference.ProviderName{
		inference.AnthropicProvider,
	}

	fmt.Println("clue- A simple CLI-based AI coding agent")
	fmt.Println("\nUsage:")
	flag.PrintDefaults()

	fmt.Println("\nModel-specific details:")

	for _, provider := range providers {
		fmt.Printf("\n%s:\n", provider)

		models := inference.ListAvailableModels(provider)
		defaultModel := inference.GetDefaultModel(provider)

		if len(models) > 0 {
			fmt.Printf("  Available models: %s\n", inference.FormatModelsForHelp(models))
		} else {
			fmt.Printf("  Models: Custom model names supported\n")
		}

		fmt.Printf("  Default model: %s\n", defaultModel)
	}

	fmt.Println("\nExamples:")
	fmt.Println("  ./code-editing-agent -provider anthropic -model claude-3-5-sonnet")
	fmt.Println("  ./code-editing-agent -list -provider openai")
}
