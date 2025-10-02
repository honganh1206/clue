package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/honganh1206/clue/agent"
	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/server/data/conversation"
	"github.com/rivo/tview"
)

func tui(ctx context.Context, agent *agent.Agent, conv *conversation.Conversation) error {
	defer agent.ShutdownMCPServers()

	app := tview.NewApplication()

	conversationView := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	displayConversationHistory(conversationView, conv)

	questionInput := tview.NewTextArea()
	questionInput.SetTitle("[blue::]Enter to send (ESC to focus conversation, Ctrl+C to quit)").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(conversationView, 0, 1, false).
		AddItem(questionInput, 5, 1, true)

	conversationView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			app.SetFocus(questionInput)
		}
		return event
	})

	questionInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyESC:
			if conversationView.GetText(false) != "" {
				app.SetFocus(conversationView)
			}
		case tcell.KeyEnter:
			content := questionInput.GetText()
			if strings.TrimSpace(content) == "" {
				return nil
			}
			questionInput.SetText("", false)
			questionInput.SetDisabled(true)

			fmt.Fprintf(conversationView, "\n[blue::]> %s\n\n", content)

			go func() {
				defer func() {
					questionInput.SetDisabled(false)
					app.Draw()
				}()

				fmt.Fprintf(conversationView, "")

				onDelta := func(delta string) {
					fmt.Fprintf(conversationView, "[white::]%s", delta)
				}

				err := agent.Run(ctx, content, onDelta)
				if err != nil {
					fmt.Fprintf(conversationView, "[red::]Error: %v[-]\n\n", err)
					return
				}

				fmt.Fprintf(conversationView, "\n\n")
				conversationView.ScrollToEnd()
			}()

			return nil
		}
		return event
	})

	return app.SetRoot(mainLayout, true).SetFocus(questionInput).Run()
}

func formatMessage(msg *message.Message) string {
	var result strings.Builder

	switch msg.Role {
	case message.UserRole:
		// TODO: Skip the user role message with tool result
		// since it prints out an unnecessary | character
		result.WriteString("[blue::]> ")
	case message.AssistantRole, message.ModelRole:
		result.WriteString("[white::]")
	}

	for _, block := range msg.Content {
		switch b := block.(type) {
		case message.TextBlock:
			result.WriteString(b.Text + "\n")
		case message.ToolUseBlock:
			result.WriteString(fmt.Sprintf("[green:]\u2713 %s %s\n", b.Name, b.Input))
		}
	}

	return result.String()
}

func displayConversationHistory(conversationView *tview.TextView, conv *conversation.Conversation) {
	if len(conv.Messages) == 0 {
		return
	}

	for _, msg := range conv.Messages {
		// This works, but is there a more efficient way?
		if msg.Role == message.UserRole && msg.Content[0].Type() == message.ToolResultType {
			continue
		}

		formattedMsg := formatMessage(msg)
		fmt.Fprintf(conversationView, "%s\n", formattedMsg)
	}

	conversationView.ScrollToEnd()
}
