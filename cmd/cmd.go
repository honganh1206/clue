package cmd

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/honganh1206/clue/api"
	"github.com/honganh1206/clue/inference"
	"github.com/honganh1206/clue/mcp"
	"github.com/honganh1206/clue/server"
	"github.com/honganh1206/clue/utils"
	"github.com/spf13/cobra"
)

var (
	llm              inference.BaseLLMClient
	llmSub           inference.BaseLLMClient
	verbose          bool
	continueConv     bool
	convID           string
	mcpServerCmd     string
	mcpServerConfigs []mcp.ServerConfig
	useTUI           bool
)

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func HelpHandler(cmd *cobra.Command, args []string) error {
	fmt.Println("clue - A simple CLI-based AI coding agent")
	fmt.Println("\nUsage:")
	fmt.Println("\tclue -provider anthropic -model claude-4-sonnet")

	return nil
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

	client := api.NewClient("")

	provider := inference.ProviderName(llm.Provider)
	llmSub.Provider = llm.Provider
	if llm.Model == "" {
		defaultModel := inference.GetDefaultModel(provider)
		defaultModelSub := inference.GetDefaultModelSubagent(provider)
		if verbose {
			fmt.Printf("No model specified, using default model for agent %s and subagent %s\n", defaultModel, defaultModelSub)
		}
		llm.Model = string(defaultModel)
		llmSub.Model = string(defaultModelSub)
	}

	// Default number of max tokens
	if llm.TokenLimit == 0 {
		llm.TokenLimit = 8192
		llmSub.TokenLimit = 8192
	}

	var convID string
	if new {
		convID = ""
	} else {
		if id != "" {
			convID = id
		} else {
			convID, err = client.GetLatestConversationID()
			if err != nil {
				return err
			}
		}
	}

	err = interactive(cmd.Context(), convID, llm, llmSub, client, mcpServerConfigs, useTUI)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}

	return nil
}

func RunServer(cmd *cobra.Command, args []string) error {
	ln, err := net.Listen("tcp", ":11435")
	if err != nil {
		return err
	}
	fmt.Printf("Running background server on %s\n", ln.Addr().String())
	err = server.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
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

	client := api.NewClient("")

	if flagsSet == 1 {
		switch showType {
		case "list":
			conversations, err := client.ListConversations()
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
	provider := inference.ProviderName(llm.Provider)
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

func MCPHandler(cmd *cobra.Command, args []string) error {
	if mcpServerCmd != "" {
		parts := strings.SplitN(mcpServerCmd, ":", 2)
		if len(parts) == 2 {
			id := strings.TrimSpace(parts[0])
			command := strings.TrimSpace(parts[1])
			if id != "" && command != "" {
				config := mcp.ServerConfig{
					ID:      id,
					Command: command,
				}
				mcpServerConfigs = append(mcpServerConfigs, config)
				if verbose {
					fmt.Printf("Added server configuration from flag: %s -> %s\n", id, command)
				}
			} else {
				return fmt.Errorf("invalid server configuration format in flag: %s (expected id:command)", mcpServerCmd)
			}
		} else {
			return fmt.Errorf("invalid server configuration format in flag: %s (expected id:command)", mcpServerCmd)
		}
	}

	if len(mcpServerConfigs) == 0 {
		return errors.New("no server configurations provided (use --server-cmd flag or provide id:command arguments)")
	}

	if err := mcp.SaveConfigs(mcpServerConfigs); err != nil {
		if verbose {
			fmt.Printf("Warning: Could not save configurations: %v\n", err)
		}
	} else if verbose {
		fmt.Printf("Saved %d server configurations to file\n", len(mcpServerConfigs))
	}

	if verbose {
		fmt.Printf("Total server configurations: %d\n", len(mcpServerConfigs))
		for _, config := range mcpServerConfigs {
			fmt.Printf("  - %s: %s\n", config.ID, config.Command)
		}
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

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start clue server",
		Args:  cobra.ExactArgs(0),
		RunE:  RunServer,
	}

	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server",
		Long: `Start an MCP (Model Context Protocol) server with the specified configuration.

Server configurations must be in the format id:command.

Examples:
  clue mcp --server-cmd "my-server:uvx mcp-server-fetch"
  clue mcp "fetch-server:uvx mcp-server-fetch"
  clue mcp "python-server:python my_mcp_server.py --port 8080"
  clue mcp --verbose "node-server:node mcp-server.js"
  clue mcp "server1:uvx mcp-server-fetch" "server2:python other_server.py"`,
		RunE: MCPHandler,
	}

	mcpCmd.Flags().StringVar(&mcpServerCmd, "server-cmd", "", "Server configuration in format id:command (e.g., 'my-server:uvx mcp-server-fetch')")

	rootCmd := &cobra.Command{
		Use:   "clue",
		Short: "An AI agent for code editing and assistance",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if configs, err := mcp.LoadConfigs(); err == nil {
				mcpServerConfigs = configs
				if verbose && len(configs) > 0 {
					fmt.Printf("Loaded %d MCP server configurations\n", len(configs))
				}
			}
			// TODO: Check if serve process is running, if not run here?
		},
		RunE: ChatHandler,
	}

	rootCmd.PersistentFlags().StringVar(&llm.Provider, "provider", string(inference.AnthropicProvider), "Provider (anthropic, openai, gemini, ollama, deepseek)")
	rootCmd.PersistentFlags().StringVar(&llm.Model, "model", "", "Model to use (depends on selected model)")
	rootCmd.PersistentFlags().Int64Var(&llm.TokenLimit, "max-tokens", 0, "Maximum number of tokens in response")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	rootCmd.Flags().BoolVarP(&continueConv, "new-conversation", "n", true, "Continue from the latest conversation")
	rootCmd.Flags().StringVarP(&convID, "id", "i", "", "Conversation ID to ")
	rootCmd.Flags().BoolVar(&useTUI, "tui", true, "Use TUI (Terminal User Interface) mode")

	rootCmd.AddCommand(versionCmd, modelCmd, conversationCmd, helpCmd, serveCmd, mcpCmd)

	return rootCmd
}
