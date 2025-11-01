package server

import (
	"database/sql"

	"github.com/honganh1206/clue/server/data/conversation"
	"github.com/honganh1206/clue/server/data/plan"
)

type Models struct {
	Conversations *conversation.ConversationModel
	Plans         *plan.PlanModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		Conversations: &conversation.ConversationModel{DB: db},
		Plans:         &plan.PlanModel{DB: db},
	}
}
