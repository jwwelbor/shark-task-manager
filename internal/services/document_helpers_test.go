package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// mockDocumentRepo implements DocumentRepo for testing.
type mockDocumentRepo struct {
	CreateOrGetFunc func(ctx context.Context, title, filePath string) (*models.Document, error)
	GetByTitleFunc  func(ctx context.Context, title string) (*models.Document, error)
}

func (m *mockDocumentRepo) CreateOrGet(ctx context.Context, title, filePath string) (*models.Document, error) {
	if m.CreateOrGetFunc != nil {
		return m.CreateOrGetFunc(ctx, title, filePath)
	}
	return nil, fmt.Errorf("CreateOrGet not implemented")
}

func (m *mockDocumentRepo) GetByTitle(ctx context.Context, title string) (*models.Document, error) {
	if m.GetByTitleFunc != nil {
		return m.GetByTitleFunc(ctx, title)
	}
	return nil, fmt.Errorf("GetByTitle not implemented")
}

func TestLinkDocumentToEntity_Success(t *testing.T) {
	doc := &models.Document{ID: 42, Title: "Design Doc", FilePath: "/docs/design.md"}
	docRepo := &mockDocumentRepo{
		CreateOrGetFunc: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			if title != "Design Doc" || filePath != "/docs/design.md" {
				t.Errorf("unexpected args: title=%q, filePath=%q", title, filePath)
			}
			return doc, nil
		},
	}

	var linkedEntityID, linkedDocID int64
	linkFn := func(ctx context.Context, entityID, documentID int64) error {
		linkedEntityID = entityID
		linkedDocID = documentID
		return nil
	}

	result, err := linkDocumentToEntity(context.Background(), docRepo, linkFn,
		99, "Design Doc", "/docs/design.md", "feature", "E07-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != doc {
		t.Errorf("expected returned doc to be %v, got %v", doc, result)
	}
	if linkedEntityID != 99 {
		t.Errorf("expected entityID 99, got %d", linkedEntityID)
	}
	if linkedDocID != 42 {
		t.Errorf("expected documentID 42, got %d", linkedDocID)
	}
}

func TestLinkDocumentToEntity_CreateOrGetFails(t *testing.T) {
	docRepo := &mockDocumentRepo{
		CreateOrGetFunc: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			return nil, fmt.Errorf("db error")
		},
	}

	linkFn := func(ctx context.Context, entityID, documentID int64) error {
		t.Fatal("linkFn should not be called when CreateOrGet fails")
		return nil
	}

	result, err := linkDocumentToEntity(context.Background(), docRepo, linkFn,
		1, "Doc", "/path", "epic", "E01")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Error("expected nil result on error")
	}
	expected := "failed to create or get document: db error"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestLinkDocumentToEntity_LinkFails(t *testing.T) {
	doc := &models.Document{ID: 10, Title: "Doc"}
	docRepo := &mockDocumentRepo{
		CreateOrGetFunc: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			return doc, nil
		},
	}

	linkFn := func(ctx context.Context, entityID, documentID int64) error {
		return fmt.Errorf("link constraint error")
	}

	result, err := linkDocumentToEntity(context.Background(), docRepo, linkFn,
		5, "Doc", "/path", "task", "E01-F01-001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Error("expected nil result on error")
	}
	expected := "failed to link document to task E01-F01-001: link constraint error"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestUnlinkDocumentFromEntity_Success(t *testing.T) {
	doc := &models.Document{ID: 42, Title: "Design Doc"}
	docRepo := &mockDocumentRepo{
		GetByTitleFunc: func(ctx context.Context, title string) (*models.Document, error) {
			if title != "Design Doc" {
				t.Errorf("unexpected title: %q", title)
			}
			return doc, nil
		},
	}

	var unlinkedEntityID, unlinkedDocID int64
	unlinkFn := func(ctx context.Context, entityID, documentID int64) error {
		unlinkedEntityID = entityID
		unlinkedDocID = documentID
		return nil
	}

	err := unlinkDocumentFromEntity(context.Background(), docRepo, unlinkFn,
		99, "Design Doc", "feature", "E07-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unlinkedEntityID != 99 {
		t.Errorf("expected entityID 99, got %d", unlinkedEntityID)
	}
	if unlinkedDocID != 42 {
		t.Errorf("expected documentID 42, got %d", unlinkedDocID)
	}
}

func TestUnlinkDocumentFromEntity_GetByTitleFails(t *testing.T) {
	docRepo := &mockDocumentRepo{
		GetByTitleFunc: func(ctx context.Context, title string) (*models.Document, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	unlinkFn := func(ctx context.Context, entityID, documentID int64) error {
		t.Fatal("unlinkFn should not be called when GetByTitle fails")
		return nil
	}

	err := unlinkDocumentFromEntity(context.Background(), docRepo, unlinkFn,
		1, "Missing Doc", "epic", "E01")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expected := "document not found: not found"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestUnlinkDocumentFromEntity_UnlinkFails(t *testing.T) {
	doc := &models.Document{ID: 10, Title: "Doc"}
	docRepo := &mockDocumentRepo{
		GetByTitleFunc: func(ctx context.Context, title string) (*models.Document, error) {
			return doc, nil
		},
	}

	unlinkFn := func(ctx context.Context, entityID, documentID int64) error {
		return fmt.Errorf("unlink constraint error")
	}

	err := unlinkDocumentFromEntity(context.Background(), docRepo, unlinkFn,
		5, "Doc", "task", "E01-F01-001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expected := "failed to unlink document from task E01-F01-001: unlink constraint error"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}
