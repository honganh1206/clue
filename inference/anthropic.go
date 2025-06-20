package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/honganh1206/clue/prompts"
	"github.com/honganh1206/clue/server/conversation"
	"github.com/honganh1206/clue/tools"
)

type AnthropicModel struct {
	client    *anthropic.Client
	model     ModelVersion
	maxTokens int64
	cache     anthropic.CacheControlEphemeralParam
}

func NewAnthropicModel(client *anthropic.Client, model ModelVersion, maxTokens int64) *AnthropicModel {
	if model == "" {
		model = ModelVersion(anthropic.ModelClaudeSonnet4_0)
	}

	if maxTokens == 0 {
		maxTokens = 1024
	}

	return &AnthropicModel{
		client:    client,
		model:     model,
		maxTokens: maxTokens,
		cache:     anthropic.NewCacheControlEphemeralParam(),
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

func (m *AnthropicModel) CompleteStream(ctx context.Context, msgs []*conversation.MessageParam, tools []tools.ToolDefinition) (*conversation.MessageResponse, error) {
	anthropicMsgs := convertToAnthropicMsgs(msgs)

	anthropicTools, err := m.convertToAnthropicTools(tools)
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
func convertToAnthropicMsgs(msgs []*conversation.MessageParam) []anthropic.MessageParam {
	anthropicMsgs := make([]anthropic.MessageParam, 0, len(msgs))

	for _, msg := range msgs {

		var anthropicMsg anthropic.MessageParam

		blocks := convertToAnthropicBlocks(msg.Content)

		if msg.Role == conversation.UserRole {
			anthropicMsg = anthropic.NewUserMessage(blocks...)
		} else if msg.Role == conversation.AssistantRole {
			anthropicMsg = anthropic.NewAssistantMessage(blocks...)
		}

		anthropicMsgs = append(anthropicMsgs, anthropicMsg)

		continue

	}

	return anthropicMsgs
}

func convertToAnthropicBlocks(genericBlocks []conversation.ContentBlock) []anthropic.ContentBlockParamUnion {
	// Unified inteface for different request types i.e. text, image, document, thinking
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(genericBlocks))

	for _, block := range genericBlocks {
		switch b := block.(type) {
		case conversation.ToolResultContentBlock:
			content, ok := b.Content.(string)
			if !ok {
				continue
			}
			toolResultBlock := anthropic.NewToolResultBlock(b.ToolUseID, content, b.IsError)
			blocks = append(blocks, toolResultBlock)
		case conversation.TextContentBlock:
			blocks = append(blocks, anthropic.NewTextBlock(b.Text))
		case conversation.ToolUseContentBlock:
			toolUseParam := anthropic.ToolUseBlockParam{
				ID:    b.ID,
				Name:  b.Name,
				Input: b.Input,
				// Maximum of 4 blocks with cache_control
				// CacheControl: anthropic.NewCacheControlEphemeralParam(),
			}

			toolUseBlock := anthropic.ContentBlockParamUnion{
				OfToolUse: &toolUseParam,
			}
			blocks = append(blocks, toolUseBlock)
		}
	}

	return blocks
}

func streamAnthropicResponse(stream *ssestream.Stream[anthropic.MessageStreamEventUnion]) (*conversation.MessageResponse, error) {
	anthropicMsg := anthropic.Message{}

	for stream.Next() {
		event := stream.Current()
		if err := anthropicMsg.Accumulate(event); err != nil {
			fmt.Printf("error accumulating event: %v\n", err)
			continue
		}

		switch event := event.AsAny().(type) {
		// Incremental updates sent during text generation
		case anthropic.ContentBlockDeltaEvent:
			print(event.Delta.Text)
		case anthropic.ContentBlockStartEvent:
		case anthropic.ContentBlockStopEvent:
			println()
		case anthropic.MessageStopEvent:
			println()
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
			println(string(apierr.DumpRequest(true)))  // Prints the serialized HTTP request
			println(string(apierr.DumpResponse(true))) // Prints the serialized HTTP response
		}
		panic(err)
	}

	return convertFromAnthropicMessage(anthropicMsg)
}

func convertFromAnthropicMessage(anthropicMsg anthropic.Message) (*conversation.MessageResponse, error) {
	msg := &conversation.MessageResponse{
		MessageParam: conversation.MessageParam{
			Role:    conversation.AssistantRole,
			Content: make([]conversation.ContentBlock, 0),
		},
	}

	for _, block := range anthropicMsg.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.TextBlock:
			msg.Content = append(msg.Content, conversation.NewTextContentBlock(block.Text))
		case anthropic.ToolUseBlock:
			err := json.Unmarshal([]byte(variant.JSON.Input.Raw()), &block.Input)
			if err != nil {
				return nil, err
			}
			msg.Content = append(msg.Content, conversation.NewToolUseContentBlock(block.ID, block.Name, block.Input))
		}
	}

	return msg, nil
}

func (m *AnthropicModel) convertToAnthropicTools(tools []tools.ToolDefinition) ([]anthropic.ToolUnionParam, error) {
	anthropicTools := make([]anthropic.ToolUnionParam, 0, len(tools))

	for _, tool := range tools {
		anthropicTool, err := m.convertToAnthropicTool(tool)
		if err != nil {
			return nil, err
		}

		anthropicTools = append(anthropicTools, *anthropicTool)
	}

	return anthropicTools, nil
}

// Convert generic schema to Anthropic schema
func (m *AnthropicModel) convertToAnthropicTool(tool tools.ToolDefinition) (*anthropic.ToolUnionParam, error) {
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
			Name:         tool.Name,
			Description:  anthropic.String(tool.Description),
			InputSchema:  anthropicSchema,
			CacheControl: m.cache,
		},
	}, nil
}
