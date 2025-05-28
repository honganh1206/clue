package conversation

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/honganh1206/clue/db"
)

//go:embed schema.sql
var schemaSQL string

var DefaultDatabasePath = ".dbs/conversation.db"

type Conversation struct {
	ID        string
	Messages  []MessageParam
	CreatedAt time.Time
}

func InitDB() (*sql.DB, error) {
	// TODO: Make this configurable?
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

func New() (*Conversation, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	return &Conversation{
		ID:        id.String(),
		Messages:  make([]MessageParam, 0),
		CreatedAt: time.Now(),
	}, nil
}

func (c *Conversation) Append(msg MessageParam) {
	now := time.Now()
	sequence := len(c.Messages)

	msg.CreatedAt = now
	msg.Sequence = sequence

	c.Messages = append(c.Messages, msg)
}

func (c *Conversation) SaveTo(db *sql.DB) error {
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

	if _, err = tx.Exec(query, c.ID, c.CreatedAt); err != nil {
		tx.Rollback()
		return err
	}

	// FIXME: Currently delete and re-insert all messages, extremely inefficient
	// There should be a lastSavedIndex to insert the latest message. Should it be a column?
	query = `
	DELETE FROM messages WHERE conversation_id = ?;
	`

	if _, err = tx.Exec(query, c.ID); err != nil {
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

	for i, msg := range c.Messages {
		jsonBytes, jsonErr := json.Marshal(msg.Content)
		if jsonErr != nil {
			tx.Rollback()
			return jsonErr
		}
		payloadString := string(jsonBytes)
		_, err = stmt.Exec(c.ID, i, payloadString, msg.CreatedAt)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
