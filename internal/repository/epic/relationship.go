package epic

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// EpicRelationshipRepository handles CRUD operations for epic relationships
type EpicRelationshipRepository struct {
	db *dbconn.DB
}

// NewEpicRelationshipRepository creates a new EpicRelationshipRepository
func NewEpicRelationshipRepository(db *dbconn.DB) *EpicRelationshipRepository {
	return &EpicRelationshipRepository{db: db}
}

// Create creates a new epic relationship
func (r *EpicRelationshipRepository) Create(ctx context.Context, rel *models.EpicRelationship) error {
	if err := rel.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO epic_relationships (
			from_epic_id, to_epic_id, relationship_type
		)
		VALUES (?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		rel.FromEpicID,
		rel.ToEpicID,
		rel.RelationshipType,
	)
	if err != nil {
		// Check for UNIQUE constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("relationship already exists: from_epic_id=%d to_epic_id=%d type=%s",
				rel.FromEpicID, rel.ToEpicID, rel.RelationshipType)
		}
		return fmt.Errorf("failed to create epic relationship: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	rel.ID = id
	return nil
}

// GetByID retrieves an epic relationship by its ID
func (r *EpicRelationshipRepository) GetByID(ctx context.Context, id int64) (*models.EpicRelationship, error) {
	query := `
		SELECT id, from_epic_id, to_epic_id, relationship_type, created_at
		FROM epic_relationships
		WHERE id = ?
	`

	rel := &models.EpicRelationship{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rel.ID,
		&rel.FromEpicID,
		&rel.ToEpicID,
		&rel.RelationshipType,
		&rel.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("epic relationship not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get epic relationship: %w", err)
	}

	return rel, nil
}

// ListRelatedEpics retrieves all epics related to a given epic
// Returns both outbound (from_epic_id = epicID) and inbound (to_epic_id = epicID) relationships
func (r *EpicRelationshipRepository) ListRelatedEpics(ctx context.Context, epicID int64) ([]*models.EpicRelationship, error) {
	query := `
		SELECT id, from_epic_id, to_epic_id, relationship_type, created_at
		FROM epic_relationships
		WHERE from_epic_id = ? OR to_epic_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, epicID, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to query epic relationships: %w", err)
	}
	defer rows.Close()

	var relationships []*models.EpicRelationship
	for rows.Next() {
		rel := &models.EpicRelationship{}
		if err := rows.Scan(
			&rel.ID,
			&rel.FromEpicID,
			&rel.ToEpicID,
			&rel.RelationshipType,
			&rel.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan epic relationship: %w", err)
		}
		relationships = append(relationships, rel)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating epic relationships: %w", err)
	}

	return relationships, nil
}

// GetRelatedEpicKeys returns a list of epic keys related to the given epic
// Filters to only the actual related epics (not self-references)
func (r *EpicRelationshipRepository) GetRelatedEpicKeys(ctx context.Context, epicID int64) ([]string, error) {
	// Get the epic's key first
	var selfKey string
	err := r.db.QueryRowContext(ctx, "SELECT key FROM epics WHERE id = ?", epicID).Scan(&selfKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic key: %w", err)
	}

	query := `
		SELECT DISTINCT e.key
		FROM epic_relationships er
		LEFT JOIN epics e ON (
			(er.from_epic_id = ? AND e.id = er.to_epic_id) OR
			(er.to_epic_id = ? AND e.id = er.from_epic_id)
		)
		WHERE (er.from_epic_id = ? OR er.to_epic_id = ?)
		  AND e.id IS NOT NULL
		ORDER BY e.key ASC
	`

	rows, err := r.db.QueryContext(ctx, query, epicID, epicID, epicID, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to query related epic keys: %w", err)
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

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating epic keys: %w", err)
	}

	return keys, nil
}

// Delete removes an epic relationship by its ID
func (r *EpicRelationshipRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM epic_relationships WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete epic relationship: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("epic relationship not found with id %d", id)
	}

	return nil
}

// DeleteByEpics removes all relationships between two specific epics
func (r *EpicRelationshipRepository) DeleteByEpics(ctx context.Context, fromEpicID, toEpicID int64) error {
	query := `
		DELETE FROM epic_relationships
		WHERE (from_epic_id = ? AND to_epic_id = ?)
		   OR (from_epic_id = ? AND to_epic_id = ?)
	`
	_, err := r.db.ExecContext(ctx, query, fromEpicID, toEpicID, toEpicID, fromEpicID)
	if err != nil {
		return fmt.Errorf("failed to delete epic relationships: %w", err)
	}
	return nil
}
