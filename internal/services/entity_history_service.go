package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EntityHistoryQuerier is the interface EntityHistoryService depends on for queries.
// Satisfied by *repository.EntityHistoryRepository.
type EntityHistoryQuerier interface {
	ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error)
}

// EntityHistoryService provides history query operations for all entity types.
// It is a read-only service that queries the entity_history table.
// History records are created by EntityService.TransitionStatus (T-E21-F08-006).
//
// This service exists alongside TaskHistoryService (which queries task_history).
// TaskHistoryService is NOT modified -- see ADR-1 in the task spec.
type EntityHistoryService struct {
	historyRepo EntityHistoryQuerier
	registry    *EntityRegistry
}

// NewEntityHistoryService creates a new EntityHistoryService.
// Both parameters are required and must be non-nil.
func NewEntityHistoryService(historyRepo EntityHistoryQuerier, registry *EntityRegistry) *EntityHistoryService {
	requireNonNil(historyRepo, "EntityHistoryService requires a non-nil EntityHistoryQuerier")
	requireNonNil(registry, "EntityHistoryService requires a non-nil EntityRegistry")
	return &EntityHistoryService{historyRepo: historyRepo, registry: registry}
}

// GetHistory retrieves status change history for a specific entity.
// entityType: one of "epic", "feature", "task", "bug", "change"
// entityKey: the entity's key (e.g., "E21", "E21-F07", "E21-F08-001", "B001", "CC-001")
// Returns []*models.EntityHistory in changed_at DESC order.
func (s *EntityHistoryService) GetHistory(ctx context.Context, entityType models.EntityType, entityKey string) ([]*models.EntityHistory, error) {
	// 1. Resolve entity type to repository via registry
	repo, err := s.registry.GetRepository(entityType)
	if err != nil {
		return nil, fmt.Errorf("failed to get history for %s %s: %w", entityType, entityKey, err)
	}

	// 2. Resolve entity key to entity (and thus to ID)
	entity, err := repo.GetByKey(ctx, entityKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get history for %s %s: %w", entityType, entityKey, err)
	}

	// 3. Query history by entity type + entity ID
	return s.historyRepo.ListByEntity(ctx, entityType, entity.GetID())
}
