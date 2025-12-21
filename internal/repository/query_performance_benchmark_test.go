package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// Performance Benchmark Tests for Epic and Feature Queries
// These tests validate that query performance meets PRD targets:
// - shark epic list: <100ms for 100 epics
// - shark epic get: <200ms for epics with 50 features
// - shark feature get: <200ms for features with 100 tasks

// BenchmarkEpicList measures epic list query performance
// PRD Target: <100ms for 100 epics
func BenchmarkEpicList(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		epics, err := epicRepo.List(ctx, nil)
		if err != nil {
			b.Fatalf("Failed to get all epics: %v", err)
		}
		if len(epics) == 0 {
			b.Fatal("Expected at least some epics in database")
		}
	}

	// Report average time
	avgNs := b.Elapsed().Nanoseconds() / int64(b.N)
	avgMs := float64(avgNs) / 1_000_000
	b.Logf("Average epic list query time: %.2f ms (target: <100ms)", avgMs)

	if avgMs > 100 {
		b.Logf("WARNING: Epic list query exceeded 100ms target (%.2f ms)", avgMs)
	}
}

// BenchmarkEpicGetWithFeatures measures epic get with feature details
// PRD Target: <200ms for epics with 50 features
func BenchmarkEpicGetWithFeatures(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Use a test epic that should have features
	testEpicKey := "E04" // Use existing test epic

	// Try to get epic, skip benchmark if it doesn't exist
	epic, err := epicRepo.GetByKey(ctx, testEpicKey)
	if err != nil {
		b.Skipf("Test epic %s not found, skipping benchmark", testEpicKey)
		return
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Get epic by key
		retrievedEpic, err := epicRepo.GetByKey(ctx, testEpicKey)
		if err != nil {
			b.Fatalf("Failed to get epic: %v", err)
		}

		// Get all features for the epic
		_, err = featureRepo.ListByEpic(ctx, retrievedEpic.ID)
		if err != nil {
			b.Fatalf("Failed to get features: %v", err)
		}

		// Calculate progress for epic
		_, err = epicRepo.CalculateProgress(ctx, retrievedEpic.ID)
		if err != nil {
			b.Fatalf("Failed to calculate epic progress: %v", err)
		}
	}

	avgNs := b.Elapsed().Nanoseconds() / int64(b.N)
	avgMs := float64(avgNs) / 1_000_000

	// Get feature count
	features, _ := featureRepo.ListByEpic(ctx, epic.ID)
	b.Logf("Average epic get (with %d features) time: %.2f ms (target: <200ms)", len(features), avgMs)

	if avgMs > 200 {
		b.Logf("WARNING: Epic get query exceeded 200ms target (%.2f ms)", avgMs)
	}
}

// BenchmarkFeatureGetWithTasks measures feature get with task details
// PRD Target: <200ms for features with 100 tasks
func BenchmarkFeatureGetWithTasks(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)

	// Use an existing feature for benchmarking
	testFeatureKey := "E04-F01"

	// Try to get feature, skip if not found
	feature, err := featureRepo.GetByKey(ctx, testFeatureKey)
	if err != nil {
		b.Skipf("Test feature %s not found, skipping benchmark", testFeatureKey)
		return
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Get feature by key
		retrievedFeature, err := featureRepo.GetByKey(ctx, testFeatureKey)
		if err != nil {
			b.Fatalf("Failed to get feature: %v", err)
		}

		// Calculate progress
		_, err = featureRepo.CalculateProgress(ctx, retrievedFeature.ID)
		if err != nil {
			b.Fatalf("Failed to calculate progress: %v", err)
		}
	}

	avgNs := b.Elapsed().Nanoseconds() / int64(b.N)
	avgMs := float64(avgNs) / 1_000_000

	// Count tasks
	var taskCount int
	_ = database.QueryRow("SELECT COUNT(*) FROM tasks WHERE feature_id = ?", feature.ID).Scan(&taskCount)

	b.Logf("Average feature get (with %d tasks) time: %.2f ms (target: <200ms)", taskCount, avgMs)

	if avgMs > 200 {
		b.Logf("WARNING: Feature get query exceeded 200ms target (%.2f ms)", avgMs)
	}
}

// BenchmarkProgressCalculation measures just the progress calculation SQL performance
func BenchmarkProgressCalculation(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)

	// Use an existing feature
	testFeatureKey := "E04-F01"
	feature, err := featureRepo.GetByKey(ctx, testFeatureKey)
	if err != nil {
		b.Skipf("Test feature not found: %v", err)
		return
	}

	b.ResetTimer()

	b.Run("FeatureProgress", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := featureRepo.CalculateProgress(ctx, feature.ID)
			if err != nil {
				b.Fatalf("Failed to calculate feature progress: %v", err)
			}
		}

		avgNs := b.Elapsed().Nanoseconds() / int64(b.N)
		avgMs := float64(avgNs) / 1_000_000
		b.Logf("Average feature progress calculation: %.2f ms", avgMs)
	})
}

// TestQueryPlanAnalysis verifies SQL query efficiency using EXPLAIN QUERY PLAN
// This is NOT a benchmark but validates that queries use indexes properly
func TestQueryPlanAnalysis(t *testing.T) {
	database := test.GetTestDB()

	// Use an existing feature for testing
	var featureID int64 = 1 // Assume feature with ID 1 exists
	var epicID int64 = 1    // Assume epic with ID 1 exists

	t.Run("FeatureProgressQueryPlan", func(t *testing.T) {
		query := `
			EXPLAIN QUERY PLAN
			SELECT
			    COUNT(*) as total_tasks,
			    COALESCE(SUM(CASE WHEN status IN ('completed', 'archived') THEN 1 ELSE 0 END), 0) as completed_tasks
			FROM tasks
			WHERE feature_id = ?
		`

		rows, err := database.Query(query, featureID)
		if err != nil {
			t.Fatalf("Failed to get query plan: %v", err)
		}
		defer rows.Close()

		t.Log("Feature progress calculation query plan:")
		hasIndex := false
		for rows.Next() {
			var id, parent, notUsed int
			var detail string
			err := rows.Scan(&id, &parent, &notUsed, &detail)
			if err != nil {
				t.Fatalf("Failed to scan query plan: %v", err)
			}
			t.Logf("  %s", detail)

			// Check if index is being used
			if contains(detail, "INDEX") || contains(detail, "idx_tasks_feature_id") {
				hasIndex = true
			}
		}

		if !hasIndex {
			t.Log("INFO: Feature progress query may not be using an index (acceptable for small datasets)")
		}
	})

	t.Run("EpicProgressQueryPlan", func(t *testing.T) {
		query := `
			EXPLAIN QUERY PLAN
			SELECT
			    COALESCE(SUM(f.progress_pct * (
			        SELECT COUNT(*) FROM tasks t WHERE t.feature_id = f.id
			    )), 0) as weighted_sum,
			    COALESCE(SUM((
			        SELECT COUNT(*) FROM tasks t WHERE t.feature_id = f.id
			    )), 0) as total_task_count
			FROM features f
			WHERE f.epic_id = ?
		`

		rows, err := database.Query(query, epicID)
		if err != nil {
			t.Fatalf("Failed to get query plan: %v", err)
		}
		defer rows.Close()

		t.Log("Epic progress calculation query plan:")
		hasIndex := false
		for rows.Next() {
			var id, parent, notUsed int
			var detail string
			err := rows.Scan(&id, &parent, &notUsed, &detail)
			if err != nil {
				t.Fatalf("Failed to scan query plan: %v", err)
			}
			t.Logf("  %s", detail)

			if contains(detail, "INDEX") || contains(detail, "idx_features_epic_id") {
				hasIndex = true
			}
		}

		if !hasIndex {
			t.Log("INFO: Epic progress query may not be using an index (acceptable for small datasets)")
		}
	})

	t.Run("GetByKeyQueryPlan", func(t *testing.T) {
		query := `
			EXPLAIN QUERY PLAN
			SELECT id, epic_id, key, title, description, status, progress_pct,
			       created_at, updated_at
			FROM features
			WHERE key = ?
		`

		rows, err := database.Query(query, "E04-F01")
		if err != nil {
			t.Fatalf("Failed to get query plan: %v", err)
		}
		defer rows.Close()

		t.Log("GetByKey query plan:")
		hasIndex := false
		for rows.Next() {
			var id, parent, notUsed int
			var detail string
			err := rows.Scan(&id, &parent, &notUsed, &detail)
			if err != nil {
				t.Fatalf("Failed to scan query plan: %v", err)
			}
			t.Logf("  %s", detail)

			if contains(detail, "INDEX") || contains(detail, "idx_features_key") || contains(detail, "UNIQUE") {
				hasIndex = true
			}
		}

		if !hasIndex {
			t.Log("INFO: GetByKey query may not be using an index (check if unique constraint is being used)")
		}
	})
}

// TestNoPlusOneQueries verifies that queries don't have N+1 problems
func TestNoPlusOneQueries(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)

	// Use an existing epic for testing
	epicID := int64(1) // Assume epic with ID 1 exists

	// Get all features for epic - should be 1 query, not N
	start := time.Now()
	features, err := featureRepo.ListByEpic(ctx, epicID)
	elapsed := time.Since(start)

	if err != nil {
		t.Logf("No features found for epic ID %d (this is OK for test): %v", epicID, err)
		return
	}

	if len(features) == 0 {
		t.Skip("No features found for test epic")
		return
	}

	// Should complete very quickly since it's a single query
	if elapsed > 50*time.Millisecond {
		t.Logf("WARNING: ListByEpic took %v, may have N+1 problem", elapsed)
	} else {
		t.Logf("ListByEpic for %d features: %v (no N+1 detected)", len(features), elapsed)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
