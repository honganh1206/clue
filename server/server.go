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

	if err := s.models.Conversations.SaveTo(conv); err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to save conversation",
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
	conv, err := s.models.Conversations.Load(id)
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

	if err := s.models.Conversations.SaveTo(&conv); err != nil {
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
	planName, hasName := parsePlanName(r.URL.Path)

	switch r.Method {
	case http.MethodPost:
		s.createPlan(w, r)
	case http.MethodGet:
		if hasName {
			s.getPlan(w, r, planName)
		} else {
			s.listPlans(w, r)
		}
	case http.MethodPut:
		s.savePlan(w, r, planName)
	case http.MethodDelete:
		if hasName {
			s.deletePlan(w, r, planName)
		} else {
			s.deletePlans(w, r)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func parsePlanName(path string) (string, bool) {
	path = strings.TrimSuffix(path, "/")

	if path == "/plans" {
		return "", false
	}

	if !strings.HasPrefix(path, "/plans/") {
		return "", false
	}

	name := strings.TrimPrefix(path, "/plans/")

	if strings.Contains(name, "/") {
		return "", false
	}

	return name, true
}

func (s *server) createPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := decodeJSON(r, &req); err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Err:     err,
		})
		return
	}

	if req.Name == "" {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Plan name is required",
			Err:     nil,
		})
		return
	}

	plan := &data.Plan{
		ID: req.Name,
	}

	err := s.models.Plans.Create(plan)
	if err != nil {
		handleError(w, &HTTPError{
			Code: http.StatusInternalServerError,
			// Failed 500 here
			Message: err.Error(),
			Err:     err,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": plan.ID})
}

func (s *server) listPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := s.models.Plans.List()
	if err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list plans",
			Err:     err,
		})
		return
	}

	writeJSON(w, http.StatusOK, plans)
}

func (s *server) getPlan(w http.ResponseWriter, r *http.Request, name string) {
	p, err := s.models.Plans.Get(name)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (s *server) savePlan(w http.ResponseWriter, r *http.Request, planName string) {
	var p data.Plan
	if err := decodeJSON(r, &p); err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Invalid plan format",
			Err:     err,
		})
		return
	}

	if p.ID != planName {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Plan name mismatch",
			Err:     nil,
		})
		return
	}

	if err := s.models.Plans.Save(&p); err != nil {
		// The code definitely broke down here
		// how do we get the detailed error?
		// handleError(w, err)
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
			Err:     nil,
		})

		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "plan saved"})
}

func (s *server) deletePlan(w http.ResponseWriter, r *http.Request, name string) {
	results := s.models.Plans.Remove([]string{name})

	if err, exists := results[name]; exists && err != nil {
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
		Names []string `json:"names"`
	}

	if err := decodeJSON(r, &req); err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Err:     err,
		})
		return
	}

	if len(req.Names) == 0 {
		handleError(w, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "No plan names provided",
			Err:     nil,
		})
		return
	}

	results := s.models.Plans.Remove(req.Names)

	writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
	})
}

