package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

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
// The returned function resolves an entity key to the full polymorphic entity.
func EntityLookupFnFromRepo(repo EntityKeyLookup) func(ctx context.Context, key string) (models.Entity, error) {
	return func(ctx context.Context, key string) (models.Entity, error) {
		entity, err := repo.GetByKey(ctx, key)
		if err != nil {
			return nil, err
		}
		if entity == nil {
			return nil, fmt.Errorf("entity not found: %s", key)
		}
		return entity, nil
	}
}

// EntityDocumentService provides shared document link/unlink/list operations
// for all entity types. It uses the polymorphic EntityDocumentLinkRepository
// to handle document associations and an entityLookupFn to resolve entity keys
// to (entityID, entityType) pairs.
type EntityDocumentService struct {
	writableRepo   EntityDocumentRepository
	linkRepo       EntityDocumentLinkRepository
	entityLookupFn func(ctx context.Context, key string) (models.Entity, error)
	projectRoot    string
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
	entityLookupFn func(ctx context.Context, key string) (models.Entity, error),
	projectRoot string,
) *EntityDocumentService {
	if projectRoot == "" {
		projectRoot = "."
	}
	return &EntityDocumentService{
		writableRepo:   writableRepo,
		linkRepo:       linkRepo,
		entityLookupFn: entityLookupFn,
		projectRoot:    projectRoot,
	}
}

// LinkDocumentByKey links a document to an entity identified by key.
// Creates the document record if it doesn't exist.
//
// Returns:
//   - *models.Document: the created or retrieved document
//   - error: if entity not found, or document creation/linking fails
func (s *EntityDocumentService) LinkDocumentByKey(ctx context.Context, entityKey, title, path string) (*models.Document, error) {
	entity, err := s.entityLookupFn(ctx, entityKey)
	if err != nil {
		return nil, fmt.Errorf("entity not found: %w", err)
	}
	entityID := entity.GetID()
	entityType := entity.GetEntityType()

	normalizedPath, err := s.normalizeDocumentPath(path, entity)
	if err != nil {
		return nil, fmt.Errorf("invalid document path: %w", err)
	}

	doc, err := s.writableRepo.CreateOrGet(ctx, title, normalizedPath)
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
	entity, err := s.entityLookupFn(ctx, entityKey)
	if err != nil {
		return fmt.Errorf("entity not found: %w", err)
	}
	entityID := entity.GetID()
	entityType := entity.GetEntityType()

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
	entity, err := s.entityLookupFn(ctx, entityKey)
	if err != nil {
		return nil, fmt.Errorf("entity not found: %w", err)
	}
	entityID := entity.GetID()
	entityType := entity.GetEntityType()

	docs, err := s.linkRepo.ListForEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents for %s %s: %w", entityType, entityKey, err)
	}
	if docs == nil {
		return []*models.Document{}, nil
	}
	return docs, nil
}

func (s *EntityDocumentService) normalizeDocumentPath(input string, entity models.Entity) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("absolute paths are not allowed: %s", trimmed)
	}

	if containsPathSeparator(trimmed) {
		return s.normalizeProjectRelativePath(trimmed)
	}

	parentPath := strings.TrimSpace(entity.GetFilePath())
	if parentPath == "" {
		return "", fmt.Errorf("cannot resolve bare filename %q without a parent entity file_path", trimmed)
	}

	parentDir := filepath.Dir(filepath.Clean(parentPath))
	return s.normalizeProjectRelativePath(filepath.Join(parentDir, trimmed))
}

func (s *EntityDocumentService) normalizeProjectRelativePath(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("path cannot resolve to project root")
	}

	absRoot, err := filepath.Abs(s.projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	target := filepath.Join(absRoot, cleaned)
	rel, err := filepath.Rel(absRoot, target)
	if err != nil {
		return "", fmt.Errorf("resolve relative path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes project root: %s", path)
	}

	return filepath.ToSlash(rel), nil
}

func containsPathSeparator(path string) bool {
	return strings.Contains(path, "/") || strings.Contains(path, `\`)
}
