package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/honganh1206/code-editing-agent/agent"
	"github.com/honganh1206/code-editing-agent/tools"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	key := os.Getenv("ANTHROPIC_API_KEY")

	client := anthropic.NewClient(option.WithAPIKey(key))
	scanner := bufio.NewScanner(os.Stdin)

	getUserMsg := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	tools := []tools.ToolDefinition{tools.ReadFileDefinition}

	agent := agent.New(&client, getUserMsg, tools)
	err = agent.Run(context.TODO()) // Empty context when unclear what context to use
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
