package server

import (
	"database/sql"

	"github.com/honganh1206/clue/server/data/conversation"
)

type Models struct {
	Conversations *conversation.ConversationModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		Conversations: &conversation.ConversationModel{DB: db},
	}
}
