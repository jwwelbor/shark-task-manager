package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// IdeaRepository handles CRUD operations for ideas
type IdeaRepository struct {
	db *DB
}

// IdeaFilter represents filtering options for listing ideas
type IdeaFilter struct {
	Status *models.IdeaStatus
}

// NewIdeaRepository creates a new IdeaRepository
func NewIdeaRepository(db *DB) *IdeaRepository {
	return &IdeaRepository{db: db}
}

// Create creates a new idea
func (r *IdeaRepository) Create(ctx context.Context, idea *models.Idea) error {
	if err := idea.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO ideas (
			key, title, description, created_date, priority, "order",
			notes, related_docs, dependencies, status
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		idea.Key,
		idea.Title,
		idea.Description,
		idea.CreatedDate,
		idea.Priority,
		idea.Order,
		idea.Notes,
		idea.RelatedDocs,
		idea.Dependencies,
		idea.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to create idea: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	idea.ID = id
	return nil
}

// GetByID retrieves an idea by its ID
func (r *IdeaRepository) GetByID(ctx context.Context, id int64) (*models.Idea, error) {
	query := `
		SELECT id, key, title, description, created_date, priority, "order",
		       notes, related_docs, dependencies, status, created_at, updated_at,
		       converted_to_type, converted_to_key, converted_at
		FROM ideas
		WHERE id = ?
	`

	idea := &models.Idea{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&idea.ID,
		&idea.Key,
		&idea.Title,
		&idea.Description,
		&idea.CreatedDate,
		&idea.Priority,
		&idea.Order,
		&idea.Notes,
		&idea.RelatedDocs,
		&idea.Dependencies,
		&idea.Status,
		&idea.CreatedAt,
		&idea.UpdatedAt,
		&idea.ConvertedToType,
		&idea.ConvertedToKey,
		&idea.ConvertedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("idea not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get idea: %w", err)
	}

	return idea, nil
}

// GetByKey retrieves an idea by its key
func (r *IdeaRepository) GetByKey(ctx context.Context, key string) (*models.Idea, error) {
	query := `
		SELECT id, key, title, description, created_date, priority, "order",
		       notes, related_docs, dependencies, status, created_at, updated_at,
		       converted_to_type, converted_to_key, converted_at
		FROM ideas
		WHERE key = ?
	`

	idea := &models.Idea{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&idea.ID,
		&idea.Key,
		&idea.Title,
		&idea.Description,
		&idea.CreatedDate,
		&idea.Priority,
		&idea.Order,
		&idea.Notes,
		&idea.RelatedDocs,
		&idea.Dependencies,
		&idea.Status,
		&idea.CreatedAt,
		&idea.UpdatedAt,
		&idea.ConvertedToType,
		&idea.ConvertedToKey,
		&idea.ConvertedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("idea not found with key %q", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get idea: %w", err)
	}

	return idea, nil
}

// List retrieves all ideas, optionally filtered by status
func (r *IdeaRepository) List(ctx context.Context, filter *IdeaFilter) ([]*models.Idea, error) {
	query := `
		SELECT id, key, title, description, created_date, priority, "order",
		       notes, related_docs, dependencies, status, created_at, updated_at,
		       converted_to_type, converted_to_key, converted_at
		FROM ideas
	`

	var args []interface{}

	if filter != nil && filter.Status != nil {
		query += " WHERE status = ?"
		args = append(args, *filter.Status)
	}

	query += " ORDER BY created_date DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list ideas: %w", err)
	}
	defer rows.Close()

	var ideas []*models.Idea
	for rows.Next() {
		idea := &models.Idea{}
		err := rows.Scan(
			&idea.ID,
			&idea.Key,
			&idea.Title,
			&idea.Description,
			&idea.CreatedDate,
			&idea.Priority,
			&idea.Order,
			&idea.Notes,
			&idea.RelatedDocs,
			&idea.Dependencies,
			&idea.Status,
			&idea.CreatedAt,
			&idea.UpdatedAt,
			&idea.ConvertedToType,
			&idea.ConvertedToKey,
			&idea.ConvertedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan idea: %w", err)
		}
		ideas = append(ideas, idea)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ideas: %w", err)
	}

	return ideas, nil
}

// Update updates an existing idea
func (r *IdeaRepository) Update(ctx context.Context, idea *models.Idea) error {
	if err := idea.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE ideas
		SET title = ?, description = ?, priority = ?, "order" = ?,
		    notes = ?, related_docs = ?, dependencies = ?, status = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		idea.Title,
		idea.Description,
		idea.Priority,
		idea.Order,
		idea.Notes,
		idea.RelatedDocs,
		idea.Dependencies,
		idea.Status,
		idea.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update idea: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("idea not found with id %d", idea.ID)
	}

	return nil
}

// Delete deletes an idea by its ID
func (r *IdeaRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM ideas WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete idea: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("idea not found with id %d", id)
	}

	return nil
}

// MarkAsConverted updates an idea's conversion tracking fields
func (r *IdeaRepository) MarkAsConverted(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error {
	query := `
		UPDATE ideas
		SET status = 'converted',
		    converted_to_type = ?,
		    converted_to_key = ?,
		    converted_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, convertedToType, convertedToKey, ideaID)
	if err != nil {
		return fmt.Errorf("failed to mark idea as converted: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("idea not found with id %d", ideaID)
	}

	return nil
}
