// Package api provides HTTP handlers for the Shark Task Manager REST API.
// Handlers are thin wrappers: parse HTTP request → call service → format response.
// All business logic lives in the service layer.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// ErrorResponse is the standard JSON error response body.
type ErrorResponse struct {
	Error   string   `json:"error"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

// respondJSON writes a JSON response with the given status code and data.
func respondJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("Failed to encode JSON response", "error", err)
		}
	}
}

// respondError writes a JSON error response.
func respondError(w http.ResponseWriter, statusCode int, message string, details ...string) {
	resp := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	if len(details) > 0 {
		resp.Details = details
	}
	respondJSON(w, statusCode, resp)
}

// handleServiceError maps common service errors to appropriate HTTP status codes.
// Services in this codebase return string-based errors; "not found" suffix is
// the conventional signal for a missing entity.
func handleServiceError(w http.ResponseWriter, err error, entityLabel string) {
	var denied *services.QuestionFullReadDeniedError
	if errors.As(err, &denied) {
		respondError(w, http.StatusForbidden, denied.Error())
		return
	}
	if strings.Contains(err.Error(), "not found") {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if entityLabel == "question" {
		if status, ok := questionErrorStatus(err); ok {
			respondError(w, status, err.Error())
			return
		}
	}

	// Generic server error — avoid leaking internal details.
	slog.Error("Service error", "entity", entityLabel, "error", err)
	respondError(w, http.StatusInternalServerError, "internal server error")
}

// questionErrorStatus classifies QuestionService's well-known business-rule
// rejection messages (state-machine preconditions, malformed resolution
// input) into their HTTP status instead of flattening every non-"not found"
// Question error to a generic 500 that hides the actionable message the
// service already built. QuestionService does not use typed errors for
// these cases; this substring classification mirrors the existing
// "not found" check above and should be replaced with typed errors if this
// list grows much further.
func questionErrorStatus(err error) (int, bool) {
	msg := err.Error()
	for _, phrase := range []string{
		"must be draft or open",
		"is already configured",
		"must be open or answering",
		"responder is not current",
		"active claim does not match responder session",
		"is not answerable",
		"must be ready for resolution",
		"is already terminal",
		"is not eligible for terminal operation",
		"resolution owner does not match configured owner",
	} {
		if strings.Contains(msg, phrase) {
			return http.StatusConflict, true
		}
	}
	for _, phrase := range []string{
		"is required",
		"are required",
		"must be trimmed",
		"must contain",
		"must be valid UTF-8",
		"does not exist",
		"escapes project root",
		"forbidden credential",
		"cannot be empty",
	} {
		if strings.Contains(msg, phrase) {
			return http.StatusBadRequest, true
		}
	}
	return 0, false
}

// pathParam extracts a named path segment from an HTTP request.
// It uses Go 1.22+ net/http pattern matching ({name} syntax).
func pathParam(r *http.Request, name string) string {
	return r.PathValue(name)
}
