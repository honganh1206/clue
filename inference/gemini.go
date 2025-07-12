package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"

	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/tools"
	"google.golang.org/genai"
)

type GeminiModel struct {
	client    *genai.Client
	model     ModelVersion
	maxTokens int64
}

func NewGeminiModel(client *genai.Client, model ModelVersion, maxTokens int64) *GeminiModel {
	return &GeminiModel{
		client:    client,
		model:     model,
		maxTokens: maxTokens,
	}
}

func (m *GeminiModel) Name() string {
	return GoogleModelName
}

func getGeminiModelName(model ModelVersion) string {
	return string(model)
}

func (m *GeminiModel) CompleteStream(ctx context.Context, msgs []*message.Message, tools []tools.ToolDefinition) (*message.Message, error) {
	contents := convertToGeminiContents(msgs)
	geminiTools, err := convertToGeminiTools(tools)
	if err != nil {
		return nil, fmt.Errorf("failed to convert tools: %w", err)
	}

	// TODO: Replace with GeminiSystemPrompt
	// sysPrompt := prompts.ClaudeSystemPrompt()
	modelName := getGeminiModelName(m.model)

	config := &genai.GenerateContentConfig{
		MaxOutputTokens: int32(m.maxTokens),
		Tools:           geminiTools,
	}

	// Add system prompt as the first message if provided
	// if sysPrompt != "" {
	// 	systemContent := &genai.Content{
	// 		Role:  "system",
	// 		Parts: []genai.Part{genai.Text(sysPrompt)},
	// 	}
	// 	contents = append([]*genai.Content{systemContent}, contents...)
	// }

	// Generate content stream using an iterator.Seq2 for K-V pairs
	// TODO: Do we need to create a shallow copy? so not to modify the original content?c
	iter := m.client.Models.GenerateContentStream(ctx, modelName, contents, config)

	response, err := streamGeminiResponse(iter)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Convert generic messages to Gemini contents
func convertToGeminiContents(msgs []*message.Message) []*genai.Content {
	contents := make([]*genai.Content, 0, len(msgs))

	for _, msg := range msgs {
		parts := convertToGeminiParts(msg.Content)
		if len(parts) == 0 {
			continue
		}

		content := &genai.Content{
			Role:  msg.Role,
			Parts: parts,
		}

		contents = append(contents, content)
	}

	return contents
}

// Convert content blocks to Gemini parts
func convertToGeminiParts(blocksUnion []message.ContentBlockUnion) []*genai.Part {
	parts := make([]*genai.Part, 0, len(blocksUnion))

	for _, block := range blocksUnion {
		var part *genai.Part
		switch block.Type {
		case message.TextType:
			if block.OfTextBlock != nil {
				part = &genai.Part{Text: block.OfTextBlock.Text}
				parts = append(parts, part)
			}
		case message.ToolUseType:
			if block.OfToolUseBlock != nil {
				var toolUseArgs map[string]any
				err := json.Unmarshal(block.OfToolUseBlock.Input, &toolUseArgs)
				if err != nil {
					// TODO: Proper error handling
					continue
				}
				part = &genai.Part{FunctionCall: &genai.FunctionCall{
					ID:   block.OfToolUseBlock.ID,
					Name: block.OfToolUseBlock.Name,
					Args: toolUseArgs,
				}}
				parts = append(parts, part)
			}
		case message.ToolResultType:
			if block.OfToolResultBlock != nil {
				var toolResponse map[string]any
				// TODO: No check ok type conversion here
				toolResponse, _ = block.OfToolResultBlock.Content.(map[string]any)
				// if ok {
				// 	continue
				// }
				part = &genai.Part{FunctionCall: &genai.FunctionCall{
					ID:   block.OfToolUseBlock.ID,
					Name: block.OfToolUseBlock.Name,
					Args: toolResponse,
				}}
				parts = append(parts, part)
			}
		}
	}

	return parts
}

// Convert generic tools to Gemini tools
func convertToGeminiTools(tools []tools.ToolDefinition) ([]*genai.Tool, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	geminiTools := make([]*genai.Tool, 0, len(tools))

	for _, tool := range tools {
		geminiTool, err := convertToGeminiTool(tool)
		if err != nil {
			return nil, err
		}
		geminiTools = append(geminiTools, geminiTool)
	}

	return geminiTools, nil
}

// Convert single tool to Gemini tool
func convertToGeminiTool(tool tools.ToolDefinition) (*genai.Tool, error) {
	// Convert the tool schema to Gemini format
	schemaBytes, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool schema: %w", err)
	}

	var schema genai.Schema
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to Gemini schema: %w", err)
	}

	functionDecl := &genai.FunctionDeclaration{
		Name:        tool.Name,
		Description: tool.Description,
		Parameters:  &schema,
	}

	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{functionDecl},
	}, nil
}

// Stream Gemini response and convert to our message format
func streamGeminiResponse(iter iter.Seq2[*genai.GenerateContentResponse, error]) (*message.Message, error) {
	var fullText string
	var toolCalls []message.ContentBlockUnion

	for resp, err := range iter {
		if err != nil {
			// Check if we've reached the end of the stream
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error streaming response: %w", err)
		}

		// Handle different response types
		if resp.Candidates != nil {
			for _, candidate := range resp.Candidates {
				if candidate.Content != nil {
					for _, part := range candidate.Content.Parts {
						if part.Text != "" {
							text := part.Text
							fmt.Print(text)
							fullText += text
						}
						if part.FunctionCall != nil {
							funcCall := part.FunctionCall
							inputBytes, err := json.Marshal(funcCall.Args)
							if err != nil {
								return nil, fmt.Errorf("failed to marshal function args: %w", err)
							}

							toolCall := message.NewToolUseContentBlock(
								fmt.Sprintf("call_%s", funcCall.Name),
								funcCall.Name,
								inputBytes,
							)
							toolCalls = append(toolCalls, toolCall)
						}
					}
				}
			}
		}
	}

	// Create response message
	msg := &message.Message{
		Role:    message.AssistantRole,
		Content: make([]message.ContentBlockUnion, 0),
	}

	// Add text content if any
	if fullText != "" {
		msg.Content = append(msg.Content, message.NewTextContentBlock(fullText))
	}

	// Add tool calls if any
	msg.Content = append(msg.Content, toolCalls...)

	return msg, nil
}
