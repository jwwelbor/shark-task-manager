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

	// UpdateStatusIfCurrent atomically updates the status only when the current
	// stored status still matches expectedCurrentStatus (case-insensitive).
	// Returns true when a row was updated, false when the status was stale.
	UpdateStatusIfCurrent(ctx context.Context, id int64, expectedCurrentStatus, newStatus string) (bool, error)

	// Update persists all fields of the entity.
	// The entity parameter must be the correct concrete type for this adapter.
	Update(ctx context.Context, entity models.Entity) error

	// GetContextData retrieves the context_data JSON string.
	GetContextData(ctx context.Context, id int64) (*string, error)

	// UpdateContextData updates the context_data JSON string.
	UpdateContextData(ctx context.Context, id int64, data *string) error
}

// noopEntityRepo is a minimal EntityRepository whose methods return zero values.
// It is used in tests and lightweight wiring scenarios where transition/status
// logic is not exercised but the constructor requires a non-nil EntityRepository.
type noopEntityRepo struct{}

func (n *noopEntityRepo) GetByKey(_ context.Context, _ string) (models.Entity, error) {
	return nil, nil
}
func (n *noopEntityRepo) GetByID(_ context.Context, _ int64) (models.Entity, error) {
	return nil, nil
}
func (n *noopEntityRepo) UpdateStatus(_ context.Context, _ int64, _ string) error { return nil }
func (n *noopEntityRepo) UpdateStatusIfCurrent(_ context.Context, _ int64, _ string, _ string) (bool, error) {
	return true, nil
}
func (n *noopEntityRepo) Update(_ context.Context, _ models.Entity) error { return nil }
func (n *noopEntityRepo) GetContextData(_ context.Context, _ int64) (*string, error) {
	return nil, nil
}
func (n *noopEntityRepo) UpdateContextData(_ context.Context, _ int64, _ *string) error { return nil }

// NewNoopEntityRepository returns an EntityRepository whose methods are no-ops.
// Intended for tests and lightweight wiring where transition logic is not exercised.
func NewNoopEntityRepository() EntityRepository {
	return &noopEntityRepo{}
}
