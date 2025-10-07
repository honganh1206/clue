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
		// TODO: Return error instead of panicking like an amateur
		// But for now, panic because if tools don't register, nothing will work
		panic(fmt.Sprintf("failed to register subagent tools: %v", err))
	}

	return &Subagent{
		llm:       llm,
		toolBox:   toolBox,
		streaming: streaming,
	}
}

// Run executes the subagent with a system prompt and user query
// It loops through tool calls until it gets a final answer
func (s *Subagent) Run(
	ctx context.Context,
	toolDescription string, // So it knows its purpose (revolutionary!)
	input string, // So it knows what to search for (groundbreaking!)
) (*message.Message, error) {
	fmt.Printf("DEBUG SUBAGENT Run: Starting - description=%s, query=%s\n", toolDescription, input)

	// TODO: The ToolDescription should be the system prompt for the subagent
	query := toolDescription + "\n\n" + input

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
		agentMsg, err := s.llm.RunInference(ctx, func(string) {}, s.streaming)
		if err != nil {
			fmt.Printf("DEBUG SUBAGENT Run: LLM inference failed: %v\n", err)
			return nil, fmt.Errorf("inference failed: %w", err)
		}
		fmt.Printf("DEBUG SUBAGENT Run: LLM response received, content blocks: %d\n", len(agentMsg.Content))

		err = s.llm.ToNativeMessage(agentMsg)
		if err != nil {
			fmt.Printf("DEBUG SUBAGENT Run: Error adding agent message to conversation: %v\n", err)
			return nil, fmt.Errorf("failed to add message to conversation: %w", err)
		}

		// Check for tool uses and execute them ALL (not just the first one)
		var toolResults []message.ContentBlock
		// hasTools := false

		for _, content := range agentMsg.Content {
			if toolUse, ok := content.(message.ToolUseBlock); ok {
				// hasTools = true
				// Execute the tool and collect result
				fmt.Printf("DEBUG SUBAGENT Run: Executing tool - name=%s, id=%s\n", toolUse.Name, toolUse.ID)
				result := s.executeTool(toolUse.ID, toolUse.Name, toolUse.Input)
				toolResults = append(toolResults, result)
			}
		}

		if len(toolResults) == 0 {
			fmt.Printf("DEBUG SUBAGENT Run: No tools called, returning final answer\n")
			return agentMsg, nil
		}

		fmt.Printf("DEBUG SUBAGENT Run: Processed %d tool results\n", len(toolResults))

		// Add tool results back to conversation for next iteration
		toolResultMsg := &message.Message{
			Role:    message.UserRole,
			Content: toolResults,
		}

		// Save to in-mem conversation slice of the subagent. Necessary?
		err = s.llm.ToNativeMessage(toolResultMsg)
		if err != nil {
			fmt.Printf("DEBUG SUBAGENT Run: Error adding tool results to conversation: %v\n", err)
			return nil, fmt.Errorf("failed to add tool results to conversation: %w", err)
		}

		// Continue loop to let LLM process tool results and either call more tools or provide final answer
	}
}

// executeTool runs a tool and returns the result
func (s *Subagent) executeTool(id, name string, input json.RawMessage) message.ContentBlock {
	fmt.Printf("DEBUG SUBAGENT executeTool: id=%s, name=%s, input=%s\n", id, name, string(input))
	var toolDef *tools.ToolDefinition
	var found bool
	for _, tool := range s.toolBox.Tools {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}

	if !found {
		errorMsg := "tool not found"
		fmt.Printf("DEBUG SUBAGENT executeTool: tool not found - name=%s\n", name)
		fmt.Printf("[red]\u2717 %s (subagent) failed\n\n", name)
		return message.NewToolResultBlock(id, name, errorMsg, true)
	}

	response, err := toolDef.Function(input)

	if err != nil {
		fmt.Printf("DEBUG SUBAGENT executeTool: error - name=%s, error=%v\n", name, err)
		fmt.Printf("[red]\u2717 (subagent) %s failed\n\n", name)
		return message.NewToolResultBlock(id, name, err.Error(), true)
	}

	fmt.Printf("DEBUG SUBAGENT executeTool: success - name=%s, response=%s\n", name, response)
	fmt.Printf("[green]\u2713 (subagent) %s %s\n\n", name, input)
	return message.NewToolResultBlock(id, name, response, false)
}
