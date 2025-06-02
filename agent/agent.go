package agent

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/honganh1206/clue/conversation"
	"github.com/honganh1206/clue/inference"
	"github.com/honganh1206/clue/prompts"
	"github.com/honganh1206/clue/tools"
	_ "github.com/mattn/go-sqlite3"
)

func Gen(conversationID string, modelConfig inference.ModelConfig, db *sql.DB) error {
	model, err := inference.Init(modelConfig)
	if err != nil {
		log.Fatalf("Failed to initialize model: %s", err.Error())
	}

	scanner := bufio.NewScanner(os.Stdin)
	getUserMsg := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	toolDefs := []tools.ToolDefinition{tools.ReadFileDefinition, tools.ListFilesDefinition, tools.EditFileDefinition}

	var a *Agent
	var conv *conversation.Conversation

	if conversationID != "" {
		conv, err = conversation.Load(conversationID, db)
		if err != nil {
			return err
		}
	} else {
		conv, err = conversation.New()
		if err != nil {
			return err
		}
	}
	a = New(model, getUserMsg, conv, toolDefs, prompts.System(), db)

	// In production, use Background() as the final root context()
	// For dev env, TODO for temporary scaffolding
	err = a.run(context.TODO())

	if err != nil {
		return err
	}

	return nil
}

type Agent struct {
	model          inference.Model
	getUserMessage func() (string, bool)
	tools          []tools.ToolDefinition
	promptPath     string
	conversation   *conversation.Conversation
	// FIXME: CRUD operations should be on its own, not a field in Agent
	db *sql.DB
}

func New(model inference.Model, getUserMsg func() (string, bool), conversation *conversation.Conversation, tools []tools.ToolDefinition, promptPath string, db *sql.DB) *Agent {
	return &Agent{
		model:          model,
		getUserMessage: getUserMsg,
		tools:          tools,
		promptPath:     promptPath,
		conversation:   conversation,
		db:             db,
	}
}

func (a *Agent) run(ctx context.Context) error {
	modelName := a.model.Name()

	fmt.Printf("Chat with %s (use 'ctrl-c' to quit)\n", modelName)

	readUserInput := true

	for {
		if readUserInput {

			fmt.Print("\u001b[94m>\u001b[0m ")
			userInput, ok := a.getUserMessage()
			if !ok {
				break
			}

			userMsg := conversation.MessageRequest{
				MessageParam: conversation.MessageParam{
					Role:    conversation.UserRole,
					Content: []conversation.ContentBlock{conversation.NewTextContentBlock(userInput)},
				},
			}
			a.conversation.Append(userMsg.MessageParam)
			a.saveConversation()
		}

		// TODO: Update with something interactive
		// fmt.Printf("\u001b[93m%s\u001b[0m: ", modelName)

		agentMsg, err := a.model.RunInference(ctx, a.conversation.Messages, a.tools)
		if err != nil {
			return err
		}

		a.conversation.Append(agentMsg.MessageParam)
		a.saveConversation()

		toolResults := []conversation.ContentBlock{}

		for _, content := range agentMsg.Content {
			switch c := content.(type) {
			case conversation.ToolUseContentBlock:
				result := a.executeTool(c.ID, c.Name, c.Input)
				toolResults = append(toolResults, result)
			}
		}

		if len(toolResults) == 0 {
			readUserInput = true
			continue
		}

		readUserInput = false

		toolResultMsg := conversation.MessageRequest{
			MessageParam: conversation.MessageParam{
				Role:    conversation.UserRole,
				Content: toolResults,
			},
		}

		a.conversation.Append(toolResultMsg.MessageParam)
		a.saveConversation()
	}

	return nil
}

func (a *Agent) executeTool(id, name string, input json.RawMessage) conversation.ContentBlock {
	var toolDef tools.ToolDefinition
	var found bool

	for _, tool := range a.tools {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}

	if !found {
		// TODO: Return proper error type
		errorMsg := "tool not found"
		return conversation.NewToolResultContentBlock(id, errorMsg, true)
	}

	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)

	response, err := toolDef.Function(input)

	if err != nil {
		return conversation.NewToolResultContentBlock(id, err.Error(), true)
	}

	return conversation.NewToolResultContentBlock(id, response, true)
}

func (a *Agent) saveConversation() error {
	// FIXME: Very drafty. Consider moving the db field out of Agent struct?
	err := a.conversation.SaveTo(a.db)
	if err != nil {
		// 4. Log any errors from history.Save to os.Stderr and return the error.
		fmt.Fprintf(os.Stderr, "Warning: could not save conversation to DB: %v\n", err)
		return err
	}

	return nil

}

// Helper function to print the entire conversation as JSON for debugging
func printConversationAsJSON(conversation []conversation.MessageParam) {
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
