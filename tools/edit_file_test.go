package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper functions for edit_file tests
func createTestFileForEdit(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_file.txt")

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	return filePath
}

// Tests for EditFile function
func TestEditFile_Success(t *testing.T) {
	content := "Hello world, this is a test"
	filePath := createTestFileForEdit(t, content)

	input := EditFileInput{
		Path:   filePath,
		OldStr: "world",
		NewStr: "universe",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := EditFile(inputJSON)

	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	// Verify file content changed
	newContent, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "Hello universe, this is a test", string(newContent))
}

func TestEditFile_MultipleReplacements(t *testing.T) {
	content := "test test test"
	filePath := createTestFileForEdit(t, content)

	input := EditFileInput{
		Path:   filePath,
		OldStr: "test",
		NewStr: "example",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := EditFile(inputJSON)

	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	// Verify all occurrences were replaced
	newContent, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "example example example", string(newContent))
}

func TestEditFile_CreateNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "new_file.txt")

	input := EditFileInput{
		Path:   filePath,
		OldStr: "",
		NewStr: "This is a new file",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := EditFile(inputJSON)

	assert.NoError(t, err)
	assert.Contains(t, result, "successfully created file")

	// Verify file was created with correct content
	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "This is a new file", string(content))
}

func TestEditFile_CreateNewFileWithDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "subdir", "nested", "new_file.txt")

	input := EditFileInput{
		Path:   filePath,
		OldStr: "",
		NewStr: "Nested file content",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := EditFile(inputJSON)

	assert.NoError(t, err)
	assert.Contains(t, result, "successfully created file")

	// Verify file was created with correct content
	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "Nested file content", string(content))
}

func TestEditFile_OldStrNotFound(t *testing.T) {
	content := "Hello world"
	filePath := createTestFileForEdit(t, content)

	input := EditFileInput{
		Path:   filePath,
		OldStr: "nonexistent",
		NewStr: "replacement",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := EditFile(inputJSON)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "old_str not found in file")

	// Verify file content unchanged
	originalContent, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, content, string(originalContent))
}

func TestEditFile_NonexistentFile(t *testing.T) {
	input := EditFileInput{
		Path:   "/nonexistent/path/file.txt",
		OldStr: "old",
		NewStr: "new",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := EditFile(inputJSON)

	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestEditFile_InvalidParameters(t *testing.T) {
	tests := []struct {
		name   string
		input  EditFileInput
		hasErr bool
	}{
		{
			name:   "empty path",
			input:  EditFileInput{Path: "", OldStr: "old", NewStr: "new"},
			hasErr: true,
		},
		{
			name:   "same old and new strings",
			input:  EditFileInput{Path: "/tmp/test.txt", OldStr: "same", NewStr: "same"},
			hasErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJSON, _ := json.Marshal(tt.input)
			result, err := EditFile(inputJSON)

			if tt.hasErr {
				assert.Error(t, err)
				assert.Empty(t, result)
				assert.Contains(t, err.Error(), "invalid input parameters")
			}
		})
	}
}

func TestEditFile_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"path": invalid json}`)

	result, err := EditFile(invalidJSON)

	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestEditFile_EmptyFile(t *testing.T) {
	filePath := createTestFileForEdit(t, "")

	input := EditFileInput{
		Path:   filePath,
		OldStr: "",
		NewStr: "New content",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := EditFile(inputJSON)

	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	// Verify content was added
	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "New content", string(content))
}

func TestEditFile_SpecialCharacters(t *testing.T) {
	content := "Hello @#$%^&*()_+ world"
	filePath := createTestFileForEdit(t, content)

	input := EditFileInput{
		Path:   filePath,
		OldStr: "@#$%^&*()_+",
		NewStr: "special",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := EditFile(inputJSON)

	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	newContent, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "Hello special world", string(newContent))
}

func TestEditFile_MultilineContent(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3"
	filePath := createTestFileForEdit(t, content)

	input := EditFileInput{
		Path:   filePath,
		OldStr: "Line 2",
		NewStr: "Modified Line 2",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := EditFile(inputJSON)

	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	newContent, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "Line 1\nModified Line 2\nLine 3", string(newContent))
}

// Tests for EditFileDefinition global variable
func TestEditFileDefinition_Structure(t *testing.T) {
	assert.Equal(t, "edit_file", EditFileDefinition.Name)
	assert.NotEmpty(t, EditFileDefinition.Description)
	assert.NotNil(t, EditFileDefinition.InputSchema)
	assert.NotNil(t, EditFileDefinition.Function)
	assert.False(t, EditFileDefinition.IsSubTool)
}

func TestEditFileDefinition_FunctionExecution(t *testing.T) {
	content := "test content"
	filePath := createTestFileForEdit(t, content)

	input := EditFileInput{
		Path:   filePath,
		OldStr: "test",
		NewStr: "modified",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := EditFileDefinition.Function(inputJSON)

	assert.NoError(t, err)
	assert.Equal(t, "OK", result)
}

// Tests for EditFileInput struct
func TestEditFileInput_JSONMarshaling(t *testing.T) {
	input := EditFileInput{
		Path:   "/test/path.txt",
		OldStr: "old text",
		NewStr: "new text",
	}

	data, err := json.Marshal(input)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"path":"/test/path.txt"`)
	assert.Contains(t, jsonStr, `"old_str":"old text"`)
	assert.Contains(t, jsonStr, `"new_str":"new text"`)
}

func TestEditFileInput_JSONUnmarshaling(t *testing.T) {
	jsonData := `{"path":"/test/path.txt","old_str":"old text","new_str":"new text"}`

	var input EditFileInput
	err := json.Unmarshal([]byte(jsonData), &input)

	assert.NoError(t, err)
	assert.Equal(t, "/test/path.txt", input.Path)
	assert.Equal(t, "old text", input.OldStr)
	assert.Equal(t, "new text", input.NewStr)
}

// Table-driven tests
func TestEditFile_VariousReplacements(t *testing.T) {
	tests := []struct {
		name            string
		originalContent string
		oldStr          string
		newStr          string
		expectedContent string
		expectError     bool
	}{
		{
			name:            "simple replacement",
			originalContent: "Hello world",
			oldStr:          "world",
			newStr:          "Go",
			expectedContent: "Hello Go",
			expectError:     false,
		},
		{
			name:            "empty replacement",
			originalContent: "Hello world!",
			oldStr:          "!",
			newStr:          "",
			expectedContent: "Hello world",
			expectError:     false,
		},
		{
			name:            "add to empty string",
			originalContent: "",
			oldStr:          "",
			newStr:          "New content",
			expectedContent: "New content",
			expectError:     false,
		},
		{
			name:            "whole file replacement",
			originalContent: "Replace me",
			oldStr:          "Replace me",
			newStr:          "I am replaced",
			expectedContent: "I am replaced",
			expectError:     false,
		},
		{
			name:            "partial word replacement",
			originalContent: "testing",
			oldStr:          "test",
			newStr:          "check",
			expectedContent: "checking",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := createTestFileForEdit(t, tt.originalContent)

			input := EditFileInput{
				Path:   filePath,
				OldStr: tt.oldStr,
				NewStr: tt.newStr,
			}
			inputJSON, _ := json.Marshal(input)

			result, err := EditFile(inputJSON)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "OK", result)

				// Verify content
				content, err := os.ReadFile(filePath)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedContent, string(content))
			}
		})
	}
}

// Integration tests
func TestEditFile_MultipleEdits(t *testing.T) {
	content := "The quick brown fox jumps over the lazy dog"
	filePath := createTestFileForEdit(t, content)

	// First edit
	input1 := EditFileInput{
		Path:   filePath,
		OldStr: "quick",
		NewStr: "fast",
	}
	inputJSON1, _ := json.Marshal(input1)
	result1, err1 := EditFile(inputJSON1)
	assert.NoError(t, err1)
	assert.Equal(t, "OK", result1)

	// Second edit
	input2 := EditFileInput{
		Path:   filePath,
		OldStr: "fox",
		NewStr: "cat",
	}
	inputJSON2, _ := json.Marshal(input2)
	result2, err2 := EditFile(inputJSON2)
	assert.NoError(t, err2)
	assert.Equal(t, "OK", result2)

	// Verify final content
	finalContent, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "The fast brown cat jumps over the lazy dog", string(finalContent))
}

// Benchmark tests
func BenchmarkEditFile_SimpleReplacement(b *testing.B) {
	content := "Hello world, this is a benchmark test"
	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "benchmark.txt")

	input := EditFileInput{
		Path:   filePath,
		OldStr: "world",
		NewStr: "benchmark",
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Recreate file for each iteration
		os.WriteFile(filePath, []byte(content), 0644)
		EditFile(inputJSON)
	}
}

func BenchmarkEditFile_LargeFile(b *testing.B) {
	// Create large content
	largeContent := ""
	for i := 0; i < 1000; i++ {
		largeContent += "This is line number " + string(rune(i)) + " in a large file for benchmark testing.\n"
	}

	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "large_benchmark.txt")

	input := EditFileInput{
		Path:   filePath,
		OldStr: "line number",
		NewStr: "row number",
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Recreate file for each iteration
		os.WriteFile(filePath, []byte(largeContent), 0644)
		EditFile(inputJSON)
	}
}
