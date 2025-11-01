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

func interactive(ctx context.Context, convID string, llmClient, llmClientSub inference.BaseLLMClient, apiClient *api.Client, mcpConfigs []mcp.ServerConfig, useTUI bool) error {
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
			&tools.CodebaseSearchAgentDefinition, // Now handled by subagent
			&tools.BashDefinition,
		},
	}

	subToolBox := &tools.ToolBox{
		Tools: []*tools.ToolDefinition{
			// TODO: Add Glob in the future
			&tools.ReadFileDefinition,
			&tools.GrepSearchDefinition,
			&tools.ListFilesDefinition,
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

	subllm, err := inference.Init(ctx, llmClientSub)

	if err != nil {
		return fmt.Errorf("failed to initialize sub-agent LLM: %w", err)
	}

	a := agent.New(llm, conv, toolBox, apiClient, mcpConfigs, true)

	sub := agent.NewSubagent(subllm, subToolBox, false)
	a.Sub = sub

	a.RegisterMCPServers()
	defer a.ShutdownMCPServers()

	if useTUI {
		err = tui(ctx, a, conv)
	} else {
		err = cli(ctx, a, conv)
	}

	if err != nil {
		return err
	}

	return nil
}
