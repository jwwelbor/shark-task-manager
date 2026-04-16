package entityrel

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// EntityRelationshipRepository provides data access for the entity_relationships table.
type EntityRelationshipRepository struct {
	db *dbconn.DB
}

// NewEntityRelationshipRepository creates a new EntityRelationshipRepository.
func NewEntityRelationshipRepository(db *dbconn.DB) *EntityRelationshipRepository {
	return &EntityRelationshipRepository{db: db}
}

// Create inserts a new polymorphic relationship record.
func (r *EntityRelationshipRepository) Create(
	ctx context.Context,
	rel *models.EntityRelationship,
) error {
	if err := rel.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO entity_relationships
			(from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		rel.FromEntityType, rel.FromEntityID,
		rel.ToEntityType, rel.ToEntityID,
		rel.RelationshipType,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("relationship already exists: %s(%d) -[%s]-> %s(%d)",
				rel.FromEntityType, rel.FromEntityID,
				rel.RelationshipType,
				rel.ToEntityType, rel.ToEntityID)
		}
		return fmt.Errorf("failed to create entity relationship: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	rel.ID = id
	return nil
}

// Delete removes a relationship by primary key.
func (r *EntityRelationshipRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM entity_relationships WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete entity relationship: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("entity relationship not found with id %d", id)
	}
	return nil
}

// DeleteByEntitiesAndType removes a specific directed relationship.
func (r *EntityRelationshipRepository) DeleteByEntitiesAndType(
	ctx context.Context,
	fromType models.EntityType, fromID int64,
	toType models.EntityType, toID int64,
	relType models.EntityRelationshipType,
) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM entity_relationships
		 WHERE from_entity_type = ? AND from_entity_id = ?
		   AND to_entity_type   = ? AND to_entity_id   = ?
		   AND relationship_type = ?`,
		fromType, fromID, toType, toID, relType,
	)
	if err != nil {
		return fmt.Errorf("failed to delete entity relationship: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("relationship not found: %s(%d) -[%s]-> %s(%d)",
			fromType, fromID, relType, toType, toID)
	}
	return nil
}

// GetByEntity returns all relationships (incoming and outgoing) for an entity.
func (r *EntityRelationshipRepository) GetByEntity(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
) ([]*models.EntityRelationship, error) {
	query := `
		SELECT id, from_entity_type, from_entity_id,
		       to_entity_type, to_entity_id, relationship_type, created_at
		FROM entity_relationships
		WHERE (from_entity_type = ? AND from_entity_id = ?)
		   OR (to_entity_type   = ? AND to_entity_id   = ?)
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query,
		entityType, entityID, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity relationships: %w", err)
	}
	defer rows.Close()
	return r.scanRelationships(rows)
}

// GetOutgoing returns outgoing relationships for an entity, optionally filtered by type.
func (r *EntityRelationshipRepository) GetOutgoing(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
	relTypes []models.EntityRelationshipType,
) ([]*models.EntityRelationship, error) {
	args := []interface{}{entityType, entityID}
	filterClause := ""
	if len(relTypes) > 0 {
		placeholders := make([]string, len(relTypes))
		for i, rt := range relTypes {
			placeholders[i] = "?"
			args = append(args, rt)
		}
		filterClause = fmt.Sprintf("AND relationship_type IN (%s)", strings.Join(placeholders, ","))
	}
	query := fmt.Sprintf(`
		SELECT id, from_entity_type, from_entity_id,
		       to_entity_type, to_entity_id, relationship_type, created_at
		FROM entity_relationships
		WHERE from_entity_type = ? AND from_entity_id = ? %s
		ORDER BY created_at ASC`, filterClause)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query outgoing relationships: %w", err)
	}
	defer rows.Close()
	return r.scanRelationships(rows)
}

// GetIncoming returns incoming relationships for an entity, optionally filtered by type.
func (r *EntityRelationshipRepository) GetIncoming(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
	relTypes []models.EntityRelationshipType,
) ([]*models.EntityRelationship, error) {
	args := []interface{}{entityType, entityID}
	filterClause := ""
	if len(relTypes) > 0 {
		placeholders := make([]string, len(relTypes))
		for i, rt := range relTypes {
			placeholders[i] = "?"
			args = append(args, rt)
		}
		filterClause = fmt.Sprintf("AND relationship_type IN (%s)", strings.Join(placeholders, ","))
	}
	query := fmt.Sprintf(`
		SELECT id, from_entity_type, from_entity_id,
		       to_entity_type, to_entity_id, relationship_type, created_at
		FROM entity_relationships
		WHERE to_entity_type = ? AND to_entity_id = ? %s
		ORDER BY created_at ASC`, filterClause)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query incoming relationships: %w", err)
	}
	defer rows.Close()
	return r.scanRelationships(rows)
}

// scanRelationships is a helper that scans query rows into EntityRelationship slices.
func (r *EntityRelationshipRepository) scanRelationships(
	rows *sql.Rows,
) ([]*models.EntityRelationship, error) {
	var rels []*models.EntityRelationship
	for rows.Next() {
		rel := &models.EntityRelationship{}
		err := rows.Scan(
			&rel.ID,
			&rel.FromEntityType, &rel.FromEntityID,
			&rel.ToEntityType, &rel.ToEntityID,
			&rel.RelationshipType,
			&rel.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity relationship: %w", err)
		}
		rels = append(rels, rel)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entity relationships: %w", err)
	}
	return rels, nil
}

// EntityRelTaskKeyAdapter implements config.TaskRelationshipRepository using
// entity_relationships instead of the legacy task_relationships table.
// It satisfies the ListRelatedTaskKeys interface used by template_helpers.go.
type EntityRelTaskKeyAdapter struct {
	db *dbconn.DB
}

// NewEntityRelTaskKeyAdapter creates a new adapter that queries entity_relationships
// for task-to-task relationships.
func NewEntityRelTaskKeyAdapter(db *dbconn.DB) *EntityRelTaskKeyAdapter {
	return &EntityRelTaskKeyAdapter{db: db}
}

// ListRelatedTaskKeys returns all task keys related to a given task (bidirectional)
// by querying the entity_relationships table for task-to-task relationships.
func (a *EntityRelTaskKeyAdapter) ListRelatedTaskKeys(ctx context.Context, taskID int64) ([]string, error) {
	query := `
		SELECT DISTINCT t.key
		FROM entity_relationships er
		JOIN tasks t ON (
			(er.from_entity_type = 'task' AND er.from_entity_id = ? AND er.to_entity_type = 'task' AND er.to_entity_id = t.id AND t.id != ?) OR
			(er.to_entity_type = 'task' AND er.to_entity_id = ? AND er.from_entity_type = 'task' AND er.from_entity_id = t.id AND t.id != ?)
		)
		ORDER BY t.key ASC
	`
	rows, err := a.db.QueryContext(ctx, query, taskID, taskID, taskID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list related task keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan task key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task keys: %w", err)
	}
	if keys == nil {
		keys = []string{}
	}
	return keys, nil
}

// EntityRelFeatureKeyAdapter implements config.FeatureRelationshipRepository using
// entity_relationships instead of the legacy feature_relationships table.
// It satisfies the ListRelatedFeatureKeys interface used by template_helpers.go.
type EntityRelFeatureKeyAdapter struct {
	db *dbconn.DB
}

// NewEntityRelFeatureKeyAdapter creates a new adapter that queries entity_relationships
// for feature-to-feature relationships.
func NewEntityRelFeatureKeyAdapter(db *dbconn.DB) *EntityRelFeatureKeyAdapter {
	return &EntityRelFeatureKeyAdapter{db: db}
}

// ListRelatedFeatures returns all feature keys related to a given feature (bidirectional)
// by querying the entity_relationships table for feature-to-feature relationships.
// This satisfies config.FeatureRelationshipRepository interface (ListRelatedFeatures).
func (a *EntityRelFeatureKeyAdapter) ListRelatedFeatures(ctx context.Context, featureID int64) ([]string, error) {
	return a.ListRelatedFeatureKeys(ctx, featureID)
}

// ListRelatedFeatureKeys returns all feature keys related to a given feature (bidirectional)
// by querying the entity_relationships table for feature-to-feature relationships.
func (a *EntityRelFeatureKeyAdapter) ListRelatedFeatureKeys(ctx context.Context, featureID int64) ([]string, error) {
	query := `
		SELECT DISTINCT f.key
		FROM entity_relationships er
		JOIN features f ON (
			(er.from_entity_type = 'feature' AND er.from_entity_id = ? AND er.to_entity_type = 'feature' AND er.to_entity_id = f.id AND f.id != ?) OR
			(er.to_entity_type = 'feature' AND er.to_entity_id = ? AND er.from_entity_type = 'feature' AND er.from_entity_id = f.id AND f.id != ?)
		)
		ORDER BY f.key ASC
	`
	rows, err := a.db.QueryContext(ctx, query, featureID, featureID, featureID, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to list related feature keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan feature key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feature keys: %w", err)
	}
	if keys == nil {
		keys = []string{}
	}
	return keys, nil
}

// EntityRelEpicKeyAdapter implements config.EpicRelationshipRepository using
// entity_relationships instead of the legacy epic_relationships table.
// It satisfies the ListRelatedEpicKeys interface used by template_helpers.go.
type EntityRelEpicKeyAdapter struct {
	db *dbconn.DB
}

// NewEntityRelEpicKeyAdapter creates a new adapter that queries entity_relationships
// for epic-to-epic relationships.
func NewEntityRelEpicKeyAdapter(db *dbconn.DB) *EntityRelEpicKeyAdapter {
	return &EntityRelEpicKeyAdapter{db: db}
}

// ListRelatedEpics returns all epic keys related to a given epic (bidirectional)
// by querying the entity_relationships table for epic-to-epic relationships.
// This satisfies config.EpicRelationshipRepository interface (ListRelatedEpics).
func (a *EntityRelEpicKeyAdapter) ListRelatedEpics(ctx context.Context, epicID int64) ([]string, error) {
	return a.ListRelatedEpicKeys(ctx, epicID)
}

// ListRelatedEpicKeys returns all epic keys related to a given epic (bidirectional)
// by querying the entity_relationships table for epic-to-epic relationships.
func (a *EntityRelEpicKeyAdapter) ListRelatedEpicKeys(ctx context.Context, epicID int64) ([]string, error) {
	query := `
		SELECT DISTINCT e.key
		FROM entity_relationships er
		JOIN epics e ON (
			(er.from_entity_type = 'epic' AND er.from_entity_id = ? AND er.to_entity_type = 'epic' AND er.to_entity_id = e.id AND e.id != ?) OR
			(er.to_entity_type = 'epic' AND er.to_entity_id = ? AND er.from_entity_type = 'epic' AND er.from_entity_id = e.id AND e.id != ?)
		)
		ORDER BY e.key ASC
	`
	rows, err := a.db.QueryContext(ctx, query, epicID, epicID, epicID, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to list related epic keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan epic key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating epic keys: %w", err)
	}
	if keys == nil {
		keys = []string{}
	}
	return keys, nil
}
