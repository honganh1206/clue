package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/honganh1206/adrift/agent"
	"github.com/honganh1206/adrift/inference"
	"github.com/honganh1206/adrift/tools"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	key := os.Getenv("ANTHROPIC_API_KEY")

	// TODO: Make this more configurable to different prompts
	promptPath, err := filepath.Abs("./prompts/system.txt")

	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}

	// TODO: Command line args to configure model
	engineConfig := inference.EngineConfig{
		Type:       "anthropic",
		PromptPath: promptPath,
		Model:      anthropic.ModelClaude3_7SonnetLatest,
		MaxTokens:  1024,
		Key:        key,
	}

	engine, err := inference.CreateEngine(engineConfig)

	scanner := bufio.NewScanner(os.Stdin)

	getUserMsg := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	// Register tools
	tools := []tools.AnthropicToolDefinition{tools.ReadFileDefinition, tools.ListFilesDefinition}
	agent := agent.New(engine, getUserMsg, tools, promptPath)
	err = agent.Run(context.TODO()) // Empty context when unclear what context to use
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
