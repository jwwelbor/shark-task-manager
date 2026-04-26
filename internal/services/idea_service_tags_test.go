package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ---------------------------------------------------------------------------
// E28-F04 T-010 — Tag integration tests for IdeaService.
//
// Mirrors the TaskService/FeatureService/EpicService/BugService/
// ChangeCardService tag tests (spec.md AC-15..AC-18b, idea row) covering:
//   AC-15  ×idea: CreateIdea with no tags and no enforcement —
//                 EnforceRequired is invoked exactly once (fast path),
//                 AttachMany is NOT. The idea is persisted.
//   AC-15b ×idea: Nil tagSvc is tolerated (graceful degradation REQ-F-018).
//   AC-16  ×idea: TagRequiredError aborts BEFORE repo.Create.
//   AC-17  ×idea: Tags provided — persist-first, attach-after ordering.
//   AC-17b ×idea: AttachMany failure propagates unchanged; entity stays
//                 persisted (ADR-F04-2).
//   AC-18  ×idea: UpdateIdea with non-empty Tags calls AttachMany exactly
//                 once; DetachOne is never invoked on update.
//   AC-18b ×idea: nil and []string{} Tags on update are both no-ops.
//
// All tests use the shared MockTagService (mock_tag_service_test.go) via the
// new SetTagService setter on IdeaService — the constructor signature itself
// is unchanged, so existing IdeaService tests continue to compile.
// ---------------------------------------------------------------------------

// newIdeaServiceWithTagSvc wires an IdeaService with the given mock tag
// service for E28-F04 tests. A nil tagSvc is passed through to exercise the
// graceful-degradation path (REQ-F-018).
func newIdeaServiceWithTagSvc(repo *MockIdeaRepository, tagSvc TagQuerier) *IdeaService {
	svc, err := NewIdeaService(repo)
	if err != nil {
		panic(err)
	}
	svc.SetTagService(tagSvc)
	return svc
}

// TestIdeaService_CreateIdea_NoTagsAndNoRequirement covers AC-15 (idea row).
// When no tags are supplied and no enforcement is configured, the service
// MUST still invoke EnforceRequired exactly once (fast-path returning nil)
// and MUST NOT invoke AttachMany. The idea is persisted.
func TestIdeaService_CreateIdea_NoTagsAndNoRequirement(t *testing.T) {
	ctx := context.Background()
	today := time.Now().Format("2006-01-02")

	repo := &MockIdeaRepository{
		GetNextSequenceForDateFunc: func(ctx context.Context, dateStr string) (int, error) {
			if dateStr != today {
				t.Errorf("expected dateStr %q, got %q", today, dateStr)
			}
			return 1, nil
		},
		CreateFunc: func(ctx context.Context, idea *models.Idea) error {
			idea.ID = 1
			return nil
		},
	}

	tagSvc := NewMockTagService() // no enforcement; no tags
	svc := newIdeaServiceWithTagSvc(repo, tagSvc)

	idea, err := svc.CreateIdea(ctx, CreateIdeaInput{
		Title: "No tags here",
		Tags:  nil,
	})
	if err != nil {
		t.Fatalf("CreateIdea() error = %v", err)
	}
	if idea == nil {
		t.Fatal("expected idea, got nil")
	}
	if tagSvc.EnforceRequiredCalls != 1 {
		t.Errorf("EnforceRequiredCalls = %d, want 1", tagSvc.EnforceRequiredCalls)
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 (no tags supplied)", tagSvc.AttachManyCalls)
	}
	if tagSvc.LastEnforceEntityType != models.EntityTypeIdea {
		t.Errorf("EnforceRequired entityType = %q, want %q",
			tagSvc.LastEnforceEntityType, models.EntityTypeIdea)
	}
}

// TestIdeaService_CreateIdea_NilTagSvcIsSkippedCleanly covers AC-15b.
// Confirms the graceful-degradation property of REQ-F-018: a nil tagSvc
// must not panic or produce errors; tag hooks simply do not run.
func TestIdeaService_CreateIdea_NilTagSvcIsSkippedCleanly(t *testing.T) {
	ctx := context.Background()

	repo := &MockIdeaRepository{
		GetNextSequenceForDateFunc: func(ctx context.Context, dateStr string) (int, error) {
			return 1, nil
		},
		CreateFunc: func(ctx context.Context, idea *models.Idea) error {
			idea.ID = 1
			return nil
		},
	}
	// Explicit nil tagSvc — production code paths that predate F04 wiring.
	svc := newIdeaServiceWithTagSvc(repo, nil)

	idea, err := svc.CreateIdea(ctx, CreateIdeaInput{
		Title: "Nil tagSvc idea",
		Tags:  []string{"voice"}, // even with tags, nil svc is OK
	})
	if err != nil {
		t.Fatalf("CreateIdea() with nil tagSvc error = %v", err)
	}
	if idea == nil {
		t.Fatal("expected idea, got nil")
	}
}

// TestIdeaService_CreateIdea_RequiredTypeMissingTagsAborts covers AC-16.
// When EnforceRequired returns *TagRequiredError, the service MUST return
// that error unchanged AND MUST NOT invoke repo.Create. This proves the
// pre-persistence ordering of the enforcement check (REQ-F-008).
func TestIdeaService_CreateIdea_RequiredTypeMissingTagsAborts(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &MockIdeaRepository{
		GetNextSequenceForDateFunc: func(ctx context.Context, dateStr string) (int, error) {
			return 1, nil
		},
		CreateFunc: func(ctx context.Context, idea *models.Idea) error {
			createCalled = true
			idea.ID = 1
			return nil
		},
	}

	tagSvc := NewMockTagService().WithEnforceRequiredFn(
		func(ctx context.Context, entityType models.EntityType, names []string) error {
			return &TagRequiredError{EntityType: string(entityType)}
		},
	)
	svc := newIdeaServiceWithTagSvc(repo, tagSvc)

	_, err := svc.CreateIdea(ctx, CreateIdeaInput{
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
	if required.EntityType != "idea" {
		t.Errorf("TagRequiredError.EntityType = %q, want %q", required.EntityType, "idea")
	}
	if createCalled {
		t.Error("repo.Create was invoked after enforcement failure (REQ-F-008 violation)")
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 after enforcement failure", tagSvc.AttachManyCalls)
	}
}

// TestIdeaService_CreateIdea_TagsProvidedAttachAfterPersist covers AC-17.
// When tags are supplied, the service MUST:
//  1. Invoke EnforceRequired first (returns nil because tags present).
//  2. Persist the entity (repo.Create).
//  3. Invoke AttachMany AFTER the entity has an ID.
//
// The event log proves the exact ordering; AttachMany receives the post-
// insert ID.
func TestIdeaService_CreateIdea_TagsProvidedAttachAfterPersist(t *testing.T) {
	ctx := context.Background()

	tagSvc := NewMockTagService()

	repo := &MockIdeaRepository{
		GetNextSequenceForDateFunc: func(ctx context.Context, dateStr string) (int, error) {
			return 1, nil
		},
		CreateFunc: func(ctx context.Context, idea *models.Idea) error {
			idea.ID = 42
			tagSvc.RecordEvent("Create")
			return nil
		},
	}
	svc := newIdeaServiceWithTagSvc(repo, tagSvc)

	_, err := svc.CreateIdea(ctx, CreateIdeaInput{
		Title: "Idea with tags",
		Tags:  []string{"voice", "auth"},
	})
	if err != nil {
		t.Fatalf("CreateIdea() error = %v", err)
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
	if tagSvc.LastAttachEntityType != models.EntityTypeIdea {
		t.Errorf("AttachMany entityType = %q, want %q",
			tagSvc.LastAttachEntityType, models.EntityTypeIdea)
	}
	// AC-17 ordering assertion: EnforceRequired → Create → AttachMany.
	gotEvents := tagSvc.EventsCopy()
	wantEvents := []string{"EnforceRequired", "Create", "AttachMany"}
	if !ideaTagSliceEq(gotEvents, wantEvents) {
		t.Errorf("event order = %v, want %v", gotEvents, wantEvents)
	}
}

// TestIdeaService_CreateIdea_AttachFailurePropagates covers AC-17b.
// When AttachMany fails (e.g., an unregistered tag), the error surfaces to
// the caller UNCHANGED and the entity REMAINS PERSISTED (matches
// ADR-F04-2: no transactions in F04; partial-write semantics accepted).
func TestIdeaService_CreateIdea_AttachFailurePropagates(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &MockIdeaRepository{
		GetNextSequenceForDateFunc: func(ctx context.Context, dateStr string) (int, error) {
			return 1, nil
		},
		CreateFunc: func(ctx context.Context, idea *models.Idea) error {
			createCalled = true
			idea.ID = 5
			return nil
		},
	}
	tagSvc := NewMockTagService().WithAttachManyFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error {
			return &UnregisteredTagError{Name: "ghost"}
		},
	)
	svc := newIdeaServiceWithTagSvc(repo, tagSvc)

	_, err := svc.CreateIdea(ctx, CreateIdeaInput{
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

// TestIdeaService_UpdateIdea_TagsAdditive covers AC-18.
// A non-empty updates.Tags triggers exactly one AttachMany call; DetachOne
// is NEVER invoked on update (removal goes through `shark idea tag rm`).
func TestIdeaService_UpdateIdea_TagsAdditive(t *testing.T) {
	ctx := context.Background()

	repo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return &models.Idea{
				ID:     1,
				Key:    "I-2026-01-15-01",
				Title:  "Existing",
				Status: models.IdeaStatusNew,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, idea *models.Idea) error { return nil },
	}

	tagSvc := NewMockTagService()
	svc := newIdeaServiceWithTagSvc(repo, tagSvc)

	_, err := svc.UpdateIdea(ctx, "I-2026-01-15-01", UpdateIdeaInput{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("UpdateIdea() with tags error = %v", err)
	}
	if tagSvc.AttachManyCalls != 1 {
		t.Errorf("AttachManyCalls = %d, want 1", tagSvc.AttachManyCalls)
	}
	if tagSvc.DetachOneCalls != 0 {
		t.Errorf("DetachOneCalls = %d, want 0 (update is additive only)", tagSvc.DetachOneCalls)
	}
	if !ideaTagSliceEq(tagSvc.LastAttachNames, []string{"voice"}) {
		t.Errorf("AttachMany names = %v, want [voice]", tagSvc.LastAttachNames)
	}
	if tagSvc.LastAttachEntityID != 1 {
		t.Errorf("AttachMany entityID = %d, want 1", tagSvc.LastAttachEntityID)
	}
	if tagSvc.LastAttachEntityType != models.EntityTypeIdea {
		t.Errorf("AttachMany entityType = %q, want %q",
			tagSvc.LastAttachEntityType, models.EntityTypeIdea)
	}
}

// TestIdeaService_UpdateIdea_EmptyTagsIsNoOp covers AC-18b.
// Both nil and explicit empty-slice update.Tags must result in zero tag
// service calls. The update itself still proceeds (title/priority/etc.).
func TestIdeaService_UpdateIdea_EmptyTagsIsNoOp(t *testing.T) {
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
			repo := &MockIdeaRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
					return &models.Idea{
						ID:     1,
						Key:    "I-2026-01-15-01",
						Title:  "Existing",
						Status: models.IdeaStatusNew,
					}, nil
				},
				UpdateFunc: func(ctx context.Context, idea *models.Idea) error { return nil },
			}
			tagSvc := NewMockTagService()
			svc := newIdeaServiceWithTagSvc(repo, tagSvc)

			// Also change title to make the update meaningful.
			newTitle := "Updated"
			_, err := svc.UpdateIdea(ctx, "I-2026-01-15-01", UpdateIdeaInput{
				Title: &newTitle,
				Tags:  tc.tags,
			})
			if err != nil {
				t.Fatalf("UpdateIdea() error = %v", err)
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

// ideaTagSliceEq is a helper used by the E28-F04 tag-integration tests above
// (duplicated from sliceEq in bug_service_test.go / changeCardTagSliceEq in
// change_card_service_tags_test.go to keep this test file self-contained).
func ideaTagSliceEq(a, b []string) bool {
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
