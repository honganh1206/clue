package inference

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/honganh1206/adrift/messages"
	"github.com/honganh1206/adrift/tools"
)

type Model interface {
	// FIXME: VERY RESOURCE-CONSUMING since we are invoking this in every loop
	// What to do? Maintain a parallel flattened view/Flatten incrementally with new messages/Modify the engine
	RunInference(ctx context.Context, conversation []messages.MessageParam, tools []tools.ToolDefinition) (*messages.MessageResponse, error)
	Name() string
}

type ModelConfig struct {
	Provider   string
	PromptPath string
	Model      string
	MaxTokens  int64
	ListModel  bool
}

func Init(config ModelConfig) (Model, error) {
	switch config.Provider {
	case AnthropicProvider:
		client := anthropic.NewClient() // Default to look up ANTHROPIC_API_KEY
		return NewAnthropicModel(&client, config.PromptPath, config.Model, config.MaxTokens), nil
	default:
		return nil, fmt.Errorf("unknown model provider: %s", config.Provider)
	}
}

func GetModelForProvider(provider ProviderName, model ModelVersion) string {
	switch provider {
	case AnthropicProvider:
		return getAnthropicModel(model)
	default:
		return string(model) // Default to using the model name directly
	}
}

func ListAvailableModels(provider ProviderName) []ModelVersion {
	switch provider {
	case AnthropicProvider:
		return []ModelVersion{
			Claude37Sonnet,
			Claude35Sonnet,
			Claude35Haiku,
			Claude3Opus,
			Claude3Sonnet, // FIXME: Deprecated soon
			Claude3Haiku,
		}
	default:
		return []ModelVersion{}
	}
}

func GetDefaultModel(provider ProviderName) string {
	switch provider {
	case AnthropicProvider:
		return anthropic.ModelClaude3_7SonnetLatest
	default:
		return ""
	}
}

// formatModelsForHelp formats a list of models for help text
func FormatModelsForHelp(models []ModelVersion) string {
	if len(models) == 0 {
		return ""
	}

	modelStrings := make([]string, len(models))
	for i, model := range models {
		modelStrings[i] = string(model)
	}
	return strings.Join(modelStrings, ", ")
}

func getAnthropicModel(model ModelVersion) string {
	switch model {
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
		return anthropic.ModelClaude3_7SonnetLatest
	}
}
