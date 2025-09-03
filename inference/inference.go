package inference

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/tools"
)

type Model interface {
	// FIXME: VERY RESOURCE-CONSUMING since we are invoking this in every loop
	// What to do? Maintain a parallel flattened view/Flatten incrementally with new messages/Modify the engine
	RunInference(ctx context.Context, msgs []*message.Message, tools []*tools.ToolDefinition) (*message.Message, error)
	Name() string
}

type ModelConfig struct {
	Provider  string
	Model     string
	MaxTokens int64
}

func Init(ctx context.Context, config ModelConfig) (Model, error) {
	switch config.Provider {
	case AnthropicProvider:
		client := anthropic.NewClient() // Default to look up ANTHROPIC_API_KEY
		return NewAnthropicModel(&client, ModelVersion(config.Model), config.MaxTokens), nil
	// case GoogleProvider:
	// 	client, err := genai.NewClient(ctx, &genai.ClientConfig{
	// 		APIKey:  os.Getenv("GEMINI_API_KEY"),
	// 		Backend: genai.BackendGeminiAPI,
	// 	})
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	return NewGeminiModel(client, ModelVersion(config.Model), config.MaxTokens), nil
	default:
		return nil, fmt.Errorf("unknown model provider: %s", config.Provider)
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
		return ModelVersion(anthropic.ModelClaudeSonnet4_0)
	case GoogleProvider:
		return ModelVersion(Gemini25Pro)
	default:
		return ""
	}
}
