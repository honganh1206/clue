package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/prompts"
	"github.com/honganh1206/clue/schema"
	"github.com/honganh1206/clue/tools"
	"google.golang.org/genai"
)

type GeminiClient struct {
	BaseLLMClient
	client       *genai.Client
	model        ModelVersion
	maxTokens    int64
	contents     []*genai.Content
	tools        []*genai.Tool
	systemPrompt string
	// TODO: field for caching
}

func NewGeminiClient(client *genai.Client, model ModelVersion, maxTokens int64) *GeminiClient {
	systemPrompt := prompts.GeminiSystemPrompt()

	return &GeminiClient{
		BaseLLMClient: BaseLLMClient{
			Provider: GoogleModelName,
		},
		client:       client,
		model:        model,
		maxTokens:    maxTokens,
		systemPrompt: systemPrompt,
	}
}

func (c *GeminiClient) ProviderName() string {
	return c.BaseLLMClient.Provider
}

func getGeminiModelName(model ModelVersion) string {
	return string(model)
}

func (c *GeminiClient) SummarizeHistory(history []*message.Message, threshold int) []*message.Message {
	return c.BaseLLMClient.BaseSummarizeHistory(history, threshold)
}

func (c *GeminiClient) TruncateMessage(msg *message.Message, threshold int) *message.Message {
	return c.BaseLLMClient.BaseTruncateMessage(msg, threshold)
}
func (c *GeminiClient) RunInference(ctx context.Context, onDelta func(string), streaming bool) (*message.Message, error) {
	if len(c.contents) == 0 {
		return nil, errors.New("gemini: no messages in conversation history")
	}

	modelName := getGeminiModelName(c.model)

	config := &genai.GenerateContentConfig{
		MaxOutputTokens:   int32(c.maxTokens),
		Tools:             c.tools,
		SystemInstruction: genai.NewContentFromText(c.systemPrompt, genai.RoleUser),
	}

	if streaming {
		// Streaming mode: use GenerateContentStream
		response := c.client.Models.GenerateContentStream(ctx, modelName, c.contents, config)

		var fullText strings.Builder
		var toolCalls []message.ContentBlock
		var outputContents []*genai.Content

		msg := &message.Message{
			Role:    message.ModelRole,
			Content: make([]message.ContentBlock, 0),
		}

		for chunk, err := range response {
			if err == io.EOF {
				break
			}

			if err != nil {
				return nil, err
			}

			if len(chunk.Candidates) == 0 || chunk.Candidates[0].Content == nil {
				return nil, fmt.Errorf("no content returned")
			}

			bestCandidate := chunk.Candidates[0]
			bestContent := bestCandidate.Content

			if len(bestContent.Parts) == 0 {
				if bestCandidate.FinishReason != "" {
					outputContents = append(outputContents, bestContent)
					continue
				}
			}

			for _, p := range bestContent.Parts {
				if p.Text != "" {
					onDelta(p.Text)
					fullText.WriteString(p.Text)
				}
				if p.FunctionCall != nil {
					fc := p.FunctionCall
					inputBytes, err := json.Marshal(fc.Args)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal function args: %w", err)
					}

					toolCall := message.NewToolUseBlock(
						fc.ID,
						fc.Name,
						inputBytes,
					)
					toolCalls = append(toolCalls, toolCall)
				}
			}

			outputContents = append(outputContents, bestContent)
		}

		if fullText.String() != "" {
			msg.Content = append(msg.Content, message.NewTextBlock(fullText.String()))
		}

		msg.Content = append(msg.Content, toolCalls...)

		return msg, nil
	} else {
		// Snapshot mode: use GenerateContent
		response, err := c.client.Models.GenerateContent(ctx, modelName, c.contents, config)
		if err != nil {
			return nil, fmt.Errorf("gemini snapshot call failed: %w", err)
		}

		if len(response.Candidates) == 0 || response.Candidates[0].Content == nil {
			return nil, fmt.Errorf("no content returned")
		}

		bestContent := response.Candidates[0].Content

		msg := &message.Message{
			Role:    message.ModelRole,
			Content: make([]message.ContentBlock, 0),
		}

		var fullText strings.Builder
		var toolCalls []message.ContentBlock

		for _, p := range bestContent.Parts {
			if p.Text != "" {
				fullText.WriteString(p.Text)
			}
			if p.FunctionCall != nil {
				fc := p.FunctionCall
				inputBytes, err := json.Marshal(fc.Args)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal function args: %w", err)
				}

				toolCall := message.NewToolUseBlock(
					fc.ID,
					fc.Name,
					inputBytes,
				)
				toolCalls = append(toolCalls, toolCall)
			}
		}

		if fullText.String() != "" {
			msg.Content = append(msg.Content, message.NewTextBlock(fullText.String()))
		}

		msg.Content = append(msg.Content, toolCalls...)

		return msg, nil
	}
}

func (c *GeminiClient) ToNativeHistory(history []*message.Message) error {
	if len(history) == 0 {
		return errors.New("gemini: empty conversation history")
	}
	c.contents = make([]*genai.Content, 0, len(history))

	for _, msg := range history {
		if err := c.ToNativeMessage(msg); err != nil {
			return err
		}
	}

	return nil
}

func (c *GeminiClient) ToNativeMessage(msg *message.Message) error {
	if msg == nil {
		return errors.New("gemini: message is nil")
	}

	parts := toGeminiParts(msg.Content)
	if len(parts) == 0 {
		return errors.New("gemini: message has no content parts")
	}

	content := &genai.Content{
		Role:  msg.Role,
		Parts: parts,
	}

	c.contents = append(c.contents, content)
	return nil
}

func (c *GeminiClient) ToNativeTools(tools []*tools.ToolDefinition) error {
	if len(tools) == 0 {
		return errors.New("gemini: no tools provided")
	}

	builtinTool := &genai.Tool{
		FunctionDeclarations: make([]*genai.FunctionDeclaration, 0, len(tools)),
	}

	for _, tool := range tools {
		geminiToolFuncDec, err := toGeminiFunctionDeclaration(tool)
		if err != nil {
			return err
		}
		builtinTool.FunctionDeclarations = append(builtinTool.FunctionDeclarations, geminiToolFuncDec)
	}

	c.tools = []*genai.Tool{builtinTool}

	return nil
}

func toGeminiParts(blocks []message.ContentBlock) []*genai.Part {
	parts := make([]*genai.Part, 0, len(blocks))

	for _, block := range blocks {
		switch b := block.(type) {
		case message.TextBlock:
			parts = append(parts, genai.NewPartFromText(b.Text))
		case message.ToolUseBlock:
			var args map[string]any

			err := json.Unmarshal(b.Input, &args)
			if err != nil {
				continue
			}

			parts = append(parts, genai.NewPartFromFunctionCall(b.Name, args))
		case message.ToolResultBlock:
			response := map[string]any{"result": b.Content}

			parts = append(parts, genai.NewPartFromFunctionResponse(b.ToolName, response))
		}
	}

	return parts
}

func toGeminiFunctionDeclaration(tool *tools.ToolDefinition) (*genai.FunctionDeclaration, error) {
	params, err := schema.ConvertToGeminiSchema(tool.InputSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema to Gemini format: %w", err)
	}

	functionDecl := &genai.FunctionDeclaration{
		Name:        tool.Name,
		Description: tool.Description,
		Parameters:  params,
	}

	return functionDecl, nil
}
