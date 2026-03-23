package entitydoc

import (
	"context"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"

	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/repository/document"
)

// helper: create a test document in the documents table and return its ID
func createTestDocument(t *testing.T, ctx context.Context, db *dbconn.DB, title, filePath string) int64 {
	t.Helper()
	docRepo := document.NewDocumentRepository(db)
	doc, err := docRepo.CreateOrGet(ctx, title, filePath)
	if err != nil {
		t.Fatalf("failed to create test document: %v", err)
	}
	return doc.ID
}

// helper: cleanup entity_documents rows for a given entity_id
func cleanupEntityDocuments(t *testing.T, ctx context.Context, db *dbconn.DB, entityID int64) {
	t.Helper()
	_, err := db.ExecContext(ctx, "DELETE FROM entity_documents WHERE entity_id = ?", entityID)
	if err != nil {
		t.Fatalf("failed to cleanup entity_documents: %v", err)
	}
}

func TestEntityDocumentRepo_Link(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityDocumentRepository(db)

	const testEntityID int64 = 99999

	// Clean up before test
	cleanupEntityDocuments(t, ctx, db, testEntityID)

	// Create test documents
	docIDs := make([]int64, 5)
	entityTypes := []models.EntityType{
		models.EntityTypeEpic,
		models.EntityTypeFeature,
		models.EntityTypeTask,
		models.EntityTypeBug,
		models.EntityTypeChange,
	}

	for i, et := range entityTypes {
		docIDs[i] = createTestDocument(t, ctx, db, "TestLink-"+string(et), "docs/test-link-"+string(et)+".md")

		err := repo.Link(ctx, et, testEntityID, docIDs[i], "")
		if err != nil {
			t.Fatalf("Link() for entity type %s returned error: %v", et, err)
		}

		// Verify row was created
		var count int
		err = database.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM entity_documents WHERE entity_type = ? AND entity_id = ? AND document_id = ?",
			et, testEntityID, docIDs[i]).Scan(&count)
		if err != nil {
			t.Fatalf("failed to verify link for %s: %v", et, err)
		}
		if count != 1 {
			t.Errorf("expected 1 row for entity type %s, got %d", et, count)
		}
	}

	// Cleanup
	cleanupEntityDocuments(t, ctx, db, testEntityID)
}

func TestEntityDocumentRepo_LinkIdempotent(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityDocumentRepository(db)

	const testEntityID int64 = 99998

	// Clean up before test
	cleanupEntityDocuments(t, ctx, db, testEntityID)

	docID := createTestDocument(t, ctx, db, "TestIdempotent", "docs/test-idempotent.md")

	// Link twice
	err := repo.Link(ctx, models.EntityTypeTask, testEntityID, docID, "general")
	if err != nil {
		t.Fatalf("first Link() returned error: %v", err)
	}

	err = repo.Link(ctx, models.EntityTypeTask, testEntityID, docID, "general")
	if err != nil {
		t.Fatalf("second Link() returned error (should be idempotent): %v", err)
	}

	// Verify exactly 1 row
	var count int
	err = database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM entity_documents WHERE entity_type = ? AND entity_id = ? AND document_id = ?",
		models.EntityTypeTask, testEntityID, docID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to verify count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected exactly 1 row after duplicate Link, got %d", count)
	}

	// Cleanup
	cleanupEntityDocuments(t, ctx, db, testEntityID)
}

func TestEntityDocumentRepo_LinkWithType(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityDocumentRepository(db)

	const testEntityID int64 = 99997

	// Clean up before test
	cleanupEntityDocuments(t, ctx, db, testEntityID)

	// AC-4: Empty link_type defaults to "general"
	docID1 := createTestDocument(t, ctx, db, "TestLinkType-Empty", "docs/test-linktype-empty.md")
	err := repo.Link(ctx, models.EntityTypeTask, testEntityID, docID1, "")
	if err != nil {
		t.Fatalf("Link() with empty link_type returned error: %v", err)
	}

	var linkType string
	err = database.QueryRowContext(ctx,
		"SELECT link_type FROM entity_documents WHERE entity_type = ? AND entity_id = ? AND document_id = ?",
		models.EntityTypeTask, testEntityID, docID1).Scan(&linkType)
	if err != nil {
		t.Fatalf("failed to query link_type: %v", err)
	}
	if linkType != "general" {
		t.Errorf("expected link_type 'general' for empty input, got '%s'", linkType)
	}

	// AC-5: Custom link_type preserved
	docID2 := createTestDocument(t, ctx, db, "TestLinkType-Spec", "docs/test-linktype-spec.md")
	err = repo.Link(ctx, models.EntityTypeTask, testEntityID, docID2, "specification")
	if err != nil {
		t.Fatalf("Link() with custom link_type returned error: %v", err)
	}

	err = database.QueryRowContext(ctx,
		"SELECT link_type FROM entity_documents WHERE entity_type = ? AND entity_id = ? AND document_id = ?",
		models.EntityTypeTask, testEntityID, docID2).Scan(&linkType)
	if err != nil {
		t.Fatalf("failed to query link_type: %v", err)
	}
	if linkType != "specification" {
		t.Errorf("expected link_type 'specification', got '%s'", linkType)
	}

	// Cleanup
	cleanupEntityDocuments(t, ctx, db, testEntityID)
}

func TestEntityDocumentRepo_Unlink(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityDocumentRepository(db)

	const testEntityID int64 = 99996

	// Clean up before test
	cleanupEntityDocuments(t, ctx, db, testEntityID)

	// AC-6: Unlink existing link
	docID := createTestDocument(t, ctx, db, "TestUnlink", "docs/test-unlink.md")
	err := repo.Link(ctx, models.EntityTypeTask, testEntityID, docID, "")
	if err != nil {
		t.Fatalf("Link() returned error: %v", err)
	}

	err = repo.Unlink(ctx, models.EntityTypeTask, testEntityID, docID)
	if err != nil {
		t.Fatalf("Unlink() returned error: %v", err)
	}

	// Verify row removed
	var count int
	err = database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM entity_documents WHERE entity_type = ? AND entity_id = ? AND document_id = ?",
		models.EntityTypeTask, testEntityID, docID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to verify unlink: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows after Unlink, got %d", count)
	}

	// Verify document itself still exists (AC-6: documents row NOT deleted)
	var docCount int
	err = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM documents WHERE id = ?", docID).Scan(&docCount)
	if err != nil {
		t.Fatalf("failed to verify document: %v", err)
	}
	if docCount != 1 {
		t.Errorf("expected document to still exist after Unlink, got count %d", docCount)
	}

	// AC-7: Unlink non-existent link is no-op
	err = repo.Unlink(ctx, models.EntityTypeTask, testEntityID, docID)
	if err != nil {
		t.Fatalf("Unlink() for non-existent link returned error (should be no-op): %v", err)
	}

	// Cleanup
	cleanupEntityDocuments(t, ctx, db, testEntityID)
}

func TestEntityDocumentRepo_ListForEntity(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityDocumentRepository(db)

	const testEntityID int64 = 99995
	const otherEntityID int64 = 99994

	// Clean up before test
	cleanupEntityDocuments(t, ctx, db, testEntityID)
	cleanupEntityDocuments(t, ctx, db, otherEntityID)

	// AC-8: Create 3 documents linked to task
	docIDs := make([]int64, 3)
	for i := 0; i < 3; i++ {
		docIDs[i] = createTestDocument(t, ctx, db,
			"TestList-"+string(rune('A'+i)),
			"docs/test-list-"+string(rune('a'+i))+".md")
		err := repo.Link(ctx, models.EntityTypeTask, testEntityID, docIDs[i], "")
		if err != nil {
			t.Fatalf("Link() doc %d returned error: %v", i, err)
		}
	}

	// Link one doc to a different entity type to verify isolation
	err := repo.Link(ctx, models.EntityTypeEpic, otherEntityID, docIDs[0], "")
	if err != nil {
		t.Fatalf("Link() to different entity returned error: %v", err)
	}

	docs, err := repo.ListForEntity(ctx, models.EntityTypeTask, testEntityID)
	if err != nil {
		t.Fatalf("ListForEntity() returned error: %v", err)
	}

	if len(docs) != 3 {
		t.Fatalf("expected 3 documents, got %d", len(docs))
	}

	// Verify each document has populated fields
	for i, doc := range docs {
		if doc.ID == 0 {
			t.Errorf("doc[%d].ID should be > 0", i)
		}
		if doc.Title == "" {
			t.Errorf("doc[%d].Title should not be empty", i)
		}
		if doc.FilePath == "" {
			t.Errorf("doc[%d].FilePath should not be empty", i)
		}
		if doc.CreatedAt.IsZero() {
			t.Errorf("doc[%d].CreatedAt should not be zero", i)
		}
	}

	// Verify ordering: most recently linked first (DESC)
	// Documents linked in rapid succession may share the same timestamp,
	// so we only verify non-ascending order (>=).
	for i := 0; i < len(docs)-1; i++ {
		if docs[i].CreatedAt.Before(docs[i+1].CreatedAt) {
			t.Errorf("documents not in DESC order: docs[%d].CreatedAt (%v) < docs[%d].CreatedAt (%v)",
				i, docs[i].CreatedAt, i+1, docs[i+1].CreatedAt)
		}
	}

	// AC-9: ListForEntity for entity with no links returns empty non-nil slice
	emptyDocs, err := repo.ListForEntity(ctx, models.EntityTypeEpic, 88888)
	if err != nil {
		t.Fatalf("ListForEntity() for empty entity returned error: %v", err)
	}
	if emptyDocs == nil {
		t.Error("expected non-nil slice for empty result, got nil")
	}
	if len(emptyDocs) != 0 {
		t.Errorf("expected 0 documents, got %d", len(emptyDocs))
	}

	// Verify cross-type isolation: epic query should only return the 1 doc linked to epic
	epicDocs, err := repo.ListForEntity(ctx, models.EntityTypeEpic, otherEntityID)
	if err != nil {
		t.Fatalf("ListForEntity() for epic returned error: %v", err)
	}
	if len(epicDocs) != 1 {
		t.Errorf("expected 1 document for epic, got %d (cross-type leak)", len(epicDocs))
	}

	// Cleanup
	cleanupEntityDocuments(t, ctx, db, testEntityID)
	cleanupEntityDocuments(t, ctx, db, otherEntityID)
}

func TestEntityDocumentRepo_InvalidEntityType(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityDocumentRepository(db)

	invalidTypes := []models.EntityType{"invalid", "", "project", "milestone"}

	for _, et := range invalidTypes {
		t.Run("Link_"+string(et), func(t *testing.T) {
			err := repo.Link(ctx, et, 1, 1, "")
			if err == nil {
				t.Error("Link() with invalid entity type should return error")
			}
			if err != nil && !strings.Contains(err.Error(), "invalid entity type") {
				t.Errorf("error should contain 'invalid entity type', got: %v", err)
			}
		})

		t.Run("Unlink_"+string(et), func(t *testing.T) {
			err := repo.Unlink(ctx, et, 1, 1)
			if err == nil {
				t.Error("Unlink() with invalid entity type should return error")
			}
			if err != nil && !strings.Contains(err.Error(), "invalid entity type") {
				t.Errorf("error should contain 'invalid entity type', got: %v", err)
			}
		})

		t.Run("ListForEntity_"+string(et), func(t *testing.T) {
			docs, err := repo.ListForEntity(ctx, et, 1)
			if err == nil {
				t.Error("ListForEntity() with invalid entity type should return error")
			}
			if err != nil && !strings.Contains(err.Error(), "invalid entity type") {
				t.Errorf("error should contain 'invalid entity type', got: %v", err)
			}
			if docs != nil {
				t.Error("expected nil docs on error")
			}
		})
	}
}

func TestEntityDocumentRepo_CascadeDelete(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityDocumentRepository(db)

	const testEntityID int64 = 99993

	// Clean up before test
	cleanupEntityDocuments(t, ctx, db, testEntityID)

	// Create a test document
	docID := createTestDocument(t, ctx, db, "TestCascade", "docs/test-cascade.md")

	// Link it
	err := repo.Link(ctx, models.EntityTypeTask, testEntityID, docID, "")
	if err != nil {
		t.Fatalf("Link() returned error: %v", err)
	}

	// Verify link exists
	var count int
	err = database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM entity_documents WHERE document_id = ?", docID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to verify link: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 entity_documents row, got %d", count)
	}

	// Delete the document (should cascade delete entity_documents row)
	_, err = database.ExecContext(ctx, "DELETE FROM documents WHERE id = ?", docID)
	if err != nil {
		t.Fatalf("failed to delete document: %v", err)
	}

	// Verify cascade: entity_documents row should be gone
	err = database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM entity_documents WHERE document_id = ?", docID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to verify cascade: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 entity_documents rows after document deletion (cascade), got %d", count)
	}

	// Verify ListForEntity reflects the deletion
	docs, err := repo.ListForEntity(ctx, models.EntityTypeTask, testEntityID)
	if err != nil {
		t.Fatalf("ListForEntity() returned error: %v", err)
	}
	for _, doc := range docs {
		if doc.ID == docID {
			t.Error("ListForEntity() should not return cascade-deleted document")
		}
	}

	// Cleanup
	cleanupEntityDocuments(t, ctx, db, testEntityID)
}

func TestEntityDocumentRepo_InvalidEntityID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityDocumentRepository(db)

	// EC-2: entity_id = 0
	err := repo.Link(ctx, models.EntityTypeTask, 0, 1, "")
	if err == nil {
		t.Error("Link() with entity_id=0 should return error")
	}
	if err != nil && !strings.Contains(err.Error(), "entity_id must be positive") {
		t.Errorf("error should contain 'entity_id must be positive', got: %v", err)
	}

	// EC-2: entity_id negative
	err = repo.Link(ctx, models.EntityTypeTask, -1, 1, "")
	if err == nil {
		t.Error("Link() with negative entity_id should return error")
	}

	// Unlink with entity_id=0
	err = repo.Unlink(ctx, models.EntityTypeTask, 0, 1)
	if err == nil {
		t.Error("Unlink() with entity_id=0 should return error")
	}

	// ListForEntity with entity_id=0
	_, err = repo.ListForEntity(ctx, models.EntityTypeTask, 0)
	if err == nil {
		t.Error("ListForEntity() with entity_id=0 should return error")
	}
}
