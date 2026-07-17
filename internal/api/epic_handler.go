package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// EpicServicer is the interface that EpicHandler depends on.
type EpicServicer interface {
	GetEpic(ctx context.Context, key string) (*models.Epic, error)
	ListEpics(ctx context.Context, filters services.EpicFilters) ([]*models.Epic, error)
	CreateEpic(ctx context.Context, input services.CreateEpicInput) (*models.Epic, error)
	UpdateEpic(ctx context.Context, key string, updates services.EpicUpdates) (*models.Epic, error)
	DeleteEpic(ctx context.Context, key string) error
	TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

// EpicHandler handles HTTP requests for epic CRUD and lifecycle operations.
type EpicHandler struct {
	svc EpicServicer
}

// NewEpicHandler constructs an EpicHandler with the given service.
func NewEpicHandler(svc EpicServicer) *EpicHandler {
	if svc == nil {
		panic("EpicHandler: svc is required")
	}
	return &EpicHandler{svc: svc}
}

// RegisterRoutes registers epic routes on the given mux.
// Route patterns:
//
//	GET    /api/v1/epics          → ListEpics
//	POST   /api/v1/epics          → CreateEpic
//	GET    /api/v1/epics/{key}    → GetEpic
//	PATCH  /api/v1/epics/{key}    → UpdateEpic
//	DELETE /api/v1/epics/{key}    → DeleteEpic
//	GET    /api/v1/epics/{key}/next-status → GetNextStatus
//	POST   /api/v1/epics/{key}/transition  → TransitionStatus
func (h *EpicHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/epics", h.ListEpics)
	mux.HandleFunc("POST /api/v1/epics", h.CreateEpic)
	mux.HandleFunc("GET /api/v1/epics/{key}", h.GetEpic)
	mux.HandleFunc("PATCH /api/v1/epics/{key}", h.UpdateEpic)
	mux.HandleFunc("DELETE /api/v1/epics/{key}", h.DeleteEpic)
	mux.HandleFunc("GET /api/v1/epics/{key}/next-status", h.GetNextStatus)
	mux.HandleFunc("POST /api/v1/epics/{key}/transition", h.TransitionStatus)
}

// GetEpic returns a single epic by key.
// GET /api/v1/epics/{key}
func (h *EpicHandler) GetEpic(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	epic, err := h.svc.GetEpic(r.Context(), key)
	if err != nil {
		handleServiceError(w, err, "epic")
		return
	}

	respondJSON(w, http.StatusOK, epic)
}

// ListEpics returns a filtered list of epics.
// GET /api/v1/epics?status=draft
func (h *EpicHandler) ListEpics(w http.ResponseWriter, r *http.Request) {
	filters := services.EpicFilters{
		Status: r.URL.Query().Get("status"),
	}

	epics, err := h.svc.ListEpics(r.Context(), filters)
	if err != nil {
		handleServiceError(w, err, "epic")
		return
	}

	respondJSON(w, http.StatusOK, epics)
}

// CreateEpicRequest is the JSON body for creating an epic.
type CreateEpicRequest struct {
	Title         string  `json:"title"`
	Description   *string `json:"description,omitempty"`
	Status        string  `json:"status,omitempty"`
	Priority      string  `json:"priority,omitempty"`
	BusinessValue *string `json:"business_value,omitempty"`
}

// CreateEpic creates a new epic.
// POST /api/v1/epics
func (h *EpicHandler) CreateEpic(w http.ResponseWriter, r *http.Request) {
	var req CreateEpicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}

	input := services.CreateEpicInput{
		Title:         req.Title,
		Description:   req.Description,
		Status:        req.Status,
		Priority:      req.Priority,
		BusinessValue: req.BusinessValue,
	}

	epic, err := h.svc.CreateEpic(r.Context(), input)
	if err != nil {
		handleServiceError(w, err, "epic")
		return
	}

	w.Header().Set("Location", "/api/v1/epics/"+epic.Key)
	respondJSON(w, http.StatusCreated, epic)
}

// UpdateEpicRequest is the JSON body for partial epic updates.
type UpdateEpicRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

// UpdateEpic applies partial updates to an existing epic.
// PATCH /api/v1/epics/{key}
func (h *EpicHandler) UpdateEpic(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	var req UpdateEpicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updates := services.EpicUpdates{
		Title:       req.Title,
		Description: req.Description,
	}

	epic, err := h.svc.UpdateEpic(r.Context(), key, updates)
	if err != nil {
		handleServiceError(w, err, "epic")
		return
	}

	respondJSON(w, http.StatusOK, epic)
}

// DeleteEpic deletes an epic by key.
// DELETE /api/v1/epics/{key}
func (h *EpicHandler) DeleteEpic(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	if err := h.svc.DeleteEpic(r.Context(), key); err != nil {
		handleServiceError(w, err, "epic")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetNextStatus returns the available next statuses for an epic.
// GET /api/v1/epics/{key}/next-status
func (h *EpicHandler) GetNextStatus(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	info, err := h.svc.GetNextStatus(r.Context(), key)
	if err != nil {
		handleServiceError(w, err, "epic")
		return
	}

	respondJSON(w, http.StatusOK, info)
}

// TransitionStatus advances an epic to a specified status.
// POST /api/v1/epics/{key}/transition
func (h *EpicHandler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	var req TransitionStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TargetStatus == "" {
		respondError(w, http.StatusBadRequest, "target_status is required")
		return
	}

	opts := services.TransitionOptions{
		Force:        req.Force,
		Reason:       req.Reason,
		Agent:        req.Agent,
		SessionID:    req.SessionID,
		FromStatus:   req.FromStatus,
		Outcome:      req.Outcome,
		ForceRepeat:  req.ForceRepeat,
		GuardAdvance: req.GuardAdvance,
	}

	result, err := h.svc.TransitionStatus(r.Context(), key, req.TargetStatus, opts)
	if err != nil {
		handleServiceError(w, err, "epic")
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Compile-time check: *services.EpicService must satisfy EpicServicer.
var _ EpicServicer = (*services.EpicService)(nil)
