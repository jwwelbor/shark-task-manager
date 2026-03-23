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

// TaskByIDResolver resolves task IDs to task models.
// Used by EntityRelationshipService for enriching relationships with task details.
type TaskByIDResolver interface {
	GetByID(ctx context.Context, id int64) (*models.Task, error)
}

// EntityRelationshipService manages polymorphic entity relationships
// with cycle detection for dependency-type relationships.
type EntityRelationshipService struct {
	repo         EntityRelationshipRepository
	taskResolver TaskByIDResolver // optional: for enriching task relationships
}

// NewEntityRelationshipService creates a new EntityRelationshipService.
// The taskResolver parameter is optional (can be nil) for callers that only use
// basic relationship CRUD. The task-specific methods (GetTaskRelationships,
// GetTaskBlockedBy, GetTaskBlocks) return an error if called without a resolver.
func NewEntityRelationshipService(repo EntityRelationshipRepository, taskResolver TaskByIDResolver) *EntityRelationshipService {
	requireNonNil(repo, "EntityRelationshipService requires a non-nil EntityRelationshipRepository")
	return &EntityRelationshipService{repo: repo, taskResolver: taskResolver}
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

// GetTaskRelationships returns all task-to-task relationships for a task,
// enriched with task details, optionally filtered by relationship type.
func (s *EntityRelationshipService) GetTaskRelationships(
	ctx context.Context, taskID int64, typeFilter []string,
) ([]RelationshipWithTask, error) {
	if s.taskResolver == nil {
		return nil, fmt.Errorf("task resolver not configured on EntityRelationshipService")
	}

	allRels, err := s.repo.GetByEntity(ctx, models.EntityTypeTask, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships for task(%d): %w", taskID, err)
	}

	return s.resolveTaskRelationships(ctx, allRels, taskID, typeFilter)
}

// GetTaskBlockedBy returns tasks that this task depends on (outgoing depends_on).
func (s *EntityRelationshipService) GetTaskBlockedBy(
	ctx context.Context, taskID int64,
) ([]RelationshipWithTask, error) {
	if s.taskResolver == nil {
		return nil, fmt.Errorf("task resolver not configured on EntityRelationshipService")
	}

	deps, err := s.repo.GetOutgoing(ctx, models.EntityTypeTask, taskID,
		[]models.EntityRelationshipType{models.EntityRelDependsOn})
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies for task(%d): %w", taskID, err)
	}

	return s.resolveTaskRelationships(ctx, deps, taskID, nil)
}

// GetTaskBlocks returns tasks that depend on this task (incoming depends_on + outgoing blocks).
func (s *EntityRelationshipService) GetTaskBlocks(
	ctx context.Context, taskID int64,
) ([]RelationshipWithTask, error) {
	if s.taskResolver == nil {
		return nil, fmt.Errorf("task resolver not configured on EntityRelationshipService")
	}

	// Incoming depends_on: other tasks that depend on this task
	incoming, err := s.repo.GetIncoming(ctx, models.EntityTypeTask, taskID,
		[]models.EntityRelationshipType{models.EntityRelDependsOn})
	if err != nil {
		return nil, fmt.Errorf("failed to get incoming dependencies for task(%d): %w", taskID, err)
	}

	// Outgoing blocks: this task explicitly blocks other tasks
	outgoing, err := s.repo.GetOutgoing(ctx, models.EntityTypeTask, taskID,
		[]models.EntityRelationshipType{models.EntityRelBlocks})
	if err != nil {
		return nil, fmt.Errorf("failed to get explicit blocks for task(%d): %w", taskID, err)
	}

	allBlocked := append(incoming, outgoing...)

	return s.resolveTaskRelationships(ctx, allBlocked, taskID, nil)
}

// resolveTaskRelationships filters entity relationships to task-to-task only,
// resolves related task IDs, and returns enriched RelationshipWithTask results.
func (s *EntityRelationshipService) resolveTaskRelationships(
	ctx context.Context,
	rels []*models.EntityRelationship,
	selfTaskID int64,
	typeFilter []string,
) ([]RelationshipWithTask, error) {
	var result []RelationshipWithTask
	for _, rel := range rels {
		// Only show task-to-task relationships
		if rel.FromEntityType != models.EntityTypeTask || rel.ToEntityType != models.EntityTypeTask {
			continue
		}

		// Apply type filter
		if len(typeFilter) > 0 {
			found := false
			for _, ft := range typeFilter {
				if string(rel.RelationshipType) == ft {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		direction := "outgoing"
		relatedTaskID := rel.ToEntityID
		if rel.FromEntityID != selfTaskID {
			direction = "incoming"
			relatedTaskID = rel.FromEntityID
		}

		relatedTask, err := s.taskResolver.GetByID(ctx, relatedTaskID)
		if err != nil {
			continue
		}

		result = append(result, RelationshipWithTask{
			RelationshipType: string(rel.RelationshipType),
			Direction:        direction,
			TaskKey:          relatedTask.Key,
			TaskTitle:        relatedTask.Title,
			TaskStatus:       string(relatedTask.Status),
		})
	}

	return result, nil
}
