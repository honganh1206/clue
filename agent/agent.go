package agent

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/honganh1206/code-editing-agent/tools"
)

type Agent struct {
	client         *anthropic.Client
	getUserMessage func() (string, bool)
	tools          []tools.ToolDefinition
}

func New(client *anthropic.Client, getUserMsg func() (string, bool), tools []tools.ToolDefinition) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMsg,
		tools:          tools,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	conversation := []anthropic.MessageParam{}

	fmt.Println("Chat with Claude (use 'ctrl-c' to quit)")

	for {
		// ANSI escape code formatting to add colors and styles to terminal output for YOU
		fmt.Print("\u001b[94mYou\u001b[0m: ")
		userInput, ok := a.getUserMessage()
		if !ok {
			break
		}

		userMsg := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
		conversation = append(conversation, userMsg)

		// Draw conclusions from prior knowledge
		agentMsg, err := a.runInference(ctx, conversation)
		if err != nil {
			return err
		}

		conversation = append(conversation, agentMsg.ToParam())

		for _, content := range agentMsg.Content {
			switch content.Type {
			// TODO: Add more, could be "code"?
			case "text":
				fmt.Printf("\u001b[93mClaude\u001b[0m: %s\n", content.Text)
			}
		}

	}

	return nil
}

func (a *Agent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	// Grouping tools together in an unified interface
	// No need to know the internal details
	anthropicTools := []anthropic.ToolUnionParam{}
	for _, tool := range a.tools {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}

	msg, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7SonnetLatest, // TODO: Make this more configurable
		MaxTokens: int64(1024),
		Messages:  conversation,
		Tools:     anthropicTools, // This will then be wrapped inside a system prompt
	})

	return msg, err
}
