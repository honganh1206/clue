package agent

import (
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
	"github.com/invopop/jsonschema"
)

type Agent struct {
	model               inference.Model
	getUserMessage      func() (string, bool)
	tools               []tools.ToolDefinition
	promptPath          string
	conversation        *conversation.Conversation
	client              *api.Client
	mcpConfigs          []mcp.ServerConfig
	mcpActiveServers    []*mcp.Server
	mcpTools            []mcp.Tools
	mcpToolExecutionMap map[string]mcpToolDetails
}

type mcpToolDetails struct {
	Server *mcp.Server
	// Original name for what?
	OriginalName string
}

func New(model inference.Model, getUserMsg func() (string, bool), conversation *conversation.Conversation, toolBox []tools.ToolDefinition, promptPath string, client *api.Client, mcpConfigs []mcp.ServerConfig) *Agent {
	agent := &Agent{
		model:          model,
		getUserMessage: getUserMsg,
		tools:          toolBox,
		promptPath:     promptPath,
		conversation:   conversation,
		client:         client,
		mcpConfigs:     mcpConfigs,
	}

	agent.mcpActiveServers = []*mcp.Server{}
	agent.mcpTools = []mcp.Tools{}
	agent.mcpToolExecutionMap = make(map[string]mcpToolDetails)

	fmt.Printf("Initializing MCP servers based on %d configurations...\n", len(agent.mcpConfigs))

	for _, serverCfg := range agent.mcpConfigs {
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
		agent.mcpActiveServers = append(agent.mcpActiveServers, server)

		// Fetch tools from this active MCP server
		fmt.Printf("Fetching tools from MCP server %s...\n", server.ID())
		toolsFromServer, err := server.ListTools(context.Background()) // Using context.Background() for now
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing tools from MCP server %s: %v\\n", server.ID(), err)
			// We might still want to keep the server active even if listing tools fails initially.
			// Depending on desired robustness, could 'continue' here or allow agent to proceed.
			continue
		}
		fmt.Printf("Fetched %d tools from MCP server %s\n", len(toolsFromServer), server.ID())
		agent.mcpTools = append(agent.mcpTools, toolsFromServer)

		for _, mcpT := range toolsFromServer {
			toolName := fmt.Sprintf("%s_%s", server.ID(), mcpT.Name)

			// This should be a separate function/method
			var paramSchema *jsonschema.Schema

			if len(mcpT.RawInputSchema) > 0 && string(mcpT.RawInputSchema) != "null" {
				schemaErr := json.Unmarshal(mcpT.RawInputSchema, &paramSchema)

				if schemaErr != nil {
					fmt.Fprintf(os.Stderr, "Error unmarshalling schema for MCP tool %s from server %s: %v\n", mcpT.Name, server.ID(), schemaErr)
					continue // Skip this tool if schema is invalid
				}
			} else {
				// Empty schema case
			}

			decl := tools.ToolDefinition{
				Name:        toolName,
				Description: mcpT.Description,
				InputSchema: paramSchema,
			}

			agent.tools = append(agent.tools, decl)
			fmt.Printf("Added MCP tool declaration to agent toolbox: %s\n", toolName)

			agent.mcpToolExecutionMap[toolName] = mcpToolDetails{
				Server:       server,
				OriginalName: mcpT.Name,
			}

		}

	}

	return agent
}

// Returns the appropriate ANSI color code for the given model name
func getModelColor(modelName string) string {
	switch modelName {
	case inference.AnthropicModelName:
		return "\u001b[38;5;208m" // Orange
	case inference.GoogleModelName:
		return "\u001b[94m" // Blue
	case inference.OpenAIModelName:
		return "\u001b[92m" // Green
	case inference.MetaModelName:
		return "\u001b[95m" // Purple/Magenta
	case inference.MistralModelName:
		return "\u001b[96m" // Cyan
	default:
		return "\u001b[97m" // White (default)
	}
}

func (a *Agent) Run(ctx context.Context) error {
	defer a.shutdownMCPServers()
	modelName := a.model.Name()
	colorCode := getModelColor(modelName)
	resetCode := "\u001b[0m"

	fmt.Printf("Chat with %s%s%s (use 'ctrl-c' to quit)\n", colorCode, modelName, resetCode)

	readUserInput := true

	for {
		if readUserInput {
			fmt.Print("\u001b[94m>\u001b[0m ")
			userInput, ok := a.getUserMessage()
			if !ok {
				break
			}

			userMsg := &message.Message{
				Role:    message.UserRole,
				Content: []message.ContentBlockUnion{message.NewTextContentBlock(userInput)},
			}

			a.conversation.Append(userMsg)
			a.saveConversation()
		}

		agentMsg, err := a.model.CompleteStream(ctx, a.conversation.Messages, a.tools)
		if err != nil {
			return err
		}

		a.conversation.Append(agentMsg)
		a.saveConversation()

		toolResults := []message.ContentBlockUnion{}

		for _, c := range agentMsg.Content {
			switch c.Type {
			// TODO: Switch case for text type should be here
			// and we need to stream the response here, not inside the model integrations
			case message.ToolUseType:
				result := a.executeTool(c.OfToolUseBlock.ID, c.OfToolUseBlock.Name, c.OfToolUseBlock.Input)
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

		a.conversation.Append(toolResultMsg)
		a.saveConversation()
	}

	return nil
}

func (a *Agent) executeTool(id, name string, input json.RawMessage) message.ContentBlockUnion {
	if execDetails, isMCPTool := a.mcpToolExecutionMap[name]; isMCPTool {
		return a.executeMCPTool(id, name, input, execDetails)
	}
	return a.executeLocalTool(id, name, input)
}

func (a *Agent) executeMCPTool(id, name string, input json.RawMessage, execDetails mcpToolDetails) message.ContentBlockUnion {
	// TODO: Copy from local tool execution
	// might need more rework
	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)
	println()

	var args map[string]any

	err := json.Unmarshal(input, &args)
	if err != nil {
		// TODO: No error handling here
	}
	if args == nil {
		// This is kinda dumb?
		args = make(map[string]any)
	}

	result, err := execDetails.Server.Call(context.Background(), name, args)

	if err != nil {
		return message.NewToolResultContentBlock(id, name,
			fmt.Sprintf("MCP tool %s execution error: %v", name, err), true)
	}
	if result == nil {
		return message.NewToolResultContentBlock(id, name, "Tool executed successfully but returned no content", false)
	}

	// We have to do this,
	// otherwise there will be an error saying
	// "all messages must have non-empty content etc."
	// even though we do have :)
	content := fmt.Sprintf("%v", result)
	if content == "" {
		return message.NewToolResultContentBlock(id, name, "Tool executed successfully but returned empty content", false)
	}

	return message.NewToolResultContentBlock(id, name, content, false)
}

func (a *Agent) executeLocalTool(id, name string, input json.RawMessage) message.ContentBlockUnion {
	var toolDef tools.ToolDefinition
	var found bool
	for _, tool := range a.tools {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}

	if !found {
		// TODO: Return proper error type
		errorMsg := "tool not found"
		return message.NewToolResultContentBlock(id, name, errorMsg, true)
	}

	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)
	println()

	response, err := toolDef.Function(input)

	if err != nil {
		return message.NewToolResultContentBlock(id, name, err.Error(), true)
	}

	return message.NewToolResultContentBlock(id, name, response, false)
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
	for _, s := range a.mcpActiveServers {
		fmt.Printf("closing MCP server: %s\n", s.ID())
		if err := s.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing MCP server %s: %v\n", s.ID(), err)
		} else {
			fmt.Printf("MCP server %s closed successfully\n", s.ID())
		}
	}
}
