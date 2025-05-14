package inference

const (
	AnthropicEngineName = "Claude"
	OpenAIEngine        = "ChatGPT"
)

const (
	AnthropicProvider = "anthropic"
)

type Provider string
type Model string

const (
	Claude37Sonnet Model = "claude-3-7-sonnet"
	Claude35Sonnet Model = "claude-3-5-sonnet"
	Claude35Haiku  Model = "claude-3-5-haiku"
	Claude3Opus    Model = "claude-3-opus"
	Claude3Sonnet  Model = "claude-3-sonnet"
	Claude3Haiku   Model = "claude-3-haiku"
)
