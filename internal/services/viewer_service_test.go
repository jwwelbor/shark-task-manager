package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/entityhistory"
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
	ListRecentAcrossEntitiesFunc func(ctx context.Context, opts entityhistory.ListRecentAcrossEntitiesOptions) ([]*entityhistory.RecentActivityRow, error)
}

func (m *mockViewerHistoryRepo) ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error) {
	if m.ListByEntityFunc != nil {
		return m.ListByEntityFunc(ctx, entityType, entityID)
	}
	return nil, nil
}

func (m *mockViewerHistoryRepo) ListRecentAcrossEntities(ctx context.Context, opts entityhistory.ListRecentAcrossEntitiesOptions) ([]*entityhistory.RecentActivityRow, error) {
	if m.ListRecentAcrossEntitiesFunc != nil {
		return m.ListRecentAcrossEntitiesFunc(ctx, opts)
	}
	return nil, nil
}

type mockViewerEntityDocRepo struct {
	ListAllFunc func(ctx context.Context) ([]*BulkEntityDoc, error)
}

func (m *mockViewerEntityDocRepo) ListAll(ctx context.Context) ([]*BulkEntityDoc, error) {
	if m.ListAllFunc != nil {
		return m.ListAllFunc(ctx)
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
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 20}, EpicID: 10, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListFunc: func(ctx context.Context) ([]*models.Task, error) {
				return []*models.Task{
					{BaseEntity: models.BaseEntity{ID: 1}, FeatureID: 20, Status: "todo"},
					{BaseEntity: models.BaseEntity{ID: 2}, FeatureID: 20, Status: "todo", BlockedReason: &reason},
					{BaseEntity: models.BaseEntity{ID: 3}, FeatureID: 20, Status: "completed"},
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

func TestViewerService_Hierarchy_EmbedsDocs(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 10, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 20}, EpicID: 10, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListFunc: func(ctx context.Context) ([]*models.Task, error) {
				return nil, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	docRepo := &mockViewerEntityDocRepo{
		ListAllFunc: func(ctx context.Context) ([]*BulkEntityDoc, error) {
			return []*BulkEntityDoc{
				{EntityType: "epic", EntityID: 10, Title: "Epic Spec", FilePath: "docs/e01/spec.md"},
				{EntityType: "feature", EntityID: 20, Title: "Feature Design", FilePath: "docs/e01/f01/design.md"},
			}, nil
		},
	}
	svc.WithEntityDocRepo(docRepo)

	resp, err := svc.Hierarchy(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Epics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(resp.Epics))
	}
	epic := resp.Epics[0]
	if len(epic.Docs) != 1 || epic.Docs[0].Title != "Epic Spec" {
		t.Errorf("expected 1 epic doc with title 'Epic Spec', got %v", epic.Docs)
	}
	if len(epic.Features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(epic.Features))
	}
	f := epic.Features[0]
	if len(f.Docs) != 1 || f.Docs[0].Title != "Feature Design" {
		t.Errorf("expected 1 feature doc with title 'Feature Design', got %v", f.Docs)
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
	capturedOpts := entityhistory.ListRecentAcrossEntitiesOptions{}
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{
			ListRecentAcrossEntitiesFunc: func(ctx context.Context, opts entityhistory.ListRecentAcrossEntitiesOptions) ([]*entityhistory.RecentActivityRow, error) {
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
	capturedOpts := entityhistory.ListRecentAcrossEntitiesOptions{}
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{
			ListRecentAcrossEntitiesFunc: func(ctx context.Context, opts entityhistory.ListRecentAcrossEntitiesOptions) ([]*entityhistory.RecentActivityRow, error) {
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
			ListRecentAcrossEntitiesFunc: func(ctx context.Context, opts entityhistory.ListRecentAcrossEntitiesOptions) ([]*entityhistory.RecentActivityRow, error) {
				return []*entityhistory.RecentActivityRow{
					{EntityType: "task", Key: "E07-F01-001", Title: "My Task", FromStatus: "todo", ToStatus: "in_progress", ChangedAt: now},
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
	rec := resp.Records[0]
	if rec.EntityType != "task" {
		t.Errorf("expected entity_type %q, got %q", "task", rec.EntityType)
	}
	if rec.Key != "E07-F01-001" {
		t.Errorf("expected key %q, got %q", "E07-F01-001", rec.Key)
	}
	if rec.ToStatus != "in_progress" {
		t.Errorf("expected to_status %q, got %q", "in_progress", rec.ToStatus)
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

// ----- Error type string tests -----

func TestSecurityError_Error(t *testing.T) {
	err := &SecurityError{Path: "/etc/passwd"}
	msg := err.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
	if !contains(msg, "/etc/passwd") {
		t.Errorf("expected path in error message, got %q", msg)
	}
}

func TestFileTooLargeError_Error(t *testing.T) {
	err := &FileTooLargeError{Path: "/some/file.md", LimitMiB: 2}
	msg := err.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
	if !contains(msg, "/some/file.md") {
		t.Errorf("expected path in error message, got %q", msg)
	}
}

// ----- Summary error path tests -----

func TestViewerService_Summary_EpicRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return nil, fmt.Errorf("epic repo error")
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.Summary(context.Background())
	if err == nil {
		t.Fatal("expected error from epic repo failure")
	}
}

func TestViewerService_Summary_FeatureRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return nil, fmt.Errorf("feature repo error")
			},
		},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.Summary(context.Background())
	if err == nil {
		t.Fatal("expected error from feature repo failure")
	}
}

func TestViewerService_Summary_TaskRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{
			ListFunc: func(ctx context.Context) ([]*models.Task, error) {
				return nil, fmt.Errorf("task repo error")
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.Summary(context.Background())
	if err == nil {
		t.Fatal("expected error from task repo failure")
	}
}

func TestViewerService_Summary_BugStatusRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{
			CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
				return nil, fmt.Errorf("bug status repo error")
			},
		},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.Summary(context.Background())
	if err == nil {
		t.Fatal("expected error from bug status repo failure")
	}
}

func TestViewerService_Summary_BugSeverityRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{
			CountBySeverityFunc: func(ctx context.Context) (map[string]int, error) {
				return nil, fmt.Errorf("bug severity repo error")
			},
		},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.Summary(context.Background())
	if err == nil {
		t.Fatal("expected error from bug severity repo failure")
	}
}

func TestViewerService_Summary_ChangeCardRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{
			CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
				return nil, fmt.Errorf("change card repo error")
			},
		},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.Summary(context.Background())
	if err == nil {
		t.Fatal("expected error from change card repo failure")
	}
}

// ----- Hierarchy error path tests -----

func TestViewerService_Hierarchy_EpicRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return nil, fmt.Errorf("epic repo error")
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.Hierarchy(context.Background())
	if err == nil {
		t.Fatal("expected error from epic repo failure")
	}
}

func TestViewerService_Hierarchy_FeatureRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 10, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return nil, fmt.Errorf("feature repo error")
			},
		},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.Hierarchy(context.Background())
	if err == nil {
		t.Fatal("expected error from feature repo failure")
	}
}

func TestViewerService_Hierarchy_TaskRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 10, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 20}, EpicID: 10, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListFunc: func(ctx context.Context) ([]*models.Task, error) {
				return nil, fmt.Errorf("task repo error")
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.Hierarchy(context.Background())
	if err == nil {
		t.Fatal("expected error from task repo failure")
	}
}

func TestViewerService_Hierarchy_SortsFeaturesByExecutionOrder(t *testing.T) {
	ord1 := 1
	ord2 := 2
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 10, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				// Return in reverse order; service should sort by execution_order.
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 22, Key: "E01-F02"}, EpicID: 10, Status: "todo", ExecutionOrder: &ord2},
					{BaseEntity: models.BaseEntity{ID: 21, Key: "E01-F01"}, EpicID: 10, Status: "todo", ExecutionOrder: &ord1},
				}, nil
			},
		},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	resp, err := svc.Hierarchy(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Epics) != 1 {
		t.Fatalf("expected 1 epic")
	}
	feats := resp.Epics[0].Features
	if len(feats) != 2 {
		t.Fatalf("expected 2 features, got %d", len(feats))
	}
	if feats[0].ExecutionOrder == nil || *feats[0].ExecutionOrder != 1 {
		t.Errorf("expected first feature to have execution_order=1, got %v", feats[0].ExecutionOrder)
	}
}

// ----- History additional tests -----

func TestViewerService_History_Epic(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 55, Key: "E07"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{
			ListByEntityFunc: func(ctx context.Context, et models.EntityType, entityID int64) ([]*models.EntityHistory, error) {
				if entityID != 55 {
					return nil, fmt.Errorf("wrong entity ID: %d", entityID)
				}
				return []*models.EntityHistory{
					{ID: 1, EntityType: et, EntityID: entityID, ToStatus: "active"},
				}, nil
			},
		},
	)
	resp, err := svc.History(context.Background(), "E07")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.EntityType != models.EntityTypeEpic {
		t.Errorf("expected EntityTypeEpic, got %q", resp.EntityType)
	}
	if len(resp.Records) != 1 {
		t.Errorf("expected 1 history record, got %d", len(resp.Records))
	}
}

func TestViewerService_History_Feature(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 77, Key: key}}, nil
			},
		},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{
			ListByEntityFunc: func(ctx context.Context, et models.EntityType, entityID int64) ([]*models.EntityHistory, error) {
				if entityID != 77 {
					return nil, fmt.Errorf("wrong entity ID: %d", entityID)
				}
				return []*models.EntityHistory{
					{ID: 1, EntityType: et, EntityID: entityID, ToStatus: "in_progress"},
				}, nil
			},
		},
	)
	resp, err := svc.History(context.Background(), "E01-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.EntityType != models.EntityTypeFeature {
		t.Errorf("expected EntityTypeFeature, got %q", resp.EntityType)
	}
	if len(resp.Records) != 1 {
		t.Errorf("expected 1 history record, got %d", len(resp.Records))
	}
}

func TestViewerService_History_EpicNotFound(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{}, nil // empty — epic not found
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.History(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error for not-found epic")
	}
}

func TestViewerService_History_HistoryRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
				return &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{
			ListByEntityFunc: func(ctx context.Context, et models.EntityType, entityID int64) ([]*models.EntityHistory, error) {
				return nil, fmt.Errorf("history repo error")
			},
		},
	)
	_, err := svc.History(context.Background(), "E01-F01-001")
	if err == nil {
		t.Fatal("expected error from history repo failure")
	}
}

// ----- File additional tests -----

func TestViewerService_File_EpicFile(t *testing.T) {
	dir := t.TempDir()
	content := "# Epic Content"
	fpath := filepath.Join(dir, "epic.md")
	if err := os.WriteFile(fpath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	relPath := "epic.md"
	svc := NewViewerService(
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", FilePath: &relPath}},
				}, nil
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		dir,
	)

	resp, err := svc.File(context.Background(), "E01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Exists {
		t.Fatal("expected Exists=true")
	}
	if resp.Content != content {
		t.Errorf("expected content %q, got %q", content, resp.Content)
	}
}

func TestViewerService_File_TaskFile(t *testing.T) {
	dir := t.TempDir()
	content := "# Task Content"
	fpath := filepath.Join(dir, "task.md")
	if err := os.WriteFile(fpath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	relPath := "task.md"
	svc := NewViewerService(
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
				return &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: key, FilePath: &relPath}}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		dir,
	)

	resp, err := svc.File(context.Background(), "E01-F01-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Exists {
		t.Fatal("expected Exists=true")
	}
	if resp.Content != content {
		t.Errorf("expected content %q, got %q", content, resp.Content)
	}
}

func TestViewerService_File_EpicNoFilePath(t *testing.T) {
	dir := t.TempDir()
	svc := NewViewerService(
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}}, // FilePath is nil
				}, nil
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		dir,
	)

	resp, err := svc.File(context.Background(), "E01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Exists {
		t.Error("expected Exists=false when epic has no file_path")
	}
}

func TestViewerService_File_TaskNoFilePath(t *testing.T) {
	dir := t.TempDir()
	svc := NewViewerService(
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
				return &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil // FilePath nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		dir,
	)

	resp, err := svc.File(context.Background(), "E01-F01-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Exists {
		t.Error("expected Exists=false when task has no file_path")
	}
}

func TestViewerService_File_EpicNotFound(t *testing.T) {
	dir := t.TempDir()
	svc := NewViewerService(
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{}, nil // empty
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		dir,
	)

	_, err := svc.File(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error for not-found epic")
	}
}

func TestViewerService_File_UnsupportedEntityType(t *testing.T) {
	// Bug key format is unsupported for file read.
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	_, err := svc.File(context.Background(), "B001")
	if err == nil {
		t.Fatal("expected error for unsupported entity type (bug)")
	}
}

// ----- FeatureTasks additional tests -----

func TestViewerService_FeatureTasks_FeatureNotFound(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return nil, fmt.Errorf("feature not found: %s", key)
			},
		},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.FeatureTasks(context.Background(), "E01-F99", FeatureTaskOptions{})
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}

func TestViewerService_FeatureTasks_TaskRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
				return nil, fmt.Errorf("task repo error")
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{})
	if err == nil {
		t.Fatal("expected error from task repo failure")
	}
}

func TestViewerService_FeatureTasks_FilterByAgent(t *testing.T) {
	agentBackend := "backend"
	agentFrontend := "frontend"
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
					{BaseEntity: models.BaseEntity{ID: 1}, Status: "todo", AgentType: &agentBackend},
					{BaseEntity: models.BaseEntity{ID: 2}, Status: "todo", AgentType: &agentFrontend},
					{BaseEntity: models.BaseEntity{ID: 3}, Status: "todo"}, // no agent
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{Agent: "backend"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Tasks) != 1 {
		t.Errorf("expected 1 backend task, got %d", len(resp.Tasks))
	}
}

func TestViewerService_FeatureTasks_Pagination(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
				tasks := make([]*models.Task, 5)
				for i := range tasks {
					tasks[i] = &models.Task{BaseEntity: models.BaseEntity{ID: int64(i + 1)}, Status: "todo"}
				}
				return tasks, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	// Offset=2, Limit=2 → should get tasks 3 and 4.
	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 5 {
		t.Errorf("expected Total=5 (pre-filter), got %d", resp.Total)
	}
	if len(resp.Tasks) != 2 {
		t.Errorf("expected 2 paginated tasks, got %d", len(resp.Tasks))
	}
}

func TestViewerService_FeatureTasks_OffsetPastEnd(t *testing.T) {
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

	// Offset=100 > len(tasks)=1 → should return empty slice.
	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{Offset: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Tasks) != 0 {
		t.Errorf("expected 0 tasks when offset past end, got %d", len(resp.Tasks))
	}
}

// ----- resolveEntityID additional coverage tests -----

func TestViewerService_History_BugKey_Unsupported(t *testing.T) {
	// Bug keys are detected by detectEntityType but resolveEntityID has no case for them.
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.History(context.Background(), "B001")
	if err == nil {
		t.Fatal("expected error for bug key (unsupported in resolveEntityID)")
	}
}

func TestViewerService_History_ChangeCardKey_Unsupported(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.History(context.Background(), "CC-001")
	if err == nil {
		t.Fatal("expected error for change card key (unsupported in resolveEntityID)")
	}
}

func TestViewerService_History_EpicRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return nil, fmt.Errorf("epic repo error")
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.History(context.Background(), "E07")
	if err == nil {
		t.Fatal("expected error from epic repo failure in resolveEntityID")
	}
}

func TestViewerService_History_FeatureRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return nil, fmt.Errorf("feature repo error")
			},
		},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.History(context.Background(), "E01-F01")
	if err == nil {
		t.Fatal("expected error from feature repo failure in resolveEntityID")
	}
}

func TestViewerService_History_TaskRepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
				return nil, fmt.Errorf("task repo error")
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	_, err := svc.History(context.Background(), "E01-F01-001")
	if err == nil {
		t.Fatal("expected error from task repo failure in resolveEntityID")
	}
}

// ----- resolveFilePath additional coverage tests -----

func TestViewerService_File_EpicRepoError(t *testing.T) {
	dir := t.TempDir()
	svc := NewViewerService(
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return nil, fmt.Errorf("epic repo error")
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		dir,
	)
	_, err := svc.File(context.Background(), "E01")
	if err == nil {
		t.Fatal("expected error from epic repo failure in resolveFilePath")
	}
}

func TestViewerService_File_FeatureRepoError(t *testing.T) {
	dir := t.TempDir()
	svc := NewViewerService(
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return nil, fmt.Errorf("feature repo error")
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
		t.Fatal("expected error from feature repo failure in resolveFilePath")
	}
}

func TestViewerService_File_TaskRepoError(t *testing.T) {
	dir := t.TempDir()
	svc := NewViewerService(
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
				return nil, fmt.Errorf("task repo error")
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		dir,
	)
	_, err := svc.File(context.Background(), "E01-F01-001")
	if err == nil {
		t.Fatal("expected error from task repo failure in resolveFilePath")
	}
}

// ----- RecentActivity additional tests -----

func TestViewerService_RecentActivity_RepoError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{
			ListRecentAcrossEntitiesFunc: func(ctx context.Context, opts entityhistory.ListRecentAcrossEntitiesOptions) ([]*entityhistory.RecentActivityRow, error) {
				return nil, fmt.Errorf("history repo error")
			},
		},
	)
	_, err := svc.RecentActivity(context.Background(), RecentActivityOptions{Limit: 10})
	if err == nil {
		t.Fatal("expected error from history repo failure")
	}
}

func TestViewerService_RecentActivity_EntityTypeFilter(t *testing.T) {
	capturedOpts := entityhistory.ListRecentAcrossEntitiesOptions{}
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{
			ListRecentAcrossEntitiesFunc: func(ctx context.Context, opts entityhistory.ListRecentAcrossEntitiesOptions) ([]*entityhistory.RecentActivityRow, error) {
				capturedOpts = opts
				return nil, nil
			},
		},
	)

	since := time.Now().Add(-24 * time.Hour)
	_, err := svc.RecentActivity(context.Background(), RecentActivityOptions{
		Limit:      10,
		EntityType: "task",
		Since:      &since,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedOpts.EntityType != "task" {
		t.Errorf("expected entity_type=%q forwarded, got %q", "task", capturedOpts.EntityType)
	}
	if capturedOpts.Since == nil || !capturedOpts.Since.Equal(since) {
		t.Errorf("expected since timestamp forwarded to repo")
	}
}

// contains is a test helper for string contains checks.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ----- TC-F009: Dependency fields on ViewerTask (T-E27-F09-001) -----

// mockViewerTaskRelRepo is a mock for the ViewerTaskRelationshipRepository interface.
type mockViewerTaskRelRepo struct {
	ListAllFunc func(ctx context.Context) ([]*ViewerTaskRelationship, error)
}

func (m *mockViewerTaskRelRepo) ListAll(ctx context.Context) ([]*ViewerTaskRelationship, error) {
	if m.ListAllFunc != nil {
		return m.ListAllFunc(ctx)
	}
	return nil, nil
}

// TestViewerTask_DependsOnFieldExists verifies that ViewerTask exposes a DependsOn []string
// field (AC-T2: struct gains DependsOn, BlockedBy, Blocks string-slice fields).
func TestViewerTask_DependsOnFieldExists(t *testing.T) {
	// Construct a ViewerTask directly and verify the dependency fields are present.
	vt := &ViewerTask{
		Task:        &models.Task{},
		StatusColor: "#aaa",
		StatusPhase: "done",
		DependsOn:   []string{"E01-F01-001"},
		BlockedBy:   []string{"E01-F01-002"},
		Blocks:      []string{"E01-F01-003"},
	}
	if len(vt.DependsOn) != 1 || vt.DependsOn[0] != "E01-F01-001" {
		t.Errorf("DependsOn field not accessible: got %v", vt.DependsOn)
	}
	if len(vt.BlockedBy) != 1 || vt.BlockedBy[0] != "E01-F01-002" {
		t.Errorf("BlockedBy field not accessible: got %v", vt.BlockedBy)
	}
	if len(vt.Blocks) != 1 || vt.Blocks[0] != "E01-F01-003" {
		t.Errorf("Blocks field not accessible: got %v", vt.Blocks)
	}
}

// TestViewerService_Hierarchy_ParsesDependsOn verifies that the DependsOn field on
// ViewerTask is populated from task_relationships (not the legacy JSON blob).
// A task whose depends_on JSON blob contains entries but has no task_relationships rows
// must have an empty DependsOn slice (task_relationships is the single source of truth).
func TestViewerService_Hierarchy_ParsesDependsOn(t *testing.T) {
	dependsOnJSON := `["E01-F01-001","E01-F01-002"]`
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 10, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 20, Key: "E01-F01"}, EpicID: 10, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListFunc: func(ctx context.Context) ([]*models.Task, error) {
				return []*models.Task{
					{
						BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01-001"},
						FeatureID:  20,
						Status:     "todo",
						DependsOn:  &dependsOnJSON, // legacy blob — must NOT be used
					},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// No task_relationships rows — DependsOn must be empty (task_relationships is authoritative).
	svc.WithTaskRelRepo(&mockViewerTaskRelRepo{
		ListAllFunc: func(ctx context.Context) ([]*ViewerTaskRelationship, error) {
			return []*ViewerTaskRelationship{}, nil
		},
	})

	resp, err := svc.Hierarchy(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Epics) == 0 || len(resp.Epics[0].Features) == 0 || len(resp.Epics[0].Features[0].Tasks) == 0 {
		t.Fatal("expected hierarchy to contain at least one task")
	}
	task := resp.Epics[0].Features[0].Tasks[0]
	// DependsOn must be empty: task_relationships is authoritative, not the JSON blob.
	if len(task.DependsOn) != 0 {
		t.Errorf("DependsOn must be empty (uses task_relationships, not JSON blob); got %v", task.DependsOn)
	}
	if task.DependsOn == nil {
		t.Error("DependsOn should be an empty slice, not nil")
	}
}

// TestViewerService_Hierarchy_DependsOnNilToEmpty verifies that a task with no DependsOn
// field gets an empty (not nil) slice.
func TestViewerService_Hierarchy_DependsOnNilToEmpty(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 10, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 20, Key: "E01-F01"}, EpicID: 10, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListFunc: func(ctx context.Context) ([]*models.Task, error) {
				return []*models.Task{
					{
						BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01-001"},
						FeatureID:  20,
						Status:     "todo",
						DependsOn:  nil, // No dependencies.
					},
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
	task := resp.Epics[0].Features[0].Tasks[0]
	if task.DependsOn == nil {
		t.Error("DependsOn should be empty slice, not nil")
	}
	if task.BlockedBy == nil {
		t.Error("BlockedBy should be empty slice, not nil")
	}
	if task.Blocks == nil {
		t.Error("Blocks should be empty slice, not nil")
	}
}

// TestViewerService_Hierarchy_PopulatesBlockedByAndBlocks verifies that when
// a ViewerTaskRelationshipRepository is wired in, the BlockedBy and Blocks fields
// are populated from task_relationships (AC-T2: Blocks / BlockedBy fields present and populated).
func TestViewerService_Hierarchy_PopulatesBlockedByAndBlocks(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 10, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 20, Key: "E01-F01"}, EpicID: 10, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListFunc: func(ctx context.Context) ([]*models.Task, error) {
				return []*models.Task{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01-001"}, FeatureID: 20, Status: "todo"},
					{BaseEntity: models.BaseEntity{ID: 2, Key: "E01-F01-002"}, FeatureID: 20, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	// Wire the relationship repository:
	// Task 2 (E01-F01-002) depends_on Task 1 (E01-F01-001):
	//   → E01-F01-001 is blocked_by nothing; E01-F01-001 blocks E01-F01-002
	//   → E01-F01-002 depends_on E01-F01-001; E01-F01-002 is not blocking anything
	svc.WithTaskRelRepo(&mockViewerTaskRelRepo{
		ListAllFunc: func(ctx context.Context) ([]*ViewerTaskRelationship, error) {
			return []*ViewerTaskRelationship{
				{FromTaskID: 2, ToTaskID: 1, RelType: "depends_on", FromKey: "E01-F01-002", ToKey: "E01-F01-001"},
			}, nil
		},
	})

	resp, err := svc.Hierarchy(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tasks := resp.Epics[0].Features[0].Tasks
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	// Find each task by key.
	var task1, task2 *ViewerTask
	for _, t2 := range tasks {
		switch t2.Key {
		case "E01-F01-001":
			task1 = t2
		case "E01-F01-002":
			task2 = t2
		}
	}
	if task1 == nil || task2 == nil {
		t.Fatal("could not find both tasks in hierarchy response")
	}

	// Task 1 (E01-F01-001) is the prerequisite: it should show up in Blocks of E01-F01-002's dependency chain.
	// From the relationship "E01-F01-002 depends_on E01-F01-001":
	// E01-F01-001 has no DependsOn, no BlockedBy, but it blocks E01-F01-002.
	if len(task1.Blocks) != 1 || task1.Blocks[0] != "E01-F01-002" {
		t.Errorf("task1.Blocks expected [E01-F01-002], got %v", task1.Blocks)
	}
	if len(task1.BlockedBy) != 0 {
		t.Errorf("task1.BlockedBy expected empty, got %v", task1.BlockedBy)
	}

	// Task 2 (E01-F01-002) depends_on task 1: it has DependsOn=[E01-F01-001], no Blocks.
	if len(task2.BlockedBy) != 1 || task2.BlockedBy[0] != "E01-F01-001" {
		t.Errorf("task2.BlockedBy expected [E01-F01-001], got %v", task2.BlockedBy)
	}
}

// TestViewerService_Hierarchy_NoNewEndpointForDependencies verifies AC-T3: the dependency
// data is embedded in the hierarchy payload and no new per-entity GET is introduced.
// This test ensures the Hierarchy() method returns dependency data without calling
// a per-task lookup (the mock task repo has no GetByKey path invoked).
func TestViewerService_Hierarchy_NoNewEndpointForDependencies(t *testing.T) {
	getByKeyCallCount := 0
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 10, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 20, Key: "E01-F01"}, EpicID: 10, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListFunc: func(ctx context.Context) ([]*models.Task, error) {
				return []*models.Task{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01-001"}, FeatureID: 20, Status: "todo"},
				}, nil
			},
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
				getByKeyCallCount++
				return nil, fmt.Errorf("should not be called during Hierarchy")
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	_, err := svc.Hierarchy(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getByKeyCallCount > 0 {
		t.Errorf("Hierarchy() issued %d per-entity GetByKey calls; expected 0 (AC-T3)", getByKeyCallCount)
	}
}

// TestViewerService_Hierarchy_GracefulWhenRelRepoNil verifies that Hierarchy()
// works correctly when taskRelRepo is nil (BlockedBy/Blocks return empty slices).
func TestViewerService_Hierarchy_GracefulWhenRelRepoNil(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 10, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 20, Key: "E01-F01"}, EpicID: 10, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListFunc: func(ctx context.Context) ([]*models.Task, error) {
				return []*models.Task{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01-001"}, FeatureID: 20, Status: "todo"},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// taskRelRepo is NOT wired (nil).

	resp, err := svc.Hierarchy(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task := resp.Epics[0].Features[0].Tasks[0]
	// Should degrade gracefully with empty slices.
	if task.BlockedBy == nil {
		t.Error("BlockedBy should be [] not nil when relRepo is nil")
	}
	if task.Blocks == nil {
		t.Error("Blocks should be [] not nil when relRepo is nil")
	}
}

// TestViewerService_Hierarchy_DependsOnUsesRelationshipsExclusively verifies that
// DependsOn is populated from task_relationships exclusively, NOT from the legacy
// tasks.depends_on JSON blob. A task with a non-empty JSON blob but no matching
// task_relationships rows must have an empty DependsOn slice.
func TestViewerService_Hierarchy_DependsOnUsesRelationshipsExclusively(t *testing.T) {
	// Task has a JSON depends_on blob with two keys, but no task_relationships rows.
	// After the fix, DependsOn must be empty (relationship table is authoritative).
	legacyJSON := `["E01-F01-999","E01-F01-998"]`
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 10, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 20, Key: "E01-F01"}, EpicID: 10, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListFunc: func(ctx context.Context) ([]*models.Task, error) {
				return []*models.Task{
					{
						BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01-001"},
						FeatureID:  20,
						Status:     "todo",
						DependsOn:  &legacyJSON, // legacy JSON blob — should be ignored
					},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// Wire relationship repo that returns no relationships for this task.
	svc.WithTaskRelRepo(&mockViewerTaskRelRepo{
		ListAllFunc: func(ctx context.Context) ([]*ViewerTaskRelationship, error) {
			return []*ViewerTaskRelationship{}, nil // no relationships
		},
	})

	resp, err := svc.Hierarchy(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task := resp.Epics[0].Features[0].Tasks[0]
	if len(task.DependsOn) != 0 {
		t.Errorf("DependsOn must be empty when task_relationships has no rows for this task; got %v", task.DependsOn)
	}
}

// TestViewerService_Hierarchy_DependsOnFromRelationships verifies that when
// task_relationships has depends_on rows, DependsOn is populated correctly
// (from task_relationships, not the JSON blob).
func TestViewerService_Hierarchy_DependsOnFromRelationships(t *testing.T) {
	// Task has no JSON blob but task_relationships says it depends on another task.
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 10, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 20, Key: "E01-F01"}, EpicID: 10, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListFunc: func(ctx context.Context) ([]*models.Task, error) {
				return []*models.Task{
					// Task 2 depends on task 1; task 1 is a prerequisite.
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01-001"}, FeatureID: 20, Status: "completed", DependsOn: nil},
					{BaseEntity: models.BaseEntity{ID: 2, Key: "E01-F01-002"}, FeatureID: 20, Status: "todo", DependsOn: nil},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// E01-F01-002 depends_on E01-F01-001 via task_relationships.
	svc.WithTaskRelRepo(&mockViewerTaskRelRepo{
		ListAllFunc: func(ctx context.Context) ([]*ViewerTaskRelationship, error) {
			return []*ViewerTaskRelationship{
				{FromTaskID: 2, ToTaskID: 1, RelType: "depends_on", FromKey: "E01-F01-002", ToKey: "E01-F01-001"},
			}, nil
		},
	})

	resp, err := svc.Hierarchy(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tasks := resp.Epics[0].Features[0].Tasks
	var task1, task2 *ViewerTask
	for _, t2 := range tasks {
		switch t2.Key {
		case "E01-F01-001":
			task1 = t2
		case "E01-F01-002":
			task2 = t2
		}
	}
	if task1 == nil || task2 == nil {
		t.Fatal("could not find both tasks in hierarchy response")
	}
	// task2 depends_on task1: task2.DependsOn should contain task1's key.
	if len(task2.DependsOn) != 1 || task2.DependsOn[0] != "E01-F01-001" {
		t.Errorf("task2.DependsOn expected [E01-F01-001], got %v", task2.DependsOn)
	}
	// task1 has no depends_on: its DependsOn should be empty.
	if len(task1.DependsOn) != 0 {
		t.Errorf("task1.DependsOn expected [], got %v", task1.DependsOn)
	}
}

// TestViewerService_FeatureTasks_DependsOnUsesRelationships verifies that FeatureTasks
// populates DependsOn from task_relationships (not from the legacy JSON blob).
func TestViewerService_FeatureTasks_DependsOnUsesRelationships(t *testing.T) {
	legacyJSON := `["E01-F01-999"]` // blob that must NOT be used
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 20, Key: key}}, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
				return []*models.Task{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01-001"}, FeatureID: 20, Status: "completed", DependsOn: nil},
					{BaseEntity: models.BaseEntity{ID: 2, Key: "E01-F01-002"}, FeatureID: 20, Status: "todo", DependsOn: &legacyJSON},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// Wire relationship repo: task 2 depends_on task 1.
	svc.WithTaskRelRepo(&mockViewerTaskRelRepo{
		ListAllFunc: func(ctx context.Context) ([]*ViewerTaskRelationship, error) {
			return []*ViewerTaskRelationship{
				{FromTaskID: 2, ToTaskID: 1, RelType: "depends_on", FromKey: "E01-F01-002", ToKey: "E01-F01-001"},
			}, nil
		},
	})

	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var task1, task2 *ViewerTask
	for _, t2 := range resp.Tasks {
		switch t2.Key {
		case "E01-F01-001":
			task1 = t2
		case "E01-F01-002":
			task2 = t2
		}
	}
	if task1 == nil || task2 == nil {
		t.Fatal("could not find both tasks in FeatureTasks response")
	}
	// task2 depends_on task1 via relationships; DependsOn should reflect this.
	if len(task2.DependsOn) != 1 || task2.DependsOn[0] != "E01-F01-001" {
		t.Errorf("task2.DependsOn expected [E01-F01-001], got %v", task2.DependsOn)
	}
	// task1 has no depends_on relationships; its DependsOn must be empty (not using JSON blob).
	if len(task1.DependsOn) != 0 {
		t.Errorf("task1.DependsOn expected [], got %v", task1.DependsOn)
	}
	// Legacy JSON blob in task2 must NOT be used: len should be 1 (from relationships), not blob content.
	// (already covered above, but be explicit about the blob having "E01-F01-999" which should NOT appear)
	for _, key := range task2.DependsOn {
		if key == "E01-F01-999" {
			t.Error("DependsOn must not contain legacy JSON blob value E01-F01-999")
		}
	}
}

// TestViewerService_FeatureTasks_DependsOnNilWhenRelRepoNil verifies that FeatureTasks
// returns an empty (non-nil) DependsOn slice when taskRelRepo is nil.
func TestViewerService_FeatureTasks_DependsOnNilWhenRelRepoNil(t *testing.T) {
	legacyJSON := `["E01-F01-999"]`
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 20, Key: key}}, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
				return []*models.Task{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01-001"}, FeatureID: 20, Status: "todo", DependsOn: &legacyJSON},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// taskRelRepo is NOT wired (nil).

	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Tasks) == 0 {
		t.Fatal("expected at least one task")
	}
	task := resp.Tasks[0]
	if task.DependsOn == nil {
		t.Error("DependsOn should be empty slice (not nil) when relRepo is nil")
	}
	if len(task.DependsOn) != 0 {
		t.Errorf("DependsOn should be empty when relRepo is nil, got %v", task.DependsOn)
	}
}

// ----- mock types for Notes / RelatedDocs -----

type mockViewerEntityNoteRepo struct {
	GetByEntityFunc func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
}

func (m *mockViewerEntityNoteRepo) GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
	if m.GetByEntityFunc != nil {
		return m.GetByEntityFunc(ctx, entityType, entityID)
	}
	return nil, nil
}

type mockViewerDocByEntityRepo struct {
	ListForEntityFunc func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error)
}

func (m *mockViewerDocByEntityRepo) ListForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error) {
	if m.ListForEntityFunc != nil {
		return m.ListForEntityFunc(ctx, entityType, entityID)
	}
	return nil, nil
}

// ----- Notes tests (TC-F020-*) -----

// TC-F020-8: nil noteRepo degrades gracefully to empty notes (AC-T2)
func TestViewerService_Notes_NilNoteRepoReturnsEmptySlice(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{{BaseEntity: models.BaseEntity{ID: 1, Key: "E27"}}}, nil
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// noteRepo NOT wired (nil).

	resp, err := svc.Notes(context.Background(), "E27")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Notes == nil {
		t.Error("Notes should be [] not nil when noteRepo is nil (AC-T2)")
	}
	if len(resp.Notes) != 0 {
		t.Errorf("expected empty Notes slice, got %d items", len(resp.Notes))
	}
	if string(resp.EntityType) != "epic" {
		t.Errorf("expected entity_type=epic, got %q", resp.EntityType)
	}
	if resp.EntityKey != "E27" {
		t.Errorf("expected entity_key=E27, got %q", resp.EntityKey)
	}
}

// TC-F020-2: Notes ordered by created_at DESC (AC-020.2 / AC-T1)
func TestViewerService_Notes_OrderedDescending(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)

	notes := []*models.EntityNote{
		{ID: 1, NoteType: "comment", Content: "first", CreatedAt: t1},
		{ID: 2, NoteType: "decision", Content: "second", CreatedAt: t2},
		{ID: 3, NoteType: "solution", Content: "third", CreatedAt: t3},
	}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{{BaseEntity: models.BaseEntity{ID: 1, Key: "E27"}}}, nil
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithNoteRepo(&mockViewerEntityNoteRepo{
		GetByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
			return notes, nil
		},
	})

	resp, err := svc.Notes(context.Background(), "E27")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Notes) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(resp.Notes))
	}
	// Repo returns ASC; service should reverse to DESC.
	if resp.Notes[0].ID != 3 {
		t.Errorf("expected first note id=3 (newest), got %d", resp.Notes[0].ID)
	}
	if resp.Notes[1].ID != 2 {
		t.Errorf("expected second note id=2, got %d", resp.Notes[1].ID)
	}
	if resp.Notes[2].ID != 1 {
		t.Errorf("expected third note id=1 (oldest), got %d", resp.Notes[2].ID)
	}
}

// TC-F020-3: Only the six specified fields are serialised (AC-020.3)
func TestViewerService_Notes_NoteDTO_Fields(t *testing.T) {
	createdBy := "agent-001"
	notes := []*models.EntityNote{
		{
			ID:        42,
			NoteType:  models.NoteTypeComment,
			Content:   "hello world",
			CreatedBy: &createdBy,
			Metadata:  ptr("should not appear"),
			CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{{BaseEntity: models.BaseEntity{ID: 1, Key: "E27"}}}, nil
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithNoteRepo(&mockViewerEntityNoteRepo{
		GetByEntityFunc: func(_ context.Context, _ models.EntityType, _ int64) ([]*models.EntityNote, error) {
			return notes, nil
		},
	})

	resp, err := svc.Notes(context.Background(), "E27")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(resp.Notes))
	}
	n := resp.Notes[0]
	if n.ID != 42 {
		t.Errorf("expected ID=42, got %d", n.ID)
	}
	if n.NoteType != "comment" {
		t.Errorf("expected note_type=comment, got %q", n.NoteType)
	}
	if n.Content != "hello world" {
		t.Errorf("expected content='hello world', got %q", n.Content)
	}
	if n.CreatedBy != "agent-001" {
		t.Errorf("expected created_by=agent-001, got %q", n.CreatedBy)
	}
	// CreatedAt must be RFC3339 UTC.
	if n.CreatedAt != "2026-01-01T12:00:00Z" {
		t.Errorf("expected created_at=2026-01-01T12:00:00Z, got %q", n.CreatedAt)
	}
}

// TC-F020-4 (service side): key normalisation delegates to detectEntityType (AC-T4)
func TestViewerService_Notes_KeyNormalization(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{{BaseEntity: models.BaseEntity{ID: 1, Key: "E27"}}}, nil
			},
		},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(_ context.Context, _ string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 10, Key: "E27-F09"}}, nil
			},
		},
		&mockViewerTaskRepo{
			GetByKeyFunc: func(_ context.Context, _ string) (*models.Task, error) {
				return &models.Task{BaseEntity: models.BaseEntity{ID: 100, Key: "E27-F09-002"}}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithNoteRepo(&mockViewerEntityNoteRepo{
		GetByEntityFunc: func(_ context.Context, _ models.EntityType, _ int64) ([]*models.EntityNote, error) {
			return nil, nil
		},
	})

	cases := []struct {
		key        string
		wantEType  string
		wantKeyPfx string // prefix of expected EntityKey
	}{
		{"e27", "epic", "E27"},
		{"E27-F09", "feature", "E27-F09"},
		{"E27-F09-002", "task", "E27-F09-002"},
		{"T-E27-F09-002", "task", "T-E27-F09-002"},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			resp, err := svc.Notes(context.Background(), tc.key)
			if err != nil {
				t.Fatalf("key=%q: unexpected error: %v", tc.key, err)
			}
			if string(resp.EntityType) != tc.wantEType {
				t.Errorf("key=%q: expected entity_type=%q, got %q", tc.key, tc.wantEType, resp.EntityType)
			}
		})
	}
}

// ----- RelatedDocs tests (TC-F021-*) -----

// TC-F021-1: Empty list returns {docs:[]} not null (AC-021.1 / AC-T3)
func TestViewerService_RelatedDocs_NilRepoReturnsEmptySlice(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{{BaseEntity: models.BaseEntity{ID: 1, Key: "E27"}}}, nil
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// docByEntityRepo NOT wired (nil).

	resp, err := svc.RelatedDocs(context.Background(), "E27")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Docs == nil {
		t.Error("Docs should be [] not nil when docByEntityRepo is nil (AC-T3)")
	}
	if len(resp.Docs) != 0 {
		t.Errorf("expected empty Docs slice, got %d items", len(resp.Docs))
	}
}

// TC-F021-2: Documents ordered most-recent-link-first (AC-021.2)
func TestViewerService_RelatedDocs_PreservesRepoOrder(t *testing.T) {
	docs := []*models.Document{
		{ID: 10, Title: "Doc C", FilePath: "docs/c.md"},
		{ID: 5, Title: "Doc B", FilePath: "docs/b.md"},
		{ID: 1, Title: "Doc A", FilePath: "docs/a.md"},
	}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{{BaseEntity: models.BaseEntity{ID: 1, Key: "E27"}}}, nil
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithDocByEntityRepo(&mockViewerDocByEntityRepo{
		ListForEntityFunc: func(_ context.Context, _ models.EntityType, _ int64) ([]*models.Document, error) {
			return docs, nil
		},
	})

	resp, err := svc.RelatedDocs(context.Background(), "E27")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Docs) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(resp.Docs))
	}
	// Service must preserve repository order (most-recent-link-first semantics).
	if resp.Docs[0].ID != 10 {
		t.Errorf("expected first doc id=10, got %d", resp.Docs[0].ID)
	}
	if resp.Docs[1].ID != 5 {
		t.Errorf("expected second doc id=5, got %d", resp.Docs[1].ID)
	}
	if resp.Docs[2].ID != 1 {
		t.Errorf("expected third doc id=1, got %d", resp.Docs[2].ID)
	}
}

// TC-F021-3: Paths returned as stored (AC-021.3)
func TestViewerService_RelatedDocs_PathReturnedAsStored(t *testing.T) {
	const storedPath = "docs/plan/E27-F09/spec.md"
	docs := []*models.Document{
		{ID: 1, Title: "Spec", FilePath: storedPath},
	}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{{BaseEntity: models.BaseEntity{ID: 1, Key: "E27"}}}, nil
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithDocByEntityRepo(&mockViewerDocByEntityRepo{
		ListForEntityFunc: func(_ context.Context, _ models.EntityType, _ int64) ([]*models.Document, error) {
			return docs, nil
		},
	})

	resp, err := svc.RelatedDocs(context.Background(), "E27")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(resp.Docs))
	}
	if resp.Docs[0].FilePath != storedPath {
		t.Errorf("expected file_path=%q, got %q", storedPath, resp.Docs[0].FilePath)
	}
}

// TC-F020-err: Error from noteRepo propagates back to caller (AC-T4).
func TestViewerService_Notes_ErrorFromRepoIsPropagated(t *testing.T) {
	repoErr := fmt.Errorf("db connection failed")

	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{{BaseEntity: models.BaseEntity{ID: 1, Key: "E27"}}}, nil
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithNoteRepo(&mockViewerEntityNoteRepo{
		GetByEntityFunc: func(_ context.Context, _ models.EntityType, _ int64) ([]*models.EntityNote, error) {
			return nil, repoErr
		},
	})

	resp, err := svc.Notes(context.Background(), "E27")
	if err == nil {
		t.Fatal("expected error from noteRepo to propagate, got nil")
	}
	if resp != nil {
		t.Errorf("expected nil response on error, got %+v", resp)
	}
}

// TC-F021-err: Error from docByEntityRepo propagates back to caller (AC-T4).
func TestViewerService_RelatedDocs_ErrorFromRepoIsPropagated(t *testing.T) {
	repoErr := fmt.Errorf("document repo unavailable")

	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{{BaseEntity: models.BaseEntity{ID: 1, Key: "E27"}}}, nil
			},
		},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithDocByEntityRepo(&mockViewerDocByEntityRepo{
		ListForEntityFunc: func(_ context.Context, _ models.EntityType, _ int64) ([]*models.Document, error) {
			return nil, repoErr
		},
	})

	resp, err := svc.RelatedDocs(context.Background(), "E27")
	if err == nil {
		t.Fatal("expected error from docByEntityRepo to propagate, got nil")
	}
	if resp != nil {
		t.Errorf("expected nil response on error, got %+v", resp)
	}
}
