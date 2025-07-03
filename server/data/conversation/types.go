package conversation

import "time"

type ConversationMetadata struct {
	ID                string
	LatestMessageTime time.Time
	MessageCount      int
	CreatedAt         time.Time
}
