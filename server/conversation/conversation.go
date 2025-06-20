package conversation

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/honganh1206/clue/server/db"
	"github.com/honganh1206/clue/utils"
)

//go:embed schema.sql
var schemaSQL string

var (
	ErrConversationNotFound = errors.New("history: conversation not found")
)

type Conversation struct {
	ID        string
	Messages  []*MessageParam
	CreatedAt time.Time
}

type ConversationMetadata struct {
	ID                string
	LatestMessageTime time.Time
	MessageCount      int
	CreatedAt         time.Time
}

func InitDB(dsn string) (*sql.DB, error) {
	dbConfig := db.Config{
		Dsn:          dsn,
		MaxOpenConns: 25,
		MaxIdleConns: 25,
		MaxIdleTime:  "15m",
	}

	conversationDb, err := db.OpenDB(dbConfig, schemaSQL)
	if err != nil {
		return nil, err
	}

	return conversationDb, nil
}

func New() (*Conversation, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	return &Conversation{
		ID:        id.String(),
		Messages:  make([]*MessageParam, 0),
		CreatedAt: time.Now(),
	}, nil
}

func (c *Conversation) Append(msg MessageParam) {
	now := time.Now()
	sequence := len(c.Messages)

	msg.CreatedAt = now
	msg.Sequence = sequence

	c.Messages = append(c.Messages, &msg)
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
		jsonBytes, jsonErr := json.Marshal(msg)
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

func List(db *sql.DB) ([]ConversationMetadata, error) {
	query := `
		SELECT
			c.id,
			c.created_at,
			COUNT(m.id) as message_count,
			COALESCE(MAX(m.created_at), c.created_at) as latest_message_at
		FROM
			conversations c
		LEFT JOIN
			messages m ON c.id = m.conversation_id
		GROUP BY
			c.id
		ORDER BY
			latest_message_at DESC;
	`

	rows, err := db.Query(query)
	if err != nil {
		// Check for missing tables
		var tableCheck string
		errTable := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='conversations'").Scan(&tableCheck)
		if errTable == sql.ErrNoRows {
			return []ConversationMetadata{}, nil // No 'conversations' table, so no conversations
		}
		return nil, fmt.Errorf("failed to query conversations: %w", err)
	}

	defer rows.Close()

	var metadataList []ConversationMetadata
	for rows.Next() {
		var meta ConversationMetadata
		var createdAt string
		var latestTimestamp string

		if err := rows.Scan(&meta.ID, &createdAt, &meta.MessageCount, &latestTimestamp); err != nil {
			return nil, fmt.Errorf("failed to scan conversation metadata: %w", err)
		}
		meta.CreatedAt, err = utils.ParseTimeWithFallback(createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse conversation created_at: %w", err)
		}

		meta.LatestMessageTime, err = utils.ParseTimeWithFallback(latestTimestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse latest_message_timestamp: %w", err)
		}
		metadataList = append(metadataList, meta)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return metadataList, nil

}

func LatestID(db *sql.DB) (string, error) {
	query := `
		SELECT id FROM conversations ORDER BY created_at DESC LIMIT 1
	`

	var id string
	err := db.QueryRow(query).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrConversationNotFound // Return custom error
		}
		return "", fmt.Errorf("failed to query for latest conversation ID: %w", err)
	}

	return id, nil
}

func Load(id string, db *sql.DB) (*Conversation, error) {
	query := `
		SELECT created_at FROM conversations WHERE id = ?
	`

	conv := &Conversation{ID: id, Messages: make([]*MessageParam, 0)}

	err := db.QueryRow(query, id).Scan(&conv.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrConversationNotFound
		}
		return nil, fmt.Errorf("failed to query conversation metadata for ID '%s': %w", id, err)
	}

	query = `
		SELECT
			sequence_number,
			payload,
			created_at
		FROM
			messages WHERE conversation_id = ?
		ORDER BY
			sequence_number ASC
	`

	rows, err := db.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages for conversation ID '%s': %w", id, err)
	}
	defer rows.Close()

	var msgs []*MessageParam

	for rows.Next() {
		var seq int
		var payload []byte
		var createdAt time.Time

		if err := rows.Scan(&seq, &payload, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan message for conversation ID '%s': %w", id, err)
		}

		// Temp struct to access the content block type
		var tempMsg struct {
			Content []struct {
				Type string `json:"type"`
			} `json:"content"`
		}

		if err := json.Unmarshal(payload, &tempMsg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal temp message payload for conversation ID '%s': %w", id, err)
		}

		// Complete message struct with proper content blocks
		var fullMsg struct {
			Role    string            `json:"role"`
			Content []json.RawMessage `json:"content"`
		}

		if err := json.Unmarshal(payload, &fullMsg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal full message payload for conversation ID '%s': %w", id, err)
		}

		msg := &MessageParam{
			Role:      fullMsg.Role,
			Content:   make([]ContentBlock, 0, len(fullMsg.Content)),
			CreatedAt: createdAt,
			Sequence:  seq,
		}

		// Unmarshal each content block based on its type
		for i, rawContent := range fullMsg.Content {
			var contentBlock ContentBlock

			switch tempMsg.Content[i].Type {
			case TextType:
				var textBlock TextContentBlock
				if err := json.Unmarshal(rawContent, &textBlock); err != nil {
					return nil, fmt.Errorf("failed to unmarshal text content block for conversation ID '%s': %w", id, err)
				}
				contentBlock = textBlock

			case ToolUseType:
				var toolUseBlock ToolUseContentBlock
				if err := json.Unmarshal(rawContent, &toolUseBlock); err != nil {
					return nil, fmt.Errorf("failed to unmarshal tool use content block for conversation ID '%s': %w", id, err)
				}
				contentBlock = toolUseBlock

			case ToolResultType:
				var toolResultBlock ToolResultContentBlock
				if err := json.Unmarshal(rawContent, &toolResultBlock); err != nil {
					return nil, fmt.Errorf("failed to unmarshal tool result content block for conversation ID '%s': %w", id, err)
				}
				contentBlock = toolResultBlock

			default:
				return nil, fmt.Errorf("unknown content block type '%s' for conversation ID '%s'", tempMsg.Content[i].Type, id)
			}

			msg.Content = append(msg.Content, contentBlock)
		}
		msgs = append(msgs, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during message rows iteration for conversation ID '%s': %w", id, err)
	}

	for _, msg := range msgs {
		conv.Messages = append(conv.Messages, msg)
	}

	return conv, nil

}
