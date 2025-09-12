package tools

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/honganh1206/clue/schema"
)

var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, list files in the current directory",
	InputSchema: ListFilesInputSchema,
	Function:    ListFiles,
}

type ListFilesInput struct {
	Path string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
}

var ListFilesInputSchema = schema.Generate[ListFilesInput]()

func ListFiles(input json.RawMessage) (string, error) {
	listFilesInput := ListFilesInput{}

	err := json.Unmarshal(input, &listFilesInput)
	if err != nil {
		panic(err)
	}

	dir := "."
	if listFilesInput.Path != "" {
		dir = listFilesInput.Path
	}

	var fileNames []string

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Adding this makes the code runs a lot faster.
		// Should have thought of this sooner :)
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if relPath != "." {
			if info.IsDir() {
				fileNames = append(fileNames, relPath+"/")
			} else {
				fileNames = append(fileNames, relPath)
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	result, err := json.Marshal(fileNames)
	if err != nil {
		return "", err
	}

	return string(result), nil
}
