package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/honganh1206/adrift/messages"
	"github.com/honganh1206/adrift/tools"
)

type AnthropicEngine struct {
	client     *anthropic.Client
	promptPath string
	model      string
	maxTokens  int64
}

type AnthropicToolDefinition struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error)
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

func (e *AnthropicEngine) Name() string {
	return AnthropicEngineName
}

func (e *AnthropicEngine) RunInference(ctx context.Context, conversation []messages.Message, tools []tools.ToolDefinition) (*messages.Message, error) {
	anthropicConversation := convertToAnthropicConversation(conversation)

	anthropicTools, err := convertToAnthropicTools(tools)
	if err != nil {
		return nil, err
	}

	systemPrompt, err := e.loadPromptFile()
	if err != nil {
		return nil, err
	}

	anthropicResponse, err := e.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     e.model,
		MaxTokens: e.maxTokens,
		Messages:  anthropicConversation,
		Tools:     anthropicTools,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
	})

	if err != nil {
		panic(err)
	}

	response, err := convertFromAnthropicMessages(anthropicResponse)
	if err != nil {
		return nil, err
	}

	return response, nil

}

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

// Convert generic requests to Anthropic messages
func convertToAnthropicConversation(conversation []messages.Message) []anthropic.MessageParam {

	anthropicMessages := make([]anthropic.MessageParam, 0, len(conversation))

	for _, msg := range conversation {

		var anthropicMsg anthropic.MessageParam

		blocks := convertToAnthropicBlocks(msg.Content)

		if msg.Role == messages.UserRole {
			anthropicMsg = anthropic.NewUserMessage(blocks...)
		} else if msg.Role == messages.AssistantRole {
			anthropicMsg = anthropic.NewAssistantMessage(blocks...)
		}

		anthropicMessages = append(anthropicMessages, anthropicMsg)

		continue

	}

	return anthropicMessages
}

func convertToAnthropicBlocks(genericBlocks []messages.ContentBlock) []anthropic.ContentBlockParamUnion {
	// Sort of an unified inteface for different request types i.e. text, image, document, thinking
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(genericBlocks))

	for _, block := range genericBlocks {
		switch block.Type {
		case messages.ToolResultType:
			blocks = append(blocks, anthropic.NewToolResultBlock(block.ID, block.Text, block.IsError))
		case messages.TextType:
			blocks = append(blocks, anthropic.NewTextBlock(block.Text))
		case messages.ToolUseType:
			// ← IMPORTANT: For tool use blocks, we maintain the ID as is
			// Create a tool use block (this will be handled by the SDK's ToParam method)

			var inputObj any // FIXME: No concrete typing prevents compile-time optimization
			// Input, for example read_file, could be the path to the file to be read
			if err := json.Unmarshal(block.Input, &inputObj); err != nil {
				inputObj = map[string]any{} // FIXME: Silent error handling
			}

			// FIXME: Consider sync.Pool to reuse toolParam and toolUseBlock
			toolParam := anthropic.ToolUseBlockParam{
				ID:    block.ID,
				Name:  block.Name,
				Input: inputObj,
			}

			toolUseBlock := anthropic.ContentBlockParamUnion{
				OfRequestToolUseBlock: &toolParam,
			}
			blocks = append(blocks, toolUseBlock)
		}
	}

	return blocks
}

// Convert Anthropic responses to generic messages
func convertFromAnthropicMessages(response *anthropic.Message) (*messages.Message, error) {
	msg := &messages.Message{
		Role:    string(response.Role), // Always assistant
		Content: make([]messages.ContentBlock, 0, len(response.Content)),
	}

	for _, block := range response.Content {

		switch block.Type {
		case messages.TextType:
			msg.Content = append(msg.Content, messages.ContentBlock{
				Type: messages.TextType,
				Text: block.Text,
			})
		case messages.ToolUseType:
			input, err := json.Marshal(block.Input)
			if err != nil {
				return nil, err
			}

			msg.Content = append(msg.Content, messages.ContentBlock{
				Type:     messages.ToolUseType,
				ID:       block.ID,
				Name:     block.Name,
				Input:    input,
				Text:     block.Text,
				ToolCall: true,
			})
		}
	}

	return msg, nil
}

func convertToAnthropicTools(tools []tools.ToolDefinition) ([]anthropic.ToolUnionParam, error) {
	anthropicTools := make([]anthropic.ToolUnionParam, 0, len(tools))

	for _, tool := range tools {
		anthropicTool, err := convertToAnthropicTool(tool)
		if err != nil {
			return nil, err
		}

		anthropicTools = append(anthropicTools, *anthropicTool)
	}

	return anthropicTools, nil
}

// Convert generic schema to Anthropic schema
func convertToAnthropicTool(tool tools.ToolDefinition) (*anthropic.ToolUnionParam, error) {
	schemaBytes, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return nil, err
	}

	var anthropicSchema anthropic.ToolInputSchemaParam
	json.Unmarshal(schemaBytes, &anthropicSchema)

	// Grouping tools together in an unified interface for code, bash and text editor?
	// No need to know the internal details
	return &anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        tool.Name,
			Description: anthropic.String(tool.Description),
			InputSchema: anthropicSchema,
		},
	}, nil
}
