package inference

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/honganh1206/clue/message"
)

// Mock implementations for testing
type MockAnthropicClient struct {
	mock.Mock
}

type MockGeminiClient struct {
	mock.Mock
}

// Test helpers
func setupTestEnv() func() {
	// Store original env vars
	originalAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	originalGeminiKey := os.Getenv("GEMINI_API_KEY")

	// Set test env vars
	os.Setenv("ANTHROPIC_API_KEY", "test-anthropic-key")
	os.Setenv("GEMINI_API_KEY", "test-gemini-key")

	// Return cleanup function
	return func() {
		if originalAnthropicKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", originalAnthropicKey)
		}

		if originalGeminiKey == "" {
			os.Unsetenv("GEMINI_API_KEY")
		} else {
			os.Setenv("GEMINI_API_KEY", originalGeminiKey)
		}
	}
}

func createTestMessage(role string, text string) *message.Message {
	return &message.Message{
		Role:    role,
		Content: []message.ContentBlock{message.NewTextBlock(text)},
	}
}

func createTestMessages(count int) []*message.Message {
	messages := make([]*message.Message, count)
	for i := range count {
		role := message.UserRole
		if i%2 == 1 {
			role = message.AssistantRole
		}
		messages[i] = createTestMessage(role, "Test message")
	}
	return messages
}

// Tests
func TestInit_AnthropicProvider_Success(t *testing.T) {
	cleanup := setupTestEnv()
	defer cleanup()

	llm := BaseLLMClient{
		Provider:   AnthropicProvider,
		Model:      string(Claude4Sonnet),
		TokenLimit: 4096,
	}

	client, err := Init(context.Background(), llm)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, AnthropicModelName, client.ProviderName())
}

func TestInit_GoogleProvider_Success(t *testing.T) {
	cleanup := setupTestEnv()
	defer cleanup()

	llm := BaseLLMClient{
		Provider:   GoogleProvider,
		Model:      string(Gemini25Pro),
		TokenLimit: 8192,
	}

	client, err := Init(context.Background(), llm)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, GoogleModelName, client.ProviderName())
}

func TestInit_GoogleProvider_MissingAPIKey(t *testing.T) {
	t.Skip("Skipping test that causes log.Fatal - needs proper error handling implementation")

	// Ensure GEMINI_API_KEY is not set
	originalKey := os.Getenv("GEMINI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("GEMINI_API_KEY", originalKey)
		}
	}()

	llm := BaseLLMClient{
		Provider:   GoogleProvider,
		Model:      string(Gemini25Pro),
		TokenLimit: 8192,
	}

	client, err := Init(context.Background(), llm)

	// The function calls log.Fatal which exits the program,
	// but in tests we can't easily test this behavior without modifying the code
	// For now, we'll skip this test or expect it to return an error
	// This test might need to be adjusted based on how we want to handle missing API keys
	_ = client
	_ = err
}

func TestInit_UnknownProvider(t *testing.T) {
	llm := BaseLLMClient{
		Provider:   "unknown_provider",
		Model:      "unknown_model",
		TokenLimit: 4096,
	}

	client, err := Init(context.Background(), llm)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "unknown model provider")
}

func TestListAvailableModels_AnthropicProvider(t *testing.T) {
	models := ListAvailableModels(AnthropicProvider)

	expectedModels := []ModelVersion{
		Claude4Opus,
		Claude4Sonnet,
		Claude37Sonnet,
		Claude35Sonnet,
		Claude35Haiku,
		Claude3Opus,
		Claude3Sonnet,
		Claude3Haiku,
	}

	assert.Equal(t, expectedModels, models)
	assert.Len(t, models, 8)
}

func TestListAvailableModels_GoogleProvider(t *testing.T) {
	models := ListAvailableModels(GoogleProvider)

	expectedModels := []ModelVersion{
		Gemini25Pro,
		Gemini25Flash,
		Gemini20Flash,
		Gemini20FlashLite,
		Gemini15Pro,
		Gemini15Flash,
	}

	assert.Equal(t, expectedModels, models)
	assert.Len(t, models, 6)
}

func TestListAvailableModels_UnknownProvider(t *testing.T) {
	models := ListAvailableModels("unknown_provider")

	assert.Empty(t, models)
}

func TestGetDefaultModel_AnthropicProvider(t *testing.T) {
	defaultModel := GetDefaultModel(AnthropicProvider)

	assert.Equal(t, Claude4Sonnet, defaultModel)
}

func TestGetDefaultModel_GoogleProvider(t *testing.T) {
	defaultModel := GetDefaultModel(GoogleProvider)

	assert.Equal(t, Gemini25Pro, defaultModel)
}

func TestGetDefaultModel_UnknownProvider(t *testing.T) {
	defaultModel := GetDefaultModel("unknown_provider")

	assert.Equal(t, ModelVersion(""), defaultModel)
}

func TestBaseLLMClient_BaseSummarizeHistory_BelowThreshold(t *testing.T) {
	client := &BaseLLMClient{}
	messages := createTestMessages(5)
	threshold := 10

	result := client.BaseSummarizeHistory(messages, threshold)

	assert.Equal(t, messages, result)
	assert.Len(t, result, 5)
}

func TestBaseLLMClient_BaseSummarizeHistory_ExactThreshold(t *testing.T) {
	client := &BaseLLMClient{}
	messages := createTestMessages(10)
	threshold := 10

	result := client.BaseSummarizeHistory(messages, threshold)

	assert.Equal(t, messages, result)
	assert.Len(t, result, 10)
}

func TestBaseLLMClient_BaseSummarizeHistory_AboveThreshold(t *testing.T) {
	client := &BaseLLMClient{}
	messages := createTestMessages(15)
	threshold := 5

	result := client.BaseSummarizeHistory(messages, threshold)

	// Should keep system prompt (first message) + most recent 5 messages
	// Total: 1 + 5 = 6 messages
	assert.Len(t, result, 6)

	// First message should be preserved (system prompt)
	assert.Equal(t, messages[0], result[0])

	// Last 5 messages should be preserved
	for i := range threshold {
		expectedIndex := len(messages) - threshold + i
		resultIndex := 1 + i // Skip the system prompt
		assert.Equal(t, messages[expectedIndex], result[resultIndex])
	}
}

func TestBaseLLMClient_BaseSummarizeHistory_EmptyHistory(t *testing.T) {
	client := &BaseLLMClient{}
	var messages []*message.Message
	threshold := 5

	result := client.BaseSummarizeHistory(messages, threshold)

	assert.Empty(t, result)
}

func TestBaseLLMClient_BaseSummarizeHistory_SingleMessage(t *testing.T) {
	client := &BaseLLMClient{}
	messages := createTestMessages(1)
	threshold := 5

	result := client.BaseSummarizeHistory(messages, threshold)

	assert.Equal(t, messages, result)
	assert.Len(t, result, 1)
}

func TestBaseLLMClient_BaseTruncateMessage_NoToolResults(t *testing.T) {
	client := &BaseLLMClient{}
	msg := createTestMessage(message.UserRole, "Simple text message")
	threshold := 100

	result := client.BaseTruncateMessage(msg, threshold)

	assert.Equal(t, msg, result)
}

func TestBaseLLMClient_BaseTruncateMessage_ToolResultBelowThreshold(t *testing.T) {
	client := &BaseLLMClient{}
	shortContent := "Short content"
	msg := &message.Message{
		Role: message.UserRole,
		Content: []message.ContentBlock{
			message.NewToolResultBlock("tool-1", "test_tool", shortContent, false),
		},
	}
	threshold := 100

	result := client.BaseTruncateMessage(msg, threshold)

	assert.Equal(t, msg, result)
	toolResult := result.Content[0].(message.ToolResultBlock)
	assert.Equal(t, shortContent, toolResult.Content)
}

func TestBaseLLMClient_BaseTruncateMessage_ToolResultAboveThreshold(t *testing.T) {
	client := &BaseLLMClient{}
	longContent := "This is a very long content that should be truncated because it exceeds the threshold limit and we need to keep it manageable"
	msg := &message.Message{
		Role: message.UserRole,
		Content: []message.ContentBlock{
			message.NewToolResultBlock("tool-1", "test_tool", longContent, false),
		},
	}
	threshold := 50

	result := client.BaseTruncateMessage(msg, threshold)

	toolResult := result.Content[0].(message.ToolResultBlock)
	assert.NotEqual(t, longContent, toolResult.Content)
	assert.Contains(t, toolResult.Content, "... [TRUNCATED] ...")

	// Verify the structure: first half + truncation marker + last half
	expectedFirstPart := longContent[:threshold/2]
	expectedLastPart := longContent[len(longContent)-threshold/2:]
	assert.Contains(t, toolResult.Content, expectedFirstPart)
	assert.Contains(t, toolResult.Content, expectedLastPart)
}

func TestBaseLLMClient_BaseTruncateMessage_MultipleToolResults(t *testing.T) {
	client := &BaseLLMClient{}
	longContent1 := "First very long content that needs to be truncated because it exceeds threshold"
	longContent2 := "Second very long content that also needs to be truncated for the same reason"
	shortContent := "Short"

	msg := &message.Message{
		Role: message.UserRole,
		Content: []message.ContentBlock{
			message.NewToolResultBlock("tool-1", "test_tool1", longContent1, false),
			message.NewTextBlock("Regular text block"),
			message.NewToolResultBlock("tool-2", "test_tool2", longContent2, false),
			message.NewToolResultBlock("tool-3", "test_tool3", shortContent, false),
		},
	}
	threshold := 30

	result := client.BaseTruncateMessage(msg, threshold)

	assert.Len(t, result.Content, 4)

	// First tool result should be truncated
	toolResult1 := result.Content[0].(message.ToolResultBlock)
	assert.Contains(t, toolResult1.Content, "... [TRUNCATED] ...")

	// Text block should remain unchanged
	textBlock := result.Content[1].(message.TextBlock)
	assert.Equal(t, "Regular text block", textBlock.Text)

	// Second tool result should be truncated
	toolResult2 := result.Content[2].(message.ToolResultBlock)
	assert.Contains(t, toolResult2.Content, "... [TRUNCATED] ...")

	// Third tool result should remain unchanged (below threshold)
	toolResult3 := result.Content[3].(message.ToolResultBlock)
	assert.Equal(t, shortContent, toolResult3.Content)
}

func TestBaseLLMClient_BaseTruncateMessage_PreservesToolResultProperties(t *testing.T) {
	client := &BaseLLMClient{}
	longContent := "Very long content that will be truncated to test that other properties are preserved correctly"
	msg := &message.Message{
		Role: message.UserRole,
		Content: []message.ContentBlock{
			message.NewToolResultBlock("tool-123", "special_tool", longContent, true), // IsError = true
		},
	}
	threshold := 20

	result := client.BaseTruncateMessage(msg, threshold)

	toolResult := result.Content[0].(message.ToolResultBlock)
	assert.Equal(t, "tool-123", toolResult.ToolUseID)
	assert.Equal(t, "special_tool", toolResult.ToolName)
	assert.True(t, toolResult.IsError) // Should preserve error status
	assert.Contains(t, toolResult.Content, "... [TRUNCATED] ...")
}

func TestBaseLLMClient_BaseTruncateMessage_ExactThresholdLength(t *testing.T) {
	client := &BaseLLMClient{}
	content := "Exactly fifty characters in this content string!!"
	msg := &message.Message{
		Role: message.UserRole,
		Content: []message.ContentBlock{
			message.NewToolResultBlock("tool-1", "test_tool", content, false),
		},
	}
	threshold := 50 // Exactly the same length

	result := client.BaseTruncateMessage(msg, threshold)

	toolResult := result.Content[0].(message.ToolResultBlock)
	assert.Equal(t, content, toolResult.Content) // Should not be truncated
}

// Table-driven tests for multiple scenarios
func TestListAvailableModels_AllProviders(t *testing.T) {
	tests := []struct {
		name          string
		provider      ProviderName
		expectedCount int
		shouldContain []ModelVersion
	}{
		{
			name:          "Anthropic provider",
			provider:      AnthropicProvider,
			expectedCount: 8,
			shouldContain: []ModelVersion{Claude4Sonnet, Claude35Haiku},
		},
		{
			name:          "Google provider",
			provider:      GoogleProvider,
			expectedCount: 6,
			shouldContain: []ModelVersion{Gemini25Pro, Gemini15Flash},
		},
		{
			name:          "Unknown provider",
			provider:      "nonexistent",
			expectedCount: 0,
			shouldContain: []ModelVersion{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			models := ListAvailableModels(tt.provider)

			assert.Len(t, models, tt.expectedCount)

			for _, expectedModel := range tt.shouldContain {
				assert.Contains(t, models, expectedModel)
			}
		})
	}
}

func TestGetDefaultModel_AllProviders(t *testing.T) {
	tests := []struct {
		name     string
		provider ProviderName
		expected ModelVersion
	}{
		{
			name:     "Anthropic default",
			provider: AnthropicProvider,
			expected: Claude4Sonnet,
		},
		{
			name:     "Google default",
			provider: GoogleProvider,
			expected: Gemini25Pro,
		},
		{
			name:     "Unknown provider default",
			provider: "unknown",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDefaultModel(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}
