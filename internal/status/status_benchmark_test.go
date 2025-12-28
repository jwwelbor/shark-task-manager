package status

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// BenchmarkGetDashboard_EmptyDatabase benchmarks dashboard with no data
func BenchmarkGetDashboard_EmptyDatabase(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Clear all data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	req := &StatusRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetDashboard(ctx, req)
		if err != nil {
			b.Fatalf("GetDashboard failed: %v", err)
		}
	}
}

// BenchmarkGetDashboard_SmallProject benchmarks dashboard with ~127 tasks (realistic small project)
func BenchmarkGetDashboard_SmallProject(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Setup: Create small project data
	setupSmallProject(b, database)

	req := &StatusRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetDashboard(ctx, req)
		if err != nil {
			b.Fatalf("GetDashboard failed: %v", err)
		}
	}
}

// BenchmarkGetDashboard_LargeProject benchmarks dashboard with ~2000 tasks (large enterprise project)
func BenchmarkGetDashboard_LargeProject(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Setup: Create large project data
	setupLargeProject(b, database)

	req := &StatusRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetDashboard(ctx, req)
		if err != nil {
			b.Fatalf("GetDashboard failed: %v", err)
		}
	}
}

// BenchmarkGetDashboard_FilteredByEpic benchmarks filtered queries
func BenchmarkGetDashboard_FilteredByEpic(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Setup: Create small project data
	setupSmallProject(b, database)

	req := &StatusRequest{EpicKey: "E01"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetDashboard(ctx, req)
		if err != nil {
			b.Fatalf("GetDashboard failed: %v", err)
		}
	}
}

// BenchmarkGetProjectSummary benchmarks the summary query alone
func BenchmarkGetProjectSummary(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	setupSmallProject(b, database)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.getProjectSummary(ctx, "")
		if err != nil {
			b.Fatalf("getProjectSummary failed: %v", err)
		}
	}
}

// BenchmarkGetEpics benchmarks epic breakdown query
func BenchmarkGetEpics(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	setupSmallProject(b, database)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.getEpics(ctx, "")
		if err != nil {
			b.Fatalf("getEpics failed: %v", err)
		}
	}
}

// BenchmarkGetActiveTasks benchmarks active tasks query
func BenchmarkGetActiveTasks(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	setupSmallProject(b, database)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.getActiveTasks(ctx, "")
		if err != nil {
			b.Fatalf("getActiveTasks failed: %v", err)
		}
	}
}

// BenchmarkGetBlockedTasks benchmarks blocked tasks query
func BenchmarkGetBlockedTasks(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	setupSmallProject(b, database)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.getBlockedTasks(ctx, "")
		if err != nil {
			b.Fatalf("getBlockedTasks failed: %v", err)
		}
	}
}

// BenchmarkDetermineEpicHealth benchmarks health calculation
func BenchmarkDetermineEpicHealth(b *testing.B) {
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.determineEpicHealth(45.5, 2)
	}
}

// BenchmarkStatusRequestValidate benchmarks validation
func BenchmarkStatusRequestValidate(b *testing.B) {
	req := &StatusRequest{
		EpicKey:      "E05",
		RecentWindow: "7d",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Validate()
	}
}

// Helper functions to setup test data

// setupSmallProject creates a small project with ~127 tasks
// Structure: 5 epics, 5 features per epic (25 features), ~5 tasks per feature
func setupSmallProject(b *testing.B, database *sql.DB) {
	ctx := context.Background()

	// Clear existing data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	// Create 5 epics
	for epicNum := 1; epicNum <= 5; epicNum++ {
		result, err := database.ExecContext(ctx, `
			INSERT INTO epics (key, title, description, status, priority)
			VALUES (?, ?, ?, 'active', 'high')
		`, fmt.Sprintf("E%02d", epicNum), fmt.Sprintf("Epic %d", epicNum), fmt.Sprintf("Description for epic %d", epicNum))
		if err != nil {
			b.Fatalf("Failed to create epic: %v", err)
		}
		epicID, _ := result.LastInsertId()

		// Create 5 features per epic
		for featureNum := 1; featureNum <= 5; featureNum++ {
			featureResult, err := database.ExecContext(ctx, `
				INSERT INTO features (epic_id, key, title, description, status)
				VALUES (?, ?, ?, ?, 'active')
			`, epicID, fmt.Sprintf("E%02d-F%02d", epicNum, featureNum), fmt.Sprintf("Feature %d", featureNum), fmt.Sprintf("Description for feature %d", featureNum))
			if err != nil {
				b.Fatalf("Failed to create feature: %v", err)
			}
			featureID, _ := featureResult.LastInsertId()

			// Create 5 tasks per feature with varied statuses and agent types
			statuses := []string{"todo", "in_progress", "ready_for_review", "completed", "blocked"}
			agents := []string{"backend", "frontend", "api", "testing", "devops"}

			for taskNum := 0; taskNum < 5; taskNum++ {
				status := statuses[taskNum%len(statuses)]
				agent := agents[taskNum%len(agents)]

				_, err := database.ExecContext(ctx, `
					INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
					VALUES (?, ?, ?, ?, ?, ?, '[]')
				`, featureID, fmt.Sprintf("T-E%02d-F%02d-%03d", epicNum, featureNum, taskNum+1), fmt.Sprintf("Task %d", taskNum+1), status, agent, (taskNum%10)+1)
				if err != nil {
					b.Fatalf("Failed to create task: %v", err)
				}

				// Add blocked reason for blocked tasks
				if status == "blocked" {
					_, _ = database.ExecContext(ctx, `
						UPDATE tasks SET blocked_reason = ? WHERE key = ?
					`, fmt.Sprintf("Blocked reason for task %d", taskNum+1), fmt.Sprintf("T-E%02d-F%02d-%03d", epicNum, featureNum, taskNum+1))
				}
			}
		}
	}
}

// setupLargeProject creates a large project with ~2000 tasks
// Structure: 20 epics, 10 features per epic (200 features), ~10 tasks per feature
func setupLargeProject(b *testing.B, database *sql.DB) {
	ctx := context.Background()

	// Clear existing data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	// Create 20 epics
	for epicNum := 1; epicNum <= 20; epicNum++ {
		result, err := database.ExecContext(ctx, `
			INSERT INTO epics (key, title, description, status, priority)
			VALUES (?, ?, ?, 'active', 'high')
		`, fmt.Sprintf("E%02d", epicNum), fmt.Sprintf("Epic %d", epicNum), fmt.Sprintf("Description for epic %d", epicNum))
		if err != nil {
			b.Fatalf("Failed to create epic: %v", err)
		}
		epicID, _ := result.LastInsertId()

		// Create 10 features per epic
		for featureNum := 1; featureNum <= 10; featureNum++ {
			featureResult, err := database.ExecContext(ctx, `
				INSERT INTO features (epic_id, key, title, description, status)
				VALUES (?, ?, ?, ?, 'active')
			`, epicID, fmt.Sprintf("E%02d-F%02d", epicNum, featureNum), fmt.Sprintf("Feature %d", featureNum), fmt.Sprintf("Description for feature %d", featureNum))
			if err != nil {
				b.Fatalf("Failed to create feature: %v", err)
			}
			featureID, _ := featureResult.LastInsertId()

			// Create 10 tasks per feature with varied statuses and agent types
			statuses := []string{"todo", "in_progress", "ready_for_review", "completed", "blocked"}
			agents := []string{"backend", "frontend", "api", "testing", "devops"}

			for taskNum := 0; taskNum < 10; taskNum++ {
				status := statuses[taskNum%len(statuses)]
				agent := agents[taskNum%len(agents)]

				_, err := database.ExecContext(ctx, `
					INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
					VALUES (?, ?, ?, ?, ?, ?, '[]')
				`, featureID, fmt.Sprintf("T-E%02d-F%02d-%03d", epicNum, featureNum, taskNum+1), fmt.Sprintf("Task %d", taskNum+1), status, agent, (taskNum%10)+1)
				if err != nil {
					b.Fatalf("Failed to create task: %v", err)
				}

				// Add blocked reason for blocked tasks
				if status == "blocked" {
					_, _ = database.ExecContext(ctx, `
						UPDATE tasks SET blocked_reason = ? WHERE key = ?
					`, fmt.Sprintf("Blocked reason for task %d", taskNum+1), fmt.Sprintf("T-E%02d-F%02d-%03d", epicNum, featureNum, taskNum+1))
				}
			}
		}
	}
}
