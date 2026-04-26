package services

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ---------------------------------------------------------------------------
// E28-F04 T-008 — Tag integration tests for EpicService.
//
// Mirrors the TaskService/BugService/FeatureService tag tests (spec.md
// AC-15..AC-18b, epic row) covering:
//   AC-15  ×epic: CreateEpic with no tags and no enforcement — EnforceRequired
//                 is invoked exactly once (fast path), AttachMany is NOT. The
//                 epic is persisted.
//   AC-15b ×epic: Nil tagSvc is tolerated (graceful degradation REQ-F-018).
//   AC-16  ×epic: TagRequiredError aborts BEFORE repo.Create.
//   AC-17  ×epic: Tags provided — persist-first, attach-after ordering.
//   AC-17b ×epic: AttachMany failure propagates unchanged; entity stays
//                 persisted (ADR-F04-2).
//   AC-18  ×epic: UpdateEpic with non-empty Tags calls AttachMany exactly
//                 once; DetachOne is never invoked on update.
//   AC-18b ×epic: nil and []string{} Tags on update are both no-ops.
//   AC-22  ×epic: No enforcement when tag_required_for only lists "task".
//
// All tests use the shared MockTagService (mock_tag_service_test.go) via the
// new SetTagService setter on EpicService — the constructor signature itself
// is unchanged, so existing EpicService tests continue to compile.
// ---------------------------------------------------------------------------

// newEpicServiceWithTagSvc wires an EpicService with the given mock tag
// service for E28-F04 tests. A nil tagSvc is passed through to exercise the
// graceful-degradation path (REQ-F-018).
func newEpicServiceWithTagSvc(repo *mockEpicRepo, tagSvc TagQuerier) *EpicService {
	svc := NewEpicService(repo, NewEntityService(newTestEpicWorkflowService()), epicRepoAsEntityRepo(repo), nil, nil)
	svc.SetTagService(tagSvc)
	return svc
}

// TestEpicService_CreateEpic_NoTagsAndNoRequirement covers AC-15 (epic row).
// When no tags are supplied and no enforcement is configured, the service
// MUST still invoke EnforceRequired exactly once (fast-path returning nil)
// and MUST NOT invoke AttachMany. The epic is persisted.
func TestEpicService_CreateEpic_NoTagsAndNoRequirement(t *testing.T) {
	ctx := context.Background()

	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil // nextEpicKey lookup
		},
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			return []*models.Epic{}, nil
		},
		createFn: func(ctx context.Context, epic *models.Epic) error {
			epic.ID = 1
			return nil
		},
	}

	tagSvc := NewMockTagService() // no enforcement; no tags
	svc := newEpicServiceWithTagSvc(repo, tagSvc)

	epic, err := svc.CreateEpic(ctx, CreateEpicInput{
		Title: "No tags here",
		Tags:  nil,
	})
	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}
	if epic == nil {
		t.Fatal("expected epic, got nil")
	}
	if tagSvc.EnforceRequiredCalls != 1 {
		t.Errorf("EnforceRequiredCalls = %d, want 1", tagSvc.EnforceRequiredCalls)
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 (no tags supplied)", tagSvc.AttachManyCalls)
	}
	if tagSvc.LastEnforceEntityType != models.EntityTypeEpic {
		t.Errorf("EnforceRequired entityType = %q, want %q",
			tagSvc.LastEnforceEntityType, models.EntityTypeEpic)
	}
}

// TestEpicService_CreateEpic_NilTagSvcIsSkippedCleanly covers AC-15b.
// Confirms the graceful-degradation property of REQ-F-018: a nil tagSvc
// must not panic or produce errors; tag hooks simply do not run.
func TestEpicService_CreateEpic_NilTagSvcIsSkippedCleanly(t *testing.T) {
	ctx := context.Background()

	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			return []*models.Epic{}, nil
		},
		createFn: func(ctx context.Context, epic *models.Epic) error {
			epic.ID = 1
			return nil
		},
	}
	// Explicit nil tagSvc — production code paths that predate F04 wiring.
	svc := newEpicServiceWithTagSvc(repo, nil)

	epic, err := svc.CreateEpic(ctx, CreateEpicInput{
		Title: "Nil tagSvc epic",
		Tags:  []string{"voice"}, // even with tags, nil svc is OK
	})
	if err != nil {
		t.Fatalf("CreateEpic() with nil tagSvc error = %v", err)
	}
	if epic == nil {
		t.Fatal("expected epic, got nil")
	}
}

// TestEpicService_CreateEpic_RequiredTypeMissingTagsAborts covers AC-16.
// When EnforceRequired returns *TagRequiredError, the service MUST return
// that error unchanged AND MUST NOT invoke repo.Create. This proves the
// pre-persistence ordering of the enforcement check (REQ-F-008).
func TestEpicService_CreateEpic_RequiredTypeMissingTagsAborts(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			return []*models.Epic{}, nil
		},
		createFn: func(ctx context.Context, epic *models.Epic) error {
			createCalled = true
			epic.ID = 1
			return nil
		},
	}

	tagSvc := NewMockTagService().WithEnforceRequiredFn(
		func(ctx context.Context, entityType models.EntityType, names []string) error {
			return &TagRequiredError{EntityType: string(entityType)}
		},
	)
	svc := newEpicServiceWithTagSvc(repo, tagSvc)

	_, err := svc.CreateEpic(ctx, CreateEpicInput{
		Title: "Should fail enforcement",
		Tags:  nil,
	})
	if err == nil {
		t.Fatal("expected TagRequiredError, got nil")
	}
	var required *TagRequiredError
	if !errors.As(err, &required) {
		t.Fatalf("expected *TagRequiredError, got %T: %v", err, err)
	}
	if required.EntityType != "epic" {
		t.Errorf("TagRequiredError.EntityType = %q, want %q", required.EntityType, "epic")
	}
	if createCalled {
		t.Error("repo.Create was invoked after enforcement failure (REQ-F-008 violation)")
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 after enforcement failure", tagSvc.AttachManyCalls)
	}
}

// TestEpicService_CreateEpic_TagsProvidedAttachAfterPersist covers AC-17.
// When tags are supplied, the service MUST:
//  1. Invoke EnforceRequired first (returns nil because tags present).
//  2. Persist the entity (repo.Create).
//  3. Invoke AttachMany AFTER the entity has an ID.
//
// The event log proves the exact ordering; AttachMany receives the post-
// insert ID.
func TestEpicService_CreateEpic_TagsProvidedAttachAfterPersist(t *testing.T) {
	ctx := context.Background()

	tagSvc := NewMockTagService()

	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			return []*models.Epic{}, nil
		},
		createFn: func(ctx context.Context, epic *models.Epic) error {
			epic.ID = 42
			tagSvc.RecordEvent("Create")
			return nil
		},
	}
	svc := newEpicServiceWithTagSvc(repo, tagSvc)

	_, err := svc.CreateEpic(ctx, CreateEpicInput{
		Title: "Epic with tags",
		Tags:  []string{"voice", "auth"},
	})
	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
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
	if tagSvc.LastAttachEntityType != models.EntityTypeEpic {
		t.Errorf("AttachMany entityType = %q, want %q",
			tagSvc.LastAttachEntityType, models.EntityTypeEpic)
	}
	// AC-17 ordering assertion: EnforceRequired → Create → AttachMany.
	gotEvents := tagSvc.EventsCopy()
	wantEvents := []string{"EnforceRequired", "Create", "AttachMany"}
	if !epicTagSliceEq(gotEvents, wantEvents) {
		t.Errorf("event order = %v, want %v", gotEvents, wantEvents)
	}
}

// TestEpicService_CreateEpic_AttachFailurePropagates covers AC-17b.
// When AttachMany fails (e.g., an unregistered tag), the error surfaces to
// the caller UNCHANGED and the entity REMAINS PERSISTED (matches ADR-F04-2:
// no transactions in F04; partial-write semantics accepted).
func TestEpicService_CreateEpic_AttachFailurePropagates(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			return []*models.Epic{}, nil
		},
		createFn: func(ctx context.Context, epic *models.Epic) error {
			createCalled = true
			epic.ID = 5
			return nil
		},
	}
	tagSvc := NewMockTagService().WithAttachManyFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error {
			return &UnregisteredTagError{Name: "ghost"}
		},
	)
	svc := newEpicServiceWithTagSvc(repo, tagSvc)

	_, err := svc.CreateEpic(ctx, CreateEpicInput{
		Title: "Attach will fail",
		Tags:  []string{"ghost"},
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

// TestEpicService_UpdateEpic_TagsAdditive covers AC-18.
// A non-empty updates.Tags triggers exactly one AttachMany call; DetachOne
// is NEVER invoked on update (removal goes through `shark epic tag rm`).
func TestEpicService_UpdateEpic_TagsAdditive(t *testing.T) {
	ctx := context.Background()

	existing := &models.Epic{
		BaseEntity: models.BaseEntity{ID: 7, Key: "E07", Title: "Existing"},
		Status:     models.EpicStatusActive,
		Priority:   models.Priority("medium"),
	}
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return existing, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error { return nil },
	}

	tagSvc := NewMockTagService()
	svc := newEpicServiceWithTagSvc(repo, tagSvc)

	_, err := svc.UpdateEpic(ctx, "E07", EpicUpdates{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("UpdateEpic() with tags error = %v", err)
	}
	if tagSvc.AttachManyCalls != 1 {
		t.Errorf("AttachManyCalls = %d, want 1", tagSvc.AttachManyCalls)
	}
	if tagSvc.DetachOneCalls != 0 {
		t.Errorf("DetachOneCalls = %d, want 0 (update is additive only)", tagSvc.DetachOneCalls)
	}
	if !epicTagSliceEq(tagSvc.LastAttachNames, []string{"voice"}) {
		t.Errorf("AttachMany names = %v, want [voice]", tagSvc.LastAttachNames)
	}
	if tagSvc.LastAttachEntityID != 7 {
		t.Errorf("AttachMany entityID = %d, want 7", tagSvc.LastAttachEntityID)
	}
	if tagSvc.LastAttachEntityType != models.EntityTypeEpic {
		t.Errorf("AttachMany entityType = %q, want %q",
			tagSvc.LastAttachEntityType, models.EntityTypeEpic)
	}
}

// TestEpicService_UpdateEpic_EmptyTagsIsNoOp covers AC-18b.
// Both nil and explicit empty-slice update.Tags must result in zero tag
// service calls. The update itself still proceeds (title/etc.).
func TestEpicService_UpdateEpic_EmptyTagsIsNoOp(t *testing.T) {
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
			existing := &models.Epic{
				BaseEntity: models.BaseEntity{ID: 7, Key: "E07", Title: "Existing"},
				Status:     models.EpicStatusActive,
				Priority:   models.Priority("medium"),
			}
			repo := &mockEpicRepo{
				getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
					return existing, nil
				},
				updateFn: func(ctx context.Context, epic *models.Epic) error { return nil },
			}
			tagSvc := NewMockTagService()
			svc := newEpicServiceWithTagSvc(repo, tagSvc)

			// Also change title to make the update meaningful.
			newTitle := "Updated"
			_, err := svc.UpdateEpic(ctx, "E07", EpicUpdates{
				Title: &newTitle,
				Tags:  tc.tags,
			})
			if err != nil {
				t.Fatalf("UpdateEpic() error = %v", err)
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

// TestEpicService_CreateEpic_AC22_NoEnforcementForEpicWhenOnlyTaskRequired
// covers the AC-22 epic-side portion: when tag_required_for lists only
// "task", creating an epic WITHOUT any tags must succeed (no enforcement
// applies for epics). The MockTagService's default EnforceRequired returns
// nil (happy path) because no fn override is configured, which correctly
// mimics the real TagService.EnforceRequired behaviour when the entity
// type is NOT in the configured list.
func TestEpicService_CreateEpic_AC22_NoEnforcementForEpicWhenOnlyTaskRequired(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			return []*models.Epic{}, nil
		},
		createFn: func(ctx context.Context, epic *models.Epic) error {
			createCalled = true
			epic.ID = 1
			return nil
		},
	}

	// Simulate a config where only "task" is listed in tag_required_for:
	// EnforceRequired receives entity type "epic", so the mock returns nil
	// just like the real service would.
	tagSvc := NewMockTagService().WithEnforceRequiredFn(
		func(ctx context.Context, entityType models.EntityType, names []string) error {
			// Only "task" is in the configured required list.
			required := []string{"task"}
			if len(names) > 0 {
				return nil
			}
			et := string(entityType)
			for _, r := range required {
				if r == et {
					return &TagRequiredError{EntityType: et}
				}
			}
			return nil
		},
	)
	svc := newEpicServiceWithTagSvc(repo, tagSvc)

	epic, err := svc.CreateEpic(ctx, CreateEpicInput{
		Title: "Epic without tags, task-only requirement",
		Tags:  nil,
	})
	if err != nil {
		t.Fatalf("CreateEpic() should succeed when only 'task' is required; got error: %v", err)
	}
	if epic == nil {
		t.Fatal("expected epic to be created")
	}
	if !createCalled {
		t.Error("expected repo.Create to be called when epic is not in tag_required_for")
	}
	if tagSvc.EnforceRequiredCalls != 1 {
		t.Errorf("EnforceRequiredCalls = %d, want 1", tagSvc.EnforceRequiredCalls)
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 (no tags supplied)", tagSvc.AttachManyCalls)
	}
}

// TestEpicService_UpdateEpic_AttachFailurePropagates covers the update-path
// variant of AC-17b: a failed AttachMany on UpdateEpic surfaces unchanged.
// The title change that preceded it has already been persisted (ADR-F04-2).
func TestEpicService_UpdateEpic_AttachFailurePropagates(t *testing.T) {
	ctx := context.Background()

	updateCalled := false
	existing := &models.Epic{
		BaseEntity: models.BaseEntity{ID: 7, Key: "E07", Title: "Existing"},
		Status:     models.EpicStatusActive,
		Priority:   models.Priority("medium"),
	}
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return existing, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			updateCalled = true
			return nil
		},
	}

	tagSvc := NewMockTagService().WithAttachManyFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error {
			return fmt.Errorf("db error")
		},
	)
	svc := newEpicServiceWithTagSvc(repo, tagSvc)

	_, err := svc.UpdateEpic(ctx, "E07", EpicUpdates{Tags: []string{"voice"}})
	if err == nil {
		t.Fatal("expected error from AttachMany, got nil")
	}
	if !updateCalled {
		t.Error("expected repo.Update to be called before AttachMany (ADR-F04-2)")
	}
}

// epicTagSliceEq is a helper used by the E28-F04 tag-integration tests
// above (duplicated from featureTagSliceEq to keep this test file self-
// contained).
func epicTagSliceEq(a, b []string) bool {
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
