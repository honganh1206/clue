# Server & Client Refactoring Suggestions

## Current Issues

1. **Repetitive code**: Error handling, JSON encoding/decoding duplicated across handlers
2. **Mixed concerns**: HTTP routing, validation, and business logic in one place
3. **Inconsistent error handling**: String matching instead of typed errors
4. **No middleware**: CORS, logging, recovery missing
5. **Hard to test**: Handlers tightly coupled to http.ResponseWriter

## Proposed Improvements

### 1. Extract Response Helpers

```go
// server/response.go
package server

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
```

### 2. Use Router with Better Path Handling

```go
// Use gorilla/mux or chi router
import "github.com/go-chi/chi/v5"

func (s *server) routes() http.Handler {
	r := chi.NewRouter()

	r.Route("/conversations", func(r chi.Router) {
		r.Get("/", s.listConversations)
		r.Post("/", s.createConversation)
		r.Get("/{id}", s.getConversation)
		r.Put("/{id}", s.saveConversation)
	})

	r.Route("/plans", func(r chi.Router) {
		r.Get("/", s.listPlans)
		r.Post("/", s.createPlan)
		r.Get("/{name}", s.getPlan)
		r.Put("/{name}", s.savePlan)
		r.Delete("/{name}", s.deletePlan)
		r.Delete("/", s.deletePlans)
	})

	return r
}
```

### 3. Add Middleware

```go
// server/middleware.go
package server

func (s *server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Logging
		log.Printf("%s %s", r.Method, r.URL.Path)

		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Recovery
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
```

### 4. Standardize Error Types

```go
// server/errors.go
package server

import (
	"errors"
	"net/http"
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
	return e.Message
}

func handleError(w http.ResponseWriter, err error) {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		writeError(w, httpErr.Code, httpErr.Message)
		return
	}

	writeError(w, http.StatusInternalServerError, "Internal server error")
}
```

### 5. Refactor Handlers Using Helpers

```go
// Simplified handler example
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

	writeJSON(w, http.StatusCreated, map[string]string{"id": conv.ID})
}
```

### 6. Client Request Helper

```go
// api/request.go
package api

func (c *Client) doRequest(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Usage:
func (c *Client) CreateConversation() (*conversation.Conversation, error) {
	var result map[string]string
	if err := c.doRequest(http.MethodPost, "/conversations", nil, &result); err != nil {
		return nil, err
	}

	return &conversation.Conversation{
		ID:       result["id"],
		Messages: make([]*message.Message, 0),
	}, nil
}
```

### 7. Validation Layer

```go
// server/validation.go
package server

type Validator interface {
	Validate() error
}

func validateRequest(w http.ResponseWriter, r *http.Request, v Validator) bool {
	if err := decodeJSON(r, v); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request format")
		return false
	}

	if err := v.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return false
	}

	return true
}
```

## Migration Strategy

1. **Phase 1**: Add helper functions (response.go, errors.go)
2. **Phase 2**: Refactor existing handlers to use helpers
3. **Phase 3**: Introduce router library (optional)
4. **Phase 4**: Add middleware layer
5. **Phase 5**: Refactor client with request helper

## Benefits

- **DRY**: Eliminate code duplication
- **Testability**: Easier to mock and test individual components
- **Maintainability**: Changes to error handling/logging in one place
- **Consistency**: Uniform response format and error handling
- **Scalability**: Easy to add new endpoints following same pattern
