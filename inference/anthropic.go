package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/prompts"
	"github.com/honganh1206/clue/tools"
)

type AnthropicClient struct {
	BaseLLMClient
	client       *anthropic.Client
	model        ModelVersion
	maxTokens    int64
	cache        anthropic.CacheControlEphemeralParam
	history      []anthropic.MessageParam
	tools        []anthropic.ToolUnionParam
	systemPrompt string
}

func NewAnthropicClient(client *anthropic.Client, model ModelVersion, maxTokens int64) *AnthropicClient {
	// TODO: The client should get an argument for this,
	// since we need subagents to have separate system prompts
	systemPrompt := prompts.ClaudeSystemPrompt()

	return &AnthropicClient{
		BaseLLMClient: BaseLLMClient{
			Provider: AnthropicModelName,
		},
		client:       client,
		model:        model,
		maxTokens:    maxTokens,
		cache:        anthropic.NewCacheControlEphemeralParam(),
		systemPrompt: systemPrompt,
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

func (c *AnthropicClient) RunInference(ctx context.Context, onDelta func(string), streaming bool) (*message.Message, error) {
	if len(c.history) == 0 {
		return nil, errors.New("anthropic: no messages in conversation history")
	}

	params := anthropic.MessageNewParams{
		Model:     getAnthropicModel(c.model),
		MaxTokens: c.maxTokens,
		Messages:  c.history,
		Tools:     c.tools,
		System: []anthropic.TextBlockParam{
			{Text: c.systemPrompt, CacheControl: c.cache}},
	}

	var resp *message.Message
	var runErr error

	if streaming {
		resp, runErr = c.runInferenceStream(ctx, params, onDelta)
	} else {
		resp, runErr = c.runInferenceSnapshot(ctx, params)
	}

	if runErr != nil {
		return nil, runErr
	}

	return resp, nil
}

func (c *AnthropicClient) runInferenceStream(ctx context.Context, params anthropic.MessageNewParams, onDelta func(string)) (*message.Message, error) {
	stream := c.client.Messages.NewStreaming(ctx, params)

	llmresp := anthropic.Message{}

	for stream.Next() {
		event := stream.Current()
		if err := llmresp.Accumulate(event); err != nil {
			fmt.Printf("error accumulating event: %v\n", err)
			continue
		}

		switch ev := event.AsAny().(type) {
		case anthropic.ContentBlockStartEvent:
		case anthropic.ContentBlockStopEvent:
			fmt.Println()
		case anthropic.MessageStopEvent:
			fmt.Println()
		case anthropic.MessageStartEvent:
		case anthropic.MessageDeltaEvent:
		default:
			fmt.Printf("Unhandled event type: %T\n", event)
		case anthropic.ContentBlockDeltaEvent:
			switch d := ev.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				if d.Text != "" {
					onDelta(d.Text)
				}
			}
		}
	}

	if streamErr := stream.Err(); streamErr != nil {
		var sb strings.Builder
		for _, blk := range llmresp.Content {
			switch v := blk.AsAny().(type) {
			case anthropic.TextBlock:
				sb.WriteString(v.Text)
			}
		}
		msg, err := toGenericMessage(llmresp)
		if err != nil {
			return nil, err
		}

		return msg, err
	}

	var sb strings.Builder
	for _, blk := range llmresp.Content {
		switch v := blk.AsAny().(type) {
		case anthropic.TextBlock:
			sb.WriteString(v.Text)
		case anthropic.ToolUseBlock:
			sb.WriteString(v.JSON.Input.Raw())
		}
	}

	msg, err := toGenericMessage(llmresp)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (c *AnthropicClient) runInferenceSnapshot(ctx context.Context, params anthropic.MessageNewParams) (*message.Message, error) {
	response, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic snapshot call failed: %w", err)
	}

	msg, err := toGenericMessage(*response)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (c *AnthropicClient) ToNativeHistory(history []*message.Message) error {
	if len(history) == 0 {
		return errors.New("anthropic: empty conversation history")
	}
	c.history = make([]anthropic.MessageParam, 0, len(history))

	for _, msg := range history {
		if err := c.ToNativeMessage(msg); err != nil {
			return err
		}
	}

	return nil
}

func (c *AnthropicClient) ToNativeMessage(msg *message.Message) error {
	if msg == nil {
		return errors.New("anthropic: message is nil")
	}

	var nativeMsg anthropic.MessageParam
	blocks := toAnthropicBlocks(msg.Content)
	switch msg.Role {
	case message.UserRole:
		nativeMsg = anthropic.NewUserMessage(blocks...)
	case message.AssistantRole:
		nativeMsg = anthropic.NewAssistantMessage(blocks...)
	default:
		return errors.New("anthropic: invalid message role")
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
		anthropicTool, err := toAnthropicTool(tool)
		if err != nil {
			return err
		}

		c.tools = append(c.tools, anthropicTool)
	}

	return nil
}

func toAnthropicBlocks(blocks []message.ContentBlock) []anthropic.ContentBlockParamUnion {
	// Unified interface for different request types i.e. text, image, document, thinking
	anthropicBlocks := make([]anthropic.ContentBlockParamUnion, 0, len(blocks))

	for _, block := range blocks {
		switch b := block.(type) {
		case message.ToolResultBlock:
			anthropicBlocks = append(anthropicBlocks, anthropic.NewToolResultBlock(b.ToolUseID, b.Content, b.IsError))
		case message.TextBlock:
			anthropicBlocks = append(anthropicBlocks, anthropic.NewTextBlock(b.Text))
		case message.ToolUseBlock:
			toolUseParam := anthropic.ToolUseBlockParam{
				ID:    b.ID,
				Name:  b.Name,
				Input: b.Input,
			}

			anthropicBlocks = append(anthropicBlocks, anthropic.ContentBlockParamUnion{
				OfToolUse: &toolUseParam,
			})
		}
	}

	return anthropicBlocks
}

func toGenericMessage(anthropicMsg anthropic.Message) (*message.Message, error) {
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
func toAnthropicTool(tool *tools.ToolDefinition) (anthropic.ToolUnionParam, error) {
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
