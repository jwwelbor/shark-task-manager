package entitydoc

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// EntityDocumentRepository handles polymorphic document linking operations
// for any entity type via the entity_documents table.
type EntityDocumentRepository struct {
	db *dbconn.DB
}

// NewEntityDocumentRepository creates a new EntityDocumentRepository
func NewEntityDocumentRepository(db *dbconn.DB) *EntityDocumentRepository {
	return &EntityDocumentRepository{db: db}
}

// validateEntityType checks that the entity type is valid
func validateEntityType(entityType models.EntityType) error {
	if !models.ValidEntityTypes[entityType] {
		return fmt.Errorf("invalid entity type: %q", entityType)
	}
	return nil
}

// validateEntityID checks that the entity ID is positive
func validateEntityID(entityID int64) error {
	if entityID <= 0 {
		return fmt.Errorf("entity_id must be positive, got %d", entityID)
	}
	return nil
}

// Link creates an association between an entity and a document.
// If linkType is empty, the SQL column default ("general") is used via COALESCE(NULLIF).
// The method is idempotent: duplicate links are silently ignored via INSERT OR IGNORE.
func (r *EntityDocumentRepository) Link(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64, linkType string) error {
	if err := validateEntityType(entityType); err != nil {
		return err
	}
	if err := validateEntityID(entityID); err != nil {
		return err
	}

	query := `
		INSERT OR IGNORE INTO entity_documents (entity_type, entity_id, document_id, link_type)
		VALUES (?, ?, ?, COALESCE(NULLIF(?, ''), 'general'))
	`

	_, err := r.db.ExecContext(ctx, query, entityType, entityID, documentID, linkType)
	if err != nil {
		return fmt.Errorf("failed to link document: %w", err)
	}

	return nil
}

// Unlink removes the association between an entity and a document.
// If no link exists, the method is a no-op (returns nil error).
// The document itself is NOT deleted.
func (r *EntityDocumentRepository) Unlink(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64) error {
	if err := validateEntityType(entityType); err != nil {
		return err
	}
	if err := validateEntityID(entityID); err != nil {
		return err
	}

	query := `
		DELETE FROM entity_documents
		WHERE entity_type = ? AND entity_id = ? AND document_id = ?
	`

	_, err := r.db.ExecContext(ctx, query, entityType, entityID, documentID)
	if err != nil {
		return fmt.Errorf("failed to unlink document: %w", err)
	}

	return nil
}

// BulkDoc holds a document's key fields alongside the entity it is linked to,
// used for bulk loading all entity-document associations in one query.
type BulkDoc struct {
	EntityType string
	EntityID   int64
	Title      string
	FilePath   string
}

// ListAll returns every entity-document link in one query, suitable for bulk
// assembly of hierarchy responses without N+1 sub-queries.
func (r *EntityDocumentRepository) ListAll(ctx context.Context) ([]*BulkDoc, error) {
	query := `
		SELECT ed.entity_type, ed.entity_id, d.title, d.file_path
		FROM documents d
		INNER JOIN entity_documents ed ON d.id = ed.document_id
		ORDER BY ed.entity_type, ed.entity_id, ed.created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all entity documents: %w", err)
	}
	defer rows.Close()

	docs := make([]*BulkDoc, 0)
	for rows.Next() {
		d := &BulkDoc{}
		if err := rows.Scan(&d.EntityType, &d.EntityID, &d.Title, &d.FilePath); err != nil {
			return nil, fmt.Errorf("failed to scan bulk entity document: %w", err)
		}
		docs = append(docs, d)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entity documents: %w", err)
	}

	return docs, nil
}

// ListForEntity returns all documents linked to a specific entity,
// ordered by link creation time (most recently linked first).
// Returns an empty non-nil slice if no documents are linked.
func (r *EntityDocumentRepository) ListForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error) {
	if err := validateEntityType(entityType); err != nil {
		return nil, err
	}
	if err := validateEntityID(entityID); err != nil {
		return nil, err
	}

	query := `
		SELECT d.id, d.title, d.file_path, ed.created_at
		FROM documents d
		INNER JOIN entity_documents ed ON d.id = ed.document_id
		WHERE ed.entity_type = ? AND ed.entity_id = ?
		ORDER BY ed.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents for entity: %w", err)
	}
	defer rows.Close()

	docs := make([]*models.Document, 0)
	for rows.Next() {
		doc := &models.Document{}
		err := rows.Scan(&doc.ID, &doc.Title, &doc.FilePath, &doc.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating documents: %w", err)
	}

	return docs, nil
}
