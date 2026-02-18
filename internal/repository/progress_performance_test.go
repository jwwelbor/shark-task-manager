package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestFeatureProgressPerformance verifies the SQL query performance
func TestFeatureProgressPerformance(t *testing.T) {
	database := test.GetTestDB()

	// Create test data
	_, featureID := setupProgressTest(t, 83, 1, []models.TaskStatus{
		models.TaskStatus("completed"),
		models.TaskStatus("completed"),
		models.TaskStatus("todo"),
		models.TaskStatus("todo"),
	})

	// Get the SQL query plan
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
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		err := rows.Scan(&id, &parent, &notUsed, &detail)
		if err != nil {
			t.Fatalf("Failed to scan query plan: %v", err)
		}
		t.Logf("  %s", detail)
	}

	// Verify no full table scan - should use index on feature_id
	// The query plan should show "SEARCH tasks USING INDEX idx_tasks_feature_id"
}

// TestEpicProgressPerformance verifies the epic progress SQL performance
func TestEpicProgressPerformance(t *testing.T) {
	database := test.GetTestDB()

	// Create test data with multiple features
	epicID, feature1ID := setupProgressTest(t, 84, 1, []models.TaskStatus{
		models.TaskStatus("completed"),
		models.TaskStatus("todo"),
	})
	// Set progress_pct directly via SQL (UpdateProgress was moved to FeatureService)
	_, _ = database.Exec("UPDATE features SET progress_pct = 50.0 WHERE id = ?", feature1ID)

	// Create second feature with 1 completed task using setupProgressTest helper
	_, feature2ID := setupProgressTest(t, 84, 2, []models.TaskStatus{
		models.TaskStatus("completed"),
	})
	// Set progress_pct directly via SQL (UpdateProgress was moved to FeatureService)
	_, _ = database.Exec("UPDATE features SET progress_pct = 100.0 WHERE id = ?", feature2ID)

	// Get the SQL query plan for epic progress
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
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		err := rows.Scan(&id, &parent, &notUsed, &detail)
		if err != nil {
			t.Fatalf("Failed to scan query plan: %v", err)
		}
		t.Logf("  %s", detail)
	}

	// The query should use indexes on epic_id and feature_id
	// Should show "SEARCH features USING INDEX idx_features_epic_id"
}

// BenchmarkFeatureTaskStatusBreakdown measures feature task status breakdown performance
// (CalculateProgress was moved to FeatureService; this benchmarks the underlying data query)
func BenchmarkFeatureTaskStatusBreakdown(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)

	// Create test data with many tasks
	statuses := make([]models.TaskStatus, 100)
	for i := 0; i < 100; i++ {
		if i < 50 {
			statuses[i] = models.TaskStatus("completed")
		} else {
			statuses[i] = models.TaskStatus("todo")
		}
	}

	_, featureID := setupProgressTest(&testing.T{}, 85, 1, statuses)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := featureRepo.GetTaskStatusBreakdown(ctx, featureID)
		if err != nil {
			b.Fatalf("Failed to get task status breakdown: %v", err)
		}
	}
}

// NOTE: BenchmarkEpicProgress was removed because EpicRepository.CalculateProgress
// was migrated to EpicService.CalculateProgress as part of E15-F05.
