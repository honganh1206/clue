package commands

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/honganh1206/clue/agent"
	"github.com/honganh1206/clue/history"
	"github.com/honganh1206/clue/inference"
	"github.com/honganh1206/clue/prompts"
	"github.com/honganh1206/clue/tools"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start a chat with the AI agent",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := history.InitDB()

		if err != nil {
			log.Fatalf("Failed to initialize database: %s", err.Error())
		}
		defer db.Close()

		// FIXME: Some way to make this more configurable?
		systemPrompt := prompts.System()

		modelConfig.PromptPath = systemPrompt

		provider := inference.ProviderName(modelConfig.Provider)
		if modelConfig.Model == "" {
			defaultModel := inference.GetDefaultModel(provider)
			if verbose {
				fmt.Printf("No model specified, using default: %s\n", defaultModel)
			}
			modelConfig.Model = string(defaultModel)
		}

		model, err := inference.Init(modelConfig)
		if err != nil {
			log.Fatalf("Failed to initialize model: %s", err.Error())
		}

		scanner := bufio.NewScanner(os.Stdin)
		getUserMsg := func() (string, bool) {
			if !scanner.Scan() {
				return "", false
			}
			return scanner.Text(), true
		}

		toolDefs := []tools.ToolDefinition{tools.ReadFileDefinition, tools.ListFilesDefinition, tools.EditFileDefinition}

		agent := agent.New(model, getUserMsg, toolDefs, prompts.System(), db)
		// In production, use Background() as the final root context()
		// For dev env, TODO for temporary scaffolding
		err = agent.Run(context.TODO())
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
		}
	},
}
