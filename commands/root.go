package commands

import (
	"os"

	"github.com/honganh1206/clue/inference"
	"github.com/spf13/cobra"
)

var (
	modelConfig inference.ModelConfig
	envPath     string
	verbose     bool
)

var rootCmd = &cobra.Command{
	Use:   "clue",
	Short: "An AI agent for code editing and assistance",
	// TODO: Update this as we progress
	Long: `Clue is a command line tool that provides an AI agent to help you with code editing and other tasks.
It supports multiple AI models from Anthropic, OpenAI (WIP), Gemini (WIP), and local models via Ollama (WIP).`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	rootCmd.PersistentFlags().StringVar(&modelConfig.Provider, "provider", string(inference.AnthropicProvider), "Provider (anthropic, openai, gemini, ollama, deepseek)")
	rootCmd.PersistentFlags().StringVar(&modelConfig.Model, "model", "", "Model to use (depends on selected model)")
	rootCmd.PersistentFlags().Int64Var(&modelConfig.MaxTokens, "max-tokens", 4096, "Maximum number of tokens in response")
	// rootCmd.PersistentFlags().StringVar(&engineConfig.OllamaHost, "ollama-host", "http://localhost:11434", "Host for Ollama API (when using Ollama engine)")
	// rootCmd.PersistentFlags().StringVar(&engineConfig.APIEndpoint, "api-endpoint", "", "Custom API endpoint for selected engine")
	rootCmd.PersistentFlags().StringVar(&envPath, "env", "./.env", "Path to .env file")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")

	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(modelCmd)
	rootCmd.AddCommand(NewConversationCmd())

	return rootCmd
}
