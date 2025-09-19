// TUI (Terminal User Interface) application for ChatGPT-like conversations
// This file contains the main TUI components and layout for an interactive chat interface
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/rivo/tview"
	"github.com/tidwall/buntdb"
)

// Role constants for message types in conversations

const (
	roleSystem    = "system"
	roleUser      = "user"
	roleAssistant = "assistant"

	// TODO: Move LLM-specific system messages to configurable prompts
	systemMessage = "You are Claude, an AI assistant created by Anthropic. Answer as concisely as possible."

	prefixSuggestTitle = "suggest me a short title for "

	// TUI page identifiers for navigation between different screens
	pageMain        = "main"
	pageEditTitle   = "editTitle"
	pageDeleteTitle = "deleteTitle"

	// Modal dialog button labels
	buttonCancel = "Cancel"
	buttonDelete = "Delete"

	// TODO: Move to configurable LLM provider settings
	maxTokens = 4097
)

var errTimeout = errors.New("timeout")

// Conversation represents a chat session with timestamp and message history
// TODO: Move to shared data models package for reuse across TUI and CLI
type Conversation struct {
	Time     int64     `json:"time"`
	Messages []Message `json:"messages"`
}

func main() {
	// TODO: Replace hardcoded OpenAI with configurable LLM provider system
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Please set `ANTHROPIC_API_KEY` environment variable. You can find your API key at https://console.anthropic.com/.")
		os.Exit(1)
	}

	// TODO: Replace with centralized database connection management
	// TODO: Make database path and type configurable
	home, err := homedir.Dir()
	if err != nil {
		log.Panic(err)
	}

	dbPath := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dbPath, 0700); err != nil {
		log.Panic(err)
	}

	dbFile := filepath.Join(dbPath, "history.db")
	f, err := os.OpenFile(dbFile, os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()

	// File locking to prevent multiple instances
	if err := flock(f, 1*time.Second); err != nil {
		if errors.Is(err, errTimeout) {
			fmt.Println("Another process is already running.")
		} else {
			fmt.Println(err)
		}
		return
	}

	// TODO: Replace buntdb with the project's standard database system
	db, err := buntdb.Open(dbFile)
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()
	db.CreateIndex("time", "*", buntdb.IndexJSON("time"))

	// TUI Component: Question input area - where users type their messages
	textArea := tview.NewTextArea()
	textArea.SetTitle("Question").SetBorder(true)

	// TUI Component: Conversation history list - shows all past conversations
	list := tview.NewList()
	list.SetTitle("History").SetBorder(true)

	// TUI Component: Main application container
	app := tview.NewApplication()

	// TUI Component: Conversation display area - shows the current chat messages
	textView := tview.NewTextView().
		SetChangedFunc(func() {
			app.Draw() // Redraw when content changes for streaming updates
		}).
		SetDynamicColors(true). // Enable colored text for user/assistant distinction
		SetRegions(true).
		SetWordWrap(true)
	textView.SetTitle("Conversation").SetBorder(true)
	// TUI Event Handling: Key bindings for conversation view
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyESC: // ESC: Switch focus to history list
			app.SetFocus(list)
		case tcell.KeyEnter: // Enter: Switch focus to question input
			app.SetFocus(textArea)
		}
		return event
	})

	// TUI State: In-memory conversation storage and UI state
	var (
		m         = make(map[string]*Conversation) // Maps conversation titles to conversation data
		isNewChat = true                           // Tracks whether we're starting a new conversation
	)
	list.SetSelectedFocusOnly(true) // Only highlight selected item when list has focus
	// TODO: Replace direct database access with conversation service layer
	// Load existing conversations from database and populate the history list
	db.View(func(tx *buntdb.Tx) error {
		err := tx.Descend("time", func(key, value string) bool {
			var c *Conversation
			if err := json.Unmarshal([]byte(value), &c); err == nil {
				m[key] = c // Store in memory for quick access

				// Add conversation to TUI history list
				list.AddItem(key, "", rune(0), func() {
					textView.SetText(toConversation(c.Messages))
				})
			}
			return true
		})
		return err
	})

	// TUI Event Handling: Update conversation view when history selection changes
	list.SetChangedFunc(func(index int, title string, secondaryText string, shortcut rune) {
		if c, ok := m[title]; ok {
			textView.SetText(toConversation(c.Messages))
		}
	})
	// TUI Event Handling: Handle conversation selection (when user presses Enter on list item)
	list.SetSelectedFunc(func(index int, title string, secondaryText string, shortcut rune) {
		list.SetSelectedFocusOnly(false) // Keep selection visible even when focus moves away
		if c, ok := m[title]; ok {
			textView.SetText(toConversation(c.Messages))
		}

		textView.ScrollToEnd() // Auto-scroll to latest message
		app.SetFocus(textArea) // Move focus to question input for immediate typing
	})

	// TUI Component: Page manager for handling modal dialogs and main view
	pages := tview.NewPages()

	// TUI Component: Input field for editing conversation titles
	editTitleInputField := tview.NewInputField().
		SetFieldWidth(40).
		SetAcceptanceFunc(tview.InputFieldMaxLength(40))

	// TUI Component: Modal dialog for confirming conversation deletion
	deleteTitleModal := tview.NewModal()
	deleteTitleModal.AddButtons([]string{buttonCancel, buttonDelete})

	// TUI Layout: Helper function to create centered modal dialogs
	// Calculates positioning based on current row to center the modal appropriately
	modal := func(p tview.Primitive, currentRow int) tview.Primitive {
		return tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
					AddItem(nil, 4+(currentRow*2), 1, false).
					AddItem(p, 1, 1, true).
					AddItem(nil, 0, 1, false), 0, 1, true).
				AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
					AddItem(nil, 0, 1, false).
					AddItem(nil, 5, 1, false), 0, 3, false), 0, 1, true).
			AddItem(nil, 1, 1, false)
	}

	// TUI Component: Search input field for finding conversations
	searchInputField := tview.NewInputField()
	searchInputField.SetTitle("Search")
	searchInputField.
		SetFieldWidth(50).
		SetAcceptanceFunc(tview.InputFieldMaxLength(50))
	searchInputField.SetBorder(true)
	// TUI Event Handling: Search functionality when user presses Enter in search field
	searchInputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			// TODO: Replace with conversation service search functionality
			titles := make([]string, 0, len(m))
			db.View(func(tx *buntdb.Tx) error {
				err := tx.Descend("time", func(key, value string) bool {
					titles = append(titles, key)
					return true
				})
				return err
			})

			text := searchInputField.GetText()
			if text != "" {
				// Use full-text search index to find matching conversations
				idx := make(index)
				idx.add(titles)
				r := idx.search(text)
				list.Clear()
				// Populate list with search results
				for _, i := range r {
					list.AddItem(titles[i], "", rune(0), func() {
						if c, ok := m[titles[i]]; ok {
							textView.SetText(toConversation(c.Messages))
						}
					})
				}
			} else {
				// Empty search - show all conversations
				list.Clear()
				for i := range titles {
					list.AddItem(titles[i], "", rune(0), func() {
						if c, ok := m[titles[i]]; ok {
							textView.SetText(toConversation(c.Messages))
						}
					})
				}
			}
			if list.GetItemCount() > 0 {
				app.SetFocus(list) // Move focus to results
			}
		}
	})

	// TUI State: Track scrolling in the history list for proper modal positioning
	var hiddenItemCount int

	// TUI Event Handling: Key bindings for history list navigation and actions
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyESC: // ESC: Switch focus to search field
			app.SetFocus(searchInputField)
		}

		_, _, _, height := list.GetInnerRect()

		// Vim-like navigation keys
		switch event.Rune() {
		case 'j': // Move down in list
			if list.GetCurrentItem() < list.GetItemCount() {
				list.SetCurrentItem(list.GetCurrentItem() + 1)
			}

			// Track scrolling for modal positioning
			if list.GetCurrentItem() >= height/2 {
				hiddenItemCount = list.GetCurrentItem() + 1 - (height / 2)
			}
		case 'k': // Move up in list
			if list.GetCurrentItem() > 0 {
				list.SetCurrentItem(list.GetCurrentItem() - 1)
			}

			// Track scrolling for modal positioning
			if list.GetCurrentItem()+1 == hiddenItemCount {
				hiddenItemCount--
			}
		case 'e': // Edit conversation title
			currentIndex := list.GetCurrentItem()
			currentTitle, _ := list.GetItemText(currentIndex)
			editTitleInputField.
				SetText(currentTitle).
				SetDoneFunc(func(key tcell.Key) {
					switch key {
					case tcell.KeyESC: // Cancel editing
						pages.HidePage(pageEditTitle)
						app.SetFocus(list)
					case tcell.KeyEnter: // Save new title
						newTitle := editTitleInputField.GetText()
						if newTitle != currentTitle {
							// TODO: Replace direct database operations with conversation service
							c, _ := json.Marshal(m[currentTitle])
							if err == nil {
								db.Update(func(tx *buntdb.Tx) error {
									_, _, err := tx.Set(newTitle, string(c), nil)
									if err != nil {
										return err
									}

									tx.Delete(currentTitle)

									// Update in-memory storage
									m[newTitle] = m[currentTitle]
									delete(m, currentTitle)

									// Update TUI list display
									list.RemoveItem(currentIndex)
									list.InsertItem(currentIndex, newTitle, "", rune(0), nil)
									list.SetCurrentItem(currentIndex)

									return nil
								})
							}
						}
						pages.HidePage(pageEditTitle)
						app.SetFocus(list)
					}
				}).
				SetBorder(false)
			// Show edit modal positioned relative to current selection
			pages.AddPage(pageEditTitle, modal(editTitleInputField, list.GetCurrentItem()-hiddenItemCount), true, false)
			pages.ShowPage(pageEditTitle)
		case 'd': // Delete conversation
			currentIndex := list.GetCurrentItem()
			currentTitle, _ := list.GetItemText(currentIndex)

			// Show confirmation modal for deletion
			deleteTitleModal.SetText(fmt.Sprintf("Are you sure you want to delete \"%s\"?", currentTitle)).
				SetFocus(0).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					switch buttonLabel {
					case buttonCancel: // Cancel deletion
						pages.HidePage(pageDeleteTitle)
						app.SetFocus(list)

					case buttonDelete: // Confirm deletion
						list.RemoveItem(currentIndex)

						// Handle UI state when no conversations remain
						if list.GetItemCount() == 0 {
							textView.Clear()
							list.SetCurrentItem(-1)
							app.SetFocus(textArea)
						}

						// TODO: Replace direct database operations with conversation service
						db.Update(func(tx *buntdb.Tx) error {
							_, err := tx.Delete(currentTitle)
							return err
						})
						delete(m, currentTitle) // Remove from memory

						pages.HidePage(pageDeleteTitle)
						if list.GetItemCount() > 0 {
							app.SetFocus(list)
						} else {
							app.SetFocus(textArea)
						}
					}
				}).
				SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
					switch event.Key() {
					case tcell.KeyESC: // ESC also cancels deletion
						pages.HidePage(pageDeleteTitle)
						app.SetFocus(list)
					}
					return event
				})
			pages.ShowPage(pageDeleteTitle)
		}

		return event
	})

	// TUI Event Handling: Question input area key bindings and message submission
	textArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyESC: // ESC: Switch to conversation view if available
			if textView.GetText(false) != "" || !isNewChat {
				app.SetFocus(textView)
			}
		case tcell.KeyEnter: // Enter: Submit message to LLM
			content := textArea.GetText()
			if strings.TrimSpace(content) == "" {
				return nil // Don't submit empty messages
			}
			textArea.SetText("", false) // Clear input after submission
			textArea.SetDisabled(true)  // Disable input while processing

			// TODO: Replace direct LLM communication with agent service layer
			titleCh := make(chan string)
			messages := make([]Message, 0)
			if textView.GetText(false) == "" {
				// Starting new conversation - add system message and request title generation
				messages = append(messages, Message{
					Role:    roleSystem,
					Content: systemMessage,
				})

				// TODO: Move title generation to agent service with configurable prompts
				go func() {
					resp, err := createChatCompletion([]Message{
						{
							Role:    roleUser,
							Content: prefixSuggestTitle + content,
						},
					}, false)
					if err != nil {
						log.Panic(err)
					}
					defer resp.Body.Close()

					body, err := io.ReadAll(resp.Body)
					if err != nil {
						log.Panic(err)
					}

					var titleResp *Response
					if err := json.Unmarshal(body, &titleResp); err == nil && len(titleResp.Content) > 0 {
						titleCh <- titleResp.Content[0].Text
					}
				}()
			} else {
				// Continuing existing conversation
				isNewChat = false

				title, _ := list.GetItemText(list.GetCurrentItem())
				if c, ok := m[title]; ok {
					messages = c.Messages
				}

				textView.ScrollToEnd()
				fmt.Fprintf(textView, "\n\n")
			}

			// Add user message to conversation
			messages = append(messages, Message{
				Role:    roleUser,
				Content: content,
			})

			// TODO: Replace token counting with agent service token management
			numTokens, err := NumTokensFromMessages(messages, "")
			if err != nil {
				log.Println(err)
				return nil
			}

			// Handle context length overflow by starting new conversation
			if numTokens > maxTokens {
				isNewChat = true
				title, _ := list.GetItemText(list.GetCurrentItem())
				go func() {
					titleCh <- addSuffixNumber(title)
				}()

				// Reset messages with context from previous conversation title
				messages = []Message{
					{
						Role:    roleSystem,
						Content: systemMessage,
					},
					{
						Role:    roleUser,
						Content: fmt.Sprintf("%s: %s", title, content),
					},
				}

				textView.Clear()
			}

			// TUI Display: Show user message in conversation view
			fmt.Fprintln(textView, "[red::]You:[-]")
			fmt.Fprintf(textView, "%s\n\n", content)

			// TODO: Replace direct LLM API calls with agent service streaming
			respCh := make(chan string)
			errCh := make(chan error, 1)
			go func() {
				resp, err := createChatCompletion(messages, true)
				if err != nil {
					errCh <- err
				}

				// Stream response from LLM API
				reader := bufio.NewReader(resp.Body)
				for {
					line, err := reader.ReadBytes('\n')
					if err != nil {
						if errors.Is(err, io.EOF) {
							close(respCh)
							return
						} else {
							errCh <- err
						}
					}

					var streamingResp *StreamingResponse
					if err := json.Unmarshal(bytes.TrimPrefix(line, []byte("data: ")), &streamingResp); err == nil {
						if streamingResp.Type == "content_block_delta" {
							respCh <- streamingResp.Delta.Text
						}
					}
				}
			}()

			// Check for immediate errors
			select {
			case err := <-errCh:
				log.Println("received error:", err)
				return nil
			default:
			}

			// TUI Display: Show assistant response header
			fmt.Fprintln(textView, "[green::]Claude:[-]")
			go func() {
				// Stream assistant response to TUI and collect full content
				var fullContent strings.Builder
				for deltaContent := range respCh {
					fmt.Fprintf(textView, deltaContent) // Real-time streaming display
					fullContent.WriteString(deltaContent)
				}

				// Add complete assistant message to conversation
				messages = append(messages, Message{
					Role:    roleAssistant,
					Content: fullContent.String(),
				})

				// Update conversation list for new conversations
				if list.GetItemCount() == 0 || isNewChat {
					list.InsertItem(0, strings.Trim(<-titleCh, "\""), "", rune(0), nil)
					list.SetCurrentItem(0)

					isNewChat = false
				}

				// TODO: Replace direct database operations with conversation service
				title, _ := list.GetItemText(list.GetCurrentItem())
				c := &Conversation{
					Time: time.Now().Unix(),
				}
				// Exclude system message from persistent storage
				// TODO: Do we need this? Since we save messages atomically?
				if messages[0].Role == roleSystem {
					c.Messages = messages[1:]
				} else {
					c.Messages = messages
				}

				// Save conversation to database
				value, err := json.Marshal(c)
				if err != nil {
					log.Panic(err)
				}
				db.Update(func(tx *buntdb.Tx) error {
					_, _, err := tx.Set(title, string(value), nil)
					return err
				})
				m[title] = c // Update in-memory cache

				fmt.Fprintf(textView, "\n\n")
				textArea.SetDisabled(false) // Re-enable input for next message
			}()

			return nil
		}
		return event
	})

	// TUI Event Handling: Global key bindings for application-wide navigation
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if textView.GetText(false) != "" {
			list.SetSelectedFocusOnly(false) // Keep selection visible when conversation is active
		}

		// Function key shortcuts for quick navigation between TUI components
		switch event.Key() {
		case tcell.KeyF1: // F1: Start new conversation
			isNewChat = true
			list.SetSelectedFocusOnly(true)
			textView.Clear()
			app.SetFocus(textArea)
		case tcell.KeyF2: // F2: Focus on conversation history
			if list.GetItemCount() > 0 {
				app.SetFocus(list)
				title, _ := list.GetItemText(list.GetCurrentItem())
				textView.SetText(toConversation(m[title].Messages))
			}
		case tcell.KeyF3: // F3: Focus on conversation view
			if textView.GetText(false) != "" {
				app.SetFocus(textView)
			}
		case tcell.KeyF4: // F4: Focus on question input
			app.SetFocus(textArea)
		case tcell.KeyCtrlS: // Ctrl+S: Focus on search
			if list.GetItemCount() > 0 {
				app.SetFocus(searchInputField)
			}
		default:
			return event
		}
		return nil
	})

	// TUI Component: Help bar showing available keyboard shortcuts
	help := tview.NewTextView().SetRegions(true).SetDynamicColors(true)
	help.SetText("F1: new chat, F2: history, F3: conversation, F4: question, enter: submit, ctrl-s: search, j/k: down/up, e: edit, d: delete, ctrl-f/b: page down/up, ctrl-c: quit").SetTextAlign(tview.AlignCenter)

	// TUI Layout: Main application layout with flexible sizing
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
						AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
							AddItem(searchInputField, 3, 1, false).   // Search field: fixed height
							AddItem(list, 0, 1, false), 0, 1, false). // History list: flexible height, 1/4 width
						AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
							AddItem(textView, 0, 1, false).                             // Conversation view: flexible height
							AddItem(textArea, 5, 1, false), 0, 3, false), 0, 1, false). // Question input: fixed height, 3/4 width
		AddItem(help, 1, 1, false) // Help bar: fixed height

	// TUI Setup: Configure page system and start application
	pages.
		AddPage(pageMain, mainFlex, true, true).
		AddPage(pageEditTitle, modal(editTitleInputField, list.GetCurrentItem()), true, false).
		AddPage(pageDeleteTitle, deleteTitleModal, true, false)

	// Start the TUI application with focus on question input
	if err := app.SetRoot(pages, true).SetFocus(textArea).Run(); err != nil {
		panic(err)
	}
}

// flock implements file locking to prevent multiple TUI instances from accessing the same database
// TODO: Move to shared utility package for reuse across CLI and TUI
func flock(f *os.File, timeout time.Duration) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
			if err == nil {
				return nil
			} else if err != syscall.EWOULDBLOCK {
				return err
			}
		case <-timer.C:
			return errTimeout
		}
	}
}

// NumTokensFromMessages calculates token count for conversation context management
// TODO: Move to agent service with support for different LLM providers and their token counting methods
func NumTokensFromMessages(messages []Message, model string) (int, error) {
	// Approximate token count for Claude (roughly 4 characters per token)
	numTokens := 0
	for _, message := range messages {
		numTokens += len(message.Content) / 4
		numTokens += len(message.Role) / 4
		numTokens += 10 // overhead per message
	}
	numTokens += 50 // system message overhead
	return numTokens, nil
}

// addSuffixNumber generates unique conversation titles when context overflow creates new conversations
// TODO: Move to conversation service for better title management
func addSuffixNumber(title string) string {
	re := regexp.MustCompile(`(.*)\s-\s(\d+)$`)
	match := re.FindStringSubmatch(title)
	if match == nil {
		return fmt.Sprintf("%s - %d", title, 2)
	}
	suffixNumber, _ := strconv.Atoi(match[2])
	return fmt.Sprintf("%s - %d", match[1], suffixNumber+1)
}

// TODO: Replace hardcoded OpenAI constants with configurable LLM provider system
const (
	messagesURL       = "https://api.anthropic.com/v1/messages"
	claude3HaikuModel = "claude-3-haiku-20240307"
	anthropicVersion  = "2023-06-01"
)

// createChatCompletion makes direct API calls to OpenAI
// TODO: Replace with agent service LLM abstraction layer supporting multiple providers
func createChatCompletion(messages []Message, stream bool) (*http.Response, error) {
	// Separate system message from regular messages for Claude API
	var systemMsg string
	var apiMessages []Message

	for _, msg := range messages {
		if msg.Role == roleSystem {
			systemMsg = msg.Content
		} else {
			apiMessages = append(apiMessages, msg)
		}
	}

	reqBody, err := json.Marshal(&Request{
		Model:     claude3HaikuModel,
		MaxTokens: 4096,
		System:    systemMsg,
		Messages:  apiMessages,
		Stream:    stream,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, messagesURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Add("x-api-key", os.Getenv("ANTHROPIC_API_KEY"))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("anthropic-version", anthropicVersion)

	client := &http.Client{}
	return client.Do(req)
}

// TODO: Move all LLM API types to shared message package for reuse across CLI and TUI

// Request represents the structure for LLM API requests
type Request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
	Stream    bool      `json:"stream,omitempty"`
}

// Message represents a single message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response represents non-streaming LLM API responses
type Response struct {
	Id      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// StreamingResponse represents streaming LLM API responses for real-time display
type StreamingResponse struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta,omitempty"`
	ContentBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content_block,omitempty"`
}

// toConversation formats message history for TUI display with color coding
// TODO: Move to TUI utilities package and make colors/labels configurable
func toConversation(messages []Message) string {
	contents := make([]string, 0)
	for _, msg := range messages {
		switch msg.Role {
		case roleUser:
			msg.Content = fmt.Sprintf("[red::]You:[-]\n%s", msg.Content)
		case roleAssistant:
			msg.Content = fmt.Sprintf("[green::]Claude:[-]\n%s", msg.Content)
		}
		contents = append(contents, msg.Content)
	}
	return strings.Join(contents, "\n\n")
}
