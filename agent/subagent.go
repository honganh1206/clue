package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honganh1206/clue/inference"
	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/tools"
)

// Subagent is a lightweight agent for executing sub-tasks like codebase_searchagent
// Unlike the main Agent, it:
// - Has limited tools (only read operations)
// - Doesn't save conversations
// - Uses snapshot mode (no streaming)
type Subagent struct {
	llm       inference.LLMClient
	toolBox   *tools.ToolBox
	streaming bool
}

func NewSubagent(llm inference.LLMClient, streaming bool) *Subagent {
	toolBox := &tools.ToolBox{
		Tools: []*tools.ToolDefinition{
			// TODO: Add Glob in the future
			&tools.ReadFileDefinition,
			&tools.GrepSearchDefinition,
			&tools.ListFilesDefinition,
		},
	}

	err := llm.ToNativeTools(toolBox.Tools)
	if err != nil {
		// TODO: Return error instead of panicking
		panic(fmt.Sprintf("failed to register subagent tools: %v", err))
	}

	return &Subagent{
		llm:       llm,
		toolBox:   toolBox,
		streaming: streaming,
	}
}

func (s *Subagent) Run(
	ctx context.Context,
	systemPrompt string,
	input string,
) (*message.Message, error) {
	// TODO: The ToolDescription should be the system prompt for the subagent
	query := systemPrompt + "\n\n" + input

	req := &message.Message{
		Role: message.UserRole,
		Content: []message.ContentBlock{
			message.NewTextBlock(query),
		},
	}

	err := s.llm.ToNativeMessage(req)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize conversation: %w", err)
	}

	for {
		resp, err := s.llm.RunInference(ctx, nil, s.streaming)
		if err != nil {
			return nil, fmt.Errorf("inference failed: %w", err)
		}

		err = s.llm.ToNativeMessage(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to add message to conversation: %w", err)
		}

		var toolResults []message.ContentBlock

		for _, content := range resp.Content {
			if toolUse, ok := content.(message.ToolUseBlock); ok {
				result := s.executeTool(toolUse.ID, toolUse.Name, toolUse.Input)
				toolResults = append(toolResults, result)
			}
		}

		if len(toolResults) == 0 {
			return resp, nil
		}

		// Add tool results back to conversation for next iteration
		toolResultMsg := &message.Message{
			Role:    message.UserRole,
			Content: toolResults,
		}

		// Save to in-mem conversation slice of the subagent
		err = s.llm.ToNativeMessage(toolResultMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to add tool results to conversation: %w", err)
		}
	}
}

func (s *Subagent) executeTool(id, name string, input json.RawMessage) message.ContentBlock {
	var toolDef *tools.ToolDefinition
	var found bool
	// TODO: Toolbox should be a map, not a list of tools
	for _, tool := range s.toolBox.Tools {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}

	if !found {
		errorMsg := "tool not found"
		return message.NewToolResultBlock(id, name, errorMsg, true)
	}

	response, err := toolDef.Function(input)

	if err != nil {
		return message.NewToolResultBlock(id, name, err.Error(), true)
	}

	return message.NewToolResultBlock(id, name, response, false)
}
