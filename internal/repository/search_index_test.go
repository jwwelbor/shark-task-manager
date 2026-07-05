package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchMigrationCreatesUnifiedFTSIndex(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	var count int
	err := repoDb.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = 'entity_search_fts'
	`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "unified FTS table should exist after migration")
}

func TestSearchMigrationDoesNotCreateLegacyTaskFTSIndex(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	var count int
	err := repoDb.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = 'task_search_fts'
	`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "legacy task-only FTS table should not exist after migration")
}

func TestSearchRepository_RebuildIndexPopulatesUnifiedIndex(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	ctx := context.Background()

	var taskID int64
	require.NoError(t, repoDb.QueryRowContext(ctx, `SELECT id FROM tasks WHERE key = 'T-E01-F01-001'`).Scan(&taskID))

	noteRepo := NewEntityNoteRepository(repoDb)
	require.NoError(t, noteRepo.Create(ctx, &models.EntityNote{
		EntityType: models.EntityTypeTask,
		EntityID:   taskID,
		NoteType:   models.NoteTypeComment,
		Content:    "Notebook-only phrase for unified index",
	}))

	repo := NewSearchRepository(repoDb)
	require.NoError(t, repo.RebuildIndex(ctx))

	rows, err := repoDb.QueryContext(ctx, `
		SELECT entity_type, key, title, body, note_text, status
		FROM entity_search_fts
		ORDER BY entity_type, key
	`)
	require.NoError(t, err)
	defer rows.Close()

	type indexedRow struct {
		entityType string
		key        string
		title      string
		body       sql.NullString
		noteText   sql.NullString
		status     string
	}

	rowsByType := map[string]indexedRow{}
	for rows.Next() {
		var row indexedRow
		require.NoError(t, rows.Scan(&row.entityType, &row.key, &row.title, &row.body, &row.noteText, &row.status))
		rowsByType[row.entityType] = row
	}
	require.NoError(t, rows.Err())

	for _, entityType := range []string{"epic", "feature", "task", "bug", "change", "tech_debt", "idea"} {
		assert.Contains(t, rowsByType, entityType, "expected %s row in unified index", entityType)
	}
	assert.Contains(t, rowsByType["task"].noteText.String, "Notebook-only phrase")
	assert.Equal(t, "new", rowsByType["idea"].status)
}

func TestSearchRepository_IndexEntityRefreshesUnifiedIndexWithNotes(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	ctx := context.Background()

	var taskID int64
	require.NoError(t, repoDb.QueryRowContext(ctx, `SELECT id FROM tasks WHERE key = 'T-E01-F01-001'`).Scan(&taskID))

	noteRepo := NewEntityNoteRepository(repoDb)
	require.NoError(t, noteRepo.Create(ctx, &models.EntityNote{
		EntityType: models.EntityTypeTask,
		EntityID:   taskID,
		NoteType:   models.NoteTypeComment,
		Content:    "Incremental quartz phrase",
	}))

	repo := NewSearchRepository(repoDb)
	require.NoError(t, repo.IndexEntity(ctx, models.EntityTypeTask, taskID))

	entityType := "task"
	results, err := repo.SearchAll(ctx, "quartz", &entityType)
	require.NoError(t, err)

	result := requireSearchResult(t, results, "task", "T-E01-F01-001")
	assert.Contains(t, strings.ToLower(result.Snippet), "<mark>quartz</mark>")
}

func TestSearchRepository_RemoveEntityDeletesUnifiedIndexRow(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	ctx := context.Background()

	var bugID int64
	require.NoError(t, repoDb.QueryRowContext(ctx, `SELECT id FROM bugs WHERE key = 'B001'`).Scan(&bugID))

	repo := NewSearchRepository(repoDb)
	require.NoError(t, repo.RebuildIndex(ctx))
	require.NoError(t, repo.RemoveEntity(ctx, models.EntityTypeBug, bugID))

	entityType := "bug"
	results, err := repo.SearchAll(ctx, "login", &entityType)
	require.NoError(t, err)
	assert.Empty(t, results)
}
