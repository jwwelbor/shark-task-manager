package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// BugRepository handles CRUD operations for bugs.
type BugRepository struct {
	db *DB
}

// NewBugRepository creates a new BugRepository.
func NewBugRepository(db *DB) *BugRepository {
	return &BugRepository{db: db}
}

// bugScanColumns is the ordered list of columns for scanning a Bug row.
const bugSelectColumns = `id, key, title, slug, description, status, severity,
	linked_entity_type, linked_entity_key, context_data, file_path,
	created_at, updated_at`

// scanBug scans a single Bug row from the given scanner.
func scanBug(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.Bug, error) {
	bug := &models.Bug{}
	err := scanner.Scan(
		&bug.ID,
		&bug.Key,
		&bug.Title,
		&bug.Slug,
		&bug.Description,
		&bug.Status,
		&bug.Severity,
		&bug.LinkedEntityType,
		&bug.LinkedEntityKey,
		&bug.ContextData,
		&bug.FilePath,
		&bug.CreatedAt,
		&bug.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return bug, nil
}

// Create creates a new bug record.
func (r *BugRepository) Create(ctx context.Context, bug *models.Bug) error {
	if err := bug.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO bugs (
			key, title, slug, description, status, severity,
			linked_entity_type, linked_entity_key, context_data, file_path
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		bug.Key,
		bug.Title,
		bug.Slug,
		bug.Description,
		bug.Status,
		bug.Severity,
		bug.LinkedEntityType,
		bug.LinkedEntityKey,
		bug.ContextData,
		bug.FilePath,
	)
	if err != nil {
		return fmt.Errorf("failed to create bug: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	bug.ID = id
	return nil
}

// GetByKey retrieves a bug by its key (case-insensitive).
func (r *BugRepository) GetByKey(ctx context.Context, key string) (*models.Bug, error) {
	query := fmt.Sprintf(`SELECT %s FROM bugs WHERE UPPER(key) = UPPER(?)`, bugSelectColumns)

	bug, err := scanBug(r.db.QueryRowContext(ctx, query, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("bug not found with key %q", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bug: %w", err)
	}

	return bug, nil
}

// GetByID retrieves a bug by its database ID.
func (r *BugRepository) GetByID(ctx context.Context, id int64) (*models.Bug, error) {
	query := fmt.Sprintf(`SELECT %s FROM bugs WHERE id = ?`, bugSelectColumns)

	bug, err := scanBug(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("bug not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bug: %w", err)
	}

	return bug, nil
}

// Update updates an existing bug record.
func (r *BugRepository) Update(ctx context.Context, bug *models.Bug) error {
	if err := bug.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE bugs
		SET title = ?, slug = ?, description = ?, status = ?, severity = ?,
			linked_entity_type = ?, linked_entity_key = ?,
			context_data = ?, file_path = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		bug.Title,
		bug.Slug,
		bug.Description,
		bug.Status,
		bug.Severity,
		bug.LinkedEntityType,
		bug.LinkedEntityKey,
		bug.ContextData,
		bug.FilePath,
		bug.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update bug: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("bug not found with id %d", bug.ID)
	}

	return nil
}

// Delete deletes a bug by its database ID.
func (r *BugRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM bugs WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete bug: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("bug not found with id %d", id)
	}

	return nil
}

// UpdateStatus updates only the status field of a bug.
func (r *BugRepository) UpdateStatus(ctx context.Context, id int64, status models.BugStatus) error {
	query := `UPDATE bugs SET status = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update bug status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("bug not found with id %d", id)
	}

	return nil
}

// GetNextKey returns the next available bug key (e.g., B001, B002, ...).
func (r *BugRepository) GetNextKey(ctx context.Context) (string, error) {
	query := `SELECT COALESCE(MAX(CAST(SUBSTR(key, 2) AS INTEGER)), 0) FROM bugs`

	var maxNum int
	err := r.db.QueryRowContext(ctx, query).Scan(&maxNum)
	if err != nil {
		return "", fmt.Errorf("failed to get next bug key: %w", err)
	}

	nextKey := fmt.Sprintf("B%03d", maxNum+1)
	return nextKey, nil
}

// BugListFilters defines filter options for listing bugs.
type BugListFilters struct {
	Status   *models.BugStatus
	Severity *models.BugSeverity
}

// List retrieves all bugs, optionally filtered.
func (r *BugRepository) List(ctx context.Context, filters *BugListFilters) ([]*models.Bug, error) {
	query := fmt.Sprintf(`SELECT %s FROM bugs`, bugSelectColumns)

	var conditions []string
	var args []interface{}

	if filters != nil {
		if filters.Status != nil {
			conditions = append(conditions, "status = ?")
			args = append(args, *filters.Status)
		}
		if filters.Severity != nil {
			conditions = append(conditions, "severity = ?")
			args = append(args, *filters.Severity)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list bugs: %w", err)
	}
	defer rows.Close()

	var bugs []*models.Bug
	for rows.Next() {
		bug, err := scanBug(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bug: %w", err)
		}
		bugs = append(bugs, bug)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bugs: %w", err)
	}

	return bugs, nil
}

// CountByStatus returns the count of bugs grouped by status.
func (r *BugRepository) CountByStatus(ctx context.Context) (map[string]int, error) {
	query := `SELECT status, COUNT(*) FROM bugs GROUP BY status`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count bugs by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[status] = count
	}

	return counts, rows.Err()
}

// CountBySeverity returns the count of bugs grouped by severity.
func (r *BugRepository) CountBySeverity(ctx context.Context) (map[string]int, error) {
	query := `SELECT severity, COUNT(*) FROM bugs GROUP BY severity`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count bugs by severity: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[severity] = count
	}

	return counts, rows.Err()
}
