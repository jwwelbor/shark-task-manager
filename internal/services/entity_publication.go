package services

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const publicationRollbackTimeout = 5 * time.Second

func rollbackPublication(ctx context.Context, deleteRow func(context.Context, int64) error, id int64, entity, successRow string, writeErr error) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), publicationRollbackTimeout)
	defer cancel()

	if rollbackErr := deleteRow(cleanupCtx, id); rollbackErr != nil {
		return errors.Join(
			fmt.Errorf("failed to write %s file: %w", entity, writeErr),
			fmt.Errorf("%s row rollback failed: %w", entity, rollbackErr),
		)
	}
	return fmt.Errorf("failed to write %s file: %w; %s rollback succeeded", entity, writeErr, successRow)
}
