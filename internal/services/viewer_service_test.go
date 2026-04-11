package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// ----- Mocks for ViewerService dependencies -----

type mockViewerEpicRepo struct {
	ListFunc func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error)
}

func (m *mockViewerEpicRepo) List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, status)
	}
	return nil, nil
}

type mockViewerFeatureRepo struct {
	ListFunc       func(ctx context.Context) ([]*models.Feature, error)
	ListByEpicFunc func(ctx context.Context, epicID int64) ([]*models.Feature, error)
	GetByKeyFunc   func(ctx context.Context, key string) (*models.Feature, error)
}

func (m *mockViewerFeatureRepo) List(ctx context.Context) ([]*models.Feature, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

func (m *mockViewerFeatureRepo) ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error) {
	if m.ListByEpicFunc != nil {
		return m.ListByEpicFunc(ctx, epicID)
	}
	return nil, nil
}

func (m *mockViewerFeatureRepo) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, nil
}

type mockViewerTaskRepo struct {
	ListFunc          func(ctx context.Context) ([]*models.Task, error)
	ListByFeatureFunc func(ctx context.Context, featureID int64) ([]*models.Task, error)
	GetByKeyFunc      func(ctx context.Context, key string) (*models.Task, error)
}

func (m *mockViewerTaskRepo) List(ctx context.Context) ([]*models.Task, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

func (m *mockViewerTaskRepo) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
	if m.ListByFeatureFunc != nil {
		return m.ListByFeatureFunc(ctx, featureID)
	}
	return nil, nil
}

func (m *mockViewerTaskRepo) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, nil
}

type mockViewerBugRepo struct {
	CountByStatusFunc   func(ctx context.Context) (map[string]int, error)
	CountBySeverityFunc func(ctx context.Context) (map[string]int, error)
}

func (m *mockViewerBugRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	if m.CountByStatusFunc != nil {
		return m.CountByStatusFunc(ctx)
	}
	return map[string]int{}, nil
}

func (m *mockViewerBugRepo) CountBySeverity(ctx context.Context) (map[string]int, error) {
	if m.CountBySeverityFunc != nil {
		return m.CountBySeverityFunc(ctx)
	}
	return map[string]int{}, nil
}

type mockViewerChangeCardRepo struct {
	CountByStatusFunc func(ctx context.Context) (map[string]int, error)
}

func (m *mockViewerChangeCardRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	if m.CountByStatusFunc != nil {
		return m.CountByStatusFunc(ctx)
	}
	return map[string]int{}, nil
}

type mockViewerHistoryRepo struct {
	ListByEntityFunc             func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error)
	ListRecentAcrossEntitiesFunc func(ctx context.Context, opts RecentActivityOptions) ([]*models.EntityHistory, error)
}

func (m *mockViewerHistoryRepo) ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error) {
	if m.ListByEntityFunc != nil {
		return m.ListByEntityFunc(ctx, entityType, entityID)
	}
	return nil, nil
}

func (m *mockViewerHistoryRepo) ListRecentAcrossEntities(ctx context.Context, opts RecentActivityOptions) ([]*models.EntityHistory, error) {
	if m.ListRecentAcrossEntitiesFunc != nil {
		return m.ListRecentAcrossEntitiesFunc(ctx, opts)
	}
	return nil, nil
}

// ----- helpers -----

func testWorkflowSvc(t *testing.T) *workflow.Service {
	t.Helper()
	// Use an empty project root so the service loads the default workflow.
	dir := t.TempDir()
	return workflow.NewService(dir)
}

func ptr[T any](v T) *T { return &v }

func buildViewerService(
	t *testing.T,
	epicRepo ViewerEpicRepository,
	featureRepo ViewerFeatureRepository,
	taskRepo ViewerTaskRepository,
	bugRepo ViewerBugRepository,
	ccRepo ViewerChangeCardRepository,
	histRepo ViewerEntityHistoryRepository,
) *ViewerService {
	t.Helper()
	return NewViewerService(
		epicRepo,
		featureRepo,
		taskRepo,
		bugRepo,
		ccRepo,
		histRepo,
		testWorkflowSvc(t),
		nil, // statusCalc optional
		t.TempDir(),
	)
}

// ----- Constructor tests -----

func TestNewViewerService_PanicsOnNilEpicRepo(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil epicRepo")
		}
	}()
	_ = NewViewerService(nil,
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		t.TempDir(),
	)
}

func TestNewViewerService_PanicsOnNilFeatureRepo(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil featureRepo")
		}
	}()
	_ = NewViewerService(&mockViewerEpicRepo{},
		nil,
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		t.TempDir(),
	)
}

func TestNewViewerService_PanicsOnNilTaskRepo(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil taskRepo")
		}
	}()
	_ = NewViewerService(&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		nil,
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		t.TempDir(),
	)
}

func TestNewViewerService_PanicsOnNilWorkflow(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil workflowSvc")
		}
	}()
	_ = NewViewerService(&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		nil, // workflowSvc — must panic
		nil,
		t.TempDir(),
	)
}

// ----- Summary tests -----

func TestViewerService_Summary_EmptyProject(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	resp, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Epics.Total != 0 {
		t.Errorf("expected 0 epics, got %d", resp.Epics.Total)
	}
	if resp.Tasks.Total != 0 {
		t.Errorf("expected 0 tasks, got %d", resp.Tasks.Total)
	}
	if resp.Tasks.BlockedCount != 0 {
		t.Errorf("expected 0 blocked tasks, got %d", resp.Tasks.BlockedCount)
	}
}

func TestViewerService_Summary_CountsEntities(t *testing.T) {
	reason := "waiting"
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
					{BaseEntity: models.BaseEntity{ID: 2, Key: "E02"}, Status: models.EpicStatusActive},
					{BaseEntity: models.BaseEntity{ID: 3, Key: "E03"}, Status: models.EpicStatusCompleted},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 1}, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListFunc: func(ctx context.Context) ([]*models.Task, error) {
				return []*models.Task{
					{BaseEntity: models.BaseEntity{ID: 1}, Status: "todo"},
					{BaseEntity: models.BaseEntity{ID: 2}, Status: "todo", BlockedReason: &reason},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Epics.Total != 3 {
		t.Errorf("expected 3 epics, got %d", resp.Epics.Total)
	}
	if resp.Features.Total != 1 {
		t.Errorf("expected 1 feature, got %d", resp.Features.Total)
	}
	if resp.Tasks.Total != 2 {
		t.Errorf("expected 2 tasks, got %d", resp.Tasks.Total)
	}
	if resp.Tasks.BlockedCount != 1 {
		t.Errorf("expected 1 blocked task, got %d", resp.Tasks.BlockedCount)
	}
}

func TestViewerService_Summary_UnknownStatusFallback(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: "totally_unknown_status"},
				}, nil
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Epics.Total != 1 {
		t.Fatalf("expected 1 epic, got %d", resp.Epics.Total)
	}
	sc := resp.Epics.ByStatus[0]
	if sc.Color != "gray" {
		t.Errorf("expected gray fallback color, got %q", sc.Color)
	}
	if sc.Phase != "unknown" {
		t.Errorf("expected 'unknown' fallback phase, got %q", sc.Phase)
	}
}

// ----- Hierarchy tests -----

func TestViewerService_Hierarchy_Empty(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	resp, err := svc.Hierarchy(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Epics) != 0 {
		t.Errorf("expected 0 epics, got %d", len(resp.Epics))
	}
}

func TestViewerService_Hierarchy_EmbedsTasks(t *testing.T) {
	reason := "blocked"
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 10, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListByEpicFunc: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 20}, EpicID: epicID, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
				return []*models.Task{
					{BaseEntity: models.BaseEntity{ID: 1}, Status: "todo"},
					{BaseEntity: models.BaseEntity{ID: 2}, Status: "todo", BlockedReason: &reason},
					{BaseEntity: models.BaseEntity{ID: 3}, Status: "completed"},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.Hierarchy(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Epics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(resp.Epics))
	}
	epic := resp.Epics[0]
	if len(epic.Features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(epic.Features))
	}
	f := epic.Features[0]
	if f.TaskCount != 3 {
		t.Errorf("expected TaskCount=3, got %d", f.TaskCount)
	}
	if f.BlockedCount != 1 {
		t.Errorf("expected BlockedCount=1, got %d", f.BlockedCount)
	}
}

// ----- detectEntityType tests -----

func TestDetectEntityType_Epic(t *testing.T) {
	et, err := detectEntityType("E07")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if et != models.EntityTypeEpic {
		t.Errorf("expected EntityTypeEpic, got %q", et)
	}
}

func TestDetectEntityType_Feature(t *testing.T) {
	et, err := detectEntityType("E07-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if et != models.EntityTypeFeature {
		t.Errorf("expected EntityTypeFeature, got %q", et)
	}
}

func TestDetectEntityType_Task_Short(t *testing.T) {
	et, err := detectEntityType("E07-F01-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if et != models.EntityTypeTask {
		t.Errorf("expected EntityTypeTask, got %q", et)
	}
}

func TestDetectEntityType_Task_Long(t *testing.T) {
	et, err := detectEntityType("T-E07-F01-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if et != models.EntityTypeTask {
		t.Errorf("expected EntityTypeTask, got %q", et)
	}
}

func TestDetectEntityType_Bug(t *testing.T) {
	et, err := detectEntityType("B042")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if et != models.EntityTypeBug {
		t.Errorf("expected EntityTypeBug, got %q", et)
	}
}

func TestDetectEntityType_ChangeCard(t *testing.T) {
	et, err := detectEntityType("CC-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if et != models.EntityTypeChange {
		t.Errorf("expected EntityTypeChange, got %q", et)
	}
}

func TestDetectEntityType_Unknown(t *testing.T) {
	_, err := detectEntityType("INVALID-KEY")
	if err == nil {
		t.Fatal("expected error for unknown key format")
	}
}

func TestDetectEntityType_CaseInsensitive(t *testing.T) {
	et, err := detectEntityType("e07-f01-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if et != models.EntityTypeTask {
		t.Errorf("expected EntityTypeTask for lowercase key, got %q", et)
	}
}

// ----- History tests -----

func TestViewerService_History_Task(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
				return &models.Task{BaseEntity: models.BaseEntity{ID: 99, Key: key}}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{
			ListByEntityFunc: func(ctx context.Context, et models.EntityType, entityID int64) ([]*models.EntityHistory, error) {
				return []*models.EntityHistory{
					{ID: 1, EntityType: et, EntityID: entityID, FromStatus: strPtr("todo"), ToStatus: "in_progress"},
				}, nil
			},
		},
	)

	resp, err := svc.History(context.Background(), "E01-F01-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.EntityType != models.EntityTypeTask {
		t.Errorf("expected EntityTypeTask, got %q", resp.EntityType)
	}
	if len(resp.Records) != 1 {
		t.Errorf("expected 1 record, got %d", len(resp.Records))
	}
}

func TestViewerService_History_UnknownKey(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.History(context.Background(), "TOTALLY-INVALID")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

// ----- File tests -----

func TestViewerService_File_NotFound(t *testing.T) {
	fp := "nonexistent/path.md"
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{FilePath: &fp}}, nil
			},
		},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.File(context.Background(), "E01-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Exists {
		t.Error("expected Exists=false for missing file")
	}
}

func TestViewerService_File_ExistsAndReadable(t *testing.T) {
	dir := t.TempDir()
	content := "# Feature Markdown\n\nHello world."
	fpath := filepath.Join(dir, "docs", "feature.md")
	if err := os.MkdirAll(filepath.Dir(fpath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fpath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	relPath := "docs/feature.md"
	svc := NewViewerService(
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{FilePath: &relPath}}, nil
			},
		},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		dir,
	)

	resp, err := svc.File(context.Background(), "E01-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Exists {
		t.Fatal("expected Exists=true for present file")
	}
	if resp.Content != content {
		t.Errorf("expected content %q, got %q", content, resp.Content)
	}
}

func TestViewerService_File_SecurityError_PathTraversal(t *testing.T) {
	dir := t.TempDir()

	// Create a file outside the project root.
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.md")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Try to traverse to outside file via absolute path in FilePath.
	svc := NewViewerService(
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{FilePath: &outsideFile}}, nil
			},
		},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		dir,
	)

	_, err := svc.File(context.Background(), "E01-F01")
	if err == nil {
		t.Fatal("expected SecurityError, got nil")
	}
	var secErr *SecurityError
	if !isSecurityError(err, &secErr) {
		t.Errorf("expected *SecurityError, got %T: %v", err, err)
	}
}

// isSecurityError is a helper to unwrap/check for *SecurityError.
func isSecurityError(err error, target **SecurityError) bool {
	if se, ok := err.(*SecurityError); ok {
		*target = se
		return true
	}
	return false
}

func TestViewerService_File_FileTooLarge(t *testing.T) {
	dir := t.TempDir()

	// Write a file larger than 2 MiB.
	bigPath := filepath.Join(dir, "big.md")
	big := make([]byte, viewerFileSizeLimit+512)
	for i := range big {
		big[i] = 'x'
	}
	if err := os.WriteFile(bigPath, big, 0o644); err != nil {
		t.Fatal(err)
	}

	relPath := "big.md"
	svc := NewViewerService(
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{FilePath: &relPath}}, nil
			},
		},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		dir,
	)

	_, err := svc.File(context.Background(), "E01-F01")
	if err == nil {
		t.Fatal("expected FileTooLargeError, got nil")
	}
	if _, ok := err.(*FileTooLargeError); !ok {
		t.Errorf("expected *FileTooLargeError, got %T: %v", err, err)
	}
}

func TestViewerService_File_NoFilePath(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				// FilePath is nil.
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
			},
		},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.File(context.Background(), "E01-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Exists {
		t.Error("expected Exists=false when entity has no file_path")
	}
}

// ----- FeatureTasks tests -----

func TestViewerService_FeatureTasks_LimitClamping(t *testing.T) {
	captured := FeatureTaskOptions{}
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
				return nil, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	// Test 0 → 200.
	captured.Limit = 0
	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", captured)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	// Test >500 → 500 (verify we don't panic with a large list).
	captured.Limit = 9999
	resp, err = svc.FeatureTasks(context.Background(), "E01-F01", captured)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestViewerService_FeatureTasks_FilterByStatus(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
				return []*models.Task{
					{BaseEntity: models.BaseEntity{ID: 1}, Status: "todo"},
					{BaseEntity: models.BaseEntity{ID: 2}, Status: "completed"},
					{BaseEntity: models.BaseEntity{ID: 3}, Status: "todo"},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{Status: "todo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 3 {
		t.Errorf("expected Total=3 (pre-filter), got %d", resp.Total)
	}
	if len(resp.Tasks) != 2 {
		t.Errorf("expected 2 filtered tasks, got %d", len(resp.Tasks))
	}
}

func TestViewerService_FeatureTasks_FilterByBlocked(t *testing.T) {
	reason := "waiting"
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
				return []*models.Task{
					{BaseEntity: models.BaseEntity{ID: 1}, Status: "todo"},
					{BaseEntity: models.BaseEntity{ID: 2}, Status: "todo", BlockedReason: &reason},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{Blocked: ptr(true)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Tasks) != 1 {
		t.Errorf("expected 1 blocked task, got %d", len(resp.Tasks))
	}
}

func TestViewerService_FeatureTasks_StatusColorPresent(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
				return []*models.Task{
					{BaseEntity: models.BaseEntity{ID: 1}, Status: "todo"},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Tasks) != 1 {
		t.Fatalf("expected 1 task")
	}
	// StatusColor must be non-empty (at minimum the gray fallback).
	if resp.Tasks[0].StatusColor == "" {
		t.Error("expected non-empty StatusColor")
	}
}

// ----- RecentActivity tests -----

func TestViewerService_RecentActivity_LimitClamp_ZeroBecomeFifty(t *testing.T) {
	capturedOpts := RecentActivityOptions{}
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{
			ListRecentAcrossEntitiesFunc: func(ctx context.Context, opts RecentActivityOptions) ([]*models.EntityHistory, error) {
				capturedOpts = opts
				return nil, nil
			},
		},
	)

	_, err := svc.RecentActivity(context.Background(), RecentActivityOptions{Limit: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedOpts.Limit != 50 {
		t.Errorf("expected limit clamped to 50, got %d", capturedOpts.Limit)
	}
}

func TestViewerService_RecentActivity_LimitClamp_OverTwoHundred(t *testing.T) {
	capturedOpts := RecentActivityOptions{}
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{
			ListRecentAcrossEntitiesFunc: func(ctx context.Context, opts RecentActivityOptions) ([]*models.EntityHistory, error) {
				capturedOpts = opts
				return nil, nil
			},
		},
	)

	_, err := svc.RecentActivity(context.Background(), RecentActivityOptions{Limit: 500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedOpts.Limit != 200 {
		t.Errorf("expected limit clamped to 200, got %d", capturedOpts.Limit)
	}
}

func TestViewerService_RecentActivity_ReturnsRecords(t *testing.T) {
	now := time.Now()
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{
			ListRecentAcrossEntitiesFunc: func(ctx context.Context, opts RecentActivityOptions) ([]*models.EntityHistory, error) {
				return []*models.EntityHistory{
					{ID: 1, EntityType: models.EntityTypeTask, EntityID: 5, FromStatus: strPtr("todo"), ToStatus: "in_progress", ChangedAt: now},
				}, nil
			},
		},
	)

	resp, err := svc.RecentActivity(context.Background(), RecentActivityOptions{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Records) != 1 {
		t.Errorf("expected 1 record, got %d", len(resp.Records))
	}
}

// ----- WorkflowMeta tests -----

func TestViewerService_WorkflowMeta_ContainsAllLevels(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.WorkflowMeta(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedLevels := []string{
		workflow.LevelEpic,
		workflow.LevelFeature,
		workflow.LevelTask,
		workflow.LevelBug,
		workflow.LevelChange,
	}
	for _, level := range expectedLevels {
		if _, ok := resp.Levels[level]; !ok {
			t.Errorf("expected level %q in WorkflowMeta response", level)
		}
	}
}

func TestViewerService_WorkflowMeta_TransitionsHaveDirection(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.WorkflowMeta(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	taskLevel, ok := resp.Levels[workflow.LevelTask]
	if !ok {
		t.Fatal("task level missing from WorkflowMeta")
	}
	for _, tr := range taskLevel.Transitions {
		switch tr.Direction {
		case "forward", "backward", "lateral":
			// valid
		default:
			t.Errorf("unexpected direction %q for transition %s→%s", tr.Direction, tr.From, tr.To)
		}
	}
}

func TestViewerService_WorkflowMeta_StatusesHaveColor(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.WorkflowMeta(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	taskLevel := resp.Levels[workflow.LevelTask]
	for _, st := range taskLevel.Statuses {
		if st.Color == "" {
			t.Errorf("status %q has empty color (expected at least 'gray' fallback)", st.Name)
		}
		if st.Phase == "" {
			t.Errorf("status %q has empty phase (expected at least 'unknown' fallback)", st.Name)
		}
	}
}
