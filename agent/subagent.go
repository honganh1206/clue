package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/honganh1206/clue/inference"
	"github.com/honganh1206/clue/mcp"
	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/server/data/conversation"
	"github.com/honganh1206/clue/tools"
)

// InitSubAgentRunner sets up the sub-agent runner for codebase_search
// This should be called during application initialization
func InitSubAgentRunner() {
	tools.SubAgentRunner = runSubAgent
}

func runSubAgent(systemPrompt, userQuery string) (string, error) {
	ctx := context.Background()

	// Create sub-agent LLM (hardcoded to Haiku for now)
	// Note: System prompt should be passed differently for each provider
	// For now, we'll prepend it to the user query as a workaround
	llm, err := inference.Init(ctx, inference.BaseLLMClient{
		Provider:   inference.AnthropicProvider,
		Model:      string(inference.Claude35Haiku),
		TokenLimit: 8192,
	})
	if err != nil {
		return "", fmt.Errorf("failed to initialize sub-agent LLM: %w", err)
	}

	// Create ephemeral conversation (system prompt is handled by LLM client)
	conv := &conversation.Conversation{
		ID:       "codebase_search_ephemeral",
		Messages: []*message.Message{},
	}

	// Setup limited toolbox for sub-agent (only safe read operations)
	subAgentToolBox := &tools.ToolBox{
		Tools: []*tools.ToolDefinition{
			&tools.ReadFileDefinition,
			&tools.GrepSearchDefinition,
			&tools.ListFilesDefinition,
		},
	}

	// Create sub-agent using same Agent struct
	// Pass nil for client (won't save conversations), streaming=false for snapshot mode
	subAgent := New(
		llm,
		conv,
		subAgentToolBox,
		nil,                  // nil client = don't save conversations
		[]mcp.ServerConfig{}, // no MCP servers for sub-agent
		false,                // streaming=false for snapshot mode
	)

	// Run sub-agent with the user query (prepend system prompt as instructions)
	fullQuery := systemPrompt + "\n\n" + userQuery
	err = subAgent.Run(ctx, fullQuery, func(delta string) {
		// No-op: sub-agent doesn't stream output
	})
	if err != nil {
		return "", fmt.Errorf("sub-agent run failed: %w", err)
	}

	// Extract final response from last assistant message
	var result strings.Builder
	for i := len(conv.Messages) - 1; i >= 0; i-- {
		if conv.Messages[i].Role == message.AssistantRole {
			for _, content := range conv.Messages[i].Content {
				if textBlock, ok := content.(message.TextBlock); ok {
					result.WriteString(textBlock.Text)
				}
			}
			break
		}
	}

	return result.String(), nil
}
