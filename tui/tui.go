package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	// Components
	conversationView := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	questionInput := tview.NewTextArea()
	questionInput.SetTitle("Enter to send").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	// Layout
	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(conversationView, 0, 1, false).
		AddItem(questionInput, 5, 1, true)

	// Event handling
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
			// Clear input after submission
			questionInput.SetText("", false)
			// Disable input during processing
			questionInput.SetDisabled(true)

			// Display user message
			if conversationView.GetText(false) != "" {
				fmt.Fprintf(conversationView, "\n\n")
			}
			fmt.Fprintf(conversationView, "[red::]You:[-]\n%s\n\n", content)

			// Simple echo response (replace with actual AI integration)
			// TODO: Receive buffer/output streaming from agent.go? and stream the output token here?
			//
			fmt.Fprintf(conversationView, "[green::]AI:[-]\n%s\n\n", "Echo: "+content)

			// Scroll to end and re-enable input
			conversationView.ScrollToEnd()
			questionInput.SetDisabled(false)
			return nil
		}
		return event
	})

	// Start app
	if err := app.SetRoot(mainLayout, true).SetFocus(questionInput).Run(); err != nil {
		panic(err)
	}
}
