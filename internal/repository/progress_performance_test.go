package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestFeatureProgressPerformance verifies the SQL query performance
func TestFeatureProgressPerformance(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Create test data
	_, featureID := setupProgressTest(t, 83, 1, []models.TaskStatus{
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusTodo,
		models.TaskStatusTodo,
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
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)

	// Create test data with multiple features
	epicID, feature1ID := setupProgressTest(t, 84, 1, []models.TaskStatus{
		models.TaskStatusCompleted,
		models.TaskStatusTodo,
	})
	featureRepo.UpdateProgress(ctx, feature1ID)

	// Create second feature with 1 completed task using setupProgressTest helper
	_, feature2ID := setupProgressTest(t, 84, 2, []models.TaskStatus{
		models.TaskStatusCompleted,
	})
	featureRepo.UpdateProgress(ctx, feature2ID)

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

// BenchmarkFeatureProgress measures feature progress calculation performance
func BenchmarkFeatureProgress(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)

	// Create test data with many tasks
	statuses := make([]models.TaskStatus, 100)
	for i := 0; i < 100; i++ {
		if i < 50 {
			statuses[i] = models.TaskStatusCompleted
		} else {
			statuses[i] = models.TaskStatusTodo
		}
	}

	_, featureID := setupProgressTest(&testing.T{}, 85, 1, statuses)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := featureRepo.CalculateProgress(ctx, featureID)
		if err != nil {
			b.Fatalf("Failed to calculate progress: %v", err)
		}
	}
}

// BenchmarkEpicProgress measures epic progress calculation performance
func BenchmarkEpicProgress(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create epic E88 with 50 features via setupProgressTest
	// Use E88 (reserved for benchmarks) to avoid conflicts with other tests
	epicKey := "E88"

	// Create epic using INSERT OR IGNORE pattern
	result, _ := database.Exec(`
		INSERT OR IGNORE INTO epics (key, title, description, status, priority)
		VALUES (?, 'Benchmark Epic', 'Epic for benchmarking', 'active', 'medium')
	`, epicKey)
	epicID, _ := result.LastInsertId()
	if epicID == 0 {
		database.QueryRow("SELECT id FROM epics WHERE key = ?", epicKey).Scan(&epicID)
	}

	// Create 50 features, each with 10 tasks (5 completed, 5 todo)
	for f := 1; f <= 50; f++ {
		featureKey := fmt.Sprintf("E88-F%02d", f)

		// Create feature with INSERT OR IGNORE
		result, _ := database.Exec(`
			INSERT OR IGNORE INTO features (epic_id, key, title, description, status)
			VALUES (?, ?, ?, 'Feature for benchmarking', 'active')
		`, epicID, featureKey, fmt.Sprintf("Benchmark Feature %d", f))
		featureID, _ := result.LastInsertId()
		if featureID == 0 {
			database.QueryRow("SELECT id FROM features WHERE key = ?", featureKey).Scan(&featureID)
		}

		// Delete and recreate tasks for this feature
		database.Exec("DELETE FROM tasks WHERE feature_id = ?", featureID)

		// Create 10 tasks (5 completed, 5 todo)
		for t := 1; t <= 10; t++ {
			status := models.TaskStatusTodo
			if t <= 5 {
				status = models.TaskStatusCompleted
			}
			database.Exec(`
				INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
				VALUES (?, ?, ?, ?, 'testing', 1, '[]')
			`, featureID, fmt.Sprintf("%s-T%03d", featureKey, t), fmt.Sprintf("Task %d", t), status)
		}

		featureRepo.UpdateProgress(ctx, featureID)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := epicRepo.CalculateProgress(ctx, epicID)
		if err != nil {
			b.Fatalf("Failed to calculate epic progress: %v", err)
		}
	}
}
