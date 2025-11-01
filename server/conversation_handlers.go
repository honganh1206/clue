package server

import (
	"net/http"
	"strings"

	"github.com/honganh1206/clue/server/data/conversation"
)

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
	conv, err := conversation.New()
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
	var conv conversation.Conversation
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
