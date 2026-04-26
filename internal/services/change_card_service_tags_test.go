package services

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ---------------------------------------------------------------------------
// E28-F04 T-009 — Tag integration tests for ChangeCardService.
//
// Mirrors the TaskService/FeatureService/EpicService/BugService tag tests
// (spec.md AC-15..AC-18b, change-card row) covering:
//   AC-15  ×change: CreateChangeCard with no tags and no enforcement —
//                  EnforceRequired is invoked exactly once (fast path),
//                  AttachMany is NOT. The change-card is persisted.
//   AC-15b ×change: Nil tagSvc is tolerated (graceful degradation REQ-F-018).
//   AC-16  ×change: TagRequiredError aborts BEFORE repo.Create.
//   AC-17  ×change: Tags provided — persist-first, attach-after ordering.
//   AC-17b ×change: AttachMany failure propagates unchanged; entity stays
//                  persisted (ADR-F04-2).
//   AC-18  ×change: UpdateChangeCard with non-empty Tags calls AttachMany
//                  exactly once; DetachOne is never invoked on update.
//   AC-18b ×change: nil and []string{} Tags on update are both no-ops.
//
// All tests use the shared MockTagService (mock_tag_service_test.go) via the
// new SetTagService setter on ChangeCardService — the constructor signature
// itself is unchanged, so existing ChangeCardService tests continue to
// compile.
// ---------------------------------------------------------------------------

// newChangeCardServiceWithTagSvc wires a ChangeCardService with the given
// mock tag service for E28-F04 tests. A nil tagSvc is passed through to
// exercise the graceful-degradation path (REQ-F-018).
func newChangeCardServiceWithTagSvc(repo *mockChangeCardRepo, tagSvc TagQuerier) *ChangeCardService {
	svc := newChangeCardService(repo, nil, nil)
	svc.SetTagService(tagSvc)
	return svc
}

// TestChangeCardService_CreateChangeCard_NoTagsAndNoRequirement covers AC-15
// (change row). When no tags are supplied and no enforcement is configured,
// the service MUST still invoke EnforceRequired exactly once (fast-path
// returning nil) and MUST NOT invoke AttachMany. The change-card is
// persisted.
func TestChangeCardService_CreateChangeCard_NoTagsAndNoRequirement(t *testing.T) {
	ctx := context.Background()

	repo := &mockChangeCardRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "CC-001", nil },
		createFn: func(ctx context.Context, card *models.ChangeCard) error {
			card.ID = 1
			return nil
		},
	}

	tagSvc := NewMockTagService() // no enforcement; no tags
	svc := newChangeCardServiceWithTagSvc(repo, tagSvc)

	card, err := svc.CreateChangeCard(ctx, CreateChangeCardInput{
		Title: "No tags here",
		Tags:  nil,
	})
	if err != nil {
		t.Fatalf("CreateChangeCard() error = %v", err)
	}
	if card == nil {
		t.Fatal("expected change-card, got nil")
	}
	if tagSvc.EnforceRequiredCalls != 1 {
		t.Errorf("EnforceRequiredCalls = %d, want 1", tagSvc.EnforceRequiredCalls)
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 (no tags supplied)", tagSvc.AttachManyCalls)
	}
	if tagSvc.LastEnforceEntityType != models.EntityTypeChange {
		t.Errorf("EnforceRequired entityType = %q, want %q",
			tagSvc.LastEnforceEntityType, models.EntityTypeChange)
	}
}

// TestChangeCardService_CreateChangeCard_NilTagSvcIsSkippedCleanly covers
// AC-15b. Confirms the graceful-degradation property of REQ-F-018: a nil
// tagSvc must not panic or produce errors; tag hooks simply do not run.
func TestChangeCardService_CreateChangeCard_NilTagSvcIsSkippedCleanly(t *testing.T) {
	ctx := context.Background()

	repo := &mockChangeCardRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "CC-001", nil },
		createFn: func(ctx context.Context, card *models.ChangeCard) error {
			card.ID = 1
			return nil
		},
	}
	// Explicit nil tagSvc — production code paths that predate F04 wiring.
	svc := newChangeCardServiceWithTagSvc(repo, nil)

	card, err := svc.CreateChangeCard(ctx, CreateChangeCardInput{
		Title: "Nil tagSvc card",
		Tags:  []string{"voice"}, // even with tags, nil svc is OK
	})
	if err != nil {
		t.Fatalf("CreateChangeCard() with nil tagSvc error = %v", err)
	}
	if card == nil {
		t.Fatal("expected change-card, got nil")
	}
}

// TestChangeCardService_CreateChangeCard_RequiredTypeMissingTagsAborts
// covers AC-16. When EnforceRequired returns *TagRequiredError, the service
// MUST return that error unchanged AND MUST NOT invoke repo.Create. This
// proves the pre-persistence ordering of the enforcement check (REQ-F-008).
func TestChangeCardService_CreateChangeCard_RequiredTypeMissingTagsAborts(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &mockChangeCardRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "CC-001", nil },
		createFn: func(ctx context.Context, card *models.ChangeCard) error {
			createCalled = true
			card.ID = 1
			return nil
		},
	}

	tagSvc := NewMockTagService().WithEnforceRequiredFn(
		func(ctx context.Context, entityType models.EntityType, names []string) error {
			return &TagRequiredError{EntityType: string(entityType)}
		},
	)
	svc := newChangeCardServiceWithTagSvc(repo, tagSvc)

	_, err := svc.CreateChangeCard(ctx, CreateChangeCardInput{
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
	if required.EntityType != "change" {
		t.Errorf("TagRequiredError.EntityType = %q, want %q", required.EntityType, "change")
	}
	if createCalled {
		t.Error("repo.Create was invoked after enforcement failure (REQ-F-008 violation)")
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 after enforcement failure", tagSvc.AttachManyCalls)
	}
}

// TestChangeCardService_CreateChangeCard_TagsProvidedAttachAfterPersist
// covers AC-17. When tags are supplied, the service MUST:
//  1. Invoke EnforceRequired first (returns nil because tags present).
//  2. Persist the entity (repo.Create).
//  3. Invoke AttachMany AFTER the entity has an ID.
//
// The event log proves the exact ordering; AttachMany receives the post-
// insert ID.
func TestChangeCardService_CreateChangeCard_TagsProvidedAttachAfterPersist(t *testing.T) {
	ctx := context.Background()

	tagSvc := NewMockTagService()

	repo := &mockChangeCardRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "CC-001", nil },
		createFn: func(ctx context.Context, card *models.ChangeCard) error {
			card.ID = 42
			tagSvc.RecordEvent("Create")
			return nil
		},
	}
	svc := newChangeCardServiceWithTagSvc(repo, tagSvc)

	_, err := svc.CreateChangeCard(ctx, CreateChangeCardInput{
		Title: "Change with tags",
		Tags:  []string{"voice", "auth"},
	})
	if err != nil {
		t.Fatalf("CreateChangeCard() error = %v", err)
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
	if tagSvc.LastAttachEntityType != models.EntityTypeChange {
		t.Errorf("AttachMany entityType = %q, want %q",
			tagSvc.LastAttachEntityType, models.EntityTypeChange)
	}
	// AC-17 ordering assertion: EnforceRequired → Create → AttachMany.
	gotEvents := tagSvc.EventsCopy()
	wantEvents := []string{"EnforceRequired", "Create", "AttachMany"}
	if !changeCardTagSliceEq(gotEvents, wantEvents) {
		t.Errorf("event order = %v, want %v", gotEvents, wantEvents)
	}
}

// TestChangeCardService_CreateChangeCard_AttachFailurePropagates covers
// AC-17b. When AttachMany fails (e.g., an unregistered tag), the error
// surfaces to the caller UNCHANGED and the entity REMAINS PERSISTED
// (matches ADR-F04-2: no transactions in F04; partial-write semantics
// accepted).
func TestChangeCardService_CreateChangeCard_AttachFailurePropagates(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &mockChangeCardRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "CC-001", nil },
		createFn: func(ctx context.Context, card *models.ChangeCard) error {
			createCalled = true
			card.ID = 5
			return nil
		},
	}
	tagSvc := NewMockTagService().WithAttachManyFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error {
			return &UnregisteredTagError{Name: "ghost"}
		},
	)
	svc := newChangeCardServiceWithTagSvc(repo, tagSvc)

	_, err := svc.CreateChangeCard(ctx, CreateChangeCardInput{
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

// TestChangeCardService_UpdateChangeCard_TagsAdditive covers AC-18.
// A non-empty updates.Tags triggers exactly one AttachMany call; DetachOne
// is NEVER invoked on update (removal goes through `shark change tag rm`).
func TestChangeCardService_UpdateChangeCard_TagsAdditive(t *testing.T) {
	ctx := context.Background()

	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{
				BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: "Existing"},
				Status:     "proposed",
				Priority:   5,
			}, nil
		},
		updateFn: func(ctx context.Context, card *models.ChangeCard) error { return nil },
	}

	tagSvc := NewMockTagService()
	svc := newChangeCardServiceWithTagSvc(repo, tagSvc)

	_, err := svc.UpdateChangeCard(ctx, "CC-001", ChangeCardUpdates{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("UpdateChangeCard() with tags error = %v", err)
	}
	if tagSvc.AttachManyCalls != 1 {
		t.Errorf("AttachManyCalls = %d, want 1", tagSvc.AttachManyCalls)
	}
	if tagSvc.DetachOneCalls != 0 {
		t.Errorf("DetachOneCalls = %d, want 0 (update is additive only)", tagSvc.DetachOneCalls)
	}
	if !changeCardTagSliceEq(tagSvc.LastAttachNames, []string{"voice"}) {
		t.Errorf("AttachMany names = %v, want [voice]", tagSvc.LastAttachNames)
	}
	if tagSvc.LastAttachEntityID != 1 {
		t.Errorf("AttachMany entityID = %d, want 1", tagSvc.LastAttachEntityID)
	}
	if tagSvc.LastAttachEntityType != models.EntityTypeChange {
		t.Errorf("AttachMany entityType = %q, want %q",
			tagSvc.LastAttachEntityType, models.EntityTypeChange)
	}
}

// TestChangeCardService_UpdateChangeCard_EmptyTagsIsNoOp covers AC-18b.
// Both nil and explicit empty-slice update.Tags must result in zero tag
// service calls. The update itself still proceeds (title/priority/etc.).
func TestChangeCardService_UpdateChangeCard_EmptyTagsIsNoOp(t *testing.T) {
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
			repo := &mockChangeCardRepo{
				getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
					return &models.ChangeCard{
						BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: "Existing"},
						Status:     "proposed",
						Priority:   5,
					}, nil
				},
				updateFn: func(ctx context.Context, card *models.ChangeCard) error { return nil },
			}
			tagSvc := NewMockTagService()
			svc := newChangeCardServiceWithTagSvc(repo, tagSvc)

			// Also change title to make the update meaningful.
			newTitle := "Updated"
			_, err := svc.UpdateChangeCard(ctx, "CC-001", ChangeCardUpdates{
				Title: &newTitle,
				Tags:  tc.tags,
			})
			if err != nil {
				t.Fatalf("UpdateChangeCard() error = %v", err)
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

// changeCardTagSliceEq is a helper used by the E28-F04 tag-integration
// tests above (duplicated from sliceEq in bug_service_test.go to keep this
// test file self-contained).
func changeCardTagSliceEq(a, b []string) bool {
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
