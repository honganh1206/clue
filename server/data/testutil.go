package data

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/honganh1206/clue/server/db"
	_ "github.com/mattn/go-sqlite3"
)

func createTestDB(t *testing.T) *sql.DB {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "clue_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	testDBPath := filepath.Join(tempDir, "test.db")

	schemas := make([]string, 2)
	schemas = append(schemas, ConversationSchema)
	schemas = append(schemas, PlanSchema)

	db, err := db.OpenDB(testDBPath, schemas...)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
