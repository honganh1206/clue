package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/honganh1206/adrift/inference"
	"github.com/honganh1206/adrift/tools"
)

type Agent struct {
	engine         inference.Engine
	getUserMessage func() (string, bool)
	tools          []tools.AnthropicToolDefinition
	promptPath     string
}

func New(engine inference.Engine, getUserMsg func() (string, bool), tools []tools.AnthropicToolDefinition, promptPath string) *Agent {
	return &Agent{
		engine:         engine,
		getUserMessage: getUserMsg,
		tools:          tools,
		promptPath:     promptPath,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	conversation := []anthropic.MessageParam{}

	fmt.Println("Chat with Claude (use 'ctrl-c' to quit)")

	readUserInput := true

	for {
		if readUserInput {

			// ANSI escape code formatting to add colors and styles to terminal output for YOU
			fmt.Print("\u001b[94mYou\u001b[0m: ")
			userInput, ok := a.getUserMessage()
			if !ok {
				break
			}

			userMsg := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
			conversation = append(conversation, userMsg)
		}

		// Draw conclusions from prior knowledge
		agentMsg, err := a.engine.RunInference(ctx, conversation, a.tools)
		if err != nil {
			return err
		}

		conversation = append(conversation, agentMsg.ToParam())

		// Sort of an unified inteface for different request types i.e. text, image, document, thinking
		toolResults := []anthropic.ContentBlockParamUnion{}

		for _, content := range agentMsg.Content {
			switch content.Type {
			// TODO: Add more, could be "code"?
			case "text":
				fmt.Printf("\u001b[93mClaude\u001b[0m: %s\n", content.Text)
			case "tool_use":
				result := a.executeTool(content.ID, content.Name, content.Input)
				toolResults = append(toolResults, result)
			}
		}

		// Reset the
		if len(toolResults) == 0 {
			readUserInput = true
			continue
		}

		// Right after the tool is invoked
		// The assistant responds with the result
		readUserInput = false
		conversation = append(conversation, anthropic.NewUserMessage(toolResults...))

	}

	return nil
}

func (a *Agent) executeTool(id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
	var toolDef tools.AnthropicToolDefinition
	var found bool

	for _, tool := range a.tools {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}

	if !found {
		return anthropic.NewToolResultBlock(id, "tool not found", true)
	}

	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)
	response, err := toolDef.Function(input)

	if err != nil {
		return anthropic.NewToolResultBlock(id, err.Error(), true)
	}

	return anthropic.NewToolResultBlock(id, response, false)
}
