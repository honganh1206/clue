# Minimal TUI Guide for CLI Agent

This guide covers building a minimal Terminal User Interface (TUI) for a CLI agent with just two core components: a conversation view and a question input field.

## Core Components Overview

### 1. Application Container (`tview.NewApplication()`)
- Main TUI application instance
- Handles event dispatching and screen updates
- Manages focus between components

### 2. Conversation View (`tview.NewTextView()`)
- Displays chat messages between user and AI
- **No border** for clean appearance
- Supports dynamic colors for user/AI distinction
- Auto-scrolls to show latest messages
- Read-only display area

### 3. Question Input (`tview.NewTextArea()`)
- Multi-line text input for user questions
- **Has border** with "Question" title
- Handles Enter key for message submission
- Clears after sending message

## Layout System

### Simple Vertical Layout
```go
mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
    AddItem(conversationView, 0, 1, false).  // Takes most space
    AddItem(questionInput, 5, 1, true)       // Fixed height of 5 lines
```

### Layout Proportions
- **Conversation View**: Flexible height (grows/shrinks with terminal)
- **Question Input**: Fixed height (5 lines)
- **No sidebars**: Single column layout only

## Component Setup

### Conversation View Configuration
```go
conversationView := tview.NewTextView().
    SetDynamicColors(true).     // Enable [color] tags
    SetWordWrap(true).          // Wrap long lines
    SetChangedFunc(func() {     // Auto-redraw on updates
        app.Draw()
    })
// NO .SetBorder(true) - keep it borderless
```

### Question Input Configuration
```go
questionInput := tview.NewTextArea().
    SetTitle("Question").
    SetBorder(true)             // Show border with title
```

## Message Submission Flow

### 1. User Input Handling
- Capture Enter key press in question input
- Get text content from input field
- Validate (don't submit empty messages)
- Clear input field immediately

### 2. Message Display
- Add user message to conversation view
- Format with color: `[red::]You:[-]\n{message}`
- Add spacing between messages

### 3. AI Response Handling
- Send user message to AI service
- Display "AI is thinking..." or similar
- Stream response back to conversation view
- Format with color: `[green::]AI:[-]\n{response}`

### 4. Focus Management
- Keep focus on question input after submission
- Allow ESC key to switch focus to conversation view
- Allow Enter in conversation view to return focus to input

## Key Event Bindings

### Question Input Events
- **Enter**: Submit message to AI
- **ESC**: Switch focus to conversation view (if not empty)

### Conversation View Events  
- **Enter**: Switch focus back to question input
- **ESC**: No action (or switch to input)

### Global Events (Optional)
- **Ctrl+C**: Quit application
- **F4**: Focus on question input

## Implementation Checklist

### Phase 1: Basic Structure
- [ ] Create tview application
- [ ] Set up conversation view (no border)
- [ ] Set up question input (with border)
- [ ] Create vertical flex layout
- [ ] Add basic key bindings

### Phase 2: Message Flow
- [ ] Handle Enter key in question input
- [ ] Clear input after submission
- [ ] Display user message in conversation view
- [ ] Add message formatting with colors

### Phase 3: AI Integration
- [ ] Connect to AI service/API
- [ ] Handle streaming responses
- [ ] Display AI responses with formatting
- [ ] Add error handling

### Phase 4: Polish
- [ ] Improve focus management
- [ ] Add input validation
- [ ] Handle edge cases (empty messages, etc.)
- [ ] Test keyboard navigation

## Minimal Code Structure

```go
func main() {
    app := tview.NewApplication()
    
    // Components
    conversationView := tview.NewTextView().
        SetDynamicColors(true).
        SetWordWrap(true)
    
    questionInput := tview.NewTextArea().
        SetTitle("Question").
        SetBorder(true)
    
    // Layout
    mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
        AddItem(conversationView, 0, 1, false).
        AddItem(questionInput, 5, 1, true)
    
    // Event handling
    questionInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        if event.Key() == tcell.KeyEnter {
            // Handle message submission
            content := questionInput.GetText()
            // ... process message
            questionInput.SetText("", false)
            return nil
        }
        return event
    })
    
    // Start app
    if err := app.SetRoot(mainLayout, true).SetFocus(questionInput).Run(); err != nil {
        panic(err)
    }
}
```

## Message Display Format

### User Messages
```
You:
{user message content}

```

### AI Messages  
```
AI:
{ai response content}

```

This creates a clean, minimal chat interface focused purely on the conversation between user and AI.