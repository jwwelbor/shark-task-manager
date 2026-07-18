package changecard

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
)

// defaultChangeCardTerminalStatuses is the fallback terminal-status list used when
// the caller does not supply TerminalStatuses in ChangeCardRepoFilter. It matches
// the shipped .sharkworkflow.json change_workflow _complete_ set.
var defaultChangeCardTerminalStatuses = []string{"completed", "declined"}

// ChangeCardRepoFilter represents filtering options for listing change-cards.
type ChangeCardRepoFilter struct {
	Status          *models.ChangeCardStatus
	EpicID          *int64
	FeatureID       *int64
	IncludeTerminal bool // if false, excludes terminal statuses from results
	// TerminalStatuses is the authoritative list of terminal statuses to exclude
	// when IncludeTerminal is false. Callers should populate this from
	// workflow.Service.GetTerminalStatuses() for the change level. When empty and
	// IncludeTerminal is false the repository falls back to the shipped default
	// list so that callers that do not yet supply this field continue to work.
	TerminalStatuses []string
}

// ChangeCardRepository handles CRUD operations for change-cards.
type ChangeCardRepository struct {
	db *dbconn.DB
}

// NewChangeCardRepository creates a new ChangeCardRepository.
func NewChangeCardRepository(db *dbconn.DB) *ChangeCardRepository {
	return &ChangeCardRepository{db: db}
}

// changeCardSelectColumns defines the column list used by all SELECT queries.
const changeCardSelectColumns = `id, key, title, description, status, priority, requested_by, assigned_to, epic_id, feature_id, related_task_id, justification, impact_analysis, rollback_plan, slug, file_path, context_data, size, created_at, updated_at`

// scanCard scans a row into a ChangeCard model.
func scanCard(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.ChangeCard, error) {
	card := &models.ChangeCard{}
	err := scanner.Scan(
		&card.ID, &card.Key, &card.Title, &card.Description, &card.Status, &card.Priority,
		&card.RequestedBy, &card.AssignedTo, &card.EpicID, &card.FeatureID, &card.RelatedTaskID,
		&card.Justification, &card.ImpactAnalysis, &card.RollbackPlan,
		&card.Slug, &card.FilePath, &card.ContextData, &card.Size, &card.CreatedAt, &card.UpdatedAt,
	)
	return card, err
}

// Create creates a new change-card in the database.
func (r *ChangeCardRepository) Create(ctx context.Context, card *models.ChangeCard) error {
	if err := card.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO change_cards (key, title, description, status, priority, requested_by, assigned_to,
			epic_id, feature_id, related_task_id, justification, impact_analysis, rollback_plan, slug, file_path, size)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		card.Key, card.Title, card.Description, card.Status, card.Priority,
		card.RequestedBy, card.AssignedTo, card.EpicID, card.FeatureID, card.RelatedTaskID,
		card.Justification, card.ImpactAnalysis, card.RollbackPlan, card.Slug, card.FilePath, card.Size,
	)
	if err != nil {
		return fmt.Errorf("failed to create change-card: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	card.ID = id
	return nil
}

// GetByKey retrieves a change-card by its key, supporting dual-key lookup (numeric and slugged).
func (r *ChangeCardRepository) GetByKey(ctx context.Context, key string) (*models.ChangeCard, error) {
	originalKey := strings.ToUpper(key)
	normalizedKey := originalKey
	if canonical, err := keys.NormalizeChangeKey(originalKey); err == nil {
		normalizedKey = canonical
	}

	// Try exact match first
	card, err := r.getByExactKey(ctx, normalizedKey)
	if err == nil {
		return card, nil
	}
	if !errors.Is(err, repoerr.ErrNotFound) {
		return nil, err
	}
	if normalizedKey != originalKey {
		card, err = r.getByExactKey(ctx, originalKey)
		if err == nil {
			return card, nil
		}
		if !errors.Is(err, repoerr.ErrNotFound) {
			return nil, err
		}
	}

	// If not found and the input is a slugged change-card key alias
	// (CC-001-some-slug, CC001-some-slug, C001-some-slug), resolve the
	// canonical key and query by (key, slug).
	if canonicalKey, slug, ok := splitChangeCardSlugKey(key); ok {
		query := `SELECT ` + changeCardSelectColumns + ` FROM change_cards WHERE key = ? AND slug = ?`
		card, scanErr := scanCard(r.db.QueryRowContext(ctx, query, canonicalKey, slug))
		if scanErr == nil {
			return card, nil
		}
	}

	return nil, fmt.Errorf("change-card not found: %s: %w", key, repoerr.ErrNotFound)
}

func splitChangeCardSlugKey(input string) (canonicalKey, slug string, ok bool) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return "", "", false
	}

	upper := strings.ToUpper(raw)
	for i := 1; i < len(upper); i++ {
		if upper[i] != '-' {
			continue
		}
		candidate := upper[:i]
		canonical, err := keys.NormalizeChangeKey(candidate)
		if err != nil {
			continue
		}
		if i+1 >= len(raw) {
			return "", "", false
		}
		return canonical, strings.ToLower(raw[i+1:]), true
	}

	return "", "", false
}

// getByExactKey retrieves a change-card by exact key match.
func (r *ChangeCardRepository) getByExactKey(ctx context.Context, key string) (*models.ChangeCard, error) {
	query := `SELECT ` + changeCardSelectColumns + ` FROM change_cards WHERE key = ?`

	card, err := scanCard(r.db.QueryRowContext(ctx, query, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("change-card not found: %s: %w", key, repoerr.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get change-card: %w", err)
	}

	return card, nil
}

// GetByID retrieves a change-card by its database ID.
func (r *ChangeCardRepository) GetByID(ctx context.Context, id int64) (*models.ChangeCard, error) {
	query := `SELECT ` + changeCardSelectColumns + ` FROM change_cards WHERE id = ?`

	card, err := scanCard(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("change-card not found with id %d: %w", id, repoerr.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get change-card: %w", err)
	}

	return card, nil
}

// Update updates an existing change-card.
func (r *ChangeCardRepository) Update(ctx context.Context, card *models.ChangeCard) error {
	if err := card.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE change_cards
		SET title = ?, description = ?, status = ?, priority = ?, requested_by = ?, assigned_to = ?,
			epic_id = ?, feature_id = ?, related_task_id = ?, justification = ?, impact_analysis = ?,
			rollback_plan = ?, slug = ?, file_path = ?, size = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		card.Title, card.Description, card.Status, card.Priority,
		card.RequestedBy, card.AssignedTo, card.EpicID, card.FeatureID, card.RelatedTaskID,
		card.Justification, card.ImpactAnalysis, card.RollbackPlan, card.Slug, card.FilePath, card.Size,
		card.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update change-card: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("change-card not found with id %d: %w", card.ID, repoerr.ErrNotFound)
	}

	return nil
}

// Delete deletes a change-card by its ID.
func (r *ChangeCardRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM change_cards WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete change-card: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("change-card not found with id %d: %w", id, repoerr.ErrNotFound)
	}

	return nil
}

// UpdateStatus updates only the status of a change-card.
func (r *ChangeCardRepository) UpdateStatus(ctx context.Context, id int64, status models.ChangeCardStatus) error {
	query := `UPDATE change_cards SET status = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update change-card status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("change-card not found with id %d: %w", id, repoerr.ErrNotFound)
	}

	return nil
}

// UpdateStatusIfCurrent atomically updates change-card status only when the
// current stored status still matches expectedStatus (case-insensitive).
func (r *ChangeCardRepository) UpdateStatusIfCurrent(ctx context.Context, id int64, expectedStatus models.ChangeCardStatus, newStatus models.ChangeCardStatus) (bool, error) {
	updated, err := dbconn.ConditionalStatusUpdate(ctx, r.db, "change_cards", id, string(expectedStatus), string(newStatus), false)
	if err != nil {
		return false, fmt.Errorf("conditionally update change-card status: %w", err)
	}
	return updated, nil
}

// UpdateContextData updates only the context_data field of a change-card.
func (r *ChangeCardRepository) UpdateContextData(ctx context.Context, id int64, contextData *string) error {
	query := `UPDATE change_cards SET context_data = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, contextData, id)
	if err != nil {
		return fmt.Errorf("failed to update change-card context data: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("change-card not found with id %d: %w", id, repoerr.ErrNotFound)
	}

	return nil
}

// List retrieves change-cards with optional filtering.
func (r *ChangeCardRepository) List(ctx context.Context, filter *ChangeCardRepoFilter) ([]*models.ChangeCard, error) {
	query := `SELECT ` + changeCardSelectColumns + ` FROM change_cards`

	var conditions []string
	var args []interface{}

	if filter != nil {
		if filter.Status != nil {
			conditions = append(conditions, "status = ?")
			args = append(args, *filter.Status)
		}

		if filter.EpicID != nil {
			conditions = append(conditions, "epic_id = ?")
			args = append(args, *filter.EpicID)
		}

		if filter.FeatureID != nil {
			conditions = append(conditions, "feature_id = ?")
			args = append(args, *filter.FeatureID)
		}

		if !filter.IncludeTerminal {
			terminalStatuses := filter.TerminalStatuses
			if len(terminalStatuses) == 0 {
				terminalStatuses = defaultChangeCardTerminalStatuses
			}
			placeholders := make([]string, len(terminalStatuses))
			for i, s := range terminalStatuses {
				placeholders[i] = "?"
				args = append(args, s)
			}
			conditions = append(conditions, "status NOT IN ("+strings.Join(placeholders, ", ")+")")
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list change-cards: %w", err)
	}
	defer rows.Close()

	var cards []*models.ChangeCard
	for rows.Next() {
		card, scanErr := scanCard(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan change-card: %w", scanErr)
		}
		cards = append(cards, card)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating change-cards: %w", err)
	}

	return cards, nil
}

// GetRecent returns the most recently created change-cards, ordered by created_at DESC.
// limit must be positive; the caller (service) is responsible for bounds-checking.
// Returns an empty (non-nil) slice if no rows exist.
func (r *ChangeCardRepository) GetRecent(ctx context.Context, limit int) ([]*models.ChangeCard, error) {
	query := `SELECT ` + changeCardSelectColumns + ` FROM change_cards ORDER BY created_at DESC LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent change-cards: %w", err)
	}
	defer rows.Close()

	cards := []*models.ChangeCard{}
	for rows.Next() {
		card, scanErr := scanCard(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan change-card: %w", scanErr)
		}
		cards = append(cards, card)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating change-cards: %w", err)
	}

	return cards, nil
}

// ListByEpic retrieves change-cards linked to a specific epic.
func (r *ChangeCardRepository) ListByEpic(ctx context.Context, epicID int64) ([]*models.ChangeCard, error) {
	query := `SELECT ` + changeCardSelectColumns + ` FROM change_cards WHERE epic_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to list change-cards by epic: %w", err)
	}
	defer rows.Close()

	var cards []*models.ChangeCard
	for rows.Next() {
		card, scanErr := scanCard(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan change-card: %w", scanErr)
		}
		cards = append(cards, card)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating change-cards: %w", err)
	}

	return cards, nil
}

// ListByFeature retrieves change-cards linked to a specific feature.
func (r *ChangeCardRepository) ListByFeature(ctx context.Context, featureID int64) ([]*models.ChangeCard, error) {
	query := `SELECT ` + changeCardSelectColumns + ` FROM change_cards WHERE feature_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to list change-cards by feature: %w", err)
	}
	defer rows.Close()

	var cards []*models.ChangeCard
	for rows.Next() {
		card, scanErr := scanCard(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan change-card: %w", scanErr)
		}
		cards = append(cards, card)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating change-cards: %w", err)
	}

	return cards, nil
}

// CountByStatus returns counts of change-cards grouped by status.
func (r *ChangeCardRepository) CountByStatus(ctx context.Context) (map[string]int, error) {
	query := `SELECT status, COUNT(*) FROM change_cards GROUP BY status`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count change-cards by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}
		counts[status] = count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating status counts: %w", err)
	}

	return counts, nil
}

// GetContextData retrieves the context data JSON string for a change-card by its ID.
func (r *ChangeCardRepository) GetContextData(ctx context.Context, id int64) (*string, error) {
	query := `SELECT context_data FROM change_cards WHERE id = ?`
	var contextData *string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&contextData)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("change-card not found with id %d: %w", id, repoerr.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get change-card context data: %w", err)
	}
	return contextData, nil
}

// GetNextKey returns the next available change-card key (e.g., CC-001, CC-002, ...).
func (r *ChangeCardRepository) GetNextKey(ctx context.Context) (string, error) {
	query := `SELECT COALESCE(MAX(CAST(SUBSTR(key, 4) AS INTEGER)), 0) FROM change_cards`

	var maxNum int
	err := r.db.QueryRowContext(ctx, query).Scan(&maxNum)
	if err != nil {
		return "", fmt.Errorf("failed to get next change-card key: %w", err)
	}

	nextKey := fmt.Sprintf("CC-%03d", maxNum+1)
	return nextKey, nil
}
