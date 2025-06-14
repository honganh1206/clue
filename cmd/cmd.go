package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/honganh1206/clue/agent"
	"github.com/honganh1206/clue/conversation"
	"github.com/honganh1206/clue/inference"
	"github.com/honganh1206/clue/prompts"
	"github.com/honganh1206/clue/utils"
	"github.com/spf13/cobra"
)

var (
	modelConfig  inference.ModelConfig
	envPath      string
	verbose      bool
	continueConv bool
	convID       string
)
var (
	Version   = "0.1.0"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func HelpHandler(cmd *cobra.Command, args []string) error {
	fmt.Println("clue - A simple CLI-based AI coding agent")
	fmt.Println("\nUsage:")
	fmt.Println("\tclue -provider anthropic -model claude-4-sonnet")

	return nil
}

func initConversationDsn() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get home directory:", err)
	}

	dsn := filepath.Join(homeDir, ".local", ".clue", "conversation.db")
	return dsn
}

func ChatHandler(cmd *cobra.Command, args []string) error {
	new, err := cmd.Flags().GetBool("new-conversation")
	if err != nil {
		return err
	}

	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}

	dsn := initConversationDsn()
	db, err := conversation.InitDB(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize database: %s", err.Error())
	}
	defer db.Close()

	// FIXME: Some way to make this more configurable?
	systemPrompt := prompts.ClaudeSystemPrompt()

	modelConfig.PromptPath = systemPrompt

	provider := inference.ProviderName(modelConfig.Provider)
	if modelConfig.Model == "" {
		defaultModel := inference.GetDefaultModel(provider)
		if verbose {
			fmt.Printf("No model specified, using default: %s\n", defaultModel)
		}
		modelConfig.Model = string(defaultModel)
	}

	var conversationID string
	if new {
		conversationID = ""
	} else {
		if id != "" {
			conversationID = id
		} else {
			conversationID, err = conversation.LatestID(db)
			if err != nil {
				return err
			}
		}
	}

	err = agent.Gen(conversationID, modelConfig, db)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}

	return nil
}

func ConversationHandler(cmd *cobra.Command, args []string) error {
	list, err := cmd.Flags().GetBool("list")
	if err != nil {
		return err
	}

	flagsSet := 0
	showType := ""

	if list {
		flagsSet++
		showType = "list"
	}

	if flagsSet > 1 {
		return errors.New("only one of '--list'")
	}

	dsn := initConversationDsn()
	db, err := conversation.InitDB(dsn)

	if err != nil {
		log.Fatalf("Failed to initialize database: %s", err.Error())
		return err
	}

	defer db.Close()

	if flagsSet == 1 {
		switch showType {
		case "list":
			conversations, err := conversation.List(db)
			if err != nil {
				log.Fatalf("Error listing conversations: %v", err)
			}

			if len(conversations) == 0 {
				fmt.Println("No conversations found.")
			} else {

				headers := []string{"ID", "Created", "Last Message", "Messages"}
				var data [][]string

				for _, conv := range conversations {
					row := []string{
						conv.ID,
						// TODO: A more read-friendly format?
						conv.CreatedAt.Format(time.RFC3339),
						conv.LatestMessageTime.Format(time.RFC3339),
						fmt.Sprintf("%d", conv.MessageCount),
					}
					data = append(data, row)
				}

				utils.RenderTable(headers, data)
			}
		}
	}

	return nil
}

func ModelHandler(cmd *cobra.Command, args []string) error {
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

	return nil
}

func NewCLI() *cobra.Command {
	modelCmd := &cobra.Command{
		Use:   "model",
		Short: "List available models for the selected provider",
		RunE:  ModelHandler,
	}

	conversationCmd := &cobra.Command{
		Use:   "conversation",
		Short: "Show conversations",
		// Args:  cobra.ExactArgs(1),
		RunE: ConversationHandler,
	}

	conversationCmd.Flags().BoolP("list", "l", false, "Display all conversations")

	helpCmd := &cobra.Command{
		Use:   "help",
		Short: "Show help",
		RunE:  HelpHandler,
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of clue",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Clue version %s (commit: %s, built: %s)\n", Version, GitCommit, BuildTime)
		},
	}
	rootCmd := &cobra.Command{
		Use:   "clue",
		Short: "An AI agent for code editing and assistance",
		RunE:  ChatHandler,
	}

	rootCmd.PersistentFlags().StringVar(&modelConfig.Provider, "provider", string(inference.AnthropicProvider), "Provider (anthropic, openai, gemini, ollama, deepseek)")
	rootCmd.PersistentFlags().StringVar(&modelConfig.Model, "model", "", "Model to use (depends on selected model)")
	rootCmd.PersistentFlags().Int64Var(&modelConfig.MaxTokens, "max-tokens", 4096, "Maximum number of tokens in response")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	rootCmd.Flags().BoolVarP(&continueConv, "new-conversation", "n", true, "Continue from the latest conversation")
	rootCmd.Flags().StringVarP(&convID, "id", "i", "", "Conversation ID to ")

	rootCmd.AddCommand(versionCmd, modelCmd, conversationCmd, helpCmd)

	return rootCmd
}
