package tools

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test helpers
func createTestFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_file.txt")

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	return filePath
}

func createTestDirectory(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// Tests for ReadFile function
func TestReadFile_Success(t *testing.T) {
	content := "This is test content"
	filePath := createTestFile(t, content)

	input := ReadFileInput{Path: filePath}
	inputJSON, _ := json.Marshal(input)

	result, err := ReadFile(&ToolData{Input: inputJSON})

	assert.NoError(t, err)
	assert.Equal(t, content, result)
}

func TestReadFile_NonexistentFile(t *testing.T) {
	input := ReadFileInput{Path: "/nonexistent/file.txt"}
	inputJSON, _ := json.Marshal(input)

	result, err := ReadFile(&ToolData{Input: inputJSON})

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.IsType(t, &fs.PathError{}, err)
}

func TestReadFile_EmptyFile(t *testing.T) {
	filePath := createTestFile(t, "")

	input := ReadFileInput{Path: filePath}
	inputJSON, _ := json.Marshal(input)

	result, err := ReadFile(&ToolData{Input: inputJSON})

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestReadFile_LargeFile(t *testing.T) {
	// Create a large content string
	largeContent := ""
	for i := 0; i < 1000; i++ {
		largeContent += "This is line " + string(rune(i)) + " of the large file content.\n"
	}

	filePath := createTestFile(t, largeContent)

	input := ReadFileInput{Path: filePath}
	inputJSON, _ := json.Marshal(input)

	result, err := ReadFile(&ToolData{Input: inputJSON})

	assert.NoError(t, err)
	assert.Equal(t, largeContent, result)
}

func TestReadFile_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"invalid": json}`)

	// ReadFile panics on JSON unmarshal error, so we need to recover from panic
	defer func() {
		if r := recover(); r != nil {
			assert.NotNil(t, r)
		}
	}()

	ReadFile(&ToolData{Input: invalidJSON})
	t.Error("Expected panic but didn't get one")
}

func TestReadFile_BinaryFile(t *testing.T) {
	// Create a binary file with some non-text content
	tmpDir := createTestDirectory(t)
	filePath := filepath.Join(tmpDir, "binary_file.bin")

	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	err := os.WriteFile(filePath, binaryData, 0644)
	assert.NoError(t, err)

	input := ReadFileInput{Path: filePath}
	inputJSON, _ := json.Marshal(input)

	result, err := ReadFile(&ToolData{Input: inputJSON})

	assert.NoError(t, err)
	assert.Equal(t, string(binaryData), result)
}

func TestReadFile_Directory(t *testing.T) {
	dirPath := createTestDirectory(t)

	input := ReadFileInput{Path: dirPath}
	inputJSON, _ := json.Marshal(input)

	result, err := ReadFile(&ToolData{Input: inputJSON})

	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestReadFile_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	content := "restricted content"
	filePath := createTestFile(t, content)

	// Change permissions to make file unreadable
	err := os.Chmod(filePath, 0000)
	assert.NoError(t, err)

	// Restore permissions for cleanup
	defer os.Chmod(filePath, 0644)

	input := ReadFileInput{Path: filePath}
	inputJSON, _ := json.Marshal(input)

	result, err := ReadFile(&ToolData{Input: inputJSON})

	assert.Error(t, err)
	assert.Empty(t, result)
}

// Tests for ReadFileDefinition global variable
func TestReadFileDefinition_Structure(t *testing.T) {
	assert.Equal(t, "read_file", ReadFileDefinition.Name)
	assert.NotEmpty(t, ReadFileDefinition.Description)
	assert.NotNil(t, ReadFileDefinition.InputSchema)
	assert.NotNil(t, ReadFileDefinition.Function)
	assert.False(t, ReadFileDefinition.IsSubTool)
}

func TestReadFileDefinition_FunctionExecution(t *testing.T) {
	content := "Test content for definition"
	filePath := createTestFile(t, content)

	input := ReadFileInput{Path: filePath}
	inputJSON, _ := json.Marshal(input)

	result, err := ReadFileDefinition.Function(&ToolData{Input: inputJSON})

	assert.NoError(t, err)
	assert.Equal(t, content, result)
}

// Tests for ReadFileInput struct
func TestReadFileInput_JSONMarshaling(t *testing.T) {
	input := ReadFileInput{Path: "/absolute/path/to/file.txt"}

	data, err := json.Marshal(input)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"path":"/absolute/path/to/file.txt"`)
}

func TestReadFileInput_JSONUnmarshaling(t *testing.T) {
	jsonData := `{"path":"/absolute/path/to/file.txt"}`

	var input ReadFileInput
	err := json.Unmarshal([]byte(jsonData), &input)

	assert.NoError(t, err)
	assert.Equal(t, "/absolute/path/to/file.txt", input.Path)
}

// Integration tests
func TestReadFileIntegration_MultipleFiles(t *testing.T) {
	// Create multiple test files
	file1Path := createTestFile(t, "Content of file 1")
	file2Path := createTestFile(t, "Content of file 2")

	// Test reading first file
	input1 := ReadFileInput{Path: file1Path}
	inputJSON1, _ := json.Marshal(input1)
	result1, err1 := ReadFile(&ToolData{Input: inputJSON1})

	assert.NoError(t, err1)
	assert.Equal(t, "Content of file 1", result1)

	// Test reading second file
	input2 := ReadFileInput{Path: file2Path}
	inputJSON2, _ := json.Marshal(input2)
	result2, err2 := ReadFile(&ToolData{Input: inputJSON2})

	assert.NoError(t, err2)
	assert.Equal(t, "Content of file 2", result2)
}

// Table-driven tests
func TestReadFile_VariousFileTypes(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name:        "text file",
			content:     "Regular text content",
			expectError: false,
		},
		{
			name:        "empty file",
			content:     "",
			expectError: false,
		},
		{
			name:        "json file",
			content:     `{"key": "value", "number": 42}`,
			expectError: false,
		},
		{
			name:        "multiline file",
			content:     "Line 1\nLine 2\nLine 3",
			expectError: false,
		},
		{
			name:        "file with special characters",
			content:     "Special chars: áéíóú ñÑ ¿¡ €£¥",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := createTestFile(t, tt.content)

			input := ReadFileInput{Path: filePath}
			inputJSON, _ := json.Marshal(input)

			result, err := ReadFile(&ToolData{Input: inputJSON})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.content, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkReadFile_SmallFile(b *testing.B) {
	content := "Small file content for benchmarking"
	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "small_file.txt")
	os.WriteFile(filePath, []byte(content), 0644)

	input := ReadFileInput{Path: filePath}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ReadFile(&ToolData{Input: inputJSON})
	}
}

func BenchmarkReadFile_LargeFile(b *testing.B) {
	// Create a larger file for benchmarking
	largeContent := ""
	for i := 0; i < 10000; i++ {
		largeContent += "This is a line of content for benchmarking large file reads.\n"
	}

	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "large_file.txt")
	os.WriteFile(filePath, []byte(largeContent), 0644)

	input := ReadFileInput{Path: filePath}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ReadFile(&ToolData{Input: inputJSON})
	}
}
