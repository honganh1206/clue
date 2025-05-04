package inference

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/honganh1206/adrift/tools"
)

// Generic chat message
type Message struct {
	Role    string
	Content []ContentBlock
}

// Block of content within a message
type ContentBlock struct {
	Type     string // Different categories like text, code, tools
	Text     string
	ID       string
	Name     string
	Input    []byte
	IsError  bool
	ToolCall bool
}

type Engine interface {
	// TODO: Not generic, still tied to Anthropic
	RunInference(ctx context.Context, conversation []anthropic.MessageParam, tools []tools.AnthropicToolDefinition) (*anthropic.Message, error)
}

type EngineConfig struct {
	Type       string // "anthropic", "openai", "ollama"
	PromptPath string
	Model      string
	MaxTokens  int64
}

func CreateEngine(config EngineConfig) (Engine, error) {
	var key string

	switch config.Type {
	case "anthropic":
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
