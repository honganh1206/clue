package testutil

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/honganh1206/clue/server/db"
	_ "github.com/mattn/go-sqlite3"
)

func CreateTestDB(t *testing.T, schema string) *sql.DB {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "clue_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	testDBPath := filepath.Join(tempDir, "test.db")

	db, err := db.OpenDB(testDBPath, schema)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
