package techdebt

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
)

// TechDebtFilters defines filter options for listing tech-debt items.
type TechDebtFilters struct {
	Status          *string
	Category        *models.TechDebtCategory
	Severity        *models.TechDebtSeverity
	IncludeTerminal bool // if false, excludes resolved + wont_fix + cancelled
}

// TechDebtRepository handles CRUD operations for tech-debt entities.
type TechDebtRepository struct {
	db *dbconn.DB
}

// NewTechDebtRepository creates a new TechDebtRepository.
func NewTechDebtRepository(db *dbconn.DB) *TechDebtRepository {
	return &TechDebtRepository{db: db}
}

// techDebtSelectColumns is the ordered list of columns for scanning a TechDebt row.
const techDebtSelectColumns = `id, key, title, slug, description, status, category, severity,
	effort_estimate, context_data, file_path, size, created_at, updated_at`

// scanTechDebt scans a single TechDebt row from the given scanner.
func scanTechDebt(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.TechDebt, error) {
	td := &models.TechDebt{}
	err := scanner.Scan(
		&td.ID,
		&td.Key,
		&td.Title,
		&td.Slug,
		&td.Description,
		&td.Status,
		&td.Category,
		&td.Severity,
		&td.EffortEstimate,
		&td.ContextData,
		&td.FilePath,
		&td.Size,
		&td.CreatedAt,
		&td.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return td, nil
}

// Create creates a new tech-debt record. If Key is empty, it auto-generates the next key.
// Slug is auto-generated from the title.
func (r *TechDebtRepository) Create(ctx context.Context, td *models.TechDebt) error {
	// Auto-generate key if not set
	if td.Key == "" {
		nextKey, err := r.GenerateNextKey(ctx)
		if err != nil {
			return fmt.Errorf("failed to generate next key: %w", err)
		}
		td.Key = nextKey
	}

	// Auto-generate slug from title
	if td.Slug == nil || *td.Slug == "" {
		slug := utils.GenerateSlug(td.Title)
		td.Slug = &slug
	}

	if err := td.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO tech_debts (
			key, title, slug, description, status, category, severity,
			effort_estimate, context_data, file_path, size
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		td.Key,
		td.Title,
		td.Slug,
		td.Description,
		td.Status,
		td.Category,
		td.Severity,
		td.EffortEstimate,
		td.ContextData,
		td.FilePath,
		td.Size,
	)
	if err != nil {
		return fmt.Errorf("failed to create tech-debt: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	td.ID = id
	return nil
}

// GetByKey retrieves a tech-debt item by its key (case-insensitive).
func (r *TechDebtRepository) GetByKey(ctx context.Context, key string) (*models.TechDebt, error) {
	normalizedKey := strings.ToUpper(strings.TrimSpace(key))

	// Direct comparison: keys are stored uppercase (model validation enforces ^TD-\d{3}$),
	// and normalizedKey is already uppercased. This lets the query use the key index.
	query := fmt.Sprintf(`SELECT %s FROM tech_debts WHERE key = ?`, techDebtSelectColumns)

	td, err := scanTechDebt(r.db.QueryRowContext(ctx, query, normalizedKey))
	if err == nil {
		return td, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get tech-debt: %w", err)
	}

	// If not found and input might contain a slug suffix (e.g., TD-001-some-slug),
	// try slug-based lookup
	parts := strings.SplitN(normalizedKey, "-", 3)
	if len(parts) >= 3 {
		numericKey := parts[0] + "-" + parts[1] // TD-001
		slug := strings.ToLower(key[len(numericKey)+1:])

		slugQuery := fmt.Sprintf(`SELECT %s FROM tech_debts WHERE key = ? AND slug = ?`, techDebtSelectColumns)
		td, scanErr := scanTechDebt(r.db.QueryRowContext(ctx, slugQuery, numericKey, slug))
		if scanErr == nil {
			return td, nil
		}
	}

	return nil, fmt.Errorf("tech-debt not found with key %q", key)
}

// GetByID retrieves a tech-debt item by its database ID.
func (r *TechDebtRepository) GetByID(ctx context.Context, id int64) (*models.TechDebt, error) {
	query := fmt.Sprintf(`SELECT %s FROM tech_debts WHERE id = ?`, techDebtSelectColumns)

	td, err := scanTechDebt(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("tech-debt not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tech-debt: %w", err)
	}

	return td, nil
}

// Update updates an existing tech-debt record (title, description, category, severity,
// effort_estimate, size).
func (r *TechDebtRepository) Update(ctx context.Context, td *models.TechDebt) error {
	if err := td.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE tech_debts
		SET title = ?, slug = ?, description = ?, category = ?, severity = ?,
			effort_estimate = ?, context_data = ?, file_path = ?, size = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		td.Title,
		td.Slug,
		td.Description,
		td.Category,
		td.Severity,
		td.EffortEstimate,
		td.ContextData,
		td.FilePath,
		td.Size,
		td.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update tech-debt: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tech-debt not found with id %d", td.ID)
	}

	return nil
}

// Delete deletes a tech-debt item by its database ID.
func (r *TechDebtRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM tech_debts WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tech-debt: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tech-debt not found with id %d", id)
	}

	return nil
}

// UpdateStatus updates only the status field of a tech-debt item.
//
// Status-change notes are persisted at the entity history layer (via EntityService),
// not on the row itself, so this function intentionally takes no notes parameter —
// matching the BugRepository.UpdateStatus signature.
func (r *TechDebtRepository) UpdateStatus(ctx context.Context, id int64, status models.TechDebtStatus) error {
	query := `UPDATE tech_debts SET status = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update tech-debt status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tech-debt not found with id %d", id)
	}

	return nil
}

// GetRecent returns the most recently created tech-debt items, ordered by created_at DESC.
// limit must be positive; the caller (service) is responsible for bounds-checking.
// Returns an empty (non-nil) slice if no rows exist.
func (r *TechDebtRepository) GetRecent(ctx context.Context, limit int) ([]*models.TechDebt, error) {
	query := fmt.Sprintf(`SELECT %s FROM tech_debts ORDER BY created_at DESC LIMIT ?`, techDebtSelectColumns)

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent tech-debts: %w", err)
	}
	defer rows.Close()

	items := []*models.TechDebt{}
	for rows.Next() {
		td, scanErr := scanTechDebt(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan tech-debt: %w", scanErr)
		}
		items = append(items, td)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tech-debts: %w", err)
	}

	return items, nil
}

// List retrieves all tech-debt items ordered by key ascending.
func (r *TechDebtRepository) List(ctx context.Context) ([]*models.TechDebt, error) {
	query := fmt.Sprintf(`SELECT %s FROM tech_debts ORDER BY key ASC`, techDebtSelectColumns)

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tech-debts: %w", err)
	}
	defer rows.Close()

	var items []*models.TechDebt
	for rows.Next() {
		td, scanErr := scanTechDebt(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan tech-debt: %w", scanErr)
		}
		items = append(items, td)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tech-debts: %w", err)
	}

	return items, nil
}

// ListWithFilters retrieves tech-debt items matching the provided filters.
func (r *TechDebtRepository) ListWithFilters(ctx context.Context, filters TechDebtFilters) ([]*models.TechDebt, error) {
	query := fmt.Sprintf(`SELECT %s FROM tech_debts`, techDebtSelectColumns)

	var conditions []string
	var args []interface{}

	if filters.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *filters.Status)
	}
	if filters.Category != nil {
		conditions = append(conditions, "category = ?")
		args = append(args, *filters.Category)
	}
	if filters.Severity != nil {
		conditions = append(conditions, "severity = ?")
		args = append(args, *filters.Severity)
	}

	// Hide terminal statuses by default unless explicitly requested or a specific
	// status filter is provided. Matches BugRepository.List behavior.
	if !filters.IncludeTerminal && filters.Status == nil {
		conditions = append(conditions, "status NOT IN ('resolved', 'wont_fix', 'cancelled')")
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY key ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tech-debts with filters: %w", err)
	}
	defer rows.Close()

	var items []*models.TechDebt
	for rows.Next() {
		td, scanErr := scanTechDebt(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan tech-debt: %w", scanErr)
		}
		items = append(items, td)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tech-debts: %w", err)
	}

	return items, nil
}

// ListByStatus retrieves tech-debt items with the specified status.
func (r *TechDebtRepository) ListByStatus(ctx context.Context, status string) ([]*models.TechDebt, error) {
	s := status
	return r.ListWithFilters(ctx, TechDebtFilters{Status: &s})
}

// ListByCategory retrieves tech-debt items with the specified category.
func (r *TechDebtRepository) ListByCategory(ctx context.Context, category string) ([]*models.TechDebt, error) {
	c := models.TechDebtCategory(category)
	return r.ListWithFilters(ctx, TechDebtFilters{Category: &c})
}

// ListBySeverity retrieves tech-debt items with the specified severity.
func (r *TechDebtRepository) ListBySeverity(ctx context.Context, severity string) ([]*models.TechDebt, error) {
	s := models.TechDebtSeverity(severity)
	return r.ListWithFilters(ctx, TechDebtFilters{Severity: &s})
}

// Count returns the total number of tech-debt items.
func (r *TechDebtRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM tech_debts`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tech-debts: %w", err)
	}

	return count, nil
}

// CountByStatus returns the count of tech-debt items grouped by status.
func (r *TechDebtRepository) CountByStatus(ctx context.Context) (map[string]int, error) {
	query := `SELECT status, COUNT(*) FROM tech_debts GROUP BY status`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count tech-debts by status: %w", err)
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

// CountByCategory returns the count of tech-debt items grouped by category.
func (r *TechDebtRepository) CountByCategory(ctx context.Context) (map[string]int, error) {
	query := `SELECT category, COUNT(*) FROM tech_debts GROUP BY category`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count tech-debts by category: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[category] = count
	}

	return counts, rows.Err()
}

// GenerateNextKey returns the next available tech-debt key (TD-001, TD-002, ...).
func (r *TechDebtRepository) GenerateNextKey(ctx context.Context) (string, error) {
	query := `SELECT COALESCE(MAX(CAST(SUBSTR(key, 4) AS INTEGER)), 0) FROM tech_debts`

	var maxNum int
	err := r.db.QueryRowContext(ctx, query).Scan(&maxNum)
	if err != nil {
		return "", fmt.Errorf("failed to get next tech-debt key: %w", err)
	}

	nextKey := fmt.Sprintf("TD-%03d", maxNum+1)
	return nextKey, nil
}

// GetContextData retrieves the context data JSON string for a tech-debt item by its ID.
func (r *TechDebtRepository) GetContextData(ctx context.Context, id int64) (*string, error) {
	query := `SELECT context_data FROM tech_debts WHERE id = ?`
	var contextData *string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&contextData)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tech-debt not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tech-debt context data: %w", err)
	}
	return contextData, nil
}

// UpdateContextData updates only the context_data field of a tech-debt item.
func (r *TechDebtRepository) UpdateContextData(ctx context.Context, id int64, contextData *string) error {
	query := `UPDATE tech_debts SET context_data = ? WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, contextData, id)
	if err != nil {
		return fmt.Errorf("failed to update tech-debt context data: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("tech-debt not found with id %d", id)
	}
	return nil
}
