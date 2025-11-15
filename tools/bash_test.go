package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests for Bash function
func TestBash_Success(t *testing.T) {
	input := BashInput{Command: "echo 'hello world'"}
	inputJSON, _ := json.Marshal(input)

	result, err := Bash(inputJSON, ToolMetadata{})

	assert.NoError(t, err)
	assert.Equal(t, "hello world", result)
}

func TestBash_EmptyCommand(t *testing.T) {
	input := BashInput{Command: ""}
	inputJSON, _ := json.Marshal(input)

	result, err := Bash(inputJSON, ToolMetadata{})

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestBash_CommandWithExitCode(t *testing.T) {
	input := BashInput{Command: "exit 1"}
	inputJSON, _ := json.Marshal(input)

	result, err := Bash(inputJSON, ToolMetadata{})

	assert.NoError(t, err) // Bash function doesn't return error for failed commands
	assert.Contains(t, result, "Command failed with error:")
}

func TestBash_MultiLineOutput(t *testing.T) {
	input := BashInput{Command: "echo -e 'line1\\nline2\\nline3'"}
	inputJSON, _ := json.Marshal(input)

	result, err := Bash(inputJSON, ToolMetadata{})

	assert.NoError(t, err)
	assert.Equal(t, "line1\nline2\nline3", result)
}

func TestBash_CommandWithArguments(t *testing.T) {
	input := BashInput{Command: "echo hello && echo world"}
	inputJSON, _ := json.Marshal(input)

	result, err := Bash(inputJSON, ToolMetadata{})

	assert.NoError(t, err)
	assert.Equal(t, "hello\nworld", result)
}

func TestBash_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"command": invalid json}`)

	result, err := Bash(invalidJSON, ToolMetadata{})

	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestBash_NonexistentCommand(t *testing.T) {
	input := BashInput{Command: "nonexistentcommandthatdoesnotexist123"}
	inputJSON, _ := json.Marshal(input)

	result, err := Bash(inputJSON, ToolMetadata{})

	assert.NoError(t, err) // Bash function doesn't return error for failed commands
	assert.Contains(t, result, "Command failed with error:")
}

func TestBash_WhitespaceOutput(t *testing.T) {
	input := BashInput{Command: "echo '   hello world   '"}
	inputJSON, _ := json.Marshal(input)

	result, err := Bash(inputJSON, ToolMetadata{})

	assert.NoError(t, err)
	assert.Equal(t, "hello world", result)
}

func TestBash_SpecialCharacters(t *testing.T) {
	input := BashInput{Command: "echo 'special chars: $@#%^&*()[]{}|\\;:,.<>?'"}
	inputJSON, _ := json.Marshal(input)

	result, err := Bash(inputJSON, ToolMetadata{})

	assert.NoError(t, err)
	assert.Equal(t, "special chars: $@#%^&*()[]{}|\\;:,.<>?", result)
}

// Tests for BashDefinition global variable
func TestBashDefinition_Structure(t *testing.T) {
	assert.Equal(t, "bash", BashDefinition.Name)
	assert.NotEmpty(t, BashDefinition.Description)
	assert.NotNil(t, BashDefinition.InputSchema)
	assert.NotNil(t, BashDefinition.Function)
	assert.False(t, BashDefinition.IsSubTool)
}

func TestBashDefinition_FunctionExecution(t *testing.T) {
	input := BashInput{Command: "echo test"}
	inputJSON, _ := json.Marshal(input)

	result, err := BashDefinition.Function(inputJSON, ToolMetadata{})

	assert.NoError(t, err)
	assert.Equal(t, "test", result)
}

// Tests for BashInput struct
func TestBashInput_JSONMarshaling(t *testing.T) {
	input := BashInput{Command: "echo hello"}

	data, err := json.Marshal(input)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"command":"echo hello"`)
}

func TestBashInput_JSONUnmarshaling(t *testing.T) {
	jsonData := `{"command":"ls -la"}`

	var input BashInput
	err := json.Unmarshal([]byte(jsonData), &input)

	assert.NoError(t, err)
	assert.Equal(t, "ls -la", input.Command)
}

// Table-driven tests for various commands
func TestBash_VariousCommands(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		expectError   bool
		shouldContain string
	}{
		{
			name:          "simple echo",
			command:       "echo hello",
			expectError:   false,
			shouldContain: "hello",
		},
		{
			name:          "date command",
			command:       "date +%Y",
			expectError:   false,
			shouldContain: "20", // Should contain "20" (for 20xx years)
		},
		{
			name:          "pwd command",
			command:       "pwd",
			expectError:   false,
			shouldContain: "/",
		},
		{
			name:          "false command",
			command:       "false",
			expectError:   false, // Bash function doesn't return error
			shouldContain: "Command failed with error:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := BashInput{Command: tt.command}
			inputJSON, _ := json.Marshal(input)

			result, err := Bash(inputJSON, ToolMetadata{})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.shouldContain != "" {
				assert.Contains(t, result, tt.shouldContain)
			}
		})
	}
}

// Benchmark tests
func BenchmarkBash_SimpleCommand(b *testing.B) {
	input := BashInput{Command: "echo benchmark"}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Bash(inputJSON, ToolMetadata{})
	}
}

func BenchmarkBash_ComplexCommand(b *testing.B) {
	input := BashInput{Command: "echo hello && echo world && echo benchmark"}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Bash(inputJSON, ToolMetadata{})
	}
}
