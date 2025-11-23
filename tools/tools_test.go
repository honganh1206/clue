package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests for ToolBox
func TestToolBox_Creation(t *testing.T) {
	toolBox := &ToolBox{
		Tools: []*ToolDefinition{},
	}

	assert.NotNil(t, toolBox)
	assert.NotNil(t, toolBox.Tools)
	assert.Len(t, toolBox.Tools, 0)
}

func TestToolBox_AddTool(t *testing.T) {
	toolBox := &ToolBox{
		Tools: []*ToolDefinition{},
	}

	testTool := &ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		Function: func(data *ToolData) (string, error) {
			return "test result", nil
		},
	}

	toolBox.Tools = append(toolBox.Tools, testTool)

	assert.Len(t, toolBox.Tools, 1)
	assert.Equal(t, "test_tool", toolBox.Tools[0].Name)
	assert.Equal(t, "A test tool", toolBox.Tools[0].Description)
}

func TestToolBox_MultipleLtools(t *testing.T) {
	toolBox := &ToolBox{
		Tools: []*ToolDefinition{
			{
				Name:        "tool1",
				Description: "First tool",
				Function:    func(data *ToolData) (string, error) { return "result1", nil },
			},
			{
				Name:        "tool2",
				Description: "Second tool",
				Function:    func(data *ToolData) (string, error) { return "result2", nil },
				IsSubTool:   true,
			},
		},
	}

	assert.Len(t, toolBox.Tools, 2)
	assert.Equal(t, "tool1", toolBox.Tools[0].Name)
	assert.Equal(t, "tool2", toolBox.Tools[1].Name)
	assert.False(t, toolBox.Tools[0].IsSubTool)
	assert.True(t, toolBox.Tools[1].IsSubTool)
}

// Tests for ToolDefinition
func TestToolDefinition_Creation(t *testing.T) {
	tool := &ToolDefinition{
		Name:        "test_tool",
		Description: "Test description",
		Function: func(data *ToolData) (string, error) {
			return "success", nil
		},
	}

	assert.Equal(t, "test_tool", tool.Name)
	assert.Equal(t, "Test description", tool.Description)
	assert.NotNil(t, tool.Function)
	assert.False(t, tool.IsSubTool)
}

func TestToolDefinition_SubTool(t *testing.T) {
	tool := &ToolDefinition{
		Name:        "sub_tool",
		Description: "Sub tool description",
		IsSubTool:   true,
		Function: func(data *ToolData) (string, error) {
			return "sub result", nil
		},
	}

	assert.True(t, tool.IsSubTool)
}

func TestToolDefinition_FunctionExecution(t *testing.T) {
	tool := &ToolDefinition{
		Name: "echo_tool",
		Function: func(data *ToolData) (string, error) {
			var message map[string]string
			err := json.Unmarshal(data.Input, &message)
			if err != nil {
				return "", err
			}
			return message["message"], nil
		},
	}

	input, _ := json.Marshal(map[string]string{"message": "hello world"})
	result, err := tool.Function(&ToolData{Input: input})

	assert.NoError(t, err)
	assert.Equal(t, "hello world", result)
}
