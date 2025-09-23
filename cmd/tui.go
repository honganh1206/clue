package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/honganh1206/clue/agent"
	"github.com/rivo/tview"
)

func TUI(ctx context.Context, a *agent.Agent) error {
	app := tview.NewApplication()

	conversationView := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	questionInput := tview.NewTextArea()
	questionInput.SetTitle("Enter to send (ESC to focus conversation, Ctrl+C to quit)").
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

			// Display user message
			fmt.Fprintf(conversationView, "[azure::]> %s\n\n", content)
			conversationView.ScrollToEnd()

			// Process with agent in background
			go func() {
				defer func() {
					questionInput.SetDisabled(false)
					app.Draw()
				}()

				// Stream agent response
				fmt.Fprintf(conversationView, "")

				onDelta := func(delta string) {
					fmt.Fprintf(conversationView, "[white::] %s", delta)
					app.Draw()
				}

				err := a.Run(ctx, content, onDelta)
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
