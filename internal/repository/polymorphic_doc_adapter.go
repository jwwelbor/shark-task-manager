package repository

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// PolymorphicDocRepoAdapter implements the config.DocumentRepository interface
// by delegating to EntityDocumentRepository.ListForEntity().
//
// This adapter allows existing service-layer callers (which depend on
// ListForEpic/ListForFeature/ListForTask) to transparently use the
// polymorphic entity_documents table without changing their calling code.
type PolymorphicDocRepoAdapter struct {
	entityDocRepo *EntityDocumentRepository
}

// NewPolymorphicDocRepoAdapter creates a new adapter wrapping the given
// EntityDocumentRepository.
func NewPolymorphicDocRepoAdapter(entityDocRepo *EntityDocumentRepository) *PolymorphicDocRepoAdapter {
	return &PolymorphicDocRepoAdapter{entityDocRepo: entityDocRepo}
}

// ListForEpic returns all documents linked to an epic via the polymorphic
// entity_documents table.
func (a *PolymorphicDocRepoAdapter) ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error) {
	return a.entityDocRepo.ListForEntity(ctx, models.EntityTypeEpic, epicID)
}

// ListForFeature returns all documents linked to a feature via the polymorphic
// entity_documents table.
func (a *PolymorphicDocRepoAdapter) ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error) {
	return a.entityDocRepo.ListForEntity(ctx, models.EntityTypeFeature, featureID)
}

// ListForTask returns all documents linked to a task via the polymorphic
// entity_documents table.
func (a *PolymorphicDocRepoAdapter) ListForTask(ctx context.Context, taskID int64) ([]*models.Document, error) {
	return a.entityDocRepo.ListForEntity(ctx, models.EntityTypeTask, taskID)
}
