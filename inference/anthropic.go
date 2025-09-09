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

type AnthropicClient struct {
	BaseLLMClient
	client    *anthropic.Client
	model     ModelVersion
	maxTokens int64
	cache     anthropic.CacheControlEphemeralParam
	history   []anthropic.MessageParam
	tools     []anthropic.ToolUnionParam
	// TODO: Field for system prompt
}

func NewAnthropicClient(client *anthropic.Client, model ModelVersion, maxTokens int64) *AnthropicClient {
	return &AnthropicClient{
		client:    client,
		model:     model,
		maxTokens: maxTokens,
		cache:     anthropic.NewCacheControlEphemeralParam(),
	}
}

func (c *AnthropicClient) ProviderName() string {
	return c.BaseLLMClient.Provider
}

func (c *AnthropicClient) SummarizeHistory(history []*message.Message, threshold int) []*message.Message {
	return c.BaseLLMClient.BaseSummarizeHistory(history, threshold)
}

func (c *AnthropicClient) TruncateMessage(msg *message.Message, threshold int) *message.Message {
	return c.BaseLLMClient.BaseTruncateMessage(msg, threshold)
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

func (c *AnthropicClient) RunInferenceStream(ctx context.Context) (*message.Message, error) {
	// TODO: This should be called once only
	systemPrompt := prompts.ClaudeSystemPrompt()

	// Optimize system prompt for caching - split into cacheable and dynamic parts
	anthropicStream := c.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     getAnthropicModel(c.model),
		MaxTokens: c.maxTokens,
		Messages:  c.history,
		Tools:     c.tools,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt, CacheControl: c.cache}},
	})

	response, err := streamAnthropicResponse(anthropicStream)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *AnthropicClient) ToNativeHistory(history []*message.Message) error {
	if len(history) == 0 {
		// TODO: Add custom error
		return nil
	}
	c.history = make([]anthropic.MessageParam, 0, len(history))

	for _, msg := range history {
		// TOD): Error handling
		_ = c.ToNativeMessage(msg)
	}

	return nil
}

func (c *AnthropicClient) ToNativeMessage(msg *message.Message) error {
	// TODO: Add null check
	var nativeMsg anthropic.MessageParam
	blocks := convertToAnthropicBlocks(msg.Content)
	switch msg.Role {
	case message.UserRole:
		nativeMsg = anthropic.NewUserMessage(blocks...)
	case message.AssistantRole:
		nativeMsg = anthropic.NewAssistantMessage(blocks...)
	}

	c.history = append(c.history, nativeMsg)
	return nil
}

func (c *AnthropicClient) ToNativeTools(tools []*tools.ToolDefinition) error {
	if len(tools) == 0 {
		return nil
	}

	c.tools = make([]anthropic.ToolUnionParam, 0, len(tools))

	for _, tool := range tools {
		anthropicTool, err := convertToAnthropicTool(tool)
		if err != nil {
			return err
		}

		c.tools = append(c.tools, anthropicTool)
	}

	return nil
}

func convertToAnthropicBlocks(blocks []message.ContentBlock) []anthropic.ContentBlockParamUnion {
	// Unified interface for different request types i.e. text, image, document, thinking
	anthropicBlocks := make([]anthropic.ContentBlockParamUnion, 0, len(blocks))

	for _, block := range blocks {
		anthropicBlocks = append(anthropicBlocks, block.ToAnthropic())
	}

	return anthropicBlocks
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
		Content: make([]message.ContentBlock, 0)}

	for _, block := range anthropicMsg.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.TextBlock:
			msg.Content = append(msg.Content, message.NewTextBlock(block.Text))
		case anthropic.ToolUseBlock:
			err := json.Unmarshal([]byte(variant.JSON.Input.Raw()), &block.Input)
			if err != nil {
				return nil, err
			}
			msg.Content = append(msg.Content, message.NewToolUseBlock(block.ID, block.Name, block.Input))
		}
	}

	return msg, nil
}

// Convert generic schema to Anthropic schema
func convertToAnthropicTool(tool *tools.ToolDefinition) (anthropic.ToolUnionParam, error) {
	schema, err := json.Marshal(tool.InputSchema)
	if err != nil {
		// return nil, err
	}

	var anthropicSchema anthropic.ToolInputSchemaParam
	if err := json.Unmarshal(schema, &anthropicSchema); err != nil {
		// return nil, fmt.Errorf("failed to unmarshal to Anthropic schema: %w", err)
	}

	// Grouping tools together in an unified interface for code, bash and text editor?
	// No need to know the internal details
	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        tool.Name,
			Description: anthropic.String(tool.Description),
			InputSchema: anthropicSchema,
			// CacheControl: anthropic.NewCacheControlEphemeralParam(),
		},
	}, nil
}
