package viewer

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// MutationHandler handles viewer write requests for epic, feature, and task entities.
// It is a thin wrapper: parse/validate -> call service -> format JSON response.
type MutationHandler struct {
	svc MutationServicer
}

// NewMutationHandler constructs a MutationHandler with the given service.
func NewMutationHandler(svc MutationServicer) *MutationHandler {
	if svc == nil {
		panic("MutationHandler: svc is required")
	}
	return &MutationHandler{svc: svc}
}

// RegisterRoutes mounts all viewer mutation routes on the given mux.
//
// Routes registered (with prefix = "/api/v1/viewer"):
//
//	PATCH /api/v1/viewer/epics/{key}
//	PATCH /api/v1/viewer/features/{key}
//	PATCH /api/v1/viewer/tasks/{key}
//	POST  /api/v1/viewer/epics/{key}/transition
//	POST  /api/v1/viewer/features/{key}/transition
//	POST  /api/v1/viewer/tasks/{key}/transition
func (h *MutationHandler) RegisterRoutes(mux *http.ServeMux, prefix string) {
	prefix = strings.TrimRight(prefix, "/")
	wrap := WithLocalCORS

	mux.Handle("PATCH "+prefix+"/epics/{key}", wrap(http.HandlerFunc(h.UpdateEpic)))
	mux.Handle("PATCH "+prefix+"/features/{key}", wrap(http.HandlerFunc(h.UpdateFeature)))
	mux.Handle("PATCH "+prefix+"/tasks/{key}", wrap(http.HandlerFunc(h.UpdateTask)))

	mux.Handle("POST "+prefix+"/epics/{key}/transition", wrap(http.HandlerFunc(h.TransitionEpic)))
	mux.Handle("POST "+prefix+"/features/{key}/transition", wrap(http.HandlerFunc(h.TransitionFeature)))
	mux.Handle("POST "+prefix+"/tasks/{key}/transition", wrap(http.HandlerFunc(h.TransitionTask)))

	mux.Handle("POST "+prefix+"/epics/{key}/notes", wrap(http.HandlerFunc(h.AddNote)))
	mux.Handle("POST "+prefix+"/features/{key}/notes", wrap(http.HandlerFunc(h.AddNote)))
	mux.Handle("POST "+prefix+"/tasks/{key}/notes", wrap(http.HandlerFunc(h.AddNote)))

	mux.Handle("POST "+prefix+"/epics/{key}/relationships", wrap(http.HandlerFunc(h.CreateRelationship)))
	mux.Handle("POST "+prefix+"/features/{key}/relationships", wrap(http.HandlerFunc(h.CreateRelationship)))
	mux.Handle("POST "+prefix+"/tasks/{key}/relationships", wrap(http.HandlerFunc(h.CreateRelationship)))

	mux.Handle("DELETE "+prefix+"/epics/{key}/relationships/{relationship_type}/{to_key}", wrap(http.HandlerFunc(h.DeleteRelationship)))
	mux.Handle("DELETE "+prefix+"/features/{key}/relationships/{relationship_type}/{to_key}", wrap(http.HandlerFunc(h.DeleteRelationship)))
	mux.Handle("DELETE "+prefix+"/tasks/{key}/relationships/{relationship_type}/{to_key}", wrap(http.HandlerFunc(h.DeleteRelationship)))
}

type epicMutationRequest struct {
	Title         *string `json:"title,omitempty"`
	Description   *string `json:"description,omitempty"`
	Priority      *string `json:"priority,omitempty"`
	BusinessValue *string `json:"business_value,omitempty"`
	Size          *int    `json:"size,omitempty"`
	ClearSize     bool    `json:"clear_size,omitempty"`
}

type featureMutationRequest struct {
	Title          *string `json:"title,omitempty"`
	Description    *string `json:"description,omitempty"`
	ExecutionOrder *int    `json:"execution_order,omitempty"`
	Size           *int    `json:"size,omitempty"`
	ClearSize      bool    `json:"clear_size,omitempty"`
}

type taskMutationRequest struct {
	Title          *string `json:"title,omitempty"`
	Description    *string `json:"description,omitempty"`
	Priority       *int    `json:"priority,omitempty"`
	AgentType      *string `json:"agent_type,omitempty"`
	ExecutionOrder *int    `json:"execution_order,omitempty"`
	Size           *int    `json:"size,omitempty"`
	ClearSize      bool    `json:"clear_size,omitempty"`
}

type transitionMutationRequest struct {
	TargetStatus string `json:"target_status"`
	Force        bool   `json:"force,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Agent        string `json:"agent,omitempty"`
}

type noteMutationRequest struct {
	NoteType  string `json:"note_type"`
	Content   string `json:"content"`
	CreatedBy string `json:"created_by,omitempty"`
}

type relationshipMutationRequest struct {
	RelationshipType string `json:"relationship_type"`
	ToKey            string `json:"to_key"`
}

const mutationBodySizeLimit = 2 * 1024 * 1024

func decodeMutationRequest(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, mutationBodySizeLimit)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func respondMutationDecodeError(w http.ResponseWriter, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		respondError(w, http.StatusRequestEntityTooLarge, "request body exceeds 2 MiB limit")
		return
	}
	respondError(w, http.StatusBadRequest, "invalid request body")
}

func validateAndNormalizeEpicKey(rawKey string) (string, error) {
	key := strings.ToUpper(strings.TrimSpace(rawKey))
	if !keys.IsEpicKey(key) {
		return "", errors.New("invalid epic key")
	}
	return key, nil
}

func validateAndNormalizeFeatureKey(rawKey string) (string, error) {
	key := strings.ToUpper(strings.TrimSpace(rawKey))
	if !keys.IsFeatureKey(key) {
		return "", errors.New("invalid feature key")
	}
	return key, nil
}

func validateAndNormalizeTaskKey(rawKey string) (string, error) {
	key, err := keys.NormalizeTaskKey(strings.TrimSpace(rawKey))
	if err != nil {
		return "", err
	}
	return key, nil
}

func handleMutationServiceError(w http.ResponseWriter, entityLabel string, err error) {
	if isNotFound(err) {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if isConflict(err) {
		respondError(w, http.StatusConflict, err.Error())
		return
	}

	slog.Error("viewer mutation failed", "entity", entityLabel, "error", err)
	respondError(w, http.StatusInternalServerError, "mutation failed")
}

func isConflict(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "duplicate") ||
		strings.Contains(lower, "cycle") ||
		strings.Contains(lower, "self") ||
		strings.Contains(lower, "already exists") ||
		strings.Contains(lower, "conflict")
}

// UpdateEpic applies partial updates to an existing epic.
// PATCH /api/v1/viewer/epics/{key}
func (h *MutationHandler) UpdateEpic(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key, err := validateAndNormalizeEpicKey(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid epic key: "+rawKey)
		return
	}

	var req epicMutationRequest
	if err := decodeMutationRequest(w, r, &req); err != nil {
		respondMutationDecodeError(w, err)
		return
	}

	updates := services.EpicUpdates{
		Title:         req.Title,
		Description:   req.Description,
		BusinessValue: stringToPriorityPtr(req.BusinessValue),
		Size:          req.Size,
		ClearSize:     req.ClearSize,
	}
	if req.Priority != nil {
		priority := models.Priority(*req.Priority)
		updates.Priority = &priority
	}

	epic, err := h.svc.UpdateEpic(r.Context(), key, updates)
	if err != nil {
		handleMutationServiceError(w, "epic", err)
		return
	}

	respondJSON(w, http.StatusOK, epic)
}

// UpdateFeature applies partial updates to an existing feature.
// PATCH /api/v1/viewer/features/{key}
func (h *MutationHandler) UpdateFeature(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key, err := validateAndNormalizeFeatureKey(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid feature key: "+rawKey)
		return
	}

	var req featureMutationRequest
	if err := decodeMutationRequest(w, r, &req); err != nil {
		respondMutationDecodeError(w, err)
		return
	}

	updates := services.FeatureUpdates{
		Title:          req.Title,
		Description:    req.Description,
		ExecutionOrder: req.ExecutionOrder,
		Size:           req.Size,
		ClearSize:      req.ClearSize,
	}

	feature, err := h.svc.UpdateFeature(r.Context(), key, updates)
	if err != nil {
		handleMutationServiceError(w, "feature", err)
		return
	}

	respondJSON(w, http.StatusOK, feature)
}

// UpdateTask applies partial updates to an existing task.
// PATCH /api/v1/viewer/tasks/{key}
func (h *MutationHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key, err := validateAndNormalizeTaskKey(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task key: "+rawKey)
		return
	}

	var req taskMutationRequest
	if err := decodeMutationRequest(w, r, &req); err != nil {
		respondMutationDecodeError(w, err)
		return
	}

	updates := services.TaskUpdates{
		Title:          req.Title,
		Description:    req.Description,
		Priority:       req.Priority,
		AgentType:      req.AgentType,
		ExecutionOrder: req.ExecutionOrder,
		Size:           req.Size,
		ClearSize:      req.ClearSize,
	}

	task, err := h.svc.UpdateTask(r.Context(), key, updates)
	if err != nil {
		handleMutationServiceError(w, "task", err)
		return
	}

	respondJSON(w, http.StatusOK, task)
}

// TransitionEpic advances an epic to a specified status.
// POST /api/v1/viewer/epics/{key}/transition
func (h *MutationHandler) TransitionEpic(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key, err := validateAndNormalizeEpicKey(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid epic key: "+rawKey)
		return
	}

	var req transitionMutationRequest
	if err := decodeMutationRequest(w, r, &req); err != nil {
		respondMutationDecodeError(w, err)
		return
	}
	if req.TargetStatus == "" {
		respondError(w, http.StatusBadRequest, "target_status is required")
		return
	}

	result, err := h.svc.TransitionEpic(r.Context(), key, req.TargetStatus, services.TransitionOptions{
		Force:  req.Force,
		Reason: req.Reason,
		Agent:  req.Agent,
	})
	if err != nil {
		handleMutationServiceError(w, "epic", err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// TransitionFeature advances a feature to a specified status.
// POST /api/v1/viewer/features/{key}/transition
func (h *MutationHandler) TransitionFeature(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key, err := validateAndNormalizeFeatureKey(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid feature key: "+rawKey)
		return
	}

	var req transitionMutationRequest
	if err := decodeMutationRequest(w, r, &req); err != nil {
		respondMutationDecodeError(w, err)
		return
	}
	if req.TargetStatus == "" {
		respondError(w, http.StatusBadRequest, "target_status is required")
		return
	}

	result, err := h.svc.TransitionFeature(r.Context(), key, req.TargetStatus, services.TransitionOptions{
		Force:  req.Force,
		Reason: req.Reason,
		Agent:  req.Agent,
	})
	if err != nil {
		handleMutationServiceError(w, "feature", err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// TransitionTask advances a task to a specified status.
// POST /api/v1/viewer/tasks/{key}/transition
func (h *MutationHandler) TransitionTask(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key, err := validateAndNormalizeTaskKey(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task key: "+rawKey)
		return
	}

	var req transitionMutationRequest
	if err := decodeMutationRequest(w, r, &req); err != nil {
		respondMutationDecodeError(w, err)
		return
	}
	if req.TargetStatus == "" {
		respondError(w, http.StatusBadRequest, "target_status is required")
		return
	}

	result, err := h.svc.TransitionTask(r.Context(), key, req.TargetStatus, services.TransitionOptions{
		Force:  req.Force,
		Reason: req.Reason,
		Agent:  req.Agent,
	})
	if err != nil {
		handleMutationServiceError(w, "task", err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// AddNote creates a note on the current entity.
// POST /api/v1/viewer/{epics|features|tasks}/{key}/notes
func (h *MutationHandler) AddNote(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key, err := validateAndNormalizeMutationKey(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid entity key: "+rawKey)
		return
	}

	var req noteMutationRequest
	if err := decodeMutationRequest(w, r, &req); err != nil {
		respondMutationDecodeError(w, err)
		return
	}
	if strings.TrimSpace(req.NoteType) == "" {
		respondError(w, http.StatusBadRequest, "note_type is required")
		return
	}
	if err := models.ValidateNoteType(req.NoteType); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		respondError(w, http.StatusBadRequest, "content cannot be empty")
		return
	}
	createdBy := strings.TrimSpace(req.CreatedBy)

	note, err := h.svc.AddNote(r.Context(), key, req.NoteType, content, createdBy)
	if err != nil {
		handleMutationServiceError(w, "note", err)
		return
	}

	respondJSON(w, http.StatusCreated, note)
}

// CreateRelationship creates a normalized directed relationship from the current entity.
// POST /api/v1/viewer/{epics|features|tasks}/{key}/relationships
func (h *MutationHandler) CreateRelationship(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key, err := validateAndNormalizeMutationKey(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid entity key: "+rawKey)
		return
	}

	var req relationshipMutationRequest
	if err := decodeMutationRequest(w, r, &req); err != nil {
		respondMutationDecodeError(w, err)
		return
	}
	if strings.TrimSpace(req.RelationshipType) == "" {
		respondError(w, http.StatusBadRequest, "relationship_type is required")
		return
	}
	if err := models.ValidateRelationshipType(req.RelationshipType); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	toKey, err := validateAndNormalizeMutationKey(req.ToKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid target entity key: "+req.ToKey)
		return
	}

	rel, err := h.svc.CreateRelationship(r.Context(), key, toKey, req.RelationshipType)
	if err != nil {
		handleMutationServiceError(w, "relationship", err)
		return
	}

	respondJSON(w, http.StatusCreated, rel)
}

// DeleteRelationship removes a normalized directed relationship from the current entity.
// DELETE /api/v1/viewer/{epics|features|tasks}/{key}/relationships/{relationship_type}/{to_key}
func (h *MutationHandler) DeleteRelationship(w http.ResponseWriter, r *http.Request) {
	rawKey := r.PathValue("key")
	key, err := validateAndNormalizeMutationKey(rawKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid entity key: "+rawKey)
		return
	}

	relationshipType := strings.TrimSpace(r.PathValue("relationship_type"))
	if relationshipType == "" {
		respondError(w, http.StatusBadRequest, "relationship_type is required")
		return
	}
	if err := models.ValidateRelationshipType(relationshipType); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	toKeyRaw := r.PathValue("to_key")
	toKey, err := validateAndNormalizeMutationKey(toKeyRaw)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid target entity key: "+toKeyRaw)
		return
	}

	if err := h.svc.DeleteRelationship(r.Context(), key, relationshipType, toKey); err != nil {
		handleMutationServiceError(w, "relationship", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validateAndNormalizeMutationKey(rawKey string) (string, error) {
	key := strings.ToUpper(strings.TrimSpace(rawKey))
	if key == "" {
		return "", errors.New("empty entity key")
	}
	switch {
	case keys.IsEpicKey(key):
		return key, nil
	case keys.IsFeatureKey(key):
		return key, nil
	case keys.IsShortTaskKey(key), keys.IsTaskKey(key):
		return keys.NormalizeTaskKey(key)
	default:
		return "", errors.New("invalid entity key")
	}
}

func stringToPriorityPtr(v *string) *models.Priority {
	if v == nil {
		return nil
	}
	priority := models.Priority(*v)
	return &priority
}
