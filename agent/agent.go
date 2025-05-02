package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/honganh1206/code-editing-agent/tools"
)

type Agent struct {
	client         *anthropic.Client
	getUserMessage func() (string, bool)
	tools          []tools.ToolDefinition
	promptPath     string
}

func New(client *anthropic.Client, getUserMsg func() (string, bool), tools []tools.ToolDefinition, promptPath string) *Agent {
	return &Agent{
		client:         client,
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
		agentMsg, err := a.runInference(ctx, conversation)
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

func (a *Agent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	// Grouping tools together in an unified interface for code, bash and text editor?
	// No need to know the internal details
	anthropicTools := []anthropic.ToolUnionParam{}
	// FIXME: A loop inside a loop in Run
	for _, tool := range a.tools {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}

	prompt, err := a.loadPromptFile()
	if err != nil {
		return nil, err
	}

	// Configurations for messages like models, modes, token count, etc.
	msg, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7SonnetLatest, // TODO: Make this more configurable
		MaxTokens: int64(1024),
		Messages:  conversation,   // Alternate between two roles: user/assistant with corresponding content
		Tools:     anthropicTools, // This will then be wrapped inside a system prompt
		System: []anthropic.TextBlockParam{
			{Text: prompt},
		},
	})

	return msg, err
}

func (a *Agent) executeTool(id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
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
		return anthropic.NewToolResultBlock(id, "tool not found", true)
	}

	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)
	response, err := toolDef.Function(input)

	if err != nil {
		return anthropic.NewToolResultBlock(id, err.Error(), true)
	}

	return anthropic.NewToolResultBlock(id, response, false)
}

func (a *Agent) loadPromptFile() (string, error) {
	if a.promptPath == "" {
		return "", nil
	}

	data, err := os.ReadFile(a.promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file: %w", err)
	}

	return string(data), nil
}
