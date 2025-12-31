package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskRepository_Create_GeneratesAndStoresSlug verifies slug generation during task creation
func TestTaskRepository_Create_GeneratesAndStoresSlug(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)

	// Seed epic and feature for foreign keys
	_, featureID := test.SeedTestData()

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E99-F99-997'")

	// Create task
	task := &models.Task{
		FeatureID: featureID,
		Key:       "T-E99-F99-997",
		Title:     "Implement User Authentication System",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}

	err := repo.Create(ctx, task)
	require.NoError(t, err)

	// Verify slug was generated and stored
	assert.NotNil(t, task.Slug, "Slug should be generated")
	assert.Equal(t, "implement-user-authentication-system", *task.Slug)

	// Verify slug is persisted in database
	retrieved, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.Slug, "Slug should be persisted")
	assert.Equal(t, "implement-user-authentication-system", *retrieved.Slug)

	// Cleanup
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()
}

// TestTaskRepository_Create_SlugHandlesSpecialCharacters verifies slug handles special characters
func TestTaskRepository_Create_SlugHandlesSpecialCharacters(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key IN ('T-E97-F01-001', 'T-E97-F01-002', 'T-E97-F01-003')")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E97-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")

	// Create a dedicated test epic for this test
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E97",
		Title:         "Test Epic for Task Slug Special Characters",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create a dedicated test feature
	featureRepo := NewFeatureRepository(db)
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E97-F01",
		Title:  "Test Feature for Task Slugs",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err, "Failed to create test feature")
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	testCases := []struct {
		title        string
		expectedSlug string
	}{
		{
			title:        "Fix Bug: Memory Leak in Worker Pool",
			expectedSlug: "fix-bug-memory-leak-in-worker-pool",
		},
		{
			title:        "Upgrade PostgreSQL -> MongoDB",
			expectedSlug: "upgrade-postgresql-mongodb",
		},
		{
			title:        "Add Support for UTF-8 & Unicode 测试",
			expectedSlug: "add-support-for-utf-8-unicode",
		},
	}

	for i, tc := range testCases {
		task := &models.Task{
			FeatureID: testFeature.ID,
			Key:       fmt.Sprintf("T-E97-F01-%03d", i+1),
			Title:     tc.title,
			Status:    models.TaskStatusTodo,
			Priority:  5,
		}

		err := repo.Create(ctx, task)
		require.NoError(t, err, "Failed to create task with key %s, title: %s", task.Key, tc.title)

		assert.NotNil(t, task.Slug, "Slug should be generated for: %s", tc.title)
		assert.Equal(t, tc.expectedSlug, *task.Slug, "Slug mismatch for: %s", tc.title)

		// Cleanup
		defer func(id int64) {
			if _, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}(task.ID)
	}
}
