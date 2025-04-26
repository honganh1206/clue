package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/joho/godotenv"
)

type Agent struct {
	client         *anthropic.Client
	getUserMessage func() (string, bool)
}

func NewAgent(client *anthropic.Client, getUserMsg func() (string, bool)) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMsg,
	}
}

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

	agent := NewAgent(&client, getUserMsg)
	err = agent.Run(context.TODO()) // Empty context when unclear what context to use
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

func (a *Agent) Run(ctx context.Context) error {
	conversation := []anthropic.MessageParam{}

	fmt.Println("Chat with Claude (use 'ctrl-c' to quit)")

	for {
		// ANSI escape code formatting to add colors and styles to terminal output for YOU
		fmt.Print("\u001b[94mYou\u001b[0m: ")
		userInput, ok := a.getUserMessage()
		if !ok {
			break
		}

		userMsg := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
		conversation = append(conversation, userMsg)

		agentMsg, err := a.runInference(ctx, conversation)
		if err != nil {
			return err
		}

		conversation = append(conversation, agentMsg.ToParam())

		for _, content := range agentMsg.Content {
			switch content.Type {
			// TODO: Add more, could be "code"?
			case "text":
				fmt.Printf("\u001b[93mClaude\u001b[0m: %s\n", content.Text)
			}
		}

	}

	return nil
}

func (a *Agent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	msg, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7SonnetLatest,
		MaxTokens: int64(1024),
		Messages:  conversation,
	})

	return msg, err
}
