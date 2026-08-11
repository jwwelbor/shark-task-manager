package viewer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	sprint "github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// ----- MockViewerServicer -----

// MockViewerServicer is a test double for ViewerServicer.
// Each method delegates to its Func field if non-nil; otherwise it returns a
// descriptive error to prevent accidental nil-pointer panics in tests.
//
// F06 changes (T-E28-F06-002, ADR-F06-5):
//   - HierarchyFunc signature changed from Hierarchy(ctx) to Hierarchy(ctx, opts HierarchyOptions)
//   - TagsFunc added for the new Tags() method
type MockViewerServicer struct {
	SummaryFunc func(ctx context.Context) (*services.SummaryResponse, error)
	// HierarchyFunc accepts opts; existing tests pass HierarchyOptions{} (zero value = no filter).
	HierarchyFunc      func(ctx context.Context, opts services.HierarchyOptions) (*services.HierarchyResponse, error)
	HistoryFunc        func(ctx context.Context, key string) (*services.HistoryResponse, error)
	FileFunc           func(ctx context.Context, key string) (*services.FileResponse, error)
	FileByPathFunc     func(ctx context.Context, filePath string) (*services.FileResponse, error)
	FolderFilesFunc    func(ctx context.Context, dirPath string) (*services.FolderFilesResponse, error)
	FeatureTasksFunc   func(ctx context.Context, featureKey string, opts services.FeatureTaskOptions) (*services.FeatureTasksResponse, error)
	RecentActivityFunc func(ctx context.Context, opts services.RecentActivityOptions) (*services.RecentActivityResponse, error)
	WorkflowMetaFunc   func(ctx context.Context) (*services.WorkflowMetaResponse, error)
	NavFoldersFunc     func(ctx context.Context) (*services.NavFoldersResponse, error)
	NotesFunc          func(ctx context.Context, key string) (*services.NotesResponse, error)
	RelatedDocsFunc    func(ctx context.Context, key string) (*services.RelatedDocsResponse, error)
	TagsFunc           func(ctx context.Context) (*services.TagsResponse, error) // NEW F06
	SprintOverviewFunc func(ctx context.Context, key string) (*services.SprintOverviewResponse, error)
	SprintPlanFunc     func(ctx context.Context, key string) (*services.SprintPlanView, error)
	SprintReportFunc   func(ctx context.Context, key string) (*services.SprintReportResponse, error)

	// TagsCallCount counts invocations of Tags() so security regression tests can
	// assert that non-GET methods on /api/v1/viewer/tags never reach the service
	// (TC-AC03-5, AC-T2). Read directly in tests; the mock is single-goroutine.
	TagsCallCount int
}

func (m *MockViewerServicer) Summary(ctx context.Context) (*services.SummaryResponse, error) {
	if m.SummaryFunc != nil {
		return m.SummaryFunc(ctx)
	}
	return nil, errors.New("SummaryFunc not set in mock")
}

func (m *MockViewerServicer) Hierarchy(ctx context.Context, opts services.HierarchyOptions) (*services.HierarchyResponse, error) {
	if m.HierarchyFunc != nil {
		return m.HierarchyFunc(ctx, opts)
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

func (m *MockViewerServicer) NavFolders(ctx context.Context) (*services.NavFoldersResponse, error) {
	if m.NavFoldersFunc != nil {
		return m.NavFoldersFunc(ctx)
	}
	return nil, errors.New("NavFoldersFunc not set in mock")
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

func (m *MockViewerServicer) Tags(ctx context.Context) (*services.TagsResponse, error) {
	m.TagsCallCount++
	if m.TagsFunc != nil {
		return m.TagsFunc(ctx)
	}
	return nil, errors.New("TagsFunc not set in mock")
}

func (m *MockViewerServicer) SprintOverview(ctx context.Context, key string) (*services.SprintOverviewResponse, error) {
	if m.SprintOverviewFunc != nil {
		return m.SprintOverviewFunc(ctx, key)
	}
	return nil, errors.New("SprintOverviewFunc not set in mock")
}

func (m *MockViewerServicer) SprintPlan(ctx context.Context, key string) (*services.SprintPlanView, error) {
	if m.SprintPlanFunc != nil {
		return m.SprintPlanFunc(ctx, key)
	}
	return nil, errors.New("SprintPlanFunc not set in mock")
}

func (m *MockViewerServicer) SprintReport(ctx context.Context, key string) (*services.SprintReportResponse, error) {
	if m.SprintReportFunc != nil {
		return m.SprintReportFunc(ctx, key)
	}
	return nil, errors.New("SprintReportFunc not set in mock")
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
					TechDebts:   &services.SummaryEntityCounts{Total: 2},
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
		for _, key := range []string{"epics", "features", "tasks", "bugs", "change_cards", "tech_debts"} {
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
			HierarchyFunc: func(_ context.Context, _ services.HierarchyOptions) (*services.HierarchyResponse, error) {
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
			HierarchyFunc: func(_ context.Context, _ services.HierarchyOptions) (*services.HierarchyResponse, error) {
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
			HierarchyFunc: func(_ context.Context, _ services.HierarchyOptions) (*services.HierarchyResponse, error) {
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

// ----- T-E28-F06-005: tag filter on GET /hierarchy -----
//
// These tests cover the UAT-rejected gap on T-E28-F06-005: the handler-side
// tag-parsing and unregistered-tag error path on /api/v1/viewer/hierarchy.
//
// Spec: REQ-F-009, REQ-F-011, REQ-NF-005, AC-08
// Test plan: TC-AC08-1, TC-AC08-2, TC-AC08-3, TC-AC08-4; IS-3
// Task ACs: AC-T1, AC-T2, AC-T3, AC-T5
//
// The mock captures the HierarchyOptions it received via a closure variable so
// each test can assert exactly what the handler forwarded. parseTagsQuery()
// in handler.go is exercised end-to-end via the URL — no direct unit-test
// shortcut, matching the test-plan's "handler layer" requirement.
func TestHandler_Hierarchy_Tags(t *testing.T) {
	// Helper: returns a mock that captures opts in *captured and returns an
	// empty hierarchy on success. Lets each subtest focus on the assertion.
	makeCaptureMock := func(captured *services.HierarchyOptions) *MockViewerServicer {
		return &MockViewerServicer{
			HierarchyFunc: func(_ context.Context, opts services.HierarchyOptions) (*services.HierarchyResponse, error) {
				*captured = opts
				return &services.HierarchyResponse{Epics: []*services.HierarchyEpic{}}, nil
			},
		}
	}

	// TC-AC08-1 / AC-T1: ?tag=voice → service receives Tags=["voice"].
	// Handler must NOT lowercase or otherwise normalize — that is REQ-NF-005's
	// service-side responsibility.
	t.Run("TC-AC08-1_single_tag_forwarded", func(t *testing.T) {
		var got services.HierarchyOptions
		mock := makeCaptureMock(&got)
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/hierarchy?tag=voice", newTestMux(mock))

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-AC08-1: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if len(got.Tags) != 1 || got.Tags[0] != "voice" {
			t.Errorf("TC-AC08-1: expected opts.Tags=[\"voice\"], got %#v", got.Tags)
		}
	})

	// TC-AC08-2 / AC-T2 / IS-3: blanks-and-whitespace handling.
	// `?tag=&tag=voice&tag=%20%20` — empty + valid + whitespace-only.
	// Per REQ-F-009/REQ-NF-005: handler trims and drops empty-after-trim, but
	// does NOT case-normalize. So the service should receive exactly ["voice"].
	t.Run("TC-AC08-2_blanks_and_whitespace_dropped", func(t *testing.T) {
		var got services.HierarchyOptions
		mock := makeCaptureMock(&got)
		// %20%20 is two spaces — they must be trimmed away to "" and dropped.
		rec := makeRequest(
			http.MethodGet,
			"/api/v1/viewer/hierarchy?tag=&tag=voice&tag=%20%20",
			newTestMux(mock),
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-AC08-2: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if len(got.Tags) != 1 || got.Tags[0] != "voice" {
			t.Errorf("TC-AC08-2: expected opts.Tags=[\"voice\"] after blanks dropped, got %#v", got.Tags)
		}
	})

	// IS-3 (focused): tab-only and multi-space tag values are also dropped.
	// Adds explicit coverage for the whitespace-trim contract of parseTagsQuery
	// beyond the simple two-space case.
	t.Run("IS-3_tab_and_mixed_whitespace_dropped", func(t *testing.T) {
		var got services.HierarchyOptions
		mock := makeCaptureMock(&got)
		// %09 = tab, %20%20%20 = three spaces. Both must be dropped.
		rec := makeRequest(
			http.MethodGet,
			"/api/v1/viewer/hierarchy?tag=%09&tag=%20%20%20",
			newTestMux(mock),
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("IS-3: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if len(got.Tags) != 0 {
			t.Errorf("IS-3: expected opts.Tags=[] after whitespace-only inputs, got %#v", got.Tags)
		}
	})

	// TC-AC08-3 / AC-T3 (positive): unregistered tag → 400 with
	// `unregistered_tags` array body.
	// Note: the spec example AC-08 shows the message coming from
	// UnregisteredTagError.Error() (e.g. "tag is not registered: ..."). We
	// assert the structural contract: 400 + non-empty error/message + the
	// offending name appears in unregistered_tags. We deliberately do NOT
	// hard-code the exact message string so a future Error() format tweak
	// does not break this test.
	t.Run("TC-AC08-3_unregistered_tag_returns_400", func(t *testing.T) {
		mock := &MockViewerServicer{
			HierarchyFunc: func(_ context.Context, opts services.HierarchyOptions) (*services.HierarchyResponse, error) {
				// Sanity: handler did forward the tag.
				if len(opts.Tags) != 1 || opts.Tags[0] != "does-not-exist" {
					t.Errorf("TC-AC08-3: expected opts.Tags=[\"does-not-exist\"], got %#v", opts.Tags)
				}
				return nil, &services.UnregisteredTagError{Name: "does-not-exist"}
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/hierarchy?tag=does-not-exist", newTestMux(mock))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("TC-AC08-3: expected 400, got %d; body: %s", rec.Code, rec.Body.String())
		}
		assertContentTypeJSON(t, rec)

		// Body must include `unregistered_tags` containing "does-not-exist".
		var body struct {
			Error            string   `json:"error"`
			Message          string   `json:"message"`
			UnregisteredTags []string `json:"unregistered_tags"`
		}
		assertJSON(t, rec, &body)
		if body.Error == "" {
			t.Error("TC-AC08-3: expected non-empty error field")
		}
		if body.Message == "" {
			t.Error("TC-AC08-3: expected non-empty message field")
		}
		if len(body.UnregisteredTags) != 1 || body.UnregisteredTags[0] != "does-not-exist" {
			t.Errorf("TC-AC08-3: expected unregistered_tags=[\"does-not-exist\"], got %#v", body.UnregisteredTags)
		}
	})

	// TC-AC08-3 (errors.As wrapping): the handler uses errors.As, so a wrapped
	// UnregisteredTagError must still produce 400 (not 500). Guards against a
	// future refactor that swaps errors.As for a type-assertion or `==` check.
	t.Run("TC-AC08-3b_wrapped_unregistered_tag_still_400", func(t *testing.T) {
		mock := &MockViewerServicer{
			HierarchyFunc: func(_ context.Context, _ services.HierarchyOptions) (*services.HierarchyResponse, error) {
				return nil, fmt.Errorf("hierarchy failed: %w", &services.UnregisteredTagError{Name: "ghost"})
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/hierarchy?tag=ghost", newTestMux(mock))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("TC-AC08-3b: expected 400 from wrapped UnregisteredTagError, got %d; body: %s",
				rec.Code, rec.Body.String())
		}
		var body struct {
			UnregisteredTags []string `json:"unregistered_tags"`
		}
		assertJSON(t, rec, &body)
		if len(body.UnregisteredTags) != 1 || body.UnregisteredTags[0] != "ghost" {
			t.Errorf("TC-AC08-3b: expected unregistered_tags=[\"ghost\"], got %#v", body.UnregisteredTags)
		}
	})

	// TC-AC08-4 / AC-T3 (regression): generic (non-tag) errors must still go
	// to 500. This guards the existing TC-H-012 path while the tag branch was
	// added; ensures the new errors.As check did not accidentally swallow
	// other errors into 400.
	t.Run("TC-AC08-4_generic_error_still_500", func(t *testing.T) {
		mock := &MockViewerServicer{
			HierarchyFunc: func(_ context.Context, _ services.HierarchyOptions) (*services.HierarchyResponse, error) {
				return nil, fmt.Errorf("db connection lost")
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/hierarchy?tag=voice", newTestMux(mock))

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("TC-AC08-4: expected 500 for generic error, got %d; body: %s", rec.Code, rec.Body.String())
		}
	})

	// AC-T5: HierarchyFunc signature must accept HierarchyOptions. This is
	// already enforced at compile time by the mock definition — we add a
	// no-tag request to confirm the zero-value HierarchyOptions{} path still
	// works (existing TC-H-010 behavior must not regress).
	t.Run("AC-T5_zero_options_path_unchanged", func(t *testing.T) {
		var got services.HierarchyOptions
		mock := makeCaptureMock(&got)
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/hierarchy", newTestMux(mock))

		if rec.Code != http.StatusOK {
			t.Fatalf("AC-T5: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if len(got.Tags) != 0 {
			t.Errorf("AC-T5: expected opts.Tags to be empty (no ?tag=), got %#v", got.Tags)
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

	// TC-H-037: Bug key accepted and normalized.
	t.Run("TC-H-037_bug_key_accepted", func(t *testing.T) {
		var receivedKey string
		mock := &MockViewerServicer{
			FileFunc: func(_ context.Context, key string) (*services.FileResponse, error) {
				receivedKey = key
				return &services.FileResponse{Exists: true, Content: "# bug", Path: "docs/bug.md"}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/file/B001", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-037: expected 200, got %d", rec.Code)
		}
		if receivedKey != "B001" {
			t.Errorf("TC-H-037: expected normalized key B001, got %q", receivedKey)
		}
	})

	// TC-H-038: Tech-debt key accepted and normalized.
	t.Run("TC-H-038_tech_debt_key_accepted", func(t *testing.T) {
		var receivedKey string
		mock := &MockViewerServicer{
			FileFunc: func(_ context.Context, key string) (*services.FileResponse, error) {
				receivedKey = key
				return &services.FileResponse{Exists: true, Content: "# tech debt", Path: "docs/tech-debt.md"}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/file/td-001", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-038: expected 200, got %d", rec.Code)
		}
		if receivedKey != "TD-001" {
			t.Errorf("TC-H-038: expected normalized key TD-001, got %q", receivedKey)
		}
	})
}

// ----- TC-H-039: GET /folder-files/{path...} -----

func TestHandler_FolderFiles(t *testing.T) {
	t.Run("TC-H-039_docs_root", func(t *testing.T) {
		var receivedPath string
		mock := &MockViewerServicer{
			FolderFilesFunc: func(_ context.Context, dirPath string) (*services.FolderFilesResponse, error) {
				receivedPath = dirPath
				return &services.FolderFilesResponse{
					DirPath: "docs",
					Entries: []*services.FolderFileEntry{
						{Name: "README.md", Path: "docs/README.md", IsDir: false, Size: 10},
						{Name: "architecture", Path: "docs/architecture", IsDir: true, Size: 0},
					},
				}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/folder-files/docs", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-039: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if receivedPath != "docs" {
			t.Errorf("TC-H-039: expected dirPath=docs, got %q", receivedPath)
		}

		var body struct {
			DirPath string                      `json:"dir_path"`
			Entries []*services.FolderFileEntry `json:"entries"`
		}
		assertJSON(t, rec, &body)
		if body.DirPath != "docs" {
			t.Errorf("TC-H-039: expected dir_path=docs, got %q", body.DirPath)
		}
		if len(body.Entries) != 2 {
			t.Fatalf("TC-H-039: expected 2 entries, got %d", len(body.Entries))
		}
	})

	t.Run("TC-H-039_nested_docs_path", func(t *testing.T) {
		var receivedPath string
		mock := &MockViewerServicer{
			FolderFilesFunc: func(_ context.Context, dirPath string) (*services.FolderFilesResponse, error) {
				receivedPath = dirPath
				return &services.FolderFilesResponse{DirPath: dirPath, Entries: []*services.FolderFileEntry{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/folder-files/docs/architecture", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-039 nested: expected 200, got %d", rec.Code)
		}
		if receivedPath != "docs/architecture" {
			t.Errorf("TC-H-039 nested: expected dirPath=docs/architecture, got %q", receivedPath)
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

// ----- T-E28-F06-005: tag filter on GET /features/{key}/tasks -----
//
// These tests cover the UAT-rejected gap on T-E28-F06-005: the handler-side
// tag-parsing, unregistered-tag error path, and 400-vs-404 ordering on
// /api/v1/viewer/features/{key}/tasks.
//
// Spec: REQ-F-009, REQ-F-012, REQ-NF-005, AC-09, AC-10
// Test plan: TC-AC10-1, TC-AC10-2, TC-AC10-3, TC-AC10-4
// Task ACs: AC-T1, AC-T4
//
// The mock captures opts via a closure variable so each test can assert
// exactly what the handler forwarded.
func TestHandler_FeatureTasks_Tags(t *testing.T) {
	// TC-AC10-4 / AC-T1: ?tag=voice happy path → 200, opts.Tags=["voice"].
	t.Run("TC-AC10-4_single_tag_forwarded", func(t *testing.T) {
		var got services.FeatureTaskOptions
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, featureKey string, opts services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				got = opts
				return &services.FeatureTasksResponse{
					FeatureKey: featureKey,
					Total:      0,
					Tasks:      []*services.ViewerTask{},
				}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/features/E01-F01/tasks?tag=voice", newTestMux(mock))

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-AC10-4: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if len(got.Tags) != 1 || got.Tags[0] != "voice" {
			t.Errorf("TC-AC10-4: expected opts.Tags=[\"voice\"], got %#v", got.Tags)
		}
	})

	// TC-AC10-4 (multi-tag + whitespace, IS-3): repeated/blank/whitespace
	// values must be trimmed and de-empty before reaching the service.
	t.Run("TC-AC10-4b_multi_tag_blanks_dropped", func(t *testing.T) {
		var got services.FeatureTaskOptions
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, featureKey string, opts services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				got = opts
				return &services.FeatureTasksResponse{
					FeatureKey: featureKey,
					Total:      0,
					Tasks:      []*services.ViewerTask{},
				}, nil
			},
		}
		rec := makeRequest(
			http.MethodGet,
			"/api/v1/viewer/features/E01-F01/tasks?tag=voice&tag=&tag=auth&tag=%20%20",
			newTestMux(mock),
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-AC10-4b: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		if len(got.Tags) != 2 || got.Tags[0] != "voice" || got.Tags[1] != "auth" {
			t.Errorf("TC-AC10-4b: expected opts.Tags=[\"voice\",\"auth\"] after blanks dropped, got %#v", got.Tags)
		}
	})

	// TC-AC10-1 / AC-T4: feature exists + tag is unregistered → 400 with
	// `unregistered_tags` array body. The 400 path comes from the service's
	// *UnregisteredTagError.
	t.Run("TC-AC10-1_feature_found_unregistered_tag_returns_400", func(t *testing.T) {
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, _ string, opts services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				if len(opts.Tags) != 1 || opts.Tags[0] != "does-not-exist" {
					t.Errorf("TC-AC10-1: expected opts.Tags=[\"does-not-exist\"], got %#v", opts.Tags)
				}
				return nil, &services.UnregisteredTagError{Name: "does-not-exist"}
			},
		}
		rec := makeRequest(
			http.MethodGet,
			"/api/v1/viewer/features/E01-F01/tasks?tag=does-not-exist",
			newTestMux(mock),
		)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("TC-AC10-1: expected 400, got %d; body: %s", rec.Code, rec.Body.String())
		}
		assertContentTypeJSON(t, rec)

		var body struct {
			Error            string   `json:"error"`
			Message          string   `json:"message"`
			UnregisteredTags []string `json:"unregistered_tags"`
		}
		assertJSON(t, rec, &body)
		if body.Error == "" {
			t.Error("TC-AC10-1: expected non-empty error field")
		}
		if len(body.UnregisteredTags) != 1 || body.UnregisteredTags[0] != "does-not-exist" {
			t.Errorf("TC-AC10-1: expected unregistered_tags=[\"does-not-exist\"], got %#v", body.UnregisteredTags)
		}
	})

	// TC-AC10-2 / AC-T4: feature missing → 404, regardless of tag value.
	// Service returns a "feature not found" error; isNotFound() must catch it
	// before the tag-error path.
	t.Run("TC-AC10-2_feature_missing_returns_404_with_tag", func(t *testing.T) {
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, featureKey string, _ services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				return nil, fmt.Errorf("feature not found: %s", featureKey)
			},
		}
		rec := makeRequest(
			http.MethodGet,
			"/api/v1/viewer/features/E99-F99/tasks?tag=voice",
			newTestMux(mock),
		)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("TC-AC10-2: expected 404 even when ?tag= present, got %d; body: %s",
				rec.Code, rec.Body.String())
		}
	})

	// TC-AC10-3 / AC-T4 (precedence): when the service returns a feature-not-
	// found error, that 404 must take precedence over any tag-validation
	// outcome. The spec says "the feature lookup happens first" (REQ-F-012).
	//
	// We model this by having the service return the not-found error even
	// though the tag is also unregistered in the underlying vocabulary; the
	// handler's job is simply to translate the error it actually receives.
	// 404 must win — and the response body must NOT contain unregistered_tags.
	t.Run("TC-AC10-3_feature_not_found_precedence_over_tag", func(t *testing.T) {
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, featureKey string, _ services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				// Service decided feature lookup failed first. The
				// UnregisteredTagError path is NOT triggered here, mirroring
				// REQ-F-012's "feature lookup first" ordering.
				return nil, fmt.Errorf("feature not found: %s", featureKey)
			},
		}
		rec := makeRequest(
			http.MethodGet,
			"/api/v1/viewer/features/E99-F99/tasks?tag=does-not-exist",
			newTestMux(mock),
		)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("TC-AC10-3: expected 404 (not 400), got %d; body: %s", rec.Code, rec.Body.String())
		}
		// Confirm the 404 body does NOT carry tag-error fields.
		var body struct {
			UnregisteredTags []string `json:"unregistered_tags"`
		}
		// Use a tolerant decode — the 404 body has no unregistered_tags key,
		// so the slice must remain nil/empty.
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if len(body.UnregisteredTags) != 0 {
			t.Errorf("TC-AC10-3: 404 response must not include unregistered_tags, got %#v",
				body.UnregisteredTags)
		}
	})

	// TC-AC10-1 (errors.As wrapping, FeatureTasks branch): wrapped
	// UnregisteredTagError must still produce 400. Mirrors the Hierarchy
	// wrapping test; both handler branches use errors.As.
	t.Run("TC-AC10-1b_wrapped_unregistered_tag_still_400", func(t *testing.T) {
		mock := &MockViewerServicer{
			FeatureTasksFunc: func(_ context.Context, _ string, _ services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) {
				return nil, fmt.Errorf("feature tasks failed: %w",
					&services.UnregisteredTagError{Name: "ghost"})
			},
		}
		rec := makeRequest(
			http.MethodGet,
			"/api/v1/viewer/features/E01-F01/tasks?tag=ghost",
			newTestMux(mock),
		)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("TC-AC10-1b: expected 400 from wrapped UnregisteredTagError, got %d; body: %s",
				rec.Code, rec.Body.String())
		}
		var body struct {
			UnregisteredTags []string `json:"unregistered_tags"`
		}
		assertJSON(t, rec, &body)
		if len(body.UnregisteredTags) != 1 || body.UnregisteredTags[0] != "ghost" {
			t.Errorf("TC-AC10-1b: expected unregistered_tags=[\"ghost\"], got %#v", body.UnregisteredTags)
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

	// TC-H-053b: tech_debt entity_type is accepted and passed through.
	t.Run("TC-H-053b_tech_debt_entity_type_accepted", func(t *testing.T) {
		called := false
		mock := &MockViewerServicer{
			RecentActivityFunc: func(_ context.Context, opts services.RecentActivityOptions) (*services.RecentActivityResponse, error) {
				called = true
				if opts.EntityType != "tech_debt" {
					t.Fatalf("TC-H-053b: expected opts.EntityType=tech_debt, got %q", opts.EntityType)
				}
				return &services.RecentActivityResponse{Records: []*services.ActivityRecord{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/recent-activity?entity_type=tech_debt", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-H-053b: expected 200, got %d", rec.Code)
		}
		if !called {
			t.Error("TC-H-053b: RecentActivityFunc should have been called")
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
	// TC-H-060: Happy path — workflow metadata includes all supported levels
	t.Run("TC-H-060_happy_path_6_levels", func(t *testing.T) {
		levels := map[string]*services.WorkflowLevelMeta{
			"epic":        {Level: "epic", Statuses: []services.WorkflowStatusMeta{{Name: "active", Color: "green"}}, Transitions: []services.WorkflowTransitionMeta{}},
			"feature":     {Level: "feature", Statuses: []services.WorkflowStatusMeta{{Name: "active", Color: "green"}}, Transitions: []services.WorkflowTransitionMeta{}},
			"task":        {Level: "task", Statuses: []services.WorkflowStatusMeta{{Name: "todo", Color: "gray"}}, Transitions: []services.WorkflowTransitionMeta{}},
			"bug":         {Level: "bug", Statuses: []services.WorkflowStatusMeta{{Name: "reported", Color: "red"}}, Transitions: []services.WorkflowTransitionMeta{}},
			"change_card": {Level: "change_card", Statuses: []services.WorkflowStatusMeta{{Name: "proposed", Color: "blue"}}, Transitions: []services.WorkflowTransitionMeta{}},
			"tech_debt":   {Level: "tech_debt", Statuses: []services.WorkflowStatusMeta{{Name: "open", Color: "orange"}}, Transitions: []services.WorkflowTransitionMeta{}},
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
		for _, levelName := range []string{"epic", "feature", "task", "bug", "change_card", "tech_debt"} {
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

// ----- TC-031 and TC-033: GET /nav-folders -----

func TestHandler_NavFolders(t *testing.T) {
	t.Run("TC-031_route_returns_ordered_navigation_folders", func(t *testing.T) {
		mock := &MockViewerServicer{
			NavFoldersFunc: func(_ context.Context) (*services.NavFoldersResponse, error) {
				return &services.NavFoldersResponse{Folders: []services.NavFolder{
					{ID: "architecture", Label: "Architecture", Path: "docs/architecture", Source: "builtin", Exists: true},
					{ID: "product", Label: "Product", Path: "docs/product", Source: "builtin", Exists: true},
					{ID: "docs/runbooks", Label: "Runbooks", Path: "docs/runbooks", Source: "config", Exists: false},
				}}, nil
			},
		}

		rec := makeRequest(http.MethodGet, "/api/v1/viewer/nav-folders", newTestMux(mock))
		if rec.Code != http.StatusOK {
			t.Fatalf("TC-031: expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		assertContentTypeJSON(t, rec)

		var body services.NavFoldersResponse
		assertJSON(t, rec, &body)
		if got, want := body.Folders, []services.NavFolder{
			{ID: "architecture", Label: "Architecture", Path: "docs/architecture", Source: "builtin", Exists: true},
			{ID: "product", Label: "Product", Path: "docs/product", Source: "builtin", Exists: true},
			{ID: "docs/runbooks", Label: "Runbooks", Path: "docs/runbooks", Source: "config", Exists: false},
		}; !reflect.DeepEqual(got, want) {
			t.Errorf("TC-031: folders = %#v, want %#v", got, want)
		}
	})

	t.Run("TC-033_service_error_returns_existing_error_shape", func(t *testing.T) {
		mock := &MockViewerServicer{
			NavFoldersFunc: func(_ context.Context) (*services.NavFoldersResponse, error) {
				return nil, errors.New("load failure")
			},
		}

		rec := makeRequest(http.MethodGet, "/api/v1/viewer/nav-folders", newTestMux(mock))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("TC-033: expected 500, got %d: %s", rec.Code, rec.Body.String())
		}
		assertContentTypeJSON(t, rec)

		var body struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		assertJSON(t, rec, &body)
		if got, want := body.Error, http.StatusText(http.StatusInternalServerError); got != want {
			t.Errorf("TC-033: error = %q, want %q", got, want)
		}
		if got, want := body.Message, "failed to load nav folders"; got != want {
			t.Errorf("TC-033: message = %q, want %q", got, want)
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

// ----- T-E28-F06-004: GET /api/v1/viewer/tags -----
// Covers AC-T1 (happy path GET), AC-T2 (non-GET methods are rejected and the
// service is never called), AC-T3 (service error → 500), and the test-plan
// cases TC-AC01-2, TC-AC02-2, TC-AC03-1..5, TC-AC14-6.
//
// These tests are the regression tripwire for REQ-F-013 (the viewer API MUST
// NOT expose any tag write endpoint) and AC-03 / REQ-NF-012 from the F06
// spec. They were missing in the original implementation; UAT 2026-04-25
// rejected the task because no test invoked the /tags route at all.

func TestHandler_Tags(t *testing.T) {
	// TC-AC01-2 / TC-AC14-6: Happy path — empty vocabulary → 200 + {"tags":[]}.
	// AC-T1: Tags array MUST be a JSON array (not null) when empty.
	t.Run("TC-AC01-2_happy_path_empty_vocabulary", func(t *testing.T) {
		mock := &MockViewerServicer{
			TagsFunc: func(_ context.Context) (*services.TagsResponse, error) {
				return &services.TagsResponse{Tags: []services.TagDTO{}}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/tags", newTestMux(mock))

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-AC01-2: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		assertContentTypeJSON(t, rec)

		// Raw-string check guards against the `null` regression: if Tags were
		// serialized as nil instead of [] the body would be {"tags":null}.
		body := strings.TrimSpace(rec.Body.String())
		if body != `{"tags":[]}` {
			t.Errorf("TC-AC01-2: expected body %q, got %q", `{"tags":[]}`, body)
		}

		// Belt-and-suspenders: also decode and check structurally.
		var decoded struct {
			Tags []services.TagDTO `json:"tags"`
		}
		assertJSON(t, rec, &decoded)
		if decoded.Tags == nil {
			t.Error("TC-AC01-2: tags field decoded as nil; expected non-nil empty slice")
		}
		if len(decoded.Tags) != 0 {
			t.Errorf("TC-AC01-2: expected 0 tags, got %d", len(decoded.Tags))
		}
	})

	// TC-AC02-2: Happy path — populated vocabulary.
	t.Run("TC-AC02-2_happy_path_populated_vocabulary", func(t *testing.T) {
		mock := &MockViewerServicer{
			TagsFunc: func(_ context.Context) (*services.TagsResponse, error) {
				return &services.TagsResponse{
					Tags: []services.TagDTO{
						{Name: "auth"},
						{Name: "voice"},
					},
				}, nil
			},
		}
		rec := makeRequest(http.MethodGet, "/api/v1/viewer/tags", newTestMux(mock))

		if rec.Code != http.StatusOK {
			t.Fatalf("TC-AC02-2: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
		}
		assertContentTypeJSON(t, rec)

		var decoded struct {
			Tags []services.TagDTO `json:"tags"`
		}
		assertJSON(t, rec, &decoded)
		if len(decoded.Tags) != 2 {
			t.Fatalf("TC-AC02-2: expected 2 tags, got %d", len(decoded.Tags))
		}
		if decoded.Tags[0].Name != "auth" || decoded.Tags[1].Name != "voice" {
			t.Errorf("TC-AC02-2: unexpected tag order/values: %+v", decoded.Tags)
		}
	})

	// AC-T3: Service error → 500, no panic.
	t.Run("AC-T3_service_error_500", func(t *testing.T) {
		mock := &MockViewerServicer{
			TagsFunc: func(_ context.Context) (*services.TagsResponse, error) {
				return nil, fmt.Errorf("tag repository unavailable")
			},
		}

		// Defer-recover sanity check: handler must not panic on service error.
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("AC-T3: handler panicked on service error: %v", r)
			}
		}()

		rec := makeRequest(http.MethodGet, "/api/v1/viewer/tags", newTestMux(mock))

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("AC-T3: expected 500, got %d; body: %s", rec.Code, rec.Body.String())
		}
		assertContentTypeJSON(t, rec)

		var errResp map[string]string
		assertJSON(t, rec, &errResp)
		if errResp["error"] == "" {
			t.Error("AC-T3: expected non-empty error field in response")
		}
	})
}

// TestHandler_Tags_NonGetMethods is the AC-03 / REQ-NF-012 / AC-T2 / REQ-F-013
// regression tripwire. POST/PUT/PATCH/DELETE to /api/v1/viewer/tags must
// produce 404 or 405, AND the underlying service Tags() call counter must
// stay at 0 — i.e. the route must not be wired up to a non-GET handler.
func TestHandler_Tags_NonGetMethods(t *testing.T) {
	mutationMethods := []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}

	mock := &MockViewerServicer{
		// If Tags() is unexpectedly invoked, return a sentinel so the assertion
		// failure points cleanly at the regression. The TagsCallCount is the
		// real assertion; this is just defense-in-depth.
		TagsFunc: func(_ context.Context) (*services.TagsResponse, error) {
			return &services.TagsResponse{Tags: []services.TagDTO{}}, nil
		},
	}
	mux := newTestMux(mock)

	for _, method := range mutationMethods {
		t.Run("TC-AC03_"+method+"_rejected", func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/viewer/tags", nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			// TC-AC03-1..4: status must be 404 or 405 (Go 1.22 ServeMux returns
			// 405 when other methods are registered for the same path; either
			// is valid per the test plan).
			if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s /api/v1/viewer/tags: expected 404 or 405, got %d; body: %s",
					method, rec.Code, rec.Body.String())
			}
		})
	}

	// TC-AC03-5 / AC-T2: After all mutation attempts, Tags() must NEVER have
	// been called. This is the strongest guarantee that no write handler is
	// wired up.
	if mock.TagsCallCount != 0 {
		t.Errorf("TC-AC03-5: TagsFunc call counter expected 0 after %d non-GET requests, got %d",
			len(mutationMethods), mock.TagsCallCount)
	}
}

// TestHandler_Tags_PathParam_NonGetMethods extends the regression tripwire to
// cover the /api/v1/viewer/tags/<name> sub-path referenced by the spec
// (REQ-F-013) and test-plan section 1 note 95 ("Test both
// /api/v1/viewer/tags and /api/v1/viewer/tags/somename paths"). No write
// handler exists at this path either; all methods (including GET) should
// fall through to 404.
func TestHandler_Tags_PathParam_NonGetMethods(t *testing.T) {
	methods := []string{
		http.MethodGet, // GET /tags/<name> is also unwired — should be 404.
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}

	mock := &MockViewerServicer{
		TagsFunc: func(_ context.Context) (*services.TagsResponse, error) {
			return &services.TagsResponse{Tags: []services.TagDTO{}}, nil
		},
	}
	mux := newTestMux(mock)

	for _, method := range methods {
		t.Run("path_param_"+method+"_rejected", func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/viewer/tags/somename", nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s /api/v1/viewer/tags/somename: expected 404 or 405, got %d; body: %s",
					method, rec.Code, rec.Body.String())
			}
		})
	}

	// Same tripwire: no method on the path-param form must reach the service.
	if mock.TagsCallCount != 0 {
		t.Errorf("path-param: TagsFunc call counter expected 0 after %d requests, got %d",
			len(methods), mock.TagsCallCount)
	}
}

func TestHandler_SprintOverview(t *testing.T) {
	var gotKey string
	mock := &MockViewerServicer{
		SprintOverviewFunc: func(_ context.Context, key string) (*services.SprintOverviewResponse, error) {
			gotKey = key
			return &services.SprintOverviewResponse{
				Sprint: &models.Sprint{Key: key},
				Backlog: &services.SprintBacklog{
					SprintKey:  key,
					SprintName: "Current Sprint",
				},
				Readiness: &services.SprintReadiness{OverallScore: 77},
				Capacity:  []services.CapacityRow{{AgentType: "frontend"}},
				Summary:   &services.SprintSummaryResult{SprintKey: key},
			}, nil
		},
	}

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/sprint/overview?key=s024", newTestMux(mock))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	assertContentTypeJSON(t, rec)

	var body map[string]json.RawMessage
	assertJSON(t, rec, &body)
	for _, key := range []string{"sprint", "backlog", "readiness", "capacity"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("missing %q in sprint overview response", key)
		}
	}
	if gotKey != "S024" {
		t.Fatalf("expected normalized sprint key S024, got %q", gotKey)
	}
}

func TestHandler_SprintPlan(t *testing.T) {
	var gotKey string
	mock := &MockViewerServicer{
		SprintPlanFunc: func(_ context.Context, key string) (*services.SprintPlanView, error) {
			gotKey = key
			return &services.SprintPlanView{
				Sprint:    &models.Sprint{Key: key},
				Backlog:   []sprint.BacklogItem{},
				Capacity:  []services.CapacityRow{},
				Readiness: &services.SprintReadiness{OverallScore: 62},
			}, nil
		},
	}

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/sprint/plan?key=S024", newTestMux(mock))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	assertContentTypeJSON(t, rec)

	var body map[string]json.RawMessage
	assertJSON(t, rec, &body)
	if _, ok := body["sprint"]; !ok {
		t.Fatal("missing sprint in sprint plan response")
	}
	if _, ok := body["readiness"]; !ok {
		t.Fatal("missing readiness in sprint plan response")
	}
	if gotKey != "S024" {
		t.Fatalf("expected normalized sprint key S024, got %q", gotKey)
	}
}

func TestHandler_SprintReport(t *testing.T) {
	var gotKey string
	mock := &MockViewerServicer{
		SprintReportFunc: func(_ context.Context, key string) (*services.SprintReportResponse, error) {
			gotKey = key
			return &services.SprintReportResponse{
				Sprint:   &models.Sprint{Key: key},
				Burndown: &services.BurndownResult{SprintKey: key},
				Velocity: &services.VelocityResult{SprintCount: 6},
				Summary:  &services.SprintSummaryResult{SprintKey: key},
			}, nil
		},
	}

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/sprint/report?key=s024", newTestMux(mock))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	assertContentTypeJSON(t, rec)

	var body map[string]json.RawMessage
	assertJSON(t, rec, &body)
	for _, key := range []string{"sprint", "burndown", "velocity", "summary"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("missing %q in sprint report response", key)
		}
	}
	if gotKey != "S024" {
		t.Fatalf("expected normalized sprint key S024, got %q", gotKey)
	}
}

// TestHandler_SprintOverview_BacklogUsesGroupedView verifies that the sprint overview
// endpoint returns backlog.view="grouped" so the sidebar JS can aggregate bucket counts.
// Regression test for T-E27-F14-002: previously the active sprint used "ordered" view,
// causing all sidebar bucket counts to show zero.
func TestHandler_SprintOverview_BacklogUsesGroupedView(t *testing.T) {
	mock := &MockViewerServicer{
		SprintOverviewFunc: func(_ context.Context, key string) (*services.SprintOverviewResponse, error) {
			return &services.SprintOverviewResponse{
				Sprint: &models.Sprint{Key: key},
				Backlog: &services.SprintBacklog{
					SprintKey:  key,
					View:       "grouped",
					TotalCount: 8,
					Groups: []*services.BacklogGroup{
						{StatusCategory: "in_development", Items: []*services.BacklogItemView{{Key: "T-E01-F01-001"}}},
						{StatusCategory: "completed", Items: []*services.BacklogItemView{{Key: "T-E01-F01-002"}, {Key: "T-E01-F01-003"}}},
					},
				},
			}, nil
		},
	}

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/sprint/overview", newTestMux(mock))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp services.SprintOverviewResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Backlog == nil {
		t.Fatal("expected backlog in response, got nil")
	}
	if resp.Backlog.View != "grouped" {
		t.Errorf("expected backlog.view=%q, got %q — sidebar bucket aggregation requires grouped view", "grouped", resp.Backlog.View)
	}
}

// TestHandler_SprintOverview_BacklogGroupsHaveStatusCategory verifies that each group
// in the backlog has a non-empty status_category so the JS SPRINT_BUCKET_MAP lookup
// can match groups to buckets (ready/in_progress/blocked/done).
func TestHandler_SprintOverview_BacklogGroupsHaveStatusCategory(t *testing.T) {
	groups := []*services.BacklogGroup{
		{StatusCategory: "todo", Items: []*services.BacklogItemView{{Key: "T-E01-F01-001"}}},
		{StatusCategory: "in_development", Items: []*services.BacklogItemView{{Key: "T-E01-F01-002"}}},
		{StatusCategory: "completed", Items: []*services.BacklogItemView{{Key: "T-E01-F01-003"}}},
	}
	mock := &MockViewerServicer{
		SprintOverviewFunc: func(_ context.Context, key string) (*services.SprintOverviewResponse, error) {
			return &services.SprintOverviewResponse{
				Sprint: &models.Sprint{Key: key},
				Backlog: &services.SprintBacklog{
					SprintKey:  key,
					View:       "grouped",
					TotalCount: 3,
					Groups:     groups,
				},
			}, nil
		},
	}

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/sprint/overview", newTestMux(mock))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp services.SprintOverviewResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Backlog.Groups) == 0 {
		t.Fatal("expected non-empty backlog.groups")
	}
	for i, g := range resp.Backlog.Groups {
		if g.StatusCategory == "" {
			t.Errorf("group[%d] has empty status_category", i)
		}
	}
}

// TestHandler_SprintOverview_BacklogGroupCountsMatchTotal verifies that the sum of
// items across all groups equals backlog.total_count.
func TestHandler_SprintOverview_BacklogGroupCountsMatchTotal(t *testing.T) {
	items := func(keys ...string) []*services.BacklogItemView {
		out := make([]*services.BacklogItemView, len(keys))
		for i, k := range keys {
			out[i] = &services.BacklogItemView{Key: k}
		}
		return out
	}
	mock := &MockViewerServicer{
		SprintOverviewFunc: func(_ context.Context, key string) (*services.SprintOverviewResponse, error) {
			return &services.SprintOverviewResponse{
				Sprint: &models.Sprint{Key: key},
				Backlog: &services.SprintBacklog{
					SprintKey:  key,
					View:       "grouped",
					TotalCount: 5,
					Groups: []*services.BacklogGroup{
						{StatusCategory: "todo", Items: items("T-E01-F01-001", "T-E01-F01-002")},
						{StatusCategory: "in_development", Items: items("T-E01-F01-003")},
						{StatusCategory: "completed", Items: items("T-E01-F01-004", "T-E01-F01-005")},
					},
				},
			}, nil
		},
	}

	rec := makeRequest(http.MethodGet, "/api/v1/viewer/sprint/overview", newTestMux(mock))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp services.SprintOverviewResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	total := 0
	for _, g := range resp.Backlog.Groups {
		total += len(g.Items)
	}
	if total != resp.Backlog.TotalCount {
		t.Errorf("sum of group item counts (%d) != backlog.total_count (%d)", total, resp.Backlog.TotalCount)
	}
}
