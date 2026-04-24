package services

// ---------------------------------------------------------------------------
// E28-F05 T-007 — GetXxxWithTags tests for all six entity services.
//
// Covers AC-20 from test-plan.md §1.6:
//   TestGet<Entity>WithTags_TwoAttachments  — happy path
//   TestGet<Entity>WithTags_ZeroAttachments — empty-slice, non-nil
//   TestGet<Entity>WithTags_NilTagSvc       — graceful degradation (REQ-F-014)
//   TestGet<Entity>WithTags_ListTagsError   — error wrapped and propagated
//
// Entities covered: Task, Epic, Feature, Bug, ChangeCard, Idea.
// All tests use MockTagService (mock_tag_service_test.go) and entity-service
// mock repositories already present in the package.
//
// No real database is used (service-layer test rule).
// ---------------------------------------------------------------------------

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ============================================================================
// TaskService.GetTaskWithTags (AC-20 ×task)
// ============================================================================

func TestGetTaskWithTags_TwoAttachments(t *testing.T) {
	ctx := context.Background()

	task := &models.Task{
		BaseEntity: models.BaseEntity{ID: 42, Key: "T-E07-F01-001", Title: "Test"},
		Status:     models.TaskStatus("todo"),
		Priority:   5,
	}
	repo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return task, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			if entityType != models.EntityTypeTask {
				t.Errorf("ListTagsForEntity entityType = %q, want %q", entityType, models.EntityTypeTask)
			}
			if entityID != 42 {
				t.Errorf("ListTagsForEntity entityID = %d, want 42", entityID)
			}
			return []string{"auth", "voice"}, nil
		},
	)

	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	gotTask, gotTags, err := svc.GetTaskWithTags(ctx, "T-E07-F01-001")
	if err != nil {
		t.Fatalf("GetTaskWithTags() error = %v", err)
	}
	if gotTask == nil {
		t.Fatal("GetTaskWithTags() task is nil, want non-nil")
	}
	if gotTask.Key != "T-E07-F01-001" {
		t.Errorf("task.Key = %q, want %q", gotTask.Key, "T-E07-F01-001")
	}
	wantTags := []string{"auth", "voice"}
	if !strSliceEq(gotTags, wantTags) {
		t.Errorf("tags = %v, want %v", gotTags, wantTags)
	}
	if tagSvc.ListTagsForEntityCalls != 1 {
		t.Errorf("ListTagsForEntityCalls = %d, want 1", tagSvc.ListTagsForEntityCalls)
	}
}

func TestGetTaskWithTags_ZeroAttachments(t *testing.T) {
	ctx := context.Background()

	task := &models.Task{
		BaseEntity: models.BaseEntity{ID: 7, Key: "T-E07-F01-002"},
		Status:     models.TaskStatus("todo"),
		Priority:   5,
	}
	repo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return task, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			return []string{}, nil // non-nil empty slice per AC-20
		},
	)

	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	_, gotTags, err := svc.GetTaskWithTags(ctx, "T-E07-F01-002")
	if err != nil {
		t.Fatalf("GetTaskWithTags() error = %v", err)
	}
	if gotTags == nil {
		t.Error("tags is nil, want non-nil empty slice")
	}
	if len(gotTags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(gotTags))
	}
}

func TestGetTaskWithTags_NilTagSvc(t *testing.T) {
	ctx := context.Background()

	task := &models.Task{
		BaseEntity: models.BaseEntity{ID: 3, Key: "T-E07-F01-003"},
		Status:     models.TaskStatus("todo"),
		Priority:   5,
	}
	repo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return task, nil
		},
	}

	// nil tagSvc → graceful degradation (REQ-F-014)
	svc := newTaskServiceWithTagSvc(repo, nil)

	gotTask, gotTags, err := svc.GetTaskWithTags(ctx, "T-E07-F01-003")
	if err != nil {
		t.Fatalf("GetTaskWithTags() with nil tagSvc error = %v", err)
	}
	if gotTask == nil {
		t.Fatal("GetTaskWithTags() task is nil, want non-nil")
	}
	if gotTags != nil {
		t.Errorf("tags = %v, want nil (graceful degradation)", gotTags)
	}
}

func TestGetTaskWithTags_ListTagsError(t *testing.T) {
	ctx := context.Background()

	task := &models.Task{
		BaseEntity: models.BaseEntity{ID: 9, Key: "T-E07-F01-004"},
		Status:     models.TaskStatus("todo"),
		Priority:   5,
	}
	repo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return task, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			return nil, fmt.Errorf("tag lookup failed")
		},
	)

	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	gotTask, gotTags, err := svc.GetTaskWithTags(ctx, "T-E07-F01-004")
	if err == nil {
		t.Fatal("GetTaskWithTags() expected error, got nil")
	}
	if gotTask != nil {
		t.Errorf("task = %v, want nil on ListTagsForEntity error (AC-T3)", gotTask)
	}
	if gotTags != nil {
		t.Errorf("tags = %v, want nil on error", gotTags)
	}
}

// ============================================================================
// EpicService.GetEpicWithTags (AC-20 ×epic)
// ============================================================================

func TestGetEpicWithTags_TwoAttachments(t *testing.T) {
	ctx := context.Background()

	epic := &models.Epic{
		BaseEntity: models.BaseEntity{ID: 11, Key: "E07", Title: "My Epic"},
		Status:     models.EpicStatusActive,
	}
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return epic, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			if entityType != models.EntityTypeEpic {
				t.Errorf("ListTagsForEntity entityType = %q, want %q", entityType, models.EntityTypeEpic)
			}
			if entityID != 11 {
				t.Errorf("ListTagsForEntity entityID = %d, want 11", entityID)
			}
			return []string{"auth", "voice"}, nil
		},
	)

	svc := newEpicServiceWithTagSvc(repo, tagSvc)

	gotEpic, gotTags, err := svc.GetEpicWithTags(ctx, "E07")
	if err != nil {
		t.Fatalf("GetEpicWithTags() error = %v", err)
	}
	if gotEpic == nil {
		t.Fatal("GetEpicWithTags() epic is nil")
	}
	wantTags := []string{"auth", "voice"}
	if !strSliceEq(gotTags, wantTags) {
		t.Errorf("tags = %v, want %v", gotTags, wantTags)
	}
}

func TestGetEpicWithTags_ZeroAttachments(t *testing.T) {
	ctx := context.Background()

	epic := &models.Epic{
		BaseEntity: models.BaseEntity{ID: 5, Key: "E05"},
		Status:     models.EpicStatusActive,
	}
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return epic, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			return []string{}, nil
		},
	)

	svc := newEpicServiceWithTagSvc(repo, tagSvc)

	_, gotTags, err := svc.GetEpicWithTags(ctx, "E05")
	if err != nil {
		t.Fatalf("GetEpicWithTags() error = %v", err)
	}
	if gotTags == nil {
		t.Error("tags is nil, want non-nil empty slice")
	}
	if len(gotTags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(gotTags))
	}
}

func TestGetEpicWithTags_NilTagSvc(t *testing.T) {
	ctx := context.Background()

	epic := &models.Epic{
		BaseEntity: models.BaseEntity{ID: 3, Key: "E03"},
		Status:     models.EpicStatusActive,
	}
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return epic, nil
		},
	}

	svc := newEpicServiceWithTagSvc(repo, nil)

	gotEpic, gotTags, err := svc.GetEpicWithTags(ctx, "E03")
	if err != nil {
		t.Fatalf("GetEpicWithTags() nil tagSvc error = %v", err)
	}
	if gotEpic == nil {
		t.Fatal("epic is nil")
	}
	if gotTags != nil {
		t.Errorf("tags = %v, want nil", gotTags)
	}
}

func TestGetEpicWithTags_ListTagsError(t *testing.T) {
	ctx := context.Background()

	epic := &models.Epic{
		BaseEntity: models.BaseEntity{ID: 4, Key: "E04"},
		Status:     models.EpicStatusActive,
	}
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return epic, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			return nil, fmt.Errorf("tag error")
		},
	)

	svc := newEpicServiceWithTagSvc(repo, tagSvc)

	gotEpic, gotTags, err := svc.GetEpicWithTags(ctx, "E04")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if gotEpic != nil {
		t.Errorf("epic = %v, want nil on error (AC-T3)", gotEpic)
	}
	if gotTags != nil {
		t.Errorf("tags = %v, want nil", gotTags)
	}
}

// ============================================================================
// FeatureService.GetFeatureWithTags (AC-20 ×feature)
// ============================================================================

func TestGetFeatureWithTags_TwoAttachments(t *testing.T) {
	ctx := context.Background()

	feature := &models.Feature{
		BaseEntity: models.BaseEntity{ID: 20, Key: "E07-F01", Title: "My Feature"},
		Status:     models.FeatureStatus("active"),
	}
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return feature, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			if entityType != models.EntityTypeFeature {
				t.Errorf("ListTagsForEntity entityType = %q, want %q", entityType, models.EntityTypeFeature)
			}
			if entityID != 20 {
				t.Errorf("ListTagsForEntity entityID = %d, want 20", entityID)
			}
			return []string{"auth", "voice"}, nil
		},
	)

	svc := newFeatureServiceWithTagSvc(repo, nil, tagSvc)

	gotFeature, gotTags, err := svc.GetFeatureWithTags(ctx, "E07-F01")
	if err != nil {
		t.Fatalf("GetFeatureWithTags() error = %v", err)
	}
	if gotFeature == nil {
		t.Fatal("feature is nil")
	}
	wantTags := []string{"auth", "voice"}
	if !strSliceEq(gotTags, wantTags) {
		t.Errorf("tags = %v, want %v", gotTags, wantTags)
	}
}

func TestGetFeatureWithTags_ZeroAttachments(t *testing.T) {
	ctx := context.Background()

	feature := &models.Feature{
		BaseEntity: models.BaseEntity{ID: 21, Key: "E07-F02"},
		Status:     models.FeatureStatus("active"),
	}
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return feature, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			return []string{}, nil
		},
	)

	svc := newFeatureServiceWithTagSvc(repo, nil, tagSvc)

	_, gotTags, err := svc.GetFeatureWithTags(ctx, "E07-F02")
	if err != nil {
		t.Fatalf("GetFeatureWithTags() error = %v", err)
	}
	if gotTags == nil {
		t.Error("tags is nil, want non-nil empty slice")
	}
	if len(gotTags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(gotTags))
	}
}

func TestGetFeatureWithTags_NilTagSvc(t *testing.T) {
	ctx := context.Background()

	feature := &models.Feature{
		BaseEntity: models.BaseEntity{ID: 22, Key: "E07-F03"},
		Status:     models.FeatureStatus("active"),
	}
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return feature, nil
		},
	}

	svc := newFeatureServiceWithTagSvc(repo, nil, nil)

	gotFeature, gotTags, err := svc.GetFeatureWithTags(ctx, "E07-F03")
	if err != nil {
		t.Fatalf("GetFeatureWithTags() nil tagSvc error = %v", err)
	}
	if gotFeature == nil {
		t.Fatal("feature is nil")
	}
	if gotTags != nil {
		t.Errorf("tags = %v, want nil", gotTags)
	}
}

func TestGetFeatureWithTags_ListTagsError(t *testing.T) {
	ctx := context.Background()

	feature := &models.Feature{
		BaseEntity: models.BaseEntity{ID: 23, Key: "E07-F04"},
		Status:     models.FeatureStatus("active"),
	}
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return feature, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			return nil, fmt.Errorf("tag error")
		},
	)

	svc := newFeatureServiceWithTagSvc(repo, nil, tagSvc)

	gotFeature, gotTags, err := svc.GetFeatureWithTags(ctx, "E07-F04")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if gotFeature != nil {
		t.Errorf("feature = %v, want nil on error (AC-T3)", gotFeature)
	}
	if gotTags != nil {
		t.Errorf("tags = %v, want nil", gotTags)
	}
}

// ============================================================================
// BugService.GetBugWithTags (AC-20 ×bug)
// ============================================================================

func TestGetBugWithTags_TwoAttachments(t *testing.T) {
	ctx := context.Background()

	bug := &models.Bug{
		BaseEntity: models.BaseEntity{ID: 30, Key: "B001", Title: "Test bug"},
		Severity:   models.BugSeverityHigh,
		Status:     "open",
	}
	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return bug, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			if entityType != models.EntityTypeBug {
				t.Errorf("ListTagsForEntity entityType = %q, want %q", entityType, models.EntityTypeBug)
			}
			if entityID != 30 {
				t.Errorf("ListTagsForEntity entityID = %d, want 30", entityID)
			}
			return []string{"auth", "voice"}, nil
		},
	)

	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tagSvc)

	gotBug, gotTags, err := svc.GetBugWithTags(ctx, "B001")
	if err != nil {
		t.Fatalf("GetBugWithTags() error = %v", err)
	}
	if gotBug == nil {
		t.Fatal("bug is nil")
	}
	wantTags := []string{"auth", "voice"}
	if !strSliceEq(gotTags, wantTags) {
		t.Errorf("tags = %v, want %v", gotTags, wantTags)
	}
}

func TestGetBugWithTags_ZeroAttachments(t *testing.T) {
	ctx := context.Background()

	bug := &models.Bug{
		BaseEntity: models.BaseEntity{ID: 31, Key: "B002"},
		Severity:   models.BugSeverityLow,
		Status:     "open",
	}
	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return bug, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			return []string{}, nil
		},
	)

	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tagSvc)

	_, gotTags, err := svc.GetBugWithTags(ctx, "B002")
	if err != nil {
		t.Fatalf("GetBugWithTags() error = %v", err)
	}
	if gotTags == nil {
		t.Error("tags is nil, want non-nil empty slice")
	}
	if len(gotTags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(gotTags))
	}
}

func TestGetBugWithTags_NilTagSvc(t *testing.T) {
	ctx := context.Background()

	bug := &models.Bug{
		BaseEntity: models.BaseEntity{ID: 32, Key: "B003"},
		Severity:   models.BugSeverityMedium,
		Status:     "open",
	}
	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return bug, nil
		},
	}

	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, nil)

	gotBug, gotTags, err := svc.GetBugWithTags(ctx, "B003")
	if err != nil {
		t.Fatalf("GetBugWithTags() nil tagSvc error = %v", err)
	}
	if gotBug == nil {
		t.Fatal("bug is nil")
	}
	if gotTags != nil {
		t.Errorf("tags = %v, want nil", gotTags)
	}
}

func TestGetBugWithTags_ListTagsError(t *testing.T) {
	ctx := context.Background()

	bug := &models.Bug{
		BaseEntity: models.BaseEntity{ID: 33, Key: "B004"},
		Severity:   models.BugSeverityCritical,
		Status:     "open",
	}
	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return bug, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			return nil, fmt.Errorf("tag error")
		},
	)

	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tagSvc)

	gotBug, gotTags, err := svc.GetBugWithTags(ctx, "B004")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if gotBug != nil {
		t.Errorf("bug = %v, want nil on error (AC-T3)", gotBug)
	}
	if gotTags != nil {
		t.Errorf("tags = %v, want nil", gotTags)
	}
}

// ============================================================================
// ChangeCardService.GetChangeCardWithTags (AC-20 ×change)
// ============================================================================

func TestGetChangeCardWithTags_TwoAttachments(t *testing.T) {
	ctx := context.Background()

	card := &models.ChangeCard{
		BaseEntity: models.BaseEntity{ID: 40, Key: "CC-001", Title: "Test card"},
		Status:     "proposed",
	}
	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return card, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			if entityType != models.EntityTypeChange {
				t.Errorf("ListTagsForEntity entityType = %q, want %q", entityType, models.EntityTypeChange)
			}
			if entityID != 40 {
				t.Errorf("ListTagsForEntity entityID = %d, want 40", entityID)
			}
			return []string{"auth", "voice"}, nil
		},
	)

	svc := newChangeCardServiceWithTagSvc(repo, tagSvc)

	gotCard, gotTags, err := svc.GetChangeCardWithTags(ctx, "CC-001")
	if err != nil {
		t.Fatalf("GetChangeCardWithTags() error = %v", err)
	}
	if gotCard == nil {
		t.Fatal("change-card is nil")
	}
	wantTags := []string{"auth", "voice"}
	if !strSliceEq(gotTags, wantTags) {
		t.Errorf("tags = %v, want %v", gotTags, wantTags)
	}
}

func TestGetChangeCardWithTags_ZeroAttachments(t *testing.T) {
	ctx := context.Background()

	card := &models.ChangeCard{
		BaseEntity: models.BaseEntity{ID: 41, Key: "CC-002"},
		Status:     "proposed",
	}
	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return card, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			return []string{}, nil
		},
	)

	svc := newChangeCardServiceWithTagSvc(repo, tagSvc)

	_, gotTags, err := svc.GetChangeCardWithTags(ctx, "CC-002")
	if err != nil {
		t.Fatalf("GetChangeCardWithTags() error = %v", err)
	}
	if gotTags == nil {
		t.Error("tags is nil, want non-nil empty slice")
	}
	if len(gotTags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(gotTags))
	}
}

func TestGetChangeCardWithTags_NilTagSvc(t *testing.T) {
	ctx := context.Background()

	card := &models.ChangeCard{
		BaseEntity: models.BaseEntity{ID: 42, Key: "CC-003"},
		Status:     "proposed",
	}
	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return card, nil
		},
	}

	svc := newChangeCardServiceWithTagSvc(repo, nil)

	gotCard, gotTags, err := svc.GetChangeCardWithTags(ctx, "CC-003")
	if err != nil {
		t.Fatalf("GetChangeCardWithTags() nil tagSvc error = %v", err)
	}
	if gotCard == nil {
		t.Fatal("change-card is nil")
	}
	if gotTags != nil {
		t.Errorf("tags = %v, want nil", gotTags)
	}
}

func TestGetChangeCardWithTags_ListTagsError(t *testing.T) {
	ctx := context.Background()

	card := &models.ChangeCard{
		BaseEntity: models.BaseEntity{ID: 43, Key: "CC-004"},
		Status:     "proposed",
	}
	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return card, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			return nil, fmt.Errorf("tag error")
		},
	)

	svc := newChangeCardServiceWithTagSvc(repo, tagSvc)

	gotCard, gotTags, err := svc.GetChangeCardWithTags(ctx, "CC-004")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if gotCard != nil {
		t.Errorf("change-card = %v, want nil on error (AC-T3)", gotCard)
	}
	if gotTags != nil {
		t.Errorf("tags = %v, want nil", gotTags)
	}
}

// ============================================================================
// IdeaService.GetIdeaWithTags (AC-20 ×idea)
// ============================================================================

func TestGetIdeaWithTags_TwoAttachments(t *testing.T) {
	ctx := context.Background()

	idea := &models.Idea{
		ID:     50,
		Key:    "I-2026-01-15-01",
		Title:  "Test idea",
		Status: models.IdeaStatusNew,
	}
	repo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return idea, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			if entityType != models.EntityTypeIdea {
				t.Errorf("ListTagsForEntity entityType = %q, want %q", entityType, models.EntityTypeIdea)
			}
			if entityID != 50 {
				t.Errorf("ListTagsForEntity entityID = %d, want 50", entityID)
			}
			return []string{"auth", "voice"}, nil
		},
	)

	svc := newIdeaServiceWithTagSvc(repo, tagSvc)

	gotIdea, gotTags, err := svc.GetIdeaWithTags(ctx, "I-2026-01-15-01")
	if err != nil {
		t.Fatalf("GetIdeaWithTags() error = %v", err)
	}
	if gotIdea == nil {
		t.Fatal("idea is nil")
	}
	wantTags := []string{"auth", "voice"}
	if !strSliceEq(gotTags, wantTags) {
		t.Errorf("tags = %v, want %v", gotTags, wantTags)
	}
}

func TestGetIdeaWithTags_ZeroAttachments(t *testing.T) {
	ctx := context.Background()

	idea := &models.Idea{
		ID:     51,
		Key:    "I-2026-01-15-02",
		Status: models.IdeaStatusNew,
	}
	repo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return idea, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			return []string{}, nil
		},
	)

	svc := newIdeaServiceWithTagSvc(repo, tagSvc)

	_, gotTags, err := svc.GetIdeaWithTags(ctx, "I-2026-01-15-02")
	if err != nil {
		t.Fatalf("GetIdeaWithTags() error = %v", err)
	}
	if gotTags == nil {
		t.Error("tags is nil, want non-nil empty slice")
	}
	if len(gotTags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(gotTags))
	}
}

func TestGetIdeaWithTags_NilTagSvc(t *testing.T) {
	ctx := context.Background()

	idea := &models.Idea{
		ID:     52,
		Key:    "I-2026-01-15-03",
		Status: models.IdeaStatusNew,
	}
	repo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return idea, nil
		},
	}

	svc := newIdeaServiceWithTagSvc(repo, nil)

	gotIdea, gotTags, err := svc.GetIdeaWithTags(ctx, "I-2026-01-15-03")
	if err != nil {
		t.Fatalf("GetIdeaWithTags() nil tagSvc error = %v", err)
	}
	if gotIdea == nil {
		t.Fatal("idea is nil")
	}
	if gotTags != nil {
		t.Errorf("tags = %v, want nil", gotTags)
	}
}

func TestGetIdeaWithTags_ListTagsError(t *testing.T) {
	ctx := context.Background()

	idea := &models.Idea{
		ID:     53,
		Key:    "I-2026-01-15-04",
		Status: models.IdeaStatusNew,
	}
	repo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return idea, nil
		},
	}

	tagSvc := NewMockTagService().WithListTagsForEntityFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
			return nil, fmt.Errorf("tag error")
		},
	)

	svc := newIdeaServiceWithTagSvc(repo, tagSvc)

	gotIdea, gotTags, err := svc.GetIdeaWithTags(ctx, "I-2026-01-15-04")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if gotIdea != nil {
		t.Errorf("idea = %v, want nil on error (AC-T3)", gotIdea)
	}
	if gotTags != nil {
		t.Errorf("tags = %v, want nil", gotTags)
	}
}

// ============================================================================
// Test helper
// ============================================================================

// strSliceEq compares two string slices for equality.
func strSliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
