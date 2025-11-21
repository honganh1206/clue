package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/honganh1206/clue/agent"
	"github.com/honganh1206/clue/inference"
	"github.com/honganh1206/clue/mcp"
	"github.com/honganh1206/clue/server/api"
	"github.com/honganh1206/clue/server/data"
	"github.com/honganh1206/clue/tools"
	"github.com/honganh1206/clue/ui"
)

// TODO: All these parameters should go into a struct
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
			&tools.FinderDefinition,
			&tools.BashDefinition,
			&tools.PlanWriteDefinition,
			&tools.PlanReadDefinition,
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

	var conv *data.Conversation
	var plan *data.Plan

	if convID != "" {
		conv, err = apiClient.GetConversation(convID)
		if err != nil {
			return err
		}
		plan, err = apiClient.GetPlan(convID)
		// TODO: There could be a case where there is no plan for a conversation
		if err != nil {
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

	ctl := ui.NewController()
	a := agent.New(llm, conv, toolBox, apiClient, mcpConfigs, plan, true, ctl)

	sub := agent.NewSubagent(subllm, subToolBox, false)
	a.Sub = sub

	a.RegisterMCPServers()
	defer a.ShutdownMCPServers()

	if useTUI {
		err = tui(ctx, a, ctl)
	} else {
		err = cli(ctx, a)
	}

	if err != nil {
		return err
	}

	return nil
}
