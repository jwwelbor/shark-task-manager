package viewer

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// editBodySizeLimit is the maximum allowed request body for the edit endpoint.
// Mirrors viewerFileSizeLimit in ViewerService (2 MiB).
const editBodySizeLimit = 2 * 1024 * 1024

// EditServicer is the interface EditHandler depends on.
// Defined here so tests can inject a mock without importing the full service.
// The concrete *services.EditService satisfies this interface.
type EditServicer interface {
	WriteFile(ctx context.Context, path string, content string) (*services.WriteFileResult, error)
}

// Compile-time check: *services.EditService must satisfy EditServicer.
var _ EditServicer = (*services.EditService)(nil)

// EditHandler handles file edit requests for the web viewer.
// It is a thin wrapper: parse JSON body → call EditService.WriteFile → format response.
type EditHandler struct {
	svc EditServicer
}

// NewEditHandler constructs an EditHandler with the given service.
func NewEditHandler(svc EditServicer) *EditHandler {
	if svc == nil {
		panic("EditHandler: svc is required")
	}
	return &EditHandler{svc: svc}
}

// putFileRequest is the expected JSON body for PUT /api/v1/edit/file.
type putFileRequest struct {
	Path    string  `json:"path"`
	Content *string `json:"content"`
}

// PutFile handles PUT /api/v1/edit/file.
// Parses JSON body {path, content}, calls svc.WriteFile, returns result or error.
func (h *EditHandler) PutFile(w http.ResponseWriter, r *http.Request) {
	// Enforce body size limit before attempting to decode.
	r.Body = http.MaxBytesReader(w, r.Body, editBodySizeLimit)

	var req putFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// MaxBytesReader surfaces an oversized body as a specific error type
		// via the message "http: request body too large".
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			respondError(w, http.StatusRequestEntityTooLarge, "request body exceeds 2 MiB limit")
			return
		}
		respondError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	// Validate required fields.
	if req.Path == "" {
		respondError(w, http.StatusBadRequest, "missing required field: path")
		return
	}
	if req.Content == nil {
		respondError(w, http.StatusBadRequest, "missing required field: content")
		return
	}

	result, err := h.svc.WriteFile(r.Context(), req.Path, *req.Content)
	if err != nil {
		var secErr *services.SecurityError
		if errors.As(err, &secErr) {
			respondError(w, http.StatusForbidden, errMsgPathEscapesRoot)
			return
		}
		slog.Error("edit file failed", "path", req.Path, "endpoint", "put_file", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	respondJSON(w, http.StatusOK, result)
}
