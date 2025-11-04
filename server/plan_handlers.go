package server

import (
	"net/http"
	"strings"

	"github.com/honganh1206/clue/server/data/plan"
)

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

	// We got the in-mem plan object, but we return only the ID
	p, err := s.models.Plans.Create(req.Name)
	if err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create plan",
			Err:     err,
		})
		return
	}

	if err := s.models.Plans.Save(p); err != nil {
		handleError(w, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Failed to save plan",
			Err:     err,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": p.ID})
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
	var p plan.Plan
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
		handleError(w, err)
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
