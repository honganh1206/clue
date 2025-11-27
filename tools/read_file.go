package tools

import (
	_ "embed"
	"encoding/json"
	"os"

	"github.com/honganh1206/clue/schema"
)

//go:embed read_file.md
var readFilePrompt string

type ReadFileInput struct {
	Path string `json:"path" jsonschema_description:"The absolute path of a file in the working directory."`
}

var ReadFileInputSchema = schema.Generate[ReadFileInput]()

var ReadFileDefinition = ToolDefinition{
	Name:        ToolNameReadFile,
	Description: readFilePrompt,
	InputSchema: ReadFileInputSchema, // Machine-readable description of the tool's input
	Function:    ReadFile,
}

func ReadFile(input ToolInput) (string, error) {
	readFileInput := ReadFileInput{}
	err := json.Unmarshal(input.RawInput, &readFileInput)
	if err != nil {
		panic(err)
	}

	content, err := os.ReadFile(readFileInput.Path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}