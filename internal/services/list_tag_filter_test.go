package services

// ---------------------------------------------------------------------------
// E28-F05 T-006 — List service methods: tag filter integration tests.
//
// This file contains the TaskService canonical test suite (AC-11..AC-15,
// AC-30, AC-30b) and the representative happy-path test (AC-16) for each
// of the remaining six entity services.
//
// All tests use mocked repositories and the shared MockTagService from
// mock_tag_service_test.go.  No real database is used.
// ---------------------------------------------------------------------------

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// ---------------------------------------------------------------------------
// Helper: build a minimal TaskService with tagSvc wired for list tests.
// Uses the existing newTaskServiceWithTagSvc (task_service_tags_test.go),
// which accepts TagAttacher (now upgraded to TagQuerier via MockTagService).
// ---------------------------------------------------------------------------

func makeTasksForListFilter() []*models.Task {
	return []*models.Task{
		{BaseEntity: models.BaseEntity{ID: 101, Key: "T-E07-F01-001", Title: "Task 101"}, Status: "todo", Priority: 5},
		{BaseEntity: models.BaseEntity{ID: 102, Key: "T-E07-F01-002", Title: "Task 102"}, Status: "todo", Priority: 5},
		{BaseEntity: models.BaseEntity{ID: 103, Key: "T-E07-F01-003", Title: "Task 103"}, Status: "todo", Priority: 5},
		{BaseEntity: models.BaseEntity{ID: 104, Key: "T-E07-F01-004", Title: "Task 104"}, Status: "todo", Priority: 5},
		{BaseEntity: models.BaseEntity{ID: 105, Key: "T-E07-F01-005", Title: "Task 105"}, Status: "todo", Priority: 5},
	}
}

// ---------------------------------------------------------------------------
// AC-11: Happy path — tag filter returns matching subset.
// EntityIDsByTags returns [101,102]; base List returns all 5 tasks.
// Service intersects: returns tasks 101 and 102 only.
// ---------------------------------------------------------------------------

func TestListTasks_TagFilterReturnsMatchingSubset(t *testing.T) {
	ctx := context.Background()

	allTasks := makeTasksForListFilter()
	repo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}

	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(
		func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
			return []int64{101, 102}, nil
		},
	)
	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	result, err := svc.ListTasks(ctx, TaskFilters{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ListTasks() returned %d tasks, want 2", len(result))
	}
	if result[0].ID != 101 || result[1].ID != 102 {
		t.Errorf("ListTasks() IDs = [%d,%d], want [101,102]", result[0].ID, result[1].ID)
	}
	if tagSvc.EntityIDsByTagsCalls != 1 {
		t.Errorf("EntityIDsByTagsCalls = %d, want 1", tagSvc.EntityIDsByTagsCalls)
	}
}

// ---------------------------------------------------------------------------
// AC-12: Tag + status filter combined.
// EntityIDsByTags returns [101,102]; base List (after status filter) returns
// tasks 101 and 103 with status "in_progress". Intersection: only 101.
// ---------------------------------------------------------------------------

func TestListTasks_TagPlusStatusFilter(t *testing.T) {
	ctx := context.Background()

	inProgressTasks := []*models.Task{
		{BaseEntity: models.BaseEntity{ID: 101, Key: "T-E07-F01-001", Title: "Task 101"}, Status: "in_progress", Priority: 5},
		{BaseEntity: models.BaseEntity{ID: 103, Key: "T-E07-F01-003", Title: "Task 103"}, Status: "in_progress", Priority: 5},
	}
	repo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return inProgressTasks, nil
		},
	}

	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(
		func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
			return []int64{101, 102}, nil
		},
	)
	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	result, err := svc.ListTasks(ctx, TaskFilters{Tags: []string{"voice"}, Status: "in_progress"})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("ListTasks() returned %d tasks, want 1", len(result))
	}
	if result[0].ID != 101 {
		t.Errorf("ListTasks() ID = %d, want 101", result[0].ID)
	}
}

// ---------------------------------------------------------------------------
// AC-13: Two tags (AND semantics).
// EntityIDsByTags returns [101] (AND intersection). Service returns [task 101].
// ---------------------------------------------------------------------------

func TestListTasks_TwoTagsAndSemantics(t *testing.T) {
	ctx := context.Background()

	allTasks := makeTasksForListFilter()
	repo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}

	capturedNames := []string{}
	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(
		func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
			capturedNames = append(capturedNames, names...)
			return []int64{101}, nil
		},
	)
	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	result, err := svc.ListTasks(ctx, TaskFilters{Tags: []string{"voice", "auth"}})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(result) != 1 || result[0].ID != 101 {
		t.Errorf("ListTasks() result IDs = %v, want [101]", idsFromTasks(result))
	}
	// Verify EntityIDsByTags was called with correct entity type.
	if tagSvc.EntityIDsByTagsCalls != 1 {
		t.Errorf("EntityIDsByTagsCalls = %d, want 1", tagSvc.EntityIDsByTagsCalls)
	}
}

// ---------------------------------------------------------------------------
// AC-14: Short-circuit — EntityIDsByTags returns empty slice.
// Base List must NOT be called.
// ---------------------------------------------------------------------------

func TestListTasks_TagFilterZeroMatches(t *testing.T) {
	ctx := context.Background()

	listCalled := false
	repo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			listCalled = true
			return makeTasksForListFilter(), nil
		},
	}

	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(
		func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
			return []int64{}, nil // no matches
		},
	)
	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	result, err := svc.ListTasks(ctx, TaskFilters{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("ListTasks() returned %d tasks, want 0 (short-circuit)", len(result))
	}
	if result == nil {
		t.Error("ListTasks() returned nil, want non-nil empty slice")
	}
	if listCalled {
		t.Error("base List was called after EntityIDsByTags returned empty (REQ-F-017 short-circuit violation)")
	}
}

// ---------------------------------------------------------------------------
// AC-15: Unregistered tag error propagates unchanged.
// ---------------------------------------------------------------------------

func TestListTasks_UnregisteredTagPropagatesError(t *testing.T) {
	ctx := context.Background()

	repo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return makeTasksForListFilter(), nil
		},
	}

	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(
		func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
			return nil, &UnregisteredTagError{Name: "does-not-exist"}
		},
	)
	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	_, err := svc.ListTasks(ctx, TaskFilters{Tags: []string{"does-not-exist"}})
	if err == nil {
		t.Fatal("ListTasks() expected error, got nil")
	}
	var unregistered *UnregisteredTagError
	if !errors.As(err, &unregistered) {
		t.Errorf("ListTasks() error type = %T, want *UnregisteredTagError", err)
	}
}

// ---------------------------------------------------------------------------
// AC-30: Nil tagSvc with non-empty Tags filter → TagFilterUnavailableError.
// ---------------------------------------------------------------------------

func TestListTasks_NilTagSvcWithTagsFilter(t *testing.T) {
	ctx := context.Background()

	repo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return makeTasksForListFilter(), nil
		},
	}
	svc := newTaskServiceWithTagSvc(repo, nil)

	_, err := svc.ListTasks(ctx, TaskFilters{Tags: []string{"voice"}})
	if err == nil {
		t.Fatal("ListTasks() with nil tagSvc and Tags filter expected error, got nil")
	}
	var unavailable *TagFilterUnavailableError
	if !errors.As(err, &unavailable) {
		t.Errorf("ListTasks() error type = %T, want *TagFilterUnavailableError", err)
	}
}

// ---------------------------------------------------------------------------
// AC-30b: Nil tagSvc with nil Tags filter → normal operation, no error.
// ---------------------------------------------------------------------------

func TestListTasks_NilTagSvcWithNilFilter(t *testing.T) {
	ctx := context.Background()

	allTasks := makeTasksForListFilter()
	repo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}
	svc := newTaskServiceWithTagSvc(repo, nil)

	result, err := svc.ListTasks(ctx, TaskFilters{Tags: nil})
	if err != nil {
		t.Fatalf("ListTasks() with nil tags unexpected error = %v", err)
	}
	if len(result) != 5 {
		t.Errorf("ListTasks() returned %d tasks, want 5", len(result))
	}
}

// ---------------------------------------------------------------------------
// AC-16 × EpicService: representative happy-path test.
// ---------------------------------------------------------------------------

func TestListEpics_TagFilterReturnsMatchingSubset(t *testing.T) {
	ctx := context.Background()

	allEpics := []*models.Epic{
		{BaseEntity: models.BaseEntity{ID: 201, Key: "E01", Title: "Epic 201"}},
		{BaseEntity: models.BaseEntity{ID: 202, Key: "E02", Title: "Epic 202"}},
		{BaseEntity: models.BaseEntity{ID: 203, Key: "E03", Title: "Epic 203"}},
	}
	repo := &mockEpicRepo{
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			return allEpics, nil
		},
	}

	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(
		func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
			if entityType != models.EntityTypeEpic {
				t.Errorf("EntityIDsByTags called with entityType=%q, want %q", entityType, models.EntityTypeEpic)
			}
			return []int64{201, 202}, nil
		},
	)
	svc := newEpicServiceWithTagSvc(repo, tagSvc)

	result, err := svc.ListEpics(ctx, EpicFilters{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("ListEpics() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ListEpics() returned %d epics, want 2", len(result))
	}
	if result[0].ID != 201 || result[1].ID != 202 {
		t.Errorf("ListEpics() IDs = [%d,%d], want [201,202]", result[0].ID, result[1].ID)
	}
}

// ---------------------------------------------------------------------------
// AC-16 × FeatureService: ListFeatures.
// ---------------------------------------------------------------------------

func TestListFeatures_TagFilterReturnsMatchingSubset(t *testing.T) {
	ctx := context.Background()

	allFeatures := []*models.Feature{
		{BaseEntity: models.BaseEntity{ID: 301, Key: "E01-F01", Title: "Feature 301"}},
		{BaseEntity: models.BaseEntity{ID: 302, Key: "E01-F02", Title: "Feature 302"}},
		{BaseEntity: models.BaseEntity{ID: 303, Key: "E01-F03", Title: "Feature 303"}},
	}
	repo := &mockFeatureRepo{
		listFn: func(ctx context.Context) ([]*models.Feature, error) {
			return allFeatures, nil
		},
	}

	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(
		func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
			if entityType != models.EntityTypeFeature {
				t.Errorf("EntityIDsByTags called with entityType=%q, want %q", entityType, models.EntityTypeFeature)
			}
			return []int64{301, 302}, nil
		},
	)
	svc := newFeatureServiceWithTagSvc(repo, nil, tagSvc)

	result, err := svc.ListFeatures(ctx, FeatureFilters{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("ListFeatures() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ListFeatures() returned %d features, want 2", len(result))
	}
	if result[0].ID != 301 || result[1].ID != 302 {
		t.Errorf("ListFeatures() IDs = [%d,%d], want [301,302]", result[0].ID, result[1].ID)
	}
}

// ---------------------------------------------------------------------------
// AC-16 × FeatureService: ListFeaturesByEpicKey.
//
// EntityIDsByTags returns [301,302]; ListByEpic returns all 3 features for the
// given epic. Service intersects: returns features 301 and 302 only.
// ---------------------------------------------------------------------------

func TestListFeaturesByEpicKey_TagFilterReturnsMatchingSubset(t *testing.T) {
	ctx := context.Background()

	allFeatures := []*models.Feature{
		{BaseEntity: models.BaseEntity{ID: 301, Key: "E01-F01", Title: "Feature 301"}},
		{BaseEntity: models.BaseEntity{ID: 302, Key: "E01-F02", Title: "Feature 302"}},
		{BaseEntity: models.BaseEntity{ID: 303, Key: "E01-F03", Title: "Feature 303"}},
	}
	repo := &mockFeatureRepo{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return allFeatures, nil
		},
	}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}}, nil
		},
	}

	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(
		func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
			if entityType != models.EntityTypeFeature {
				t.Errorf("EntityIDsByTags called with entityType=%q, want %q", entityType, models.EntityTypeFeature)
			}
			return []int64{301, 302}, nil
		},
	)
	svc := newFeatureServiceWithTagSvc(repo, epicLookup, tagSvc)

	result, err := svc.ListFeaturesByEpicKey(ctx, FeatureFilters{EpicKey: "E01", Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("ListFeaturesByEpicKey() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ListFeaturesByEpicKey() returned %d features, want 2", len(result))
	}
	if result[0].ID != 301 || result[1].ID != 302 {
		t.Errorf("ListFeaturesByEpicKey() IDs = [%d,%d], want [301,302]", result[0].ID, result[1].ID)
	}
	if tagSvc.EntityIDsByTagsCalls != 1 {
		t.Errorf("EntityIDsByTagsCalls = %d, want 1", tagSvc.EntityIDsByTagsCalls)
	}
}

// ---------------------------------------------------------------------------
// AC-16 × BugService: ListBugs.
// ---------------------------------------------------------------------------

func TestListBugs_TagFilterReturnsMatchingSubset(t *testing.T) {
	ctx := context.Background()

	allBugs := []*models.Bug{
		{BaseEntity: models.BaseEntity{ID: 401, Key: "B001", Title: "Bug 401"}},
		{BaseEntity: models.BaseEntity{ID: 402, Key: "B002", Title: "Bug 402"}},
		{BaseEntity: models.BaseEntity{ID: 403, Key: "B003", Title: "Bug 403"}},
	}

	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(
		func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
			if entityType != models.EntityTypeBug {
				t.Errorf("EntityIDsByTags called with entityType=%q, want %q", entityType, models.EntityTypeBug)
			}
			return []int64{401, 402}, nil
		},
	)
	repo := &mockBugRepo{
		listFn: func(ctx context.Context, f *repository.BugListFilters) ([]*models.Bug, error) {
			return allBugs, nil
		},
	}
	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tagSvc)

	result, err := svc.ListBugs(ctx, BugFilters{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("ListBugs() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ListBugs() returned %d bugs, want 2", len(result))
	}
	if result[0].ID != 401 || result[1].ID != 402 {
		t.Errorf("ListBugs() IDs = [%d,%d], want [401,402]", result[0].ID, result[1].ID)
	}
}

// ---------------------------------------------------------------------------
// AC-16 × ChangeCardService: ListChangeCards.
// ---------------------------------------------------------------------------

func TestListChangeCards_TagFilterReturnsMatchingSubset(t *testing.T) {
	ctx := context.Background()

	allCards := []*models.ChangeCard{
		{BaseEntity: models.BaseEntity{ID: 501, Key: "CC-001", Title: "Card 501"}},
		{BaseEntity: models.BaseEntity{ID: 502, Key: "CC-002", Title: "Card 502"}},
		{BaseEntity: models.BaseEntity{ID: 503, Key: "CC-003", Title: "Card 503"}},
	}

	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(
		func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
			if entityType != models.EntityTypeChange {
				t.Errorf("EntityIDsByTags called with entityType=%q, want %q", entityType, models.EntityTypeChange)
			}
			return []int64{501, 502}, nil
		},
	)
	repo := &mockChangeCardRepo{
		listFn: func(ctx context.Context, filter *repository.ChangeCardRepoFilter) ([]*models.ChangeCard, error) {
			return allCards, nil
		},
	}
	svc := newChangeCardServiceWithTagSvc(repo, tagSvc)

	result, err := svc.ListChangeCards(ctx, ChangeCardFilters{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("ListChangeCards() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ListChangeCards() returned %d cards, want 2", len(result))
	}
	if result[0].ID != 501 || result[1].ID != 502 {
		t.Errorf("ListChangeCards() IDs = [%d,%d], want [501,502]", result[0].ID, result[1].ID)
	}
}

// ---------------------------------------------------------------------------
// AC-16 × IdeaService: ListIdeas.
// ---------------------------------------------------------------------------

func TestListIdeas_TagFilterReturnsMatchingSubset(t *testing.T) {
	ctx := context.Background()

	allIdeas := []*models.Idea{
		{ID: 601, Key: "1", Title: "Idea 601"},
		{ID: 602, Key: "2", Title: "Idea 602"},
		{ID: 603, Key: "3", Title: "Idea 603"},
	}

	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(
		func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
			if entityType != models.EntityTypeIdea {
				t.Errorf("EntityIDsByTags called with entityType=%q, want %q", entityType, models.EntityTypeIdea)
			}
			return []int64{601, 602}, nil
		},
	)
	repo := &MockIdeaRepository{
		ListFunc: func(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error) {
			return allIdeas, nil
		},
	}
	svc := newIdeaServiceWithTagSvc(repo, tagSvc)

	result, err := svc.ListIdeas(ctx, IdeaFilters{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("ListIdeas() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ListIdeas() returned %d ideas, want 2", len(result))
	}
	if result[0].ID != 601 || result[1].ID != 602 {
		t.Errorf("ListIdeas() IDs = [%d,%d], want [601,602]", result[0].ID, result[1].ID)
	}
}

// ---------------------------------------------------------------------------
// Helper: extract IDs from task slice.
// ---------------------------------------------------------------------------

func idsFromTasks(tasks []*models.Task) []int64 {
	ids := make([]int64, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	return ids
}
