package inference

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/honganh1206/code-editing-agent/tools"
)

type AnthropicEngine struct {
	client     *anthropic.Client
	promptPath string
	model      string
	maxTokens  int64
}

func NewAnthropicEngine(client *anthropic.Client, promptPath string, model string, maxTokens int64) *AnthropicEngine {
	if model == "" {
		model = anthropic.ModelClaude3_7SonnetLatest
	}

	if maxTokens == 0 {
		maxTokens = 1024
	}

	return &AnthropicEngine{
		client:     client,
		promptPath: promptPath,
		model:      model,
		maxTokens:  maxTokens,
	}
}

func (e *AnthropicEngine) RunInference(ctx context.Context, conversation []anthropic.MessageParam, tools []tools.AnthropicToolDefinition) (*anthropic.Message, error) {
	// Grouping tools together in an unified interface for code, bash and text editor?
	// No need to know the internal details
	anthropicTools := []anthropic.ToolUnionParam{}

	// FIXME: A loop inside a loop in Run
	for _, tool := range tools {

		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}

	prompt, err := e.loadPromptFile()
	if err != nil {
		return nil, err
	}

	// Configurations for messages like models, modes, token count, etc.
	msg, err := e.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     e.model, // TODO: Make this more configurable
		MaxTokens: e.maxTokens,
		Messages:  conversation,   // Alternate between two roles: user/assistant with corresponding content
		Tools:     anthropicTools, // This will then be wrapped inside a system prompt
		System: []anthropic.TextBlockParam{
			{Text: prompt},
		},
	})

	return msg, err
}

// TODO: This should be for all engines to use
func (e *AnthropicEngine) loadPromptFile() (string, error) {
	if e.promptPath == "" {
		return "", nil
	}

	data, err := os.ReadFile(e.promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file: %w", err)
	}

	return string(data), nil
}
