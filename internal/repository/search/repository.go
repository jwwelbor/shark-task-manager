package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/searchindex"
)

// SearchRepository handles full-text search operations using FTS5
type SearchRepository struct {
	db *dbconn.DB
}

// NewSearchRepository creates a new SearchRepository
func NewSearchRepository(db *dbconn.DB) *SearchRepository {
	return &SearchRepository{db: db}
}

// EntitySearchResult represents a cross-entity search result.
// Severity is populated only for bugs; omitted for all other entity types.
// ID is the primary-key of the matched entity in its source table. It is
// guaranteed to be non-zero for every row returned by SearchAll. The field
// is required by the tag post-filter in the search service (REQ-F-012).
type EntitySearchResult struct {
	EntityType string  `json:"entity_type"`
	ID         int64   `json:"id,omitempty"`
	Key        string  `json:"key"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Severity   string  `json:"severity,omitempty"` // bugs only
	Rank       float64 `json:"rank"`
	Snippet    string  `json:"snippet,omitempty"`
}

// SearchAll performs an FTS5 search across all indexed Shark entity types.
// If entityType is non-nil and non-empty, results are filtered in SQL to that
// type only. Valid entityType values: "epic", "feature", "task", "bug",
// "change", "tech_debt", "idea", and "question".
// An empty query returns an empty result set (no error).
func (r *SearchRepository) SearchAll(ctx context.Context, query string, entityType *string) ([]*EntitySearchResult, error) {
	matchQuery := buildFTSQuery(query)
	if matchQuery == "" {
		return []*EntitySearchResult{}, nil
	}

	typeFilter := ""
	if entityType != nil {
		typeFilter = strings.TrimSpace(*entityType)
	}

	searchSQL := `
		SELECT
			entity_type,
			entity_id,
			key,
			title,
			status,
			severity,
			bm25(entity_search_fts) AS rank,
			snippet(entity_search_fts, -1, '<mark>', '</mark>', '...', 20) AS snippet
		FROM entity_search_fts
		WHERE entity_search_fts MATCH ?
		  AND (? = '' OR entity_type = ?)
		ORDER BY rank, entity_type, key
	`

	rows, err := r.db.QueryContext(ctx, searchSQL, matchQuery, typeFilter, typeFilter)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var results []*EntitySearchResult
	for rows.Next() {
		res := &EntitySearchResult{}
		if err := rows.Scan(
			&res.EntityType,
			&res.ID,
			&res.Key,
			&res.Title,
			&res.Status,
			&res.Severity,
			&res.Rank,
			&res.Snippet,
		); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		results = append(results, res)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return results, nil
}

func buildFTSQuery(query string) string {
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return ""
	}

	terms := make([]string, 0, len(fields))
	for _, field := range fields {
		term := strings.TrimSpace(field)
		if term == "" {
			continue
		}
		term = strings.ReplaceAll(term, `"`, `""`)
		terms = append(terms, `"`+term+`"`)
	}

	return strings.Join(terms, " ")
}

// RebuildIndex rebuilds the unified FTS5 search index from current source data.
func (r *SearchRepository) RebuildIndex(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM entity_search_fts"); err != nil {
		return fmt.Errorf("failed to clear unified search index: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, searchindex.RebuildSQL()); err != nil {
		return fmt.Errorf("failed to rebuild unified search index: %w", err)
	}

	return nil
}

// IndexEntity indexes or updates a single row in the unified entity FTS index.
// Unsupported entity types are ignored so polymorphic note writes on non-search
// entity types (for example, sprints) remain valid.
func (r *SearchRepository) IndexEntity(ctx context.Context, entityType models.EntityType, entityID int64) error {
	insertSQL, ok := searchindex.InsertSQL(entityType)
	if !ok {
		return nil
	}

	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin search index transaction: %w", err)
	}
	defer func() {
		// Best-effort cleanup: after Commit, Rollback reports sql.ErrTxDone.
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, "DELETE FROM entity_search_fts WHERE entity_type = ? AND entity_id = ?", string(entityType), entityID); err != nil {
		return fmt.Errorf("failed to delete existing unified search index row: %w", err)
	}
	if _, err := tx.ExecContext(ctx, insertSQL, entityID); err != nil {
		return fmt.Errorf("failed to index %s %d: %w", entityType, entityID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit search index transaction: %w", err)
	}

	return nil
}

// RemoveEntity removes a single row from the unified entity FTS index.
// Unsupported entity types are ignored for the same reason as IndexEntity.
func (r *SearchRepository) RemoveEntity(ctx context.Context, entityType models.EntityType, entityID int64) error {
	if !searchindex.SupportsEntity(entityType) {
		return nil
	}
	if _, err := r.db.ExecContext(ctx, "DELETE FROM entity_search_fts WHERE entity_type = ? AND entity_id = ?", string(entityType), entityID); err != nil {
		return fmt.Errorf("failed to remove %s %d from unified search index: %w", entityType, entityID, err)
	}
	return nil
}
