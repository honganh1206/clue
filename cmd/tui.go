package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/honganh1206/clue/agent"
	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/progress"
	"github.com/honganh1206/clue/server/data/conversation"
	"github.com/honganh1206/clue/utils"
	"github.com/rivo/tview"
)

func tui(ctx context.Context, agent *agent.Agent, conv *conversation.Conversation) error {
	app := tview.NewApplication()

	conversationView := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	isFirstInput := len(conv.Messages) == 0
	if isFirstInput {
		conversationView.SetTextAlign(tview.AlignCenter)
		displayWelcomeMessage(conversationView)
	} else {
		displayConversationHistory(conversationView, conv)
	}

	relPath := displayRelativePath()

	questionInput := tview.NewTextArea()
	questionInput.SetTitle("[blue::]Enter to send (ESC to focus conversation)").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetDrawFunc(renderRelativePath(relPath))

	spinnerView := tview.NewTextView().
		SetDynamicColors(true).
		SetText("")

	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(conversationView, 0, 1, false).
		AddItem(questionInput, 5, 1, true).
		AddItem(spinnerView, 1, 0, false)

	conversationView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			app.SetFocus(questionInput)
		}
		return event
	})

	questionInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if isFirstInput && event.Key() == tcell.KeyRune {
			conversationView.Clear()
			conversationView.SetTextAlign(tview.AlignLeft)
			isFirstInput = false
		}

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

			fmt.Fprintf(conversationView, "[blue::]> %s\n\n", content)

			spinner := progress.NewSpinner(getRandomSpinnerMessage())
			firstDelta := true
			spinCh := make(chan bool, 1)

			go func() {
				ticker := time.NewTicker(50 * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case stop := <-spinCh:
						if stop {
							// Clear the spinner text to hide it from the UI when the agent finishes processing
							spinnerView.SetText("")
							app.Draw()
							return
						}
					case <-ticker.C:
						if spinner != nil {
							spinnerView.SetText(spinner.String())
							app.Draw()
						}
					}
				}
			}()

			go func() {
				defer func() {
					if spinner != nil {
						spinner.Stop()
					}
					spinCh <- true
					questionInput.SetDisabled(false)
					app.Draw()
				}()

				onDelta := func(delta string) {
					// Run spinner on tool result delta
					isToolResult := strings.HasPrefix(delta, "[green]") || strings.Contains(delta, "\u2713")

					if firstDelta && !isToolResult && spinner != nil {
						// Only stop spinner on actual LLM text response, not tool use
						spinner.Stop()
						// Signal the spinner goroutine to clear the spinner text (SetText("")) since the LLM has started responding
						spinCh <- true
						firstDelta = false
					}

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
		result.WriteString("\n[blue::]> ")
	case message.AssistantRole, message.ModelRole:
		result.WriteString("\n[white::]")
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

func displayWelcomeMessage(conversationView *tview.TextView) {
	// Add vertical padding to center the info box
	// This creates empty lines before the content
	fmt.Fprintf(conversationView, "\n\n\n\n\n\n\n\n")

	fmt.Fprintf(conversationView, "%s\n", utils.RenderBox(
		fmt.Sprintf("Clue v%s", Version),
		[]string{
			"Thank you for using Clue!",
			"",
			"Feel free to make a contribution - this app is open source",
			"",
			"Press Ctrl+C to exit",
		},
	))
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
		fmt.Fprintf(conversationView, "%s", formattedMsg)
	}

	conversationView.ScrollToEnd()
}

func getRandomSpinnerMessage() string {
	messages := []string{
		"Almost there...",
		"Hold on...",
		"Just a moment...",
		"Figuring it out...",
		"Communicating with the alien intelligence...",
		"Beep booping...",
		"Consulting the machines...",
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return messages[r.Intn(len(messages))]
}

// renderRelativePath returns a custom draw function for the question input area
// that overlays the relative path in the bottom-right corner of the input box
func renderRelativePath(relPath string) func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
	return func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
		pathText := fmt.Sprintf("[blue::]%s[-]", relPath)
		pathWidth := len(relPath)

		rightX := x + width - pathWidth - 2
		bottomY := y + height - 1

		if rightX > x && bottomY >= y {
			tview.Print(screen, pathText, rightX, bottomY, pathWidth, tview.AlignLeft, tcell.ColorDefault)
		}

		return x + 1, y + 1, width - 2, height - 2
	}
}

func displayRelativePath() string {
	cwd, err := os.Getwd()
	if err != nil {
		// Any chance that this could fail?
		cwd = "."
	}

	homeDir, _ := os.UserHomeDir()
	// What do the negative scenarios imply here?
	if homeDir == "" || !strings.HasPrefix(cwd, homeDir) {
		// We are not at home
		return ""
	}

	relativePath := strings.TrimPrefix(cwd, homeDir)
	if relativePath == "" {
		// In this case cwd == homeDir
		relativePath = "~"
	} else {
		parts := strings.Split(strings.Trim(relativePath, string(filepath.Separator)), string(filepath.Separator))
		// Pretty obvious from this point
		if len(parts) > 2 {
			relativePath = fmt.Sprintf("~/.../%s/%s", parts[len(parts)-2], parts[len(parts)-1])
		} else if len(parts) == 2 {
			relativePath = fmt.Sprintf("~/%s/%s", parts[0], parts[1])
		} else if len(parts) == 1 {
			relativePath = fmt.Sprintf("~/%s", parts[0])
		}
	}

	return relativePath
}
