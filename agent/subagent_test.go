package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/tools"
)

// Test helpers for subagent
func createTestSubagent() (*Subagent, *MockLLMClient) {
	mockLLM := &MockLLMClient{}
	toolBox := &tools.ToolBox{
		Tools: []*tools.ToolDefinition{
			{
				Name:        "test_tool",
				Description: "A test tool for subagent",
				Function: func(input tools.ToolInput) (string, error) {
					return "subagent test result", nil
				},
			},
		},
	}

	// Mock successful tool registration
	mockLLM.On("ToNativeTools", toolBox.Tools).Return(nil)

	subagent := NewSubagent(&Config{
		LLM:       mockLLM,
		ToolBox:   toolBox,
		Streaming: false,
	})
	return subagent, mockLLM
}

// Tests
func TestNewSubagent_Success(t *testing.T) {
	mockLLM := &MockLLMClient{}
	toolBox := &tools.ToolBox{
		Tools: []*tools.ToolDefinition{
			{
				Name:        "read_file",
				Description: "Read a file",
				Function:    func(input tools.ToolInput) (string, error) { return "", nil },
			},
		},
	}

	mockLLM.On("ToNativeTools", toolBox.Tools).Return(nil)

	subagent := NewSubagent(&Config{
		LLM:       mockLLM,
		ToolBox:   toolBox,
		Streaming: true,
	})

	assert.NotNil(t, subagent)
	assert.Equal(t, mockLLM, subagent.llm)
	assert.Equal(t, toolBox, subagent.toolBox)
	assert.True(t, subagent.streaming)

	mockLLM.AssertExpectations(t)
}

func TestNewSubagent_ToNativeToolsError(t *testing.T) {
	mockLLM := &MockLLMClient{}
	toolBox := &tools.ToolBox{
		Tools: []*tools.ToolDefinition{
			{
				Name:        "invalid_tool",
				Description: "An invalid tool",
			},
		},
	}

	mockLLM.On("ToNativeTools", toolBox.Tools).Return(errors.New("failed to register tools"))

	// This should panic due to the error
	assert.Panics(t, func() {
		NewSubagent(&Config{
			LLM:       mockLLM,
			ToolBox:   toolBox,
			Streaming: false,
		})
	})

	mockLLM.AssertExpectations(t)
}

func TestSubagent_Run_TextOnlyResponse(t *testing.T) {
	subagent, mockLLM := createTestSubagent()

	// Expected response without tool use
	expectedResponse := &message.Message{
		Role: message.AssistantRole,
		Content: []message.ContentBlock{
			message.NewTextBlock("This is a text-only response"),
		},
		CreatedAt: time.Now(),
	}

	// Setup mocks
	mockLLM.On("ToNativeMessage", mock.MatchedBy(func(msg *message.Message) bool {
		return msg.Role == message.UserRole
	})).Return(nil)
	mockLLM.On("RunInference", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.Anything, false).Return(expectedResponse, nil)
	mockLLM.On("ToNativeMessage", expectedResponse).Return(nil)

	ctx := context.Background()
	systemPrompt := "You are a helpful assistant"
	input := "What is the capital of France?"

	result, err := subagent.Run(ctx, systemPrompt, input)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)

	mockLLM.AssertExpectations(t)
}

func TestSubagent_Run_WithToolUse(t *testing.T) {
	subagent, mockLLM := createTestSubagent()

	// Tool use response
	toolInput, _ := json.Marshal(map[string]string{"query": "test"})
	toolUseResponse := &message.Message{
		Role: message.AssistantRole,
		Content: []message.ContentBlock{
			message.NewToolUseBlock("tool-456", "test_tool", toolInput),
		},
		CreatedAt: time.Now(),
	}

	// Final response after tool execution
	finalResponse := &message.Message{
		Role: message.AssistantRole,
		Content: []message.ContentBlock{
			message.NewTextBlock("Tool execution completed successfully"),
		},
		CreatedAt: time.Now(),
	}

	// Setup mocks
	mockLLM.On("ToNativeMessage", mock.MatchedBy(func(msg *message.Message) bool {
		return msg.Role == message.UserRole && len(msg.Content) == 1
	})).Return(nil).Once() // Initial user message

	mockLLM.On("RunInference", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.Anything, false).Return(toolUseResponse, nil).Once()
	mockLLM.On("ToNativeMessage", toolUseResponse).Return(nil).Once()

	// Tool result message
	mockLLM.On("ToNativeMessage", mock.MatchedBy(func(msg *message.Message) bool {
		return msg.Role == message.UserRole && len(msg.Content) == 1
	})).Return(nil).Once() // Tool result message

	mockLLM.On("RunInference", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.Anything, false).Return(finalResponse, nil).Once()
	mockLLM.On("ToNativeMessage", finalResponse).Return(nil).Once()

	ctx := context.Background()
	systemPrompt := "Use tools to help answer questions"
	input := "Use the test tool"

	result, err := subagent.Run(ctx, systemPrompt, input)

	assert.NoError(t, err)
	assert.Equal(t, finalResponse, result)

	mockLLM.AssertExpectations(t)
}

func TestSubagent_Run_InitialMessageError(t *testing.T) {
	subagent, mockLLM := createTestSubagent()

	expectedError := errors.New("failed to initialize message")
	mockLLM.On("ToNativeMessage", mock.Anything).Return(expectedError)

	ctx := context.Background()
	systemPrompt := "System prompt"
	input := "User input"

	result, err := subagent.Run(ctx, systemPrompt, input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to initialize conversation")

	mockLLM.AssertExpectations(t)
}

func TestSubagent_Run_InferenceError(t *testing.T) {
	subagent, mockLLM := createTestSubagent()

	expectedError := errors.New("inference failed")
	mockLLM.On("ToNativeMessage", mock.Anything).Return(nil)
	mockLLM.On("RunInference", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.Anything, false).Return(nil, expectedError)

	ctx := context.Background()
	systemPrompt := "System prompt"
	input := "User input"

	result, err := subagent.Run(ctx, systemPrompt, input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "inference failed")

	mockLLM.AssertExpectations(t)
}

func TestSubagent_Run_ResponseMessageError(t *testing.T) {
	subagent, mockLLM := createTestSubagent()

	response := &message.Message{
		Role: message.AssistantRole,
		Content: []message.ContentBlock{
			message.NewTextBlock("Response"),
		},
	}

	expectedError := errors.New("failed to add response message")
	mockLLM.On("ToNativeMessage", mock.MatchedBy(func(msg *message.Message) bool {
		return msg.Role == message.UserRole
	})).Return(nil)
	mockLLM.On("RunInference", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.Anything, false).Return(response, nil)
	mockLLM.On("ToNativeMessage", response).Return(expectedError)

	ctx := context.Background()
	systemPrompt := "System prompt"
	input := "User input"

	result, err := subagent.Run(ctx, systemPrompt, input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to add message to conversation")

	mockLLM.AssertExpectations(t)
}

func TestSubagent_Run_ToolResultMessageError(t *testing.T) {
	subagent, mockLLM := createTestSubagent()

	toolInput, _ := json.Marshal(map[string]string{"query": "test"})
	toolUseResponse := &message.Message{
		Role: message.AssistantRole,
		Content: []message.ContentBlock{
			message.NewToolUseBlock("tool-789", "test_tool", toolInput),
		},
	}

	expectedError := errors.New("failed to add tool result message")

	mockLLM.On("ToNativeMessage", mock.MatchedBy(func(msg *message.Message) bool {
		return msg.Role == message.UserRole && len(msg.Content) == 1
	})).Return(nil).Once() // Initial user message

	mockLLM.On("RunInference", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.Anything, false).Return(toolUseResponse, nil)
	mockLLM.On("ToNativeMessage", toolUseResponse).Return(nil)

	// Tool result message should fail
	mockLLM.On("ToNativeMessage", mock.MatchedBy(func(msg *message.Message) bool {
		return msg.Role == message.UserRole && len(msg.Content) == 1
	})).Return(expectedError).Once() // Tool result message

	ctx := context.Background()
	systemPrompt := "System prompt"
	input := "User input"

	result, err := subagent.Run(ctx, systemPrompt, input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to add tool results to conversation")

	mockLLM.AssertExpectations(t)
}

func TestSubagent_executeTool_Success(t *testing.T) {
	subagent, _ := createTestSubagent()

	toolInput, _ := json.Marshal(map[string]string{"param": "value"})

	result := subagent.executeTool("tool-123", "test_tool", toolInput)

	assert.IsType(t, message.ToolResultBlock{}, result)
	toolResult := result.(message.ToolResultBlock)
	assert.Equal(t, "tool-123", toolResult.ToolUseID)
	assert.Equal(t, "test_tool", toolResult.ToolName)
	assert.Equal(t, "subagent test result", toolResult.Content)
	assert.False(t, toolResult.IsError)
}

func TestSubagent_executeTool_ToolNotFound(t *testing.T) {
	subagent, _ := createTestSubagent()

	toolInput, _ := json.Marshal(map[string]string{"param": "value"})

	result := subagent.executeTool("tool-123", "nonexistent_tool", toolInput)

	assert.IsType(t, message.ToolResultBlock{}, result)
	toolResult := result.(message.ToolResultBlock)
	assert.Equal(t, "tool-123", toolResult.ToolUseID)
	assert.Equal(t, "nonexistent_tool", toolResult.ToolName)
	assert.Equal(t, "tool not found", toolResult.Content)
	assert.True(t, toolResult.IsError)
}

func TestSubagent_executeTool_ToolError(t *testing.T) {
	mockLLM := &MockLLMClient{}
	toolBox := &tools.ToolBox{
		Tools: []*tools.ToolDefinition{
			{
				Name:        "error_tool",
				Description: "A tool that returns an error",
				Function: func(input tools.ToolInput) (string, error) {
					return "", errors.New("tool execution failed")
				},
			},
		},
	}

	mockLLM.On("ToNativeTools", toolBox.Tools).Return(nil)
	subagent := NewSubagent(&Config{
		LLM:       mockLLM,
		ToolBox:   toolBox,
		Streaming: false,
	})

	toolInput, _ := json.Marshal(map[string]string{"param": "value"})

	result := subagent.executeTool("tool-123", "error_tool", toolInput)

	assert.IsType(t, message.ToolResultBlock{}, result)
	toolResult := result.(message.ToolResultBlock)
	assert.Equal(t, "tool-123", toolResult.ToolUseID)
	assert.Equal(t, "error_tool", toolResult.ToolName)
	assert.Equal(t, "tool execution failed", toolResult.Content)
	assert.True(t, toolResult.IsError)
}

func TestSubagent_Run_SystemPromptConcatenation(t *testing.T) {
	subagent, mockLLM := createTestSubagent()

	expectedResponse := &message.Message{
		Role: message.AssistantRole,
		Content: []message.ContentBlock{
			message.NewTextBlock("Response"),
		},
	}

	// Capture the user message to verify prompt concatenation
	var capturedUserMessage *message.Message
	mockLLM.On("ToNativeMessage", mock.MatchedBy(func(msg *message.Message) bool {
		if msg.Role == message.UserRole {
			capturedUserMessage = msg
			return true
		}
		return false
	})).Return(nil)
	mockLLM.On("RunInference", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.Anything, false).Return(expectedResponse, nil)
	mockLLM.On("ToNativeMessage", expectedResponse).Return(nil)

	ctx := context.Background()
	systemPrompt := "You are a helpful assistant"
	input := "What is 2+2?"

	result, err := subagent.Run(ctx, systemPrompt, input)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)

	// Verify the system prompt and input were concatenated
	assert.NotNil(t, capturedUserMessage)
	assert.Len(t, capturedUserMessage.Content, 1)
	if textBlock, ok := capturedUserMessage.Content[0].(message.TextBlock); ok {
		expectedText := systemPrompt + "\n\n" + input
		assert.Equal(t, expectedText, textBlock.Text)
	}

	mockLLM.AssertExpectations(t)
}

func TestSubagent_Run_MultipleToolCalls(t *testing.T) {
	mockLLM := &MockLLMClient{}
	toolBox := &tools.ToolBox{
		Tools: []*tools.ToolDefinition{
			{
				Name:        "tool1",
				Description: "First tool",
				Function: func(input tools.ToolInput) (string, error) {
					return "result1", nil
				},
			},
			{
				Name:        "tool2",
				Description: "Second tool",
				Function: func(input tools.ToolInput) (string, error) {
					return "result2", nil
				},
			},
		},
	}

	mockLLM.On("ToNativeTools", toolBox.Tools).Return(nil)
	subagent := NewSubagent(&Config{
		LLM:       mockLLM,
		ToolBox:   toolBox,
		Streaming: false,
	})

	// Response with multiple tool uses
	toolInput1, _ := json.Marshal(map[string]string{"param": "value1"})
	toolInput2, _ := json.Marshal(map[string]string{"param": "value2"})
	multiToolResponse := &message.Message{
		Role: message.AssistantRole,
		Content: []message.ContentBlock{
			message.NewToolUseBlock("tool-1", "tool1", toolInput1),
			message.NewToolUseBlock("tool-2", "tool2", toolInput2),
		},
	}

	finalResponse := &message.Message{
		Role: message.AssistantRole,
		Content: []message.ContentBlock{
			message.NewTextBlock("All tools executed"),
		},
	}

	// Setup mocks
	mockLLM.On("ToNativeMessage", mock.MatchedBy(func(msg *message.Message) bool {
		return msg.Role == message.UserRole && len(msg.Content) == 1
	})).Return(nil).Once() // Initial user message

	mockLLM.On("RunInference", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.Anything, false).Return(multiToolResponse, nil).Once()
	mockLLM.On("ToNativeMessage", multiToolResponse).Return(nil).Once()

	// Tool results message (should contain 2 tool results)
	mockLLM.On("ToNativeMessage", mock.MatchedBy(func(msg *message.Message) bool {
		return msg.Role == message.UserRole && len(msg.Content) == 2
	})).Return(nil).Once()

	mockLLM.On("RunInference", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.Anything, false).Return(finalResponse, nil).Once()
	mockLLM.On("ToNativeMessage", finalResponse).Return(nil).Once()

	ctx := context.Background()
	systemPrompt := "Execute multiple tools"
	input := "Run both tools"

	result, err := subagent.Run(ctx, systemPrompt, input)

	assert.NoError(t, err)
	assert.Equal(t, finalResponse, result)

	mockLLM.AssertExpectations(t)
}

func TestSubagent_Run_StreamingMode(t *testing.T) {
	mockLLM := &MockLLMClient{}
	toolBox := &tools.ToolBox{
		Tools: []*tools.ToolDefinition{},
	}

	mockLLM.On("ToNativeTools", toolBox.Tools).Return(nil)
	subagent := NewSubagent(&Config{
		LLM:       mockLLM,
		ToolBox:   toolBox,
		Streaming: true,
	}) // Enable streaming

	expectedResponse := &message.Message{
		Role: message.AssistantRole,
		Content: []message.ContentBlock{
			message.NewTextBlock("Streaming response"),
		},
	}

	mockLLM.On("ToNativeMessage", mock.Anything).Return(nil)
	mockLLM.On("RunInference", mock.MatchedBy(func(ctx context.Context) bool { return true }), mock.Anything, true).Return(expectedResponse, nil) // Should use streaming=true
	mockLLM.On("ToNativeMessage", expectedResponse).Return(nil)

	ctx := context.Background()
	result, err := subagent.Run(ctx, "System", "Input")

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	assert.True(t, subagent.streaming)

	mockLLM.AssertExpectations(t)
}
