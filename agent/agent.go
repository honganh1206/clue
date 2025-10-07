package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	sub *Subagent
}

func New(llm, subllm inference.LLMClient, conversation *conversation.Conversation, toolBox *tools.ToolBox, client *api.Client, mcpConfigs []mcp.ServerConfig, streaming bool) *Agent {
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
	agent.registerMCPServers()
	agent.sub = NewSubagent(subllm, false)

	return agent
}

// Run handles a single user message and returns the agent's response
// This method is designed for TUI integration where streaming is handled externally
func (a *Agent) Run(ctx context.Context, userInput string, onDelta func(string)) error {

	readUserInput := true

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

func (a *Agent) registerMCPServers() {
	// fmt.Printf("Initializing MCP servers based on %d configurations...\n", len(a.mcp.ServerConfigs))

	for _, serverCfg := range a.mcp.ServerConfigs {
		// fmt.Printf("Attempting to create MCP server instance for ID %s (command: %s)\n", serverCfg.ID, serverCfg.Command)
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

		// fmt.Printf("MCP Server %s started successfully.\n", serverCfg.ID)
		a.mcp.ActiveServers = append(a.mcp.ActiveServers, server)

		// fmt.Printf("Fetching tools from MCP server %s...\n", server.ID())
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
		// fmt.Printf("Added MCP tools to agent toolbox: %v\n", mcpToolNames)
	}
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
		onDelta(fmt.Sprintf("[red]\u2717 %s failed\n\n", name))
	} else {
		onDelta(fmt.Sprintf("[green]\u2713 %s %s\n\n", name, input))
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
	fmt.Printf("DEBUG AGENT executeLocalTool: id=%s, name=%s, input=%s\n", id, name, string(input))
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
		errorMsg := "tool not found"
		fmt.Printf("DEBUG AGENT executeLocalTool: tool not found - name=%s\n", name)
		return message.NewToolResultBlock(id, name, errorMsg, true)
	}
	var response string
	var err error

	// Problem is that runSubagent returns a whole Message
	// but we need toolDef to know we are invoking codebase_search
	// and the for loops in agent.go and subagent.go are quite redundant

	if toolDef.IsSubTool {
		fmt.Printf("DEBUG AGENT executeLocalTool: Calling subagent for tool=%s\n", name)
		toolResultMsg, err := a.runSubagent(id, name, toolDef.Description, input)
		if err != nil {
			fmt.Printf("DEBUG AGENT executeLocalTool: Subagent failed - error=%v\n", err)
			return message.NewToolResultBlock(id, name, err.Error(), true)
		}

		var finalAnswer strings.Builder
		for _, content := range toolResultMsg.Content {
			if textBlk, ok := content.(message.TextBlock); ok {
				finalAnswer.WriteString(textBlk.Text)
			} else if toolResultBlk, ok := content.(message.ToolResultBlock); ok {
				finalAnswer.WriteString(toolResultBlk.Content)
			}
		}

		response = finalAnswer.String()
		fmt.Printf("DEBUG AGENT executeLocalTool: Subagent response received - name=%s, response_length=%d\n", name, len(response))
	} else {
		fmt.Printf("DEBUG AGENT executeLocalTool: Executing regular tool=%s\n", name)
		response, err = toolDef.Function(input)
	}

	if err != nil {
		return message.NewToolResultBlock(id, name, err.Error(), true)
	}

	fmt.Printf("DEBUG AGENT executeLocalTool: success - name=%s, response=%s\n", name, response)
	return message.NewToolResultBlock(id, name, response, false)
}

func (a *Agent) runSubagent(id, name, toolDescription string, input json.RawMessage) (*message.Message, error) {
	fmt.Printf("DEBUG AGENT runSubagent: Starting - tool=%s, id=%s\n", name, id)
	// The OG user input still needs to be processed by the main agent
	// before we pass it to the subagent
	var searchInput struct {
		Query string `json:"query"`
	}
	err := json.Unmarshal(input, &searchInput)
	if err != nil {
		fmt.Printf("DEBUG AGENT runSubagent: Error unmarshaling input: %v\n", err)
		// Check errors instead of pretending nothing went wrong
		return nil, err
	}

	fmt.Printf("DEBUG AGENT runSubagent: Calling subagent with query=%s, description=%s\n", searchInput.Query, toolDescription)
	// Actually call the subagent with the query (revolutionary concept!)
	result, err := a.sub.Run(context.Background(), toolDescription, searchInput.Query)
	if err != nil {
		fmt.Printf("DEBUG AGENT runSubagent: Subagent error: %v\n", err)
		// Check errors instead of pretending nothing went wrong
		return nil, err
	}

	fmt.Printf("DEBUG AGENT runSubagent: Subagent completed successfully, content blocks: %d\n", len(result.Content))
	return result, nil
}

func (a *Agent) saveConversation() error {
	// Skip if no client (for sub-agents)
	if a.client == nil {
		return nil
	}

	if len(a.conversation.Messages) > 0 {
		err := a.client.SaveConversation(a.conversation)
		if err != nil {
			fmt.Printf("DEBUG: Failed conversation details - ConversationID: %s\n", a.conversation.ID)
			return err
		}
	}

	return nil
}

func (a *Agent) ShutdownMCPServers() {
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
func (a *Agent) streamResponse(ctx context.Context, onDelta func(string)) (*message.Message, error) {
	fmt.Printf("DEBUG AGENT streamResponse: Starting main agent LLM inference\n")
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
		fmt.Printf("DEBUG AGENT streamResponse: Main agent LLM error: %v\n", streamErr)
		return nil, streamErr
	}

	fmt.Printf("DEBUG AGENT streamResponse: Main agent LLM completed successfully, content blocks: %d\n", len(msg.Content))
	return msg, nil
}
