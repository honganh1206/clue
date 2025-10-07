package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/honganh1206/clue/agent"
	"github.com/honganh1206/clue/api"
	"github.com/honganh1206/clue/inference"
	"github.com/honganh1206/clue/mcp"
	"github.com/honganh1206/clue/server/data/conversation"
	"github.com/honganh1206/clue/tools"
)

func interactive(ctx context.Context, convID string, llmClient inference.BaseLLMClient, apiClient *api.Client, mcpConfigs []mcp.ServerConfig) error {
	llm, err := inference.Init(ctx, llmClient)
	if err != nil {
		log.Fatalf("Failed to initialize model: %s", err.Error())
	}

	toolBox := &tools.ToolBox{
		Tools: []*tools.ToolDefinition{
			&tools.ReadFileDefinition,
			&tools.ListFilesDefinition,
			&tools.EditFileDefinition,
			&tools.GrepSearchDefinition,
			&tools.CodebaseSearchDefinition, // Now handled by subagent
			&tools.BashDefinition,
		},
	}
	var conv *conversation.Conversation

	if convID != "" {
		conv, err = apiClient.GetConversation(convID)
		if err != nil {
			return err
		}
	} else {
		conv, err = apiClient.CreateConversation()
		if err != nil {
			return err
		}
	}

	subllm, err := inference.Init(ctx, inference.BaseLLMClient{
		Provider:   inference.AnthropicProvider,
		Model:      string(inference.Claude35Haiku),
		TokenLimit: 8192,
	})

	if err != nil {
		return fmt.Errorf("failed to initialize sub-agent LLM: %w", err)
	}

	a := agent.New(llm, subllm, conv, toolBox, apiClient, mcpConfigs, true)

	err = tui(ctx, a, conv)

	if err != nil {
		return err
	}

	return nil
}
