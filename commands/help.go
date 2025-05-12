package commands

import (
	"flag"
	"fmt"

	"github.com/honganh1206/adrift/inference"
)

// WIP
func showHelp() {
	providers := []inference.Provider{
		inference.AnthropicProvider,
	}

	fmt.Println("adrift- A simple CLI-based AI coding agent")
	fmt.Println("\nUsage:")
	flag.PrintDefaults()

	fmt.Println("\nEngine-specific details:")

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

		// // Engine-specific notes
		// switch provider {
		// case :
		// 	fmt.Println("  Note: Requires Ollama to be running. Set custom host with -ollama-host")
		// case EngineLocalLLM:
		// 	fmt.Println("  Note: Specify the complete model path with -model")
		// case EngineAzureOpenAI:
		// 	fmt.Println("  Note: Requires Azure-specific configuration in .env file")
		// }
	}

	fmt.Println("\nExamples:")
	fmt.Println("  ./code-editing-agent -engine anthropic -model claude-3-5-sonnet")
	// fmt.Println("  ./code-editing-agent -engine openai -model gpt-4o -max-tokens 2048")
	// fmt.Println("  ./code-editing-agent -engine ollama -model llama3 -ollama-host http://localhost:11434")
	// fmt.Println("  ./code-editing-agent -engine gemini -model gemini-2.5-pro")
	fmt.Println("  ./code-editing-agent -list-models -engine openai")
}
