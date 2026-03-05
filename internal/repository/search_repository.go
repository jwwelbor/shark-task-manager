package repository

import (
	"context"
	"fmt"
	"strings"
)

// SearchRepository handles full-text search operations using FTS5
type SearchRepository struct {
	db *DB
}

// NewSearchRepository creates a new SearchRepository
func NewSearchRepository(db *DB) *SearchRepository {
	return &SearchRepository{db: db}
}

// SearchResult represents a search result with task information
type SearchResult struct {
	TaskKey     string  `json:"task_key"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Rank        float64 `json:"rank"`              // FTS5 relevance score
	Snippet     string  `json:"snippet,omitempty"` // Highlighted snippet
}

// EntitySearchResult represents a cross-entity search result.
// Severity is populated only for bugs; omitted for all other entity types.
type EntitySearchResult struct {
	EntityType string `json:"entity_type"`
	Key        string `json:"key"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	Severity   string `json:"severity,omitempty"` // bugs only
}

// SearchAll performs a LIKE-based full-text search across epics, features, tasks,
// bugs, and change_cards. If entityType is non-nil and non-empty, results are
// filtered to that type only.
// Valid entityType values: "epic", "feature", "task", "bug", "change".
// An empty query returns an empty result set (no error).
func (r *SearchRepository) SearchAll(ctx context.Context, query string, entityType *string) ([]*EntitySearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return []*EntitySearchResult{}, nil
	}

	pattern := "%" + query + "%"

	// UNION query across all entity tables. Column order: entity_type, key, title, status, severity.
	unionSQL := `
		SELECT 'epic'    AS entity_type, key, title, status, '' AS severity
		FROM epics
		WHERE title LIKE ? OR key LIKE ?

		UNION ALL

		SELECT 'feature' AS entity_type, key, title, status, '' AS severity
		FROM features
		WHERE title LIKE ? OR key LIKE ?

		UNION ALL

		SELECT 'task'    AS entity_type, key, title, status, '' AS severity
		FROM tasks
		WHERE title LIKE ? OR key LIKE ?
		   OR COALESCE(description, '') LIKE ?

		UNION ALL

		SELECT 'bug'     AS entity_type, key, title, CAST(status AS TEXT), CAST(severity AS TEXT) AS severity
		FROM bugs
		WHERE title LIKE ? OR key LIKE ?
		   OR COALESCE(description, '') LIKE ?

		UNION ALL

		SELECT 'change'  AS entity_type, key, title, CAST(status AS TEXT), '' AS severity
		FROM change_cards
		WHERE title LIKE ? OR key LIKE ?
		   OR COALESCE(description, '') LIKE ?

		ORDER BY entity_type, key
	`

	args := []interface{}{
		pattern, pattern, // epic: title, key
		pattern, pattern, // feature: title, key
		pattern, pattern, pattern, // task: title, key, description
		pattern, pattern, pattern, // bug: title, key, description
		pattern, pattern, pattern, // change_card: title, key, description
	}

	rows, err := r.db.QueryContext(ctx, unionSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var results []*EntitySearchResult
	for rows.Next() {
		res := &EntitySearchResult{}
		if err := rows.Scan(&res.EntityType, &res.Key, &res.Title, &res.Status, &res.Severity); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		// Apply entity type filter if requested
		if entityType != nil && *entityType != "" && res.EntityType != *entityType {
			continue
		}

		results = append(results, res)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return results, nil
}

// RebuildIndex rebuilds the FTS5 search index from current task data
func (r *SearchRepository) RebuildIndex(ctx context.Context) error {
	// First, clear the existing FTS index
	if _, err := r.db.ExecContext(ctx, "DELETE FROM task_search_fts"); err != nil {
		return fmt.Errorf("failed to clear search index: %w", err)
	}

	// Rebuild from tasks, task_notes, and task_criteria
	query := `
		INSERT INTO task_search_fts (task_key, title, description, note_content, criterion_text, metadata_text)
		SELECT
			t.key as task_key,
			t.title,
			COALESCE(t.description, ''),
			COALESCE((SELECT GROUP_CONCAT(content, ' ') FROM entity_notes WHERE entity_type = 'task' AND entity_id = t.id), ''),
			COALESCE((SELECT GROUP_CONCAT(criterion, ' ') FROM task_criteria WHERE task_id = t.id), ''),
			COALESCE(t.agent_type || ' ' || t.status, '')
		FROM tasks t
	`

	if _, err := r.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to rebuild search index: %w", err)
	}

	return nil
}

// IndexTask indexes or updates a single task in the FTS5 index
func (r *SearchRepository) IndexTask(ctx context.Context, taskID int64) error {
	// First, get the task key for deletion
	var taskKey string
	err := r.db.QueryRowContext(ctx, "SELECT key FROM tasks WHERE id = ?", taskID).Scan(&taskKey)
	if err != nil {
		return fmt.Errorf("failed to get task key: %w", err)
	}

	// Delete existing entry
	if _, err := r.db.ExecContext(ctx, "DELETE FROM task_search_fts WHERE task_key = ?", taskKey); err != nil {
		return fmt.Errorf("failed to delete from search index: %w", err)
	}

	// Re-index the task with all its notes and criteria
	query := `
		INSERT INTO task_search_fts (task_key, title, description, note_content, criterion_text, metadata_text)
		SELECT
			t.key,
			t.title,
			COALESCE(t.description, ''),
			COALESCE((SELECT GROUP_CONCAT(content, ' ') FROM entity_notes WHERE entity_type = 'task' AND entity_id = t.id), ''),
			COALESCE((SELECT GROUP_CONCAT(criterion, ' ') FROM task_criteria WHERE task_id = t.id), ''),
			COALESCE(t.agent_type || ' ' || t.status, '')
		FROM tasks t
		WHERE t.id = ?
	`

	if _, err := r.db.ExecContext(ctx, query, taskID); err != nil {
		return fmt.Errorf("failed to index task: %w", err)
	}

	return nil
}

// Search performs a full-text search across all indexed task data
// The query uses FTS5 MATCH syntax. Examples:
// - "database" - find tasks containing "database"
// - "database AND schema" - find tasks with both words
// - "database OR migration" - find tasks with either word
// - "\"exact phrase\"" - find exact phrase
func (r *SearchRepository) Search(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	if query == "" {
		return []*SearchResult{}, nil
	}

	// Sanitize query for FTS5 - escape double quotes
	query = strings.ReplaceAll(query, `"`, `""`)

	// FTS5 search with ranking
	searchQuery := `
		SELECT
			task_key,
			title,
			description,
			rank
		FROM task_search_fts
		WHERE task_search_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, searchQuery, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		result := &SearchResult{}
		err := rows.Scan(
			&result.TaskKey,
			&result.Title,
			&result.Description,
			&result.Rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return results, nil
}

// SearchWithSnippets performs search and includes highlighted snippets
func (r *SearchRepository) SearchWithSnippets(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	if query == "" {
		return []*SearchResult{}, nil
	}

	// Sanitize query for FTS5
	query = strings.ReplaceAll(query, `"`, `""`)

	// FTS5 search with snippet generation
	// snippet(table, column_index, before, after, ellipsis, max_tokens)
	searchQuery := `
		SELECT
			task_key,
			title,
			description,
			rank,
			snippet(task_search_fts, 1, '<mark>', '</mark>', '...', 15) as snippet
		FROM task_search_fts
		WHERE task_search_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, searchQuery, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search with snippets: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		result := &SearchResult{}
		err := rows.Scan(
			&result.TaskKey,
			&result.Title,
			&result.Description,
			&result.Rank,
			&result.Snippet,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return results, nil
}

// SearchByEpic searches within a specific epic
func (r *SearchRepository) SearchByEpic(ctx context.Context, epicKey string, query string, limit int) ([]*SearchResult, error) {
	if query == "" {
		return []*SearchResult{}, nil
	}

	// Sanitize query
	query = strings.ReplaceAll(query, `"`, `""`)

	// Search within epic by filtering task keys
	// Task keys have format T-E{epic}-F{feature}-{task}
	searchQuery := `
		SELECT
			task_key,
			title,
			description,
			rank
		FROM task_search_fts
		WHERE task_search_fts MATCH ?
		AND task_key LIKE ?
		ORDER BY rank
		LIMIT ?
	`

	pattern := fmt.Sprintf("T-%s-%%", epicKey)
	rows, err := r.db.QueryContext(ctx, searchQuery, query, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute epic search: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		result := &SearchResult{}
		err := rows.Scan(
			&result.TaskKey,
			&result.Title,
			&result.Description,
			&result.Rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return results, nil
}

// SearchByFeature searches within a specific feature
func (r *SearchRepository) SearchByFeature(ctx context.Context, featureKey string, query string, limit int) ([]*SearchResult, error) {
	if query == "" {
		return []*SearchResult{}, nil
	}

	// Sanitize query
	query = strings.ReplaceAll(query, `"`, `""`)

	// Task keys have format T-E{epic}-F{feature}-{task}
	searchQuery := `
		SELECT
			task_key,
			title,
			description,
			rank
		FROM task_search_fts
		WHERE task_search_fts MATCH ?
		AND task_key LIKE ?
		ORDER BY rank
		LIMIT ?
	`

	pattern := fmt.Sprintf("T-%s-%%", featureKey)
	rows, err := r.db.QueryContext(ctx, searchQuery, query, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute feature search: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		result := &SearchResult{}
		err := rows.Scan(
			&result.TaskKey,
			&result.Title,
			&result.Description,
			&result.Rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return results, nil
}
