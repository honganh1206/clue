package data

import (
	"database/sql"
)

type Models struct {
	Conversations *ConversationModel
	Plans         *PlanModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		Conversations: &ConversationModel{DB: db},
		Plans:         &PlanModel{DB: db},
	}
}
