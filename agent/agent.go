package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honganh1206/clue/api"
	"github.com/honganh1206/clue/inference"
	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/server/data/conversation"
	"github.com/honganh1206/clue/tools"
)

type Agent struct {
	model          inference.Model
	getUserMessage func() (string, bool)
	tools          []tools.ToolDefinition
	promptPath     string
	conversation   *conversation.Conversation
	client         *api.Client
}

func New(model inference.Model, getUserMsg func() (string, bool), conversation *conversation.Conversation, tools []tools.ToolDefinition, promptPath string, client *api.Client) *Agent {
	return &Agent{
		model:          model,
		getUserMessage: getUserMsg,
		tools:          tools,
		promptPath:     promptPath,
		conversation:   conversation,
		client:         client,
	}
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
		return message.NewToolResultContentBlock(id, errorMsg, true)
	}

	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)
	println()

	response, err := toolDef.Function(input)

	if err != nil {
		return message.NewToolResultContentBlock(id, err.Error(), true)
	}

	return message.NewToolResultContentBlock(id, response, false)
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
