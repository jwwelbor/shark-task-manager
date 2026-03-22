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

// EntityDocumentLinkRepository defines the polymorphic document linking interface.
// It handles Link/Unlink/ListForEntity for any entity type via entity_type + entity_id.
// The concrete *repository.EntityDocumentRepository satisfies this interface.
type EntityDocumentLinkRepository interface {
	Link(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64, linkType string) error
	Unlink(ctx context.Context, entityType models.EntityType, entityID int64, documentID int64) error
	ListForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error)
}

// EntityKeyLookup defines the minimal interface needed to resolve an entity key
// to a polymorphic models.Entity. This is satisfied by EntityRepository and by
// any adapter wrapping a typed repository whose GetByKey returns a concrete type
// implementing models.Entity.
type EntityKeyLookup interface {
	GetByKey(ctx context.Context, key string) (models.Entity, error)
}

// EntityLookupFnFromRepo creates an entity lookup function from any EntityKeyLookup.
// This eliminates duplicate closures in SetWritableDocRepo methods across services.
// The returned function resolves an entity key to (entityID, entityType) via the
// polymorphic Entity interface.
func EntityLookupFnFromRepo(repo EntityKeyLookup) func(ctx context.Context, key string) (int64, models.EntityType, error) {
	return func(ctx context.Context, key string) (int64, models.EntityType, error) {
		entity, err := repo.GetByKey(ctx, key)
		if err != nil {
			return 0, "", err
		}
		if entity == nil {
			return 0, "", fmt.Errorf("entity not found: %s", key)
		}
		return entity.GetID(), entity.GetEntityType(), nil
	}
}

// EntityDocumentService provides shared document link/unlink/list operations
// for all entity types. It uses the polymorphic EntityDocumentLinkRepository
// to handle document associations and an entityLookupFn to resolve entity keys
// to (entityID, entityType) pairs.
type EntityDocumentService struct {
	writableRepo   EntityDocumentRepository
	linkRepo       EntityDocumentLinkRepository
	entityLookupFn func(ctx context.Context, key string) (int64, models.EntityType, error)
}

// NewEntityDocumentService creates a new EntityDocumentService.
//
// Parameters:
//   - writableRepo: repository supporting CreateOrGet and GetByTitle operations
//   - linkRepo: polymorphic repository for Link/Unlink/ListForEntity operations
//   - entityLookupFn: function to resolve an entity key to (entityID, entityType)
func NewEntityDocumentService(
	writableRepo EntityDocumentRepository,
	linkRepo EntityDocumentLinkRepository,
	entityLookupFn func(ctx context.Context, key string) (int64, models.EntityType, error),
) *EntityDocumentService {
	return &EntityDocumentService{
		writableRepo:   writableRepo,
		linkRepo:       linkRepo,
		entityLookupFn: entityLookupFn,
	}
}

// LinkDocumentByKey links a document to an entity identified by key.
// Creates the document record if it doesn't exist.
//
// Returns:
//   - *models.Document: the created or retrieved document
//   - error: if entity not found, or document creation/linking fails
func (s *EntityDocumentService) LinkDocumentByKey(ctx context.Context, entityKey, title, path string) (*models.Document, error) {
	entityID, entityType, err := s.entityLookupFn(ctx, entityKey)
	if err != nil {
		return nil, fmt.Errorf("entity not found: %w", err)
	}

	doc, err := s.writableRepo.CreateOrGet(ctx, title, path)
	if err != nil {
		return nil, fmt.Errorf("failed to create or get document: %w", err)
	}

	if err := s.linkRepo.Link(ctx, entityType, entityID, doc.ID, "general"); err != nil {
		return nil, fmt.Errorf("failed to link document to %s %s: %w", entityType, entityKey, err)
	}

	return doc, nil
}

// UnlinkDocumentByKey removes a document link from an entity identified by key.
// This operation is idempotent for missing documents.
//
// Returns:
//   - error: if entity not found, or unlinking fails
func (s *EntityDocumentService) UnlinkDocumentByKey(ctx context.Context, entityKey, title string) error {
	entityID, entityType, err := s.entityLookupFn(ctx, entityKey)
	if err != nil {
		return fmt.Errorf("entity not found: %w", err)
	}

	doc, err := s.writableRepo.GetByTitle(ctx, title)
	if err != nil {
		// Document doesn't exist -- idempotent, treat as success
		return nil
	}

	if err := s.linkRepo.Unlink(ctx, entityType, entityID, doc.ID); err != nil {
		return fmt.Errorf("failed to unlink document from %s %s: %w", entityType, entityKey, err)
	}

	return nil
}

// ListDocumentsByKey returns all documents linked to an entity identified by key.
// Returns an empty non-nil slice if no documents exist.
//
// Returns:
//   - []*models.Document: list of linked documents (never nil)
//   - error: if entity not found or listing fails
func (s *EntityDocumentService) ListDocumentsByKey(ctx context.Context, entityKey string) ([]*models.Document, error) {
	entityID, entityType, err := s.entityLookupFn(ctx, entityKey)
	if err != nil {
		return nil, fmt.Errorf("entity not found: %w", err)
	}

	docs, err := s.linkRepo.ListForEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents for %s %s: %w", entityType, entityKey, err)
	}
	if docs == nil {
		return []*models.Document{}, nil
	}
	return docs, nil
}
