package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/honganh1206/clue/schema"
)

// TODO: Embed markdown tool prompt
// Apparently Claude loves replacing strings (known through experiments?)
var EditFileDefinition = ToolDefinition{
	Name: ToolNameEditFile,
	Description: `Make edits to a text file.

	Replace 'old_str' with 'new_str' in the given file. 'old_str' and 'new_str' MUST be different from each other.

	If the file specified with path doesn't exist, it will be created`,
	InputSchema: EditFileInputSchema,
	Function:    EditFile,
}

type EditFileInput struct {
	Path   string `json:"path" jsonschema_description:"The path to the file"`
	OldStr string `json:"old_str" jsonschema_description:"Text to search for - must match exactly and must only have one match exactly"`
	NewStr string `json:"new_str" jsonschema_description:"Text to replace old_str with"`
}

var EditFileInputSchema = schema.Generate[EditFileInput]()

func EditFile(data *ToolData) (string, error) {
	editFileInput := EditFileInput{}
	err := json.Unmarshal(data.Input, &editFileInput)
	if err != nil {
		return "", err
	}

	if editFileInput.Path == "" || editFileInput.OldStr == editFileInput.NewStr {
		return "", fmt.Errorf("invalid input parameters")
	}

	content, err := os.ReadFile(editFileInput.Path)
	if err != nil {
		if os.IsNotExist(err) && editFileInput.OldStr == "" {
			return createNewFile(editFileInput.Path, editFileInput.NewStr)
		}
		return "", err
	}

	oldContent := string(content)
	// Replace all occurences
	newContent := strings.ReplaceAll(oldContent, editFileInput.OldStr, editFileInput.NewStr)

	if oldContent == newContent && editFileInput.OldStr != "" {
		return "", fmt.Errorf("old_str not found in file")
	}

	err = os.WriteFile(editFileInput.Path, []byte(newContent), 0o644)
	if err != nil {
		return "", err
	}

	return "OK", nil
}

func createNewFile(filePath, content string) (string, error) {
	dir := path.Dir(filePath)
	if dir != "." {
		// Default permission for dir
		err := os.MkdirAll(dir, 0o755)
		if err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Permission to read and write file
	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	return fmt.Sprintf("successfully created file %s", filePath), nil
}

