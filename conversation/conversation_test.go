package conversation

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func createTestDB(t *testing.T) *sql.DB {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "conversation_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	testDBPath := filepath.Join(tempDir, "test.db")

	originalPath := DefaultDatabasePath
	DefaultDatabasePath = testDBPath
	t.Cleanup(func() {
		DefaultDatabasePath = originalPath
	})

	db, err := InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestConversation_Append(t *testing.T) {
	conv, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	msg := MessageParam{
		Role: UserRole,
		Content: []ContentBlock{
			NewTextContentBlock("Hello, world!"),
		},
	}

	conv.Append(msg)

	if len(conv.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(conv.Messages))
	}

	appended := conv.Messages[0]
	if appended.Role != UserRole {
		t.Errorf("Expected role %s, got %s", UserRole, appended.Role)
	}
	if appended.Sequence != 0 {
		t.Errorf("Expected sequence 0, got %d", appended.Sequence)
	}
	if appended.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}

	msg2 := MessageParam{
		Role: AssistantRole,
		Content: []ContentBlock{
			NewTextContentBlock("Hello back!"),
		},
	}

	conv.Append(msg2)

	if len(conv.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(conv.Messages))
	}

	appended2 := conv.Messages[1]
	if appended2.Sequence != 1 {
		t.Errorf("Expected sequence 1, got %d", appended2.Sequence)
	}
	if appended2.CreatedAt.Before(appended.CreatedAt) {
		t.Error("Second message CreatedAt should be after first message")
	}
}

func TestConversation_SaveTo(t *testing.T) {
	db := createTestDB(t)

	conv, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	conv.Append(MessageParam{
		Role: UserRole,
		Content: []ContentBlock{
			NewTextContentBlock("First message"),
		},
	})

	conv.Append(MessageParam{
		Role: AssistantRole,
		Content: []ContentBlock{
			NewTextContentBlock("Second message"),
		},
	})

	if err := conv.SaveTo(db); err != nil {
		t.Fatalf("SaveTo() failed: %v", err)
	}

	var savedID string
	var savedCreatedAt time.Time
	err = db.QueryRow("SELECT id, created_at FROM conversations WHERE id = ?", conv.ID).
		Scan(&savedID, &savedCreatedAt)
	if err != nil {
		t.Fatalf("Failed to query saved conversation: %v", err)
	}

	if savedID != conv.ID {
		t.Errorf("Expected ID %s, got %s", conv.ID, savedID)
	}

	rows, err := db.Query("SELECT sequence_number, payload FROM messages WHERE conversation_id = ? ORDER BY sequence_number", conv.ID)
	if err != nil {
		t.Fatalf("Failed to query saved messages: %v", err)
	}
	defer rows.Close()

	messageCount := 0
	for rows.Next() {
		var sequence int
		var payload string
		if err := rows.Scan(&sequence, &payload); err != nil {
			t.Fatalf("Failed to scan message row: %v", err)
		}

		if sequence != messageCount {
			t.Errorf("Expected sequence %d, got %d", messageCount, sequence)
		}

		messageCount++
	}

	if messageCount != len(conv.Messages) {
		t.Errorf("Expected %d saved messages, got %d", len(conv.Messages), messageCount)
	}
}

func TestConversation_SaveTo_DuplicateConversation(t *testing.T) {
	db := createTestDB(t)

	conv, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	conv.Append(MessageParam{
		Role: UserRole,
		Content: []ContentBlock{
			NewTextContentBlock("Test message"),
		},
	})

	// Save conversation first time
	if err := conv.SaveTo(db); err != nil {
		t.Fatalf("First SaveTo() failed: %v", err)
	}

	// Add another message and save again
	conv.Append(MessageParam{
		Role: AssistantRole,
		Content: []ContentBlock{
			NewTextContentBlock("Response message"),
		},
	})

	if err := conv.SaveTo(db); err != nil {
		t.Fatalf("Second SaveTo() failed: %v", err)
	}

	// Verify only one conversation record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM conversations WHERE id = ?", conv.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count conversations: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 conversation record, got %d", count)
	}

	// Verify correct number of messages
	err = db.QueryRow("SELECT COUNT(*) FROM messages WHERE conversation_id = ?", conv.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count messages: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 message records, got %d", count)
	}
}
