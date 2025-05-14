package commands

import (
	"fmt"
	"os"

	"github.com/honganh1206/adrift/inference"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	engineConfig inference.EngineConfig
	envPath      string
	verbose      bool
)

// The base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "adrift",
	Short: "An AI agent for code editing and assistance",
	// TODO: Update this as we progress
	Long: `Adrift is a command line tool that provides an AI agent to help you with code editing and other tasks.
It supports multiple AI engines including Anthropic, OpenAI (WIP), Gemini (WIP), and local models via Ollama (WIP).`,
	// Run before any subcommand
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := godotenv.Load(envPath)
		if err != nil && verbose {
			fmt.Printf("Warning: Error loading .env file: %v\n", err)
			fmt.Println("Continuing without environment variables from .env file...")
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	rootCmd.PersistentFlags().StringVar(&engineConfig.Type, "engine", string(inference.AnthropicProvider), "Engine type (anthropic, openai, gemini, ollama, deepseek)")
	rootCmd.PersistentFlags().StringVar(&engineConfig.Model, "model", "", "Model to use (depends on selected engine)")
	rootCmd.PersistentFlags().Int64Var(&engineConfig.MaxTokens, "max-tokens", 1024, "Maximum number of tokens in response")
	// rootCmd.PersistentFlags().StringVar(&engineConfig.OllamaHost, "ollama-host", "http://localhost:11434", "Host for Ollama API (when using Ollama engine)")
	// rootCmd.PersistentFlags().StringVar(&engineConfig.APIEndpoint, "api-endpoint", "", "Custom API endpoint for selected engine")
	rootCmd.PersistentFlags().StringVar(&envPath, "env", "./.env", "Path to .env file")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")

	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(listModelsCmd)

	return rootCmd
}
