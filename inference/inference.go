package inference

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/prompts"
	"github.com/honganh1206/clue/tools"
	"google.golang.org/genai"
)

type LLMClient interface {
	// TODO: This still needs some rewrites.
	// We must separate RunInference into 2 signatures: One for snapshot and one for streaming.
	// The two signatures should share the same params, only differ in return type.
	// Refer to https://github.com/madebywelch/anthropic-go/blob/main/pkg/anthropic/client/client.go for the design.
	// The onDelta should be in agent.go, and we need to remove the streaming flag.
	RunInference(ctx context.Context, onDelta func(string), streaming bool) (*message.Message, error)
	SummarizeHistory(history []*message.Message, threshold int) []*message.Message
	// ApplySlidingWindow(history []*message.Message, windowSize int) []*message.Message
	TruncateMessage(msg *message.Message, threshold int) *message.Message
	ProviderName() string
	ToNativeHistory(history []*message.Message) error
	ToNativeMessage(msg *message.Message) error
	ToNativeTools(tools []*tools.ToolDefinition) error
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
		sysPrompt := prompts.ClaudeSystemPrompt()
		return NewAnthropicClient(&client, ModelVersion(llm.Model), llm.TokenLimit, sysPrompt), nil
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
		return Claude45Sonnet
	case GoogleProvider:
		return Gemini25Pro
	default:
		return ""
	}
}

func GetDefaultModelSubagent(provider ProviderName) ModelVersion {
	switch provider {
	case AnthropicProvider:
		return Claude45Haiku
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

	// TODO: Call a subagent to summarize old messages

	// Keep the most recent messages
	recentMessages := history[len(history)-threshold:]
	summarizedHistory = append(summarizedHistory, recentMessages...)

	return summarizedHistory
}

func (b *BaseLLMClient) BaseTruncateMessage(msg *message.Message, threshold int) *message.Message {
	for i, b := range msg.Content {
		// TODO: A new parameter to specify which keys to preserve
		// Should we add check to continue if not ToolResultBlock?
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
