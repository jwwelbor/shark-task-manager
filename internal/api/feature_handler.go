package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// FeatureServicer is the interface that FeatureHandler depends on.
type FeatureServicer interface {
	GetFeature(ctx context.Context, key string) (*models.Feature, error)
	ListFeatures(ctx context.Context, filters services.FeatureFilters) ([]*models.Feature, error)
	CreateFeature(ctx context.Context, input services.CreateFeatureInput) (*models.Feature, error)
	UpdateFeature(ctx context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error)
	DeleteFeature(ctx context.Context, key string) error
	TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

// FeatureHandler handles HTTP requests for feature CRUD and lifecycle operations.
type FeatureHandler struct {
	svc FeatureServicer
}

// NewFeatureHandler constructs a FeatureHandler with the given service.
func NewFeatureHandler(svc FeatureServicer) *FeatureHandler {
	if svc == nil {
		panic("FeatureHandler: svc is required")
	}
	return &FeatureHandler{svc: svc}
}

// RegisterRoutes registers feature routes on the given mux.
// Route patterns:
//
//	GET    /api/v1/features          → ListFeatures
//	POST   /api/v1/features          → CreateFeature
//	GET    /api/v1/features/{key}    → GetFeature
//	PATCH  /api/v1/features/{key}    → UpdateFeature
//	DELETE /api/v1/features/{key}    → DeleteFeature
//	GET    /api/v1/features/{key}/next-status → GetNextStatus
//	POST   /api/v1/features/{key}/transition  → TransitionStatus
func (h *FeatureHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/features", h.ListFeatures)
	mux.HandleFunc("POST /api/v1/features", h.CreateFeature)
	mux.HandleFunc("GET /api/v1/features/{key}", h.GetFeature)
	mux.HandleFunc("PATCH /api/v1/features/{key}", h.UpdateFeature)
	mux.HandleFunc("DELETE /api/v1/features/{key}", h.DeleteFeature)
	mux.HandleFunc("GET /api/v1/features/{key}/next-status", h.GetNextStatus)
	mux.HandleFunc("POST /api/v1/features/{key}/transition", h.TransitionStatus)
}

// GetFeature returns a single feature by key.
// GET /api/v1/features/{key}
func (h *FeatureHandler) GetFeature(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	feature, err := h.svc.GetFeature(r.Context(), key)
	if err != nil {
		handleServiceError(w, err, "feature")
		return
	}

	respondJSON(w, http.StatusOK, feature)
}

// ListFeatures returns a filtered list of features.
// GET /api/v1/features?epic=E07&status=draft
func (h *FeatureHandler) ListFeatures(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filters := services.FeatureFilters{
		EpicKey: q.Get("epic"),
		Status:  q.Get("status"),
	}

	features, err := h.svc.ListFeatures(r.Context(), filters)
	if err != nil {
		handleServiceError(w, err, "feature")
		return
	}

	respondJSON(w, http.StatusOK, features)
}

// CreateFeatureRequest is the JSON body for creating a feature.
type CreateFeatureRequest struct {
	EpicKey        string  `json:"epic_key"`
	Title          string  `json:"title"`
	Description    *string `json:"description,omitempty"`
	Status         string  `json:"status,omitempty"`
	ExecutionOrder *int    `json:"execution_order,omitempty"`
}

// CreateFeature creates a new feature.
// POST /api/v1/features
func (h *FeatureHandler) CreateFeature(w http.ResponseWriter, r *http.Request) {
	var req CreateFeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" || req.EpicKey == "" {
		respondError(w, http.StatusBadRequest, "epic_key and title are required")
		return
	}

	input := services.CreateFeatureInput{
		EpicKey:        req.EpicKey,
		Title:          req.Title,
		Description:    req.Description,
		Status:         req.Status,
		ExecutionOrder: req.ExecutionOrder,
	}

	feature, err := h.svc.CreateFeature(r.Context(), input)
	if err != nil {
		handleServiceError(w, err, "feature")
		return
	}

	w.Header().Set("Location", "/api/v1/features/"+feature.Key)
	respondJSON(w, http.StatusCreated, feature)
}

// UpdateFeatureRequest is the JSON body for partial feature updates.
type UpdateFeatureRequest struct {
	Title          *string `json:"title,omitempty"`
	Description    *string `json:"description,omitempty"`
	ExecutionOrder *int    `json:"execution_order,omitempty"`
}

// UpdateFeature applies partial updates to an existing feature.
// PATCH /api/v1/features/{key}
func (h *FeatureHandler) UpdateFeature(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	var req UpdateFeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updates := services.FeatureUpdates{
		Title:          req.Title,
		Description:    req.Description,
		ExecutionOrder: req.ExecutionOrder,
	}

	feature, err := h.svc.UpdateFeature(r.Context(), key, updates)
	if err != nil {
		handleServiceError(w, err, "feature")
		return
	}

	respondJSON(w, http.StatusOK, feature)
}

// DeleteFeature deletes a feature by key.
// DELETE /api/v1/features/{key}
func (h *FeatureHandler) DeleteFeature(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	if err := h.svc.DeleteFeature(r.Context(), key); err != nil {
		handleServiceError(w, err, "feature")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetNextStatus returns the available next statuses for a feature.
// GET /api/v1/features/{key}/next-status
func (h *FeatureHandler) GetNextStatus(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	info, err := h.svc.GetNextStatus(r.Context(), key)
	if err != nil {
		handleServiceError(w, err, "feature")
		return
	}

	respondJSON(w, http.StatusOK, info)
}

// TransitionStatus advances a feature to a specified status.
// POST /api/v1/features/{key}/transition
func (h *FeatureHandler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
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
		Force:  req.Force,
		Reason: req.Reason,
		Agent:  req.Agent,
	}

	result, err := h.svc.TransitionStatus(r.Context(), key, req.TargetStatus, opts)
	if err != nil {
		handleServiceError(w, err, "feature")
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Compile-time check: *services.FeatureService must satisfy FeatureServicer.
var _ FeatureServicer = (*services.FeatureService)(nil)
