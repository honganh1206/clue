package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func OpenDB(dsn string, schemas ...string) (*sql.DB, error) {
	dbDir := filepath.Dir(dsn)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		err = os.MkdirAll(dbDir, 0o755)
		if err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	for _, schema := range schemas {
		if err = initializeSchema(db, schema); err != nil {
			db.Close()
			return nil, err
		}
	}
	// Yes we are hardcoding the configs here
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)

	duration, err := time.ParseDuration("15m")
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func initializeSchema(db *sql.DB, schema string) error {
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema initialization SQL: %w", err)
	}
	return nil
}
