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
	GoogleProvider    = "google"
)

type ProviderName string
type ModelVersion string

const (
	// Claude
	Claude4Opus    ModelVersion = "claude-4-opus"
	Claude4Sonnet  ModelVersion = "claude-4-sonnet"
	Claude37Sonnet ModelVersion = "claude-3-7-sonnet"
	Claude35Sonnet ModelVersion = "claude-3-5-sonnet"
	Claude35Haiku  ModelVersion = "claude-3-5-haiku"
	Claude3Opus    ModelVersion = "claude-3-opus"
	Claude3Sonnet  ModelVersion = "claude-3-sonnet"
	Claude3Haiku   ModelVersion = "claude-3-haiku"
	// Gemini
	Gemini25Pro       ModelVersion = "gemini-2.5-pro"
	Gemini25Flash     ModelVersion = "gemini-2.5-flash"
	Gemini20Flash     ModelVersion = "gemini-2.0-flash"
	Gemini20FlashLite ModelVersion = "gemini-2.0-flash-lite"
	Gemini15Pro       ModelVersion = "gemini-1.5-pro"
	Gemini15Flash     ModelVersion = "gemini-1.5-flash"
)
