package inference

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/honganh1206/adrift/messages"
	"github.com/honganh1206/adrift/tools"
)

type Engine interface {
	RunInference(ctx context.Context, conversation []messages.Message, tools []tools.ToolDefinition) (*messages.Message, error)
	Name() string
}

type EngineConfig struct {
	Type       string // "anthropic", "openai", "ollama"
	PromptPath string
	Model      string
	MaxTokens  int64
	ListModel  bool
}

func CreateEngine(config EngineConfig) (Engine, error) {
	var key string

	switch config.Type {
	case AnthropicProvider:
		key = os.Getenv("ANTHROPIC_API_KEY")
		client := anthropic.NewClient(option.WithAPIKey(key))
		return NewAnthropicEngine(&client, config.PromptPath, config.Model, config.MaxTokens), nil

	// case "openai":
	// 	apiKey := os.Getenv("OPENAI_API_KEY")
	// 	if apiKey == "" {
	// 		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	// 	}
	// 	client := openai.NewClient(apiKey)
	// 	return NewOpenAIEngine(client, config.PromptPath, config.Model, int(config.MaxTokens)), nil

	// case "ollama":
	// 	// Implement for Ollama
	// 	return nil, fmt.Errorf("ollama engine not implemented yet")

	default:
		return nil, fmt.Errorf("unknown engine type: %s", config.Type)
	}
}

func GetModelForProvider(provider Provider, model Model) string {
	switch provider {
	case AnthropicProvider:
		return getAnthropicModel(model)
	default:
		return string(model) // Default to using the model name directly
	}
}

func ListAvailableModels(provider Provider) []Model {
	switch provider {
	case AnthropicProvider:
		return []Model{
			Claude37Sonnet,
			Claude35Sonnet,
			Claude35Haiku,
			Claude3Opus,
			Claude3Sonnet, // FIXME: Deprecated soon
			Claude3Haiku,
		}
	default:
		return []Model{}
	}
}

func GetDefaultModel(provider Provider) string {
	switch provider {
	case AnthropicProvider:
		return anthropic.ModelClaude3_7SonnetLatest
	default:
		return ""
	}
}

// formatModelsForHelp formats a list of models for help text
func FormatModelsForHelp(models []Model) string {
	if len(models) == 0 {
		return ""
	}

	modelStrings := make([]string, len(models))
	for i, model := range models {
		modelStrings[i] = string(model)
	}
	return strings.Join(modelStrings, ", ")
}

func getAnthropicModel(model Model) string {
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
