package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestCreateOrGetNewDocument creates a new document
func TestCreateOrGetNewDocument(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	doc, err := docRepo.CreateOrGet(ctx, "OAuth Spec", "docs/oauth.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	if doc.ID == 0 {
		t.Error("Expected document ID to be set")
	}
	if doc.Title != "OAuth Spec" {
		t.Errorf("Expected title 'OAuth Spec', got %q", doc.Title)
	}
	if doc.FilePath != "docs/oauth.md" {
		t.Errorf("Expected file path 'docs/oauth.md', got %q", doc.FilePath)
	}
}

// TestCreateOrGetExistingDocument reuses existing document
func TestCreateOrGetExistingDocument(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	// Create first
	doc1, err := docRepo.CreateOrGet(ctx, "Architecture", "docs/arch.md")
	if err != nil {
		t.Fatalf("First CreateOrGet failed: %v", err)
	}

	// Get existing (same title and path)
	doc2, err := docRepo.CreateOrGet(ctx, "Architecture", "docs/arch.md")
	if err != nil {
		t.Fatalf("Second CreateOrGet failed: %v", err)
	}

	if doc1.ID != doc2.ID {
		t.Errorf("Expected same document ID, got %d and %d", doc1.ID, doc2.ID)
	}
}

// TestGetByID retrieves document by ID
func TestGetByID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	created, err := docRepo.CreateOrGet(ctx, "API Design", "docs/api.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	retrieved, err := docRepo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, retrieved.ID)
	}
	if retrieved.Title != "API Design" {
		t.Errorf("Expected title 'API Design', got %q", retrieved.Title)
	}
}

// TestDeleteDocument removes document
func TestDeleteDocument(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	doc, err := docRepo.CreateOrGet(ctx, "ToDelete", "docs/delete.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	err = docRepo.Delete(ctx, doc.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = docRepo.GetByID(ctx, doc.ID)
	if err == nil {
		t.Error("Expected error for deleted document")
	}
}

// TestDocumentReuseSameTitlePath reuses document with same title and path
func TestDocumentReuseSameTitlePath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)
	epicRepo := NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E93'")

	// Create dedicated epic for this test
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E93",
		Title: "Test Epic for Document Reuse"}, Status: models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, testEpic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	// Link same document to different parents (via CreateOrGet)
	entityDocRepo := NewEntityDocumentRepository(db)
	doc1, err := docRepo.CreateOrGet(ctx, "SharedDoc", "docs/shared.md")
	if err != nil {
		t.Fatalf("Failed to create doc1: %v", err)
	}
	_ = entityDocRepo.Link(ctx, models.EntityTypeEpic, testEpic.ID, doc1.ID, "")

	doc2, err := docRepo.CreateOrGet(ctx, "SharedDoc", "docs/shared.md")
	if err != nil {
		t.Fatalf("Failed to create doc2: %v", err)
	}
	if doc1.ID != doc2.ID {
		t.Error("Expected document reuse for same title and path")
	}
}

// TestGetByIDNotFound returns error for missing document
func TestGetByIDNotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	_, err := docRepo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("Expected error for non-existent document")
	}
}

// TestGetByTitle retrieves document by title only
func TestGetByTitle(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	created, err := docRepo.CreateOrGet(ctx, "Title Only Doc", "docs/titleonly.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	retrieved, err := docRepo.GetByTitle(ctx, "Title Only Doc")
	if err != nil {
		t.Fatalf("GetByTitle failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, retrieved.ID)
	}
	if retrieved.Title != "Title Only Doc" {
		t.Errorf("Expected title 'Title Only Doc', got %q", retrieved.Title)
	}
	if retrieved.FilePath != "docs/titleonly.md" {
		t.Errorf("Expected file path 'docs/titleonly.md', got %q", retrieved.FilePath)
	}
}

// TestGetByTitleNotFound returns error for missing document title
func TestGetByTitleNotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	_, err := docRepo.GetByTitle(ctx, "NonexistentTitle")
	if err == nil {
		t.Error("Expected error for non-existent document title")
	}
}
