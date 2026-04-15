package viewer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// ----- MockViewerServicer -----

// MockViewerServicer is a test double for ViewerServicer.
// Each method delegates to its Func field if non-nil; otherwise it returns a
// descriptive error to prevent accidental nil-pointer panics in tests.
type MockViewerServicer struct {
	SummaryFunc        func(ctx context.Context) (*services.SummaryResponse, error)
	HierarchyFunc      func(ctx context.Context) (*services.HierarchyResponse, error)
	HistoryFunc        func(ctx context.Context, key string) (*services.HistoryResponse, error)
	FileFunc           func(ctx context.Context, key string) (*services.FileResponse, error)
	FileByPathFunc     func(ctx context.Context, filePath string) (*services.FileResponse, error)
	FolderFilesFunc    func(ctx context.Context, dirPath string) (*services.FolderFilesResponse, error)
	FeatureTasksFunc   func(ctx context.Context, featureKey string, opts services.FeatureTaskOptions) (*services.FeatureTasksResponse, error)
	RecentActivityFunc func(ctx context.Context, opts services.RecentActivityOptions) (*services.RecentActivityResponse, error)
	WorkflowMetaFunc   func(ctx context.Context) (*services.WorkflowMetaResponse, error)
	NotesFunc          func(ctx context.Context, key string) (*services.NotesResponse, error)
	RelatedDocsFunc    func(ctx context.Context, key string) (*services.RelatedDocsResponse, error)
}

func (m *MockViewerServicer) Summary(ctx context.Context) (*services.SummaryResponse, error) {
	if m.SummaryFunc != nil {
		return m.SummaryFunc(ctx)
	}
	return nil, errors.New("SummaryFunc not set in mock")
}

func (m *MockViewerServicer) Hierarchy(ctx context.Context) (*services.HierarchyResponse, error) {
	if m.HierarchyFunc != nil {
		return m.HierarchyFunc(ctx)
	}
	return nil, errors.New("HierarchyFunc not set in mock")
}

func (m *MockViewerServicer) History(ctx context.Context, key string) (*services.HistoryResponse, error) {
	if m.HistoryFunc != nil {
		return m.HistoryFunc(ctx, key)
	}
	return nil, errors.New("HistoryFunc not set in mock")
}

func (m *MockViewerServicer) File(ctx context.Context, key string) (*services.FileResponse, error) {
	if m.FileFunc != nil {
		return m.FileFunc(ctx, key)
	}
	return nil, errors.New("FileFunc not set in mock")
}

func (m *MockViewerServicer) FileByPath(ctx context.Context, filePath string) (*services.FileResponse, error) {
	if m.FileByPathFunc != nil {
		return m.FileByPathFunc(ctx, filePath)
	}
	return nil, errors.New("FileByPathFunc not set in mock")
}

func (m *MockViewerServicer) FolderFiles(ctx context.Context, dirPath string) (*services.FolderFilesResponse, error) {
	if m.FolderFilesFunc != nil {
		return m.FolderFilesFunc(ctx, dirPath)
	}
	return nil, errors.New("FolderFilesFunc not set in mock")
}

func (m *MockViewerServicer) FeatureTasks(ctx context.Context, featureKey string, opts services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
	if m.FeatureTasksFunc != nil {
		return m.FeatureTasksFunc(ctx, featureKey, opts)
	}
	return nil, errors.New("FeatureTasksFunc not set in mock")
}

func (m *MockViewerServicer) RecentActivity(ctx context.Context, opts services.RecentActivityOptions) (*services.RecentActivityResponse, error) {
	if m.RecentActivityFunc != nil {
		return m.RecentActivityFunc(ctx, opts)
	}
	return nil, errors.New("RecentActivityFunc not set in mock")
}

func (m *MockViewerServicer) WorkflowMeta(ctx context.Context) (*services.WorkflowMetaResponse, error) {
	if m.WorkflowMetaFunc != nil {
		return m.WorkflowMetaFunc(ctx)
	}
	return nil, errors.New("WorkflowMetaFunc not set in mock")
}

func (m *MockViewerServicer) Notes(ctx context.Context, key string) (*services.NotesResponse, error) {
	if m.NotesFunc != nil {
		return m.NotesFunc(ctx, key)
	}
	return nil, errors.New("NotesFunc not set in mock")
}

func (m *MockViewerServicer) RelatedDocs(ctx context.Context, key string) (*services.RelatedDocsResponse, error) {
	if m.RelatedDocsFunc != nil {
		return m.RelatedDocsFunc(ctx, key)
	}
	return nil, errors.New("RelatedDocsFunc not set in mock")
}

// ----- helpers -----

// newTestMux creates an http.ServeMux with ViewerHandler routes registered.
func newTestMux(mock *MockViewerServicer) *http.ServeMux {
	mux := http.NewServeMux()
	NewViewerHandler(mock).RegisterRoutes(mux, "/api/v1/viewer")
	return mux
}

// makeRequest issues a GET request (no body) to the mux and returns the recorder.
func makeRequest(method, path string, handler http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// assertJSON asserts that rec.Body parses as JSON into dest without error.
func assertJSON(t *testing.T, rec *httptest.ResponseRecorder, dest interface{}) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(dest); err != nil {
		t.Fatalf("failed to decode JSON response: %v\nbody: %s", err, rec.Body.String())
	}
}

// assertContentTypeJSON asserts Content-Type is application/json.
func assertContentTypeJSON(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	ct := rec.Header().Get("Content-Type")
	if ct == "" || ct[:16] != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

// ----- TC-H-001 through TC-H-003: GET /summary -----

func TestHandler_Summary(t *testing.T) {
	// TC-H-001: Happy path — populated project
	t.Run("TC-H-001_happy_path_populated", func(t *testing.T) {
		mock := &MockViewerServicer{
			SummaryFunc: func(_ context.Context) (*services.SummaryResponse, error) {
				return &services.SummaryResponse{
					Epics: services.SummaryEntityCounts{
						Total:    3,
						ByStatus: []services.StatusColorInfo{{Status: "active", Count: 3, Color: "green", Phase: "development"}},
					},
					Features:    services.SummaryEntityCounts{Total: 5},
					Tasks:       services.SummaryTaskCounts{SummaryEntityCounts: services.SummaryEntityCounts{Total: 20}},
					Bugs:        services.SummaryBugCounts{SummaryEntityCounts: services.SummaryEntityCounts{Total: 4}},
					ChangeCards: services.SummaryEntityCounts{Total: 1},
				}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/summary", newTestMux(mock))

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-001: expected 200, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var body map[string]json.RawMessage
		assertJSON(t, rec, &body)
		for _, key := range []string{"epics", "features", "tasks", "bugs", "change_cards"} {
			if _, ok := body[key]; !ok {
				t.Errorf("TC-H-001: missing key %q in response", key)
			}
		}
	})

	// TC-H-002: Happy path — empty project
	t.Run("TC-H-002_happy_path_empty", func(t *testing.T) {
		mock := &MockViewerServicer{
			SummaryFunc: func(_ context.Context) (*services.SummaryResponse, error) {
				return &services.SummaryResponse{}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/summary", newTestMux(mock))

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-002: expected 200, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var body struct {
			Tasks struct {
				Total int `json:"total"`
			} `json:"tasks"`
			Epics struct {
				Total int `json:"total"`
			} `json:"epics"`
		}
		assertJSON(t, rec, &body)
		if body.Tasks.Total != 0 {
			t.Errorf("TC-H-002: expected tasks.total = 0, got %d", body.Tasks.Total)
		}
		if body.Epics.Total != 0 {
			t.Errorf("TC-H-002: expected epics.total = 0, got %d", body.Epics.Total)
		}
	})

	// TC-H-003: Service error → 500
	t.Run("TC-H-003_service_error_500", func(t *testing.T) {
		mock := &MockViewerServicer{
			SummaryFunc: func(_ context.Context) (*services.SummaryResponse, error) {
				return nil, errors.New("db failure")
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/summary", newTestMux(mock))

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("TC-H-003: expected 500, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var errResp map[string]string
		assertJSON(t, rec, &errResp)
		if errResp["error"] == "" {
			t.Error("TC-H-003: expected non-empty error field in response")
		}
	})
}

// ----- TC-H-010 through TC-H-011: GET /hierarchy -----

func TestHandler_Hierarchy(t *testing.T) {
	// TC-H-010: Happy path — 1 epic, 2 features
	t.Run("TC-H-010_happy_path_one_epic_two_features", func(t *testing.T) {
		epic := &models.Epic{}
		epic.Key = "E01"
		epic.Title = "Test Epic"
		feat1 := &models.Feature{}
		feat1.Key = "E01-F01"
		feat2 := &models.Feature{}
		feat2.Key = "E01-F02"

		mock := &MockViewerServicer{
			HierarchyFunc: func(_ context.Context) (*services.HierarchyResponse, error) {
				return &services.HierarchyResponse{
					Epics: []*services.HierarchyEpic{
						{
							Epic:        epic,
							StatusColor: "green",
							StatusPhase: "active",
							Features: []*services.HierarchyFeature{
								{Feature: feat1, TaskCount: 3, BlockedCount: 0, StatusColor: "yellow"},
								{Feature: feat2, TaskCount: 2, BlockedCount: 1, StatusColor: "red"},
							},
						},
					},
				}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/hierarchy", newTestMux(mock))

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-010: expected 200, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var body struct {
			Epics []struct {
				Features []struct {
					TaskCount    int    `json:"task_count"`
					BlockedCount int    `json:"blocked_count"`
					StatusColor  string `json:"status_color"`
				} `json:"features"`
			} `json:"epics"`
		}
		assertJSON(t, rec, &body)
		if len(body.Epics) != 1 {
			t.Fatalf("TC-H-010: expected 1 epic, got %d", len(body.Epics))
		}
		if len(body.Epics[0].Features) != 2 {
			t.Fatalf("TC-H-010: expected 2 features, got %d", len(body.Epics[0].Features))
		}
		for i, f := range body.Epics[0].Features {
			if f.StatusColor == "" {
				t.Errorf("TC-H-010: features[%d].status_color is empty", i)
			}
		}
	})

	// TC-H-011: Empty project
	t.Run("TC-H-011_empty_project", func(t *testing.T) {
		mock := &MockViewerServicer{
			HierarchyFunc: func(_ context.Context) (*services.HierarchyResponse, error) {
				return &services.HierarchyResponse{Epics: []*services.HierarchyEpic{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/hierarchy", newTestMux(mock))

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-011: expected 200, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var body struct {
			Epics []interface{} `json:"epics"`
		}
		assertJSON(t, rec, &body)
		if len(body.Epics) != 0 {
			t.Errorf("TC-H-011: expected empty epics array, got %d items", len(body.Epics))
		}
	})

	// TC-H-012: Service error → 500
	t.Run("TC-H-012_service_error_500", func(t *testing.T) {
		mock := &MockViewerServicer{
			HierarchyFunc: func(_ context.Context) (*services.HierarchyResponse, error) {
				return nil, fmt.Errorf("db error")
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/hierarchy", newTestMux(mock))

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("TC-H-012: expected 500, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var errResp map[string]string
		assertJSON(t, rec, &errResp)
		if errResp["error"] == "" {
			t.Error("TC-H-012: expected non-empty error field in response")
		}
	})
}

// ----- TC-H-020 through TC-H-027: GET /history/{key} -----

func TestHandler_History(t *testing.T) {
	now := time.Now().UTC()

	happyHistoryFunc := func(key string) func(context.Context, string) (*services.HistoryResponse, error) {
		return func(_ context.Context, k string) (*services.HistoryResponse, error) {
			if k != key {
				return nil, fmt.Errorf("history not found for %s", k)
			}
			t1 := now
			t2 := now.Add(-time.Hour)
			return &services.HistoryResponse{
				EntityType: models.EntityTypeEpic,
				EntityKey:  key,
				Records: []*models.EntityHistory{
					{ToStatus: "active", ChangedAt: t1},
					{ToStatus: "todo", ChangedAt: t2},
				},
			}, nil
		}
	}

	// TC-H-020: Happy path — epic key
	t.Run("TC-H-020_epic_key", func(t *testing.T) {
		mock := &MockViewerServicer{HistoryFunc: happyHistoryFunc("E01")}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/history/E01", newTestMux(mock))

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-020: expected 200, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var body struct {
			EntityKey string `json:"entity_key"`
			Records   []struct {
				ChangedAt time.Time `json:"changed_at"`
			} `json:"records"`
		}
		assertJSON(t, rec, &body)
		if body.EntityKey != "E01" {
			t.Errorf("TC-H-020: expected entity_key=E01, got %q", body.EntityKey)
		}
		if len(body.Records) != 2 {
			t.Errorf("TC-H-020: expected 2 records, got %d", len(body.Records))
		}
		if len(body.Records) == 2 && !body.Records[0].ChangedAt.After(body.Records[1].ChangedAt) {
			t.Error("TC-H-020: first record should be newer than second")
		}
	})

	// TC-H-021: Happy path — feature key
	t.Run("TC-H-021_feature_key", func(t *testing.T) {
		mock := &MockViewerServicer{
			HistoryFunc: func(_ context.Context, key string) (*services.HistoryResponse, error) {
				return &services.HistoryResponse{EntityKey: key, Records: []*models.EntityHistory{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/history/E01-F01", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-021: expected 200, got %d", rec.Code)
		}
	})

	// TC-H-022: Happy path — task key (short format)
	t.Run("TC-H-022_task_key", func(t *testing.T) {
		mock := &MockViewerServicer{
			HistoryFunc: func(_ context.Context, key string) (*services.HistoryResponse, error) {
				return &services.HistoryResponse{EntityKey: key, Records: []*models.EntityHistory{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/history/E01-F01-001", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-022: expected 200, got %d", rec.Code)
		}
	})

	// TC-H-023: Happy path — bug key
	t.Run("TC-H-023_bug_key", func(t *testing.T) {
		mock := &MockViewerServicer{
			HistoryFunc: func(_ context.Context, key string) (*services.HistoryResponse, error) {
				return &services.HistoryResponse{EntityKey: key, Records: []*models.EntityHistory{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/history/B001", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-023: expected 200, got %d", rec.Code)
		}
	})

	// TC-H-024: Happy path — change-card key
	t.Run("TC-H-024_change_card_key", func(t *testing.T) {
		mock := &MockViewerServicer{
			HistoryFunc: func(_ context.Context, key string) (*services.HistoryResponse, error) {
				return &services.HistoryResponse{EntityKey: key, Records: []*models.EntityHistory{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/history/CC-001", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-024: expected 200, got %d", rec.Code)
		}
	})

	// TC-H-025: Invalid key format → 400; HistoryFunc must NOT be called
	t.Run("TC-H-025_invalid_key_400", func(t *testing.T) {
		called := false
		mock := &MockViewerServicer{
			HistoryFunc: func(_ context.Context, _ string) (*services.HistoryResponse, error) {
				called = true
				return nil, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/history/NOT-A-KEY", newTestMux(mock))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("TC-H-025: expected 400, got %d", rec.Code)
		}
		if called {
			t.Error("TC-H-025: HistoryFunc should NOT have been called")
		}
		assertContentTypeJSON(t, rec)
		var errResp map[string]string
		assertJSON(t, rec, &errResp)
		if errResp["error"] == "" {
			t.Error("TC-H-025: expected non-empty error field")
		}
	})

	// TC-H-026: Valid format, not found → 404
	t.Run("TC-H-026_not_found_404", func(t *testing.T) {
		mock := &MockViewerServicer{
			HistoryFunc: func(_ context.Context, key string) (*services.HistoryResponse, error) {
				return nil, fmt.Errorf("entity not found: %s", key)
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/history/E99", newTestMux(mock))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("TC-H-026: expected 404, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)
	})

	// TC-H-027: Lowercase key normalized → valid lookup
	t.Run("TC-H-027_lowercase_normalized", func(t *testing.T) {
		var receivedKey string
		mock := &MockViewerServicer{
			HistoryFunc: func(_ context.Context, key string) (*services.HistoryResponse, error) {
				receivedKey = key
				return &services.HistoryResponse{EntityKey: key, Records: []*models.EntityHistory{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/history/e01-f01-001", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-027: expected 200, got %d; received key=%q", rec.Code, receivedKey)
		}
		if receivedKey != "E01-F01-001" {
			t.Errorf("TC-H-027: expected normalized key E01-F01-001, got %q", receivedKey)
		}
	})

	// TC-H-028: Generic (non-not-found) error → 500
	t.Run("TC-H-028_generic_error_500", func(t *testing.T) {
		mock := &MockViewerServicer{
			HistoryFunc: func(_ context.Context, key string) (*services.HistoryResponse, error) {
				return nil, fmt.Errorf("generic error")
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/history/T-E27-F01-001", newTestMux(mock))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("TC-H-028: expected 500, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var errResp map[string]string
		assertJSON(t, rec, &errResp)
		if errResp["error"] == "" {
			t.Error("TC-H-028: expected non-empty error field in response")
		}
	})
}

// ----- TC-H-030 through TC-H-036: GET /file/{key} -----

func TestHandler_File(t *testing.T) {
	// TC-H-030: Happy path — file exists
	t.Run("TC-H-030_file_exists", func(t *testing.T) {
		mock := &MockViewerServicer{
			FileFunc: func(_ context.Context, key string) (*services.FileResponse, error) {
				return &services.FileResponse{
					Exists:  true,
					Content: "# E01 Epic",
					Path:    "docs/plan/E01/epic.md",
				}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/file/E01", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-030: expected 200, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var body struct {
			Exists  bool   `json:"exists"`
			Content string `json:"content"`
			Path    string `json:"path"`
		}
		assertJSON(t, rec, &body)
		if !body.Exists {
			t.Error("TC-H-030: expected exists=true")
		}
		if body.Content == "" {
			t.Error("TC-H-030: expected non-empty content")
		}
		if body.Path == "" {
			t.Error("TC-H-030: expected non-empty path")
		}
	})

	// TC-H-031: File missing on disk — exists:false, HTTP 200
	t.Run("TC-H-031_file_missing_200", func(t *testing.T) {
		mock := &MockViewerServicer{
			FileFunc: func(_ context.Context, key string) (*services.FileResponse, error) {
				return &services.FileResponse{Exists: false}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/file/E01", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-031: expected 200, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var body struct {
			Exists bool `json:"exists"`
		}
		assertJSON(t, rec, &body)
		if body.Exists {
			t.Error("TC-H-031: expected exists=false")
		}
	})

	// TC-H-032: Security error (path traversal) → 403
	t.Run("TC-H-032_security_error_403", func(t *testing.T) {
		mock := &MockViewerServicer{
			FileFunc: func(_ context.Context, key string) (*services.FileResponse, error) {
				return nil, &services.SecurityError{Path: "../../etc/passwd"}
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/file/E01", newTestMux(mock))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("TC-H-032: expected 403, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)
	})

	// TC-H-033: File too large → 413
	t.Run("TC-H-033_file_too_large_413", func(t *testing.T) {
		mock := &MockViewerServicer{
			FileFunc: func(_ context.Context, key string) (*services.FileResponse, error) {
				return nil, &services.FileTooLargeError{Path: "docs/plan/E01/epic.md", LimitMiB: 2}
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/file/E01", newTestMux(mock))
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("TC-H-033: expected 413, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)
	})

	// TC-H-034: Invalid key format → 400; FileFunc must NOT be called
	t.Run("TC-H-034_invalid_key_400", func(t *testing.T) {
		called := false
		mock := &MockViewerServicer{
			FileFunc: func(_ context.Context, _ string) (*services.FileResponse, error) {
				called = true
				return nil, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/file/bad-key!!", newTestMux(mock))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("TC-H-034: expected 400, got %d", rec.Code)
		}
		if called {
			t.Error("TC-H-034: FileFunc should NOT have been called")
		}
	})

	// TC-H-035: Valid format, not found → 404
	t.Run("TC-H-035_not_found_404", func(t *testing.T) {
		mock := &MockViewerServicer{
			FileFunc: func(_ context.Context, key string) (*services.FileResponse, error) {
				return nil, fmt.Errorf("entity not found: %s", key)
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/file/E99", newTestMux(mock))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("TC-H-035: expected 404, got %d", rec.Code)
		}
	})

	// TC-H-036: T-prefixed task key accepted
	t.Run("TC-H-036_t_prefix_task_key_accepted", func(t *testing.T) {
		mock := &MockViewerServicer{
			FileFunc: func(_ context.Context, key string) (*services.FileResponse, error) {
				return &services.FileResponse{Exists: true, Content: "# task", Path: "docs/task.md"}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/file/T-E01-F01-001", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-036: expected 200, got %d", rec.Code)
		}
	})
}

// ----- TC-H-040 through TC-H-044: GET /features/{key}/tasks -----

func TestHandler_FeatureTasks(t *testing.T) {
	taskWithColor := func(key string) *services.ViewerTask {
		task := &models.Task{}
		task.Key = key
		task.Status = "todo"
		return &services.ViewerTask{Task: task, StatusColor: "gray", StatusPhase: "planning"}
	}

	// TC-H-040: Happy path
	t.Run("TC-H-040_happy_path", func(t *testing.T) {
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, featureKey string, opts services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				tasks := make([]*services.ViewerTask, 5)
				for i := range tasks {
					tasks[i] = taskWithColor(fmt.Sprintf("E01-F01-%03d", i+1))
				}
				return &services.FeatureTasksResponse{
					FeatureKey: featureKey,
					Total:      5,
					Tasks:      tasks,
				}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/features/E01-F01/tasks", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-040: expected 200, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var body struct {
			FeatureKey string        `json:"feature_key"`
			Total      int           `json:"total"`
			Tasks      []interface{} `json:"tasks"`
		}
		assertJSON(t, rec, &body)
		if body.FeatureKey == "" {
			t.Error("TC-H-040: expected non-empty feature_key")
		}
		if body.Total != 5 {
			t.Errorf("TC-H-040: expected total=5, got %d", body.Total)
		}
		if len(body.Tasks) != 5 {
			t.Errorf("TC-H-040: expected 5 tasks, got %d", len(body.Tasks))
		}
	})

	// TC-H-041: limit clamp — limit=9999 clamped to 500
	t.Run("TC-H-041_limit_clamped_to_500", func(t *testing.T) {
		var capturedLimit int
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, featureKey string, opts services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				capturedLimit = opts.Limit
				return &services.FeatureTasksResponse{FeatureKey: featureKey, Total: 0, Tasks: []*services.ViewerTask{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/features/E01-F01/tasks?limit=9999", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-041: expected 200, got %d", rec.Code)
		}
		if capturedLimit != 500 {
			t.Errorf("TC-H-041: expected opts.Limit=500 (clamped), got %d", capturedLimit)
		}
	})

	// TC-H-042: Invalid feature key → 400
	t.Run("TC-H-042_invalid_feature_key_400", func(t *testing.T) {
		called := false
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, _ string, _ services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				called = true
				return nil, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/features/NOTAFEATURE/tasks", newTestMux(mock))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("TC-H-042: expected 400, got %d", rec.Code)
		}
		if called {
			t.Error("TC-H-042: FeatureTasksFunc should NOT have been called")
		}
	})

	// TC-H-043: Valid feature key, not found → 404
	t.Run("TC-H-043_not_found_404", func(t *testing.T) {
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, featureKey string, opts services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				return nil, fmt.Errorf("feature not found: %s", featureKey)
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/features/E99-F99/tasks", newTestMux(mock))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("TC-H-043: expected 404, got %d", rec.Code)
		}
	})

	// TC-H-044: blocked=invalid → 400
	t.Run("TC-H-044_blocked_invalid_400", func(t *testing.T) {
		called := false
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, _ string, _ services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				called = true
				return nil, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/features/E01-F01/tasks?blocked=maybe", newTestMux(mock))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("TC-H-044: expected 400, got %d", rec.Code)
		}
		if called {
			t.Error("TC-H-044: FeatureTasksFunc should NOT have been called")
		}
	})

	// TC-H-045: Service error → 500
	t.Run("TC-H-045_service_error_500", func(t *testing.T) {
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, _ string, _ services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				return nil, fmt.Errorf("db error")
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/features/E01-F01/tasks", newTestMux(mock))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("TC-H-045: expected 500, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var errResp map[string]string
		assertJSON(t, rec, &errResp)
		if errResp["error"] == "" {
			t.Error("TC-H-045: expected non-empty error field in response")
		}
	})

	// TC-H-046: Default limit used when no ?limit= query param
	t.Run("TC-H-046_default_limit_when_no_param", func(t *testing.T) {
		var capturedLimit int
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, featureKey string, opts services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				capturedLimit = opts.Limit
				return &services.FeatureTasksResponse{FeatureKey: featureKey, Total: 0, Tasks: []*services.ViewerTask{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/features/E01-F01/tasks", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-046: expected 200, got %d", rec.Code)
		}
		// Default limit for FeatureTasks is 200 (parseIntClamp default)
		if capturedLimit != 200 {
			t.Errorf("TC-H-046: expected default opts.Limit=200, got %d", capturedLimit)
		}
	})
}

// ----- TC-H-050 through TC-H-055: GET /recent-activity -----

func TestHandler_RecentActivity(t *testing.T) {
	makeActivityResponse := func(n int) *services.RecentActivityResponse {
		records := make([]*services.ActivityRecord, n)
		for i := range records {
			records[i] = &services.ActivityRecord{
				EntityType: "task",
				Key:        fmt.Sprintf("E01-F01-%03d", i+1),
				Title:      fmt.Sprintf("Task %d", i+1),
				ToStatus:   "in_progress",
				ChangedAt:  time.Now().UTC(),
			}
		}
		return &services.RecentActivityResponse{Records: records}
	}

	// TC-H-050: Happy path — no filters
	t.Run("TC-H-050_no_filters", func(t *testing.T) {
		mock := &MockViewerServicer{
			RecentActivityFunc: func(_ context.Context, opts services.RecentActivityOptions) (*services.RecentActivityResponse, error) {
				return makeActivityResponse(10), nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/recent-activity", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-050: expected 200, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var body struct {
			Records []struct {
				EntityType string    `json:"entity_type"`
				Key        string    `json:"key"`
				Title      string    `json:"title"`
				ToStatus   string    `json:"to_status"`
				ChangedAt  time.Time `json:"changed_at"`
			} `json:"records"`
		}
		assertJSON(t, rec, &body)
		if len(body.Records) != 10 {
			t.Errorf("TC-H-050: expected 10 records, got %d", len(body.Records))
		}
	})

	// TC-H-051: entity_type=task filter forwarded to service
	t.Run("TC-H-051_entity_type_filter", func(t *testing.T) {
		var capturedEntityType string
		mock := &MockViewerServicer{
			RecentActivityFunc: func(_ context.Context, opts services.RecentActivityOptions) (*services.RecentActivityResponse, error) {
				capturedEntityType = opts.EntityType
				return &services.RecentActivityResponse{Records: []*services.ActivityRecord{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/recent-activity?entity_type=task", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-051: expected 200, got %d", rec.Code)
		}
		if capturedEntityType != "task" {
			t.Errorf("TC-H-051: expected EntityType=task, got %q", capturedEntityType)
		}
	})

	// TC-H-052: since filter — valid RFC3339 forwarded
	t.Run("TC-H-052_since_filter_valid", func(t *testing.T) {
		var capturedSince *time.Time
		mock := &MockViewerServicer{
			RecentActivityFunc: func(_ context.Context, opts services.RecentActivityOptions) (*services.RecentActivityResponse, error) {
				capturedSince = opts.Since
				return &services.RecentActivityResponse{Records: []*services.ActivityRecord{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/recent-activity?since=2026-01-01T00:00:00Z", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-052: expected 200, got %d", rec.Code)
		}
		if capturedSince == nil {
			t.Error("TC-H-052: expected non-nil opts.Since")
		}
	})

	// TC-H-053: Invalid entity_type → 400; RecentActivityFunc must NOT be called
	t.Run("TC-H-053_invalid_entity_type_400", func(t *testing.T) {
		called := false
		mock := &MockViewerServicer{
			RecentActivityFunc: func(_ context.Context, _ services.RecentActivityOptions) (*services.RecentActivityResponse, error) {
				called = true
				return nil, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/recent-activity?entity_type=blorp", newTestMux(mock))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("TC-H-053: expected 400, got %d", rec.Code)
		}
		if called {
			t.Error("TC-H-053: RecentActivityFunc should NOT have been called")
		}
	})

	// TC-H-054: Malformed since → 400
	t.Run("TC-H-054_malformed_since_400", func(t *testing.T) {
		called := false
		mock := &MockViewerServicer{
			RecentActivityFunc: func(_ context.Context, _ services.RecentActivityOptions) (*services.RecentActivityResponse, error) {
				called = true
				return nil, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/recent-activity?since=not-a-date", newTestMux(mock))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("TC-H-054: expected 400, got %d", rec.Code)
		}
		if called {
			t.Error("TC-H-054: RecentActivityFunc should NOT have been called")
		}
	})

	// TC-H-055: limit clamp — limit=9999 silently clamped to 200
	t.Run("TC-H-055_limit_clamped_to_200", func(t *testing.T) {
		var capturedLimit int
		mock := &MockViewerServicer{
			RecentActivityFunc: func(_ context.Context, opts services.RecentActivityOptions) (*services.RecentActivityResponse, error) {
				capturedLimit = opts.Limit
				return &services.RecentActivityResponse{Records: []*services.ActivityRecord{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/recent-activity?limit=9999", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-055: expected 200, got %d", rec.Code)
		}
		if capturedLimit != 200 {
			t.Errorf("TC-H-055: expected opts.Limit=200 (clamped), got %d", capturedLimit)
		}
	})
}

// ----- TC-H-060 through TC-H-061: GET /workflow-meta -----

func TestHandler_WorkflowMeta(t *testing.T) {
	// TC-H-060: Happy path — short workflow with 5 levels
	t.Run("TC-H-060_happy_path_5_levels", func(t *testing.T) {
		levels := map[string]*services.WorkflowLevelMeta{
			"epic":        {Level: "epic", Statuses: []services.WorkflowStatusMeta{{Name: "active", Color: "green"}}, Transitions: []services.WorkflowTransitionMeta{}},
			"feature":     {Level: "feature", Statuses: []services.WorkflowStatusMeta{{Name: "active", Color: "green"}}, Transitions: []services.WorkflowTransitionMeta{}},
			"task":        {Level: "task", Statuses: []services.WorkflowStatusMeta{{Name: "todo", Color: "gray"}}, Transitions: []services.WorkflowTransitionMeta{}},
			"bug":         {Level: "bug", Statuses: []services.WorkflowStatusMeta{{Name: "reported", Color: "red"}}, Transitions: []services.WorkflowTransitionMeta{}},
			"change_card": {Level: "change_card", Statuses: []services.WorkflowStatusMeta{{Name: "proposed", Color: "blue"}}, Transitions: []services.WorkflowTransitionMeta{}},
		}
		mock := &MockViewerServicer{
			WorkflowMetaFunc: func(_ context.Context) (*services.WorkflowMetaResponse, error) {
				return &services.WorkflowMetaResponse{Levels: levels}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/workflow-meta", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-060: expected 200, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var body struct {
			Levels map[string]struct {
				Statuses    []interface{} `json:"statuses"`
				Transitions []interface{} `json:"transitions"`
			} `json:"levels"`
		}
		assertJSON(t, rec, &body)
		for _, levelName := range []string{"epic", "feature", "task", "bug", "change_card"} {
			if _, ok := body.Levels[levelName]; !ok {
				t.Errorf("TC-H-060: missing level %q in response", levelName)
			}
		}
	})

	// TC-H-061: Missing level emitted as empty object (bug level with empty statuses)
	t.Run("TC-H-061_missing_level_as_empty_object", func(t *testing.T) {
		levels := map[string]*services.WorkflowLevelMeta{
			"bug": {Level: "bug", Statuses: []services.WorkflowStatusMeta{}, Transitions: []services.WorkflowTransitionMeta{}},
		}
		mock := &MockViewerServicer{
			WorkflowMetaFunc: func(_ context.Context) (*services.WorkflowMetaResponse, error) {
				return &services.WorkflowMetaResponse{Levels: levels}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/workflow-meta", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-061: expected 200, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var body struct {
			Levels map[string]json.RawMessage `json:"levels"`
		}
		assertJSON(t, rec, &body)
		if _, ok := body.Levels["bug"]; !ok {
			t.Error("TC-H-061: expected levels.bug to be present even with empty statuses/transitions")
		}
	})

	// TC-H-062: Service error → 500
	t.Run("TC-H-062_service_error_500", func(t *testing.T) {
		mock := &MockViewerServicer{
			WorkflowMetaFunc: func(_ context.Context) (*services.WorkflowMetaResponse, error) {
				return nil, fmt.Errorf("db error")
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/workflow-meta", newTestMux(mock))

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("TC-H-062: expected 500, got %d", rec.Code)
		}
		assertContentTypeJSON(t, rec)

		var errResp map[string]string
		assertJSON(t, rec, &errResp)
		if errResp["error"] == "" {
			t.Error("TC-H-062: expected non-empty error field in response")
		}
	})
}

// ----- TC-H-070 through TC-H-074: CORS middleware -----

func TestCORSMiddleware(t *testing.T) {
	// Inner handler that just returns 200 with a marker body.
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// TC-H-070: localhost origin → CORS headers echoed
	t.Run("TC-H-070_localhost_cors_echoed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		rec := httptest.NewRecorder()

		WithLocalCORS(innerHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-070: expected 200, got %d", rec.Code)
		}
		acao := rec.Header().Get("Access-Control-Allow-Origin")
		if acao != "http://localhost:5173" {
			t.Errorf("TC-H-070: expected ACAO=http://localhost:5173, got %q", acao)
		}
		vary := rec.Header().Get("Vary")
		if vary == "" {
			t.Error("TC-H-070: expected Vary header to be set")
		}
	})

	// TC-H-071: 127.0.0.1 origin → CORS headers echoed
	t.Run("TC-H-071_127001_cors_echoed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://127.0.0.1:3000")
		rec := httptest.NewRecorder()

		WithLocalCORS(innerHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-071: expected 200, got %d", rec.Code)
		}
		acao := rec.Header().Get("Access-Control-Allow-Origin")
		if acao != "http://127.0.0.1:3000" {
			t.Errorf("TC-H-071: expected ACAO=http://127.0.0.1:3000, got %q", acao)
		}
	})

	// TC-H-072: Non-local origin → no CORS header
	t.Run("TC-H-072_external_origin_no_cors", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://evil.example.com")
		rec := httptest.NewRecorder()

		WithLocalCORS(innerHandler).ServeHTTP(rec, req)

		acao := rec.Header().Get("Access-Control-Allow-Origin")
		if acao != "" {
			t.Errorf("TC-H-072: expected no ACAO header for external origin, got %q", acao)
		}
		// Inner handler still runs (non-local does NOT block — browser does)
		if rec.Code != http.StatusOK {
			t.Errorf("TC-H-072: expected inner handler to still run (200), got %d", rec.Code)
		}
	})

	// TC-H-073: OPTIONS preflight → 204, Access-Control-Allow-Methods set, no body
	t.Run("TC-H-073_options_preflight_204", func(t *testing.T) {
		innerCalled := false
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			innerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/viewer/summary", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		rec := httptest.NewRecorder()

		WithLocalCORS(inner).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("TC-H-073: expected 204, got %d", rec.Code)
		}
		if innerCalled {
			t.Error("TC-H-073: inner handler should NOT have been called for OPTIONS preflight")
		}
		methods := rec.Header().Get("Access-Control-Allow-Methods")
		if methods == "" {
			t.Error("TC-H-073: expected Access-Control-Allow-Methods header")
		}
		if rec.Body.Len() != 0 {
			t.Errorf("TC-H-073: expected empty body, got %q", rec.Body.String())
		}
	})

	// TC-H-074: No Origin header → no CORS header, request proceeds normally
	t.Run("TC-H-074_no_origin_no_cors", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		// No Origin header set
		rec := httptest.NewRecorder()

		WithLocalCORS(innerHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-074: expected 200, got %d", rec.Code)
		}
		acao := rec.Header().Get("Access-Control-Allow-Origin")
		if acao != "" {
			t.Errorf("TC-H-074: expected no ACAO header when no Origin, got %q", acao)
		}
	})
}

// ----- Notes handler tests (TC-F020-*) -----

// TC-F020-3: Only the five specified NoteDTO fields are serialised (AC-020.3).
// Verifies that metadata and updated_at are not present in the JSON output.
func TestViewerHandler_Notes_DTOFieldsOnly(t *testing.T) {
	mock := &MockViewerServicer{
		NotesFunc: func(ctx context.Context, key string) (*services.NotesResponse, error) {
			return &services.NotesResponse{
				EntityType: "epic",
				EntityKey:  "E27",
				Notes: []services.NoteDTO{
					{
						ID:        42,
						NoteType:  "decision",
						Content:   "Use DESC ordering",
						CreatedBy: "agent",
						CreatedAt: "2026-01-15T10:00:00Z",
					},
				},
			}, nil
		},
	}
	mux := newTestMux(mock)

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/notes/E27", mux)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()

	// Required fields must be present.
	requiredFields := []string{`"id"`, `"note_type"`, `"content"`, `"created_by"`, `"created_at"`}
	for _, field := range requiredFields {
		if !strings.Contains(body, field) {
			t.Errorf("NoteDTO JSON missing required field %s; body: %s", field, body)
		}
	}

	// Forbidden fields must be absent (AC-020.3).
	forbiddenFields := []string{`"metadata"`, `"updated_at"`}
	for _, field := range forbiddenFields {
		if strings.Contains(body, field) {
			t.Errorf("NoteDTO JSON contains forbidden field %s (AC-020.3); body: %s", field, body)
		}
	}
}

// TC-F020-1: Response JSON shape matches contract (empty notes = [])
func TestViewerHandler_Notes_ResponseShape(t *testing.T) {
	mock := &MockViewerServicer{
		NotesFunc: func(ctx context.Context, key string) (*services.NotesResponse, error) {
			return &services.NotesResponse{
				EntityType: "feature",
				EntityKey:  "E27-F09",
				Notes:      []services.NoteDTO{},
			}, nil
		},
	}
	mux := newTestMux(mock)

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/notes/E27-F09", mux)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var resp services.NotesResponse
	assertJSON(t, rec, &resp)
	if resp.Notes == nil {
		t.Error("notes should be [] not null")
	}
	if len(resp.Notes) != 0 {
		t.Errorf("expected empty notes, got %d", len(resp.Notes))
	}
	if resp.EntityKey != "E27-F09" {
		t.Errorf("expected entity_key=E27-F09, got %q", resp.EntityKey)
	}
}

// TC-F020-1 (with notes): response includes note data
func TestViewerHandler_Notes_WithNotes(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	mock := &MockViewerServicer{
		NotesFunc: func(ctx context.Context, key string) (*services.NotesResponse, error) {
			return &services.NotesResponse{
				EntityType: "feature",
				EntityKey:  key,
				Notes: []services.NoteDTO{
					{ID: 1, NoteType: "comment", Content: "hello", CreatedAt: now.Format(time.RFC3339)},
					{ID: 2, NoteType: "decision", Content: "world", CreatedAt: now.Format(time.RFC3339)},
				},
			}, nil
		},
	}
	mux := newTestMux(mock)

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/notes/E27-F09", mux)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp services.NotesResponse
	assertJSON(t, rec, &resp)
	if len(resp.Notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(resp.Notes))
	}
}

// TC-F020-5: 404 returned for unknown entity
func TestViewerHandler_Notes_NotFound(t *testing.T) {
	mock := &MockViewerServicer{
		NotesFunc: func(ctx context.Context, key string) (*services.NotesResponse, error) {
			return nil, fmt.Errorf("entity not found: %s", key)
		},
	}
	mux := newTestMux(mock)

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/notes/E27-F09", mux)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// TC-F020-6: 400 returned for malformed key
func TestViewerHandler_Notes_BadKey(t *testing.T) {
	mock := &MockViewerServicer{}
	mux := newTestMux(mock)

	// Path traversal attempt
	rec := makeRequest(http.MethodGet, "/api/v1/viewer/notes/..%2F..%2Fetc%2Fpasswd", mux)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for path traversal, got %d", rec.Code)
	}
}

// TC-F020-4: Handler accepts key shapes (short task, long task, feature, epic, lower/upper)
func TestViewerHandler_Notes_KeyShapes(t *testing.T) {
	cases := []struct {
		path string
		want int
	}{
		{"/api/v1/viewer/notes/E27-F09-002", http.StatusOK},
		{"/api/v1/viewer/notes/T-E27-F09-002", http.StatusOK},
		{"/api/v1/viewer/notes/E27-F09", http.StatusOK},
		{"/api/v1/viewer/notes/E27", http.StatusOK},
		{"/api/v1/viewer/notes/e27-f09", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			mock := &MockViewerServicer{
				NotesFunc: func(ctx context.Context, key string) (*services.NotesResponse, error) {
					return &services.NotesResponse{Notes: []services.NoteDTO{}}, nil
				},
			}
			mux := newTestMux(mock)
			rec := makeRequest(http.MethodGet, tc.path, mux)
			if rec.Code != tc.want {
				t.Errorf("path=%q: expected %d, got %d; body: %s", tc.path, tc.want, rec.Code, rec.Body.String())
			}
		})
	}
}

// TC-F020-7: CORS behaviour identical to existing endpoints (localhost allowed)
func TestViewerHandler_Notes_CORSBehavior(t *testing.T) {
	mock := &MockViewerServicer{
		NotesFunc: func(ctx context.Context, key string) (*services.NotesResponse, error) {
			return &services.NotesResponse{Notes: []services.NoteDTO{}}, nil
		},
	}
	mux := newTestMux(mock)

	// Localhost origin should be echoed.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/viewer/notes/E27-F09", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("expected ACAO=http://localhost:5173, got %q", got)
	}

	// Non-local origin should not receive CORS header.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/viewer/notes/E27-F09", nil)
	req2.Header.Set("Origin", "https://evil.example.com")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if got := rec2.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO for evil origin, got %q", got)
	}
}

// ----- RelatedDocs handler tests (TC-F021-*) -----

// TC-F021-1: Empty list returns {"docs":[]} not null
func TestViewerHandler_RelatedDocs_EmptyResponse(t *testing.T) {
	mock := &MockViewerServicer{
		RelatedDocsFunc: func(ctx context.Context, key string) (*services.RelatedDocsResponse, error) {
			return &services.RelatedDocsResponse{
				EntityType: "feature",
				EntityKey:  "E27-F09",
				Docs:       []services.RelatedDocDTO{},
			}, nil
		},
	}
	mux := newTestMux(mock)

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/related-docs/E27-F09", mux)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var resp services.RelatedDocsResponse
	assertJSON(t, rec, &resp)
	if resp.Docs == nil {
		t.Error("docs should be [] not null")
	}
	if len(resp.Docs) != 0 {
		t.Errorf("expected empty docs, got %d", len(resp.Docs))
	}
}

// TC-F021-2/3: Documents with paths returned as stored
func TestViewerHandler_RelatedDocs_WithDocs(t *testing.T) {
	mock := &MockViewerServicer{
		RelatedDocsFunc: func(ctx context.Context, key string) (*services.RelatedDocsResponse, error) {
			return &services.RelatedDocsResponse{
				EntityType: "feature",
				EntityKey:  key,
				Docs: []services.RelatedDocDTO{
					{ID: 10, Title: "Spec", FilePath: "docs/plan/E27-F09/spec.md"},
					{ID: 5, Title: "Design", FilePath: "docs/plan/E27-F09/design.md"},
				},
			}, nil
		},
	}
	mux := newTestMux(mock)

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/related-docs/E27-F09", mux)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp services.RelatedDocsResponse
	assertJSON(t, rec, &resp)
	if len(resp.Docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(resp.Docs))
	}
	if resp.Docs[0].ID != 10 {
		t.Errorf("expected first doc id=10, got %d", resp.Docs[0].ID)
	}
	if resp.Docs[0].FilePath != "docs/plan/E27-F09/spec.md" {
		t.Errorf("expected path stored verbatim, got %q", resp.Docs[0].FilePath)
	}
}

// TC-F021-4: Key normalisation matches REQ-F-020 (table-driven)
func TestViewerHandler_RelatedDocs_KeyShapes(t *testing.T) {
	cases := []string{
		"/api/v1/viewer/related-docs/E27-F09-002",
		"/api/v1/viewer/related-docs/T-E27-F09-002",
		"/api/v1/viewer/related-docs/E27-F09",
		"/api/v1/viewer/related-docs/E27",
		"/api/v1/viewer/related-docs/e27-f09",
	}

	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			mock := &MockViewerServicer{
				RelatedDocsFunc: func(ctx context.Context, key string) (*services.RelatedDocsResponse, error) {
					return &services.RelatedDocsResponse{Docs: []services.RelatedDocDTO{}}, nil
				},
			}
			mux := newTestMux(mock)
			rec := makeRequest(http.MethodGet, path, mux)
			if rec.Code != http.StatusOK {
				t.Errorf("path=%q: expected 200, got %d; body: %s", path, rec.Code, rec.Body.String())
			}
		})
	}
}

// TC-F021-4 (CORS): Related-docs endpoint CORS matches notes endpoint
func TestViewerHandler_RelatedDocs_CORSBehavior(t *testing.T) {
	mock := &MockViewerServicer{
		RelatedDocsFunc: func(ctx context.Context, key string) (*services.RelatedDocsResponse, error) {
			return &services.RelatedDocsResponse{Docs: []services.RelatedDocDTO{}}, nil
		},
	}
	mux := newTestMux(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/viewer/related-docs/E27-F09", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("expected ACAO=http://localhost:3000, got %q", got)
	}
}

// RelatedDocs: 404 for unknown entity
func TestViewerHandler_RelatedDocs_NotFound(t *testing.T) {
	mock := &MockViewerServicer{
		RelatedDocsFunc: func(ctx context.Context, key string) (*services.RelatedDocsResponse, error) {
			return nil, fmt.Errorf("entity not found: %s", key)
		},
	}
	mux := newTestMux(mock)

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/related-docs/E27-F09", mux)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
