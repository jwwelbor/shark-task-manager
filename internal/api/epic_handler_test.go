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

// mockEpicService is a test double for EpicServicer.
type mockEpicService struct {
	getEpicFn          func(ctx context.Context, key string) (*models.Epic, error)
	listEpicsFn        func(ctx context.Context, filters services.EpicFilters) ([]*models.Epic, error)
	createEpicFn       func(ctx context.Context, input services.CreateEpicInput) (*models.Epic, error)
	updateEpicFn       func(ctx context.Context, key string, updates services.EpicUpdates) (*models.Epic, error)
	deleteEpicFn       func(ctx context.Context, key string) error
	transitionStatusFn func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	getNextStatusFn    func(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

func (m *mockEpicService) GetEpic(ctx context.Context, key string) (*models.Epic, error) {
	if m.getEpicFn != nil {
		return m.getEpicFn(ctx, key)
	}
	return nil, nil
}

func (m *mockEpicService) ListEpics(ctx context.Context, filters services.EpicFilters) ([]*models.Epic, error) {
	if m.listEpicsFn != nil {
		return m.listEpicsFn(ctx, filters)
	}
	return nil, nil
}

func (m *mockEpicService) CreateEpic(ctx context.Context, input services.CreateEpicInput) (*models.Epic, error) {
	if m.createEpicFn != nil {
		return m.createEpicFn(ctx, input)
	}
	return nil, nil
}

func (m *mockEpicService) UpdateEpic(ctx context.Context, key string, updates services.EpicUpdates) (*models.Epic, error) {
	if m.updateEpicFn != nil {
		return m.updateEpicFn(ctx, key, updates)
	}
	return nil, nil
}

func (m *mockEpicService) DeleteEpic(ctx context.Context, key string) error {
	if m.deleteEpicFn != nil {
		return m.deleteEpicFn(ctx, key)
	}
	return nil
}

func (m *mockEpicService) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	if m.transitionStatusFn != nil {
		return m.transitionStatusFn(ctx, key, targetStatus, opts)
	}
	return nil, nil
}

func (m *mockEpicService) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	if m.getNextStatusFn != nil {
		return m.getNextStatusFn(ctx, key)
	}
	return nil, nil
}

func newEpicHandlerMux(svc EpicServicer) *http.ServeMux {
	mux := http.NewServeMux()
	NewEpicHandler(svc).RegisterRoutes(mux)
	return mux
}

func TestEpicHandler_GetEpic(t *testing.T) {
	t.Run("returns epic when found", func(t *testing.T) {
		svc := &mockEpicService{
			getEpicFn: func(_ context.Context, key string) (*models.Epic, error) {
				epic := &models.Epic{}
				epic.Key = key
				epic.Title = "My Epic"
				return epic, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/epics/E07", nil)
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var epic models.Epic
		if err := json.NewDecoder(rec.Body).Decode(&epic); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if epic.Title != "My Epic" {
			t.Errorf("expected title 'My Epic', got %s", epic.Title)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		svc := &mockEpicService{
			getEpicFn: func(_ context.Context, key string) (*models.Epic, error) {
				return nil, fmt.Errorf("epic not found with key %s", key)
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/epics/E99", nil)
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestEpicHandler_ListEpics(t *testing.T) {
	t.Run("returns epics with status filter", func(t *testing.T) {
		var capturedFilters services.EpicFilters
		svc := &mockEpicService{
			listEpicsFn: func(_ context.Context, filters services.EpicFilters) ([]*models.Epic, error) {
				capturedFilters = filters
				e1 := &models.Epic{}
				e1.Key = "E01"
				e1.Title = "Epic 1"
				e2 := &models.Epic{}
				e2.Key = "E02"
				e2.Title = "Epic 2"
				return []*models.Epic{e1, e2}, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/epics?status=draft", nil)
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if capturedFilters.Status != "draft" {
			t.Errorf("expected Status draft, got %s", capturedFilters.Status)
		}

		var epics []*models.Epic
		if err := json.NewDecoder(rec.Body).Decode(&epics); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(epics) != 2 {
			t.Errorf("expected 2 epics, got %d", len(epics))
		}
	})
}

func TestEpicHandler_CreateEpic(t *testing.T) {
	t.Run("creates epic with valid input", func(t *testing.T) {
		svc := &mockEpicService{
			createEpicFn: func(_ context.Context, input services.CreateEpicInput) (*models.Epic, error) {
				epic := &models.Epic{}
				epic.Key = "E10"
				epic.Title = input.Title
				return epic, nil
			},
		}

		body := `{"title":"New Epic","priority":"high"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/epics", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
		}

		if loc := rec.Header().Get("Location"); loc != "/api/v1/epics/E10" {
			t.Errorf("expected Location /api/v1/epics/E10, got %s", loc)
		}
	})

	t.Run("returns 400 when title missing", func(t *testing.T) {
		svc := &mockEpicService{}
		body := `{"priority":"high"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/epics", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("returns 400 on invalid JSON", func(t *testing.T) {
		svc := &mockEpicService{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/epics", bytes.NewBufferString("not-json"))
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestEpicHandler_UpdateEpic(t *testing.T) {
	t.Run("updates epic and returns 200", func(t *testing.T) {
		newTitle := "Updated Epic"
		svc := &mockEpicService{
			updateEpicFn: func(_ context.Context, key string, updates services.EpicUpdates) (*models.Epic, error) {
				epic := &models.Epic{}
				epic.Key = key
				epic.Title = *updates.Title
				return epic, nil
			},
		}

		body := `{"title":"Updated Epic"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/epics/E07", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var epic models.Epic
		if err := json.NewDecoder(rec.Body).Decode(&epic); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if epic.Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, epic.Title)
		}
	})

	t.Run("returns 400 on invalid JSON", func(t *testing.T) {
		svc := &mockEpicService{}
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/epics/E07", bytes.NewBufferString("{invalid"))
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestEpicHandler_DeleteEpic(t *testing.T) {
	t.Run("deletes epic and returns 204", func(t *testing.T) {
		svc := &mockEpicService{
			deleteEpicFn: func(_ context.Context, key string) error {
				return nil
			},
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/epics/E07", nil)
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		svc := &mockEpicService{
			deleteEpicFn: func(_ context.Context, key string) error {
				return fmt.Errorf("epic not found with key %s", key)
			},
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/epics/E99", nil)
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestEpicHandler_GetNextStatus(t *testing.T) {
	t.Run("returns next status info", func(t *testing.T) {
		svc := &mockEpicService{
			getNextStatusFn: func(_ context.Context, key string) (*services.NextStatusInfo, error) {
				return &services.NextStatusInfo{
					EntityKey:     key,
					CurrentStatus: "draft",
				}, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/epics/E07/next-status", nil)
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

func TestEpicHandler_TransitionStatus(t *testing.T) {
	t.Run("transitions epic status", func(t *testing.T) {
		var capturedKey, capturedTarget string
		svc := &mockEpicService{
			transitionStatusFn: func(_ context.Context, key, targetStatus string, _ services.TransitionOptions) (*services.TransitionResult, error) {
				capturedKey = key
				capturedTarget = targetStatus
				return &services.TransitionResult{
					EntityKey:    key,
					FromStatus:   "draft",
					ToStatus:     targetStatus,
					Transitioned: true,
				}, nil
			},
		}

		body := `{"target_status":"in_progress","reason":"starting work"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/epics/E07/transition", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if capturedKey != "E07" {
			t.Errorf("expected key E07, got %s", capturedKey)
		}
		if capturedTarget != "in_progress" {
			t.Errorf("expected target in_progress, got %s", capturedTarget)
		}
	})

	t.Run("returns 400 when target_status missing", func(t *testing.T) {
		svc := &mockEpicService{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/epics/E07/transition", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()
		newEpicHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestNewEpicHandler_PanicsOnNilService(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil service, but did not panic")
		}
	}()
	NewEpicHandler(nil)
}
