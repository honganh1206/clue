package server

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/honganh1206/clue/server/data/conversation"
	"github.com/honganh1206/clue/server/data/plan"
	"github.com/honganh1206/clue/server/db"
	_ "github.com/mattn/go-sqlite3"
)

type server struct {
	addr   net.Addr
	db     *sql.DB
	models *Models
}

func Serve(ln net.Listener) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get home directory:", err)
	}

	// TODO: This should have their own function
	// to be used directly by the CLI agent
	dsn := filepath.Join(homeDir, ".clue", "clue.db")

	db, err := db.OpenDB(dsn, conversation.Schema, plan.Schema)
	if err != nil {
		log.Fatalf("Failed to initialize database: %s", err.Error())
	}
	defer db.Close()

	srv := &server{
		addr:   ln.Addr(),
		db:     db,
		models: NewModels(db),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Register conversation handlers
	mux.HandleFunc("/conversations", srv.conversationHandler)
	mux.HandleFunc("/conversations/", srv.conversationHandler)

	// Register plan handlers
	mux.HandleFunc("/plans", srv.planHandler)
	mux.HandleFunc("/plans/", srv.planHandler)

	server := &http.Server{Handler: mux, Addr: ":11435"}
	return server.Serve(ln)
}
