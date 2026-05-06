package sprint

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// SprintRepository handles CRUD operations for sprints.
type SprintRepository struct {
	db *dbconn.DB
}

// NewSprintRepository creates a new SprintRepository.
func NewSprintRepository(db *dbconn.DB) *SprintRepository {
	return &SprintRepository{db: db}
}

// sprintSelectColumns is the ordered list of columns for scanning a Sprint row.
const sprintSelectColumns = `id, key, name, goal, start_date, end_date, status, slug, file_path, created_at, updated_at`

// scanSprint scans a single Sprint row from the given scanner.
func scanSprint(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.Sprint, error) {
	sprint := &models.Sprint{}
	err := scanner.Scan(
		&sprint.ID,
		&sprint.Key,
		&sprint.Name,
		&sprint.Goal,
		&sprint.StartDate,
		&sprint.EndDate,
		&sprint.Status,
		&sprint.Slug,
		&sprint.FilePath,
		&sprint.CreatedAt,
		&sprint.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return sprint, nil
}

// Create creates a new sprint record.
func (r *SprintRepository) Create(ctx context.Context, sprint *models.Sprint) error {
	if err := sprint.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO sprints (
			key, name, goal, start_date, end_date, status, slug, file_path
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		sprint.Key,
		sprint.Name,
		sprint.Goal,
		sprint.StartDate,
		sprint.EndDate,
		sprint.Status,
		sprint.Slug,
		sprint.FilePath,
	)
	if err != nil {
		return fmt.Errorf("failed to create sprint: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	sprint.ID = id
	return nil
}

// GetByKey retrieves a sprint by its key (case-insensitive).
func (r *SprintRepository) GetByKey(ctx context.Context, key string) (*models.Sprint, error) {
	query := fmt.Sprintf(`SELECT %s FROM sprints WHERE UPPER(key) = UPPER(?)`, sprintSelectColumns)

	sprint, err := scanSprint(r.db.QueryRowContext(ctx, query, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sprint not found with key %q", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint: %w", err)
	}

	return sprint, nil
}

// GetByID retrieves a sprint by its database ID.
func (r *SprintRepository) GetByID(ctx context.Context, id int64) (*models.Sprint, error) {
	query := fmt.Sprintf(`SELECT %s FROM sprints WHERE id = ?`, sprintSelectColumns)

	sprint, err := scanSprint(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sprint not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint: %w", err)
	}

	return sprint, nil
}

// Update updates an existing sprint record.
func (r *SprintRepository) Update(ctx context.Context, sprint *models.Sprint) error {
	if err := sprint.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE sprints
		SET name = ?, goal = ?, start_date = ?, end_date = ?,
			status = ?, slug = ?, file_path = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		sprint.Name,
		sprint.Goal,
		sprint.StartDate,
		sprint.EndDate,
		sprint.Status,
		sprint.Slug,
		sprint.FilePath,
		sprint.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update sprint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("sprint not found with id %d", sprint.ID)
	}

	return nil
}

// Delete deletes a sprint by its database ID.
func (r *SprintRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM sprints WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete sprint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("sprint not found with id %d", id)
	}

	return nil
}

// UpdateStatus updates only the status field of a sprint (atomic operation).
func (r *SprintRepository) UpdateStatus(ctx context.Context, id int64, status models.SprintStatus) error {
	query := `UPDATE sprints SET status = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update sprint status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("sprint not found with id %d", id)
	}

	return nil
}

// GetNextKey returns the next available sprint key (e.g., S001, S002, ...).
func (r *SprintRepository) GetNextKey(ctx context.Context) (string, error) {
	query := `SELECT COALESCE(MAX(CAST(SUBSTR(key, 2) AS INTEGER)), 0) FROM sprints`

	var maxNum int
	err := r.db.QueryRowContext(ctx, query).Scan(&maxNum)
	if err != nil {
		return "", fmt.Errorf("failed to get next sprint key: %w", err)
	}

	nextKey := fmt.Sprintf("S%03d", maxNum+1)
	return nextKey, nil
}

// SprintListFilters defines filter options for listing sprints.
type SprintListFilters struct {
	Status *models.SprintStatus
}

// List retrieves all sprints, optionally filtered.
func (r *SprintRepository) List(ctx context.Context, filters *SprintListFilters) ([]*models.Sprint, error) {
	query := fmt.Sprintf(`SELECT %s FROM sprints`, sprintSelectColumns)

	var conditions []string
	var args []interface{}

	if filters != nil {
		if filters.Status != nil {
			conditions = append(conditions, "status = ?")
			args = append(args, *filters.Status)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sprints: %w", err)
	}
	defer rows.Close()

	var sprints []*models.Sprint
	for rows.Next() {
		sprint, err := scanSprint(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sprint: %w", err)
		}
		sprints = append(sprints, sprint)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sprints: %w", err)
	}

	return sprints, nil
}
