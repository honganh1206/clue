package server

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/honganh1206/clue/server/data"
	"github.com/honganh1206/clue/server/db"

	_ "github.com/mattn/go-sqlite3"
)

type server struct {
	addr   net.Addr
	db     *sql.DB
	models *data.Models
}

func Serve(ln net.Listener) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get home directory:", err)
	}

	// TODO: This should have their own function
	// to be used directly by the CLI agent
	dsn := filepath.Join(homeDir, ".clue", "clue.db")

	db, err := db.OpenDB(dsn, data.ConversationSchema, data.PlanSchema)
	if err != nil {
		log.Fatalf("Failed to initialize database: %s", err.Error())
	}
	defer db.Close()

	srv := &server{
		addr:   ln.Addr(),
		db:     db,
		models: data.NewModels(db),
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

	if strings.Contains(id, "/") {
		return "", false
	}

	return id, true
}

func (s *server) createConversation(w http.ResponseWriter, r *http.Request) {
	conv, err := data.NewConversation()
	if err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create conversation",
			Err:     err,
		})
		return
	}

	if err := s.models.Conversations.Create(conv); err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create conversation",
			Err:     err,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": conv.ID})
}

func (s *server) listConversations(w http.ResponseWriter, r *http.Request) {
	conversations, err := s.models.Conversations.List()
	if err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list conversations",
			Err:     err,
		})
		return
	}

	writeJSON(w, http.StatusOK, conversations)
}

func (s *server) getConversation(w http.ResponseWriter, r *http.Request, id string) {
	conv, err := s.models.Conversations.Get(id)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, conv)
}

func (s *server) saveConversation(w http.ResponseWriter, r *http.Request, conversationID string) {
	var conv data.Conversation
	if err := decodeJSON(r, &conv); err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Invalid conversation format",
			Err:     err,
		})
		return
	}

	if conv.ID != conversationID {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Conversation ID mismatch",
			Err:     nil,
		})
		return
	}

	if err := s.models.Conversations.Save(&conv); err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to save conversation",
			Err:     err,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "conversation saved"})
}

func (s *server) planHandler(w http.ResponseWriter, r *http.Request) {
	planID, hasID := parsePlanID(r.URL.Path)
	switch r.Method {
	case http.MethodPost:
		s.createPlan(w, r)
	case http.MethodGet:
		s.getPlan(w, r, planID)
	case http.MethodPut:
		s.savePlan(w, r, planID)
	case http.MethodDelete:
		if hasID {
			s.deletePlan(w, r, planID)
		} else {
			s.deletePlans(w, r)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func parsePlanID(path string) (string, bool) {
	path = strings.TrimSuffix(path, "/")

	if path == "/plans" {
		return "", false
	}

	if !strings.HasPrefix(path, "/plans/") {
		return "", false
	}

	id := strings.TrimPrefix(path, "/plans/")

	if strings.Contains(id, "/") {
		return "", false
	}

	return id, true
}

func (s *server) createPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConversationID string `json:"conversation_id"`
	}

	if err := decodeJSON(r, &req); err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Err:     err,
		})
		return
	}

	if req.ConversationID == "" {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Conversation ID is required",
			Err:     nil,
		})
		return
	}

	plan, err := data.NewPlan(req.ConversationID)
	if err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
			Err:     err,
		})
		return
	}

	err = s.models.Plans.Create(plan)
	if err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
			Err:     err,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": plan.ID})
}

func (s *server) getPlan(w http.ResponseWriter, r *http.Request, id string) {
	p, err := s.models.Plans.Get(id)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (s *server) savePlan(w http.ResponseWriter, r *http.Request, planID string) {
	var p data.Plan
	if err := decodeJSON(r, &p); err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Invalid plan format",
			Err:     err,
		})
		return
	}

	if p.ID != planID {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Plan ID mismatch",
			Err:     nil,
		})
		return
	}

	if err := s.models.Plans.Save(&p); err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
			Err:     err,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "plan saved"})
}

func (s *server) deletePlan(w http.ResponseWriter, r *http.Request, id string) {
	results := s.models.Plans.Remove([]string{id})

	if err, exists := results[id]; exists && err != nil {
		handleError(w, err)
		return
	}

	if err, exists := results["_"]; exists && err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to delete plan",
			Err:     err,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "plan deleted"})
}

func (s *server) deletePlans(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}

	if err := decodeJSON(r, &req); err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Err:     err,
		})
		return
	}

	if len(req.IDs) == 0 {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "No plan IDs provided",
			Err:     nil,
		})
		return
	}

	results := s.models.Plans.Remove(req.IDs)

	writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
	})
}
