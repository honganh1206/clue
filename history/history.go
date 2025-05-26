package history

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/honganh1206/clue/db"
	"github.com/honganh1206/clue/messages"
)

//go:embed schema.sql
var schemaSQL string

var DefaultDatabasePath = ".dbs/history.db"

type Message struct {
	Payload   messages.MessageParam
	CreatedAt time.Time
}

type Conversation struct {
	ID        string
	Messages  []*Message
	CreatedAt time.Time
}

func InitDB() (*sql.DB, error) {
	// TODO: Make this configurable
	dbConfig := db.Config{
		Dsn:          DefaultDatabasePath,
		MaxOpenConns: 25,
		MaxIdleConns: 25,
		MaxIdleTime:  "15m",
	}

	historyDb, err := db.InitDB(dbConfig, schemaSQL)
	if err != nil {
		return nil, err
	}

	return historyDb, nil
}

func NewConversation() (*Conversation, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	return &Conversation{
		ID:        id.String(),
		Messages:  make([]*Message, 0),
		CreatedAt: time.Now(),
	}, nil
}

func (c *Conversation) Append(payload messages.MessageParam) {
	msg := &Message{
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	c.Messages = append(c.Messages, msg)
}

func SaveTo(db *sql.DB, conversation *Conversation) error {
	// Begin a transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// TODO: Do I need to init a context for timeouts/graceful cancellation/tracing and logging?

	query := `
	INSERT OR IGNORE INTO conversations (id, created_at)
	VALUES(?, ?);
	`

	if _, err = tx.Exec(query, conversation.ID, conversation.CreatedAt); err != nil {
		tx.Rollback()
		return err
	}

	query = `
	DELETE FROM messages WHERE conversation_id = ?;
	`

	if _, err = tx.Exec(query, conversation.ID); err != nil {
		tx.Rollback()
		return err
	}

	query = `
	INSERT INTO messages (conversation_id, sequence_number, payload, created_at)
	VALUES (?, ?, ?, ?);
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for i, msg := range conversation.Messages {
		jsonBytes, jsonErr := json.Marshal(msg.Payload)
		if jsonErr != nil {
			tx.Rollback()
			return jsonErr
		}
		payloadString := string(jsonBytes)
		_, err = stmt.Exec(conversation.ID, i, payloadString, msg.CreatedAt)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
