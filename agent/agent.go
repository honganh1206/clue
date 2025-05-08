package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honganh1206/adrift/inference"
	"github.com/honganh1206/adrift/tools"
)

type Agent struct {
	engine         inference.Engine
	getUserMessage func() (string, bool)
	tools          []tools.ToolDefinition
	promptPath     string
}

func New(engine inference.Engine, getUserMsg func() (string, bool), tools []tools.ToolDefinition, promptPath string) *Agent {
	return &Agent{
		engine:         engine,
		getUserMessage: getUserMsg,
		tools:          tools,
		promptPath:     promptPath,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	conversation := []inference.Message{}

	// TODO: Get the name of the engine to interpolate to this string
	fmt.Println("Chat with Claude (use 'ctrl-c' to quit)")

	readUserInput := true

	for {
		if readUserInput {

			// ANSI escape code formatting to add colors and styles to terminal output for YOU
			fmt.Print("\u001b[94mYou\u001b[0m: ")
			userInput, ok := a.getUserMessage()
			if !ok {
				break
			}

			userMsg := inference.Message{
				Role: "user",
				Content: []inference.ContentBlock{
					{
						Type: "text", // TODO: Hardcoding type here?
						Text: userInput,
					},
				},
			}
			conversation = append(conversation, userMsg)
		}
		// fmt.Printf("DEBUG - Sending message to engine: %+v\n", conversation[len(conversation)-1])
		// Draw conclusions from prior knowledge
		agentMsg, err := a.engine.RunInference(ctx, conversation, a.tools)
		if err != nil {
			return err
		}
		// DEBUG: Print the current conversation as JSON
		// a.printConversationAsJSON(conversation)
		// DEBUG: Print the new agent message as JSON
		// a.printMessageAsJSON("New agent message", *agentMsg)

		conversation = append(conversation, *agentMsg)
		toolResults := []inference.ContentBlock{}

		for _, content := range agentMsg.Content {
			switch content.Type {
			// TODO: Add more, could be "code"?
			case "text":
				// TODO: Get the name of the engine to interpolate to this string
				fmt.Printf("\u001b[93mClaude\u001b[0m: %s\n", content.Text)
			case "tool_use":
				result := a.executeTool(content.ID, content.Name, content.Input)
				toolResults = append(toolResults, result)
			}
		}

		if len(toolResults) == 0 {
			// No tools were used, waiting for user input
			readUserInput = true
			continue
		}

		readUserInput = false

		toolResultMsg := inference.Message{
			Role:    "user", // MUST BE THIS
			Content: toolResults,
		}

		conversation = append(conversation, toolResultMsg)

		// DEBUG: Print the tool result message as JSON before adding to conversation
		// a.printMessageAsJSON("Tool Results Message", toolResultMsg)

		// fmt.Printf("DEBUG - conversation now has %d messages\n", len(conversation))

	}

	return nil
}

// FIXME: Should return anthropic.ContentBlockParamUnion
func (a *Agent) executeTool(id, name string, input json.RawMessage) inference.ContentBlock {
	var toolDef tools.ToolDefinition
	var found bool

	// fmt.Printf("DEBUG - Executing tool: ID=%s, Name=%s\n", id, name)
	// fmt.Printf("DEBUG - Tool input: %s\n", string(input))

	for _, tool := range a.tools {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}

	if !found {
		return inference.ContentBlock{
			Type:    "tool_result",
			ID:      id,
			Text:    "tool not found",
			IsError: true,
		}
	}

	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)

	response, err := toolDef.Function(input)

	if err != nil {
		return inference.ContentBlock{
			Type:    "tool_result",
			ID:      id,
			Text:    err.Error(),
			IsError: true,
		}
	}

	// fmt.Printf("DEBUG - Tool executed successfully. Result length: %d\n", len(response))
	// fmt.Printf("DEBUG - Result preview: %s\n", response[:min(30, len(response))]+"...")

	result := inference.ContentBlock{
		Type:    "tool_result",
		ID:      id,
		Text:    response,
		IsError: false,
	}
	// DEBUG: Print the tool result block
	// resultJSON, _ := json.MarshalIndent(result, "", "  ")
	// fmt.Printf("\n===== DEBUG: Tool Result Block =====\n")
	// fmt.Printf("ID: %s (should match tool_use ID)\n", id)
	// fmt.Println(string(resultJSON))
	// fmt.Printf("=====\n\n")

	// fmt.Printf("DEBUG - Tool result created: %+v\n", result)

	return result
}

// // Helper function to print a message as formatted JSON for debugging
// func (a *Agent) printMessageAsJSON(label string, message inference.Message) {
// 	jsonData, err := json.MarshalIndent(message, "", "  ")
// 	if err != nil {
// 		fmt.Printf("ERROR: Could not marshal message to JSON: %v\n", err)
// 		return
// 	}

// 	fmt.Printf("\n===== DEBUG: %s =====\n", label)
// 	fmt.Println(string(jsonData))
// 	fmt.Printf("=====\n\n")
// }

// // Helper function to print the entire conversation as JSON for debugging
// func (a *Agent) printConversationAsJSON(conversation []inference.Message) {
// 	fmt.Printf("\n===== DEBUG: Conversation (length: %d) =====\n", len(conversation))
// 	for i, msg := range conversation {
// 		jsonData, err := json.MarshalIndent(msg, "", "  ")
// 		if err != nil {
// 			fmt.Printf("ERROR: Could not marshal message %d to JSON: %v\n", i, err)
// 			continue
// 		}
// 		fmt.Printf("--- Message %d (%s) ---\n", i, msg.Role)
// 		fmt.Println(string(jsonData))
// 	}
// 	fmt.Printf("=====\n\n")
// }
