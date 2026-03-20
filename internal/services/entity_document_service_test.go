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

// ============================================================================
// EntityDocumentService.LinkDocumentByKey Tests
// ============================================================================

func TestEntityDocumentService_LinkDocumentByKey_HappyPath(t *testing.T) {
	var capturedEntityID, capturedDocID int64

	repo := &mockEntityDocumentRepo{
		createOrGetFn: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			return &models.Document{ID: 42, Title: title, FilePath: filePath}, nil
		},
	}

	svc := NewEntityDocumentService(
		repo,
		models.EntityTypeTask,
		func(ctx context.Context, entityID, documentID int64) error {
			capturedEntityID = entityID
			capturedDocID = documentID
			return nil
		},
		nil, nil,
		func(ctx context.Context, key string) (int64, error) {
			if key != "E07-F01-001" {
				t.Errorf("expected key 'E07-F01-001', got %q", key)
			}
			return 10, nil
		},
	)

	doc, err := svc.LinkDocumentByKey(context.Background(), "E07-F01-001", "Design Doc", "docs/design.md")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if doc.ID != 42 {
		t.Errorf("expected doc ID 42, got %d", doc.ID)
	}
	if capturedEntityID != 10 {
		t.Errorf("expected entity ID 10, got %d", capturedEntityID)
	}
	if capturedDocID != 42 {
		t.Errorf("expected doc ID 42, got %d", capturedDocID)
	}
}

func TestEntityDocumentService_LinkDocumentByKey_EntityNotFound(t *testing.T) {
	repo := &mockEntityDocumentRepo{}

	svc := NewEntityDocumentService(
		repo, models.EntityTypeEpic, nil, nil, nil,
		func(ctx context.Context, key string) (int64, error) {
			return 0, fmt.Errorf("not found")
		},
	)

	_, err := svc.LinkDocumentByKey(context.Background(), "E99", "Doc", "path.md")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "epic not found: not found" {
		t.Errorf("unexpected error: %v", got)
	}
}

// ============================================================================
// EntityDocumentService.UnlinkDocumentByKey Tests
// ============================================================================

func TestEntityDocumentService_UnlinkDocumentByKey_HappyPath(t *testing.T) {
	var unlinkCalled bool

	repo := &mockEntityDocumentRepo{
		getByTitleFn: func(ctx context.Context, title string) (*models.Document, error) {
			return &models.Document{ID: 42, Title: title}, nil
		},
	}

	svc := NewEntityDocumentService(
		repo, models.EntityTypeFeature,
		nil,
		func(ctx context.Context, entityID, documentID int64) error {
			unlinkCalled = true
			if entityID != 5 {
				t.Errorf("expected entity ID 5, got %d", entityID)
			}
			if documentID != 42 {
				t.Errorf("expected doc ID 42, got %d", documentID)
			}
			return nil
		},
		nil,
		func(ctx context.Context, key string) (int64, error) {
			return 5, nil
		},
	)

	err := svc.UnlinkDocumentByKey(context.Background(), "E07-F01", "My Doc")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !unlinkCalled {
		t.Error("expected unlink function to be called")
	}
}

func TestEntityDocumentService_UnlinkDocumentByKey_DocumentNotFound_Idempotent(t *testing.T) {
	repo := &mockEntityDocumentRepo{
		getByTitleFn: func(ctx context.Context, title string) (*models.Document, error) {
			return nil, fmt.Errorf("document not found")
		},
	}

	svc := NewEntityDocumentService(
		repo, "task",
		nil, nil, nil,
		func(ctx context.Context, key string) (int64, error) {
			return 5, nil
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

	svc := NewEntityDocumentService(
		repo, models.EntityTypeBug,
		nil, nil, nil,
		func(ctx context.Context, key string) (int64, error) {
			return 0, fmt.Errorf("not found")
		},
	)

	err := svc.UnlinkDocumentByKey(context.Background(), "B999", "Doc")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "bug not found: not found" {
		t.Errorf("unexpected error: %v", got)
	}
}

// ============================================================================
// EntityDocumentService.ListDocumentsByKey Tests
// ============================================================================

func TestEntityDocumentService_ListDocumentsByKey_HappyPath(t *testing.T) {
	repo := &mockEntityDocumentRepo{}
	expectedDocs := []*models.Document{
		{ID: 1, Title: "Doc 1"},
		{ID: 2, Title: "Doc 2"},
	}

	svc := NewEntityDocumentService(
		repo, models.EntityTypeEpic,
		nil, nil,
		func(ctx context.Context, entityID int64) ([]*models.Document, error) {
			if entityID != 7 {
				t.Errorf("expected entity ID 7, got %d", entityID)
			}
			return expectedDocs, nil
		},
		func(ctx context.Context, key string) (int64, error) {
			return 7, nil
		},
	)

	docs, err := svc.ListDocumentsByKey(context.Background(), "E07")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}
}

func TestEntityDocumentService_ListDocumentsByKey_NilListFn(t *testing.T) {
	repo := &mockEntityDocumentRepo{}

	svc := NewEntityDocumentService(
		repo, "task",
		nil, nil,
		nil, // no list function
		func(ctx context.Context, key string) (int64, error) {
			return 1, nil
		},
	)

	docs, err := svc.ListDocumentsByKey(context.Background(), "E07-F01-001")

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

func TestEntityDocumentService_ListDocumentsByKey_EntityNotFound(t *testing.T) {
	repo := &mockEntityDocumentRepo{}

	svc := NewEntityDocumentService(
		repo, models.EntityTypeChange,
		nil, nil,
		func(ctx context.Context, entityID int64) ([]*models.Document, error) {
			return nil, nil
		},
		func(ctx context.Context, key string) (int64, error) {
			return 0, fmt.Errorf("not found")
		},
	)

	_, err := svc.ListDocumentsByKey(context.Background(), "CC-999")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "change not found: not found" {
		t.Errorf("unexpected error: %v", got)
	}
}

func TestEntityDocumentService_ListDocumentsByKey_NilResult(t *testing.T) {
	repo := &mockEntityDocumentRepo{}

	svc := NewEntityDocumentService(
		repo, models.EntityTypeFeature,
		nil, nil,
		func(ctx context.Context, entityID int64) ([]*models.Document, error) {
			return nil, nil // returns nil slice
		},
		func(ctx context.Context, key string) (int64, error) {
			return 1, nil
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
