# TUI Architecture Overview

This document provides a comprehensive overview of the Terminal User Interface (TUI) architecture for the Claude chat application built with the `tview` library.

## Core Architecture

The TUI application is built around a component-based architecture where each UI element has specific responsibilities and communicates through event handlers and shared state.

### Component Hierarchy

```
Application (tview.Application)
└── Pages (tview.Pages) - Modal management
    ├── Main Page
    │   └── MainFlex (tview.Flex) - Root layout container
    │       ├── Top Row (Flex - Column direction)
    │       │   ├── Left Panel (1/4 width)
    │       │   │   ├── Search Field (Fixed height: 3)
    │       │   │   └── History List (Flexible)
    │       │   └── Right Panel (3/4 width)
    │       │       ├── Conversation View (Flexible)
    │       │       └── Question Input (Fixed height: 5)
    │       └── Help Bar (Fixed height: 1)
    ├── Edit Title Modal (Dynamic)
    └── Delete Confirmation Modal (Dynamic)
```

## Core Components

### 1. Application Container (`app`)
- **Type**: `tview.Application`
- **Purpose**: Main application controller and event loop
- **Responsibilities**:
  - Global key binding management
  - Focus coordination between components
  - Screen rendering and updates
  - Application lifecycle management

### 2. Page Manager (`pages`)
- **Type**: `tview.Pages`
- **Purpose**: Manages different screens and modal dialogs
- **Pages**:
  - `pageMain`: Primary interface
  - `pageEditTitle`: Title editing modal
  - `pageDeleteTitle`: Delete confirmation modal

### 3. Question Input Area (`textArea`)
- **Type**: `tview.TextArea`
- **Purpose**: Multi-line text input for user messages
- **Features**:
  - Message submission on Enter
  - ESC navigation to conversation view
  - Automatic clearing after submission
  - Disable/enable during API calls

### 4. Conversation History (`list`)
- **Type**: `tview.List`
- **Purpose**: Shows chronological list of past conversations
- **Features**:
  - Vim-style navigation (`j`/`k`)
  - Conversation selection and preview
  - Title editing (`e` key)
  - Conversation deletion (`d` key)
  - Focus-dependent selection highlighting

### 5. Conversation Display (`textView`)
- **Type**: `tview.TextView`
- **Purpose**: Shows current conversation messages
- **Features**:
  - Real-time streaming updates via `SetChangedFunc`
  - Color-coded messages (red for user, green for Claude)
  - Auto-scrolling to latest messages
  - Word wrapping for long messages
  - Navigation key bindings

### 6. Search Field (`searchInputField`)
- **Type**: `tview.InputField`
- **Purpose**: Full-text search through conversation history
- **Features**:
  - Real-time conversation filtering
  - Full-text search index integration
  - Results update the history list dynamically

### 7. Help Bar (`help`)
- **Type**: `tview.TextView`
- **Purpose**: Displays available keyboard shortcuts
- **Content**: Static help text with all available commands

## Layout System

### Main Layout Structure
The interface uses a nested `tview.Flex` system:

```
MainFlex (Vertical)
├── Content Row (Horizontal)
│   ├── Left Panel (Vertical, 25% width)
│   │   ├── Search (Fixed: 3 rows)
│   │   └── History (Flexible)
│   └── Right Panel (Vertical, 75% width)
│       ├── Conversation (Flexible)
│       └── Input (Fixed: 5 rows)
└── Help (Fixed: 1 row)
```

### Modal Dialog System
Modal dialogs are created using the `modal()` helper function that:
- Creates a centered overlay using nested Flex containers
- Calculates positioning based on current list scroll position
- Handles dynamic positioning relative to selected items

## Event Handling Architecture

### Key Binding System
The application uses a hierarchical key binding system:

1. **Component-Level Bindings**: Each component handles its own specific keys
2. **Global Bindings**: Function keys (F1-F4) work application-wide
3. **Modal Bindings**: Special handling for dialog interactions

### Focus Management Flow
```
Search Field (Ctrl+S)
    ↓ Enter
History List (F2, ESC from others)
    ↓ Enter / j,k navigation
Conversation View (F3, ESC from input)
    ↓ Enter
Question Input (F4, default focus)
    ↓ Enter (submit) → API call → back to input
```

### Event Handlers by Component

#### Question Input (`textArea`)
- `Enter`: Submit message → API call → stream response
- `ESC`: Switch to conversation view (if available)

#### History List (`list`)
- `j/k`: Vim-style navigation
- `Enter`: Select conversation and switch to input
- `e`: Edit conversation title (opens modal)
- `d`: Delete conversation (opens confirmation modal)
- `ESC`: Switch to search field

#### Conversation View (`textView`)
- `Enter`: Switch to question input
- `ESC`: Switch to history list

#### Global Application (`app`)
- `F1`: New chat (clear conversation, focus input)
- `F2`: Focus history list
- `F3`: Focus conversation view
- `F4`: Focus question input
- `Ctrl+S`: Focus search field

## State Management

### In-Memory State
- `m map[string]*Conversation`: Conversation cache indexed by title
- `isNewChat bool`: Tracks whether starting new conversation
- `hiddenItemCount int`: Scroll position tracking for modal positioning

### Persistent State
- **Database**: BuntDB with time-based indexing
- **File Location**: `~/.claude/history.db`
- **Locking**: File-based locking prevents multiple instances

### State Synchronization
1. **Load**: Database → In-memory map → UI list population
2. **Update**: User action → In-memory update → Database write → UI refresh
3. **Search**: Database query → Filtered results → UI update

## Data Flow

### Conversation Loading Flow
```
App Start → Database.View() → JSON.Unmarshal() → 
Memory Map Population → List.AddItem() → UI Ready
```

### Message Submission Flow
```
User Input → Validation → API Request (Streaming) → 
Real-time UI Updates → Response Complete → 
Database Storage → Memory Update
```

### Real-Time Streaming
The application handles streaming responses through:
1. **Goroutine**: Background API call processing
2. **Channels**: Communication between API handler and UI
3. **SetChangedFunc**: Automatic UI redraw on content updates
4. **Streaming Parser**: JSON line-by-line processing

## Error Handling & Edge Cases

### Focus Management Edge Cases
- Empty conversation list handling
- Modal dialog focus trapping
- Invalid state recovery (no conversations)

### API Integration
- Connection timeout handling
- Streaming response parsing
- Error message display in conversation view
- Input field disable/enable during processing

### Data Persistence
- Database connection failure recovery
- File locking timeout handling
- JSON marshaling/unmarshaling error handling
- Conversation title uniqueness management

## UI/UX Features

### Visual Feedback
- **Color Coding**: Red for user messages, green for Claude responses
- **Focus Indicators**: Different highlighting for focused vs selected items
- **Dynamic Updates**: Real-time streaming text display
- **Modal Overlays**: Centered dialogs with proper z-ordering

### Keyboard Shortcuts
- **Function Keys**: Quick component switching (F1-F4)
- **Vim-Style**: `j/k` navigation in lists
- **Common Patterns**: ESC for cancel/back, Enter for confirm/submit
- **Power User**: `e` for edit, `d` for delete, Ctrl+S for search

### Responsive Layout
- **Flexible Sizing**: Components adapt to terminal size
- **Fixed Elements**: Help bar and input areas maintain consistent height
- **Proportional Splits**: 25/75 split between history and conversation

## Integration Points

### Database Layer
- **BuntDB**: Embedded key-value store with JSON indexing
- **Time Index**: Conversations sorted by creation time
- **Atomic Updates**: Transaction-based consistency

### API Layer
- **Anthropic Claude**: REST API with streaming support
- **Authentication**: API key-based authentication
- **Request/Response**: JSON-based message format
- **Streaming**: Server-sent events for real-time updates

### File System
- **Configuration**: Environment variable-based API key
- **Data Storage**: User home directory (`.claude/`)
- **File Locking**: POSIX flock for single-instance enforcement

This architecture provides a responsive, feature-rich terminal interface that efficiently manages conversations while providing real-time interaction with the Claude AI assistant.