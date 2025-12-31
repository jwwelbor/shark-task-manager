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

// TestFeatureRepository_Create_GeneratesAndStoresSlug verifies slug generation during feature creation
func TestFeatureRepository_Create_GeneratesAndStoresSlug(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewFeatureRepository(db)

	// Seed epic first (INSERT OR IGNORE ensures idempotency)
	epicID, _ := test.SeedTestData()

	// Clean up test data AFTER seeding epic
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E99-F98'")

	// Create feature
	feature := &models.Feature{
		EpicID: epicID,
		Key:    "E99-F98",
		Title:  "Implement User Authentication System",
		Status: models.FeatureStatusDraft,
	}

	err := repo.Create(ctx, feature)
	require.NoError(t, err)

	// Verify slug was generated and stored
	assert.NotNil(t, feature.Slug, "Slug should be generated")
	assert.Equal(t, "implement-user-authentication-system", *feature.Slug)

	// Verify slug is persisted in database
	retrieved, err := repo.GetByID(ctx, feature.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.Slug, "Slug should be persisted")
	assert.Equal(t, "implement-user-authentication-system", *retrieved.Slug)

	// Cleanup
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()
}

// TestFeatureRepository_Create_SlugHandlesSpecialCharacters verifies slug handles special characters
func TestFeatureRepository_Create_SlugHandlesSpecialCharacters(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key IN ('E98-F10', 'E98-F11', 'E98-F12')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create a dedicated test epic for this test
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E98",
		Title:         "Test Epic for Slug Special Characters",
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

	epicID := testEpic.ID
	t.Logf("Using epicID: %d", epicID)

	testCases := []struct {
		title        string
		expectedSlug string
	}{
		{
			title:        "Fix Bug: API Endpoint (v2.1)",
			expectedSlug: "fix-bug-api-endpoint-v2-1",
		},
		{
			title:        "Upgrade PostgreSQL -> MongoDB",
			expectedSlug: "upgrade-postgresql-mongodb",
		},
		{
			title:        "Add Support for UTF-8 & Unicode",
			expectedSlug: "add-support-for-utf-8-unicode",
		},
	}

	for i, tc := range testCases {
		feature := &models.Feature{
			EpicID: epicID,
			Key:    fmt.Sprintf("E98-F%02d", 10+i),
			Title:  tc.title,
			Status: models.FeatureStatusDraft,
		}

		err := repo.Create(ctx, feature)
		require.NoError(t, err, "Failed to create feature with key %s, title: %s", feature.Key, tc.title)

		assert.NotNil(t, feature.Slug, "Slug should be generated for: %s", tc.title)
		assert.Equal(t, tc.expectedSlug, *feature.Slug, "Slug mismatch for: %s", tc.title)

		// Cleanup
		defer func(id int64) {
			if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", id); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}(feature.ID)
	}
}
