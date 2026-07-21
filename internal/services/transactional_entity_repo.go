package services

import "context"

// transactionalEntityRepo overlays transaction-aware status writes on an
// existing EntityRepository while delegating all reads and non-status methods.
type transactionalEntityRepo struct {
	EntityRepository
	updateStatus          func(context.Context, int64, string) error
	updateStatusIfCurrent func(context.Context, int64, string, string) (bool, error)
}

func (r *transactionalEntityRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.updateStatus(ctx, id, status)
}

func (r *transactionalEntityRepo) UpdateStatusIfCurrent(ctx context.Context, id int64, expectedCurrentStatus, newStatus string) (bool, error) {
	return r.updateStatusIfCurrent(ctx, id, expectedCurrentStatus, newStatus)
}
