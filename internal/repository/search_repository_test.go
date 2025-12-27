package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSearchTestDB(t *testing.T) *DB {
	testDB, err := db.InitDB(":memory:")
	require.NoError(t, err)
	return &DB{DB: testDB}
}

// isFTS5Available checks if FTS5 extension is available
func isFTS5Available(db *DB) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task_search_fts'").Scan(&count)
	return err == nil && count > 0
}

// skipIfNoFTS5 skips the test if FTS5 is not available
func skipIfNoFTS5(t *testing.T, db *DB) {
	if !isFTS5Available(db) {
		t.Skip("FTS5 not available, skipping search test")
	}
}

func createTestDataForSearch(t *testing.T, db *DB) (int64, int64) {
	// Create epic
	epicRepo := NewEpicRepository(db)
	epic := &models.Epic{
		Key:      "E01",
		Title:    "Database Features",
		Status:   "active",
		Priority: "high",
	}
	require.NoError(t, epicRepo.Create(context.Background(), epic))

	// Create feature
	featureRepo := NewFeatureRepository(db)
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E01-F01",
		Title:  "Migration Tools",
		Status: "active",
	}
	require.NoError(t, featureRepo.Create(context.Background(), feature))

	// Create tasks
	taskRepo := NewTaskRepository(db)
	desc1 := "Implement automated database schema migrations"
	agent1 := models.AgentTypeBackend
	task1 := &models.Task{
		FeatureID:   feature.ID,
		Key:         "T-E01-F01-001",
		Title:       "Database migration system",
		Description: &desc1,
		Status:      "todo",
		Priority:    5,
		AgentType:   &agent1,
	}
	require.NoError(t, taskRepo.Create(context.Background(), task1))

	desc2 := "Add full-text search using FTS5"
	agent2 := models.AgentTypeBackend
	task2 := &models.Task{
		FeatureID:   feature.ID,
		Key:         "T-E01-F01-002",
		Title:       "Search functionality",
		Description: &desc2,
		Status:      "in_progress",
		Priority:    8,
		AgentType:   &agent2,
	}
	require.NoError(t, taskRepo.Create(context.Background(), task2))

	desc3 := "Create admin dashboard for monitoring"
	agent3 := models.AgentTypeFrontend
	task3 := &models.Task{
		FeatureID:   feature.ID,
		Key:         "T-E01-F01-003",
		Title:       "Frontend dashboard",
		Description: &desc3,
		Status:      "todo",
		Priority:    3,
		AgentType:   &agent3,
	}
	require.NoError(t, taskRepo.Create(context.Background(), task3))

	return task1.ID, task2.ID
}

func TestSearchRepository_RebuildIndex(t *testing.T) {
	db := setupSearchTestDB(t)
	defer db.Close()
	skipIfNoFTS5(t, db)

	task1ID, task2ID := createTestDataForSearch(t, db)

	// Add some notes and criteria
	noteRepo := NewTaskNoteRepository(db)
	criteriaRepo := NewTaskCriteriaRepository(db)
	ctx := context.Background()

	note1 := &models.TaskNote{
		TaskID:   task1ID,
		NoteType: models.NoteTypeComment,
		Content:  "Need to support PostgreSQL and MySQL",
	}
	require.NoError(t, noteRepo.Create(ctx, note1))

	criteria1 := &models.TaskCriteria{
		TaskID:    task2ID,
		Criterion: "FTS5 index created",
		Status:    models.CriteriaStatusComplete,
	}
	require.NoError(t, criteriaRepo.Create(ctx, criteria1))

	// Rebuild index
	searchRepo := NewSearchRepository(db)
	err := searchRepo.RebuildIndex(ctx)
	assert.NoError(t, err)

	// Search should now work
	results, err := searchRepo.Search(ctx, "database", 10)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestSearchRepository_Search_Basic(t *testing.T) {
	db := setupSearchTestDB(t)
	defer db.Close()
	skipIfNoFTS5(t, db)

	createTestDataForSearch(t, db)
	searchRepo := NewSearchRepository(db)
	ctx := context.Background()

	// Rebuild index first
	require.NoError(t, searchRepo.RebuildIndex(ctx))

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectTaskKey string
	}{
		{
			name:          "search for database",
			query:         "database",
			expectedCount: 1,
			expectTaskKey: "T-E01-F01-001",
		},
		{
			name:          "search for search",
			query:         "search",
			expectedCount: 1,
			expectTaskKey: "T-E01-F01-002",
		},
		{
			name:          "search for frontend",
			query:         "frontend",
			expectedCount: 1,
			expectTaskKey: "T-E01-F01-003",
		},
		{
			name:          "empty query",
			query:         "",
			expectedCount: 0,
			expectTaskKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := searchRepo.Search(ctx, tt.query, 10)
			assert.NoError(t, err)
			assert.Len(t, results, tt.expectedCount)

			if tt.expectedCount > 0 {
				found := false
				for _, r := range results {
					if r.TaskKey == tt.expectTaskKey {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected to find task %s", tt.expectTaskKey)
			}
		})
	}
}

func TestSearchRepository_SearchWithSnippets(t *testing.T) {
	db := setupSearchTestDB(t)
	defer db.Close()
	skipIfNoFTS5(t, db)

	createTestDataForSearch(t, db)
	searchRepo := NewSearchRepository(db)
	ctx := context.Background()

	// Rebuild index
	require.NoError(t, searchRepo.RebuildIndex(ctx))

	// Search with snippets
	results, err := searchRepo.SearchWithSnippets(ctx, "migration", 10)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)

	// Check that snippet is populated
	if len(results) > 0 {
		assert.NotEmpty(t, results[0].Snippet)
	}
}

func TestSearchRepository_SearchByEpic(t *testing.T) {
	db := setupSearchTestDB(t)
	defer db.Close()
	skipIfNoFTS5(t, db)

	// Create data in epic E01
	createTestDataForSearch(t, db)

	// Create another epic with different tasks
	epicRepo := NewEpicRepository(db)
	epic2 := &models.Epic{
		Key:      "E02",
		Title:    "User Features",
		Status:   "active",
		Priority: "medium",
	}
	require.NoError(t, epicRepo.Create(context.Background(), epic2))

	featureRepo := NewFeatureRepository(db)
	feature2 := &models.Feature{
		EpicID: epic2.ID,
		Key:    "E02-F01",
		Title:  "Authentication",
		Status: "active",
	}
	require.NoError(t, featureRepo.Create(context.Background(), feature2))

	taskRepo := NewTaskRepository(db)
	desc4 := "Optimize database connections"
	agent4 := models.AgentTypeBackend
	task4 := &models.Task{
		FeatureID:   feature2.ID,
		Key:         "T-E02-F01-001",
		Title:       "Database connection pooling",
		Description: &desc4,
		Status:      "todo",
		Priority:    5,
		AgentType:   &agent4,
	}
	require.NoError(t, taskRepo.Create(context.Background(), task4))

	searchRepo := NewSearchRepository(db)
	ctx := context.Background()

	// Rebuild index
	require.NoError(t, searchRepo.RebuildIndex(ctx))

	// Search only in E01
	results, err := searchRepo.SearchByEpic(ctx, "E01", "database", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "T-E01-F01-001", results[0].TaskKey)

	// Search only in E02
	results, err = searchRepo.SearchByEpic(ctx, "E02", "database", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "T-E02-F01-001", results[0].TaskKey)
}

func TestSearchRepository_SearchByFeature(t *testing.T) {
	db := setupSearchTestDB(t)
	defer db.Close()
	skipIfNoFTS5(t, db)

	createTestDataForSearch(t, db)
	searchRepo := NewSearchRepository(db)
	ctx := context.Background()

	// Rebuild index
	require.NoError(t, searchRepo.RebuildIndex(ctx))

	// Search within specific feature
	results, err := searchRepo.SearchByFeature(ctx, "E01-F01", "database", 10)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)

	// All results should be from E01-F01
	for _, r := range results {
		assert.Contains(t, r.TaskKey, "T-E01-F01-")
	}
}

func TestSearchRepository_IndexTask(t *testing.T) {
	db := setupSearchTestDB(t)
	defer db.Close()
	skipIfNoFTS5(t, db)

	task1ID, _ := createTestDataForSearch(t, db)
	searchRepo := NewSearchRepository(db)
	ctx := context.Background()

	// Index a single task
	err := searchRepo.IndexTask(ctx, task1ID)
	assert.NoError(t, err)

	// Search should work for that task
	results, err := searchRepo.Search(ctx, "migration", 10)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestSearchRepository_Search_Limit(t *testing.T) {
	db := setupSearchTestDB(t)
	defer db.Close()
	skipIfNoFTS5(t, db)

	// Create many tasks
	epicRepo := NewEpicRepository(db)
	epic := &models.Epic{
		Key:      "E10",
		Title:    "Test Epic",
		Status:   "active",
		Priority: "high",
	}
	require.NoError(t, epicRepo.Create(context.Background(), epic))

	featureRepo := NewFeatureRepository(db)
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E10-F01",
		Title:  "Test Feature",
		Status: "active",
	}
	require.NoError(t, featureRepo.Create(context.Background(), feature))

	taskRepo := NewTaskRepository(db)
	desc := "This is a common description"
	agent := models.AgentTypeBackend
	for i := 1; i <= 10; i++ {
		task := &models.Task{
			FeatureID:   feature.ID,
			Key:         fmt.Sprintf("T-E10-F01-%03d", i),
			Title:       fmt.Sprintf("Common task %d", i),
			Description: &desc,
			Status:      "todo",
			Priority:    5,
			AgentType:   &agent,
		}
		require.NoError(t, taskRepo.Create(context.Background(), task))
	}

	searchRepo := NewSearchRepository(db)
	ctx := context.Background()

	// Rebuild index
	require.NoError(t, searchRepo.RebuildIndex(ctx))

	// Search with limit
	results, err := searchRepo.Search(ctx, "common", 5)
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(results), 5)
}
