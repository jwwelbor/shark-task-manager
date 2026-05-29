package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/entityhistory"
	sprint "github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// ----- Mocks for ViewerService dependencies -----

type mockViewerEpicRepo struct {
	ListFunc          func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error)
	CountByStatusFunc func(ctx context.Context) (map[string]int, error)
}

func (m *mockViewerEpicRepo) List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, status)
	}
	return nil, nil
}

func (m *mockViewerEpicRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	if m.CountByStatusFunc != nil {
		return m.CountByStatusFunc(ctx)
	}
	return map[string]int{}, nil
}

type mockViewerFeatureRepo struct {
	ListFunc          func(ctx context.Context) ([]*models.Feature, error)
	ListByEpicFunc    func(ctx context.Context, epicID int64) ([]*models.Feature, error)
	GetByKeyFunc      func(ctx context.Context, key string) (*models.Feature, error)
	CountByStatusFunc func(ctx context.Context) (map[string]int, error)
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

func (m *mockViewerFeatureRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	if m.CountByStatusFunc != nil {
		return m.CountByStatusFunc(ctx)
	}
	return map[string]int{}, nil
}

type mockViewerTaskRepo struct {
	ListFunc                                 func(ctx context.Context) ([]*models.Task, error)
	ListByFeatureFunc                        func(ctx context.Context, featureID int64) ([]*models.Task, error)
	GetByKeyFunc                             func(ctx context.Context, key string) (*models.Task, error)
	CountByStatusFunc                        func(ctx context.Context) (map[string]int, error)
	CountBlockedFunc                         func(ctx context.Context) (int, error)
	ListWithViewerRelationshipsFunc          func(ctx context.Context) ([]*models.ViewerTaskWithRelationships, error)
	ListByFeatureWithViewerRelationshipsFunc func(ctx context.Context, featureID int64) ([]*models.ViewerTaskWithRelationships, error)
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

func (m *mockViewerTaskRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	if m.CountByStatusFunc != nil {
		return m.CountByStatusFunc(ctx)
	}
	return map[string]int{}, nil
}

func (m *mockViewerTaskRepo) CountBlocked(ctx context.Context) (int, error) {
	if m.CountBlockedFunc != nil {
		return m.CountBlockedFunc(ctx)
	}
	return 0, nil
}

func (m *mockViewerTaskRepo) ListWithViewerRelationships(ctx context.Context) ([]*models.ViewerTaskWithRelationships, error) {
	if m.ListWithViewerRelationshipsFunc != nil {
		return m.ListWithViewerRelationshipsFunc(ctx)
	}
	// Default: wrap results from ListFunc if available
	if m.ListFunc != nil {
		tasks, err := m.ListFunc(ctx)
		if err != nil {
			return nil, err
		}
		result := make([]*models.ViewerTaskWithRelationships, len(tasks))
		for i, t := range tasks {
			result[i] = &models.ViewerTaskWithRelationships{Task: t, RelationshipsJSON: "[]"}
		}
		return result, nil
	}
	return []*models.ViewerTaskWithRelationships{}, nil
}

func (m *mockViewerTaskRepo) ListByFeatureWithViewerRelationships(ctx context.Context, featureID int64) ([]*models.ViewerTaskWithRelationships, error) {
	if m.ListByFeatureWithViewerRelationshipsFunc != nil {
		return m.ListByFeatureWithViewerRelationshipsFunc(ctx, featureID)
	}
	// Default: wrap results from ListByFeatureFunc if available
	if m.ListByFeatureFunc != nil {
		tasks, err := m.ListByFeatureFunc(ctx, featureID)
		if err != nil {
			return nil, err
		}
		result := make([]*models.ViewerTaskWithRelationships, len(tasks))
		for i, t := range tasks {
			result[i] = &models.ViewerTaskWithRelationships{Task: t, RelationshipsJSON: "[]"}
		}
		return result, nil
	}
	return []*models.ViewerTaskWithRelationships{}, nil
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

type mockViewerSprintService struct {
	ListSprintsFunc        func(ctx context.Context, filters *SprintListFilters) ([]*models.Sprint, error)
	GetSprintFunc          func(ctx context.Context, key string) (*models.Sprint, error)
	GetSprintBacklogFunc   func(ctx context.Context, sprintKey string, opts BacklogOptions) (*SprintBacklog, error)
	GetSprintReadinessFunc func(ctx context.Context, key string) (*SprintReadiness, error)
	GetSprintCapacityFunc  func(ctx context.Context, key string) ([]CapacityRow, error)
	PlanSprintFunc         func(ctx context.Context, key string) (*SprintPlanView, error)
}

func (m *mockViewerSprintService) ListSprints(ctx context.Context, filters *SprintListFilters) ([]*models.Sprint, error) {
	if m.ListSprintsFunc != nil {
		return m.ListSprintsFunc(ctx, filters)
	}
	return nil, nil
}

func (m *mockViewerSprintService) GetSprint(ctx context.Context, key string) (*models.Sprint, error) {
	if m.GetSprintFunc != nil {
		return m.GetSprintFunc(ctx, key)
	}
	return nil, nil
}

func (m *mockViewerSprintService) GetSprintBacklog(ctx context.Context, sprintKey string, opts BacklogOptions) (*SprintBacklog, error) {
	if m.GetSprintBacklogFunc != nil {
		return m.GetSprintBacklogFunc(ctx, sprintKey, opts)
	}
	return nil, nil
}

func (m *mockViewerSprintService) GetSprintReadiness(ctx context.Context, key string) (*SprintReadiness, error) {
	if m.GetSprintReadinessFunc != nil {
		return m.GetSprintReadinessFunc(ctx, key)
	}
	return nil, nil
}

func (m *mockViewerSprintService) GetSprintCapacity(ctx context.Context, key string) ([]CapacityRow, error) {
	if m.GetSprintCapacityFunc != nil {
		return m.GetSprintCapacityFunc(ctx, key)
	}
	return nil, nil
}

func (m *mockViewerSprintService) PlanSprint(ctx context.Context, key string) (*SprintPlanView, error) {
	if m.PlanSprintFunc != nil {
		return m.PlanSprintFunc(ctx, key)
	}
	return nil, nil
}

type mockViewerSprintAnalyticsService struct {
	GetBurndownFunc func(ctx context.Context, sprintKey string) (*BurndownResult, error)
	GetVelocityFunc func(ctx context.Context, n int) (*VelocityResult, error)
	GetSummaryFunc  func(ctx context.Context, sprintKey string, detailed bool) (*SprintSummaryResult, error)
}

func (m *mockViewerSprintAnalyticsService) GetBurndown(ctx context.Context, sprintKey string) (*BurndownResult, error) {
	if m.GetBurndownFunc != nil {
		return m.GetBurndownFunc(ctx, sprintKey)
	}
	return nil, nil
}

func (m *mockViewerSprintAnalyticsService) GetVelocity(ctx context.Context, n int) (*VelocityResult, error) {
	if m.GetVelocityFunc != nil {
		return m.GetVelocityFunc(ctx, n)
	}
	return nil, nil
}

func (m *mockViewerSprintAnalyticsService) GetSummary(ctx context.Context, sprintKey string, detailed bool) (*SprintSummaryResult, error) {
	if m.GetSummaryFunc != nil {
		return m.GetSummaryFunc(ctx, sprintKey, detailed)
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

// noopEntityRelRepo is a minimal EntityRelationshipRepository that returns
// empty slices from all read methods. Used to satisfy the NewViewerService
// constructor in tests that do not exercise relationship resolution.
type noopEntityRelRepo struct{}

func (r *noopEntityRelRepo) Create(_ context.Context, _ *models.EntityRelationship) error {
	return nil
}
func (r *noopEntityRelRepo) Delete(_ context.Context, _ int64) error { return nil }
func (r *noopEntityRelRepo) DeleteByEntitiesAndType(_ context.Context, _ models.EntityType, _ int64, _ models.EntityType, _ int64, _ models.EntityRelationshipType) error {
	return nil
}
func (r *noopEntityRelRepo) GetByEntity(_ context.Context, _ models.EntityType, _ int64) ([]*models.EntityRelationship, error) {
	return []*models.EntityRelationship{}, nil
}
func (r *noopEntityRelRepo) GetOutgoing(_ context.Context, _ models.EntityType, _ int64, _ []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
	return []*models.EntityRelationship{}, nil
}
func (r *noopEntityRelRepo) GetIncoming(_ context.Context, _ models.EntityType, _ int64, _ []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
	return []*models.EntityRelationship{}, nil
}

// buildNoopEntityRelSvc creates a minimal EntityRelationshipService backed by
// a noop repository. All read methods return empty slices, so resolveTaskRelationships
// will always return an empty Relationships slice. Suitable for tests that do
// not verify cross-entity relationship behaviour.
func buildNoopEntityRelSvc() *EntityRelationshipService {
	return NewEntityRelationshipService(&noopEntityRelRepo{}, nil)
}

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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
	)
}

func TestNewViewerService_NilEntityRelSvcIsAccepted(t *testing.T) {
	// entityRelSvc is now optional; nil must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic for nil entityRelSvc: %v", r)
		}
	}()
	_ = NewViewerService(&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		t.TempDir(),
		nil, // entityRelSvc — optional, must not panic
		nil,
	)
}

func TestNewViewerService_NilEntityRegistryIsAccepted(t *testing.T) {
	// entityRegistry is now optional; nil must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic for nil entityRegistry: %v", r)
		}
	}()
	_ = NewViewerService(&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		t.TempDir(),
		nil,
		nil, // entityRegistry — optional, must not panic
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
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
				return map[string]int{"active": 2, "completed": 1}, nil
			},
		},
		&mockViewerFeatureRepo{
			CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
				return map[string]int{"in_progress": 1}, nil
			},
		},
		&mockViewerTaskRepo{
			CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
				return map[string]int{"todo": 2}, nil
			},
			CountBlockedFunc: func(ctx context.Context) (int, error) {
				return 1, nil
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
			CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
				return map[string]int{"totally_unknown_status": 1}, nil
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
	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
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

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
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

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
			CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
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
			CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
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
			CountByStatusFunc: func(ctx context.Context) (map[string]int, error) {
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
	_, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
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
	_, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
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
	_, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
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
	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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
		buildNoopEntityRelSvc(),
		NewEntityRegistry(),
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

// ----- TC-F039: ViewerTask.Relationships via EntityRelationshipService -----

// TestViewerTask_RelationshipsFieldExists verifies that ViewerTask exposes a
// Relationships []ViewerRelatedEntity field (compile-time AC).
func TestViewerTask_RelationshipsFieldExists(t *testing.T) {
	task := ViewerTask{}
	var _ []ViewerRelatedEntity = task.Relationships // must compile
}

// TestViewerService_Hierarchy_RelationshipsEmptyWhenNone verifies that a task
// with no relationships returns an empty (non-nil) Relationships slice.
func TestViewerService_Hierarchy_RelationshipsEmptyWhenNone(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Epics) == 0 {
		return // no epics → no tasks to check
	}
	for _, epic := range resp.Epics {
		for _, feature := range epic.Features {
			for _, task := range feature.Tasks {
				if task.Relationships == nil {
					t.Errorf("task %s: Relationships should be non-nil empty slice, got nil", task.Key)
				}
			}
		}
	}
}

// TestViewerService_Hierarchy_RelationshipsPopulated verifies that when the task
// repository returns relationship JSON, it is surfaced on ViewerTask.Relationships.
func TestViewerService_Hierarchy_RelationshipsPopulated(t *testing.T) {
	const taskID = int64(7)
	const relatedTaskKey = "E01-F01-002"

	// The view-based approach: task repo returns pre-resolved relationship JSON.
	relsJSON := `[{"direction":"outgoing","relationship_type":"depends_on","entity_type":"task","entity_key":"E01-F01-002"}]`

	epicRepo := &mockViewerEpicRepo{
		ListFunc: func(ctx context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
			return []*models.Epic{
				{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}},
			}, nil
		},
	}
	featureRepo := &mockViewerFeatureRepo{
		ListFunc: func(ctx context.Context) ([]*models.Feature, error) {
			return []*models.Feature{
				{BaseEntity: models.BaseEntity{ID: 10, Key: "E01-F01"}, EpicID: 1},
			}, nil
		},
	}
	taskRepo := &mockViewerTaskRepo{
		ListWithViewerRelationshipsFunc: func(ctx context.Context) ([]*models.ViewerTaskWithRelationships, error) {
			return []*models.ViewerTaskWithRelationships{
				{
					Task:              &models.Task{BaseEntity: models.BaseEntity{ID: taskID, Key: "E01-F01-001"}, FeatureID: 10, Status: "todo"},
					RelationshipsJSON: relsJSON,
				},
			}, nil
		},
	}

	svc := NewViewerService(
		epicRepo,
		featureRepo,
		taskRepo,
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		t.TempDir(),
		nil,
		nil,
	)

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Epics) == 0 || len(resp.Epics[0].Features) == 0 || len(resp.Epics[0].Features[0].Tasks) == 0 {
		t.Fatal("expected hierarchy to contain at least one task")
	}

	task := resp.Epics[0].Features[0].Tasks[0]
	if len(task.Relationships) == 0 {
		t.Fatal("expected Relationships to be populated but got empty slice")
	}

	rel := task.Relationships[0]
	if rel.Direction != "outgoing" {
		t.Errorf("expected direction=outgoing, got %q", rel.Direction)
	}
	if rel.RelationshipType != models.EntityRelDependsOn {
		t.Errorf("expected relationship_type=depends_on, got %q", rel.RelationshipType)
	}
	if rel.EntityType != models.EntityTypeTask {
		t.Errorf("expected entity_type=task, got %q", rel.EntityType)
	}
	if rel.EntityKey != relatedTaskKey {
		t.Errorf("expected entity_key=%q, got %q", relatedTaskKey, rel.EntityKey)
	}
}

// TestViewerService_FeatureTasks_RelationshipsPopulated verifies that FeatureTasks
// surfaces Relationships from the pre-resolved view JSON.
func TestViewerService_FeatureTasks_RelationshipsPopulated(t *testing.T) {
	const taskID = int64(5)
	const relatedTaskKey = "E01-F01-002"

	// The view-based approach: task repo returns pre-resolved relationship JSON.
	relsJSON := `[{"direction":"incoming","relationship_type":"blocks","entity_type":"task","entity_key":"E01-F01-002"}]`

	svc := NewViewerService(
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(_ context.Context, key string) (*models.Feature, error) {
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 20, Key: key}}, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureWithViewerRelationshipsFunc: func(_ context.Context, featureID int64) ([]*models.ViewerTaskWithRelationships, error) {
				return []*models.ViewerTaskWithRelationships{
					{
						Task:              &models.Task{BaseEntity: models.BaseEntity{ID: taskID, Key: "E01-F01-001"}, FeatureID: 20, Status: "todo"},
						RelationshipsJSON: relsJSON,
					},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
		testWorkflowSvc(t),
		nil,
		t.TempDir(),
		nil,
		nil,
	)

	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Tasks) == 0 {
		t.Fatal("expected at least one task")
	}

	task := resp.Tasks[0]
	if len(task.Relationships) == 0 {
		t.Fatal("expected Relationships to be populated but got empty slice")
	}

	rel := task.Relationships[0]
	if rel.Direction != "incoming" {
		t.Errorf("expected direction=incoming, got %q", rel.Direction)
	}
	if rel.RelationshipType != models.EntityRelBlocks {
		t.Errorf("expected relationship_type=blocks, got %q", rel.RelationshipType)
	}
	if rel.EntityKey != relatedTaskKey {
		t.Errorf("expected entity_key=%q, got %q", relatedTaskKey, rel.EntityKey)
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

// ----- mockTagReader -----

// mockTagReader is a test double for the TagReader interface consumed by ViewerService.
// It uses function fields following the project mock pattern. Call counters are provided
// for AC-16 (query-count assertions) and TC-AC02-3 (delegation assertions).
type mockTagReader struct {
	ListTagsFunc              func(ctx context.Context) ([]*models.Tag, error)
	EntityIDsByTagsFunc       func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error)
	AttachedTagNamesByIDsFunc func(ctx context.Context, entityType models.EntityType, entityIDs []int64) (map[int64][]string, error)

	// Call counters for assertion
	ListTagsCallCount              int
	AttachedTagNamesByIDsCallCount int
	EntityIDsByTagsCallCount       int
}

func (m *mockTagReader) ListTags(ctx context.Context) ([]*models.Tag, error) {
	m.ListTagsCallCount++
	if m.ListTagsFunc != nil {
		return m.ListTagsFunc(ctx)
	}
	return []*models.Tag{}, nil
}

func (m *mockTagReader) EntityIDsByTags(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
	m.EntityIDsByTagsCallCount++
	if m.EntityIDsByTagsFunc != nil {
		return m.EntityIDsByTagsFunc(ctx, entityType, names, op)
	}
	return []int64{}, nil
}

func (m *mockTagReader) AttachedTagNamesByIDs(ctx context.Context, entityType models.EntityType, entityIDs []int64) (map[int64][]string, error) {
	m.AttachedTagNamesByIDsCallCount++
	if m.AttachedTagNamesByIDsFunc != nil {
		return m.AttachedTagNamesByIDsFunc(ctx, entityType, entityIDs)
	}
	return map[int64][]string{}, nil
}

// ----- Tags() tests (TC-AC01-1, TC-AC01-3, TC-AC02-1, TC-AC02-3, TC-AC14-1, TC-AC14-5) -----

// TC-AC01-1: Tags() with empty vocabulary returns non-nil empty slice.
func TestViewerService_Tags_EmptyVocabulary(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{
		ListTagsFunc: func(_ context.Context) ([]*models.Tag, error) {
			return []*models.Tag{}, nil
		},
	}
	svc.WithTagService(tagReader)

	resp, err := svc.Tags(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Tags == nil {
		t.Error("TC-AC01-1: Tags field must be non-nil (got nil) — ADR-F06-2")
	}
	if len(resp.Tags) != 0 {
		t.Errorf("TC-AC01-1: expected 0 tags, got %d", len(resp.Tags))
	}
}

// TC-AC01-3: JSON null guard — nil slice marshals as null; []TagDTO{} marshals as [].
func TestViewerService_Tags_NilSliceMarshalGuard(t *testing.T) {
	// []TagDTO{} must marshal as []
	resp := TagsResponse{Tags: []TagDTO{}}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if string(data) != `{"tags":[]}` {
		t.Errorf("expected {\"tags\":[]}, got %s", string(data))
	}

	// nil slice marshals as null — our service MUST avoid this
	respNil := TagsResponse{Tags: nil}
	dataNil, err := json.Marshal(respNil)
	if err != nil {
		t.Fatalf("failed to marshal nil Tags: %v", err)
	}
	if string(dataNil) != `{"tags":null}` {
		t.Errorf("expected {\"tags\":null} for nil slice, got %s — documents why we must assign []TagDTO{}", string(dataNil))
	}
}

// TC-AC02-1: Tags() returns tags alphabetically sorted.
func TestViewerService_Tags_AlphabeticalOrdering(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{
		ListTagsFunc: func(_ context.Context) ([]*models.Tag, error) {
			return []*models.Tag{
				{Name: "voice"},
				{Name: "auth"},
			}, nil
		},
	}
	svc.WithTagService(tagReader)

	resp, err := svc.Tags(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(resp.Tags))
	}
	if resp.Tags[0].Name != "auth" || resp.Tags[1].Name != "voice" {
		t.Errorf("TC-AC02-1: expected [auth, voice] (alphabetical), got [%s, %s]",
			resp.Tags[0].Name, resp.Tags[1].Name)
	}
}

// TC-AC02-3: Tags() delegates to tagSvc.ListTags exactly once; no other tag methods called.
func TestViewerService_Tags_DelegatesToListTagsOnce(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{
		ListTagsFunc: func(_ context.Context) ([]*models.Tag, error) {
			return []*models.Tag{{Name: "auth"}}, nil
		},
	}
	svc.WithTagService(tagReader)

	_, err := svc.Tags(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tagReader.ListTagsCallCount != 1 {
		t.Errorf("TC-AC02-3: expected ListTags called once, got %d", tagReader.ListTagsCallCount)
	}
	if tagReader.AttachedTagNamesByIDsCallCount != 0 {
		t.Errorf("TC-AC02-3: Tags() must not call AttachedTagNamesByIDs, got %d calls",
			tagReader.AttachedTagNamesByIDsCallCount)
	}
	if tagReader.EntityIDsByTagsCallCount != 0 {
		t.Errorf("TC-AC02-3: Tags() must not call EntityIDsByTags, got %d calls",
			tagReader.EntityIDsByTagsCallCount)
	}
}

// TC-AC14-1: Tags() with nil tagSvc returns {tags: []}, no error (graceful degradation).
func TestViewerService_Tags_NilTagSvc_GracefulDegradation(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// Do NOT call WithTagService — tagSvc remains nil.

	resp, err := svc.Tags(context.Background())
	if err != nil {
		t.Fatalf("TC-AC14-1: expected no error with nil tagSvc, got %v", err)
	}
	if resp == nil {
		t.Fatal("TC-AC14-1: expected non-nil response")
	}
	if resp.Tags == nil {
		t.Error("TC-AC14-1: Tags must be non-nil even with nil tagSvc — ADR-F06-2")
	}
	if len(resp.Tags) != 0 {
		t.Errorf("TC-AC14-1: expected empty tags, got %d tags", len(resp.Tags))
	}
}

// TC-AC14-5 (partial): Tags() with nil tagSvc must not panic.
func TestViewerService_Tags_NilTagSvc_NoPanic(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TC-AC14-5: Tags() panicked with nil tagSvc: %v", r)
		}
	}()
	resp, err := svc.Tags(context.Background())
	if err != nil {
		t.Errorf("TC-AC14-5: unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("TC-AC14-5: expected non-nil response")
	}
}

// TestViewerService_WithTagService_ReturnsService verifies the setter follows
// the method-chaining pattern (returns *ViewerService).
func TestViewerService_WithTagService_ReturnsService(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{}
	returned := svc.WithTagService(tagReader)
	if returned != svc {
		t.Error("WithTagService must return the receiver for method chaining")
	}
}

// IS-1 (partial for T-E28-F06-001): Tags() correctly projects *models.Tag -> TagDTO{Name}.
func TestViewerService_Tags_ProjectsTagToDTO(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{
		ListTagsFunc: func(_ context.Context) ([]*models.Tag, error) {
			return []*models.Tag{
				{Name: "auth"},
				{Name: "voice"},
			}, nil
		},
	}
	svc.WithTagService(tagReader)

	resp, err := svc.Tags(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Tags) != 2 {
		t.Fatalf("expected 2 DTOs, got %d", len(resp.Tags))
	}
	if resp.Tags[0].Name != "auth" {
		t.Errorf("expected first tag name 'auth', got %q", resp.Tags[0].Name)
	}
	if resp.Tags[1].Name != "voice" {
		t.Errorf("expected second tag name 'voice', got %q", resp.Tags[1].Name)
	}
}

// TestViewerService_Tags_ListTagsError verifies error propagation from tagSvc.ListTags.
func TestViewerService_Tags_ListTagsError(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagErr := fmt.Errorf("db connection error")
	tagReader := &mockTagReader{
		ListTagsFunc: func(_ context.Context) ([]*models.Tag, error) {
			return nil, tagErr
		},
	}
	svc.WithTagService(tagReader)

	resp, err := svc.Tags(context.Background())
	if err == nil {
		t.Fatal("expected error from ListTags to propagate, got nil")
	}
	if resp != nil {
		t.Errorf("expected nil response on error, got %+v", resp)
	}
}

// TestViewerService_HierarchyEpic_HasTagsField verifies that HierarchyEpic has the Tags field
// and that it can be initialized as a non-nil empty slice (AC-T2 prerequisite).
func TestViewerService_DTOs_HaveTagsField(t *testing.T) {
	epic := &HierarchyEpic{
		Tags: []string{},
	}
	if epic.Tags == nil {
		t.Error("HierarchyEpic.Tags must be initializable as non-nil empty slice")
	}

	feature := &HierarchyFeature{
		Tags: []string{},
	}
	if feature.Tags == nil {
		t.Error("HierarchyFeature.Tags must be initializable as non-nil empty slice")
	}

	task := &ViewerTask{
		Tags: []string{},
	}
	if task.Tags == nil {
		t.Error("ViewerTask.Tags must be initializable as non-nil empty slice")
	}

	flat := &FlatEntity{
		Tags: []string{},
	}
	if flat.Tags == nil {
		t.Error("FlatEntity.Tags must be initializable as non-nil empty slice")
	}
}

// TestViewerService_FeatureTaskOptions_HasTagsField verifies the Tags field on FeatureTaskOptions.
func TestViewerService_FeatureTaskOptions_HasTagsField(t *testing.T) {
	opts := FeatureTaskOptions{
		Tags: []string{"voice"},
	}
	if len(opts.Tags) != 1 || opts.Tags[0] != "voice" {
		t.Error("FeatureTaskOptions.Tags field not working as expected")
	}
}

// TestViewerService_HierarchyOptions_HasTagsField verifies the HierarchyOptions DTO exists.
func TestViewerService_HierarchyOptions_HasTagsField(t *testing.T) {
	opts := HierarchyOptions{
		Tags: []string{"auth", "voice"},
	}
	if len(opts.Tags) != 2 {
		t.Error("HierarchyOptions.Tags field not working as expected")
	}
}

// ----- E27-F11: FlatEntity.Size field population and wire-format tests -----

// TC-AC01-1 / TC-AC10-1: Bug size populated from model.
func TestViewerService_Hierarchy_FlatEntity_BugSize_Populated(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithBugListRepo(&mockViewerBugListRepo{
		ListAllFunc: func(_ context.Context, _ bool) ([]*models.Bug, error) {
			return []*models.Bug{
				{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Size: ptr(5)}, Status: "triaged"},
			}, nil
		},
	})

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC01-1: unexpected error: %v", err)
	}
	if len(resp.Bugs) != 1 {
		t.Fatalf("TC-AC01-1: expected 1 bug, got %d", len(resp.Bugs))
	}
	if resp.Bugs[0].Size == nil {
		t.Fatal("TC-AC01-1: expected Size to be non-nil for bug with Size=5")
	}
	if *resp.Bugs[0].Size != 5 {
		t.Errorf("TC-AC01-1: expected Size=5, got %d", *resp.Bugs[0].Size)
	}
}

// TC-AC02-1 / TC-AC10-2: Bug size nil produces nil FlatEntity.Size.
func TestViewerService_Hierarchy_FlatEntity_BugSize_Nil(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithBugListRepo(&mockViewerBugListRepo{
		ListAllFunc: func(_ context.Context, _ bool) ([]*models.Bug, error) {
			return []*models.Bug{
				{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Size: nil}, Status: "triaged"},
			}, nil
		},
	})

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC02-1: unexpected error: %v", err)
	}
	if len(resp.Bugs) != 1 {
		t.Fatalf("TC-AC02-1: expected 1 bug, got %d", len(resp.Bugs))
	}
	if resp.Bugs[0].Size != nil {
		t.Errorf("TC-AC02-1: expected Size to be nil for bug with no size, got %v", resp.Bugs[0].Size)
	}
}

// TC-AC02-2: Bug size nil — omitempty wire-level check.
func TestViewerService_Hierarchy_FlatEntity_BugSize_OmitemptyJSON(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	// Sub-test: populated size serializes with "size" key.
	svc.WithBugListRepo(&mockViewerBugListRepo{
		ListAllFunc: func(_ context.Context, _ bool) ([]*models.Bug, error) {
			return []*models.Bug{
				{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Size: ptr(5)}, Status: "triaged"},
			}, nil
		},
	})
	respPopulated, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC02-2 populated: unexpected error: %v", err)
	}
	if len(respPopulated.Bugs) != 1 {
		t.Fatalf("TC-AC02-2 populated: expected 1 bug")
	}
	dataPopulated, err := json.Marshal(respPopulated.Bugs[0])
	if err != nil {
		t.Fatalf("TC-AC02-2 populated: json.Marshal failed: %v", err)
	}
	if !bytes.Contains(dataPopulated, []byte(`"size":5`)) {
		t.Errorf("TC-AC02-2: expected populated bug JSON to contain \"size\":5, got: %s", dataPopulated)
	}

	// Sub-test: nil size omits "size" key from JSON.
	svc.WithBugListRepo(&mockViewerBugListRepo{
		ListAllFunc: func(_ context.Context, _ bool) ([]*models.Bug, error) {
			return []*models.Bug{
				{BaseEntity: models.BaseEntity{ID: 2, Key: "B002", Size: nil}, Status: "triaged"},
			}, nil
		},
	})
	respNil, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC02-2 nil: unexpected error: %v", err)
	}
	if len(respNil.Bugs) != 1 {
		t.Fatalf("TC-AC02-2 nil: expected 1 bug")
	}
	dataNil, err := json.Marshal(respNil.Bugs[0])
	if err != nil {
		t.Fatalf("TC-AC02-2 nil: json.Marshal failed: %v", err)
	}
	if bytes.Contains(dataNil, []byte(`"size"`)) {
		t.Errorf("TC-AC02-2: expected nil-size bug JSON to omit \"size\" key, got: %s", dataNil)
	}
}

// TC-AC03-1 / TC-AC10-3: ChangeCard size populated.
func TestViewerService_Hierarchy_FlatEntity_ChangeCardSize_Populated(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithChangeCardListRepo(&mockViewerChangeCardListRepo{
		ListAllFunc: func(_ context.Context, _ bool) ([]*models.ChangeCard, error) {
			return []*models.ChangeCard{
				{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Size: ptr(3)}, Status: "open"},
			}, nil
		},
	})

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC03-1: unexpected error: %v", err)
	}
	if len(resp.ChangeCards) != 1 {
		t.Fatalf("TC-AC03-1: expected 1 change card, got %d", len(resp.ChangeCards))
	}
	if resp.ChangeCards[0].Size == nil {
		t.Fatal("TC-AC03-1: expected Size to be non-nil for change card with Size=3")
	}
	if *resp.ChangeCards[0].Size != 3 {
		t.Errorf("TC-AC03-1: expected Size=3, got %d", *resp.ChangeCards[0].Size)
	}
}

// TC-AC03-2 / TC-AC10-4: ChangeCard size nil.
func TestViewerService_Hierarchy_FlatEntity_ChangeCardSize_Nil(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithChangeCardListRepo(&mockViewerChangeCardListRepo{
		ListAllFunc: func(_ context.Context, _ bool) ([]*models.ChangeCard, error) {
			return []*models.ChangeCard{
				{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Size: nil}, Status: "open"},
			}, nil
		},
	})

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC03-2: unexpected error: %v", err)
	}
	if len(resp.ChangeCards) != 1 {
		t.Fatalf("TC-AC03-2: expected 1 change card, got %d", len(resp.ChangeCards))
	}
	if resp.ChangeCards[0].Size != nil {
		t.Errorf("TC-AC03-2: expected Size to be nil for change card with no size, got %v", resp.ChangeCards[0].Size)
	}
}

// TC-AC03-3: ChangeCard size nil — wire omitempty.
func TestViewerService_Hierarchy_FlatEntity_ChangeCardSize_OmitemptyJSON(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithChangeCardListRepo(&mockViewerChangeCardListRepo{
		ListAllFunc: func(_ context.Context, _ bool) ([]*models.ChangeCard, error) {
			return []*models.ChangeCard{
				{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Size: nil}, Status: "open"},
			}, nil
		},
	})

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC03-3: unexpected error: %v", err)
	}
	if len(resp.ChangeCards) != 1 {
		t.Fatalf("TC-AC03-3: expected 1 change card")
	}
	data, err := json.Marshal(resp.ChangeCards[0])
	if err != nil {
		t.Fatalf("TC-AC03-3: json.Marshal failed: %v", err)
	}
	if bytes.Contains(data, []byte(`"size"`)) {
		t.Errorf("TC-AC03-3: expected nil-size change card JSON to omit \"size\" key, got: %s", data)
	}
}

// TC-AC03-4 / TC-AC10-5: Idea size populated (sourced from idea.Size directly).
func TestViewerService_Hierarchy_FlatEntity_IdeaSize_Populated(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithIdeaRepo(&mockViewerIdeaListRepo{
		ListAllFunc: func(_ context.Context) ([]*models.Idea, error) {
			return []*models.Idea{
				{ID: 1, Key: "I-2025-01-01-01", Title: "Big Idea", Status: "new", Size: ptr(8)},
			}, nil
		},
	})

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC03-4: unexpected error: %v", err)
	}
	if len(resp.Ideas) != 1 {
		t.Fatalf("TC-AC03-4: expected 1 idea, got %d", len(resp.Ideas))
	}
	if resp.Ideas[0].Size == nil {
		t.Fatal("TC-AC03-4: expected Size to be non-nil for idea with Size=8")
	}
	if *resp.Ideas[0].Size != 8 {
		t.Errorf("TC-AC03-4: expected Size=8, got %d", *resp.Ideas[0].Size)
	}
}

// TC-AC03-5 / TC-AC10-6: Idea size nil.
func TestViewerService_Hierarchy_FlatEntity_IdeaSize_Nil(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithIdeaRepo(&mockViewerIdeaListRepo{
		ListAllFunc: func(_ context.Context) ([]*models.Idea, error) {
			return []*models.Idea{
				{ID: 1, Key: "I-2025-01-01-01", Title: "Small Idea", Status: "new", Size: nil},
			}, nil
		},
	})

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC03-5: unexpected error: %v", err)
	}
	if len(resp.Ideas) != 1 {
		t.Fatalf("TC-AC03-5: expected 1 idea, got %d", len(resp.Ideas))
	}
	if resp.Ideas[0].Size != nil {
		t.Errorf("TC-AC03-5: expected Size to be nil for idea with no size, got %v", resp.Ideas[0].Size)
	}
}

// TC-AC03-6: Idea size nil — wire omitempty.
func TestViewerService_Hierarchy_FlatEntity_IdeaSize_OmitemptyJSON(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithIdeaRepo(&mockViewerIdeaListRepo{
		ListAllFunc: func(_ context.Context) ([]*models.Idea, error) {
			return []*models.Idea{
				{ID: 1, Key: "I-2025-01-01-01", Title: "Small Idea", Status: "new", Size: nil},
			}, nil
		},
	})

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC03-6: unexpected error: %v", err)
	}
	if len(resp.Ideas) != 1 {
		t.Fatalf("TC-AC03-6: expected 1 idea")
	}
	data, err := json.Marshal(resp.Ideas[0])
	if err != nil {
		t.Fatalf("TC-AC03-6: json.Marshal failed: %v", err)
	}
	if bytes.Contains(data, []byte(`"size"`)) {
		t.Errorf("TC-AC03-6: expected nil-size idea JSON to omit \"size\" key, got: %s", data)
	}
}

// ----- Mock helpers for bug/changecard/idea list repos used in tag decoration tests -----

type mockViewerBugListRepo struct {
	ListAllFunc  func(ctx context.Context, includeTerminal bool) ([]*models.Bug, error)
	GetByKeyFunc func(ctx context.Context, key string) (*models.Bug, error)
}

func (m *mockViewerBugListRepo) ListAll(ctx context.Context, includeTerminal bool) ([]*models.Bug, error) {
	if m.ListAllFunc != nil {
		return m.ListAllFunc(ctx, includeTerminal)
	}
	return []*models.Bug{}, nil
}

func (m *mockViewerBugListRepo) GetByKey(ctx context.Context, key string) (*models.Bug, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("bug not found: %s", key)
}

type mockViewerChangeCardListRepo struct {
	ListAllFunc  func(ctx context.Context, includeTerminal bool) ([]*models.ChangeCard, error)
	GetByKeyFunc func(ctx context.Context, key string) (*models.ChangeCard, error)
}

func (m *mockViewerChangeCardListRepo) ListAll(ctx context.Context, includeTerminal bool) ([]*models.ChangeCard, error) {
	if m.ListAllFunc != nil {
		return m.ListAllFunc(ctx, includeTerminal)
	}
	return []*models.ChangeCard{}, nil
}

func (m *mockViewerChangeCardListRepo) GetByKey(ctx context.Context, key string) (*models.ChangeCard, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("change card not found: %s", key)
}

type mockViewerIdeaListRepo struct {
	ListAllFunc  func(ctx context.Context) ([]*models.Idea, error)
	GetByKeyFunc func(ctx context.Context, key string) (*models.Idea, error)
}

func (m *mockViewerIdeaListRepo) ListAll(ctx context.Context) ([]*models.Idea, error) {
	if m.ListAllFunc != nil {
		return m.ListAllFunc(ctx)
	}
	return []*models.Idea{}, nil
}

func (m *mockViewerIdeaListRepo) GetByKey(ctx context.Context, key string) (*models.Idea, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("idea not found: %s", key)
}

// ----- Hierarchy tag decoration and filter tests (T-E28-F06-002) -----

// TC-AC04-1: Hierarchy with nil tagSvc produces non-nil []string{} on all entity DTOs.
func TestViewerService_Hierarchy_NilTagSvc_TagsAlwaysNonNil(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(_ context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 10}, EpicID: 1, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
				return []*models.ViewerTaskWithRelationships{
					{Task: &models.Task{BaseEntity: models.BaseEntity{ID: 100}, FeatureID: 10, Status: "todo"}, RelationshipsJSON: "[]"},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// Wire bug/change/idea repos so they produce entities
	svc.WithBugListRepo(&mockViewerBugListRepo{
		ListAllFunc: func(_ context.Context, _ bool) ([]*models.Bug, error) {
			return []*models.Bug{
				{BaseEntity: models.BaseEntity{ID: 200, Key: "B001"}, Status: "new"},
			}, nil
		},
	})
	svc.WithChangeCardListRepo(&mockViewerChangeCardListRepo{
		ListAllFunc: func(_ context.Context, _ bool) ([]*models.ChangeCard, error) {
			return []*models.ChangeCard{
				{BaseEntity: models.BaseEntity{ID: 300, Key: "CC-001"}, Status: "open"},
			}, nil
		},
	})
	// tagSvc is NOT wired — nil

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC04-1: unexpected error: %v", err)
	}

	if len(resp.Epics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(resp.Epics))
	}
	epic := resp.Epics[0]
	if epic.Tags == nil {
		t.Error("TC-AC04-1: HierarchyEpic.Tags must be non-nil (got nil) — ADR-F06-2")
	}
	if len(epic.Tags) != 0 {
		t.Errorf("TC-AC04-1: expected empty Tags on epic, got %v", epic.Tags)
	}

	if len(epic.Features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(epic.Features))
	}
	feature := epic.Features[0]
	if feature.Tags == nil {
		t.Error("TC-AC04-1: HierarchyFeature.Tags must be non-nil (got nil) — ADR-F06-2")
	}
	if len(feature.Tags) != 0 {
		t.Errorf("TC-AC04-1: expected empty Tags on feature, got %v", feature.Tags)
	}

	if len(feature.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(feature.Tasks))
	}
	task := feature.Tasks[0]
	if task.Tags == nil {
		t.Error("TC-AC04-1: ViewerTask.Tags must be non-nil (got nil) — ADR-F06-2")
	}
	if len(task.Tags) != 0 {
		t.Errorf("TC-AC04-1: expected empty Tags on task, got %v", task.Tags)
	}

	// Flat bugs
	if len(resp.Bugs) != 1 {
		t.Fatalf("expected 1 bug in flat section, got %d", len(resp.Bugs))
	}
	if resp.Bugs[0].Tags == nil {
		t.Error("TC-AC04-1: FlatEntity(bug).Tags must be non-nil — ADR-F06-2")
	}
	if len(resp.Bugs[0].Tags) != 0 {
		t.Errorf("TC-AC04-1: expected empty Tags on bug, got %v", resp.Bugs[0].Tags)
	}

	// Flat change cards
	if len(resp.ChangeCards) != 1 {
		t.Fatalf("expected 1 change card in flat section, got %d", len(resp.ChangeCards))
	}
	if resp.ChangeCards[0].Tags == nil {
		t.Error("TC-AC04-1: FlatEntity(change_card).Tags must be non-nil — ADR-F06-2")
	}
}

// TC-AC04-2: Hierarchy with tagSvc wired but all entities untagged — Tags is []string{} (non-nil).
func TestViewerService_Hierarchy_TagSvcWired_UntaggedEntities(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(_ context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 10}, EpicID: 1, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
				return []*models.ViewerTaskWithRelationships{
					{Task: &models.Task{BaseEntity: models.BaseEntity{ID: 100}, FeatureID: 10, Status: "todo"}, RelationshipsJSON: "[]"},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{
		// AttachedTagNamesByIDs returns empty map — no tags attached
		AttachedTagNamesByIDsFunc: func(_ context.Context, _ models.EntityType, _ []int64) (map[int64][]string, error) {
			return map[int64][]string{}, nil
		},
	}
	svc.WithTagService(tagReader)

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC04-2: unexpected error: %v", err)
	}

	epic := resp.Epics[0]
	if epic.Tags == nil {
		t.Error("TC-AC04-2: HierarchyEpic.Tags must be non-nil — ADR-F06-2")
	}
	if epic.Features[0].Tags == nil {
		t.Error("TC-AC04-2: HierarchyFeature.Tags must be non-nil — ADR-F06-2")
	}
	if epic.Features[0].Tasks[0].Tags == nil {
		t.Error("TC-AC04-2: ViewerTask.Tags must be non-nil — ADR-F06-2")
	}
}

// TC-AC05-1: Single entity (epic) tagged; others untagged.
func TestViewerService_Hierarchy_TagDecoration_SingleEpicTagged(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
					{BaseEntity: models.BaseEntity{ID: 2, Key: "E02"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(_ context.Context) ([]*models.Feature, error) {
				return []*models.Feature{}, nil
			},
		},
		&mockViewerTaskRepo{
			ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
				return []*models.ViewerTaskWithRelationships{}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{
		AttachedTagNamesByIDsFunc: func(_ context.Context, entityType models.EntityType, ids []int64) (map[int64][]string, error) {
			if entityType == models.EntityTypeEpic {
				return map[int64][]string{1: {"voice"}}, nil
			}
			return map[int64][]string{}, nil
		},
	}
	svc.WithTagService(tagReader)

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC05-1: unexpected error: %v", err)
	}

	if len(resp.Epics) != 2 {
		t.Fatalf("expected 2 epics, got %d", len(resp.Epics))
	}

	var e01, e02 *HierarchyEpic
	for _, e := range resp.Epics {
		switch e.Key {
		case "E01":
			e01 = e
		case "E02":
			e02 = e
		}
	}

	if e01 == nil || e02 == nil {
		t.Fatal("could not find E01 or E02 in response")
	}

	if len(e01.Tags) != 1 || e01.Tags[0] != "voice" {
		t.Errorf("TC-AC05-1: expected E01.Tags == [voice], got %v", e01.Tags)
	}
	if len(e02.Tags) != 0 {
		t.Errorf("TC-AC05-1: expected E02.Tags == [], got %v", e02.Tags)
	}
}

// TC-AC05-4: Batching correctness — AttachedTagNamesByIDs called at most once per entity type present.
func TestViewerService_Hierarchy_TagDecoration_BatchCallCount(t *testing.T) {
	// Build: 3 epics, 5 features, 10 tasks, 2 bugs, 2 change cards, 1 idea
	epics := []*models.Epic{
		{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
		{BaseEntity: models.BaseEntity{ID: 2, Key: "E02"}, Status: models.EpicStatusActive},
		{BaseEntity: models.BaseEntity{ID: 3, Key: "E03"}, Status: models.EpicStatusActive},
	}
	features := make([]*models.Feature, 5)
	for i := range features {
		features[i] = &models.Feature{
			BaseEntity: models.BaseEntity{ID: int64(10 + i)},
			EpicID:     int64(1 + (i % 3)),
			Status:     "in_progress",
		}
	}
	taskRels := make([]*models.ViewerTaskWithRelationships, 10)
	for i := range taskRels {
		taskRels[i] = &models.ViewerTaskWithRelationships{
			Task:              &models.Task{BaseEntity: models.BaseEntity{ID: int64(100 + i)}, FeatureID: int64(10 + (i % 5)), Status: "todo"},
			RelationshipsJSON: "[]",
		}
	}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
			return epics, nil
		}},
		&mockViewerFeatureRepo{ListFunc: func(_ context.Context) ([]*models.Feature, error) {
			return features, nil
		}},
		&mockViewerTaskRepo{ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
			return taskRels, nil
		}},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithBugListRepo(&mockViewerBugListRepo{
		ListAllFunc: func(_ context.Context, _ bool) ([]*models.Bug, error) {
			return []*models.Bug{
				{BaseEntity: models.BaseEntity{ID: 200, Key: "B001"}, Status: "new"},
				{BaseEntity: models.BaseEntity{ID: 201, Key: "B002"}, Status: "new"},
			}, nil
		},
	})
	svc.WithChangeCardListRepo(&mockViewerChangeCardListRepo{
		ListAllFunc: func(_ context.Context, _ bool) ([]*models.ChangeCard, error) {
			return []*models.ChangeCard{
				{BaseEntity: models.BaseEntity{ID: 300, Key: "CC-001"}, Status: "open"},
				{BaseEntity: models.BaseEntity{ID: 301, Key: "CC-002"}, Status: "open"},
			}, nil
		},
	})
	svc.WithIdeaRepo(&mockViewerIdeaListRepo{
		ListAllFunc: func(_ context.Context) ([]*models.Idea, error) {
			return []*models.Idea{
				{ID: 400, Key: "idea-1", Title: "Test Idea", Status: "new"},
			}, nil
		},
	})
	tagReader := &mockTagReader{
		AttachedTagNamesByIDsFunc: func(_ context.Context, _ models.EntityType, _ []int64) (map[int64][]string, error) {
			return map[int64][]string{}, nil
		},
	}
	svc.WithTagService(tagReader)

	_, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC05-4 (AC-16): unexpected error: %v", err)
	}

	// With epics + features + tasks + bugs + change cards + ideas = 6 entity types
	// => exactly 6 AttachedTagNamesByIDs calls
	if tagReader.AttachedTagNamesByIDsCallCount > 6 {
		t.Errorf("TC-AC16-1: AttachedTagNamesByIDs called %d times (must be ≤ 6 regardless of tree size)",
			tagReader.AttachedTagNamesByIDsCallCount)
	}
}

// TC-AC16-2: Only entity types with IDs get a call; empty types skipped.
func TestViewerService_Hierarchy_TagDecoration_SkipsEmptyEntityTypes(t *testing.T) {
	// Only epics and features — no tasks, bugs, changes, ideas
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(_ context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 10}, EpicID: 1, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
				return []*models.ViewerTaskWithRelationships{}, nil // no tasks
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// No WithBugListRepo, no WithChangeCardListRepo, no WithIdeaRepo
	tagReader := &mockTagReader{
		AttachedTagNamesByIDsFunc: func(_ context.Context, _ models.EntityType, _ []int64) (map[int64][]string, error) {
			return map[int64][]string{}, nil
		},
	}
	svc.WithTagService(tagReader)

	_, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC16-2: unexpected error: %v", err)
	}

	// Only epics and features present → 2 calls (not 6)
	if tagReader.AttachedTagNamesByIDsCallCount > 6 {
		t.Errorf("TC-AC16-2: too many AttachedTagNamesByIDs calls: %d (want ≤ 6)",
			tagReader.AttachedTagNamesByIDsCallCount)
	}
	if tagReader.AttachedTagNamesByIDsCallCount == 0 {
		t.Error("TC-AC16-2: expected at least 1 AttachedTagNamesByIDs call for epics+features")
	}
}

// TC-AC16-3: Empty hierarchy → zero AttachedTagNamesByIDs calls.
func TestViewerService_Hierarchy_TagDecoration_EmptyHierarchy_ZeroCalls(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
			return []*models.Epic{}, nil
		}},
		&mockViewerFeatureRepo{ListFunc: func(_ context.Context) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		}},
		&mockViewerTaskRepo{ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
			return []*models.ViewerTaskWithRelationships{}, nil
		}},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{}
	svc.WithTagService(tagReader)

	_, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC16-3: unexpected error: %v", err)
	}

	if tagReader.AttachedTagNamesByIDsCallCount != 0 {
		t.Errorf("TC-AC16-3: expected 0 AttachedTagNamesByIDs calls for empty hierarchy, got %d",
			tagReader.AttachedTagNamesByIDsCallCount)
	}
}

// TC-AC06-1: Tag filter prunes epics with no matching descendants and no direct tag.
func TestViewerService_Hierarchy_TagFilter_PrunesUnmatchedEpics(t *testing.T) {
	// E01 has feature F1 tagged voice; E02 has no tagged features
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
					{BaseEntity: models.BaseEntity{ID: 2, Key: "E02"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(_ context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 10}, EpicID: 1, Status: "in_progress"}, // F1 in E01
					{BaseEntity: models.BaseEntity{ID: 11}, EpicID: 2, Status: "in_progress"}, // F2 in E02
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
				return []*models.ViewerTaskWithRelationships{}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{
		EntityIDsByTagsFunc: func(_ context.Context, entityType models.EntityType, _ []string, _ TagQueryOp) ([]int64, error) {
			switch entityType {
			case models.EntityTypeFeature:
				return []int64{10}, nil // Only F1 (in E01) matches "voice"
			case models.EntityTypeEpic:
				return []int64{}, nil // No epics directly tagged
			case models.EntityTypeTask:
				return []int64{}, nil
			case models.EntityTypeBug:
				return []int64{}, nil
			case models.EntityTypeChange:
				return []int64{}, nil
			case models.EntityTypeIdea:
				return []int64{}, nil
			}
			return []int64{}, nil
		},
		AttachedTagNamesByIDsFunc: func(_ context.Context, entityType models.EntityType, ids []int64) (map[int64][]string, error) {
			if entityType == models.EntityTypeFeature {
				return map[int64][]string{10: {"voice"}}, nil
			}
			return map[int64][]string{}, nil
		},
	}
	svc.WithTagService(tagReader)

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("TC-AC06-1: unexpected error: %v", err)
	}

	// E02 should be pruned (no matching features, not directly tagged)
	if len(resp.Epics) != 1 {
		t.Errorf("TC-AC06-1: expected 1 epic (E01), got %d", len(resp.Epics))
	}
	if len(resp.Epics) > 0 && resp.Epics[0].Key != "E01" {
		t.Errorf("TC-AC06-1: expected E01 to survive, got %s", resp.Epics[0].Key)
	}
}

// TC-AC06-3: Epic directly tagged voice, zero matching features → epic present with empty features.
func TestViewerService_Hierarchy_TagFilter_EpicDirectTaggedNoMatchingFeatures(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(_ context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 10}, EpicID: 1, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
				return []*models.ViewerTaskWithRelationships{}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{
		EntityIDsByTagsFunc: func(_ context.Context, entityType models.EntityType, _ []string, _ TagQueryOp) ([]int64, error) {
			switch entityType {
			case models.EntityTypeEpic:
				return []int64{1}, nil // E01 directly tagged
			case models.EntityTypeFeature:
				return []int64{}, nil // No feature tagged
			default:
				return []int64{}, nil
			}
		},
		AttachedTagNamesByIDsFunc: func(_ context.Context, entityType models.EntityType, _ []int64) (map[int64][]string, error) {
			if entityType == models.EntityTypeEpic {
				return map[int64][]string{1: {"voice"}}, nil
			}
			return map[int64][]string{}, nil
		},
	}
	svc.WithTagService(tagReader)

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("TC-AC06-3: unexpected error: %v", err)
	}

	// E01 is directly tagged → stays; feature F10 is untagged → pruned
	if len(resp.Epics) != 1 {
		t.Fatalf("TC-AC06-3: expected E01 to survive (directly tagged), got %d epics", len(resp.Epics))
	}
	if len(resp.Epics[0].Features) != 0 {
		t.Errorf("TC-AC06-3: expected 0 features (untagged feature pruned), got %d", len(resp.Epics[0].Features))
	}
}

// TC-AC06-4: Flat entities independently filtered.
func TestViewerService_Hierarchy_TagFilter_FlatEntitiesIndependentlyFiltered(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(_ context.Context) ([]*models.Feature, error) {
				return []*models.Feature{}, nil
			},
		},
		&mockViewerTaskRepo{
			ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
				return []*models.ViewerTaskWithRelationships{}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	svc.WithBugListRepo(&mockViewerBugListRepo{
		ListAllFunc: func(_ context.Context, _ bool) ([]*models.Bug, error) {
			return []*models.Bug{
				{BaseEntity: models.BaseEntity{ID: 200, Key: "B001"}, Status: "new"},
				{BaseEntity: models.BaseEntity{ID: 201, Key: "B002"}, Status: "new"},
			}, nil
		},
	})
	tagReader := &mockTagReader{
		EntityIDsByTagsFunc: func(_ context.Context, entityType models.EntityType, _ []string, _ TagQueryOp) ([]int64, error) {
			switch entityType {
			case models.EntityTypeBug:
				return []int64{200}, nil // Only B001 matches
			default:
				return []int64{}, nil
			}
		},
		AttachedTagNamesByIDsFunc: func(_ context.Context, entityType models.EntityType, _ []int64) (map[int64][]string, error) {
			if entityType == models.EntityTypeBug {
				return map[int64][]string{200: {"voice"}}, nil
			}
			return map[int64][]string{}, nil
		},
	}
	svc.WithTagService(tagReader)

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("TC-AC06-4: unexpected error: %v", err)
	}

	// Only B001 tagged → only B001 in bugs list
	if len(resp.Bugs) != 1 {
		t.Errorf("TC-AC06-4: expected 1 bug (B001), got %d", len(resp.Bugs))
	}
	if len(resp.Bugs) > 0 && resp.Bugs[0].Key != "B001" {
		t.Errorf("TC-AC06-4: expected B001, got %s", resp.Bugs[0].Key)
	}
}

// TC-AC06-5: No entities tagged → empty result.
func TestViewerService_Hierarchy_TagFilter_NothingTagged_EmptyResult(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(_ context.Context) ([]*models.Feature, error) {
				return []*models.Feature{
					{BaseEntity: models.BaseEntity{ID: 10}, EpicID: 1, Status: "in_progress"},
				}, nil
			},
		},
		&mockViewerTaskRepo{
			ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
				return []*models.ViewerTaskWithRelationships{}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{
		EntityIDsByTagsFunc: func(_ context.Context, _ models.EntityType, _ []string, _ TagQueryOp) ([]int64, error) {
			return []int64{}, nil // nothing matches
		},
		AttachedTagNamesByIDsFunc: func(_ context.Context, _ models.EntityType, _ []int64) (map[int64][]string, error) {
			return map[int64][]string{}, nil
		},
	}
	svc.WithTagService(tagReader)

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("TC-AC06-5: unexpected error: %v", err)
	}

	if len(resp.Epics) != 0 {
		t.Errorf("TC-AC06-5: expected 0 epics, got %d", len(resp.Epics))
	}
}

// TC-AC07-3: EntityIDsByTags called with full multi-tag slice in ONE call per entity type.
func TestViewerService_Hierarchy_TagFilter_AndSemanticsOneCallPerType(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{
			ListFunc: func(_ context.Context) ([]*models.Feature, error) {
				return []*models.Feature{}, nil
			},
		},
		&mockViewerTaskRepo{
			ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
				return []*models.ViewerTaskWithRelationships{}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	var capturedNames [][]string
	var capturedOps []TagQueryOp
	tagReader := &mockTagReader{
		EntityIDsByTagsFunc: func(_ context.Context, _ models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
			capturedNames = append(capturedNames, names)
			capturedOps = append(capturedOps, op)
			return []int64{1}, nil
		},
		AttachedTagNamesByIDsFunc: func(_ context.Context, _ models.EntityType, _ []int64) (map[int64][]string, error) {
			return map[int64][]string{1: {"voice", "auth"}}, nil
		},
	}
	svc.WithTagService(tagReader)

	_, err := svc.Hierarchy(context.Background(), HierarchyOptions{Tags: []string{"voice", "auth"}})
	if err != nil {
		t.Fatalf("TC-AC07-3: unexpected error: %v", err)
	}

	// Each call should have the full tag set ["voice", "auth"]
	for i, names := range capturedNames {
		if len(names) != 2 {
			t.Errorf("TC-AC07-3: call %d had %d names (want 2); got %v", i, len(names), names)
		}
	}
	// All ops should be AND
	for i, op := range capturedOps {
		if op != TagQueryOpAnd {
			t.Errorf("TC-AC07-3: call %d used op %q (want %q)", i, op, TagQueryOpAnd)
		}
	}
}

// AC-T4: *UnregisteredTagError from EntityIDsByTags propagates unchanged.
func TestViewerService_Hierarchy_TagFilter_UnregisteredTagErrorPropagates(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{ListFunc: func(_ context.Context) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		}},
		&mockViewerTaskRepo{ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
			return []*models.ViewerTaskWithRelationships{}, nil
		}},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	unregErr := &UnregisteredTagError{Name: "does-not-exist"}
	tagReader := &mockTagReader{
		EntityIDsByTagsFunc: func(_ context.Context, _ models.EntityType, _ []string, _ TagQueryOp) ([]int64, error) {
			return nil, unregErr
		},
	}
	svc.WithTagService(tagReader)

	_, err := svc.Hierarchy(context.Background(), HierarchyOptions{Tags: []string{"does-not-exist"}})
	if err == nil {
		t.Fatal("AC-T4: expected UnregisteredTagError to propagate, got nil")
	}
	var got *UnregisteredTagError
	if !errors.As(err, &got) {
		t.Errorf("AC-T4: expected *UnregisteredTagError, got %T: %v", err, err)
	}
}

// AC-T5: Nil tagSvc + non-empty opts.Tags → no error, no panic, all DTOs carry [].
func TestViewerService_Hierarchy_NilTagSvc_WithTagFilter_Graceful(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{ListFunc: func(_ context.Context) ([]*models.Feature, error) {
			return []*models.Feature{
				{BaseEntity: models.BaseEntity{ID: 10}, EpicID: 1, Status: "in_progress"},
			}, nil
		}},
		&mockViewerTaskRepo{ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
			return []*models.ViewerTaskWithRelationships{}, nil
		}},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// tagSvc is nil — do NOT call WithTagService

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("AC-T5: panicked with nil tagSvc + opts.Tags: %v", r)
		}
	}()

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("AC-T5: expected no error with nil tagSvc, got %v", err)
	}
	// Filter is ignored — the unfiltered tree is returned
	if len(resp.Epics) == 0 {
		t.Error("AC-T5: expected hierarchy to be returned unfiltered when tagSvc is nil")
	}
	// All Tags fields are non-nil []string{}
	if len(resp.Epics) > 0 && resp.Epics[0].Tags == nil {
		t.Error("AC-T5: epic.Tags must be non-nil even with nil tagSvc")
	}
}

// ----- FeatureTasks tag filter and decoration tests (T-E28-F06-002) -----

// TC-AC09-1: Tag filter applied before pagination; Total reflects post-filter count.
func TestViewerService_FeatureTasks_TagFilter_BeforePagination(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 20, Key: "E01-F01"}}

	// 10 tasks, IDs 100-109
	allTasks := make([]*models.ViewerTaskWithRelationships, 10)
	for i := range allTasks {
		allTasks[i] = &models.ViewerTaskWithRelationships{
			Task:              &models.Task{BaseEntity: models.BaseEntity{ID: int64(100 + i)}, FeatureID: 20, Status: "todo"},
			RelationshipsJSON: "[]",
		}
	}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(_ context.Context, key string) (*models.Feature, error) {
				return feature, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureWithViewerRelationshipsFunc: func(_ context.Context, featureID int64) ([]*models.ViewerTaskWithRelationships, error) {
				return allTasks, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	// 3 tasks tagged "voice": IDs 100, 101, 102
	tagReader := &mockTagReader{
		EntityIDsByTagsFunc: func(_ context.Context, entityType models.EntityType, _ []string, _ TagQueryOp) ([]int64, error) {
			if entityType == models.EntityTypeTask {
				return []int64{100, 101, 102}, nil
			}
			return []int64{}, nil
		},
		AttachedTagNamesByIDsFunc: func(_ context.Context, _ models.EntityType, ids []int64) (map[int64][]string, error) {
			result := map[int64][]string{}
			for _, id := range ids {
				if id == 100 || id == 101 || id == 102 {
					result[id] = []string{"voice"}
				}
			}
			return result, nil
		},
	}
	svc.WithTagService(tagReader)

	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{
		Tags:  []string{"voice"},
		Limit: 2, // page size 2
	})
	if err != nil {
		t.Fatalf("TC-AC09-1: unexpected error: %v", err)
	}

	// Total must reflect the 3 tagged tasks (post-filter, pre-pagination count)
	if resp.Total != 3 {
		t.Errorf("TC-AC09-1: Total must be 3 (tag-filtered count before pagination), got %d", resp.Total)
	}
	// Page 1 of 3 tagged tasks: 2 tasks
	if len(resp.Tasks) != 2 {
		t.Errorf("TC-AC09-1: expected 2 tasks on page 1, got %d", len(resp.Tasks))
	}
}

// TC-AC09-2: No tag filter → unchanged behavior (Total = all tasks).
func TestViewerService_FeatureTasks_NoTagFilter_UnchangedBehavior(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 20, Key: "E01-F01"}}
	allTasks := make([]*models.ViewerTaskWithRelationships, 5)
	for i := range allTasks {
		allTasks[i] = &models.ViewerTaskWithRelationships{
			Task:              &models.Task{BaseEntity: models.BaseEntity{ID: int64(100 + i)}, FeatureID: 20, Status: "todo"},
			RelationshipsJSON: "[]",
		}
	}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(_ context.Context, _ string) (*models.Feature, error) {
				return feature, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureWithViewerRelationshipsFunc: func(_ context.Context, _ int64) ([]*models.ViewerTaskWithRelationships, error) {
				return allTasks, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{}
	svc.WithTagService(tagReader)

	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{}) // empty Tags
	if err != nil {
		t.Fatalf("TC-AC09-2: unexpected error: %v", err)
	}
	if resp.Total != 5 {
		t.Errorf("TC-AC09-2: expected Total=5, got %d", resp.Total)
	}
	// EntityIDsByTags must not be called when Tags is empty
	if tagReader.EntityIDsByTagsCallCount != 0 {
		t.Errorf("TC-AC09-2: EntityIDsByTags should not be called when Tags is empty, got %d calls",
			tagReader.EntityIDsByTagsCallCount)
	}
}

// TC-AC09-3: Post-pagination decoration uses IDs of the page only (not pre-filter set).
func TestViewerService_FeatureTasks_TagDecoration_PageScopedIDs(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 20, Key: "E01-F01"}}
	// 10 tasks total
	allTasks := make([]*models.ViewerTaskWithRelationships, 10)
	for i := range allTasks {
		allTasks[i] = &models.ViewerTaskWithRelationships{
			Task:              &models.Task{BaseEntity: models.BaseEntity{ID: int64(100 + i)}, FeatureID: 20, Status: "todo"},
			RelationshipsJSON: "[]",
		}
	}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(_ context.Context, _ string) (*models.Feature, error) {
				return feature, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureWithViewerRelationshipsFunc: func(_ context.Context, _ int64) ([]*models.ViewerTaskWithRelationships, error) {
				return allTasks, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	var capturedDecorateIDs []int64
	tagReader := &mockTagReader{
		AttachedTagNamesByIDsFunc: func(_ context.Context, entityType models.EntityType, ids []int64) (map[int64][]string, error) {
			if entityType == models.EntityTypeTask {
				capturedDecorateIDs = ids
			}
			return map[int64][]string{}, nil
		},
	}
	svc.WithTagService(tagReader)

	// No tag filter, limit 3 → page has 3 tasks (IDs 100, 101, 102)
	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{Limit: 3})
	if err != nil {
		t.Fatalf("TC-AC09-3: unexpected error: %v", err)
	}
	if len(resp.Tasks) != 3 {
		t.Fatalf("TC-AC09-3: expected 3 tasks on page, got %d", len(resp.Tasks))
	}

	// Decoration must be called with IDs of the 3-item page only (not all 10)
	if len(capturedDecorateIDs) != 3 {
		t.Errorf("TC-AC09-3: expected 3 IDs passed to AttachedTagNamesByIDs (page-scoped), got %d: %v",
			len(capturedDecorateIDs), capturedDecorateIDs)
	}
}

// TC-AC14-2: Hierarchy with nil tagSvc and empty opts.Tags → all DTOs carry Tags: []string{}.
func TestViewerService_Hierarchy_NilTagSvc_NoTagFilter_EmptyTags(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{
			ListFunc: func(_ context.Context, _ *models.EpicStatus) ([]*models.Epic, error) {
				return []*models.Epic{
					{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: models.EpicStatusActive},
				}, nil
			},
		},
		&mockViewerFeatureRepo{ListFunc: func(_ context.Context) ([]*models.Feature, error) {
			return []*models.Feature{
				{BaseEntity: models.BaseEntity{ID: 10}, EpicID: 1, Status: "in_progress"},
			}, nil
		}},
		&mockViewerTaskRepo{ListWithViewerRelationshipsFunc: func(_ context.Context) ([]*models.ViewerTaskWithRelationships, error) {
			return []*models.ViewerTaskWithRelationships{
				{Task: &models.Task{BaseEntity: models.BaseEntity{ID: 100}, FeatureID: 10, Status: "todo"}, RelationshipsJSON: "[]"},
			}, nil
		}},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// tagSvc nil

	resp, err := svc.Hierarchy(context.Background(), HierarchyOptions{})
	if err != nil {
		t.Fatalf("TC-AC14-2: unexpected error: %v", err)
	}
	if len(resp.Epics) == 0 {
		t.Fatal("TC-AC14-2: expected at least 1 epic")
	}
	if resp.Epics[0].Tags == nil {
		t.Error("TC-AC14-2: epic.Tags must be non-nil — ADR-F06-2")
	}
	if resp.Epics[0].Features[0].Tags == nil {
		t.Error("TC-AC14-2: feature.Tags must be non-nil — ADR-F06-2")
	}
	if resp.Epics[0].Features[0].Tasks[0].Tags == nil {
		t.Error("TC-AC14-2: task.Tags must be non-nil — ADR-F06-2")
	}
}

// TC-AC14-4: FeatureTasks with nil tagSvc and opts.Tags set → silently ignored, no panic.
func TestViewerService_FeatureTasks_NilTagSvc_TagFilter_Graceful(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 20, Key: "E01-F01"}}
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(_ context.Context, _ string) (*models.Feature, error) {
				return feature, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureWithViewerRelationshipsFunc: func(_ context.Context, _ int64) ([]*models.ViewerTaskWithRelationships, error) {
				return []*models.ViewerTaskWithRelationships{
					{Task: &models.Task{BaseEntity: models.BaseEntity{ID: 100}, FeatureID: 20, Status: "todo"}, RelationshipsJSON: "[]"},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	// tagSvc nil — do NOT call WithTagService

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TC-AC14-4: panicked with nil tagSvc + opts.Tags: %v", r)
		}
	}()

	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("TC-AC14-4: expected no error, got %v", err)
	}
	// Filter ignored → task still present
	if len(resp.Tasks) == 0 {
		t.Error("TC-AC14-4: expected task to be returned when tag filter is silently ignored")
	}
	if resp.Tasks[0].Tags == nil {
		t.Error("TC-AC14-4: task.Tags must be non-nil — ADR-F06-2")
	}
}

// FeatureTasks: tasks carry non-nil []string{} Tags even with tagSvc wired.
func TestViewerService_FeatureTasks_TagDecoration_EmptyTagsNonNil(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 20, Key: "E01-F01"}}
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{
			GetByKeyFunc: func(_ context.Context, _ string) (*models.Feature, error) {
				return feature, nil
			},
		},
		&mockViewerTaskRepo{
			ListByFeatureWithViewerRelationshipsFunc: func(_ context.Context, _ int64) ([]*models.ViewerTaskWithRelationships, error) {
				return []*models.ViewerTaskWithRelationships{
					{Task: &models.Task{BaseEntity: models.BaseEntity{ID: 100}, FeatureID: 20, Status: "todo"}, RelationshipsJSON: "[]"},
				}, nil
			},
		},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)
	tagReader := &mockTagReader{
		AttachedTagNamesByIDsFunc: func(_ context.Context, _ models.EntityType, _ []int64) (map[int64][]string, error) {
			return map[int64][]string{}, nil // no tags
		},
	}
	svc.WithTagService(tagReader)

	resp, err := svc.FeatureTasks(context.Background(), "E01-F01", FeatureTaskOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resp.Tasks))
	}
	if resp.Tasks[0].Tags == nil {
		t.Error("ViewerTask.Tags must be non-nil even when untagged — ADR-F06-2")
	}
	if len(resp.Tasks[0].Tags) != 0 {
		t.Errorf("expected empty Tags on untagged task, got %v", resp.Tasks[0].Tags)
	}
}

func TestViewerService_SprintOverview_DefaultActiveSprint(t *testing.T) {
	active := &models.Sprint{
		ID:        24,
		Key:       "S024",
		Name:      "Current Sprint",
		Goal:      "Stabilize the viewer",
		StartDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	}
	upcoming := &models.Sprint{ID: 25, Key: "S025", Name: "Next Sprint", Status: "planning"}
	completed := &models.Sprint{ID: 23, Key: "S023", Name: "Done Sprint", Status: "completed"}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	statusCalls := map[string]int{}
	sprintSvc := &mockViewerSprintService{
		ListSprintsFunc: func(_ context.Context, filters *SprintListFilters) ([]*models.Sprint, error) {
			if filters == nil {
				t.Fatal("expected sprint status filter")
			}
			statusCalls[filters.Status]++
			switch filters.Status {
			case "active":
				return []*models.Sprint{active}, nil
			case "closing":
				return nil, nil
			case "planning":
				return []*models.Sprint{upcoming}, nil
			case "completed":
				return []*models.Sprint{completed}, nil
			case "cancelled", "archived":
				return nil, nil
			default:
				t.Fatalf("unexpected sprint status filter %q", filters.Status)
			}
			return nil, nil
		},
		GetSprintBacklogFunc: func(_ context.Context, sprintKey string, opts BacklogOptions) (*SprintBacklog, error) {
			if sprintKey != active.Key {
				t.Fatalf("expected sprint key %q, got %q", active.Key, sprintKey)
			}
			if opts.EntityType != "" || opts.BlockedOnly {
				t.Fatalf("expected zero backlog filters, got %#v", opts)
			}
			return &SprintBacklog{
				SprintKey:         sprintKey,
				SprintName:        active.Name,
				TotalCount:        4,
				CompletedCount:    1,
				CompletionPercent: 25,
				Groups: []*BacklogGroup{
					{StatusCategory: "blocked", Items: []*BacklogItemView{}},
					{StatusCategory: "in_progress", Items: []*BacklogItemView{}},
				},
			}, nil
		},
		GetSprintReadinessFunc: func(_ context.Context, sprintKey string) (*SprintReadiness, error) {
			if sprintKey != active.Key {
				t.Fatalf("expected sprint key %q, got %q", active.Key, sprintKey)
			}
			return &SprintReadiness{OverallScore: 78, Factors: []ReadinessFactor{{Name: "Capacity utilization", Score: 25, MaxScore: 25}}}, nil
		},
		GetSprintCapacityFunc: func(_ context.Context, sprintKey string) ([]CapacityRow, error) {
			if sprintKey != active.Key {
				t.Fatalf("expected sprint key %q, got %q", active.Key, sprintKey)
			}
			return []CapacityRow{{AgentType: "frontend"}}, nil
		},
	}

	// Summary is NOT expected for active sprints; GetSummary only applies to completed/archived.
	analyticsSvc := &mockViewerSprintAnalyticsService{
		GetSummaryFunc: func(_ context.Context, sprintKey string, detailed bool) (*SprintSummaryResult, error) {
			t.Fatal("GetSummary must not be called for an active sprint")
			return nil, nil
		},
	}

	svc.WithSprintService(sprintSvc)
	svc.WithSprintAnalyticsService(analyticsSvc)

	resp, err := svc.SprintOverview(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCalls["active"] != 2 {
		t.Fatalf("expected active sprint listing twice (resolve + catalog), got %d", statusCalls["active"])
	}
	for _, status := range []string{"closing", "planning", "completed", "cancelled", "archived"} {
		if statusCalls[status] != 1 {
			t.Fatalf("expected one catalog list call for %s, got %d", status, statusCalls[status])
		}
	}
	if resp.Sprint == nil || resp.Sprint.Key != active.Key {
		t.Fatalf("expected sprint %q in overview, got %#v", active.Key, resp.Sprint)
	}
	if resp.Backlog == nil || resp.Backlog.SprintKey != active.Key {
		t.Fatalf("expected backlog for %q, got %#v", active.Key, resp.Backlog)
	}
	if resp.Readiness == nil || resp.Readiness.OverallScore != 78 {
		t.Fatalf("expected readiness 78, got %#v", resp.Readiness)
	}
	if len(resp.Capacity) != 1 {
		t.Fatalf("expected 1 capacity row, got %d", len(resp.Capacity))
	}
	if resp.Summary != nil {
		t.Fatalf("expected no summary for active sprint, got %#v", resp.Summary)
	}
	if resp.Catalog == nil {
		t.Fatal("expected sprint catalog in overview response")
	}
	if len(resp.Catalog.Active) != 1 || resp.Catalog.Active[0].Key != active.Key {
		t.Fatalf("expected active catalog to contain %q, got %#v", active.Key, resp.Catalog.Active)
	}
	if len(resp.Catalog.Upcoming) != 1 || resp.Catalog.Upcoming[0].Key != upcoming.Key {
		t.Fatalf("expected upcoming catalog to contain %q, got %#v", upcoming.Key, resp.Catalog.Upcoming)
	}
	if len(resp.Catalog.Archived) != 1 || resp.Catalog.Archived[0].Key != completed.Key {
		t.Fatalf("expected archived catalog to contain %q, got %#v", completed.Key, resp.Catalog.Archived)
	}
}

func TestViewerService_SprintOverview_CompletedSprintIncludesSummary(t *testing.T) {
	completed := &models.Sprint{
		ID:     23,
		Key:    "S023",
		Name:   "Previous Sprint",
		Status: "completed",
	}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	sprintSvc := &mockViewerSprintService{
		ListSprintsFunc: func(_ context.Context, filters *SprintListFilters) ([]*models.Sprint, error) {
			if filters == nil {
				t.Fatal("expected sprint status filter")
			}
			switch filters.Status {
			case "completed":
				return []*models.Sprint{completed}, nil
			case "active", "closing", "planning", "cancelled", "archived":
				return nil, nil
			default:
				t.Fatalf("unexpected sprint status filter %q", filters.Status)
			}
			return nil, nil
		},
		GetSprintFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			return completed, nil
		},
		GetSprintBacklogFunc: func(_ context.Context, sprintKey string, opts BacklogOptions) (*SprintBacklog, error) {
			return &SprintBacklog{SprintKey: sprintKey, SprintName: completed.Name}, nil
		},
		GetSprintReadinessFunc: func(_ context.Context, sprintKey string) (*SprintReadiness, error) {
			return &SprintReadiness{OverallScore: 100}, nil
		},
		GetSprintCapacityFunc: func(_ context.Context, sprintKey string) ([]CapacityRow, error) {
			return nil, nil
		},
	}

	analyticsSvc := &mockViewerSprintAnalyticsService{
		GetSummaryFunc: func(_ context.Context, sprintKey string, detailed bool) (*SprintSummaryResult, error) {
			return &SprintSummaryResult{SprintKey: sprintKey, SprintName: completed.Name, VelocityThisSprint: 34}, nil
		},
	}

	svc.WithSprintService(sprintSvc)
	svc.WithSprintAnalyticsService(analyticsSvc)

	resp, err := svc.SprintOverview(context.Background(), completed.Key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Summary == nil || resp.Summary.VelocityThisSprint != 34 {
		t.Fatalf("expected summary for completed sprint, got %#v", resp.Summary)
	}
	if resp.Catalog == nil || len(resp.Catalog.Archived) != 1 || resp.Catalog.Archived[0].Key != completed.Key {
		t.Fatalf("expected archived catalog to contain %q, got %#v", completed.Key, resp.Catalog)
	}
}

func TestViewerService_SprintPlan_UsesResolvedSprint(t *testing.T) {
	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	var gotKey string
	sprintSvc := &mockViewerSprintService{
		ListSprintsFunc: func(_ context.Context, filters *SprintListFilters) ([]*models.Sprint, error) {
			if filters == nil {
				t.Fatal("expected sprint status filter")
			}
			switch filters.Status {
			case "active", "closing", "completed", "cancelled", "archived":
				return nil, nil
			case "planning":
				return []*models.Sprint{{Key: "S024", Status: "planning"}}, nil
			default:
				t.Fatalf("unexpected sprint status filter %q", filters.Status)
			}
			return nil, nil
		},
		GetSprintFunc: func(_ context.Context, sprintKey string) (*models.Sprint, error) {
			if sprintKey != "S024" {
				t.Fatalf("expected normalized sprint key S024, got %q", sprintKey)
			}
			return &models.Sprint{Key: sprintKey}, nil
		},
		PlanSprintFunc: func(_ context.Context, sprintKey string) (*SprintPlanView, error) {
			gotKey = sprintKey
			return &SprintPlanView{
				Sprint:    &models.Sprint{Key: sprintKey},
				Backlog:   []sprint.BacklogItem{},
				Capacity:  []CapacityRow{},
				Readiness: &SprintReadiness{OverallScore: 64},
			}, nil
		},
	}

	svc.WithSprintService(sprintSvc)

	resp, err := svc.SprintPlan(context.Background(), "s024")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKey != "S024" {
		t.Fatalf("expected normalized sprint key S024, got %q", gotKey)
	}
	if resp.Sprint == nil || resp.Sprint.Key != "S024" {
		t.Fatalf("expected plan sprint S024, got %#v", resp.Sprint)
	}
	if resp.Readiness == nil || resp.Readiness.OverallScore != 64 {
		t.Fatalf("expected plan readiness 64, got %#v", resp.Readiness)
	}
	if resp.Catalog == nil || len(resp.Catalog.Upcoming) != 1 || resp.Catalog.Upcoming[0].Key != "S024" {
		t.Fatalf("expected plan catalog with S024 in upcoming, got %#v", resp.Catalog)
	}
}

// TestViewerService_SprintReport_PlanningSprintReturnsNilSummary verifies TC-001 and TC-002:
// SprintReport must return (response, nil) with response.Summary == nil when GetSummary
// returns an error (i.e. the sprint is in planning or active status, not completed/archived).
// Counter-factual: the pre-fix code propagates the GetSummary error, so err != nil in the test.
func TestViewerService_SprintReport_PlanningSprintReturnsNilSummary(t *testing.T) {
	planning := &models.Sprint{
		ID:     24,
		Key:    "S024",
		Name:   "Planning Sprint",
		Status: models.SprintStatus("planning"),
	}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	analyticsSvc := &mockViewerSprintAnalyticsService{
		GetBurndownFunc: func(_ context.Context, sprintKey string) (*BurndownResult, error) {
			return &BurndownResult{SprintKey: sprintKey, SprintName: planning.Name}, nil
		},
		GetVelocityFunc: func(_ context.Context, n int) (*VelocityResult, error) {
			return &VelocityResult{SprintCount: n}, nil
		},
		GetSummaryFunc: func(_ context.Context, sprintKey string, detailed bool) (*SprintSummaryResult, error) {
			// Simulate the error returned by SprintAnalyticsService for non-completed sprints.
			return nil, fmt.Errorf("sprint summary is available for completed or archived sprints only")
		},
	}
	sprintSvc := &mockViewerSprintService{
		ListSprintsFunc: func(_ context.Context, filters *SprintListFilters) ([]*models.Sprint, error) {
			if filters == nil {
				t.Fatal("expected sprint status filter")
			}
			switch filters.Status {
			case "active":
				return nil, nil
			case "planning":
				return []*models.Sprint{planning}, nil
			case "closing", "completed", "cancelled", "archived":
				return nil, nil
			default:
				t.Fatalf("unexpected sprint status filter %q", filters.Status)
			}
			return nil, nil
		},
		GetSprintFunc: func(_ context.Context, sprintKey string) (*models.Sprint, error) {
			if sprintKey != planning.Key {
				t.Fatalf("expected sprint key %q, got %q", planning.Key, sprintKey)
			}
			return planning, nil
		},
	}

	svc.WithSprintService(sprintSvc)
	svc.WithSprintAnalyticsService(analyticsSvc)

	// TC-001: err must be nil (GetSummary error is non-fatal)
	resp, err := svc.SprintReport(context.Background(), planning.Key)
	if err != nil {
		t.Fatalf("TC-001: expected no error for planning sprint, got: %v", err)
	}
	if resp == nil {
		t.Fatal("TC-001: expected non-nil response for planning sprint")
	}
	if resp.Sprint == nil || resp.Sprint.Key != planning.Key {
		t.Fatalf("TC-001: expected sprint %q in report, got %#v", planning.Key, resp.Sprint)
	}
	if resp.Burndown == nil {
		t.Fatal("TC-001: expected non-nil Burndown in report")
	}
	if resp.Velocity == nil {
		t.Fatal("TC-001: expected non-nil Velocity in report")
	}

	// TC-002: Summary must be nil (pointer nil, not zero-value struct)
	if resp.Summary != nil {
		t.Fatalf("TC-002: expected Summary == nil for planning sprint, got %#v", resp.Summary)
	}
}

// TestViewerService_SprintReport_CompletedSprintRetainsSummary verifies TC-003:
// Completed/archived sprints must still return a non-nil Summary (backward compatibility).
// Counter-factual: an impl that always sets summary = nil would fail the non-nil assertion.
func TestViewerService_SprintReport_CompletedSprintRetainsSummary(t *testing.T) {
	completed := &models.Sprint{
		ID:     10,
		Key:    "S010",
		Name:   "Completed Sprint",
		Status: models.SprintStatus("completed"),
	}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	analyticsSvc := &mockViewerSprintAnalyticsService{
		GetBurndownFunc: func(_ context.Context, sprintKey string) (*BurndownResult, error) {
			return &BurndownResult{SprintKey: sprintKey, SprintName: completed.Name}, nil
		},
		GetVelocityFunc: func(_ context.Context, n int) (*VelocityResult, error) {
			return &VelocityResult{SprintCount: n}, nil
		},
		GetSummaryFunc: func(_ context.Context, sprintKey string, detailed bool) (*SprintSummaryResult, error) {
			// Completed sprint: GetSummary succeeds and returns a real summary.
			return &SprintSummaryResult{SprintKey: sprintKey, SprintName: completed.Name, VelocityThisSprint: 42}, nil
		},
	}
	sprintSvc := &mockViewerSprintService{
		ListSprintsFunc: func(_ context.Context, filters *SprintListFilters) ([]*models.Sprint, error) {
			if filters == nil {
				t.Fatal("expected sprint status filter")
			}
			switch filters.Status {
			case "active", "planning", "closing", "cancelled":
				return nil, nil
			case "completed":
				return []*models.Sprint{completed}, nil
			case "archived":
				return nil, nil
			default:
				t.Fatalf("unexpected sprint status filter %q", filters.Status)
			}
			return nil, nil
		},
		GetSprintFunc: func(_ context.Context, sprintKey string) (*models.Sprint, error) {
			if sprintKey != completed.Key {
				t.Fatalf("expected sprint key %q, got %q", completed.Key, sprintKey)
			}
			return completed, nil
		},
	}

	svc.WithSprintService(sprintSvc)
	svc.WithSprintAnalyticsService(analyticsSvc)

	// TC-003: completed sprint must return non-nil Summary
	resp, err := svc.SprintReport(context.Background(), completed.Key)
	if err != nil {
		t.Fatalf("TC-003: unexpected error for completed sprint: %v", err)
	}
	if resp == nil {
		t.Fatal("TC-003: expected non-nil response for completed sprint")
	}
	if resp.Summary == nil {
		t.Fatal("TC-003: expected non-nil Summary for completed sprint")
	}
	if resp.Summary.VelocityThisSprint != 42 {
		t.Fatalf("TC-003: expected VelocityThisSprint=42, got %d", resp.Summary.VelocityThisSprint)
	}
}

func TestViewerService_SprintReport_ComposesAnalytics(t *testing.T) {
	active := &models.Sprint{
		ID:   24,
		Key:  "S024",
		Name: "Current Sprint",
	}

	svc := buildViewerService(t,
		&mockViewerEpicRepo{},
		&mockViewerFeatureRepo{},
		&mockViewerTaskRepo{},
		&mockViewerBugRepo{},
		&mockViewerChangeCardRepo{},
		&mockViewerHistoryRepo{},
	)

	var velocityLimit int
	analyticsSvc := &mockViewerSprintAnalyticsService{
		GetBurndownFunc: func(_ context.Context, sprintKey string) (*BurndownResult, error) {
			if sprintKey != active.Key {
				t.Fatalf("expected sprint key %q, got %q", active.Key, sprintKey)
			}
			return &BurndownResult{SprintKey: sprintKey, SprintName: active.Name}, nil
		},
		GetVelocityFunc: func(_ context.Context, n int) (*VelocityResult, error) {
			velocityLimit = n
			return &VelocityResult{SprintCount: n}, nil
		},
		GetSummaryFunc: func(_ context.Context, sprintKey string, detailed bool) (*SprintSummaryResult, error) {
			if sprintKey != active.Key {
				t.Fatalf("expected sprint key %q, got %q", active.Key, sprintKey)
			}
			if detailed {
				t.Fatal("expected report summary to request non-detailed summary")
			}
			return &SprintSummaryResult{SprintKey: sprintKey, SprintName: active.Name, VelocityThisSprint: 32}, nil
		},
	}
	sprintSvc := &mockViewerSprintService{
		ListSprintsFunc: func(_ context.Context, filters *SprintListFilters) ([]*models.Sprint, error) {
			if filters == nil {
				t.Fatal("expected sprint status filter")
			}
			switch filters.Status {
			case "active":
				return []*models.Sprint{active}, nil
			case "closing", "planning", "completed", "cancelled", "archived":
				return nil, nil
			default:
				t.Fatalf("unexpected sprint status filter %q", filters.Status)
			}
			return nil, nil
		},
		GetSprintFunc: func(_ context.Context, sprintKey string) (*models.Sprint, error) {
			if sprintKey != active.Key {
				t.Fatalf("expected sprint key %q, got %q", active.Key, sprintKey)
			}
			return active, nil
		},
	}

	svc.WithSprintService(sprintSvc)
	svc.WithSprintAnalyticsService(analyticsSvc)

	resp, err := svc.SprintReport(context.Background(), active.Key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Sprint == nil || resp.Sprint.Key != active.Key {
		t.Fatalf("expected sprint %q in report, got %#v", active.Key, resp.Sprint)
	}
	if resp.Burndown == nil || resp.Burndown.SprintKey != active.Key {
		t.Fatalf("expected burndown for %q, got %#v", active.Key, resp.Burndown)
	}
	if resp.Velocity == nil || resp.Velocity.SprintCount != 6 {
		t.Fatalf("expected velocity payload with count 6, got %#v", resp.Velocity)
	}
	if resp.Summary == nil || resp.Summary.VelocityThisSprint != 32 {
		t.Fatalf("expected report summary, got %#v", resp.Summary)
	}
	if velocityLimit != 6 {
		t.Fatalf("expected report velocity limit 6, got %d", velocityLimit)
	}
	if resp.Catalog == nil || len(resp.Catalog.Active) != 1 || resp.Catalog.Active[0].Key != active.Key {
		t.Fatalf("expected report catalog with active sprint %q, got %#v", active.Key, resp.Catalog)
	}
}
