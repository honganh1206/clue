package cmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/honganh1206/tinker/agent"
	"github.com/honganh1206/tinker/message"
	"github.com/honganh1206/tinker/server/data"
	"github.com/honganh1206/tinker/ui"
	"github.com/honganh1206/tinker/utils"
	"github.com/rivo/tview"
)

func tui(ctx context.Context, agent *agent.Agent, ctl *ui.Controller) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	app := tview.NewApplication()

	conversationView := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		}).ScrollToEnd()

	isFirstInput := len(agent.Conv.Messages) == 0
	if isFirstInput {
		conversationView.SetTextAlign(tview.AlignCenter)
		displayWelcomeMessage(conversationView)
	} else {
		displayConversationHistory(conversationView, agent.Conv)
	}
	relPath := displayRelativePath()

	questionInput := tview.NewTextArea()
	questionInput.SetTitle("[blue]Enter to send (ESC to focus conversation)").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetDrawFunc(renderRelativePath(relPath))
	questionInput.SetFocusFunc(func() {
		questionInput.SetBorderColor(tcell.ColorGreen)
	})
	questionInput.SetBlurFunc(func() {
		questionInput.SetBorderColor(tcell.ColorWhite)
	})

	spinnerView := tview.NewTextView().
		SetDynamicColors(true).
		SetText("")

	planView := tview.NewTextView().
		SetDynamicColors(true)
	planView.SetBorder(true)

	inputFlex := tview.NewFlex()

	inputHeight := 5
	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(conversationView, 0, 1, false).
		AddItem(inputFlex, inputHeight, 0, true).
		AddItem(spinnerView, 1, 0, false)

	conversationView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			app.SetFocus(questionInput)
		}
		return event
	})

	// TODO: This should be in a separate function
	renderPlan := func(s *ui.State) {
		inputFlex.Clear()
		plan := s.Plan
		if plan == nil || len(plan.Steps) == 0 {
			inputFlex.AddItem(questionInput, 0, 1, true)
			mainLayout.ResizeItem(inputFlex, 5, 0)
		} else {
			planView.SetText(formatPlanSteps(plan))
			inputFlex.
				AddItem(questionInput, 0, 1, true).
				AddItem(planView, 0, 1, false)

			newHeight := max(5, len(plan.Steps)+2)
			mainLayout.ResizeItem(inputFlex, newHeight, 0)
		}
	}

	initialState := &ui.State{Plan: agent.Plan}
	renderPlan(initialState)

	go func() {
		updateCh := ctl.Subscribe()

		for s := range updateCh {
			renderPlan(s)
		}
	}()

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

			// User input
			fmt.Fprintf(conversationView, "[blue::i]> %s\n\n", content)

			// Should call this only
			go streamContent(app, ctx, conversationView, questionInput, spinnerView, content, agent)

			return nil
		}
		return event
	})

	if err := app.SetRoot(mainLayout, true).EnableMouse(true).SetFocus(questionInput).Run(); err != nil {
		panic(err)
	}

	return nil
}

func formatMessage(msg *message.Message, nextMsg *message.Message) string {
	var result strings.Builder

	switch msg.Role {
	case message.UserRole:
		result.WriteString("\n[blue::]> ")
	case message.AssistantRole, message.ModelRole:
		result.WriteString("\n[white::]")
	}

	toolErrors := make(map[string]bool)
	if nextMsg != nil && nextMsg.Role == message.UserRole {
		for _, block := range nextMsg.Content {
			if tr, ok := block.(message.ToolResultBlock); ok && tr.IsError {
				toolErrors[tr.ToolUseID] = true
			}
		}
	}

	for _, block := range msg.Content {
		switch b := block.(type) {
		case message.TextBlock:
			result.WriteString(b.Text + "\n")
		case message.ToolUseBlock:
			isError := toolErrors[b.ID]
			inputBytes, _ := json.Marshal(b.Input)
			result.WriteString(agent.FormatToolResultMessage(b.Name, inputBytes, isError))
		}
	}

	return result.String()
}

func formatWelcomeMessage() string {
	return utils.RenderBox(
		fmt.Sprintf("Tinker v%s", Version),
		[]string{
			"Thank you for using Tinker!",
			"",
			"Feel free to make a contribution - this app is open source",
			"",
			"Press Ctrl+C to exit",
		},
	)
}

func displayWelcomeMessage(conversationView *tview.TextView) {
	// Add vertical padding to center the info box
	// This creates empty lines before the content
	fmt.Fprintf(conversationView, "\n\n\n\n\n\n\n\n")

	fmt.Fprintf(conversationView, "%s\n", formatWelcomeMessage())
}

func displayConversationHistory(conversationView *tview.TextView, conv *data.Conversation) {
	if len(conv.Messages) == 0 {
		return
	}

	for i, msg := range conv.Messages {
		if msg.Role == message.UserRole && len(msg.Content) > 0 && msg.Content[0].Type() == message.ToolResultType {
			continue
		}

		var nextMsg *message.Message
		if i+1 < len(conv.Messages) {
			nextMsg = conv.Messages[i+1]
		}

		formattedMsg := formatMessage(msg, nextMsg)
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

func formatPlanSteps(plan *data.Plan) string {
	if plan == nil || len(plan.Steps) == 0 {
		return ""
	}

	var result strings.Builder

	for _, step := range plan.Steps {
		statusColor := "white"
		statusSymbol := "○"
		if strings.ToUpper(step.Status) == "DONE" {
			statusColor = "green"
			statusSymbol = "✓"
		}
		result.WriteString(fmt.Sprintf("[%s::]%s %s[-]\n", statusColor, statusSymbol, step.Description))
	}

	return result.String()
}

// TODO: The number + order of arguments passed in here are atrocious.
// Are we going to make it C-like? Can we make it better?
func streamContent(app *tview.Application, ctx context.Context, conversationView *tview.TextView, questionInput *tview.TextArea, spinnerView *tview.TextView, content string, agent *agent.Agent) {
	spinner := ui.NewSpinner(getRandomSpinnerMessage(), ui.SpinnerStar)

	stop := startSpinner(app, ctx, spinner, spinnerView)
	go func() {
		defer func() {
			stop <- true
			questionInput.SetDisabled(false)
			app.Draw()
		}()

		onDelta := func(delta string) {
			// conversationView is append only, meaning we can replace the text that has already printed out
			// so bye bye printing out tool being executed
			fmt.Fprintf(conversationView, "[white]%s", delta)
		}

		err := agent.Run(ctx, content, onDelta)
		if err != nil {
			fmt.Fprintf(conversationView, "[red::]Error: %v[-]\n\n", err)
			return
		}

		fmt.Fprintf(conversationView, "\n\n")
		conversationView.ScrollToEnd()
	}()
}

func startSpinner(app *tview.Application, ctx context.Context, spinner *ui.Spinner, spinnerView *tview.TextView) chan bool {
	stop := make(chan bool)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				spinner.Stop()
				spinnerView.SetText("")
				return
			default:
				spinnerView.SetText(spinner.String())
				app.Draw()
			}
		}
	}()

	return stop
}