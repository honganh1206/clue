package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

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
}

func New(llm inference.LLMClient, conversation *conversation.Conversation, toolBox *tools.ToolBox, client *api.Client, mcpConfigs []mcp.ServerConfig) *Agent {
	agent := &Agent{
		llm:          llm,
		toolBox:      toolBox,
		conversation: conversation,
		client:       client,
	}

	agent.mcp.ServerConfigs = mcpConfigs
	agent.mcp.ActiveServers = []*mcp.Server{}
	agent.mcp.Tools = []mcp.Tools{}
	agent.mcp.ToolMap = make(map[string]mcp.ToolDetails)
	agent.registerMCPServers()

	return agent
}

func (a *Agent) Run(ctx context.Context) error {
	defer a.shutdownMCPServers()
	modelName := a.llm.ProviderName()
	colorCode := getModelColor(modelName)
	resetCode := "\u001b[0m"

	fmt.Printf("Chat with %s%s%s (use 'ctrl-c' to quit)\n", colorCode, modelName, resetCode)

	readUserInput := true

	a.conversation.Messages = a.llm.SummarizeHistory(a.conversation.Messages, 20)

	if len(a.conversation.Messages) != 0 {
		// TODO: Pass the continue conversation flag here?
		// At this point the conversation is still null
		a.llm.ToNativeHistory(a.conversation.Messages)
	}

	a.llm.ToNativeTools(a.toolBox.Tools)

	for {
		if readUserInput {
			fmt.Print("\u001b[94m>\u001b[0m ")
			userInput, ok := getUserMessage()
			if !ok {
				break
			}

			userMsg := &message.Message{
				Role:    message.UserRole,
				Content: []message.ContentBlock{message.NewTextBlock(userInput)},
			}
			// TODO: Error handling
			_ = a.llm.ToNativeMessage(userMsg)
			a.conversation.Append(userMsg)
			a.saveConversation()
		}

		agentMsg, err := a.llm.RunInferenceStream(ctx)
		if err != nil {
			return err
		}
		_ = a.llm.ToNativeMessage(agentMsg)
		a.conversation.Append(agentMsg)
		a.saveConversation()

		toolResults := []message.ContentBlock{}

		for _, c := range agentMsg.Content {
			switch block := c.(type) {
			// TODO: Switch case for text type should be here
			// and we need to stream the response here, not inside the model integrations
			// and we can do proper output formatting here instead
			case message.ToolUseBlock:
				result := a.executeTool(block.ID, block.Name, block.Input)
				toolResults = append(toolResults, result)
			}
		}

		if len(toolResults) == 0 {
			readUserInput = true
			continue
		}

		readUserInput = false

		toolResultMsg := &message.Message{
			Role:    message.UserRole,
			Content: toolResults,
		}

		_ = a.llm.ToNativeMessage(toolResultMsg)
		a.conversation.Append(toolResultMsg)
		a.saveConversation()
	}

	return nil
}

func (a *Agent) registerMCPServers() {
	fmt.Printf("Initializing MCP servers based on %d configurations...\n", len(a.mcp.ServerConfigs))

	for _, serverCfg := range a.mcp.ServerConfigs {
		fmt.Printf("Attempting to create MCP server instance for ID %s (command: %s)\n", serverCfg.ID, serverCfg.Command)
		server, err := mcp.NewServer(serverCfg.ID, serverCfg.Command)
		if err != nil {
			// TODO: Better error handling
			continue
		}

		if server == nil {
			fmt.Fprintf(os.Stderr, "Error creating MCP server instance for ID %s (command: %s): NewServer returned nil\\n", serverCfg.ID, serverCfg.Command)
			continue
		}

		if err := server.Start(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting MCP server %s (command: %s): %v\n", serverCfg.ID, serverCfg.Command, err)
			continue
		}

		fmt.Printf("MCP Server %s started successfully.\n", serverCfg.ID)
		a.mcp.ActiveServers = append(a.mcp.ActiveServers, server)

		fmt.Printf("Fetching tools from MCP server %s...\n", server.ID())
		tool, err := server.ListTools(context.Background()) // Using context.Background() for now
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing tools from MCP server %s: %v\\n", server.ID(), err)
			// We might still want to keep the server active even if listing tools fails initially.
			// Depending on desired robustness, could 'continue' here or allow agent to proceed.
			continue
			// return
		}
		fmt.Printf("Fetched %d tools from MCP server %s\n", len(tool), server.ID())
		a.mcp.Tools = append(a.mcp.Tools, tool)

		for _, t := range tool {
			toolName := fmt.Sprintf("%s_%s", server.ID(), t.Name)

			decl := &tools.ToolDefinition{
				Name:        toolName,
				Description: t.Description,
				InputSchema: t.InputSchema,
			}

			a.toolBox.Tools = append(a.toolBox.Tools, decl)

			a.mcp.ToolMap[toolName] = mcp.ToolDetails{
				Server: server,
				Name:   t.Name,
			}
		}
	}

	// Print all MCP tools that were added
	if len(a.mcp.ToolMap) > 0 {
		var mcpToolNames []string
		for toolName := range a.mcp.ToolMap {
			mcpToolNames = append(mcpToolNames, toolName)
		}
		fmt.Printf("Added MCP tools to agent toolbox: %v\n", mcpToolNames)
	}
}

func (a *Agent) executeTool(id, name string, input json.RawMessage) message.ContentBlock {
	if execDetails, isMCPTool := a.mcp.ToolMap[name]; isMCPTool {
		return a.executeMCPTool(id, name, input, execDetails)
	}
	return a.executeLocalTool(id, name, input)
}

func (a *Agent) executeMCPTool(id, name string, input json.RawMessage, toolDetails mcp.ToolDetails) message.ContentBlock {
	// TODO: Copy from local tool execution
	// might need more rework
	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)
	fmt.Println()

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

func (a *Agent) executeLocalTool(id, name string, input json.RawMessage) message.ContentBlock {
	var toolDef *tools.ToolDefinition
	var found bool
	for _, tool := range a.toolBox.Tools {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}

	if !found {
		// TODO: Return proper error type
		errorMsg := "tool not found"
		return message.NewToolResultBlock(id, name, errorMsg, true)
	}

	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)
	fmt.Println()

	response, err := toolDef.Function(input)

	if err != nil {
		return message.NewToolResultBlock(id, name, err.Error(), true)
	}

	return message.NewToolResultBlock(id, name, response, false)
}

// TODO: Should this be in interactive.go?
func getUserMessage() (string, bool) {
	scanner := bufio.NewScanner(os.Stdin)

	if !scanner.Scan() {
		return "", false
	}
	fmt.Println()
	return scanner.Text(), true
}

// Returns the appropriate ANSI color code for the given model name
func getModelColor(modelName string) string {
	switch modelName {
	case inference.AnthropicModelName:
		return "\u001b[38;5;208m" // Orange
	case inference.GoogleModelName:
		return "\u001b[94m" // Blue
	default:
		return "\u001b[97m" // White (default)
	}
}

func (a *Agent) saveConversation() error {
	if len(a.conversation.Messages) > 0 {
		err := a.client.SaveConversation(a.conversation)
		if err != nil {
			fmt.Printf("DEBUG: Failed conversation details - ConversationID: %s\n", a.conversation.ID)
			return err
		}
	}

	return nil
}

func (a *Agent) shutdownMCPServers() {
	fmt.Println("shutting down MCP servers...")
	for _, s := range a.mcp.ActiveServers {
		fmt.Printf("closing MCP server: %s\n", s.ID())
		if err := s.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing MCP server %s: %v\n", s.ID(), err)
		} else {
			fmt.Printf("MCP server %s closed successfully\n", s.ID())
		}
	}
}
