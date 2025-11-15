package tools

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper functions for grep tests

func createTestDirectoryForGrep(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"file1.txt": "Hello world\nThis is a test file\nContains some text",
		"file2.go":  "package main\nfunc main() {\n\tfmt.Println(\"Hello Go\")\n}",
		"file3.md":  "# Markdown File\nThis is **bold** text\nAnd some *italic* text",
	}

	for filename, content := range files {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	return tmpDir
}

// Tests for GrepSearch function
func TestGrepSearch_Success(t *testing.T) {
	// Skip if ripgrep is not available
	if !isRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available, skipping test")
	}

	testDir := createTestDirectoryForGrep(t)

	input := GrepSearchInput{
		Pattern:   "Hello",
		Directory: testDir,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := GrepSearch(inputJSON, ToolMetadata{})

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "[")
	assert.Contains(t, result, "]")
}

func TestGrepSearch_EmptyResult(t *testing.T) {
	if !isRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available, skipping test")
	}

	testDir := createTestDirectoryForGrep(t)

	input := GrepSearchInput{
		Pattern:   "nonexistentpattern12345",
		Directory: testDir,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := GrepSearch(inputJSON, ToolMetadata{})

	assert.NoError(t, err)
	assert.Equal(t, "[]", result)
}

func TestGrepSearch_NoDirectory(t *testing.T) {
	if !isRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available, skipping test")
	}

	input := GrepSearchInput{
		Pattern: "package",
	}
	inputJSON, _ := json.Marshal(input)

	_, err := GrepSearch(inputJSON, ToolMetadata{})

	// Should not error even if no matches found in current directory
	assert.NoError(t, err)
}

func TestGrepSearch_EmptyPattern(t *testing.T) {
	input := GrepSearchInput{
		Pattern:   "",
		Directory: "/tmp",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := GrepSearch(inputJSON, ToolMetadata{})

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "invalid pattern parameter")
}

func TestGrepSearch_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"pattern": invalid json}`)

	result, err := GrepSearch(invalidJSON, ToolMetadata{})

	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestGrepSearch_NonexistentDirectory(t *testing.T) {
	if !isRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available, skipping test")
	}

	input := GrepSearchInput{
		Pattern:   "test",
		Directory: "/nonexistent/directory/path",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := GrepSearch(inputJSON, ToolMetadata{})

	// Should return error for nonexistent directory
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestGrepSearch_RegexPattern(t *testing.T) {
	if !isRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available, skipping test")
	}

	testDir := createTestDirectoryForGrep(t)

	input := GrepSearchInput{
		Pattern:   "H[ae]llo",
		Directory: testDir,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := GrepSearch(inputJSON, ToolMetadata{})

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Tests for GrepSearchDefinition global variable
func TestGrepSearchDefinition_Structure(t *testing.T) {
	assert.Equal(t, "grep_search", GrepSearchDefinition.Name)
	assert.NotEmpty(t, GrepSearchDefinition.Description)
	assert.NotNil(t, GrepSearchDefinition.InputSchema)
	assert.NotNil(t, GrepSearchDefinition.Function)
	assert.False(t, GrepSearchDefinition.IsSubTool)
}

func TestGrepSearchDefinition_FunctionExecution(t *testing.T) {
	if !isRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available, skipping test")
	}

	testDir := createTestDirectoryForGrep(t)

	input := GrepSearchInput{
		Pattern:   "test",
		Directory: testDir,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := GrepSearchDefinition.Function(inputJSON)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Tests for GrepSearchInput struct
func TestGrepSearchInput_JSONMarshaling(t *testing.T) {
	input := GrepSearchInput{
		Pattern:   "search pattern",
		Directory: "/test/directory",
	}

	data, err := json.Marshal(input)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"pattern":"search pattern"`)
	assert.Contains(t, jsonStr, `"directory":"/test/directory"`)
}

func TestGrepSearchInput_JSONMarshalingWithoutDirectory(t *testing.T) {
	input := GrepSearchInput{
		Pattern: "search pattern",
	}

	data, err := json.Marshal(input)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"pattern":"search pattern"`)
	// Directory should be omitted when empty due to omitempty tag
	assert.NotContains(t, jsonStr, `"directory"`)
}

func TestGrepSearchInput_JSONUnmarshaling(t *testing.T) {
	jsonData := `{"pattern":"test pattern","directory":"/test/dir"}`

	var input GrepSearchInput
	err := json.Unmarshal([]byte(jsonData), &input)

	assert.NoError(t, err)
	assert.Equal(t, "test pattern", input.Pattern)
	assert.Equal(t, "/test/dir", input.Directory)
}

func TestGrepSearchInput_JSONUnmarshalingWithoutDirectory(t *testing.T) {
	jsonData := `{"pattern":"test pattern"}`

	var input GrepSearchInput
	err := json.Unmarshal([]byte(jsonData), &input)

	assert.NoError(t, err)
	assert.Equal(t, "test pattern", input.Pattern)
	assert.Empty(t, input.Directory)
}

// Table-driven tests
func TestGrepSearch_VariousPatterns(t *testing.T) {
	if !isRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available, skipping test")
	}

	testDir := createTestDirectoryForGrep(t)

	tests := []struct {
		name        string
		pattern     string
		expectMatch bool
	}{
		{
			name:        "simple word",
			pattern:     "Hello",
			expectMatch: true,
		},
		{
			name:        "case sensitive",
			pattern:     "hello",
			expectMatch: false, // Our test files have "Hello" not "hello"
		},
		{
			name:        "word boundary",
			pattern:     "\\btest\\b",
			expectMatch: true,
		},
		{
			name:        "nonexistent pattern",
			pattern:     "xyz123nonexistent",
			expectMatch: false,
		},
		{
			name:        "special characters",
			pattern:     "\\*\\*bold\\*\\*",
			expectMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := GrepSearchInput{
				Pattern:   tt.pattern,
				Directory: testDir,
			}
			inputJSON, _ := json.Marshal(input)

			result, err := GrepSearch(inputJSON, ToolMetadata{})

			assert.NoError(t, err)

			if tt.expectMatch {
				assert.NotEqual(t, "[]", result)
				assert.Contains(t, result, "[")
			} else {
				assert.Equal(t, "[]", result)
			}
		})
	}
}

// Helper function to check if ripgrep is available
func isRipgrepAvailable() bool {
	cmd := exec.Command("rg", "--version")
	err := cmd.Run()
	if err != nil {
		return false
	}
	return true
}

// Integration tests
func TestGrepSearch_MultipleFiles(t *testing.T) {
	if !isRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available, skipping test")
	}

	tmpDir := t.TempDir()

	// Create multiple files with different content
	files := map[string]string{
		"config.json": `{"name": "test", "version": "1.0.0"}`,
		"readme.md":   "# Test Project\nThis is a test project for grep functionality",
		"main.go":     "package main\n\nfunc main() {\n\tprintln(\"test\")\n}",
		"data.txt":    "test data\nmore test information\nfinal test line",
		"other.log":   "INFO: Starting application\nERROR: Test failed\nINFO: Shutting down",
	}

	for filename, content := range files {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	input := GrepSearchInput{
		Pattern:   "test",
		Directory: tmpDir,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := GrepSearch(inputJSON, ToolMetadata{})

	assert.NoError(t, err)
	assert.NotEqual(t, "[]", result)
	assert.Contains(t, result, "[")
	assert.Contains(t, result, "]")
}

// Benchmark tests
func BenchmarkGrepSearch_SimplePattern(b *testing.B) {
	if !isRipgrepAvailable() {
		b.Skip("ripgrep (rg) not available, skipping benchmark")
	}

	testDir := createTestDirectoryForGrep(&testing.T{})
	input := GrepSearchInput{
		Pattern:   "Hello",
		Directory: testDir,
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GrepSearch(inputJSON, ToolMetadata{})
	}
}

func BenchmarkGrepSearch_ComplexPattern(b *testing.B) {
	if !isRipgrepAvailable() {
		b.Skip("ripgrep (rg) not available, skipping benchmark")
	}

	testDir := createTestDirectoryForGrep(&testing.T{})
	input := GrepSearchInput{
		Pattern:   "\\b[A-Z][a-z]+\\b",
		Directory: testDir,
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GrepSearch(inputJSON, ToolMetadata{})
	}
}
