package services

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ---------------------------------------------------------------------------
// E28-F04 T-006 — Tag integration tests for TaskService.
//
// Mirrors the BugService tag tests (task-plan.md §1.2 task row) covering:
//   AC-15  ×task: CreateTask with no tags and no enforcement — EnforceRequired
//                 is invoked exactly once (fast path), AttachMany is NOT.
//   AC-15b ×task: Nil tagSvc is tolerated (graceful degradation REQ-F-018).
//   AC-16  ×task: TagRequiredError aborts BEFORE repo.Create.
//   AC-17  ×task: Tags provided — persist-first, attach-after ordering.
//   AC-17b ×task: AttachMany failure propagates unchanged; entity stays persisted
//                 (ADR-F04-2).
//   AC-18  ×task: UpdateTask with non-empty Tags calls AttachMany exactly once;
//                 DetachOne is never invoked on update.
//   AC-18b ×task: nil and []string{} Tags on update are both no-ops.
//
// All tests use the shared MockTagService (mock_tag_service_test.go) via the
// new SetTagService setter on TaskService — the constructor signature itself
// is unchanged, so existing TaskService tests continue to compile.
// ---------------------------------------------------------------------------

// newTaskServiceWithTagSvc wires a TaskService with the given mock tag
// service for E28-F04 tests. A nil tagSvc is passed through to exercise
// the graceful-degradation path (REQ-F-018).
func newTaskServiceWithTagSvc(repo TaskRepository, tagSvc TagQuerier) *TaskService {
	svc := NewTaskService(repo, NewEntityService(newMockWorkflowService()), nil)
	svc.SetTagService(tagSvc)
	return svc
}

// TestTaskService_CreateTask_NoTagsAndNoRequirement covers AC-15 (task row).
// When no tags are supplied and no enforcement is configured, the service
// MUST still invoke EnforceRequired exactly once (fast-path returning nil)
// and MUST NOT invoke AttachMany. The task is persisted.
func TestTaskService_CreateTask_NoTagsAndNoRequirement(t *testing.T) {
	ctx := context.Background()

	repo := &MockTaskRepository{
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			task.ID = 1
			return nil
		},
	}

	tagSvc := NewMockTagService() // no enforcement; no tags
	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	task, err := svc.CreateTask(ctx, CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "No tags here",
		AgentType:  "backend",
		Priority:   5,
		Tags:       nil,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if task == nil {
		t.Fatal("expected task, got nil")
	}
	if tagSvc.EnforceRequiredCalls != 1 {
		t.Errorf("EnforceRequiredCalls = %d, want 1", tagSvc.EnforceRequiredCalls)
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 (no tags supplied)", tagSvc.AttachManyCalls)
	}
	if tagSvc.LastEnforceEntityType != models.EntityTypeTask {
		t.Errorf("EnforceRequired entityType = %q, want %q",
			tagSvc.LastEnforceEntityType, models.EntityTypeTask)
	}
}

// TestTaskService_CreateTask_NilTagSvcIsSkippedCleanly covers AC-15b.
// Confirms the graceful-degradation property of REQ-F-018: a nil tagSvc
// must not panic or produce errors; tag hooks simply do not run.
func TestTaskService_CreateTask_NilTagSvcIsSkippedCleanly(t *testing.T) {
	ctx := context.Background()

	repo := &MockTaskRepository{
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			task.ID = 1
			return nil
		},
	}
	// Explicit nil tagSvc — production code paths that predate F04 wiring.
	svc := newTaskServiceWithTagSvc(repo, nil)

	task, err := svc.CreateTask(ctx, CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "Nil tagSvc task",
		AgentType:  "backend",
		Priority:   5,
		Tags:       []string{"voice"}, // even with tags, nil svc is OK
	})
	if err != nil {
		t.Fatalf("CreateTask() with nil tagSvc error = %v", err)
	}
	if task == nil {
		t.Fatal("expected task, got nil")
	}
}

// TestTaskService_CreateTask_RequiredTypeMissingTagsAborts covers AC-16.
// When EnforceRequired returns *TagRequiredError, the service MUST return
// that error unchanged AND MUST NOT invoke repo.Create. This proves the
// pre-persistence ordering of the enforcement check (REQ-F-008).
func TestTaskService_CreateTask_RequiredTypeMissingTagsAborts(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &MockTaskRepository{
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			createCalled = true
			task.ID = 1
			return nil
		},
	}

	tagSvc := NewMockTagService().WithEnforceRequiredFn(
		func(ctx context.Context, entityType models.EntityType, names []string) error {
			return &TagRequiredError{EntityType: string(entityType)}
		},
	)
	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	_, err := svc.CreateTask(ctx, CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "Should fail enforcement",
		AgentType:  "backend",
		Priority:   5,
		Tags:       nil,
	})
	if err == nil {
		t.Fatal("expected TagRequiredError, got nil")
	}
	var required *TagRequiredError
	if !errors.As(err, &required) {
		t.Fatalf("expected *TagRequiredError, got %T: %v", err, err)
	}
	if required.EntityType != "task" {
		t.Errorf("TagRequiredError.EntityType = %q, want %q", required.EntityType, "task")
	}
	if createCalled {
		t.Error("repo.Create was invoked after enforcement failure (REQ-F-008 violation)")
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 after enforcement failure", tagSvc.AttachManyCalls)
	}
}

// TestTaskService_CreateTask_TagsProvidedAttachAfterPersist covers AC-17.
// When tags are supplied, the service MUST:
//  1. Invoke EnforceRequired first (returns nil because tags present).
//  2. Persist the entity (repo.Create).
//  3. Invoke AttachMany AFTER the entity has an ID.
//
// The event log proves the exact ordering; AttachMany receives the post-
// insert ID.
func TestTaskService_CreateTask_TagsProvidedAttachAfterPersist(t *testing.T) {
	ctx := context.Background()

	tagSvc := NewMockTagService()

	repo := &MockTaskRepository{
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			task.ID = 42
			tagSvc.RecordEvent("Create")
			return nil
		},
	}
	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	_, err := svc.CreateTask(ctx, CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "Task with tags",
		AgentType:  "backend",
		Priority:   5,
		Tags:       []string{"voice", "auth"},
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if tagSvc.EnforceRequiredCalls != 1 {
		t.Errorf("EnforceRequiredCalls = %d, want 1", tagSvc.EnforceRequiredCalls)
	}
	if tagSvc.AttachManyCalls != 1 {
		t.Errorf("AttachManyCalls = %d, want 1", tagSvc.AttachManyCalls)
	}
	if tagSvc.LastAttachEntityID != 42 {
		t.Errorf("AttachMany entityID = %d, want 42 (post-insert id)", tagSvc.LastAttachEntityID)
	}
	if tagSvc.LastAttachEntityType != models.EntityTypeTask {
		t.Errorf("AttachMany entityType = %q, want %q",
			tagSvc.LastAttachEntityType, models.EntityTypeTask)
	}
	// AC-17 ordering assertion: EnforceRequired → Create → AttachMany.
	gotEvents := tagSvc.EventsCopy()
	wantEvents := []string{"EnforceRequired", "Create", "AttachMany"}
	if !taskTagSliceEq(gotEvents, wantEvents) {
		t.Errorf("event order = %v, want %v", gotEvents, wantEvents)
	}
}

// TestTaskService_CreateTask_AttachFailurePropagates covers AC-17b.
// When AttachMany fails (e.g., an unregistered tag), the error surfaces
// to the caller UNCHANGED and the entity REMAINS PERSISTED (matches
// ADR-F04-2: no transactions in F04; partial-write semantics accepted).
func TestTaskService_CreateTask_AttachFailurePropagates(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &MockTaskRepository{
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			createCalled = true
			task.ID = 5
			return nil
		},
	}
	tagSvc := NewMockTagService().WithAttachManyFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error {
			return &UnregisteredTagError{Name: "ghost"}
		},
	)
	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	_, err := svc.CreateTask(ctx, CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "Attach will fail",
		AgentType:  "backend",
		Priority:   5,
		Tags:       []string{"ghost"},
	})
	if err == nil {
		t.Fatal("expected UnregisteredTagError, got nil")
	}
	var unregistered *UnregisteredTagError
	if !errors.As(err, &unregistered) {
		t.Fatalf("expected *UnregisteredTagError unchanged, got %T: %v", err, err)
	}
	if !createCalled {
		t.Error("entity was not persisted before AttachMany failure (expected persisted per ADR-F04-2)")
	}
}

// TestTaskService_UpdateTask_TagsAdditive covers AC-18.
// A non-empty updates.Tags triggers exactly one AttachMany call; DetachOne
// is NEVER invoked on update (removal goes through `shark task tag rm`).
func TestTaskService_UpdateTask_TagsAdditive(t *testing.T) {
	ctx := context.Background()

	repo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 1, Key: "T-E07-F01-001", Title: "Existing"},
				Status:     models.TaskStatus("todo"),
				Priority:   5,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error { return nil },
	}

	tagSvc := NewMockTagService()
	svc := newTaskServiceWithTagSvc(repo, tagSvc)

	_, err := svc.UpdateTask(ctx, "E07-F01-001", TaskUpdates{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("UpdateTask() with tags error = %v", err)
	}
	if tagSvc.AttachManyCalls != 1 {
		t.Errorf("AttachManyCalls = %d, want 1", tagSvc.AttachManyCalls)
	}
	if tagSvc.DetachOneCalls != 0 {
		t.Errorf("DetachOneCalls = %d, want 0 (update is additive only)", tagSvc.DetachOneCalls)
	}
	if !taskTagSliceEq(tagSvc.LastAttachNames, []string{"voice"}) {
		t.Errorf("AttachMany names = %v, want [voice]", tagSvc.LastAttachNames)
	}
	if tagSvc.LastAttachEntityID != 1 {
		t.Errorf("AttachMany entityID = %d, want 1", tagSvc.LastAttachEntityID)
	}
	if tagSvc.LastAttachEntityType != models.EntityTypeTask {
		t.Errorf("AttachMany entityType = %q, want %q",
			tagSvc.LastAttachEntityType, models.EntityTypeTask)
	}
}

// TestTaskService_UpdateTask_EmptyTagsIsNoOp covers AC-18b.
// Both nil and explicit empty-slice update.Tags must result in zero tag
// service calls. The update itself still proceeds (title/priority/etc.).
func TestTaskService_UpdateTask_EmptyTagsIsNoOp(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		tags []string
	}{
		{"nil tags", nil},
		{"empty slice tags", []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockTaskRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
					return &models.Task{
						BaseEntity: models.BaseEntity{ID: 1, Key: "T-E07-F01-001", Title: "Existing"},
						Status:     models.TaskStatus("todo"),
						Priority:   5,
					}, nil
				},
				UpdateFunc: func(ctx context.Context, task *models.Task) error { return nil },
			}
			tagSvc := NewMockTagService()
			svc := newTaskServiceWithTagSvc(repo, tagSvc)

			// Also change title to make the update meaningful.
			newTitle := "Updated"
			_, err := svc.UpdateTask(ctx, "E07-F01-001", TaskUpdates{
				Title: &newTitle,
				Tags:  tc.tags,
			})
			if err != nil {
				t.Fatalf("UpdateTask() error = %v", err)
			}
			if tagSvc.AttachManyCalls != 0 {
				t.Errorf("AttachManyCalls = %d, want 0 for %s", tagSvc.AttachManyCalls, tc.name)
			}
			if tagSvc.DetachOneCalls != 0 {
				t.Errorf("DetachOneCalls = %d, want 0 for %s", tagSvc.DetachOneCalls, tc.name)
			}
			if tagSvc.EnforceRequiredCalls != 0 {
				t.Errorf("EnforceRequiredCalls = %d, want 0 on update for %s",
					tagSvc.EnforceRequiredCalls, tc.name)
			}
		})
	}
}

// taskTagSliceEq is a helper used by the E28-F04 tag-integration tests
// above (duplicated from sliceEq in bug_service_test.go to keep this
// test file self-contained).
func taskTagSliceEq(a, b []string) bool {
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
