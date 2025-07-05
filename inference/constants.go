package inference

const (
	AnthropicModelName = "Claude"
	OpenAIModelName    = "ChatGPT"
	GoogleModelName    = "Gemini"
	MetaModelName      = "Llama"
	MistralModelName   = "Mistral"
)

const (
	AnthropicProvider = "anthropic"
)

type ProviderName string
type ModelVersion string

const (
	Claude4Opus    ModelVersion = "claude-4-opus"
	Claude4Sonnet  ModelVersion = "claude-4-sonnet"
	Claude37Sonnet ModelVersion = "claude-3-7-sonnet"
	Claude35Sonnet ModelVersion = "claude-3-5-sonnet"
	Claude35Haiku  ModelVersion = "claude-3-5-haiku"
	Claude3Opus    ModelVersion = "claude-3-opus"
	Claude3Sonnet  ModelVersion = "claude-3-sonnet"
	Claude3Haiku   ModelVersion = "claude-3-haiku"
)
