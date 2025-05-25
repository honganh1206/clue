package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/honganh1206/clue/messages"
	"github.com/honganh1206/clue/tools"
)

type AnthropicModel struct {
	client     *anthropic.Client
	promptPath string
	model      ModelVersion
	maxTokens  int64
}

func NewAnthropicModel(client *anthropic.Client, promptPath string, model ModelVersion, maxTokens int64) *AnthropicModel {
	if model == "" {
		model = ModelVersion(anthropic.ModelClaudeSonnet4_0)
	}

	if maxTokens == 0 {
		maxTokens = 1024
	}

	return &AnthropicModel{
		client:     client,
		promptPath: promptPath,
		model:      model,
		maxTokens:  maxTokens,
	}
}

func (m *AnthropicModel) Name() string {
	return AnthropicModelName
}

func getAnthropicModel(model ModelVersion) anthropic.Model {
	switch model {
	case Claude4Opus:
		return anthropic.ModelClaudeOpus4_0
	case Claude4Sonnet:
		return anthropic.ModelClaudeSonnet4_0
	case Claude37Sonnet:
		return anthropic.ModelClaude3_7SonnetLatest
	case Claude35Sonnet:
		return anthropic.ModelClaude3_5SonnetLatest
	case Claude35Haiku:
		return anthropic.ModelClaude3_5HaikuLatest
	case Claude3Opus:
		return anthropic.ModelClaude3OpusLatest
	case Claude3Sonnet:
		// FIXME: Deprecated soon
		return anthropic.ModelClaude_3_Sonnet_20240229
	case Claude3Haiku:
		return anthropic.ModelClaude_3_Haiku_20240307
	default:
		return anthropic.ModelClaudeSonnet4_0
	}
}

func (m *AnthropicModel) RunInference(ctx context.Context, conversation []messages.MessageParam, tools []tools.ToolDefinition) (*messages.MessageResponse, error) {
	anthropicConversation := convertToAnthropicConversation(conversation)

	anthropicTools, err := convertToAnthropicTools(tools)
	if err != nil {
		return nil, fmt.Errorf("failed to convert tools: %w", err)
	}

	systemPrompt, err := m.loadPromptFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load prompt: %w", err)
	}

	anthropicStream := m.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     getAnthropicModel(m.model),
		MaxTokens: m.maxTokens,
		Messages:  anthropicConversation,
		Tools:     anthropicTools,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
	})

	response, err := streamFromAnthropicResponse(anthropicStream)
	if err != nil {
		return nil, err
	}

	return response, nil

}

// Convert generic conversation to Anthropic one
func convertToAnthropicConversation(conversation []messages.MessageParam) []anthropic.MessageParam {
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
		switch b := block.(type) {
		case messages.ToolResultContentBlock:
			content, ok := b.Content.(string)
			if !ok {
				continue
			}
			blocks = append(blocks, anthropic.NewToolResultBlock(b.ToolUseID, content, b.IsError))
		case messages.TextContentBlock:
			blocks = append(blocks, anthropic.NewTextBlock(b.Text))
		case messages.ToolUseContentBlock:
			toolParam := anthropic.ToolUseBlockParam{
				ID:    b.ID,
				Name:  b.Name,
				Input: b.Input,
			}

			toolUseBlock := anthropic.ContentBlockParamUnion{
				OfToolUse: &toolParam,
			}
			blocks = append(blocks, toolUseBlock)
		}
	}

	return blocks
}

func streamFromAnthropicResponse(stream *ssestream.Stream[anthropic.MessageStreamEventUnion]) (*messages.MessageResponse, error) {
	anthropicMsg := anthropic.Message{}

	for stream.Next() {
		event := stream.Current()
		// Weird: This does not work with list_files({})
		// Since it leads to error calling MarshalJSON for json.RawMessage: Unexpected end of JSON input
		// FIXME: Should not skip error handling here
		// if err := anthropicMsg.Accumulate(event); err != nil {
		// 	panic(err)
		// 	// return nil, fmt.Errorf("stream error mid-processing: %w", err)
		// }

		anthropicMsg.Accumulate(event)

		switch event := event.AsAny().(type) {
		// Incremental updates sent during text generation
		case anthropic.ContentBlockDeltaEvent:
			fmt.Print(event.Delta.Text)
		case anthropic.ContentBlockStartEvent:
			// if event.ContentBlock.Name != "" {
			// 	print(event.ContentBlock.Name + ": ")
			// }
		case anthropic.ContentBlockStopEvent:
			println()
		case anthropic.MessageStopEvent:
		case anthropic.MessageStartEvent:
		case anthropic.MessageDeltaEvent:
		default:
			fmt.Printf("Unhandled event type: %T\n", event)
		}
	}

	if stream.Err() != nil {
		panic(stream.Err())
	}

	return convertFromAnthropicMessage(anthropicMsg)
}

func convertFromAnthropicMessage(anthropicMsg anthropic.Message) (*messages.MessageResponse, error) {
	msg := &messages.MessageResponse{
		MessageParam: messages.MessageParam{
			Role:    messages.AssistantRole,
			Content: make([]messages.ContentBlock, 0),
		},
	}

	for _, block := range anthropicMsg.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.TextBlock:
			msg.Content = append(msg.Content, messages.NewTextContentBlock(block.Text))
		case anthropic.ToolUseBlock:
			err := json.Unmarshal([]byte(variant.JSON.Input.Raw()), &block.Input)
			if err != nil {
				return nil, err
			}
			msg.Content = append(msg.Content, messages.NewToolUseContentBlock(block.ID, block.Name, block.Input))
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

func (m *AnthropicModel) loadPromptFile() (string, error) {
	if m.promptPath == "" {
		return "", nil
	}

	data, err := os.ReadFile(m.promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file: %w", err)
	}

	return string(data), nil
}
