package commands

import (
	"fmt"

	"github.com/honganh1206/adrift/inference"
	"github.com/spf13/cobra"
)

// listModelsCmd represents the listModels command
var listModelsCmd = &cobra.Command{
	Use:   "list",
	Short: "List available models for the selected provider",
	Run: func(cmd *cobra.Command, args []string) {
		provider := inference.ProviderName(modelConfig.Provider)
		models := inference.ListAvailableModels(provider)

		if len(models) > 0 {
			fmt.Printf("Available models for %s:\n", provider)
			for _, model := range models {
				fmt.Printf("  - %s\n", model)
			}
		} else {
			fmt.Printf("For %s, specify your custom model name with the --model flag\n", provider)
		}
	},
}
