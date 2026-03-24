package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// TaskServicer is the interface that TaskHandler depends on.
// Defined here so tests can inject a mock without importing the full service.
type TaskServicer interface {
	GetTask(ctx context.Context, key string) (*models.Task, error)
	ListTasks(ctx context.Context, filters services.TaskFilters) ([]*models.Task, error)
	CreateTask(ctx context.Context, input services.CreateTaskInput) (*models.Task, error)
	UpdateTask(ctx context.Context, key string, updates services.TaskUpdates) (*models.Task, error)
	DeleteTask(ctx context.Context, key string) error
	TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

// TaskHandler handles HTTP requests for task CRUD and lifecycle operations.
type TaskHandler struct {
	svc TaskServicer
}

// NewTaskHandler constructs a TaskHandler with the given service.
func NewTaskHandler(svc TaskServicer) *TaskHandler {
	if svc == nil {
		panic("TaskHandler: svc is required")
	}
	return &TaskHandler{svc: svc}
}

// RegisterRoutes registers task routes on the given mux.
// Route patterns (Go 1.22+):
//
//	GET    /api/v1/tasks          → ListTasks
//	POST   /api/v1/tasks          → CreateTask
//	GET    /api/v1/tasks/{key}    → GetTask
//	PATCH  /api/v1/tasks/{key}    → UpdateTask
//	DELETE /api/v1/tasks/{key}    → DeleteTask
//	GET    /api/v1/tasks/{key}/next-status → GetNextStatus
//	POST   /api/v1/tasks/{key}/transition  → TransitionStatus
func (h *TaskHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/tasks", h.ListTasks)
	mux.HandleFunc("POST /api/v1/tasks", h.CreateTask)
	mux.HandleFunc("GET /api/v1/tasks/{key}", h.GetTask)
	mux.HandleFunc("PATCH /api/v1/tasks/{key}", h.UpdateTask)
	mux.HandleFunc("DELETE /api/v1/tasks/{key}", h.DeleteTask)
	mux.HandleFunc("GET /api/v1/tasks/{key}/next-status", h.GetNextStatus)
	mux.HandleFunc("POST /api/v1/tasks/{key}/transition", h.TransitionStatus)
}

// GetTask returns a single task by key.
// GET /api/v1/tasks/{key}
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	task, err := h.svc.GetTask(r.Context(), key)
	if err != nil {
		handleServiceError(w, err, "task")
		return
	}

	respondJSON(w, http.StatusOK, task)
}

// ListTasks returns a filtered list of tasks.
// GET /api/v1/tasks?epic=E07&feature=F01&status=todo&agent=backend&show_all=true&limit=50&offset=0
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filters := services.TaskFilters{
		EpicKey:     q.Get("epic"),
		FeatureKey:  q.Get("feature"),
		Status:      q.Get("status"),
		AgentType:   q.Get("agent"),
		TitleSearch: q.Get("title_search"),
		ShowAll:     q.Get("show_all") == "true",
		Blocked:     q.Get("blocked") == "true",
	}

	if v, err := strconv.Atoi(q.Get("limit")); err == nil {
		filters.Limit = v
	}
	if v, err := strconv.Atoi(q.Get("offset")); err == nil {
		filters.Offset = v
	}

	tasks, err := h.svc.ListTasks(r.Context(), filters)
	if err != nil {
		handleServiceError(w, err, "task")
		return
	}

	respondJSON(w, http.StatusOK, tasks)
}

// CreateTaskRequest is the JSON body for creating a task.
type CreateTaskRequest struct {
	EpicKey        string   `json:"epic_key"`
	FeatureKey     string   `json:"feature_key"`
	Title          string   `json:"title"`
	AgentType      string   `json:"agent_type,omitempty"`
	Priority       int      `json:"priority,omitempty"`
	ExecutionOrder int      `json:"execution_order,omitempty"`
	DependsOn      []string `json:"depends_on,omitempty"`
	Description    string   `json:"description,omitempty"`
}

// CreateTask creates a new task.
// POST /api/v1/tasks
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" || req.EpicKey == "" || req.FeatureKey == "" {
		respondError(w, http.StatusBadRequest, "epic_key, feature_key, and title are required")
		return
	}

	input := services.CreateTaskInput{
		EpicKey:        req.EpicKey,
		FeatureKey:     req.FeatureKey,
		Title:          req.Title,
		AgentType:      req.AgentType,
		Priority:       req.Priority,
		ExecutionOrder: req.ExecutionOrder,
		DependsOn:      req.DependsOn,
		Description:    req.Description,
	}

	task, err := h.svc.CreateTask(r.Context(), input)
	if err != nil {
		handleServiceError(w, err, "task")
		return
	}

	w.Header().Set("Location", "/api/v1/tasks/"+task.Key)
	respondJSON(w, http.StatusCreated, task)
}

// UpdateTaskRequest is the JSON body for partial task updates.
type UpdateTaskRequest struct {
	Title          *string `json:"title,omitempty"`
	Description    *string `json:"description,omitempty"`
	Priority       *int    `json:"priority,omitempty"`
	AgentType      *string `json:"agent_type,omitempty"`
	ExecutionOrder *int    `json:"execution_order,omitempty"`
}

// UpdateTask applies partial updates to an existing task.
// PATCH /api/v1/tasks/{key}
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updates := services.TaskUpdates{
		Title:          req.Title,
		Description:    req.Description,
		Priority:       req.Priority,
		AgentType:      req.AgentType,
		ExecutionOrder: req.ExecutionOrder,
	}

	task, err := h.svc.UpdateTask(r.Context(), key, updates)
	if err != nil {
		handleServiceError(w, err, "task")
		return
	}

	respondJSON(w, http.StatusOK, task)
}

// DeleteTask deletes a task by key.
// DELETE /api/v1/tasks/{key}
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	if err := h.svc.DeleteTask(r.Context(), key); err != nil {
		handleServiceError(w, err, "task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetNextStatus returns the available next statuses for a task.
// GET /api/v1/tasks/{key}/next-status
func (h *TaskHandler) GetNextStatus(w http.ResponseWriter, r *http.Request) {
	key := pathParam(r, "key")

	info, err := h.svc.GetNextStatus(r.Context(), key)
	if err != nil {
		handleServiceError(w, err, "task")
		return
	}

	respondJSON(w, http.StatusOK, info)
}

// TransitionStatusRequest is the JSON body for a status transition.
type TransitionStatusRequest struct {
	TargetStatus string `json:"target_status"`
	Force        bool   `json:"force,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Agent        string `json:"agent,omitempty"`
}

// TransitionStatus advances a task to a specified status.
// POST /api/v1/tasks/{key}/transition
func (h *TaskHandler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
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
		handleServiceError(w, err, "task")
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Compile-time check: *services.TaskService must satisfy TaskServicer.
var _ TaskServicer = (*services.TaskService)(nil)
