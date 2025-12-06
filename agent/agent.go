package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/honganh1206/tinker/inference"
	"github.com/honganh1206/tinker/mcp"
	"github.com/honganh1206/tinker/message"
	"github.com/honganh1206/tinker/schema"
	"github.com/honganh1206/tinker/server/api"
	"github.com/honganh1206/tinker/server/data"
	"github.com/honganh1206/tinker/tools"
	"github.com/honganh1206/tinker/ui"
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

	isError := false
	if toolResult, ok := result.(message.ToolResultBlock); ok && toolResult.IsError {
		isError = true
	}
	onDelta(FormatToolResultMessage(name, input, isError))

	return result
}

func FormatToolResultMessage(name string, input json.RawMessage, isError bool) string {
	var detail string

	switch name {
	case tools.ToolNameReadFile:
		i, err := schema.DecodeRaw[tools.ReadFileInput](input)
		if err == nil {
			detail = i.Path
		}
		return ui.FormatToolResult(ui.ToolResultFormat{Name: "Read", Detail: detail, IsError: isError})

	case tools.ToolNameEditFile:
		i, err := schema.DecodeRaw[tools.EditFileInput](input)
		if err == nil {
			detail = i.Path
		}
		return ui.FormatToolResult(ui.ToolResultFormat{Name: "Edit", Detail: detail, IsError: isError})

	case tools.ToolNameListFiles:
		i, err := schema.DecodeRaw[tools.ListFilesInput](input)
		if err == nil {
			detail = i.Path
		}
		return ui.FormatListFilesToolResult(ui.ToolResultFormat{Name: "List", Detail: detail, IsError: isError})

	case tools.ToolNameBash:
		i, err := schema.DecodeRaw[tools.BashInput](input)
		if err == nil {
			detail = i.Command
		}
		return ui.FormatToolResult(ui.ToolResultFormat{Name: "Bash", Detail: detail, IsError: isError})

	case tools.ToolNameFinder:
		i, err := schema.DecodeRaw[tools.FinderInput](input)
		if err == nil {
			detail = i.Query
		}
		return ui.FormatToolResult(ui.ToolResultFormat{Name: "Finder", Detail: detail, IsError: isError})

	case tools.ToolNameGrepSearch:
		i, err := schema.DecodeRaw[tools.GrepSearchInput](input)
		if err == nil {
			detail = i.Pattern
		}
		return ui.FormatToolResult(ui.ToolResultFormat{Name: "Grep", Detail: detail, IsError: isError})

	case tools.ToolNamePlanRead, tools.ToolNamePlanWrite:
		return ui.FormatToolResult(ui.ToolResultFormat{Name: "Plan", IsError: isError})

	default:
		return ui.FormatToolResult(ui.ToolResultFormat{Name: name, IsError: isError})
	}
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
	var toolOutput string
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

		toolOutput = final.String()
	} else {
		toolInput := tools.ToolInput{
			RawInput: input,
			ToolObject: &tools.ToolObject{
				Plan: &data.Plan{},
			},
		}

		switch toolDef.Name {
		case tools.ToolNamePlanWrite, tools.ToolNamePlanRead:
			// Special treatment: Tools dealing with plans need more fields populated
			toolOutput, err = a.executePlanTool(toolDef, toolInput)
		// TODO: Should we use a.Plan for the main agent to refer to its own plan,
		// instead of forcing it to use plan_read?
		default:
			toolOutput, err = toolDef.Function(toolInput)
		}
	}

	if err != nil {
		return message.NewToolResultBlock(id, name, err.Error(), true)
	}

	// Temp casting to ToolOutput type
	return message.NewToolResultBlock(id, name, string(toolOutput), false)
}

func (a *Agent) executePlanTool(toolDef *tools.ToolDefinition, toolInput tools.ToolInput) (string, error) {
	var p *data.Plan
	var err error

	p, err = a.Client.GetPlan(a.Conv.ID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			p, err = a.Client.CreatePlan(a.Conv.ID)
			if err != nil {
				return "", fmt.Errorf("plan_write: failed to create new plan for conversation with ID '%s' for adding steps: %w", a.Conv.ID, err)
			}
		} else {
			return "", fmt.Errorf("plan_write: failed to get plan with conversation ID '%s': %w", a.Conv.ID, err)
		}
	}

	if p == nil {
		return "", fmt.Errorf("plan_write: plan object is nil after GetPlan/CreatePlan")
	}
	toolInput.Plan = p

	response, err := toolDef.Function(toolInput)

	if err = a.Client.SavePlan(p); err != nil {
		return "", fmt.Errorf("plan_write: failed to save plan '%s' after setting status: %w", a.Conv.ID, err)
	}

	// Synchronization step, just to be sure
	a.Plan = p

	// Send an update plan event to the UI
	go func() {
		a.ctl.Publish(&ui.State{Plan: p})
	}()

	return response, nil
}

func (a *Agent) runSubagent(id, name, toolDescription string, rawInput json.RawMessage) (*message.Message, error) {
	// The OG input from the user gets processed by the main agent
	// and the subagent will consume the processed input.
	// This is for the maybe future of task delegation
	var input tools.FinderInput

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

