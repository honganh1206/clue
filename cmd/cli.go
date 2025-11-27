package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/honganh1206/tinker/agent"
	"github.com/honganh1206/tinker/message"
	"github.com/honganh1206/tinker/server/data"
)

const (
	colorReset = "\033[0m"
	colorBlue  = "\033[34m"
	colorGreen = "\033[32m"
	colorRed   = "\033[31m"
)

func cli(ctx context.Context, a *agent.Agent) error {
	isFirstInput := len(a.Conv.Messages) == 0

	if isFirstInput {
		printWelcome()
	} else {
		printConversationHistory(a.Conv)
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf("\n%s> %s", colorBlue, colorReset)
		if !scanner.Scan() {
			break
		}
		// Space between input and output
		fmt.Println()

		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" {
			continue
		}

		onDelta := func(delta string) {
			// Convert tview color tags to ANSI codes
			delta = strings.ReplaceAll(delta, "[green::]", colorGreen)
			delta = strings.ReplaceAll(delta, "[red::]", colorRed)
			delta = strings.ReplaceAll(delta, "[blue::]", colorBlue)
			delta = strings.ReplaceAll(delta, "[white::]", colorReset)
			delta = strings.ReplaceAll(delta, "[-]", colorReset)
			fmt.Print(delta)
		}

		err := a.Run(ctx, userInput, onDelta)
		if err != nil {
			fmt.Printf("\n%sError: %v%s\n", colorRed, err, colorReset)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}

func printWelcome() {
	fmt.Println(formatWelcomeMessage())
}

func printConversationHistory(conv *data.Conversation) {
	if len(conv.Messages) == 0 {
		return
	}

	for _, msg := range conv.Messages {
		if msg.Role == message.UserRole && len(msg.Content) > 0 && msg.Content[0].Type() == message.ToolResultType {
			continue
		}

		formattedMsg := formatMessagePlain(msg)
		fmt.Print(formattedMsg)
	}
}

func formatMessagePlain(msg *message.Message) string {
	var result strings.Builder

	switch msg.Role {
	case message.UserRole:
		result.WriteString(fmt.Sprintf("\n%s> ", colorBlue))
	case message.AssistantRole, message.ModelRole:
		result.WriteString(fmt.Sprintf("\n%s ", colorReset))
	}

	for _, block := range msg.Content {
		switch b := block.(type) {
		case message.TextBlock:
			result.WriteString(b.Text + "\n")
		case message.ToolUseBlock:
			result.WriteString(fmt.Sprintf("%s\u2713 %s %s\n", colorGreen, b.Name, b.Input))
		}
	}

	return result.String()
}
