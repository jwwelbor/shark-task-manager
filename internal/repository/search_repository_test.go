package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchRepository_RebuildIndexUsesUnifiedSearchPath(t *testing.T) {
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
		Content:    "Unified only note phrase",
	}))

	searchRepo := NewSearchRepository(repoDb)
	require.NoError(t, searchRepo.RebuildIndex(ctx))

	entityType := "task"
	results, err := searchRepo.SearchAll(ctx, "unified", &entityType)
	require.NoError(t, err)

	result := requireSearchResult(t, results, "task", "T-E01-F01-001")
	assert.NotZero(t, result.Rank)
	assert.Contains(t, result.Snippet, "<mark>")
}

func TestSearchRepository_IndexEntitySupersedesLegacyTaskIndexer(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	ctx := context.Background()

	var taskID int64
	require.NoError(t, repoDb.QueryRowContext(ctx, `SELECT id FROM tasks WHERE key = 'T-E01-F01-001'`).Scan(&taskID))
	_, err := repoDb.ExecContext(ctx, `
		UPDATE tasks
		SET title = 'Quartz migration title'
		WHERE id = ?
	`, taskID)
	require.NoError(t, err)

	searchRepo := NewSearchRepository(repoDb)
	require.NoError(t, searchRepo.IndexEntity(ctx, models.EntityTypeTask, taskID))

	entityType := "task"
	results, err := searchRepo.SearchAll(ctx, "quartz", &entityType)
	require.NoError(t, err)

	result := requireSearchResult(t, results, "task", "T-E01-F01-001")
	assert.Equal(t, "Quartz migration title", result.Title)
}

func TestSearchRepository_RemoveEntitySupersedesLegacyTaskSearchDeletes(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	ctx := context.Background()

	var taskID int64
	require.NoError(t, repoDb.QueryRowContext(ctx, `SELECT id FROM tasks WHERE key = 'T-E01-F01-001'`).Scan(&taskID))

	searchRepo := NewSearchRepository(repoDb)
	require.NoError(t, searchRepo.RebuildIndex(ctx))
	require.NoError(t, searchRepo.RemoveEntity(ctx, models.EntityTypeTask, taskID))

	entityType := "task"
	results, err := searchRepo.SearchAll(ctx, "implement", &entityType)
	require.NoError(t, err)
	assert.Empty(t, results)
}
