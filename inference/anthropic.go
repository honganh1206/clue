package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/prompts"
	"github.com/honganh1206/clue/tools"
)

type AnthropicModel struct {
	client    *anthropic.Client
	model     ModelVersion
	maxTokens int64
	cache     anthropic.CacheControlEphemeralParam
}

func NewAnthropicModel(client *anthropic.Client, model ModelVersion, maxTokens int64) *AnthropicModel {
	return &AnthropicModel{
		client:    client,
		model:     model,
		maxTokens: maxTokens,
		// Maximum of 4 blocks ~ 256 bytes?
		cache: anthropic.NewCacheControlEphemeralParam(),
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
	case Claude3Haiku:
		return anthropic.ModelClaude_3_Haiku_20240307
	default:
		return anthropic.ModelClaudeSonnet4_0
	}
}

func (m *AnthropicModel) CompleteStream(ctx context.Context, msgs []*message.Message, tools []tools.ToolDefinition) (*message.Message, error) {
	anthropicMsgs := convertToAnthropicMsgs(msgs)

	anthropicTools, err := convertToAnthropicTools(tools)
	if err != nil {
		return nil, fmt.Errorf("failed to convert tools: %w", err)
	}

	systemPrompt := prompts.ClaudeSystemPrompt()

	anthropicStream := m.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     getAnthropicModel(m.model),
		MaxTokens: m.maxTokens,
		Messages:  anthropicMsgs,
		Tools:     anthropicTools,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt, CacheControl: m.cache}},
	})

	response, err := streamAnthropicResponse(anthropicStream)
	if err != nil {
		return nil, err
	}

	return response, nil

}

// Convert generic messages to Anthropic ones
func convertToAnthropicMsgs(msgs []*message.Message) []anthropic.MessageParam {
	anthropicMsgs := make([]anthropic.MessageParam, 0, len(msgs))

	for _, msg := range msgs {

		var anthropicMsg anthropic.MessageParam

		blocks := convertToAnthropicBlocks(msg.Content)

		switch msg.Role {
		case message.UserRole:
			anthropicMsg = anthropic.NewUserMessage(blocks...)
		case message.AssistantRole:
			anthropicMsg = anthropic.NewAssistantMessage(blocks...)
		}

		anthropicMsgs = append(anthropicMsgs, anthropicMsg)

		continue

	}

	return anthropicMsgs
}

func convertToAnthropicBlocks(blocksUnion []message.ContentBlockUnion) []anthropic.ContentBlockParamUnion {
	// Unified inteface for different request types i.e. text, image, document, thinking
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(blocksUnion))

	for _, block := range blocksUnion {
		switch block.Type {
		case message.ToolResultType:
			if block.OfToolResultBlock != nil {
				content, ok := block.OfToolResultBlock.Content.(string)
				if !ok {
					// Not sure this is the right way to handle things?
					continue
				}
				toolResultBlock := anthropic.NewToolResultBlock(block.OfToolResultBlock.ToolUseID, content, block.OfToolResultBlock.IsError)
				blocks = append(blocks, toolResultBlock)
			}
		case message.TextType:
			if block.OfTextBlock != nil {
				blocks = append(blocks, anthropic.NewTextBlock(block.OfTextBlock.Text))
			}
		case message.ToolUseType:
			if block.OfToolUseBlock != nil {
				toolUseParam := anthropic.ToolUseBlockParam{
					ID:    block.OfToolUseBlock.ID,
					Name:  block.OfToolUseBlock.Name,
					Input: block.OfToolUseBlock.Input,
				}

				toolUseBlock := anthropic.ContentBlockParamUnion{
					OfToolUse: &toolUseParam,
				}
				blocks = append(blocks, toolUseBlock)
			}
		}
	}

	return blocks
}

func streamAnthropicResponse(stream *ssestream.Stream[anthropic.MessageStreamEventUnion]) (*message.Message, error) {
	anthropicMsg := anthropic.Message{}

	for stream.Next() {
		event := stream.Current()
		if err := anthropicMsg.Accumulate(event); err != nil {
			fmt.Printf("error accumulating event: %v\n", err)
			continue
		}

		switch event := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			print(event.Delta.Text)
		case anthropic.ContentBlockStartEvent:
		case anthropic.ContentBlockStopEvent:
			fmt.Println()
		case anthropic.MessageStopEvent:
			fmt.Println()
		case anthropic.MessageStartEvent:
		case anthropic.MessageDeltaEvent:
		default:
			fmt.Printf("Unhandled event type: %T\n", event)
		}
	}

	if err := stream.Err(); err != nil {
		// TODO: Make the agent retry the operation instead
		// The tokens must flow
		var apierr *anthropic.Error
		if errors.As(err, &apierr) {
			println(string(apierr.DumpResponse(true))) // Prints the serialized HTTP response
		}
		panic(err)
	}

	return convertFromAnthropicMessage(anthropicMsg)
}

func convertFromAnthropicMessage(anthropicMsg anthropic.Message) (*message.Message, error) {
	msg := &message.Message{
		Role:    message.AssistantRole,
		Content: make([]message.ContentBlockUnion, 0)}

	for _, block := range anthropicMsg.Content {
		var v message.ContentBlockUnion
		switch variant := block.AsAny().(type) {
		case anthropic.TextBlock:
			v = message.NewTextContentBlock(block.Text)
			msg.Content = append(msg.Content, v)
		case anthropic.ToolUseBlock:
			err := json.Unmarshal([]byte(variant.JSON.Input.Raw()), &block.Input)
			if err != nil {
				return nil, err
			}
			v = message.NewToolUseContentBlock(block.ID, block.Name, block.Input)
			msg.Content = append(msg.Content, v)
		}
	}

	return msg, nil
}

func convertToAnthropicTools(tools []tools.ToolDefinition) ([]anthropic.ToolUnionParam, error) {
	if len(tools) == 0 {
		return nil, nil
	}

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
	schema, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return nil, err
	}

	var anthropicSchema anthropic.ToolInputSchemaParam
	if err := json.Unmarshal(schema, &anthropicSchema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to Anthropic schema: %w", err)
	}

	// Grouping tools together in an unified interface for code, bash and text editor?
	// No need to know the internal details
	return &anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        tool.Name,
			Description: anthropic.String(tool.Description),
			InputSchema: anthropicSchema,
			// CacheControl: m.cache,
		},
	}, nil
}
