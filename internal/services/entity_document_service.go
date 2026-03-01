package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EntityDocumentRepository defines the minimal interface for document operations
// shared across all entity types (Task, Feature, Epic).
// This replaces the DocumentRepo interface from the removed document_helpers.go.
type EntityDocumentRepository interface {
	CreateOrGet(ctx context.Context, title, filePath string) (*models.Document, error)
	GetByTitle(ctx context.Context, title string) (*models.Document, error)
}

// EntityDocumentService provides shared document link/unlink/list operations
// parameterized by entity type. It is used by TaskService, FeatureService, and
// EpicService to avoid duplicating document management logic.
//
// Each entity service creates an EntityDocumentService with the appropriate
// writable repository and read-only list repository for its entity type.
type EntityDocumentService struct {
	writableRepo EntityDocumentRepository
	linkFn       func(ctx context.Context, entityID, documentID int64) error
	unlinkFn     func(ctx context.Context, entityID, documentID int64) error
}

// NewEntityDocumentService creates a new EntityDocumentService for a specific entity type.
//
// Parameters:
//   - writableRepo: repository supporting CreateOrGet and GetByTitle operations
//   - linkFn: entity-specific function to link a document (e.g., LinkToTask, LinkToFeature, LinkToEpic)
//   - unlinkFn: entity-specific function to unlink a document
func NewEntityDocumentService(
	writableRepo EntityDocumentRepository,
	linkFn func(ctx context.Context, entityID, documentID int64) error,
	unlinkFn func(ctx context.Context, entityID, documentID int64) error,
) *EntityDocumentService {
	return &EntityDocumentService{
		writableRepo: writableRepo,
		linkFn:       linkFn,
		unlinkFn:     unlinkFn,
	}
}

// LinkDocument links a document to an entity, creating the document record if it doesn't exist.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - entityID: the database ID of the entity to link the document to
//   - title: document title
//   - path: document file path
//   - entityType: human-readable entity type for error messages (e.g., "task", "feature", "epic")
//   - entityKey: human-readable entity key for error messages (e.g., "E07-F01-001")
//
// Returns:
//   - *models.Document: the created or retrieved document
//   - error: if document creation/retrieval or linking fails
func (s *EntityDocumentService) LinkDocument(
	ctx context.Context,
	entityID int64,
	title, path string,
	entityType, entityKey string,
) (*models.Document, error) {
	doc, err := s.writableRepo.CreateOrGet(ctx, title, path)
	if err != nil {
		return nil, fmt.Errorf("failed to create or get document: %w", err)
	}

	if err := s.linkFn(ctx, entityID, doc.ID); err != nil {
		return nil, fmt.Errorf("failed to link document to %s %s: %w", entityType, entityKey, err)
	}

	return doc, nil
}

// UnlinkDocument removes the link between a document and an entity.
// This operation is idempotent: it succeeds even if the document is not linked.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - entityID: the database ID of the entity to unlink the document from
//   - title: document title to look up and unlink
//   - entityType: human-readable entity type for error messages
//   - entityKey: human-readable entity key for error messages
//
// Returns:
//   - error: if document lookup or unlinking fails (missing document is treated as success)
func (s *EntityDocumentService) UnlinkDocument(
	ctx context.Context,
	entityID int64,
	title string,
	entityType, entityKey string,
) error {
	doc, err := s.writableRepo.GetByTitle(ctx, title)
	if err != nil {
		// Document doesn't exist — idempotent, treat as success
		return nil
	}

	if err := s.unlinkFn(ctx, entityID, doc.ID); err != nil {
		return fmt.Errorf("failed to unlink document from %s %s: %w", entityType, entityKey, err)
	}

	return nil
}
