package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/honganh1206/tinker/server/data"
)

var (
	ErrNotFound      = errors.New("resource not found")
	ErrBadRequest    = errors.New("bad request")
	ErrInternalError = errors.New("internal error")
)

type HTTPError struct {
	Code    int
	Message string
	Err     error
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func handleError(w http.ResponseWriter, err error) {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		writeError(w, httpErr.Code, httpErr.Message)
		return
	}

	if errors.Is(err, data.ErrConversationNotFound) || errors.Is(err, data.ErrPlanNotFound) {
		writeError(w, http.StatusNotFound, "Resource not found")
		return
	}

	if strings.Contains(err.Error(), "not found") {
		writeError(w, http.StatusNotFound, "Resource not found")
		return
	}

	writeError(w, http.StatusInternalServerError, "Internal server error")
}
