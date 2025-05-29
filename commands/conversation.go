package commands

import (
	"fmt"
	"log"
	"time"

	"github.com/honganh1206/clue/conversation"
	"github.com/spf13/cobra"
)

var conversationCmd = &cobra.Command{
	Use:   "conversation",
	Short: "Show conversations",
}

func NewConversationCmd() *cobra.Command {
	conversationCmd.AddCommand(conversationListCmd)

	return conversationCmd
}

var conversationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all conversations",
	RunE:  ListHandler,
}

func ListHandler(cmd *cobra.Command, args []string) error {
	db, err := conversation.InitDB()

	if err != nil {
		log.Fatalf("Failed to initialize database: %s", err.Error())
		return err
	}

	defer db.Close()

	conversations, err := conversation.List(db)
	if err != nil {
		log.Fatalf("Error listing conversations: %v", err)
	}

	if len(conversations) == 0 {
		fmt.Println("No conversations found.")
	} else {
		fmt.Println("Conversations:")
		for _, conv := range conversations {
			fmt.Printf("  ID: %s, Created: %s, Last Message: %s, Messages: %d\n",
				conv.ID, conv.CreatedAt.Format(time.RFC3339),
				conv.LatestMessageTime.Format(time.RFC3339), conv.MessageCount)
		}
	}
	return nil
}
