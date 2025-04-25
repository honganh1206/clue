package main

import (
	"bufio"
	"context"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
)

type Agent struct {
	client         *anthropic.Client
	getUserMessage func() (string, bool)
}

func main() {
	client := anthropic.NewClient()
	scanner := bufio.NewScanner(os.Stdin)

	getUserMsg := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	agent := NewAgent(&client, getUserMsg)
	err := agent.Run(context.TODO())
}
