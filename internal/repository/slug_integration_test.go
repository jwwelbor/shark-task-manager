package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/slug"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSlugStorage_AllEntities validates that slugs are properly stored and retrieved
// across all entity types (epics, features, tasks).
//
// This is an integration test for T-E07-F11-007: Test slug storage across all entity types
func TestSlugStorage_AllEntities(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)
	taskRepo := NewTaskRepository(db)

	t.Run("Epic slug storage", func(t *testing.T) {
		epic := &models.Epic{
			Key:      "E99",
			Title:    "Test Epic for Slug Validation",
			Status:   "active",
			Priority: "high",
		}

		err := epicRepo.Create(ctx, epic)
		require.NoError(t, err, "Epic creation should succeed")

		// Verify slug was generated and stored
		expectedSlug := slug.Generate(epic.Title)
		assert.NotEmpty(t, expectedSlug, "Generated slug should not be empty")
		require.NotNil(t, epic.Slug, "Epic slug pointer should not be nil")
		assert.Equal(t, expectedSlug, *epic.Slug, "Epic slug should match generated value")

		// Verify slug is persisted in database
		retrieved, err := epicRepo.GetByID(ctx, epic.ID)
		require.NoError(t, err, "Epic retrieval should succeed")
		require.NotNil(t, retrieved.Slug, "Retrieved epic slug pointer should not be nil")
		assert.Equal(t, expectedSlug, *retrieved.Slug, "Retrieved epic slug should match stored value")

		defer func() {
			if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}()
	})

	t.Run("Feature slug storage", func(t *testing.T) {
		// Create parent epic first
		epic := &models.Epic{
			Key:      "E99",
			Title:    "Parent Epic",
			Status:   "active",
			Priority: "high",
		}
		err := epicRepo.Create(ctx, epic)
		require.NoError(t, err)

		feature := &models.Feature{
			EpicID: epic.ID,
			Key:    "E99-F01",
			Title:  "Test Feature with Special Characters: API & UI!",
			Status: "active",
		}

		err = featureRepo.Create(ctx, feature)
		require.NoError(t, err, "Feature creation should succeed")

		// Verify slug was generated and stored
		expectedSlug := slug.Generate(feature.Title)
		assert.NotEmpty(t, expectedSlug, "Generated slug should not be empty")
		require.NotNil(t, feature.Slug, "Feature slug pointer should not be nil")
		assert.Equal(t, expectedSlug, *feature.Slug, "Feature slug should match generated value")
		assert.Equal(t, "test-feature-with-special-characters-api-ui", *feature.Slug, "Slug should normalize special characters")

		// Verify slug is persisted in database
		retrieved, err := featureRepo.GetByID(ctx, feature.ID)
		require.NoError(t, err, "Feature retrieval should succeed")
		require.NotNil(t, retrieved.Slug, "Retrieved feature slug pointer should not be nil")
		assert.Equal(t, expectedSlug, *retrieved.Slug, "Retrieved feature slug should match stored value")

		defer func() {
			if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}()
		defer func() {
			if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}()
	})

	t.Run("Task slug storage", func(t *testing.T) {
		// Create parent epic and feature
		epic := &models.Epic{
			Key:      "E99",
			Title:    "Parent Epic",
			Status:   "active",
			Priority: "high",
		}
		err := epicRepo.Create(ctx, epic)
		require.NoError(t, err)

		feature := &models.Feature{
			EpicID: epic.ID,
			Key:    "E99-F01",
			Title:  "Parent Feature",
			Status: "active",
		}
		err = featureRepo.Create(ctx, feature)
		require.NoError(t, err)

		backendAgent := "backend"
		task := &models.Task{
			FeatureID: feature.ID,
			Key:       "T-E99-F99-001",
			Title:     "Implement user authentication with OAuth2",
			Status:    "todo",
			AgentType: &backendAgent,
			Priority:  5,
		}

		err = taskRepo.Create(ctx, task)
		require.NoError(t, err, "Task creation should succeed")

		// Verify slug was generated and stored
		expectedSlug := slug.Generate(task.Title)
		assert.NotEmpty(t, expectedSlug, "Generated slug should not be empty")
		require.NotNil(t, task.Slug, "Task slug pointer should not be nil")
		assert.Equal(t, expectedSlug, *task.Slug, "Task slug should match generated value")
		assert.Equal(t, "implement-user-authentication-with-oauth2", *task.Slug, "Slug should be URL-friendly")

		// Verify slug is persisted in database
		retrieved, err := taskRepo.GetByID(ctx, task.ID)
		require.NoError(t, err, "Task retrieval should succeed")
		require.NotNil(t, retrieved.Slug, "Retrieved task slug pointer should not be nil")
		assert.Equal(t, expectedSlug, *retrieved.Slug, "Retrieved task slug should match stored value")

		defer func() {
			if _, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}()
		defer func() {
			if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}()
		defer func() {
			if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}()
	})
}

// TestSlugStorage_EmptyTitle validates handling of empty or invalid titles
func TestSlugStorage_EmptyTitle(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)
	taskRepo := NewTaskRepository(db)

	t.Run("Task with special-chars-only title gets empty slug", func(t *testing.T) {
		// Create parent epic and feature
		epic := &models.Epic{
			Key:      "E99",
			Title:    "Parent Epic",
			Status:   "active",
			Priority: "high",
		}
		err := epicRepo.Create(ctx, epic)
		require.NoError(t, err)

		feature := &models.Feature{
			EpicID: epic.ID,
			Key:    "E99-F01",
			Title:  "Parent Feature",
			Status: "active",
		}
		err = featureRepo.Create(ctx, feature)
		require.NoError(t, err)

		generalAgent := "general"
		task := &models.Task{
			FeatureID: feature.ID,
			Key:       "T-E99-F99-999",
			Title:     "!@#$%^&*()", // Only special characters
			Status:    "todo",
			AgentType: &generalAgent,
			Priority:  5,
		}

		err = taskRepo.Create(ctx, task)
		require.NoError(t, err, "Task creation should succeed even with special-chars-only title")

		// Verify empty slug is stored (slug generation returns empty for invalid titles)
		require.NotNil(t, task.Slug, "Slug pointer should not be nil")
		assert.Empty(t, *task.Slug, "Slug should be empty for special-chars-only title")

		// Verify empty slug is persisted
		retrieved, err := taskRepo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved.Slug, "Retrieved slug pointer should not be nil")
		assert.Empty(t, *retrieved.Slug, "Retrieved task should have empty slug")

		defer func() {
			if _, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}()
		defer func() {
			if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}()
		defer func() {
			if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}()
	})
}

// TestSlugStorage_UnicodeHandling validates unicode normalization in slugs
func TestSlugStorage_UnicodeHandling(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E98', 'E99')")

	epicRepo := NewEpicRepository(db)

	t.Run("Epic with unicode characters normalizes correctly", func(t *testing.T) {
		epic := &models.Epic{
			Key:      "E99",
			Title:    "Améliorer la sécurité avec façade", // French with accents
			Status:   "active",
			Priority: "high",
		}

		err := epicRepo.Create(ctx, epic)
		require.NoError(t, err)

		// Verify unicode is normalized (accents removed)
		require.NotNil(t, epic.Slug, "Slug pointer should not be nil")
		assert.Equal(t, "ameliorer-la-securite-avec-facade", *epic.Slug, "Unicode characters should be normalized")

		// Verify normalization is persisted
		retrieved, err := epicRepo.GetByID(ctx, epic.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved.Slug, "Retrieved slug pointer should not be nil")
		assert.Equal(t, "ameliorer-la-securite-avec-facade", *retrieved.Slug)

		defer func() {
			if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}()
	})
}

// TestSlugStorage_Uniqueness validates that slugs don't need to be unique
// (keys handle uniqueness, slugs are for readability only)
func TestSlugStorage_Uniqueness(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E98', 'E97')")

	epicRepo := NewEpicRepository(db)

	t.Run("Multiple epics can have same slug", func(t *testing.T) {
		epic1 := &models.Epic{
			Key:      "E98",
			Title:    "User Authentication", // Same title -> same slug
			Status:   "active",
			Priority: "high",
		}
		err := epicRepo.Create(ctx, epic1)
		require.NoError(t, err)

		epic2 := &models.Epic{
			Key:      "E97",
			Title:    "User Authentication", // Same title -> same slug
			Status:   "active",
			Priority: "high",
		}
		err = epicRepo.Create(ctx, epic2)
		require.NoError(t, err, "Second epic with same slug should be allowed (slugs don't need uniqueness)")

		assert.Equal(t, epic1.Slug, epic2.Slug, "Both epics should have same slug")
		assert.NotEqual(t, epic1.Key, epic2.Key, "But different keys (keys must be unique)")

		defer func() {
			if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id IN (?, ?)", epic1.ID, epic2.ID); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}()
	})
}

// TestSlugStorage_Retrieval validates that slugs can be used for path resolution
func TestSlugStorage_Retrieval(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)
	taskRepo := NewTaskRepository(db)

	// Setup: Create epic -> feature -> task hierarchy
	epic := &models.Epic{
		Key:      "E99",
		Title:    "Core Infrastructure",
		Status:   "active",
		Priority: "high",
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E99-F01",
		Title:  "Database Optimization",
		Status: "active",
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	backendAgent := "backend"
	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E99-F01-001",
		Title:     "Add database indexes for performance",
		Status:    "todo",
		AgentType: &backendAgent,
		Priority:  8,
	}
	err = taskRepo.Create(ctx, task)
	require.NoError(t, err)

	t.Run("Query slug values for path resolution", func(t *testing.T) {
		// Simulate PathResolver pattern: query database for slug without reading files
		var epicSlug, featureSlug, taskSlug sql.NullString

		err := database.QueryRowContext(ctx, "SELECT slug FROM epics WHERE key = ?", epic.Key).Scan(&epicSlug)
		require.NoError(t, err)
		assert.True(t, epicSlug.Valid, "Epic slug should be non-null")
		assert.Equal(t, "core-infrastructure", epicSlug.String)

		err = database.QueryRowContext(ctx, "SELECT slug FROM features WHERE key = ?", feature.Key).Scan(&featureSlug)
		require.NoError(t, err)
		assert.True(t, featureSlug.Valid, "Feature slug should be non-null")
		assert.Equal(t, "database-optimization", featureSlug.String)

		err = database.QueryRowContext(ctx, "SELECT slug FROM tasks WHERE key = ?", task.Key).Scan(&taskSlug)
		require.NoError(t, err)
		assert.True(t, taskSlug.Valid, "Task slug should be non-null")
		assert.Equal(t, "add-database-indexes-for-performance", taskSlug.String)

		// Verify no file I/O needed - all slug data available from database
		// This is the core benefit: PathResolver can build paths without reading files
	})

	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()
}
