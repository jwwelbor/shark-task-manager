package services

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ---------------------------------------------------------------------------
// E28-F04 T-007 — Tag integration tests for FeatureService.
//
// Mirrors the TaskService/BugService tag tests (spec.md AC-15..AC-18b, feature
// row) covering:
//   AC-15  ×feature: CreateFeature with no tags and no enforcement —
//                   EnforceRequired is invoked exactly once (fast path),
//                   AttachMany is NOT. The feature is persisted.
//   AC-15b ×feature: Nil tagSvc is tolerated (graceful degradation REQ-F-018).
//   AC-16  ×feature: TagRequiredError aborts BEFORE repo.Create.
//   AC-17  ×feature: Tags provided — persist-first, attach-after ordering.
//   AC-17b ×feature: AttachMany failure propagates unchanged; entity stays
//                   persisted (ADR-F04-2).
//   AC-18  ×feature: UpdateFeature with non-empty Tags calls AttachMany
//                   exactly once; DetachOne is never invoked on update.
//   AC-18b ×feature: nil and []string{} Tags on update are both no-ops.
//
// All tests use the shared MockTagService (mock_tag_service_test.go) via the
// new SetTagService setter on FeatureService — the constructor signature
// itself is unchanged, so existing FeatureService tests continue to compile.
// ---------------------------------------------------------------------------

// newFeatureServiceWithTagSvc wires a FeatureService with the given mock tag
// service for E28-F04 tests. A nil tagSvc is passed through to exercise the
// graceful-degradation path (REQ-F-018).
func newFeatureServiceWithTagSvc(repo *mockFeatureRepo, epicLookup FeatureEpicLookup, tagSvc TagQuerier) *FeatureService {
	svc := NewFeatureService(repo, NewEntityService(newTestFeatureWorkflowService()), featureRepoAsEntityRepo(repo), nil, epicLookup)
	svc.SetTagService(tagSvc)
	return svc
}

// TestFeatureService_CreateFeature_NoTagsAndNoRequirement covers AC-15
// (feature row). When no tags are supplied and no enforcement is configured,
// the service MUST still invoke EnforceRequired exactly once (fast-path
// returning nil) and MUST NOT invoke AttachMany. The feature is persisted.
func TestFeatureService_CreateFeature_NoTagsAndNoRequirement(t *testing.T) {
	ctx := context.Background()

	repo := &mockFeatureRepo{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		},
		createFn: func(ctx context.Context, feature *models.Feature) error {
			feature.ID = 1
			return nil
		},
	}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}, Status: models.EpicStatusActive}, nil
		},
	}

	tagSvc := NewMockTagService() // no enforcement; no tags
	svc := newFeatureServiceWithTagSvc(repo, epicLookup, tagSvc)

	feature, err := svc.CreateFeature(ctx, CreateFeatureInput{
		EpicKey: "E01",
		Title:   "No tags here",
		Tags:    nil,
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if feature == nil {
		t.Fatal("expected feature, got nil")
	}
	if tagSvc.EnforceRequiredCalls != 1 {
		t.Errorf("EnforceRequiredCalls = %d, want 1", tagSvc.EnforceRequiredCalls)
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 (no tags supplied)", tagSvc.AttachManyCalls)
	}
	if tagSvc.LastEnforceEntityType != models.EntityTypeFeature {
		t.Errorf("EnforceRequired entityType = %q, want %q",
			tagSvc.LastEnforceEntityType, models.EntityTypeFeature)
	}
}

// TestFeatureService_CreateFeature_NilTagSvcIsSkippedCleanly covers AC-15b.
// Confirms the graceful-degradation property of REQ-F-018: a nil tagSvc
// must not panic or produce errors; tag hooks simply do not run.
func TestFeatureService_CreateFeature_NilTagSvcIsSkippedCleanly(t *testing.T) {
	ctx := context.Background()

	repo := &mockFeatureRepo{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		},
		createFn: func(ctx context.Context, feature *models.Feature) error {
			feature.ID = 1
			return nil
		},
	}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}, Status: models.EpicStatusActive}, nil
		},
	}
	// Explicit nil tagSvc — production code paths that predate F04 wiring.
	svc := newFeatureServiceWithTagSvc(repo, epicLookup, nil)

	feature, err := svc.CreateFeature(ctx, CreateFeatureInput{
		EpicKey: "E01",
		Title:   "Nil tagSvc feature",
		Tags:    []string{"voice"}, // even with tags, nil svc is OK
	})
	if err != nil {
		t.Fatalf("CreateFeature() with nil tagSvc error = %v", err)
	}
	if feature == nil {
		t.Fatal("expected feature, got nil")
	}
}

// TestFeatureService_CreateFeature_RequiredTypeMissingTagsAborts covers
// AC-16. When EnforceRequired returns *TagRequiredError, the service MUST
// return that error unchanged AND MUST NOT invoke repo.Create. This proves
// the pre-persistence ordering of the enforcement check (REQ-F-008).
func TestFeatureService_CreateFeature_RequiredTypeMissingTagsAborts(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &mockFeatureRepo{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		},
		createFn: func(ctx context.Context, feature *models.Feature) error {
			createCalled = true
			feature.ID = 1
			return nil
		},
	}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}, Status: models.EpicStatusActive}, nil
		},
	}

	tagSvc := NewMockTagService().WithEnforceRequiredFn(
		func(ctx context.Context, entityType models.EntityType, names []string) error {
			return &TagRequiredError{EntityType: string(entityType)}
		},
	)
	svc := newFeatureServiceWithTagSvc(repo, epicLookup, tagSvc)

	_, err := svc.CreateFeature(ctx, CreateFeatureInput{
		EpicKey: "E01",
		Title:   "Should fail enforcement",
		Tags:    nil,
	})
	if err == nil {
		t.Fatal("expected TagRequiredError, got nil")
	}
	var required *TagRequiredError
	if !errors.As(err, &required) {
		t.Fatalf("expected *TagRequiredError, got %T: %v", err, err)
	}
	if required.EntityType != "feature" {
		t.Errorf("TagRequiredError.EntityType = %q, want %q", required.EntityType, "feature")
	}
	if createCalled {
		t.Error("repo.Create was invoked after enforcement failure (REQ-F-008 violation)")
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 after enforcement failure", tagSvc.AttachManyCalls)
	}
}

// TestFeatureService_CreateFeature_TagsProvidedAttachAfterPersist covers
// AC-17. When tags are supplied, the service MUST:
//  1. Invoke EnforceRequired first (returns nil because tags present).
//  2. Persist the entity (repo.Create).
//  3. Invoke AttachMany AFTER the entity has an ID.
//
// The event log proves the exact ordering; AttachMany receives the post-
// insert ID.
func TestFeatureService_CreateFeature_TagsProvidedAttachAfterPersist(t *testing.T) {
	ctx := context.Background()

	tagSvc := NewMockTagService()

	repo := &mockFeatureRepo{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		},
		createFn: func(ctx context.Context, feature *models.Feature) error {
			feature.ID = 42
			tagSvc.RecordEvent("Create")
			return nil
		},
	}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}, Status: models.EpicStatusActive}, nil
		},
	}
	svc := newFeatureServiceWithTagSvc(repo, epicLookup, tagSvc)

	_, err := svc.CreateFeature(ctx, CreateFeatureInput{
		EpicKey: "E01",
		Title:   "Feature with tags",
		Tags:    []string{"voice", "auth"},
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
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
	if tagSvc.LastAttachEntityType != models.EntityTypeFeature {
		t.Errorf("AttachMany entityType = %q, want %q",
			tagSvc.LastAttachEntityType, models.EntityTypeFeature)
	}
	// AC-17 ordering assertion: EnforceRequired → Create → AttachMany.
	gotEvents := tagSvc.EventsCopy()
	wantEvents := []string{"EnforceRequired", "Create", "AttachMany"}
	if !featureTagSliceEq(gotEvents, wantEvents) {
		t.Errorf("event order = %v, want %v", gotEvents, wantEvents)
	}
}

// TestFeatureService_CreateFeature_AttachFailurePropagates covers AC-17b.
// When AttachMany fails (e.g., an unregistered tag), the error surfaces to
// the caller UNCHANGED and the entity REMAINS PERSISTED (matches ADR-F04-2:
// no transactions in F04; partial-write semantics accepted).
func TestFeatureService_CreateFeature_AttachFailurePropagates(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &mockFeatureRepo{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		},
		createFn: func(ctx context.Context, feature *models.Feature) error {
			createCalled = true
			feature.ID = 5
			return nil
		},
	}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}, Status: models.EpicStatusActive}, nil
		},
	}
	tagSvc := NewMockTagService().WithAttachManyFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error {
			return &UnregisteredTagError{Name: "ghost"}
		},
	)
	svc := newFeatureServiceWithTagSvc(repo, epicLookup, tagSvc)

	_, err := svc.CreateFeature(ctx, CreateFeatureInput{
		EpicKey: "E01",
		Title:   "Attach will fail",
		Tags:    []string{"ghost"},
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

// TestFeatureService_UpdateFeature_TagsAdditive covers AC-18.
// A non-empty updates.Tags triggers exactly one AttachMany call; DetachOne
// is NEVER invoked on update (removal goes through `shark feature tag rm`).
func TestFeatureService_UpdateFeature_TagsAdditive(t *testing.T) {
	ctx := context.Background()

	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				BaseEntity: models.BaseEntity{ID: 1, Key: "E07-F01", Title: "Existing"},
				EpicID:     1,
				Status:     models.FeatureStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error { return nil },
	}

	tagSvc := NewMockTagService()
	svc := newFeatureServiceWithTagSvc(repo, nil, tagSvc)

	_, err := svc.UpdateFeature(ctx, "E07-F01", FeatureUpdates{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("UpdateFeature() with tags error = %v", err)
	}
	if tagSvc.AttachManyCalls != 1 {
		t.Errorf("AttachManyCalls = %d, want 1", tagSvc.AttachManyCalls)
	}
	if tagSvc.DetachOneCalls != 0 {
		t.Errorf("DetachOneCalls = %d, want 0 (update is additive only)", tagSvc.DetachOneCalls)
	}
	if !featureTagSliceEq(tagSvc.LastAttachNames, []string{"voice"}) {
		t.Errorf("AttachMany names = %v, want [voice]", tagSvc.LastAttachNames)
	}
	if tagSvc.LastAttachEntityID != 1 {
		t.Errorf("AttachMany entityID = %d, want 1", tagSvc.LastAttachEntityID)
	}
	if tagSvc.LastAttachEntityType != models.EntityTypeFeature {
		t.Errorf("AttachMany entityType = %q, want %q",
			tagSvc.LastAttachEntityType, models.EntityTypeFeature)
	}
}

// TestFeatureService_UpdateFeature_EmptyTagsIsNoOp covers AC-18b.
// Both nil and explicit empty-slice update.Tags must result in zero tag
// service calls. The update itself still proceeds (title/etc.).
func TestFeatureService_UpdateFeature_EmptyTagsIsNoOp(t *testing.T) {
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
			repo := &mockFeatureRepo{
				getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
					return &models.Feature{
						BaseEntity: models.BaseEntity{ID: 1, Key: "E07-F01", Title: "Existing"},
						EpicID:     1,
						Status:     models.FeatureStatusDraft,
					}, nil
				},
				updateFn: func(ctx context.Context, feature *models.Feature) error { return nil },
			}
			tagSvc := NewMockTagService()
			svc := newFeatureServiceWithTagSvc(repo, nil, tagSvc)

			// Also change title to make the update meaningful.
			newTitle := "Updated"
			_, err := svc.UpdateFeature(ctx, "E07-F01", FeatureUpdates{
				Title: &newTitle,
				Tags:  tc.tags,
			})
			if err != nil {
				t.Fatalf("UpdateFeature() error = %v", err)
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

// featureTagSliceEq is a helper used by the E28-F04 tag-integration tests
// above (duplicated from sliceEq in bug_service_test.go to keep this
// test file self-contained).
func featureTagSliceEq(a, b []string) bool {
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
