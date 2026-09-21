package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// mockTaskService is a test double for TaskServicer.
type mockTaskService struct {
	getTaskFn          func(ctx context.Context, key string) (*models.Task, error)
	listTasksFn        func(ctx context.Context, filters services.TaskFilters) ([]*models.Task, error)
	createTaskFn       func(ctx context.Context, input services.CreateTaskInput) (*models.Task, bool, error)
	updateTaskFn       func(ctx context.Context, key string, updates services.TaskUpdates) (*models.Task, error)
	deleteTaskFn       func(ctx context.Context, key string) error
	transitionStatusFn func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	getNextStatusFn    func(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

func (m *mockTaskService) GetTask(ctx context.Context, key string) (*models.Task, error) {
	if m.getTaskFn != nil {
		return m.getTaskFn(ctx, key)
	}
	return nil, nil
}

func (m *mockTaskService) ListTasks(ctx context.Context, filters services.TaskFilters) ([]*models.Task, error) {
	if m.listTasksFn != nil {
		return m.listTasksFn(ctx, filters)
	}
	return nil, nil
}

func (m *mockTaskService) CreateTask(ctx context.Context, input services.CreateTaskInput) (*models.Task, bool, error) {
	if m.createTaskFn != nil {
		return m.createTaskFn(ctx, input)
	}
	return nil, false, nil
}

func (m *mockTaskService) UpdateTask(ctx context.Context, key string, updates services.TaskUpdates) (*models.Task, error) {
	if m.updateTaskFn != nil {
		return m.updateTaskFn(ctx, key, updates)
	}
	return nil, nil
}

func (m *mockTaskService) DeleteTask(ctx context.Context, key string) error {
	if m.deleteTaskFn != nil {
		return m.deleteTaskFn(ctx, key)
	}
	return nil
}

func (m *mockTaskService) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	if m.transitionStatusFn != nil {
		return m.transitionStatusFn(ctx, key, targetStatus, opts)
	}
	return nil, nil
}

func (m *mockTaskService) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	if m.getNextStatusFn != nil {
		return m.getNextStatusFn(ctx, key)
	}
	return nil, nil
}

// newTaskHandlerMux creates a mux with the TaskHandler registered, for use in tests.
func newTaskHandlerMux(svc TaskServicer) *http.ServeMux {
	mux := http.NewServeMux()
	NewTaskHandler(svc).RegisterRoutes(mux)
	return mux
}

func TestTaskHandler_GetTask(t *testing.T) {
	t.Run("returns task when found", func(t *testing.T) {
		svc := &mockTaskService{
			getTaskFn: func(_ context.Context, key string) (*models.Task, error) {
				if key != "E07-F01-001" {
					t.Errorf("unexpected key: %s", key)
				}
				task := &models.Task{Status: "todo"}
				task.Key = "E07-F01-001"
				task.Title = "Test Task"
				return task, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/E07-F01-001", nil)
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var task models.Task
		if err := json.NewDecoder(rec.Body).Decode(&task); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if task.Key != "E07-F01-001" {
			t.Errorf("expected key E07-F01-001, got %s", task.Key)
		}
		if task.Title != "Test Task" {
			t.Errorf("expected title 'Test Task', got %s", task.Title)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		svc := &mockTaskService{
			getTaskFn: func(_ context.Context, key string) (*models.Task, error) {
				return nil, fmt.Errorf("task not found with key %s", key)
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/E07-F01-999", nil)
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}

		var errResp ErrorResponse
		if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if errResp.Error == "" {
			t.Error("expected non-empty error field")
		}
	})
}

func TestTaskHandler_ListTasks(t *testing.T) {
	t.Run("returns tasks with filters", func(t *testing.T) {
		var capturedFilters services.TaskFilters
		svc := &mockTaskService{
			listTasksFn: func(_ context.Context, filters services.TaskFilters) ([]*models.Task, error) {
				capturedFilters = filters
				t1 := &models.Task{Status: "todo"}
				t1.Key = "E07-F01-001"
				t1.Title = "Task 1"
				t2 := &models.Task{Status: "todo"}
				t2.Key = "E07-F01-002"
				t2.Title = "Task 2"
				return []*models.Task{t1, t2}, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?epic=E07&feature=F01&status=todo", nil)
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if capturedFilters.EpicKey != "E07" {
			t.Errorf("expected EpicKey E07, got %s", capturedFilters.EpicKey)
		}
		if capturedFilters.FeatureKey != "F01" {
			t.Errorf("expected FeatureKey F01, got %s", capturedFilters.FeatureKey)
		}
		if capturedFilters.Status != "todo" {
			t.Errorf("expected Status todo, got %s", capturedFilters.Status)
		}

		var tasks []*models.Task
		if err := json.NewDecoder(rec.Body).Decode(&tasks); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(tasks))
		}
	})

	t.Run("returns empty list when none found", func(t *testing.T) {
		svc := &mockTaskService{
			listTasksFn: func(_ context.Context, _ services.TaskFilters) ([]*models.Task, error) {
				return []*models.Task{}, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

func TestTaskHandler_CreateTask(t *testing.T) {
	t.Run("creates task with valid input", func(t *testing.T) {
		var capturedInput services.CreateTaskInput
		svc := &mockTaskService{
			createTaskFn: func(_ context.Context, input services.CreateTaskInput) (*models.Task, bool, error) {
				capturedInput = input
				task := &models.Task{Status: "todo"}
				task.Key = "E07-F01-003"
				task.Title = input.Title
				return task, false, nil
			},
		}

		body := `{"epic_key":"E07","feature_key":"F01","title":"New Task","agent_type":"backend","priority":7}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if capturedInput.EpicKey != "E07" {
			t.Errorf("expected EpicKey E07, got %s", capturedInput.EpicKey)
		}
		if capturedInput.Title != "New Task" {
			t.Errorf("expected Title 'New Task', got %s", capturedInput.Title)
		}
		if capturedInput.AgentType != "backend" {
			t.Errorf("expected AgentType 'backend', got %s", capturedInput.AgentType)
		}

		if loc := rec.Header().Get("Location"); loc != "/api/v1/tasks/E07-F01-003" {
			t.Errorf("expected Location /api/v1/tasks/E07-F01-003, got %s", loc)
		}
	})

	t.Run("returns 400 when title missing", func(t *testing.T) {
		svc := &mockTaskService{}
		body := `{"epic_key":"E07","feature_key":"F01"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("returns 400 when body is invalid JSON", func(t *testing.T) {
		svc := &mockTaskService{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString("not-json"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestTaskHandler_UpdateTask(t *testing.T) {
	t.Run("updates task fields", func(t *testing.T) {
		newTitle := "Updated Title"
		svc := &mockTaskService{
			updateTaskFn: func(_ context.Context, key string, updates services.TaskUpdates) (*models.Task, error) {
				if key != "E07-F01-001" {
					t.Errorf("unexpected key: %s", key)
				}
				t := &models.Task{Status: "todo"}
				t.Key = key
				t.Title = *updates.Title
				return t, nil
			},
		}

		body := `{"title":"Updated Title"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/E07-F01-001", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}

		var task models.Task
		if err := json.NewDecoder(rec.Body).Decode(&task); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if task.Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, task.Title)
		}
	})

	t.Run("returns 400 on invalid JSON", func(t *testing.T) {
		svc := &mockTaskService{}
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/E07-F01-001", bytes.NewBufferString("{invalid"))
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestTaskHandler_DeleteTask(t *testing.T) {
	t.Run("deletes task and returns 204", func(t *testing.T) {
		var deletedKey string
		svc := &mockTaskService{
			deleteTaskFn: func(_ context.Context, key string) error {
				deletedKey = key
				return nil
			},
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/E07-F01-001", nil)
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
		if deletedKey != "E07-F01-001" {
			t.Errorf("expected deleted key E07-F01-001, got %s", deletedKey)
		}
	})

	t.Run("returns 404 when task not found", func(t *testing.T) {
		svc := &mockTaskService{
			deleteTaskFn: func(_ context.Context, key string) error {
				return fmt.Errorf("task not found with key %s", key)
			},
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/E07-F01-999", nil)
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestTaskHandler_GetNextStatus(t *testing.T) {
	t.Run("returns next status info", func(t *testing.T) {
		svc := &mockTaskService{
			getNextStatusFn: func(_ context.Context, key string) (*services.NextStatusInfo, error) {
				return &services.NextStatusInfo{
					EntityKey:     key,
					CurrentStatus: "todo",
				}, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/E07-F01-001/next-status", nil)
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

func TestTaskHandler_TransitionStatus(t *testing.T) {
	t.Run("transitions task status", func(t *testing.T) {
		var capturedTarget string
		svc := &mockTaskService{
			transitionStatusFn: func(_ context.Context, key, targetStatus string, _ services.TransitionOptions) (*services.TransitionResult, error) {
				capturedTarget = targetStatus
				return &services.TransitionResult{
					EntityKey:  key,
					FromStatus: "todo",
					ToStatus:   targetStatus,
				}, nil
			},
		}

		body := `{"target_status":"in_progress"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/E07-F01-001/transition", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if capturedTarget != "in_progress" {
			t.Errorf("expected target in_progress, got %s", capturedTarget)
		}
	})

	t.Run("returns 400 when target_status missing", func(t *testing.T) {
		svc := &mockTaskService{}
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/E07-F01-001/transition", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("returns 400 on invalid JSON", func(t *testing.T) {
		svc := &mockTaskService{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/E07-F01-001/transition", bytes.NewBufferString("{invalid"))
		rec := httptest.NewRecorder()
		newTaskHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestNewTaskHandler_PanicsOnNilService(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil service, but did not panic")
		}
	}()
	NewTaskHandler(nil)
}
