package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honganh1206/adrift/inference"
	"github.com/honganh1206/adrift/messages"
	"github.com/honganh1206/adrift/tools"
)

type Agent struct {
	model          inference.Model
	getUserMessage func() (string, bool)
	tools          []tools.ToolDefinition
	promptPath     string
}

func New(model inference.Model, getUserMsg func() (string, bool), tools []tools.ToolDefinition, promptPath string) *Agent {
	return &Agent{
		model:          model,
		getUserMessage: getUserMsg,
		tools:          tools,
		promptPath:     promptPath,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	conversation := []messages.MessageParam{}

	modelName := a.model.Name()

	fmt.Printf("Chat with %s (use 'ctrl-c' to quit)\n", modelName)

	readUserInput := true

	for {
		if readUserInput {

			fmt.Print("\u001b[94mYou\u001b[0m: ")
			userInput, ok := a.getUserMessage()
			if !ok {
				break
			}

			userMsg := messages.MessageRequest{
				MessageParam: messages.MessageParam{
					Role: messages.UserRole,
					Content: []messages.ContentBlock{
						{
							Type: tools.TextType,
							Text: userInput,
						},
					},
				},
			}
			conversation = append(conversation, userMsg.MessageParam)
		}

		// TODO: Update with something interactive?
		fmt.Printf("\u001b[93m%s\u001b[0m: ", modelName)
		agentMsg, err := a.model.RunInference(ctx, conversation, a.tools)
		if err != nil {
			return err
		}
		// a.printConversationAsJSON(conversation)

		conversation = append(conversation, *&agentMsg.MessageParam)
		toolResults := []messages.ContentBlock{}

		for _, content := range agentMsg.Content {
			switch content.Type {
			case tools.ToolUseType:
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

		toolResultMsg := messages.MessageRequest{
			MessageParam: messages.MessageParam{
				Role:    messages.UserRole,
				Content: toolResults,
			},
		}

		conversation = append(conversation, toolResultMsg.MessageParam)

		// fmt.Printf("DEBUG - conversation now has %d messages\n", len(conversation))

	}

	return nil
}

func (a *Agent) executeTool(id, name string, input json.RawMessage) messages.ContentBlock {
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
		return messages.ContentBlock{
			Type:    tools.ToolResultType,
			ID:      id,
			Text:    "tool not found",
			IsError: true,
		}
	}

	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)

	response, err := toolDef.Function(input)

	if err != nil {
		return messages.ContentBlock{
			Type:    tools.ToolResultType,
			ID:      id,
			Text:    err.Error(),
			IsError: true,
		}
	}

	// fmt.Printf("DEBUG - Tool executed successfully. Result length: %d\n", len(response))
	// fmt.Printf("DEBUG - Result preview: %s\n", response[:min(30, len(response))]+"...")

	result := messages.ContentBlock{
		Type:    tools.ToolResultType,
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

// Helper function to print the entire conversation as JSON for debugging
func (a *Agent) printConversationAsJSON(conversation []messages.MessageParam) {
	fmt.Printf("\n===== DEBUG: Conversation (length: %d) =====\n", len(conversation))
	for i, msg := range conversation {
		jsonData, err := json.MarshalIndent(msg, "", "  ")
		if err != nil {
			fmt.Printf("ERROR: Could not marshal message %d to JSON: %v\n", i, err)
			continue
		}
		fmt.Printf("--- Message %d (%s) ---\n", i, msg.Role)
		fmt.Println(string(jsonData))
	}
	fmt.Printf("=====\n\n")
}
