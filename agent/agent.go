package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/honganh1206/clue/inference"
	"github.com/honganh1206/clue/mcp"
	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/server/api"
	"github.com/honganh1206/clue/server/data"
	"github.com/honganh1206/clue/tools"
	"github.com/honganh1206/clue/ui"
)

type PlanUpdateCallback func(*data.Plan)

type Agent struct {
	LLM     inference.LLMClient
	ToolBox *tools.ToolBox
	Conv    *data.Conversation
	Plan    *data.Plan
	Client  *api.Client
	ctl     *ui.Controller
	MCP     mcp.Config
	// TODO: Default to be streaming. Be a dictator :)
	streaming bool
	// In the future it could be a map of agents, keys are task ID
	Sub *Subagent
}

type Config struct {
	LLM          inference.LLMClient
	Conversation *data.Conversation
	ToolBox      *tools.ToolBox
	Client       *api.Client
	MCPConfigs   []mcp.ServerConfig
	Plan         *data.Plan
	Streaming    bool
	Controller   *ui.Controller
}

func New(config *Config) *Agent {
	agent := &Agent{
		LLM:       config.LLM,
		ToolBox:   config.ToolBox,
		Conv:      config.Conversation,
		Plan:      config.Plan,
		Client:    config.Client,
		streaming: config.Streaming,
		ctl:       config.Controller,
	}

	agent.MCP.ServerConfigs = config.MCPConfigs
	agent.MCP.ActiveServers = []*mcp.Server{}
	agent.MCP.Tools = []mcp.Tools{}
	agent.MCP.ToolMap = make(map[string]mcp.ToolDetails)

	return agent
}

// Run handles a single user message and returns the agent's response
// This method is designed for TUI integration where streaming is handled externally
func (a *Agent) Run(ctx context.Context, userInput string, onDelta func(string)) error {
	readUserInput := true

	// TODO: Add flag to know when to summarize
	a.Conv.Messages = a.LLM.SummarizeHistory(a.Conv.Messages, 20)

	if len(a.Conv.Messages) != 0 {
		a.LLM.ToNativeHistory(a.Conv.Messages)
	}

	a.LLM.ToNativeTools(a.ToolBox.Tools)

	for {
		if readUserInput {
			userMsg := &message.Message{
				Role:    message.UserRole,
				Content: []message.ContentBlock{message.NewTextBlock(userInput)},
			}

			err := a.LLM.ToNativeMessage(userMsg)
			if err != nil {
				return err
			}

			a.Conv.Append(userMsg)
		}

		agentMsg, err := a.streamResponse(ctx, onDelta)
		if err != nil {
			return err
		}

		err = a.LLM.ToNativeMessage(agentMsg)
		if err != nil {
			return err
		}

		a.Conv.Append(agentMsg)

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
			a.saveConversation()
			break
		}

		readUserInput = false

		toolResultMsg := &message.Message{
			Role:    message.UserRole,
			Content: toolResults,
		}

		err = a.LLM.ToNativeMessage(toolResultMsg)
		if err != nil {
			return err
		}

		a.Conv.Append(toolResultMsg)
	}
	return nil
}

func (a *Agent) executeTool(id, name string, input json.RawMessage, onDelta func(string)) message.ContentBlock {
	var result message.ContentBlock
	if execDetails, isMCPTool := a.MCP.ToolMap[name]; isMCPTool {
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
	for _, tool := range a.ToolBox.Tools {
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
		toolResultMsg, err := a.runSubagent(id, name, toolDef.Description, input)
		// 25k tokens is best practice from Anthropic
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
		toolData := &tools.ToolData{
			Input: input,
		}

		switch toolDef.Name {
		case tools.ToolNamePlanWrite, tools.ToolNamePlanRead:
			// Special treatment: Tools dealing with plans need more fields populated
			response, err = a.executePlanTool(toolDef, toolData)
		default:
			response, err = toolDef.Function(toolData)
		}
	}

	if err != nil {
		return message.NewToolResultBlock(id, name, err.Error(), true)
	}

	return message.NewToolResultBlock(id, name, response, false)
}

// TODO: The ToolData struct should not have a Client field
// All CRUD operations for Plan should be executed here
// The plan_write and plan_read tools should only receive a Plan object
func (a *Agent) executePlanTool(toolDef *tools.ToolDefinition, toolData *tools.ToolData) (string, error) {
	toolData.Client = a.Client
	toolData.ConversationID = a.Conv.ID

	response, err := toolDef.Function(toolData)
	if err != nil {
		return "", err
	}
	// Reflect the plan on the UI
	if toolDef.Name == tools.ToolNamePlanWrite {
		a.Plan, _ = a.Client.GetPlan(a.Conv.ID)
		a.PublishPlan()
	}
	return response, nil
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

func (a *Agent) PublishPlan() {
	a.ctl.Publish(&ui.State{Plan: a.Plan})
}

func (a *Agent) saveConversation() error {
	if len(a.Conv.Messages) > 0 {
		err := a.Client.SaveConversation(a.Conv)
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
		msg, streamErr = a.LLM.RunInference(ctx, onDelta, a.streaming)
	}()

	wg.Wait()

	if streamErr != nil {
		return nil, streamErr
	}

	return msg, nil
}
