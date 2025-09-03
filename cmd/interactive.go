package cmd

import (
	"context"
	"log"

	"github.com/honganh1206/clue/agent"
	"github.com/honganh1206/clue/api"
	"github.com/honganh1206/clue/inference"
	"github.com/honganh1206/clue/mcp"
	"github.com/honganh1206/clue/server/data/conversation"
	"github.com/honganh1206/clue/tools"
)

func interactive(ctx context.Context, convID string, modelConfig inference.ModelConfig, client *api.Client, mcpConfigs []mcp.ServerConfig) error {
	model, err := inference.Init(ctx, modelConfig)
	if err != nil {
		log.Fatalf("Failed to initialize model: %s", err.Error())
	}

	toolBox := &tools.ToolBox{
		Tools: []*tools.ToolDefinition{
			&tools.ReadFileDefinition,
			&tools.ListFilesDefinition,
			&tools.EditFileDefinition,
			&tools.GrepSearchDefinition,
			&tools.CodeJudgeDefinition,
		},
	}
	var conv *conversation.Conversation

	if convID != "" {
		conv, err = client.GetConversation(convID)
		if err != nil {
			return err
		}
	} else {
		conv, err = client.CreateConversation()
		if err != nil {
			return err
		}
	}

	a := agent.New(model, conv, toolBox, client, mcpConfigs)

	// In production, use Background() as the final root context()
	// For dev env, TODO for temporary scaffolding
	err = a.Run(ctx)

	if err != nil {
		return err
	}

	return nil
}
