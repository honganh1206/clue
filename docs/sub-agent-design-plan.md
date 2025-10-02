# Sub-Agent System Design Plan (Ultra-Minimal Approach)

## Overview
This plan outlines the **most minimal** refactoring possible: add snapshot mode to LLMClient and **reuse the existing Agent struct** for sub-agents. No need for separate `subagent.go` file.

**Key Principles**:
- **Reuse the existing `Agent` struct completely**
- Add streaming flag to LLMClient interface
- Hardcode values (env vars later)
- Get it working first, optimize later

## Current State Analysis

### What Already Exists
1. **`agent/agent.go`**: Complete agent implementation
   - `Run()` method with tool-calling loop
   - Conversation saving (can be skipped if client is nil)
   - Full tool execution logic
   - **This is all we need!**

2. **`agent/subagent.go`**: Duplicate code (TO BE DELETED)
   - Just duplicates what `agent.go` already does
   - Not needed if we reuse Agent struct

3. **`tools/codebase_search.go`**: Broken implementation
   - References undefined `llmFactory`
   - Can be fixed by creating an Agent instance

### Current Problems
1. `codebase_search.go` can't create LLM clients
2. No snapshot mode in LLMClient
3. Code duplication between agent and subagent

---

## Ultra-Minimal Architecture

```
┌────────────────────────┐
│  Agent Struct          │
│  - Can save conv       │  ←── Main Agent (streaming=true)
│  - Can skip saving     │  ←── Sub-Agent (streaming=false)
│  - Same Run() logic    │
└────────────────────────┘

Just create different Agent instances!
```

**Insight**: The `Agent` struct already has everything we need. We just need to:
1. Create it with different LLM client (Haiku vs Sonnet)
2. Pass nil for `client` so it doesn't save conversations
3. Use different toolbox (limited tools)
4. Add streaming flag to control streaming vs snapshot

---

## Implementation Plan

### Step 1: Add Streaming Flag to LLMClient Interface

#### 1.1 Update `inference/inference.go`
**File**: `inference/inference.go`

**Change**: Add `streaming` parameter to `RunInferenceStream`

```go
type LLMClient interface {
    // Add streaming bool parameter
    RunInferenceStream(ctx context.Context, onDelta func(string), streaming bool) (*message.Message, error)
    
    SummarizeHistory(history []*message.Message, threshold int) []*message.Message
    TruncateMessage(msg *message.Message, threshold int) *message.Message
    ProviderName() string
    ToNativeHistory(history []*message.Message) error
    ToNativeMessage(msg *message.Message) error
    ToNativeTools(tools []*tools.ToolDefinition) error
}
```

**Notes**:
- Keep method name as `RunInferenceStream` (no rename needed)
- When `streaming=false`, implementation can ignore `onDelta`
- When `streaming=true`, calls `onDelta` for each text delta

---

#### 1.2 Update Provider Implementations

Update each provider's implementation to support the streaming flag:

**Anthropic** (`inference/anthropic.go`):
```go
func (c *AnthropicClient) RunInferenceStream(
    ctx context.Context, 
    onDelta func(string),
    streaming bool,
) (*message.Message, error) {
    params := // ... build params
    
    if streaming {
        // Existing streaming code
        stream := c.client.Messages.NewStreaming(ctx, params)
        // ... handle events, call onDelta
    } else {
        // NEW: Snapshot mode
        response, err := c.client.Messages.New(ctx, params)
        if err != nil {
            return nil, err
        }
        // Convert to message.Message and return
    }
}
```

**Gemini** (`inference/gemini.go`):
```go
func (c *GeminiClient) RunInferenceStream(
    ctx context.Context,
    onDelta func(string),
    streaming bool,
) (*message.Message, error) {
    if streaming {
        // Use GenerateContentStream()
    } else {
        // Use GenerateContent()
    }
}
```

---

### Step 2: Update Agent to Support Streaming Flag

#### 2.1 Update `agent/agent.go`

**File**: `agent/agent.go`

**Changes**:

1. **Add field to Agent struct** (line 18):
```go
type Agent struct {
    llm          inference.LLMClient
    toolBox      *tools.ToolBox
    conversation *conversation.Conversation
    client       *api.Client
    mcp          mcp.Config
    streaming    bool  // NEW: control streaming mode
}
```

2. **Update New() constructor** (line 26):
```go
func New(llm inference.LLMClient, conversation *conversation.Conversation, toolBox *tools.ToolBox, client *api.Client, mcpConfigs []mcp.ServerConfig, streaming bool) *Agent {
    agent := &Agent{
        llm:          llm,
        toolBox:      toolBox,
        conversation: conversation,
        client:       client,
        streaming:    streaming,  // NEW
    }
    // ... rest stays same
}
```

3. **Update streamResponse()** (line 293):
```go
func (a *Agent) streamResponse(ctx context.Context, onDelta func(string)) (*message.Message, error) {
    var streamErr error
    var msg *message.Message

    var wg sync.WaitGroup
    wg.Add(1)

    go func() {
        defer wg.Done()
        // USE: a.streaming flag
        msg, streamErr = a.llm.RunInferenceStream(ctx, onDelta, a.streaming)
    }()

    wg.Wait()

    if streamErr != nil {
        return nil, streamErr
    }

    return msg, nil
}
```

4. **Update saveConversation()** to handle nil client (line 261):
```go
func (a *Agent) saveConversation() error {
    // NEW: Skip if no client (for sub-agents)
    if a.client == nil {
        return nil
    }
    
    if len(a.conversation.Messages) > 0 {
        err := a.client.SaveConversation(a.conversation)
        if err != nil {
            fmt.Printf("DEBUG: Failed conversation details - ConversationID: %s\n", a.conversation.ID)
            return err
        }
    }

    return nil
}
```

---

### Step 3: Delete `agent/subagent.go`

**File**: `agent/subagent.go`

**Action**: **DELETE THIS ENTIRE FILE**

We don't need it anymore! The `Agent` struct can do everything.

---

### Step 4: Fix Codebase Search Tool

#### 4.1 Update `tools/codebase_search.go`

**File**: `tools/codebase_search.go`

**Complete rewrite using Agent struct**:
```go
package tools

import (
    "context"
    _ "embed"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/honganh1206/clue/agent"
    "github.com/honganh1206/clue/inference"
    "github.com/honganh1206/clue/message"
    "github.com/honganh1206/clue/schema"
    "github.com/honganh1206/clue/server/data/conversation"
)

//go:embed codebase_search.md
var codebaseSearchPrompt string

var CodebaseSearchDefinition = ToolDefinition{
    Name:        "codebase_search",
    Description: codebaseSearchPrompt,
    InputSchema: CodebaseSearchInputSchema,
    Function:    CodebaseSearch,
}

type CodebaseSearchInput struct {
    Query string `json:"query" jsonschema_description:"The search query describing what you're looking for in the codebase. Be specific and include context."`
}

var CodebaseSearchInputSchema = schema.Generate[CodebaseSearchInput]()

func CodebaseSearch(input json.RawMessage) (string, error) {
    var searchInput CodebaseSearchInput
    err := json.Unmarshal(input, &searchInput)
    if err != nil {
        return "", err
    }

    if searchInput.Query == "" {
        return "", fmt.Errorf("query parameter is required")
    }

    ctx := context.Background()

    // Create sub-agent LLM (hardcoded to Haiku for now)
    llm, err := inference.Init(ctx, inference.BaseLLMClient{
        Provider:   "anthropic",
        Model:      "claude-3-5-haiku-20241022",
        TokenLimit: 8192,
    })
    if err != nil {
        return "", fmt.Errorf("failed to initialize sub-agent LLM: %w", err)
    }

    // Create ephemeral conversation with system prompt
    conv := &conversation.Conversation{
        ID: "codebase_search_ephemeral",
        Messages: []*message.Message{
            {
                Role: message.SystemRole,
                Content: []message.ContentBlock{
                    message.NewTextBlock(codebaseSearchPrompt),
                },
            },
        },
    }

    // Setup limited toolbox for sub-agent (only safe read operations)
    subAgentToolBox := &ToolBox{
        Tools: []*ToolDefinition{
            &ReadFileDefinition,
            &GrepSearchDefinition,
            &ListFilesDefinition,
        },
    }

    // Create sub-agent using same Agent struct
    // Pass nil for client (won't save conversations), streaming=false for snapshot mode
    subAgent := agent.New(
        llm,
        conv,
        subAgentToolBox,
        nil,           // nil client = don't save conversations
        []mcp.ServerConfig{}, // no MCP servers for sub-agent
        false,         // streaming=false for snapshot mode
    )

    // Run sub-agent with the user query
    err = subAgent.Run(ctx, searchInput.Query, func(delta string) {
        // No-op: sub-agent doesn't stream output
    })
    if err != nil {
        return "", fmt.Errorf("sub-agent search failed: %w", err)
    }

    // Extract final response from last assistant message
    var result strings.Builder
    for i := len(conv.Messages) - 1; i >= 0; i-- {
        if conv.Messages[i].Role == message.AssistantRole {
            for _, content := range conv.Messages[i].Content {
                if textBlock, ok := content.(message.TextBlock); ok {
                    result.WriteString(textBlock.Text)
                }
            }
            break
        }
    }

    return result.String(), nil
}
```

**Key points**:
- **Reuses `Agent` struct** - no need for separate subagent code!
- Passes `nil` for client → skips conversation saving
- Passes `false` for streaming → uses snapshot mode
- Creates ephemeral conversation with system prompt
- Extracts final result from conversation messages
- Limited toolbox (Read, Grep, list_files only)

## Summary

This **ultra-minimal** plan eliminates code duplication by reusing the `Agent` struct:

**Just 4 changes**:
1. **Add streaming flag** to `RunInferenceStream()` method in LLMClient interface
2. **Update LLM implementations** (Anthropic, Gemini) to support snapshot mode
3. **Modify Agent struct** to accept streaming flag and handle nil client
4. **Fix codebase_search** to create an Agent instance (with nil client + streaming=false)
5. **Delete `agent/subagent.go`** - not needed anymore!

**Files to modify**:
- `inference/inference.go` - Add `streaming bool` parameter to interface
- `inference/anthropic.go` - Implement snapshot mode 
- `inference/gemini.go` - Implement snapshot mode
- `agent/agent.go` - Add streaming field, update constructor, handle nil client
- `tools/codebase_search.go` - Create Agent instance instead of using undefined factory

**Files to delete**:
- `agent/subagent.go` - **DELETE** (replaced by reusing Agent struct)

**Update existing code**:
- All callers of `agent.New()` must pass `streaming` parameter (pass `true` for existing behavior)

**Estimated effort**: 2-3 hours

**Benefits**:
- ✅ No code duplication
- ✅ Single source of truth for agent logic
- ✅ Simpler architecture
- ✅ Easy to understand and maintain

**What's NOT in this plan** (future improvements):
- Environment variable configuration (hardcode Haiku for now)
- Extensive testing (manual testing first)
- Documentation updates (once it works)
- Tool output truncation (can add back later if needed)
