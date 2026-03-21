package services

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EntityRepository provides polymorphic data access for any entity type.
// It wraps typed repositories to support cross-cutting operations.
//
// Implementations are thin adapters that delegate to typed repositories
// and perform type assertions where needed.
type EntityRepository interface {
	// GetByKey retrieves an entity by its key.
	GetByKey(ctx context.Context, key string) (models.Entity, error)

	// GetByID retrieves an entity by its database ID.
	GetByID(ctx context.Context, id int64) (models.Entity, error)

	// UpdateStatus updates the status field of an entity.
	UpdateStatus(ctx context.Context, id int64, status string) error

	// Update persists all fields of the entity.
	// The entity parameter must be the correct concrete type for this adapter.
	Update(ctx context.Context, entity models.Entity) error

	// GetContextData retrieves the context_data JSON string.
	GetContextData(ctx context.Context, id int64) (*string, error)

	// UpdateContextData updates the context_data JSON string.
	UpdateContextData(ctx context.Context, id int64, data *string) error
}
