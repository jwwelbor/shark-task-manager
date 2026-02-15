package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// FeatureRelationshipRepository handles CRUD operations for feature relationships
type FeatureRelationshipRepository struct {
	db *DB
}

// NewFeatureRelationshipRepository creates a new FeatureRelationshipRepository
func NewFeatureRelationshipRepository(db *DB) *FeatureRelationshipRepository {
	return &FeatureRelationshipRepository{db: db}
}

// Create creates a new feature relationship
func (r *FeatureRelationshipRepository) Create(ctx context.Context, rel *models.FeatureRelationship) error {
	if err := rel.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO feature_relationships (
			from_feature_id, to_feature_id, relationship_type
		)
		VALUES (?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		rel.FromFeatureID,
		rel.ToFeatureID,
		rel.RelationshipType,
	)
	if err != nil {
		// Check for UNIQUE constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("relationship already exists: from_feature_id=%d to_feature_id=%d type=%s",
				rel.FromFeatureID, rel.ToFeatureID, rel.RelationshipType)
		}
		return fmt.Errorf("failed to create feature relationship: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	rel.ID = id
	return nil
}

// GetByID retrieves a feature relationship by its ID
func (r *FeatureRelationshipRepository) GetByID(ctx context.Context, id int64) (*models.FeatureRelationship, error) {
	query := `
		SELECT id, from_feature_id, to_feature_id, relationship_type, created_at
		FROM feature_relationships
		WHERE id = ?
	`

	rel := &models.FeatureRelationship{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rel.ID,
		&rel.FromFeatureID,
		&rel.ToFeatureID,
		&rel.RelationshipType,
		&rel.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("feature relationship not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feature relationship: %w", err)
	}

	return rel, nil
}

// ListRelatedFeatures retrieves all features related to a given feature
// Returns both outbound (from_feature_id = featureID) and inbound (to_feature_id = featureID) relationships
func (r *FeatureRelationshipRepository) ListRelatedFeatures(ctx context.Context, featureID int64) ([]*models.FeatureRelationship, error) {
	query := `
		SELECT id, from_feature_id, to_feature_id, relationship_type, created_at
		FROM feature_relationships
		WHERE from_feature_id = ? OR to_feature_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, featureID, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to query feature relationships: %w", err)
	}
	defer rows.Close()

	var relationships []*models.FeatureRelationship
	for rows.Next() {
		rel := &models.FeatureRelationship{}
		if err := rows.Scan(
			&rel.ID,
			&rel.FromFeatureID,
			&rel.ToFeatureID,
			&rel.RelationshipType,
			&rel.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan feature relationship: %w", err)
		}
		relationships = append(relationships, rel)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feature relationships: %w", err)
	}

	return relationships, nil
}

// GetRelatedFeatureKeys returns a list of feature keys related to the given feature
// Filters to only the actual related features (not self-references)
func (r *FeatureRelationshipRepository) GetRelatedFeatureKeys(ctx context.Context, featureID int64) ([]string, error) {
	// Get the feature's key first
	var selfKey string
	err := r.db.QueryRowContext(ctx, "SELECT key FROM features WHERE id = ?", featureID).Scan(&selfKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature key: %w", err)
	}

	query := `
		SELECT DISTINCT f.key
		FROM feature_relationships fr
		LEFT JOIN features f ON (
			(fr.from_feature_id = ? AND f.id = fr.to_feature_id) OR
			(fr.to_feature_id = ? AND f.id = fr.from_feature_id)
		)
		WHERE (fr.from_feature_id = ? OR fr.to_feature_id = ?)
		  AND f.id IS NOT NULL
		ORDER BY f.key ASC
	`

	rows, err := r.db.QueryContext(ctx, query, featureID, featureID, featureID, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to query related feature keys: %w", err)
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

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feature keys: %w", err)
	}

	return keys, nil
}

// Delete removes a feature relationship by its ID
func (r *FeatureRelationshipRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM feature_relationships WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete feature relationship: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feature relationship not found with id %d", id)
	}

	return nil
}

// DeleteByFeatures removes all relationships between two specific features
func (r *FeatureRelationshipRepository) DeleteByFeatures(ctx context.Context, fromFeatureID, toFeatureID int64) error {
	query := `
		DELETE FROM feature_relationships
		WHERE (from_feature_id = ? AND to_feature_id = ?)
		   OR (from_feature_id = ? AND to_feature_id = ?)
	`
	_, err := r.db.ExecContext(ctx, query, fromFeatureID, toFeatureID, toFeatureID, fromFeatureID)
	if err != nil {
		return fmt.Errorf("failed to delete feature relationships: %w", err)
	}
	return nil
}
