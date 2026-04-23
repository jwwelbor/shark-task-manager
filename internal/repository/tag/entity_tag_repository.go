package tag

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/repository/repoutil"
)

// entityTagTracer is the per-type tracer for EntityTagRepository operations.
var entityTagTracer = repoutil.NewTracer("internal/repository/tag/entity_tag")

// EntityTagRepository handles read/write operations on the entity_tags
// polymorphic join table. It owns SQL only — no business logic lives here.
type EntityTagRepository struct {
	db *dbconn.DB
}

// NewEntityTagRepository creates a new EntityTagRepository backed by db.
func NewEntityTagRepository(db *dbconn.DB) *EntityTagRepository {
	return &EntityTagRepository{db: db}
}

// Attach creates an (entity_type, entity_id, tag_id) row.
// Uses INSERT OR IGNORE so that attaching the same combination twice is idempotent.
func (r *EntityTagRepository) Attach(ctx context.Context, entityType models.EntityType, entityID, tagID int64) (retErr error) {
	ctx, span := entityTagTracer.Start(ctx, "EntityTagRepository.Attach",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "INSERT"),
			attribute.String("db.table", "entity_tags"),
			attribute.String("entity.type", string(entityType)),
			attribute.Int64("entity.id", entityID),
			attribute.Int64("tag.id", tagID),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		INSERT OR IGNORE INTO entity_tags (entity_type, entity_id, tag_id)
		VALUES (?, ?, ?)
	`

	if _, err := r.db.ExecContext(ctx, query, entityType, entityID, tagID); err != nil {
		return fmt.Errorf("attach tag %d to %s/%d: %w", tagID, entityType, entityID, err)
	}
	return nil
}

// Detach removes the (entity_type, entity_id, tag_id) row if it exists.
// Removing a non-existent link is a no-op.
func (r *EntityTagRepository) Detach(ctx context.Context, entityType models.EntityType, entityID, tagID int64) (retErr error) {
	ctx, span := entityTagTracer.Start(ctx, "EntityTagRepository.Detach",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "DELETE"),
			attribute.String("db.table", "entity_tags"),
			attribute.String("entity.type", string(entityType)),
			attribute.Int64("entity.id", entityID),
			attribute.Int64("tag.id", tagID),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		DELETE FROM entity_tags
		WHERE entity_type = ? AND entity_id = ? AND tag_id = ?
	`

	if _, err := r.db.ExecContext(ctx, query, entityType, entityID, tagID); err != nil {
		return fmt.Errorf("detach tag %d from %s/%d: %w", tagID, entityType, entityID, err)
	}
	return nil
}

// ListByEntity returns all EntityTagLink rows for the given entity, ordered by tag_id.
// Returns an empty (non-nil) slice when no tags are attached.
func (r *EntityTagRepository) ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) (_ []*models.EntityTagLink, retErr error) {
	ctx, span := entityTagTracer.Start(ctx, "EntityTagRepository.ListByEntity",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "entity_tags"),
			attribute.String("entity.type", string(entityType)),
			attribute.Int64("entity.id", entityID),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, entity_type, entity_id, tag_id, created_at
		FROM entity_tags
		WHERE entity_type = ? AND entity_id = ?
		ORDER BY tag_id ASC
	`

	return r.queryLinks(ctx, query, entityType, entityID)
}

// ListByTag returns all EntityTagLink rows across all entity types for the given
// tag, ordered by entity_type, entity_id.
// Returns an empty (non-nil) slice when no entities carry this tag.
func (r *EntityTagRepository) ListByTag(ctx context.Context, tagID int64) (_ []*models.EntityTagLink, retErr error) {
	ctx, span := entityTagTracer.Start(ctx, "EntityTagRepository.ListByTag",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "entity_tags"),
			attribute.Int64("tag.id", tagID),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, entity_type, entity_id, tag_id, created_at
		FROM entity_tags
		WHERE tag_id = ?
		ORDER BY entity_type ASC, entity_id ASC
	`

	return r.queryLinks(ctx, query, tagID)
}

// ListByEntityType returns EntityTagLink rows filtered by both tagID and entityType.
// Supports "list all <entity type> with this tag" queries (ADR-5 §idx_entity_tags_tag_entity).
// Returns an empty (non-nil) slice when no matches exist.
func (r *EntityTagRepository) ListByEntityType(ctx context.Context, entityType models.EntityType, tagID int64) (_ []*models.EntityTagLink, retErr error) {
	ctx, span := entityTagTracer.Start(ctx, "EntityTagRepository.ListByEntityType",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "entity_tags"),
			attribute.String("entity.type", string(entityType)),
			attribute.Int64("tag.id", tagID),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, entity_type, entity_id, tag_id, created_at
		FROM entity_tags
		WHERE tag_id = ? AND entity_type = ?
		ORDER BY entity_id ASC
	`

	return r.queryLinks(ctx, query, tagID, entityType)
}

// CountByTag returns the total number of entity_tags rows for the given tag.
// Used by TagRepository.Delete to enforce the ErrTagInUse check (ADR-9).
// Returns 0 for a non-existent tagID (not an error).
func (r *EntityTagRepository) CountByTag(ctx context.Context, tagID int64) (_ int64, retErr error) {
	ctx, span := entityTagTracer.Start(ctx, "EntityTagRepository.CountByTag",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "entity_tags"),
			attribute.Int64("tag.id", tagID),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `SELECT COUNT(1) FROM entity_tags WHERE tag_id = ?`

	var count int64
	if err := r.db.QueryRowContext(ctx, query, tagID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count by tag %d: %w", tagID, err)
	}
	return count, nil
}

// queryLinks is a shared row-scanner used by all List* methods.
// It accepts variadic args to allow flexible WHERE clause parameters.
func (r *EntityTagRepository) queryLinks(ctx context.Context, query string, args ...interface{}) ([]*models.EntityTagLink, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query entity_tags: %w", err)
	}
	defer rows.Close()

	links := make([]*models.EntityTagLink, 0)
	for rows.Next() {
		l := &models.EntityTagLink{}
		if err := rows.Scan(&l.ID, &l.EntityType, &l.EntityID, &l.TagID, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan entity_tag link: %w", err)
		}
		links = append(links, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entity_tags: %w", err)
	}
	return links, nil
}
