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
	epicRepo := NewEpicRepository(db)

	// Clean up test data first (use E89 to avoid conflict with E90-E99 range used by progress tests)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E89-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E89'")

	// Create dedicated epic for this test
	testEpic := &models.Epic{
		Key:      "E89",
		Title:    "Test Epic for Feature Slug",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create feature
	feature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E89-F01",
		Title:  "Implement User Authentication System",
		Status: models.FeatureStatusDraft,
	}

	err = repo.Create(ctx, feature)
	require.NoError(t, err)
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Verify slug was generated and stored
	assert.NotNil(t, feature.Slug, "Slug should be generated")
	assert.Equal(t, "implement-user-authentication-system", *feature.Slug)

	// Verify slug is persisted in database
	retrieved, err := repo.GetByKey(ctx, "E89-F01")
	require.NoError(t, err)
	assert.NotNil(t, retrieved.Slug, "Slug should be persisted")
	assert.Equal(t, "implement-user-authentication-system", *retrieved.Slug)
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

// TestFeatureRepository_GetByKey_NumericAndSluggedKeys verifies dual key lookup support
func TestFeatureRepository_GetByKey_NumericAndSluggedKeys(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E97-F15'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")

	// Create test epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E97",
		Title:         "Test Epic for Dual Key Lookup",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	// Create test feature with slug
	feature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E97-F15",
		Title:  "User Authentication Feature",
		Status: models.FeatureStatusDraft,
	}
	err = repo.Create(ctx, feature)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID) }()

	// Verify slug was generated
	require.NotNil(t, feature.Slug)
	expectedSlug := "user-authentication-feature"
	assert.Equal(t, expectedSlug, *feature.Slug)

	testCases := []struct {
		name        string
		queryKey    string
		shouldFind  bool
		description string
	}{
		{
			name:        "Full key lookup",
			queryKey:    "E97-F15",
			shouldFind:  true,
			description: "Standard full key (E97-F15) should work",
		},
		{
			name:        "Numeric key only",
			queryKey:    "F15",
			shouldFind:  true,
			description: "Numeric key (F15) should work",
		},
		{
			name:        "Lowercase numeric key",
			queryKey:    "f15",
			shouldFind:  true,
			description: "Lowercase numeric key (f15) should work",
		},
		{
			name:        "Slugged key with dash",
			queryKey:    "F15-user-authentication-feature",
			shouldFind:  true,
			description: "Slugged key (F15-user-authentication-feature) should work",
		},
		{
			name:        "Lowercase slugged key",
			queryKey:    "f15-user-authentication-feature",
			shouldFind:  true,
			description: "Lowercase slugged key (f15-user-authentication-feature) should work",
		},
		{
			name:        "Full key with slug",
			queryKey:    "E97-F15-user-authentication-feature",
			shouldFind:  true,
			description: "Full key with slug (E97-F15-user-authentication-feature) should work",
		},
		{
			name:        "Invalid key",
			queryKey:    "F88",
			shouldFind:  false,
			description: "Non-existent key (F88) should not find anything",
		},
		{
			name:        "Invalid key with different number",
			queryKey:    "F25",
			shouldFind:  false,
			description: "Non-existent key (F25) should not find anything",
		},
		{
			name:        "Wrong slug",
			queryKey:    "F15-wrong-slug",
			shouldFind:  false,
			description: "Wrong slug (F15-wrong-slug) should not find anything",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := repo.GetByKey(ctx, tc.queryKey)

			if tc.shouldFind {
				require.NoError(t, err, "GetByKey(%s) should succeed: %s", tc.queryKey, tc.description)
				require.NotNil(t, result, "GetByKey(%s) should return feature: %s", tc.queryKey, tc.description)
				assert.Equal(t, "E97-F15", result.Key, "Should return correct feature")
				assert.Equal(t, "User Authentication Feature", result.Title)
			} else {
				if err == nil && result != nil {
					t.Logf("DEBUG: Unexpected result for %s: key=%s, title=%s", tc.queryKey, result.Key, result.Title)
				}
				require.Error(t, err, "GetByKey(%s) should fail: %s", tc.queryKey, tc.description)
				assert.Nil(t, result, "GetByKey(%s) should not return feature: %s", tc.queryKey, tc.description)
			}
		})
	}
}

// TestFeatureRepository_GetByKey_MultipleFeaturesSameEpic verifies numeric key resolves correctly when multiple features exist
func TestFeatureRepository_GetByKey_MultipleFeaturesSameEpic(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key IN ('E96-F20', 'E96-F21')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E96'")

	// Create test epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E96",
		Title:         "Test Epic for Multiple Features",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	// Create two features with same numeric part but different epic
	feature1 := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E96-F20",
		Title:  "First Feature",
		Status: models.FeatureStatusDraft,
	}
	err = repo.Create(ctx, feature1)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature1.ID) }()

	feature2 := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E96-F21",
		Title:  "Second Feature",
		Status: models.FeatureStatusDraft,
	}
	err = repo.Create(ctx, feature2)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature2.ID) }()

	// Test numeric key lookup for F20
	result, err := repo.GetByKey(ctx, "F20")
	require.NoError(t, err)
	assert.Equal(t, "E96-F20", result.Key)
	assert.Equal(t, "First Feature", result.Title)

	// Test numeric key lookup for F21
	result, err = repo.GetByKey(ctx, "F21")
	require.NoError(t, err)
	assert.Equal(t, "E96-F21", result.Key)
	assert.Equal(t, "Second Feature", result.Title)

	// Test slugged key lookup
	if feature1.Slug != nil {
		result, err = repo.GetByKey(ctx, "F20-"+*feature1.Slug)
		require.NoError(t, err)
		assert.Equal(t, "E96-F20", result.Key)
	}
}

// TestFeatureRepository_UpdateCustomPath removed - custom_folder_path feature no longer supported
