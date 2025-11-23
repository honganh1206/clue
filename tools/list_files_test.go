package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper functions for list_files tests
func createTestDirectoryForList(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Create test structure
	structure := map[string]string{
		"file1.txt":         "content1",
		"file2.go":          "package main",
		"subdir/file3.md":   "# Title",
		"subdir/file4.json": `{"key": "value"}`,
		"empty_dir/.keep":   "",
		".hidden_file":      "hidden content",
	}

	for path, content := range structure {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)

		// Create directory if it doesn't exist
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		// Create file
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", fullPath, err)
		}
	}

	return tmpDir
}

func createEmptyTestDirectory(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func createTestDirectoryWithGit(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Create regular files
	files := map[string]string{
		"main.go":     "package main",
		"README.md":   "# Project",
		".git/config": "[core]",
		".git/HEAD":   "ref: refs/heads/main",
		".gitignore":  "*.log",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)

		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", fullPath, err)
		}
	}

	return tmpDir
}

// Tests for ListFiles function
func TestListFiles_Success(t *testing.T) {
	testDir := createTestDirectoryForList(t)

	input := ListFilesInput{Path: testDir}
	inputJSON, _ := json.Marshal(input)

	result, err := ListFiles(&ToolData{Input: inputJSON})

	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Parse result as JSON array
	var files []string
	err = json.Unmarshal([]byte(result), &files)
	assert.NoError(t, err)
	assert.Greater(t, len(files), 0)

	// Check that some expected files are present
	fileMap := make(map[string]bool)
	for _, file := range files {
		fileMap[file] = true
	}

	assert.True(t, fileMap["file1.txt"])
	assert.True(t, fileMap["file2.go"])
	assert.True(t, fileMap["subdir/"])
	assert.True(t, fileMap["subdir/file3.md"])
}

func TestListFiles_EmptyDirectory(t *testing.T) {
	emptyDir := createEmptyTestDirectory(t)

	input := ListFilesInput{Path: emptyDir}
	inputJSON, _ := json.Marshal(input)

	result, err := ListFiles(&ToolData{Input: inputJSON})

	assert.NoError(t, err)
	assert.Equal(t, "null", result)
}

func TestListFiles_NoPathProvided(t *testing.T) {
	input := ListFilesInput{}
	inputJSON, _ := json.Marshal(input)

	result, err := ListFiles(&ToolData{Input: inputJSON})

	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Should return files from current directory
	var files []string
	err = json.Unmarshal([]byte(result), &files)
	assert.NoError(t, err)
}

func TestListFiles_NonexistentDirectory(t *testing.T) {
	input := ListFilesInput{Path: "/nonexistent/directory"}
	inputJSON, _ := json.Marshal(input)

	result, err := ListFiles(&ToolData{Input: inputJSON})

	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestListFiles_GitDirectorySkipped(t *testing.T) {
	testDir := createTestDirectoryWithGit(t)

	input := ListFilesInput{Path: testDir}
	inputJSON, _ := json.Marshal(input)

	result, err := ListFiles(&ToolData{Input: inputJSON})

	assert.NoError(t, err)

	var files []string
	err = json.Unmarshal([]byte(result), &files)
	assert.NoError(t, err)

	// Check that .git directory contents are not included
	for _, file := range files {
		assert.NotContains(t, file, ".git/config")
		assert.NotContains(t, file, ".git/HEAD")
	}

	// But .gitignore should be included
	fileMap := make(map[string]bool)
	for _, file := range files {
		fileMap[file] = true
	}
	assert.True(t, fileMap[".gitignore"])
	assert.True(t, fileMap["main.go"])
	assert.True(t, fileMap["README.md"])
}

func TestListFiles_DirectoryIndicator(t *testing.T) {
	testDir := createTestDirectoryForList(t)

	input := ListFilesInput{Path: testDir}
	inputJSON, _ := json.Marshal(input)

	result, err := ListFiles(&ToolData{Input: inputJSON})

	assert.NoError(t, err)

	var files []string
	err = json.Unmarshal([]byte(result), &files)
	assert.NoError(t, err)

	// Check that directories have trailing slash
	fileMap := make(map[string]bool)
	for _, file := range files {
		fileMap[file] = true
	}

	assert.True(t, fileMap["subdir/"])
	assert.True(t, fileMap["empty_dir/"])
}

func TestListFiles_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"path": invalid json}`)

	result, err := ListFiles(&ToolData{Input: invalidJSON})

	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestListFiles_RelativePaths(t *testing.T) {
	testDir := createTestDirectoryForList(t)

	input := ListFilesInput{Path: testDir}
	inputJSON, _ := json.Marshal(input)

	result, err := ListFiles(&ToolData{Input: inputJSON})

	assert.NoError(t, err)

	var files []string
	err = json.Unmarshal([]byte(result), &files)
	assert.NoError(t, err)

	// All paths should be relative, not absolute
	for _, file := range files {
		assert.False(t, filepath.IsAbs(file))
	}
}

// Tests for ListFilesDefinition global variable
func TestListFilesDefinition_Structure(t *testing.T) {
	assert.Equal(t, "list_files", ListFilesDefinition.Name)
	assert.NotEmpty(t, ListFilesDefinition.Description)
	assert.NotNil(t, ListFilesDefinition.InputSchema)
	assert.NotNil(t, ListFilesDefinition.Function)
	assert.False(t, ListFilesDefinition.IsSubTool)
}

func TestListFilesDefinition_FunctionExecution(t *testing.T) {
	testDir := createTestDirectoryForList(t)

	input := ListFilesInput{Path: testDir}
	inputJSON, _ := json.Marshal(input)

	result, err := ListFilesDefinition.Function(&ToolData{Input: inputJSON})

	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	var files []string
	err = json.Unmarshal([]byte(result), &files)
	assert.NoError(t, err)
	assert.Greater(t, len(files), 0)
}

// Tests for ListFilesInput struct
func TestListFilesInput_JSONMarshaling(t *testing.T) {
	input := ListFilesInput{Path: "/test/directory"}

	data, err := json.Marshal(input)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"path":"/test/directory"`)
}

func TestListFilesInput_JSONMarshalingWithoutPath(t *testing.T) {
	input := ListFilesInput{}

	data, err := json.Marshal(input)
	assert.NoError(t, err)

	// Path should be omitted when empty due to omitempty tag
	jsonStr := string(data)
	assert.NotContains(t, jsonStr, `"path"`)
}

func TestListFilesInput_JSONUnmarshaling(t *testing.T) {
	jsonData := `{"path":"/test/directory"}`

	var input ListFilesInput
	err := json.Unmarshal([]byte(jsonData), &input)

	assert.NoError(t, err)
	assert.Equal(t, "/test/directory", input.Path)
}

func TestListFilesInput_JSONUnmarshalingWithoutPath(t *testing.T) {
	jsonData := `{}`

	var input ListFilesInput
	err := json.Unmarshal([]byte(jsonData), &input)

	assert.NoError(t, err)
	assert.Empty(t, input.Path)
}

// Table-driven tests
func TestListFiles_VariousDirectoryStructures(t *testing.T) {
	tests := []struct {
		name            string
		setupFunc       func(t *testing.T) string
		expectedFiles   []string
		unexpectedFiles []string
	}{
		{
			name: "simple directory",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				os.WriteFile(filepath.Join(dir, "simple.txt"), []byte("content"), 0644)
				return dir
			},
			expectedFiles:   []string{"simple.txt"},
			unexpectedFiles: []string{},
		},
		{
			name: "nested directories",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, "nested", "deep"), 0755)
				os.WriteFile(filepath.Join(dir, "nested", "deep", "file.txt"), []byte("content"), 0644)
				return dir
			},
			expectedFiles:   []string{"nested/", "nested/deep/", "nested/deep/file.txt"},
			unexpectedFiles: []string{},
		},
		{
			name: "mixed files and directories",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content1"), 0644)
				os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
				os.WriteFile(filepath.Join(dir, "subdir", "file2.txt"), []byte("content2"), 0644)
				return dir
			},
			expectedFiles:   []string{"file1.txt", "subdir/", "subdir/file2.txt"},
			unexpectedFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := tt.setupFunc(t)

			input := ListFilesInput{Path: testDir}
			inputJSON, _ := json.Marshal(input)

			result, err := ListFiles(&ToolData{Input: inputJSON})

			assert.NoError(t, err)

			var files []string
			err = json.Unmarshal([]byte(result), &files)
			assert.NoError(t, err)

			fileMap := make(map[string]bool)
			for _, file := range files {
				fileMap[file] = true
			}

			for _, expected := range tt.expectedFiles {
				assert.True(t, fileMap[expected], "Expected file %s not found", expected)
			}

			for _, unexpected := range tt.unexpectedFiles {
				assert.False(t, fileMap[unexpected], "Unexpected file %s found", unexpected)
			}
		})
	}
}

// Integration tests
func TestListFiles_ComplexDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a complex directory structure
	structure := []string{
		"README.md",
		"main.go",
		"pkg/utils/helper.go",
		"pkg/models/user.go",
		"cmd/cli/main.go",
		"docs/README.md",
		"tests/unit/user_test.go",
		"tests/integration/api_test.go",
		"configs/app.yaml",
		".env.example",
	}

	for _, path := range structure {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)

		err := os.MkdirAll(dir, 0755)
		assert.NoError(t, err)

		err = os.WriteFile(fullPath, []byte("content"), 0644)
		assert.NoError(t, err)
	}

	input := ListFilesInput{Path: tmpDir}
	inputJSON, _ := json.Marshal(input)

	result, err := ListFiles(&ToolData{Input: inputJSON})

	assert.NoError(t, err)

	var files []string
	err = json.Unmarshal([]byte(result), &files)
	assert.NoError(t, err)

	// Should have all files plus directories
	assert.Greater(t, len(files), len(structure))

	fileMap := make(map[string]bool)
	for _, file := range files {
		fileMap[file] = true
	}

	// Check for some key files and directories
	assert.True(t, fileMap["README.md"])
	assert.True(t, fileMap["main.go"])
	assert.True(t, fileMap["pkg/"])
	assert.True(t, fileMap["pkg/utils/"])
	assert.True(t, fileMap["pkg/utils/helper.go"])
	assert.True(t, fileMap["cmd/"])
	assert.True(t, fileMap["tests/"])
}

// Benchmark tests
func BenchmarkListFiles_SmallDirectory(b *testing.B) {
	// Create small directory with few files
	dir := b.TempDir()
	for i := 0; i < 10; i++ {
		filename := filepath.Join(dir, "file"+string(rune('0'+i))+".txt")
		os.WriteFile(filename, []byte("content"), 0644)
	}

	input := ListFilesInput{Path: dir}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ListFiles(&ToolData{Input: inputJSON})
	}
}

func BenchmarkListFiles_LargeDirectory(b *testing.B) {
	// Create larger directory structure
	dir := b.TempDir()

	// Create multiple subdirectories with files
	for i := 0; i < 5; i++ {
		subdir := filepath.Join(dir, "subdir"+string(rune('0'+i)))
		os.MkdirAll(subdir, 0755)

		for j := 0; j < 20; j++ {
			filename := filepath.Join(subdir, "file"+string(rune('0'+j))+".txt")
			os.WriteFile(filename, []byte("content"), 0644)
		}
	}

	input := ListFilesInput{Path: dir}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ListFiles(&ToolData{Input: inputJSON})
	}
}
