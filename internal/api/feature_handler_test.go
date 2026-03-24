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

// mockFeatureService is a test double for FeatureServicer.
type mockFeatureService struct {
	getFeatureFn       func(ctx context.Context, key string) (*models.Feature, error)
	listFeaturesFn     func(ctx context.Context, filters services.FeatureFilters) ([]*models.Feature, error)
	createFeatureFn    func(ctx context.Context, input services.CreateFeatureInput) (*models.Feature, error)
	updateFeatureFn    func(ctx context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error)
	deleteFeatureFn    func(ctx context.Context, key string) error
	transitionStatusFn func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	getNextStatusFn    func(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

func (m *mockFeatureService) GetFeature(ctx context.Context, key string) (*models.Feature, error) {
	if m.getFeatureFn != nil {
		return m.getFeatureFn(ctx, key)
	}
	return nil, nil
}

func (m *mockFeatureService) ListFeatures(ctx context.Context, filters services.FeatureFilters) ([]*models.Feature, error) {
	if m.listFeaturesFn != nil {
		return m.listFeaturesFn(ctx, filters)
	}
	return nil, nil
}

func (m *mockFeatureService) CreateFeature(ctx context.Context, input services.CreateFeatureInput) (*models.Feature, error) {
	if m.createFeatureFn != nil {
		return m.createFeatureFn(ctx, input)
	}
	return nil, nil
}

func (m *mockFeatureService) UpdateFeature(ctx context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error) {
	if m.updateFeatureFn != nil {
		return m.updateFeatureFn(ctx, key, updates)
	}
	return nil, nil
}

func (m *mockFeatureService) DeleteFeature(ctx context.Context, key string) error {
	if m.deleteFeatureFn != nil {
		return m.deleteFeatureFn(ctx, key)
	}
	return nil
}

func (m *mockFeatureService) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	if m.transitionStatusFn != nil {
		return m.transitionStatusFn(ctx, key, targetStatus, opts)
	}
	return nil, nil
}

func (m *mockFeatureService) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	if m.getNextStatusFn != nil {
		return m.getNextStatusFn(ctx, key)
	}
	return nil, nil
}

func newFeatureHandlerMux(svc FeatureServicer) *http.ServeMux {
	mux := http.NewServeMux()
	NewFeatureHandler(svc).RegisterRoutes(mux)
	return mux
}

func TestFeatureHandler_GetFeature(t *testing.T) {
	t.Run("returns feature when found", func(t *testing.T) {
		svc := &mockFeatureService{
			getFeatureFn: func(_ context.Context, key string) (*models.Feature, error) {
				f := &models.Feature{}
				f.Key = key
				f.Title = "My Feature"
				return f, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/features/E07-F01", nil)
		rec := httptest.NewRecorder()
		newFeatureHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var feature models.Feature
		if err := json.NewDecoder(rec.Body).Decode(&feature); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if feature.Title != "My Feature" {
			t.Errorf("expected title 'My Feature', got %s", feature.Title)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		svc := &mockFeatureService{
			getFeatureFn: func(_ context.Context, key string) (*models.Feature, error) {
				return nil, fmt.Errorf("feature not found with key %s", key)
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/features/E07-F99", nil)
		rec := httptest.NewRecorder()
		newFeatureHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestFeatureHandler_ListFeatures(t *testing.T) {
	t.Run("returns features with epic filter", func(t *testing.T) {
		var capturedFilters services.FeatureFilters
		svc := &mockFeatureService{
			listFeaturesFn: func(_ context.Context, filters services.FeatureFilters) ([]*models.Feature, error) {
				capturedFilters = filters
				f1 := &models.Feature{}
				f1.Key = "E07-F01"
				f1.Title = "Feature 1"
				return []*models.Feature{f1}, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/features?epic=E07", nil)
		rec := httptest.NewRecorder()
		newFeatureHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if capturedFilters.EpicKey != "E07" {
			t.Errorf("expected EpicKey E07, got %s", capturedFilters.EpicKey)
		}
	})
}

func TestFeatureHandler_CreateFeature(t *testing.T) {
	t.Run("creates feature with valid input", func(t *testing.T) {
		svc := &mockFeatureService{
			createFeatureFn: func(_ context.Context, input services.CreateFeatureInput) (*models.Feature, error) {
				f := &models.Feature{}
				f.Key = "E07-F05"
				f.Title = input.Title
				return f, nil
			},
		}

		body := `{"epic_key":"E07","title":"New Feature"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/features", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		newFeatureHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
		}

		if loc := rec.Header().Get("Location"); loc != "/api/v1/features/E07-F05" {
			t.Errorf("expected Location /api/v1/features/E07-F05, got %s", loc)
		}
	})

	t.Run("returns 400 when title missing", func(t *testing.T) {
		svc := &mockFeatureService{}
		body := `{"epic_key":"E07"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/features", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newFeatureHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestFeatureHandler_UpdateFeature(t *testing.T) {
	t.Run("updates feature and returns 200", func(t *testing.T) {
		newTitle := "Updated Feature"
		svc := &mockFeatureService{
			updateFeatureFn: func(_ context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error) {
				f := &models.Feature{}
				f.Key = key
				f.Title = *updates.Title
				return f, nil
			},
		}

		body := `{"title":"Updated Feature"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/features/E07-F01", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newFeatureHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var feature models.Feature
		if err := json.NewDecoder(rec.Body).Decode(&feature); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if feature.Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, feature.Title)
		}
	})
}

func TestFeatureHandler_DeleteFeature(t *testing.T) {
	t.Run("deletes feature and returns 204", func(t *testing.T) {
		svc := &mockFeatureService{
			deleteFeatureFn: func(_ context.Context, key string) error {
				return nil
			},
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/features/E07-F01", nil)
		rec := httptest.NewRecorder()
		newFeatureHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		svc := &mockFeatureService{
			deleteFeatureFn: func(_ context.Context, key string) error {
				return fmt.Errorf("feature not found with key %s", key)
			},
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/features/E07-F99", nil)
		rec := httptest.NewRecorder()
		newFeatureHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestFeatureHandler_TransitionStatus(t *testing.T) {
	t.Run("transitions feature status", func(t *testing.T) {
		svc := &mockFeatureService{
			transitionStatusFn: func(_ context.Context, key, targetStatus string, _ services.TransitionOptions) (*services.TransitionResult, error) {
				return &services.TransitionResult{
					EntityKey:  key,
					FromStatus: "draft",
					ToStatus:   targetStatus,
				}, nil
			},
		}

		body := `{"target_status":"in_progress"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/features/E07-F01/transition", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		newFeatureHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("returns 400 when target_status missing", func(t *testing.T) {
		svc := &mockFeatureService{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/features/E07-F01/transition", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()
		newFeatureHandlerMux(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestNewFeatureHandler_PanicsOnNilService(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil service, but did not panic")
		}
	}()
	NewFeatureHandler(nil)
}
