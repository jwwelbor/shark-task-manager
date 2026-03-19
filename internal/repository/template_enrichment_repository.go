package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// TemplateEnrichmentRepository provides consolidated enrichment data
// for template variable population via single-query lookups.
type TemplateEnrichmentRepository struct {
	db *DB
}

// NewTemplateEnrichmentRepository creates a new TemplateEnrichmentRepository.
func NewTemplateEnrichmentRepository(db *DB) *TemplateEnrichmentRepository {
	return &TemplateEnrichmentRepository{db: db}
}

// GetTaskEnrichment fetches all enrichment data for a task in a single query.
// Returns a zero-valued struct (not an error) if the task ID does not exist.
func (r *TemplateEnrichmentRepository) GetTaskEnrichment(ctx context.Context, taskID int64) (*config.TemplateEnrichmentData, error) {
	query := `
SELECT
    COALESCE(
        (SELECT old_status FROM task_history
         WHERE task_id = t.id ORDER BY timestamp DESC LIMIT 1),
        ''
    ) AS previous_status,
    COALESCE(f.title, '') AS parent_title,
    COALESCE(e.title, '') AS grandparent_title,
    COALESCE(
        (SELECT content FROM entity_notes
         WHERE entity_type = 'task' AND entity_id = t.id
         ORDER BY created_at DESC LIMIT 1),
        ''
    ) AS latest_note_content,
    COALESCE(
        (SELECT note_type FROM entity_notes
         WHERE entity_type = 'task' AND entity_id = t.id
         ORDER BY created_at DESC LIMIT 1),
        ''
    ) AS latest_note_type,
    (SELECT COUNT(*) FROM entity_notes
     WHERE entity_type = 'task' AND entity_id = t.id
    ) AS notes_count,
    (SELECT COUNT(*) FROM entity_notes
     WHERE entity_type = 'task' AND entity_id = t.id
     AND note_type = 'rejection'
    ) AS rejection_count,
    (SELECT COUNT(*) FROM tasks
     WHERE feature_id = t.feature_id
    ) AS sibling_total,
    (SELECT COUNT(*) FROM tasks
     WHERE feature_id = t.feature_id AND status = 'completed'
    ) AS sibling_completed,
    (SELECT COUNT(*) FROM tasks
     WHERE feature_id = t.feature_id AND status = 'blocked'
    ) AS sibling_blocked
FROM tasks t
LEFT JOIN features f ON t.feature_id = f.id
LEFT JOIN epics e ON f.epic_id = e.id
WHERE t.id = ?`

	data := &config.TemplateEnrichmentData{}
	err := r.db.QueryRowContext(ctx, query, taskID).Scan(
		&data.PreviousStatus,
		&data.ParentTitle,
		&data.GrandparentTitle,
		&data.LatestNoteContent,
		&data.LatestNoteType,
		&data.NotesCount,
		&data.RejectionCount,
		&data.SiblingTotal,
		&data.SiblingCompleted,
		&data.SiblingBlocked,
	)
	if err == sql.ErrNoRows {
		return &config.TemplateEnrichmentData{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task enrichment for task %d: %w", taskID, err)
	}

	return data, nil
}

// GetFeatureEnrichment fetches all enrichment data for a feature in a single query.
// Returns a zero-valued struct (not an error) if the feature ID does not exist.
func (r *TemplateEnrichmentRepository) GetFeatureEnrichment(ctx context.Context, featureID int64) (*config.TemplateEnrichmentData, error) {
	query := `
SELECT
    '' AS previous_status,
    COALESCE(e.title, '') AS parent_title,
    '' AS grandparent_title,
    COALESCE(
        (SELECT content FROM entity_notes
         WHERE entity_type = 'feature' AND entity_id = f.id
         ORDER BY created_at DESC LIMIT 1),
        ''
    ) AS latest_note_content,
    COALESCE(
        (SELECT note_type FROM entity_notes
         WHERE entity_type = 'feature' AND entity_id = f.id
         ORDER BY created_at DESC LIMIT 1),
        ''
    ) AS latest_note_type,
    (SELECT COUNT(*) FROM entity_notes
     WHERE entity_type = 'feature' AND entity_id = f.id
    ) AS notes_count,
    (SELECT COUNT(*) FROM entity_notes
     WHERE entity_type = 'feature' AND entity_id = f.id
     AND note_type = 'rejection'
    ) AS rejection_count,
    (SELECT COUNT(*) FROM tasks WHERE feature_id = f.id) AS sibling_total,
    (SELECT COUNT(*) FROM tasks
     WHERE feature_id = f.id AND status = 'completed'
    ) AS sibling_completed,
    (SELECT COUNT(*) FROM tasks
     WHERE feature_id = f.id AND status = 'blocked'
    ) AS sibling_blocked
FROM features f
LEFT JOIN epics e ON f.epic_id = e.id
WHERE f.id = ?`

	data := &config.TemplateEnrichmentData{}
	err := r.db.QueryRowContext(ctx, query, featureID).Scan(
		&data.PreviousStatus,
		&data.ParentTitle,
		&data.GrandparentTitle,
		&data.LatestNoteContent,
		&data.LatestNoteType,
		&data.NotesCount,
		&data.RejectionCount,
		&data.SiblingTotal,
		&data.SiblingCompleted,
		&data.SiblingBlocked,
	)
	if err == sql.ErrNoRows {
		return &config.TemplateEnrichmentData{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feature enrichment for feature %d: %w", featureID, err)
	}

	return data, nil
}

// GetEpicEnrichment fetches all enrichment data for an epic in a single query.
// Returns a zero-valued struct (not an error) if the epic ID does not exist.
func (r *TemplateEnrichmentRepository) GetEpicEnrichment(ctx context.Context, epicID int64) (*config.TemplateEnrichmentData, error) {
	query := `
SELECT
    '' AS previous_status,
    '' AS parent_title,
    '' AS grandparent_title,
    COALESCE(
        (SELECT content FROM entity_notes
         WHERE entity_type = 'epic' AND entity_id = e.id
         ORDER BY created_at DESC LIMIT 1),
        ''
    ) AS latest_note_content,
    COALESCE(
        (SELECT note_type FROM entity_notes
         WHERE entity_type = 'epic' AND entity_id = e.id
         ORDER BY created_at DESC LIMIT 1),
        ''
    ) AS latest_note_type,
    (SELECT COUNT(*) FROM entity_notes
     WHERE entity_type = 'epic' AND entity_id = e.id
    ) AS notes_count,
    (SELECT COUNT(*) FROM entity_notes
     WHERE entity_type = 'epic' AND entity_id = e.id
     AND note_type = 'rejection'
    ) AS rejection_count,
    (SELECT COUNT(*) FROM features WHERE epic_id = e.id) AS sibling_total,
    (SELECT COUNT(*) FROM features
     WHERE epic_id = e.id AND status = 'completed'
    ) AS sibling_completed,
    (SELECT COUNT(*) FROM features
     WHERE epic_id = e.id AND status = 'blocked'
    ) AS sibling_blocked
FROM epics e
WHERE e.id = ?`

	data := &config.TemplateEnrichmentData{}
	err := r.db.QueryRowContext(ctx, query, epicID).Scan(
		&data.PreviousStatus,
		&data.ParentTitle,
		&data.GrandparentTitle,
		&data.LatestNoteContent,
		&data.LatestNoteType,
		&data.NotesCount,
		&data.RejectionCount,
		&data.SiblingTotal,
		&data.SiblingCompleted,
		&data.SiblingBlocked,
	)
	if err == sql.ErrNoRows {
		return &config.TemplateEnrichmentData{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get epic enrichment for epic %d: %w", epicID, err)
	}

	return data, nil
}
