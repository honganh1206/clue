package inference

const (
	AnthropicModelName = "Claude"
	GoogleModelName    = "Gemini"
)

const (
	AnthropicProvider = "anthropic"
	GoogleProvider    = "google"
)

type (
	ProviderName string
	ModelVersion string
)

const (
	// Claude
	Claude41Opus   ModelVersion = "claude-4-1-opus"
	Claude4Opus    ModelVersion = "claude-4-opus"
	Claude3Opus    ModelVersion = "claude-3-opus"
	Claude45Sonnet ModelVersion = "claude-4-5-sonnet"
	Claude4Sonnet  ModelVersion = "claude-4-sonnet"
	Claude37Sonnet ModelVersion = "claude-3-7-sonnet"
	Claude35Sonnet ModelVersion = "claude-3-5-sonnet"
	Claude3Sonnet  ModelVersion = "claude-3-sonnet"
	Claude45Haiku  ModelVersion = "claude-4-5-haiku"
	Claude35Haiku  ModelVersion = "claude-3-5-haiku"
	Claude3Haiku   ModelVersion = "claude-3-haiku"
	// Gemini
	Gemini3Pro        ModelVersion = "gemini-3-pro-preview"
	Gemini25Pro       ModelVersion = "gemini-2.5-pro"
	Gemini25Flash     ModelVersion = "gemini-2.5-flash"
	Gemini20Flash     ModelVersion = "gemini-2.0-flash"
	Gemini20FlashLite ModelVersion = "gemini-2.0-flash-lite"
	Gemini15Pro       ModelVersion = "gemini-1.5-pro"
	Gemini15Flash     ModelVersion = "gemini-1.5-flash"
)
