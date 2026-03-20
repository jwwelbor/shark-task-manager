package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EntityDocumentRepository defines the minimal interface for document operations
// shared across all entity types (Task, Feature, Epic, Bug, ChangeCard).
type EntityDocumentRepository interface {
	CreateOrGet(ctx context.Context, title, filePath string) (*models.Document, error)
	GetByTitle(ctx context.Context, title string) (*models.Document, error)
}

// EntityDocumentService provides shared document link/unlink/list operations
// parameterized by entity type. It is used by all entity services to avoid
// duplicating document management logic.
//
// The service uses function callbacks for entity-specific operations:
//   - linkFn: links a document to an entity (e.g., LinkToEpic, LinkToFeature)
//   - unlinkFn: unlinks a document from an entity
//   - listFn: lists documents for an entity by ID
//   - entityLookupFn: looks up an entity by key, returning its ID
type EntityDocumentService struct {
	writableRepo   EntityDocumentRepository
	entityType     models.EntityType // entity type for error messages
	linkFn         func(ctx context.Context, entityID, documentID int64) error
	unlinkFn       func(ctx context.Context, entityID, documentID int64) error
	listFn         func(ctx context.Context, entityID int64) ([]*models.Document, error)
	entityLookupFn func(ctx context.Context, key string) (int64, error)
}

// NewEntityDocumentService creates a new EntityDocumentService for a specific entity type.
//
// Parameters:
//   - writableRepo: repository supporting CreateOrGet and GetByTitle operations
//   - entityType: entity type for error messages (e.g., models.EntityTypeEpic, models.EntityTypeFeature)
//   - linkFn: entity-specific function to link a document (e.g., LinkToTask, LinkToFeature, LinkToEpic)
//   - unlinkFn: entity-specific function to unlink a document
//   - listFn: entity-specific function to list documents by entity ID (can be nil for graceful degradation)
//   - entityLookupFn: function to look up entity by key and return its ID
func NewEntityDocumentService(
	writableRepo EntityDocumentRepository,
	entityType models.EntityType,
	linkFn func(ctx context.Context, entityID, documentID int64) error,
	unlinkFn func(ctx context.Context, entityID, documentID int64) error,
	listFn func(ctx context.Context, entityID int64) ([]*models.Document, error),
	entityLookupFn func(ctx context.Context, key string) (int64, error),
) *EntityDocumentService {
	return &EntityDocumentService{
		writableRepo:   writableRepo,
		entityType:     entityType,
		linkFn:         linkFn,
		unlinkFn:       unlinkFn,
		listFn:         listFn,
		entityLookupFn: entityLookupFn,
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
	entityType models.EntityType, entityKey string,
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
	entityType models.EntityType, entityKey string,
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

// LinkDocumentByKey links a document to an entity identified by key.
// Creates the document record if it doesn't exist.
//
// Returns:
//   - *models.Document: the created or retrieved document
//   - error: if entity not found, or document creation/linking fails
func (s *EntityDocumentService) LinkDocumentByKey(ctx context.Context, entityKey, title, path string) (*models.Document, error) {
	entityID, err := s.entityLookupFn(ctx, entityKey)
	if err != nil {
		return nil, fmt.Errorf("%s not found: %w", s.entityType, err)
	}

	return s.LinkDocument(ctx, entityID, title, path, s.entityType, entityKey)
}

// UnlinkDocumentByKey removes a document link from an entity identified by key.
// This operation is idempotent for missing documents.
//
// Returns:
//   - error: if entity not found, or unlinking fails
func (s *EntityDocumentService) UnlinkDocumentByKey(ctx context.Context, entityKey, title string) error {
	entityID, err := s.entityLookupFn(ctx, entityKey)
	if err != nil {
		return fmt.Errorf("%s not found: %w", s.entityType, err)
	}

	return s.UnlinkDocument(ctx, entityID, title, s.entityType, entityKey)
}

// ListDocumentsByKey returns all documents linked to an entity identified by key.
// Returns an empty slice if listFn is nil (graceful degradation).
//
// Returns:
//   - []*models.Document: list of linked documents (never nil)
//   - error: if entity not found or listing fails
func (s *EntityDocumentService) ListDocumentsByKey(ctx context.Context, entityKey string) ([]*models.Document, error) {
	if s.listFn == nil {
		return []*models.Document{}, nil
	}

	entityID, err := s.entityLookupFn(ctx, entityKey)
	if err != nil {
		return nil, fmt.Errorf("%s not found: %w", s.entityType, err)
	}

	docs, err := s.listFn(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents for %s %s: %w", s.entityType, entityKey, err)
	}
	if docs == nil {
		return []*models.Document{}, nil
	}
	return docs, nil
}
