package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EntityRelationshipRepository defines the data access contract for
// EntityRelationshipService. The concrete implementation is
// *repository.EntityRelationshipRepository.
type EntityRelationshipRepository interface {
	// Create inserts a new polymorphic relationship.
	// Returns an error if the UNIQUE constraint is violated.
	Create(ctx context.Context, rel *models.EntityRelationship) error

	// Delete removes a relationship by its primary key.
	Delete(ctx context.Context, id int64) error

	// DeleteByEntitiesAndType removes the specific directed relationship
	// between two entities of the given type.
	DeleteByEntitiesAndType(
		ctx context.Context,
		fromType models.EntityType, fromID int64,
		toType models.EntityType, toID int64,
		relType models.EntityRelationshipType,
	) error

	// GetByEntity returns all relationships where the entity appears
	// on either the from or to side (bidirectional).
	GetByEntity(
		ctx context.Context,
		entityType models.EntityType,
		entityID int64,
	) ([]*models.EntityRelationship, error)

	// GetOutgoing returns relationships where this entity is the source,
	// optionally filtered by one or more relationship types.
	GetOutgoing(
		ctx context.Context,
		entityType models.EntityType,
		entityID int64,
		relTypes []models.EntityRelationshipType,
	) ([]*models.EntityRelationship, error)

	// GetIncoming returns relationships where this entity is the target,
	// optionally filtered by one or more relationship types.
	GetIncoming(
		ctx context.Context,
		entityType models.EntityType,
		entityID int64,
		relTypes []models.EntityRelationshipType,
	) ([]*models.EntityRelationship, error)
}

// EntityRelationshipService manages polymorphic entity relationships
// with cycle detection for dependency-type relationships.
type EntityRelationshipService struct {
	repo EntityRelationshipRepository
}

// NewEntityRelationshipService creates a new EntityRelationshipService.
func NewEntityRelationshipService(repo EntityRelationshipRepository) *EntityRelationshipService {
	requireNonNil(repo, "EntityRelationshipService requires a non-nil EntityRelationshipRepository")
	return &EntityRelationshipService{repo: repo}
}

// CreateRelationship validates and creates a new relationship between two entities.
// For cyclic relationship types (depends_on, blocks), cycle detection is performed
// before creation regardless of whether the entities are the same type.
func (s *EntityRelationshipService) CreateRelationship(
	ctx context.Context,
	fromType models.EntityType, fromID int64,
	toType models.EntityType, toID int64,
	relType models.EntityRelationshipType,
) (*models.EntityRelationship, error) {
	rel := &models.EntityRelationship{
		FromEntityType:   fromType,
		FromEntityID:     fromID,
		ToEntityType:     toType,
		ToEntityID:       toID,
		RelationshipType: relType,
	}

	// Structural validation (checks entity types, rel type, self-reference)
	if err := rel.Validate(); err != nil {
		return nil, fmt.Errorf("invalid relationship: %w", err)
	}

	// Cycle detection for cyclic relationship types (applies across entity types)
	if models.CyclicRelationshipTypes[rel.RelationshipType] {
		hasCycle, err := s.DetectCycle(ctx, fromType, fromID, toType, toID, relType)
		if err != nil {
			return nil, fmt.Errorf("cycle detection failed: %w", err)
		}
		if hasCycle {
			return nil, fmt.Errorf("cannot create relationship: would create a cycle (%s(%d) -[%s]-> %s(%d))",
				fromType, fromID, relType, toType, toID)
		}
	}

	if err := s.repo.Create(ctx, rel); err != nil {
		return nil, fmt.Errorf("failed to create relationship: %w", err)
	}

	return rel, nil
}

// DeleteRelationship removes a relationship by its primary key ID.
func (s *EntityRelationshipService) DeleteRelationship(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete relationship %d: %w", id, err)
	}
	return nil
}

// UnlinkEntities removes a specific directed relationship between two entities.
func (s *EntityRelationshipService) UnlinkEntities(
	ctx context.Context,
	fromType models.EntityType, fromID int64,
	toType models.EntityType, toID int64,
	relType models.EntityRelationshipType,
) error {
	if err := s.repo.DeleteByEntitiesAndType(ctx, fromType, fromID, toType, toID, relType); err != nil {
		return fmt.Errorf("failed to unlink %s(%d) -[%s]-> %s(%d): %w",
			fromType, fromID, relType, toType, toID, err)
	}
	return nil
}

// GetRelationships returns all relationships (incoming and outgoing) for an entity.
func (s *EntityRelationshipService) GetRelationships(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
) ([]*models.EntityRelationship, error) {
	rels, err := s.repo.GetByEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships for %s(%d): %w", entityType, entityID, err)
	}
	return rels, nil
}

// GetOutgoing returns relationships where this entity is the source,
// optionally filtered by relationship types.
func (s *EntityRelationshipService) GetOutgoing(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
	relTypes []models.EntityRelationshipType,
) ([]*models.EntityRelationship, error) {
	rels, err := s.repo.GetOutgoing(ctx, entityType, entityID, relTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to get outgoing relationships for %s(%d): %w", entityType, entityID, err)
	}
	return rels, nil
}

// GetIncoming returns relationships where this entity is the target,
// optionally filtered by relationship types.
func (s *EntityRelationshipService) GetIncoming(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
	relTypes []models.EntityRelationshipType,
) ([]*models.EntityRelationship, error) {
	rels, err := s.repo.GetIncoming(ctx, entityType, entityID, relTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to get incoming relationships for %s(%d): %w", entityType, entityID, err)
	}
	return rels, nil
}

// DetectCycle performs depth-first search to detect whether adding a relationship
// from (fromType, fromID) to (toType, toID) of the given relType would create
// a cycle. Cycle detection applies for cyclic relationship types (depends_on, blocks)
// across all entity types -- the DFS follows edges regardless of entity type.
//
// Returns true if a cycle would be created, false otherwise.
func (s *EntityRelationshipService) DetectCycle(
	ctx context.Context,
	fromType models.EntityType, fromID int64,
	toType models.EntityType, toID int64,
	relType models.EntityRelationshipType,
) (bool, error) {
	// Only check for cyclic relationship types
	if !models.CyclicRelationshipTypes[relType] {
		return false, nil
	}

	// DFS from (toType, toID) following same relType edges.
	// If we reach (fromType, fromID), a cycle exists.
	type node struct {
		entityType models.EntityType
		entityID   int64
	}

	visited := map[node]bool{}
	stack := []node{{toType, toID}}
	target := node{fromType, fromID}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if current == target {
			return true, nil
		}

		if visited[current] {
			continue
		}
		visited[current] = true

		// Get outgoing edges of same relType
		outgoing, err := s.repo.GetOutgoing(ctx, current.entityType, current.entityID,
			[]models.EntityRelationshipType{relType})
		if err != nil {
			return false, fmt.Errorf("cycle detection query failed: %w", err)
		}

		for _, rel := range outgoing {
			next := node{rel.ToEntityType, rel.ToEntityID}
			if !visited[next] {
				stack = append(stack, next)
			}
		}
	}

	return false, nil
}
