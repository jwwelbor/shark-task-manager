// Package portfolio provides set-oriented read models for portfolio advice.
package portfolio

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// ChildStateRow is one feature or task in an epic's descendant snapshot.
type ChildStateRow struct {
	EpicID          int64
	EpicKey         string
	EntityType      models.EntityType
	EntityKey       string
	Title           string
	Status          string
	DirectParentKey string
	ProgressPct     *float64
}

// EpicRelationshipRow is one supported stored relationship between epic IDs.
// Endpoint fields are nil when a stored relationship references a missing epic.
type EpicRelationshipRow struct {
	FromEpicID       int64
	FromKey          *string
	FromStatus       *string
	RelationshipType models.EntityRelationshipType
	ToEpicID         int64
	ToKey            *string
	ToStatus         *string
}

// Repository reads the set-oriented data needed to assemble portfolio advice.
type Repository struct {
	db *dbconn.DB
}

// NewRepository creates a portfolio repository.
func NewRepository(db *dbconn.DB) *Repository {
	return &Repository{db: db}
}

// ListChildStates returns all feature and task state grouped by verified epic
// ownership. The direct parent distinguishes epic-owned features from
// feature-owned tasks without deriving ownership from entity keys.
func (r *Repository) ListChildStates(ctx context.Context) ([]ChildStateRow, error) {
	const query = `
		SELECT e.id AS epic_id,
		       e.key AS epic_key,
		       ? AS entity_type,
		       f.key AS entity_key,
		       f.title AS title,
		       f.status AS status,
		       e.key AS direct_parent_key,
		       f.progress_pct AS progress_pct
		FROM features f
		JOIN epics e ON e.id = f.epic_id
		UNION ALL
		SELECT e.id AS epic_id,
		       e.key AS epic_key,
		       ? AS entity_type,
		       t.key AS entity_key,
		       t.title AS title,
		       t.status AS status,
		       f.key AS direct_parent_key,
		       CAST(NULL AS REAL) AS progress_pct
		FROM tasks t
		JOIN features f ON f.id = t.feature_id
		JOIN epics e ON e.id = f.epic_id
		ORDER BY epic_key ASC, entity_type ASC, entity_key ASC
	`

	rows, err := r.db.QueryContext(ctx, query, models.EntityTypeFeature, models.EntityTypeTask)
	if err != nil {
		return nil, fmt.Errorf("query portfolio child states: %w", err)
	}
	defer rows.Close()

	result := make([]ChildStateRow, 0)
	for rows.Next() {
		var row ChildStateRow
		var progress sql.NullFloat64
		if err := rows.Scan(
			&row.EpicID,
			&row.EpicKey,
			&row.EntityType,
			&row.EntityKey,
			&row.Title,
			&row.Status,
			&row.DirectParentKey,
			&progress,
		); err != nil {
			return nil, fmt.Errorf("scan portfolio child state: %w", err)
		}
		if progress.Valid {
			row.ProgressPct = &progress.Float64
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate portfolio child states: %w", err)
	}
	return result, nil
}

// ListEpicRelationships returns supported directed epic relationships. LEFT
// JOINs preserve rows with a missing endpoint so callers can report dangling
// evidence instead of silently discarding it.
func (r *Repository) ListEpicRelationships(ctx context.Context) ([]EpicRelationshipRow, error) {
	const query = `
		SELECT er.from_entity_id,
		       from_epic.key,
		       from_epic.status,
		       er.relationship_type,
		       er.to_entity_id,
		       to_epic.key,
		       to_epic.status
		FROM entity_relationships er
		LEFT JOIN epics from_epic ON from_epic.id = er.from_entity_id
		LEFT JOIN epics to_epic ON to_epic.id = er.to_entity_id
		WHERE er.from_entity_type = ?
		  AND er.to_entity_type = ?
		  AND er.relationship_type IN (?, ?, ?)
		ORDER BY COALESCE(from_epic.key, '') ASC,
		         er.relationship_type ASC,
		         COALESCE(to_epic.key, '') ASC,
		         er.from_entity_id ASC,
		         er.to_entity_id ASC
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		models.EntityTypeEpic,
		models.EntityTypeEpic,
		models.EntityRelDependsOn,
		models.EntityRelBlocks,
		models.EntityRelFollows,
	)
	if err != nil {
		return nil, fmt.Errorf("query portfolio epic relationships: %w", err)
	}
	defer rows.Close()

	result := make([]EpicRelationshipRow, 0)
	for rows.Next() {
		var row EpicRelationshipRow
		var fromKey, fromStatus, toKey, toStatus sql.NullString
		if err := rows.Scan(
			&row.FromEpicID,
			&fromKey,
			&fromStatus,
			&row.RelationshipType,
			&row.ToEpicID,
			&toKey,
			&toStatus,
		); err != nil {
			return nil, fmt.Errorf("scan portfolio epic relationship: %w", err)
		}
		row.FromKey = nullStringPointer(fromKey)
		row.FromStatus = nullStringPointer(fromStatus)
		row.ToKey = nullStringPointer(toKey)
		row.ToStatus = nullStringPointer(toStatus)
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate portfolio epic relationships: %w", err)
	}
	return result, nil
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}
