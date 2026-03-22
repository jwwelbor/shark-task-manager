package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// mockEntityDocumentRepo implements EntityDocumentRepository for testing.
type mockEntityDocumentRepo struct {
	createOrGetFn func(ctx context.Context, title, filePath string) (*models.Document, error)
	getByTitleFn  func(ctx context.Context, title string) (*models.Document, error)
}

func (m *mockEntityDocumentRepo) CreateOrGet(ctx context.Context, title, filePath string) (*models.Document, error) {
	if m.createOrGetFn != nil {
		return m.createOrGetFn(ctx, title, filePath)
	}
	return nil, fmt.Errorf("CreateOrGet not implemented")
}

func (m *mockEntityDocumentRepo) GetByTitle(ctx context.Context, title string) (*models.Document, error) {
	if m.getByTitleFn != nil {
		return m.getByTitleFn(ctx, title)
	}
	return nil, fmt.Errorf("GetByTitle not implemented")
}

// mockEntityDocumentLinkRepo implements EntityDocumentLinkRepository for testing.
type mockEntityDocumentLinkRepo struct {
	linkFn          func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64, linkType string) error
	unlinkFn        func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64) error
	listForEntityFn func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error)
}

func (m *mockEntityDocumentLinkRepo) Link(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64, linkType string) error {
	if m.linkFn != nil {
		return m.linkFn(ctx, entityType, entityID, documentID, linkType)
	}
	return fmt.Errorf("Link not implemented")
}

func (m *mockEntityDocumentLinkRepo) Unlink(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64) error {
	if m.unlinkFn != nil {
		return m.unlinkFn(ctx, entityType, entityID, documentID)
	}
	return fmt.Errorf("Unlink not implemented")
}

func (m *mockEntityDocumentLinkRepo) ListForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error) {
	if m.listForEntityFn != nil {
		return m.listForEntityFn(ctx, entityType, entityID)
	}
	return nil, fmt.Errorf("ListForEntity not implemented")
}

// ============================================================================
// EntityDocumentService.LinkDocumentByKey Tests
// ============================================================================

func TestEntityDocumentService_LinkDocumentByKey_HappyPath(t *testing.T) {
	var capturedEntityType models.EntityType
	var capturedEntityID, capturedDocID int64
	var capturedLinkType string

	repo := &mockEntityDocumentRepo{
		createOrGetFn: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			return &models.Document{ID: 42, Title: title, FilePath: filePath}, nil
		},
	}

	linkRepo := &mockEntityDocumentLinkRepo{
		linkFn: func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64, linkType string) error {
			capturedEntityType = entityType
			capturedEntityID = entityID
			capturedDocID = documentID
			capturedLinkType = linkType
			return nil
		},
	}

	svc := NewEntityDocumentService(
		repo,
		linkRepo,
		func(ctx context.Context, key string) (int64, models.EntityType, error) {
			if key != "E07-F01-001" {
				t.Errorf("expected key 'E07-F01-001', got %q", key)
			}
			return 10, models.EntityTypeTask, nil
		},
	)

	doc, err := svc.LinkDocumentByKey(context.Background(), "E07-F01-001", "Design Doc", "docs/design.md")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if doc.ID != 42 {
		t.Errorf("expected doc ID 42, got %d", doc.ID)
	}
	if capturedEntityType != models.EntityTypeTask {
		t.Errorf("expected entity type %q, got %q", models.EntityTypeTask, capturedEntityType)
	}
	if capturedEntityID != 10 {
		t.Errorf("expected entity ID 10, got %d", capturedEntityID)
	}
	if capturedDocID != 42 {
		t.Errorf("expected doc ID 42, got %d", capturedDocID)
	}
	if capturedLinkType != "general" {
		t.Errorf("expected link type 'general', got %q", capturedLinkType)
	}
}

func TestEntityDocumentService_LinkDocumentByKey_EntityNotFound(t *testing.T) {
	repo := &mockEntityDocumentRepo{}
	linkRepo := &mockEntityDocumentLinkRepo{}

	svc := NewEntityDocumentService(
		repo, linkRepo,
		func(ctx context.Context, key string) (int64, models.EntityType, error) {
			return 0, "", fmt.Errorf("not found")
		},
	)

	_, err := svc.LinkDocumentByKey(context.Background(), "E99", "Doc", "path.md")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "entity not found: not found" {
		t.Errorf("unexpected error: %v", got)
	}
}

func TestEntityDocumentService_LinkDocumentByKey_CreateOrGetFails(t *testing.T) {
	var linkCalled bool

	repo := &mockEntityDocumentRepo{
		createOrGetFn: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			return nil, fmt.Errorf("db error")
		},
	}

	linkRepo := &mockEntityDocumentLinkRepo{
		linkFn: func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64, linkType string) error {
			linkCalled = true
			return nil
		},
	}

	svc := NewEntityDocumentService(
		repo, linkRepo,
		func(ctx context.Context, key string) (int64, models.EntityType, error) {
			return 10, models.EntityTypeTask, nil
		},
	)

	_, err := svc.LinkDocumentByKey(context.Background(), "E07-F01-001", "Doc", "path.md")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "failed to create or get document: db error" {
		t.Errorf("unexpected error: %v", got)
	}
	if linkCalled {
		t.Error("Link should not have been called when CreateOrGet fails")
	}
}

func TestEntityDocumentService_LinkDocumentByKey_LinkFails(t *testing.T) {
	repo := &mockEntityDocumentRepo{
		createOrGetFn: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			return &models.Document{ID: 42, Title: title, FilePath: filePath}, nil
		},
	}

	linkRepo := &mockEntityDocumentLinkRepo{
		linkFn: func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64, linkType string) error {
			return fmt.Errorf("FK violation")
		},
	}

	svc := NewEntityDocumentService(
		repo, linkRepo,
		func(ctx context.Context, key string) (int64, models.EntityType, error) {
			return 10, models.EntityTypeEpic, nil
		},
	)

	_, err := svc.LinkDocumentByKey(context.Background(), "E07", "Doc", "path.md")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "failed to link document to epic E07: FK violation" {
		t.Errorf("unexpected error: %v", got)
	}
}

// ============================================================================
// EntityDocumentService.UnlinkDocumentByKey Tests
// ============================================================================

func TestEntityDocumentService_UnlinkDocumentByKey_HappyPath(t *testing.T) {
	var capturedEntityType models.EntityType
	var capturedEntityID, capturedDocID int64

	repo := &mockEntityDocumentRepo{
		getByTitleFn: func(ctx context.Context, title string) (*models.Document, error) {
			return &models.Document{ID: 42, Title: title}, nil
		},
	}

	linkRepo := &mockEntityDocumentLinkRepo{
		unlinkFn: func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64) error {
			capturedEntityType = entityType
			capturedEntityID = entityID
			capturedDocID = documentID
			return nil
		},
	}

	svc := NewEntityDocumentService(
		repo, linkRepo,
		func(ctx context.Context, key string) (int64, models.EntityType, error) {
			return 5, models.EntityTypeFeature, nil
		},
	)

	err := svc.UnlinkDocumentByKey(context.Background(), "E07-F01", "My Doc")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedEntityType != models.EntityTypeFeature {
		t.Errorf("expected entity type %q, got %q", models.EntityTypeFeature, capturedEntityType)
	}
	if capturedEntityID != 5 {
		t.Errorf("expected entity ID 5, got %d", capturedEntityID)
	}
	if capturedDocID != 42 {
		t.Errorf("expected doc ID 42, got %d", capturedDocID)
	}
}

func TestEntityDocumentService_UnlinkDocumentByKey_DocumentNotFound_Idempotent(t *testing.T) {
	repo := &mockEntityDocumentRepo{
		getByTitleFn: func(ctx context.Context, title string) (*models.Document, error) {
			return nil, fmt.Errorf("document not found")
		},
	}

	linkRepo := &mockEntityDocumentLinkRepo{}

	svc := NewEntityDocumentService(
		repo, linkRepo,
		func(ctx context.Context, key string) (int64, models.EntityType, error) {
			return 5, models.EntityTypeTask, nil
		},
	)

	err := svc.UnlinkDocumentByKey(context.Background(), "E07-F01-001", "Missing Doc")

	// Idempotent: document-not-found is treated as success
	if err != nil {
		t.Fatalf("expected nil error (idempotent), got: %v", err)
	}
}

func TestEntityDocumentService_UnlinkDocumentByKey_EntityNotFound(t *testing.T) {
	repo := &mockEntityDocumentRepo{}
	linkRepo := &mockEntityDocumentLinkRepo{}

	svc := NewEntityDocumentService(
		repo, linkRepo,
		func(ctx context.Context, key string) (int64, models.EntityType, error) {
			return 0, "", fmt.Errorf("not found")
		},
	)

	err := svc.UnlinkDocumentByKey(context.Background(), "B999", "Doc")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "entity not found: not found" {
		t.Errorf("unexpected error: %v", got)
	}
}

func TestEntityDocumentService_UnlinkDocumentByKey_UnlinkFails(t *testing.T) {
	repo := &mockEntityDocumentRepo{
		getByTitleFn: func(ctx context.Context, title string) (*models.Document, error) {
			return &models.Document{ID: 42, Title: title}, nil
		},
	}

	linkRepo := &mockEntityDocumentLinkRepo{
		unlinkFn: func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64) error {
			return fmt.Errorf("db error")
		},
	}

	svc := NewEntityDocumentService(
		repo, linkRepo,
		func(ctx context.Context, key string) (int64, models.EntityType, error) {
			return 5, models.EntityTypeBug, nil
		},
	)

	err := svc.UnlinkDocumentByKey(context.Background(), "B001", "Doc")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "failed to unlink document from bug B001: db error" {
		t.Errorf("unexpected error: %v", got)
	}
}

// ============================================================================
// EntityDocumentService.ListDocumentsByKey Tests
// ============================================================================

func TestEntityDocumentService_ListDocumentsByKey_HappyPath(t *testing.T) {
	var capturedEntityType models.EntityType
	var capturedEntityID int64

	repo := &mockEntityDocumentRepo{}
	expectedDocs := []*models.Document{
		{ID: 1, Title: "Doc 1"},
		{ID: 2, Title: "Doc 2"},
	}

	linkRepo := &mockEntityDocumentLinkRepo{
		listForEntityFn: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error) {
			capturedEntityType = entityType
			capturedEntityID = entityID
			return expectedDocs, nil
		},
	}

	svc := NewEntityDocumentService(
		repo, linkRepo,
		func(ctx context.Context, key string) (int64, models.EntityType, error) {
			return 7, models.EntityTypeEpic, nil
		},
	)

	docs, err := svc.ListDocumentsByKey(context.Background(), "E07")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}
	if capturedEntityType != models.EntityTypeEpic {
		t.Errorf("expected entity type %q, got %q", models.EntityTypeEpic, capturedEntityType)
	}
	if capturedEntityID != 7 {
		t.Errorf("expected entity ID 7, got %d", capturedEntityID)
	}
}

func TestEntityDocumentService_ListDocumentsByKey_EntityNotFound(t *testing.T) {
	repo := &mockEntityDocumentRepo{}
	linkRepo := &mockEntityDocumentLinkRepo{}

	svc := NewEntityDocumentService(
		repo, linkRepo,
		func(ctx context.Context, key string) (int64, models.EntityType, error) {
			return 0, "", fmt.Errorf("not found")
		},
	)

	_, err := svc.ListDocumentsByKey(context.Background(), "CC-999")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "entity not found: not found" {
		t.Errorf("unexpected error: %v", got)
	}
}

func TestEntityDocumentService_ListDocumentsByKey_NilResult(t *testing.T) {
	repo := &mockEntityDocumentRepo{}

	linkRepo := &mockEntityDocumentLinkRepo{
		listForEntityFn: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error) {
			return nil, nil // returns nil slice
		},
	}

	svc := NewEntityDocumentService(
		repo, linkRepo,
		func(ctx context.Context, key string) (int64, models.EntityType, error) {
			return 1, models.EntityTypeFeature, nil
		},
	)

	docs, err := svc.ListDocumentsByKey(context.Background(), "E07-F01")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if docs == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 documents, got %d", len(docs))
	}
}

func TestEntityDocumentService_ListDocumentsByKey_ListError(t *testing.T) {
	repo := &mockEntityDocumentRepo{}

	linkRepo := &mockEntityDocumentLinkRepo{
		listForEntityFn: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error) {
			return nil, fmt.Errorf("db error")
		},
	}

	svc := NewEntityDocumentService(
		repo, linkRepo,
		func(ctx context.Context, key string) (int64, models.EntityType, error) {
			return 1, models.EntityTypeEpic, nil
		},
	)

	_, err := svc.ListDocumentsByKey(context.Background(), "E07")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "failed to list documents for epic E07: db error" {
		t.Errorf("unexpected error: %v", got)
	}
}

// ============================================================================
// EntityDocumentService All Entity Types Test (AC-7)
// ============================================================================

// ============================================================================
// EntityLookupFnFromRepo Tests
// ============================================================================

// mockEntityKeyLookup implements EntityKeyLookup for testing.
type mockEntityKeyLookup struct {
	getByKeyFunc func(ctx context.Context, key string) (models.Entity, error)
}

func (m *mockEntityKeyLookup) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("not implemented")
}

func TestEntityLookupFnFromRepo_Success(t *testing.T) {
	mock := &mockEntityKeyLookup{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			epic := &models.Epic{}
			epic.ID = 42
			epic.Key = key
			return epic, nil
		},
	}

	lookupFn := EntityLookupFnFromRepo(mock)
	id, entityType, err := lookupFn(context.Background(), "E07")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != 42 {
		t.Errorf("expected ID 42, got %d", id)
	}
	if entityType != models.EntityTypeEpic {
		t.Errorf("expected entity type %q, got %q", models.EntityTypeEpic, entityType)
	}
}

func TestEntityLookupFnFromRepo_RepoError(t *testing.T) {
	mock := &mockEntityKeyLookup{
		getByKeyFunc: func(_ context.Context, _ string) (models.Entity, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	lookupFn := EntityLookupFnFromRepo(mock)
	_, _, err := lookupFn(context.Background(), "E99")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "not found" {
		t.Errorf("expected 'not found' error, got %q", err.Error())
	}
}

func TestEntityLookupFnFromRepo_NilEntity(t *testing.T) {
	mock := &mockEntityKeyLookup{
		getByKeyFunc: func(_ context.Context, _ string) (models.Entity, error) {
			return nil, nil
		},
	}

	lookupFn := EntityLookupFnFromRepo(mock)
	_, _, err := lookupFn(context.Background(), "E99")

	if err == nil {
		t.Fatal("expected error for nil entity, got nil")
	}
	if err.Error() != "entity not found: E99" {
		t.Errorf("expected 'entity not found: E99', got %q", err.Error())
	}
}

func TestEntityLookupFnFromRepo_DifferentEntityTypes(t *testing.T) {
	tests := []struct {
		name         string
		entity       models.Entity
		expectedType models.EntityType
		expectedID   int64
	}{
		{
			name: "epic",
			entity: func() models.Entity {
				e := &models.Epic{}
				e.ID = 1
				return e
			}(),
			expectedType: models.EntityTypeEpic,
			expectedID:   1,
		},
		{
			name: "feature",
			entity: func() models.Entity {
				f := &models.Feature{}
				f.ID = 2
				return f
			}(),
			expectedType: models.EntityTypeFeature,
			expectedID:   2,
		},
		{
			name: "task",
			entity: func() models.Entity {
				tk := &models.Task{}
				tk.ID = 3
				return tk
			}(),
			expectedType: models.EntityTypeTask,
			expectedID:   3,
		},
		{
			name: "bug",
			entity: func() models.Entity {
				b := &models.Bug{}
				b.ID = 4
				return b
			}(),
			expectedType: models.EntityTypeBug,
			expectedID:   4,
		},
		{
			name: "change card",
			entity: func() models.Entity {
				c := &models.ChangeCard{}
				c.ID = 5
				return c
			}(),
			expectedType: models.EntityTypeChange,
			expectedID:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockEntityKeyLookup{
				getByKeyFunc: func(_ context.Context, _ string) (models.Entity, error) {
					return tt.entity, nil
				},
			}

			lookupFn := EntityLookupFnFromRepo(mock)
			id, entityType, err := lookupFn(context.Background(), "test-key")

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if id != tt.expectedID {
				t.Errorf("expected ID %d, got %d", tt.expectedID, id)
			}
			if entityType != tt.expectedType {
				t.Errorf("expected entity type %q, got %q", tt.expectedType, entityType)
			}
		})
	}
}

func TestEntityDocumentService_LinkDocumentByKey_AllEntityTypes(t *testing.T) {
	tests := []struct {
		name       string
		entityKey  string
		entityType models.EntityType
		entityID   int64
	}{
		{"epic", "E07", models.EntityTypeEpic, 1},
		{"feature", "E07-F01", models.EntityTypeFeature, 2},
		{"task", "E07-F01-001", models.EntityTypeTask, 3},
		{"bug", "B001", models.EntityTypeBug, 4},
		{"change", "CC-001", models.EntityTypeChange, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedEntityType models.EntityType
			var capturedEntityID int64

			repo := &mockEntityDocumentRepo{
				createOrGetFn: func(ctx context.Context, title, filePath string) (*models.Document, error) {
					return &models.Document{ID: 99, Title: title, FilePath: filePath}, nil
				},
			}

			linkRepo := &mockEntityDocumentLinkRepo{
				linkFn: func(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64, linkType string) error {
					capturedEntityType = entityType
					capturedEntityID = entityID
					return nil
				},
			}

			svc := NewEntityDocumentService(
				repo, linkRepo,
				func(ctx context.Context, key string) (int64, models.EntityType, error) {
					return tt.entityID, tt.entityType, nil
				},
			)

			doc, err := svc.LinkDocumentByKey(context.Background(), tt.entityKey, "Test Doc", "test.md")

			if err != nil {
				t.Fatalf("expected no error for %s, got: %v", tt.name, err)
			}
			if doc == nil {
				t.Fatalf("expected document for %s, got nil", tt.name)
			}
			if capturedEntityType != tt.entityType {
				t.Errorf("expected entity type %q, got %q", tt.entityType, capturedEntityType)
			}
			if capturedEntityID != tt.entityID {
				t.Errorf("expected entity ID %d, got %d", tt.entityID, capturedEntityID)
			}
		})
	}
}
