package feature

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/repository/epic"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFeatureRepository_Create_GeneratesAndStoresSlug verifies slug generation during feature creation
func TestFeatureRepository_Create_GeneratesAndStoresSlug(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Clean up test data first (use E89 to avoid conflict with E90-E99 range used by progress tests)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E89-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E89'")

	// Create dedicated epic for this test
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E89",
		Title: "Test Epic for Feature Slug"}, Status: models.EpicStatusActive,
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
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E89-F01",
		Title: "Implement User Authentication System"}, EpicID: testEpic.ID,

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

// TestFeatureRepository_GetByKey_WithTaskKeySuffix verifies that GetByKey handles task-key-style
// inputs like "E88-F01-015" by returning the parent feature "E88-F01" instead of failing.
// This fixes the bug where `shark feature get E15-F11-015` returned "no rows in result set"
// because "015" was treated as a slug instead of a task number suffix.
func TestFeatureRepository_GetByKey_WithTaskKeySuffix(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Clean up test data first (use E88 - not used by other tests)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E88-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E88'")

	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E88",
		Title: "Test Epic for Task Key Suffix"}, Status: models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err)
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID)
	}()

	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E88-F01",
		Title: "My Feature"}, EpicID: testEpic.ID,

		Status: models.FeatureStatusActive,
	}
	err = repo.Create(ctx, feature)
	require.NoError(t, err)
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	}()

	tests := []struct {
		name        string
		key         string
		wantFeature string // expected feature key
	}{
		{
			name:        "task key format E88-F01-001 returns parent feature",
			key:         "E88-F01-001",
			wantFeature: "E88-F01",
		},
		{
			name:        "task key format E88-F01-015 returns parent feature",
			key:         "E88-F01-015",
			wantFeature: "E88-F01",
		},
		{
			name:        "task key format E88-F01-999 returns parent feature",
			key:         "E88-F01-999",
			wantFeature: "E88-F01",
		},
		{
			name:        "case insensitive task key format e88-f01-001",
			key:         "e88-f01-001",
			wantFeature: "E88-F01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.GetByKey(ctx, tt.key)
			require.NoError(t, err, "GetByKey(%q) should return parent feature, not error", tt.key)
			assert.Equal(t, tt.wantFeature, result.Key)
		})
	}

	t.Run("non-numeric suffix is still treated as slug", func(t *testing.T) {
		// "E88-F01-somefeature" should fail (it's a slug lookup for a different feature)
		_, err := repo.GetByKey(ctx, "E88-F01-somefeature")
		assert.Error(t, err, "GetByKey with non-matching slug should return error")
	})
}

// TestFeatureRepository_Create_SlugHandlesSpecialCharacters verifies slug handles special characters
func TestFeatureRepository_Create_SlugHandlesSpecialCharacters(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key IN ('E98-F10', 'E98-F11', 'E98-F12')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create a dedicated test epic for this test
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E98",
		Title: "Test Epic for Slug Special Characters"}, Status: models.EpicStatusActive,
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
		feature := &models.Feature{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("E98-F%02d", 10+i),
			Title: tc.title}, EpicID: epicID,

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
	db := dbconn.NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E97-F15'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")

	// Create test epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E97",
		Title: "Test Epic for Dual Key Lookup"}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	// Create test feature with slug
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E97-F15",
		Title: "User Authentication Feature"}, EpicID: testEpic.ID,

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
	db := dbconn.NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key IN ('E96-F20', 'E96-F21')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E96'")

	// Create test epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E96",
		Title: "Test Epic for Multiple Features"}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	// Create two features with same numeric part but different epic
	feature1 := &models.Feature{BaseEntity: models.BaseEntity{Key: "E96-F20",
		Title: "First Feature"}, EpicID: testEpic.ID,

		Status: models.FeatureStatusDraft,
	}
	err = repo.Create(ctx, feature1)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature1.ID) }()

	feature2 := &models.Feature{BaseEntity: models.BaseEntity{Key: "E96-F21",
		Title: "Second Feature"}, EpicID: testEpic.ID,

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

// TestFeatureRepository_UpdateStatus verifies atomic status updates via UpdateStatus.
func TestFeatureRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Clean up test data first (use E87 to avoid conflict with other tests)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E87-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E87'")

	// Create dedicated epic for this test
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E87",
		Title: "Test Epic for UpdateStatus"}, Status: models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID)
	}()

	// Create feature with initial status
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E87-F01",
		Title: "Feature for Status Update Test"}, EpicID: testEpic.ID,
		Status: models.FeatureStatusDraft,
	}
	err = repo.Create(ctx, feature)
	require.NoError(t, err)
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	}()

	t.Run("successful status update", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, feature.ID, models.FeatureStatusActive)
		require.NoError(t, err)

		// Verify status was updated
		updated, err := repo.GetByID(ctx, feature.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FeatureStatusActive, updated.Status)
	})

	t.Run("update to completed status", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, feature.ID, models.FeatureStatusCompleted)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, feature.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FeatureStatusCompleted, updated.Status)
	})

	t.Run("non-existent feature returns error", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, 999999, models.FeatureStatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "feature not found")
	})
}

// TestFeatureRepository_UpdateCustomPath removed - custom_folder_path feature no longer supported

// TestFeatureRepository_UpdateCascadesOrder verifies that updating a feature's execution order
// automatically resequences all other features in the same epic
func TestFeatureRepository_UpdateCascadesOrder(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	featureRepo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E99-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

	// Create test epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E99",
		Title: "Test Epic for Feature Order Cascade"}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	// Create four features with sequential orders: a-1, b-2, c-3, d-4
	order1, order2, order3, order4 := 1, 2, 3, 4
	featureA := &models.Feature{BaseEntity: models.BaseEntity{Key: "E99-F01",
		Title: "Feature A"}, EpicID: testEpic.ID,

		Status:         models.FeatureStatusDraft,
		ExecutionOrder: &order1,
	}
	featureB := &models.Feature{BaseEntity: models.BaseEntity{Key: "E99-F02",
		Title: "Feature B"}, EpicID: testEpic.ID,

		Status:         models.FeatureStatusDraft,
		ExecutionOrder: &order2,
	}
	featureC := &models.Feature{BaseEntity: models.BaseEntity{Key: "E99-F03",
		Title: "Feature C"}, EpicID: testEpic.ID,

		Status:         models.FeatureStatusDraft,
		ExecutionOrder: &order3,
	}
	featureD := &models.Feature{BaseEntity: models.BaseEntity{Key: "E99-F04",
		Title: "Feature D"}, EpicID: testEpic.ID,

		Status:         models.FeatureStatusDraft,
		ExecutionOrder: &order4,
	}

	err = featureRepo.Create(ctx, featureA)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureA.ID) }()

	err = featureRepo.Create(ctx, featureB)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureB.ID) }()

	err = featureRepo.Create(ctx, featureC)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureC.ID) }()

	err = featureRepo.Create(ctx, featureD)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureD.ID) }()

	// When: Update feature D's order from 4 to 2
	newOrder := 2
	featureD.ExecutionOrder = &newOrder
	err = featureRepo.Update(ctx, featureD)
	require.NoError(t, err, "Failed to update feature D's order")

	// Then: Verify cascade - expected order: a-1, d-2, b-3, c-4
	// Get all features for this epic
	features, err := featureRepo.ListByEpic(ctx, testEpic.ID)
	require.NoError(t, err, "Failed to list features by epic ID")
	require.Len(t, features, 4, "Should have 4 features")

	// Build a map for easy verification
	featureOrders := make(map[string]int)
	for _, feature := range features {
		if feature.ExecutionOrder != nil {
			featureOrders[feature.Title] = *feature.ExecutionOrder
		}
	}

	// Verify expected orders
	assert.Equal(t, 1, featureOrders["Feature A"], "Feature A should be at order 1")
	assert.Equal(t, 2, featureOrders["Feature D"], "Feature D should be at order 2 (moved)")
	assert.Equal(t, 3, featureOrders["Feature B"], "Feature B should be at order 3 (shifted)")
	assert.Equal(t, 4, featureOrders["Feature C"], "Feature C should be at order 4 (shifted)")
}

// TestFeatureRepository_UpdateNoResequence_PreservesDuplicateOrders verifies that
// UpdateNoResequence sets the target feature's execution_order without renumbering
// siblings, allowing intentional duplicate-order groups (parallel work) to be
// formed via `shark feature update --parallel`.
func TestFeatureRepository_UpdateNoResequence_PreservesDuplicateOrders(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	featureRepo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E96-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E96'")

	// Create test epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E96",
		Title: "Test Epic for Feature No-Resequence"}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	require.NoError(t, epicRepo.Create(ctx, testEpic), "Failed to create test epic")
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	// Seed features at orders 1, 2, 3
	order1, order2, order3 := 1, 2, 3
	featureA := &models.Feature{BaseEntity: models.BaseEntity{Key: "E96-F01", Title: "Feature A"},
		EpicID: testEpic.ID, Status: models.FeatureStatusDraft, ExecutionOrder: &order1}
	featureB := &models.Feature{BaseEntity: models.BaseEntity{Key: "E96-F02", Title: "Feature B"},
		EpicID: testEpic.ID, Status: models.FeatureStatusDraft, ExecutionOrder: &order2}
	featureC := &models.Feature{BaseEntity: models.BaseEntity{Key: "E96-F03", Title: "Feature C"},
		EpicID: testEpic.ID, Status: models.FeatureStatusDraft, ExecutionOrder: &order3}
	require.NoError(t, featureRepo.Create(ctx, featureA))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureA.ID) }()
	require.NoError(t, featureRepo.Create(ctx, featureB))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureB.ID) }()
	require.NoError(t, featureRepo.Create(ctx, featureC))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureC.ID) }()

	// When: move Feature C to order=1 WITHOUT cascade
	newOrder := 1
	featureC.ExecutionOrder = &newOrder
	require.NoError(t, featureRepo.UpdateNoResequence(ctx, featureC), "UpdateNoResequence should succeed")

	// Then: A and C share order=1 (parallel batch); B remains at 2
	features, err := featureRepo.ListByEpic(ctx, testEpic.ID)
	require.NoError(t, err, "Failed to list features by epic ID")
	require.Len(t, features, 3, "Should have 3 features")

	featureOrders := make(map[string]int)
	for _, feature := range features {
		require.NotNil(t, feature.ExecutionOrder, "Feature %s should have an execution order", feature.Title)
		featureOrders[feature.Title] = *feature.ExecutionOrder
	}

	assert.Equal(t, 1, featureOrders["Feature A"], "Feature A should remain at order 1")
	assert.Equal(t, 2, featureOrders["Feature B"], "Feature B should remain at order 2 (no cascade)")
	assert.Equal(t, 1, featureOrders["Feature C"], "Feature C should be at order 1 alongside Feature A")
}

// TestFeatureRepository_UpdateNoResequence_FastPath verifies the TD-008 fast
// path: the --parallel feature update lands the new row state and drains the
// connection pool.
//
// NOTE: db.Stats().InUse==0 holds for both the tx and non-tx paths, so this
// does not prove a transaction was never opened — it is a leak/post-condition
// check. Proving "no BeginTx" would require a driver wrapper (out of scope).
func TestFeatureRepository_UpdateNoResequence_FastPath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	featureRepo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E94-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E94'")

	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E94",
		Title: "Test Epic Feature TD-008 NoTx"}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	order1 := 1
	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E94-F01", Title: "Feature NoTx"},
		EpicID: testEpic.ID, Status: models.FeatureStatusDraft, ExecutionOrder: &order1}
	require.NoError(t, featureRepo.Create(ctx, testFeature))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	statsBefore := database.Stats()

	// Exercise the fast path.
	newOrder := 7
	testFeature.ExecutionOrder = &newOrder
	require.NoError(t, featureRepo.UpdateNoResequence(ctx, testFeature))

	statsAfter := database.Stats()

	// Post-condition: no connection left in use (true for any clean exit, tx or
	// not — a leak check, not a proof that BEGIN/COMMIT is gone; see the doc).
	assert.Equal(t, 0, statsAfter.InUse, "no connection should be in-use after UpdateNoResequence returns")
	assert.GreaterOrEqual(t, statsAfter.OpenConnections, statsBefore.OpenConnections,
		"OpenConnections should not have decreased")

	got, err := featureRepo.GetByKey(ctx, "E94-F01")
	require.NoError(t, err)
	require.NotNil(t, got.ExecutionOrder)
	assert.Equal(t, 7, *got.ExecutionOrder)
}

// TestFeatureRepository_UpdateStatusTx tests the transactional status update method.
func TestFeatureRepository_UpdateStatusTx(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Create a dedicated epic for this test
	testEpic := &models.Epic{
		BaseEntity: models.BaseEntity{Key: "E88", Title: "Epic for UpdateStatusTx Feature Test"},
		Status:     models.EpicStatusActive,
		Priority:   models.PriorityMedium,
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E88-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E88'")
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE epic_id = ?", testEpic.ID)
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID)
	}()

	t.Run("commits_status_update", func(t *testing.T) {
		feature := &models.Feature{
			BaseEntity: models.BaseEntity{Key: fmt.Sprintf("E88-F%02d", 1), Title: "Feature UpdateStatusTx Commit"},
			EpicID:     testEpic.ID,
			Status:     models.FeatureStatusDraft,
		}
		require.NoError(t, repo.Create(ctx, feature))
		defer func() {
			_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
		}()

		tx, err := database.BeginTx(ctx, nil)
		require.NoError(t, err)

		err = repo.UpdateStatusTx(ctx, tx, feature.ID, "active", nil, nil)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())

		// Verify the update was committed
		updated, err := repo.GetByID(ctx, feature.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FeatureStatus("active"), updated.Status)
	})

	t.Run("rollback_restores_original_status", func(t *testing.T) {
		feature := &models.Feature{
			BaseEntity: models.BaseEntity{Key: fmt.Sprintf("E88-F%02d", 2), Title: "Feature UpdateStatusTx Rollback"},
			EpicID:     testEpic.ID,
			Status:     models.FeatureStatusDraft,
		}
		require.NoError(t, repo.Create(ctx, feature))
		defer func() {
			_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
		}()

		tx, err := database.BeginTx(ctx, nil)
		require.NoError(t, err)

		err = repo.UpdateStatusTx(ctx, tx, feature.ID, "active", nil, nil)
		require.NoError(t, err)
		require.NoError(t, tx.Rollback())

		// Verify original status is preserved
		current, err := repo.GetByID(ctx, feature.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FeatureStatusDraft, current.Status)
	})

	t.Run("not_found_returns_error", func(t *testing.T) {
		tx, err := database.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()

		err = repo.UpdateStatusTx(ctx, tx, 999999999, "active", nil, nil)
		assert.Error(t, err, "expected error for non-existent feature ID")
	})
}

// ptrIntFeature returns a pointer to n; helper for size round-trip tests.
func ptrIntFeature(n int) *int { return &n }

// TestFeatureRepository_SizeRoundTrip verifies that Size persists through Create,
// GetByKey, and Update without information loss (TC-F010-B).
func TestFeatureRepository_SizeRoundTrip(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	epicKey := "E97"
	featureKey := "E97-F01"

	// Clean up before test
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = ?", featureKey)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create parent epic
	testEpic := &models.Epic{
		BaseEntity: models.BaseEntity{Key: epicKey, Title: "Size RT Epic"},
		Status:     models.EpicStatusDraft,
		Priority:   models.PriorityMedium,
	}
	require.NoError(t, epicRepo.Create(ctx, testEpic), "Failed to create parent epic")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID)
	}()

	// Step 1: Create with Size = ptr(5)
	feat := &models.Feature{
		BaseEntity: models.BaseEntity{
			Key:   featureKey,
			Title: "Size Round Trip Feature",
			Size:  ptrIntFeature(5),
		},
		EpicID: testEpic.ID,
		Status: models.FeatureStatusDraft,
	}
	require.NoError(t, repo.Create(ctx, feat), "Create() failed")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feat.ID)
	}()

	// Read back and assert Size == 5
	got, err := repo.GetByKey(ctx, featureKey)
	require.NoError(t, err, "GetByKey() failed")
	if got.Size == nil {
		t.Fatal("expected Size to be non-nil after Create")
	}
	assert.Equal(t, 5, *got.Size, "expected Size=5 after Create")

	// Step 2: Update Size = ptr(1)
	got.Size = ptrIntFeature(1)
	require.NoError(t, repo.Update(ctx, got), "Update() failed")

	got2, err := repo.GetByKey(ctx, featureKey)
	require.NoError(t, err, "GetByKey() after update failed")
	if got2.Size == nil {
		t.Fatal("expected Size to be non-nil after Update to 1")
	}
	assert.Equal(t, 1, *got2.Size, "expected Size=1 after update")

	// Step 3: Update Size = nil
	got2.Size = nil
	require.NoError(t, repo.Update(ctx, got2), "Update() to nil failed")

	got3, err := repo.GetByKey(ctx, featureKey)
	require.NoError(t, err, "GetByKey() after nil update failed")
	assert.Nil(t, got3.Size, "expected Size=nil after clearing")
}

// --- GetRecent tests (T-E07-F17-002) ---

// seedFeaturesWithTimestamps creates n features under epicID with created_at staggered
// by 1 second each (oldest first). Uses direct SQL INSERT to bypass key-format validation
// and allow arbitrary timestamps. Returns feature IDs for deferred cleanup.
func seedFeaturesWithTimestamps(t *testing.T, _ *FeatureRepository, db *dbconn.DB, epicID int64, n int) []int64 {
	t.Helper()
	ctx := context.Background()
	baseTime := time.Now().UTC().Add(-time.Duration(n) * time.Second)
	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("E90-F%02d", i+1)
		ts := baseTime.Add(time.Duration(i) * time.Second)
		result, err := db.ExecContext(ctx,
			`INSERT OR IGNORE INTO features (epic_id, key, title, status, created_at, updated_at)
			 VALUES (?, ?, ?, 'draft', ?, CURRENT_TIMESTAMP)`,
			epicID, key, fmt.Sprintf("Recent Feature %d", i+1), ts.Format("2006-01-02T15:04:05Z"),
		)
		require.NoError(t, err, "seedFeaturesWithTimestamps: INSERT failed for key %s", key)
		id, err := result.LastInsertId()
		require.NoError(t, err)
		ids = append(ids, id)
	}
	return ids
}

// TestFeatureRepository_GetRecent_OrdersByCreatedAtDesc seeds 5 features with distinct timestamps
// and asserts that GetRecent returns them in created_at DESC order.
func TestFeatureRepository_GetRecent_OrdersByCreatedAtDesc(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Pre-cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E90-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E90'")

	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E90", Title: "Recent Test Epic"}, Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	ids := seedFeaturesWithTimestamps(t, repo, db, testEpic.ID, 5)
	defer func() {
		for _, id := range ids {
			_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", id)
		}
	}()

	features, err := repo.GetRecent(ctx, 5)
	require.NoError(t, err)
	require.Len(t, features, 5)

	for i := 1; i < len(features); i++ {
		assert.True(t, !features[i-1].CreatedAt.Before(features[i].CreatedAt),
			"expected features[%d].CreatedAt >= features[%d].CreatedAt", i-1, i)
	}
}

// TestFeatureRepository_GetRecent_LimitRespected seeds 10 features and asserts GetRecent(ctx, 3)
// returns exactly 3 rows.
func TestFeatureRepository_GetRecent_LimitRespected(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Pre-cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E90-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E90'")

	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E90", Title: "Recent Test Epic"}, Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	ids := seedFeaturesWithTimestamps(t, repo, db, testEpic.ID, 10)
	defer func() {
		for _, id := range ids {
			_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", id)
		}
	}()

	features, err := repo.GetRecent(ctx, 3)
	require.NoError(t, err)
	assert.Len(t, features, 3, "GetRecent(3) must return exactly 3 rows")
}

// TestFeatureRepository_GetRecent_EmptyTable asserts that GetRecent returns a non-nil
// empty slice (not nil) when no features matching our prefix exist.
func TestFeatureRepository_GetRecent_EmptyTable(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewFeatureRepository(db)

	// Clean up our test prefix to minimize interference
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E90-F%'")

	features, err := repo.GetRecent(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, features, "GetRecent must return a non-nil slice")
}

// TestFeatureRepository_GetRecent_LimitExceedsRowCount seeds 2 features and asserts that
// GetRecent(ctx, 100) returns exactly 2 (all rows, not an error).
func TestFeatureRepository_GetRecent_LimitExceedsRowCount(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewFeatureRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Pre-cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E90-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E90'")

	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E90", Title: "Recent Test Epic"}, Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	ids := seedFeaturesWithTimestamps(t, repo, db, testEpic.ID, 2)
	defer func() {
		for _, id := range ids {
			_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", id)
		}
	}()

	features, err := repo.GetRecent(ctx, 100)
	require.NoError(t, err)
	// At least 2 rows must be returned; there may be more from the test DB.
	assert.GreaterOrEqual(t, len(features), 2, "GetRecent(100) must return all available rows when limit > row count")
}
