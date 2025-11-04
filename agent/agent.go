package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/honganh1206/clue/api"
	"github.com/honganh1206/clue/inference"
	"github.com/honganh1206/clue/mcp"
	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/server/data/conversation"
	"github.com/honganh1206/clue/tools"
)

type Agent struct {
	llm          inference.LLMClient
	toolBox      *tools.ToolBox
	conversation *conversation.Conversation
	client       *api.Client
	mcp          mcp.Config
	streaming    bool
	// In the future it could be a map of agents, keys are task ID
	Sub *Subagent
}

func New(llm inference.LLMClient, conversation *conversation.Conversation, toolBox *tools.ToolBox, client *api.Client, mcpConfigs []mcp.ServerConfig, streaming bool) *Agent {
	agent := &Agent{
		llm:          llm,
		toolBox:      toolBox,
		conversation: conversation,
		client:       client,
		streaming:    streaming,
	}

	agent.mcp.ServerConfigs = mcpConfigs
	agent.mcp.ActiveServers = []*mcp.Server{}
	agent.mcp.Tools = []mcp.Tools{}
	agent.mcp.ToolMap = make(map[string]mcp.ToolDetails)

	return agent
}

// Run handles a single user message and returns the agent's response
// This method is designed for TUI integration where streaming is handled externally
func (a *Agent) Run(ctx context.Context, userInput string, onDelta func(string)) error {
	readUserInput := true

	// TODO: Add flag to know when to summarize
	a.conversation.Messages = a.llm.SummarizeHistory(a.conversation.Messages, 20)

	if len(a.conversation.Messages) != 0 {
		a.llm.ToNativeHistory(a.conversation.Messages)
	}

	a.llm.ToNativeTools(a.toolBox.Tools)

	for {
		if readUserInput {
			userMsg := &message.Message{
				Role:    message.UserRole,
				Content: []message.ContentBlock{message.NewTextBlock(userInput)},
			}

			err := a.llm.ToNativeMessage(userMsg)
			if err != nil {
				return err
			}

			a.conversation.Append(userMsg)
			a.saveConversation()
		}

		agentMsg, err := a.streamResponse(ctx, onDelta)
		if err != nil {
			return err
		}

		err = a.llm.ToNativeMessage(agentMsg)
		if err != nil {
			return err
		}

		a.conversation.Append(agentMsg)
		a.saveConversation()

		toolResults := []message.ContentBlock{}
		for _, c := range agentMsg.Content {
			switch block := c.(type) {
			case message.ToolUseBlock:
				result := a.executeTool(block.ID, block.Name, block.Input, onDelta)
				toolResults = append(toolResults, result)
			}
		}

		if len(toolResults) == 0 {
			// If we reach this case, it means we have finished processing the tool results
			// and we are safe to return the text response from the agent and wait for the next input.
			readUserInput = true
			break
		}

		readUserInput = false

		toolResultMsg := &message.Message{
			Role:    message.UserRole,
			Content: toolResults,
		}

		err = a.llm.ToNativeMessage(toolResultMsg)
		if err != nil {
			return err
		}

		a.conversation.Append(toolResultMsg)
		a.saveConversation()
	}
	return nil
}

func (a *Agent) executeTool(id, name string, input json.RawMessage, onDelta func(string)) message.ContentBlock {
	var result message.ContentBlock
	if execDetails, isMCPTool := a.mcp.ToolMap[name]; isMCPTool {
		result = a.executeMCPTool(id, name, input, execDetails)
	} else {
		result = a.executeLocalTool(id, name, input)
	}

	// TODO: Shorten the relative/absolute path and underline it.
	// For content to edit, remove it from the display?
	if toolResult, ok := result.(message.ToolResultBlock); ok && toolResult.IsError {
		onDelta(fmt.Sprintf("[red::]\u2717 %s failed[-]\n\n", name))
	} else {
		onDelta(fmt.Sprintf("[green::]\u2713 %s %s[-]\n\n", name, input))
	}

	return result
}

func (a *Agent) executeMCPTool(id, name string, input json.RawMessage, toolDetails mcp.ToolDetails) message.ContentBlock {
	var args map[string]any

	err := json.Unmarshal(input, &args)
	if err != nil {
		// TODO: No error handling here
	}
	if args == nil {
		// This is kinda dumb?
		args = make(map[string]any)
	}

	result, err := toolDetails.Server.Call(context.Background(), name, args)
	if err != nil {
		return message.NewToolResultBlock(id, name,
			fmt.Sprintf("MCP tool %s execution error: %v", name, err), true)
	}
	if result == nil {
		return message.NewToolResultBlock(id, name, "Tool executed successfully but returned no content", false)
	}

	// We have to do this,
	// otherwise there will be an error saying
	// "all messages must have non-empty content etc."
	// even though we do have :)
	content := fmt.Sprintf("%v", result)
	if content == "" {
		return message.NewToolResultBlock(id, name, "Tool executed successfully but returned empty content", false)
	}

	return message.NewToolResultBlock(id, name, content, false)
}

// TODO: Return proper error type
func (a *Agent) executeLocalTool(id, name string, input json.RawMessage) message.ContentBlock {
	var toolDef *tools.ToolDefinition
	var found bool
	// TODO: Toolbox should be a map, not a list of tools
	for _, tool := range a.toolBox.Tools {
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
	var response string
	var err error

	if toolDef.IsSubTool {
		// Make the subagent invoke tools
		toolResultMsg, err := a.runSubagent(id, name, toolDef.Description, input)
		// TODO: Exceed limit of 200k tool result. Trying truncation
		// 25k is best practice from Anthropic
		truncatedResult := a.Sub.llm.TruncateMessage(toolResultMsg, 25000)
		if err != nil {
			return message.NewToolResultBlock(id, name, err.Error(), true)
		}

		var final strings.Builder
		// Iterating over block type is quite tiring?
		for _, content := range truncatedResult.Content {
			switch blk := content.(type) {
			case message.TextBlock:
				final.WriteString(blk.Text)
			case message.ToolResultBlock:
				final.WriteString(blk.Content)
			}
		}

		response = final.String()
	} else {
		// The main agent invokes tools
		response, err = toolDef.Function(input)
	}

	if err != nil {
		return message.NewToolResultBlock(id, name, err.Error(), true)
	}

	return message.NewToolResultBlock(id, name, response, false)
}

func (a *Agent) runSubagent(id, name, toolDescription string, rawInput json.RawMessage) (*message.Message, error) {
	// The OG input from the user gets processed by the main agent
	// and the subagent will consume the processed input.
	// This is for the maybe future of task delegation
	var input struct {
		Query string `json:"query"`
	}

	err := json.Unmarshal(rawInput, &input)
	if err != nil {
		// Check errors instead of pretending nothing went wrong
		return nil, err
	}

	// Can we pass the original background context of the main agent?
	// Or should we let each agent has their own context?
	result, err := a.Sub.Run(context.Background(), toolDescription, input.Query)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (a *Agent) saveConversation() error {
	if len(a.conversation.Messages) > 0 {
		err := a.client.SaveConversation(a.conversation)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *Agent) streamResponse(ctx context.Context, onDelta func(string)) (*message.Message, error) {
	var streamErr error
	var msg *message.Message

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		msg, streamErr = a.llm.RunInference(ctx, onDelta, a.streaming)
	}()

	wg.Wait()

	if streamErr != nil {
		return nil, streamErr
	}

	return msg, nil
}
