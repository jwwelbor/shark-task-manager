package services

import (
	"context"
	"log/slog"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// SearchIndexer is the service-layer seam for keeping the unified FTS index
// current after entity and note writes. It is optional on services; nil means
// search indexing is disabled for that service instance.
type SearchIndexer interface {
	IndexEntity(ctx context.Context, entityType models.EntityType, entityID int64) error
	RemoveEntity(ctx context.Context, entityType models.EntityType, entityID int64) error
}

func indexEntityIfConfigured(ctx context.Context, indexer SearchIndexer, entityType models.EntityType, entityID int64) error {
	if indexer == nil || entityID == 0 {
		return nil
	}
	if err := indexer.IndexEntity(ctx, entityType, entityID); err != nil {
		slog.WarnContext(ctx,
			"search index update failed; rebuild the unified search index to recover",
			"entity_type", entityType,
			"entity_id", entityID,
			"error", err,
		)
	}
	return nil
}

func removeEntityFromIndexIfConfigured(ctx context.Context, indexer SearchIndexer, entityType models.EntityType, entityID int64) error {
	if indexer == nil || entityID == 0 {
		return nil
	}
	if err := indexer.RemoveEntity(ctx, entityType, entityID); err != nil {
		slog.WarnContext(ctx,
			"search index remove failed; rebuild the unified search index to recover",
			"entity_type", entityType,
			"entity_id", entityID,
			"error", err,
		)
	}
	return nil
}
