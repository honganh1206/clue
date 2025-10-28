package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	server := &http.Server{Handler: mux, Addr: ":11435"}
	return server.Serve(ln)
}

func (s *server) conversationHandler(w http.ResponseWriter, r *http.Request) {
	convID, hasID := parseConvID(r.URL.Path)

	switch r.Method {
	case http.MethodPost:
		s.createConversation(w, r)
	case http.MethodGet:
		if hasID {
			s.getConversation(w, r, convID)
		} else {
			s.listConversations(w, r)
		}
	case http.MethodPut:
		s.saveConversation(w, r, convID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func parseConvID(path string) (string, bool) {
	path = strings.TrimSuffix(path, "/")

	if path == "/conversations" {
		return "", false
	}

	if !strings.HasPrefix(path, "/conversations/") {
		return "", false
	}

	id := strings.TrimPrefix(path, "/conversations/")

	// Ensure no path segment?
	if strings.Contains(id, "/") {
		return "", false
	}

	return id, true
}

func (s *server) createConversation(w http.ResponseWriter, r *http.Request) {
	conv, err := conversation.New()
	if err != nil {
		http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
		return
	}

	if err := s.models.Conversations.SaveTo(conv); err != nil {
		http.Error(w, "Failed to save conversation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": conv.ID})
}

func (s *server) listConversations(w http.ResponseWriter, r *http.Request) {
	conversations, err := s.models.Conversations.List()
	if err != nil {
		http.Error(w, "Failed to list conversations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversations)
}

func (s *server) getConversation(w http.ResponseWriter, r *http.Request, id string) {
	conv, err := s.models.Conversations.Load(id)
	if err != nil {
		if err == conversation.ErrConversationNotFound {
			http.Error(w, "Conversation not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to load conversation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

func (s *server) saveConversation(w http.ResponseWriter, r *http.Request, conversationID string) {
	var conv conversation.Conversation
	if err := json.NewDecoder(r.Body).Decode(&conv); err != nil {
		http.Error(w, "Invalid conversation format", http.StatusBadRequest)
		return
	}

	// Ensure the conversation ID matches the URL parameter
	if conv.ID != conversationID {
		http.Error(w, "Conversation ID mismatch", http.StatusBadRequest)
		return
	}

	// Save the entire conversation
	if err := s.models.Conversations.SaveTo(&conv); err != nil {
		http.Error(w, "Failed to save conversation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "conversation saved"})
}
