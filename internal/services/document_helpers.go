package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// DocumentRepo defines the common document operations shared across all entity types.
// All three writable document repository interfaces (EpicWritableDocumentRepository,
// FeatureWritableDocumentRepository, TaskWritableDocumentRepository) include these methods.
type DocumentRepo interface {
	CreateOrGet(ctx context.Context, title, filePath string) (*models.Document, error)
	GetByTitle(ctx context.Context, title string) (*models.Document, error)
}

// linkDocumentToEntity is a shared helper that links a document to any entity type.
// It creates or retrieves the document, then calls the entity-specific link function.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - docRepo: document repository for CreateOrGet
//   - linkFn: entity-specific function to link doc to entity (e.g., LinkToEpic, LinkToFeature, LinkToTask)
//   - entityID: the database ID of the entity to link the document to
//   - docTitle: the document title
//   - docPath: the document file path
//   - entityType: human-readable entity type for error messages (e.g., "epic", "feature", "task")
//   - entityKey: human-readable entity key for error messages (e.g., "E07", "E07-F01")
//
// Returns:
//   - *models.Document: the created or retrieved document
//   - error: if document creation/retrieval or linking fails
func linkDocumentToEntity(
	ctx context.Context,
	docRepo DocumentRepo,
	linkFn func(ctx context.Context, entityID, documentID int64) error,
	entityID int64,
	docTitle, docPath string,
	entityType, entityKey string,
) (*models.Document, error) {
	doc, err := docRepo.CreateOrGet(ctx, docTitle, docPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create or get document: %w", err)
	}

	if err := linkFn(ctx, entityID, doc.ID); err != nil {
		return nil, fmt.Errorf("failed to link document to %s %s: %w", entityType, entityKey, err)
	}

	return doc, nil
}

// unlinkDocumentFromEntity is a shared helper that unlinks a document from any entity type.
// It looks up the document by title, then calls the entity-specific unlink function.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - docRepo: document repository for GetByTitle
//   - unlinkFn: entity-specific function to unlink doc from entity
//   - entityID: the database ID of the entity to unlink the document from
//   - docTitle: the document title to look up
//   - entityType: human-readable entity type for error messages
//   - entityKey: human-readable entity key for error messages
//
// Returns:
//   - error: if document lookup or unlinking fails
func unlinkDocumentFromEntity(
	ctx context.Context,
	docRepo DocumentRepo,
	unlinkFn func(ctx context.Context, entityID, documentID int64) error,
	entityID int64,
	docTitle string,
	entityType, entityKey string,
) error {
	doc, err := docRepo.GetByTitle(ctx, docTitle)
	if err != nil {
		return fmt.Errorf("document not found: %w", err)
	}

	if err := unlinkFn(ctx, entityID, doc.ID); err != nil {
		return fmt.Errorf("failed to unlink document from %s %s: %w", entityType, entityKey, err)
	}

	return nil
}
