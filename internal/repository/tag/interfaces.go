// Package tag provides TagRepository (vocabulary CRUD) and EntityTagRepository
// (polymorphic join-table operations) for the entity-tagging feature (E28).
//
// Per architecture.md §1.3: repositories own SQL only. No business logic,
// no config reads, no password checks, no workflow rules live here.
//
// Service layer consumers depend on the interfaces exported by this file, not
// on the concrete struct types, enabling mock injection in service tests.
package tag

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TagRepositoryInterface defines the vocabulary CRUD contract consumed by TagService.
// Concrete type: *TagRepository.
type TagRepositoryInterface interface {
	// Create inserts a new tag with the given name and returns the created Tag.
	// Returns an error if the name already exists (UNIQUE constraint).
	Create(ctx context.Context, name string) (*models.Tag, error)

	// GetByName looks up a tag by name using case-insensitive matching (NOCASE collation).
	// Returns an error if not found.
	GetByName(ctx context.Context, name string) (*models.Tag, error)

	// GetByID looks up a tag by its primary key.
	// Returns an error if not found.
	GetByID(ctx context.Context, id int64) (*models.Tag, error)

	// List returns all tags in the vocabulary, ordered by name ascending.
	List(ctx context.Context) ([]*models.Tag, error)

	// Rename renames the tag identified by id to newName (per ADR-8).
	// Issues a single UPDATE tags SET name = ? — no entity_tags rows are touched.
	// Returns ErrTagConflict if newName already exists.
	// Returns an error if id is not found.
	Rename(ctx context.Context, id int64, newName string) (*models.Tag, error)

	// Delete removes the tag identified by id (per ADR-9).
	// When force is false and the tag has entity_tags rows, returns ErrTagInUse.
	// When force is true, deletes all entity_tags rows for this tag in the same
	// transaction before removing the tags row.
	Delete(ctx context.Context, id int64, force bool) error
}

// EntityTagRepositoryInterface defines the polymorphic join-table contract consumed
// by TagService. Concrete type: *EntityTagRepository.
type EntityTagRepositoryInterface interface {
	// Attach creates an (entity_type, entity_id, tag_id) row. Idempotent —
	// a duplicate attach does not return an error (INSERT OR IGNORE semantics).
	Attach(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error

	// Detach removes an (entity_type, entity_id, tag_id) row if it exists.
	// Removing a non-existent link is a no-op (not an error).
	Detach(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error

	// ListByEntity returns all EntityTagLink rows for the given entity, ordered by tag_id.
	ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityTagLink, error)

	// ListByTag returns all EntityTagLink rows across all entity types for a given tag.
	ListByTag(ctx context.Context, tagID int64) ([]*models.EntityTagLink, error)

	// ListByEntityType returns EntityTagLink rows filtered by both tagID and entityType.
	// Supports "list all <entity type> with this tag" queries (ADR-5 SQL shape).
	ListByEntityType(ctx context.Context, entityType models.EntityType, tagID int64) ([]*models.EntityTagLink, error)

	// CountByTag returns the total number of entity_tags rows for a given tag.
	// Used by TagService to decide whether Delete(force=false) should fail with ErrTagInUse.
	CountByTag(ctx context.Context, tagID int64) (int64, error)
}
