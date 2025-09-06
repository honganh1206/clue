package inference

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/tools"
	"google.golang.org/genai"
)

type LLMClient interface {
	// FIXME: VERY RESOURCE-CONSUMING since we are invoking this in every loop
	// What to do? Maintain a parallel flattened view/Flatten incrementally with new messages/Modify the engine
	RunInferenceStream(ctx context.Context, history []*message.Message, tools []*tools.ToolDefinition) (*message.Message, error)
	SummarizeHistory(history []*message.Message, threshold int) []*message.Message
	// ApplySlidingWindow(history []*message.Message, windowSize int) []*message.Message
	TruncateMessage(msg *message.Message, threshold int) *message.Message
	ProviderName() string
	// TODO: Implement these
	// ToNativeMessageStructure()
	// ToNativeToolSchema()
}

type BaseLLMClient struct {
	Provider   string
	Model      string
	TokenLimit int64
}

func Init(ctx context.Context, llm BaseLLMClient) (LLMClient, error) {
	switch llm.Provider {
	case AnthropicProvider:
		client := anthropic.NewClient() // Default to look up ANTHROPIC_API_KEY
		return NewAnthropicClient(&client, ModelVersion(llm.Model), llm.TokenLimit), nil
	case GoogleProvider:
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  os.Getenv("GEMINI_API_KEY"),
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			log.Fatal(err)
		}
		return NewGeminiClient(client, ModelVersion(llm.Model), llm.TokenLimit), nil
	default:
		return nil, fmt.Errorf("unknown model provider: %s", llm.Provider)
	}
}

func ListAvailableModels(provider ProviderName) []ModelVersion {
	switch provider {
	case AnthropicProvider:
		return []ModelVersion{
			Claude4Opus,
			Claude4Sonnet,
			Claude37Sonnet,
			Claude35Sonnet,
			Claude35Haiku,
			Claude3Opus,
			Claude3Sonnet, // FIXME: Deprecated soon
			Claude3Haiku,
		}
	case GoogleProvider:
		return []ModelVersion{
			Gemini25Pro,
			Gemini25Flash,
			Gemini20Flash,
			Gemini20FlashLite,
			Gemini15Pro,
			Gemini15Flash,
		}
	default:
		return []ModelVersion{}
	}
}

func GetDefaultModel(provider ProviderName) ModelVersion {
	switch provider {
	case AnthropicProvider:
		return Claude35Sonnet
	case GoogleProvider:
		return Gemini25Flash
	default:
		return ""
	}
}

func (b *BaseLLMClient) BaseSummarizeHistory(history []*message.Message, threshold int) []*message.Message {
	if len(history) <= threshold {
		return history
	}

	var summarizedHistory []*message.Message
	// Keep the system prompt
	summarizedHistory = append(summarizedHistory, history[0])

	// TODO: Call a smaller agent to summarize old messages?

	// Keep the most recent messages
	recentMessages := history[len(history)-threshold:]
	summarizedHistory = append(summarizedHistory, recentMessages...)

	return summarizedHistory
}

func (b *BaseLLMClient) BaseTruncateMessage(msg *message.Message, threshold int) *message.Message {
	for i, b := range msg.Content {
		if b.Type() != message.ToolResultType {
			continue
		}

		// TODO: A new parameter to specify which keys to preserve
		if toolResult, ok := b.(message.ToolResultBlock); ok {
			if len(toolResult.Content) < threshold {
				return msg
			}
			truncated := toolResult.Content[:threshold/2] +
				"\n... [TRUNCATED] ...\n" +
				toolResult.Content[len(toolResult.Content)-threshold/2:]
			msg.Content[i] = message.NewToolResultBlock(
				toolResult.ToolUseID,
				toolResult.ToolName,
				truncated,
				toolResult.IsError,
			)
		}
	}
	return msg
}
