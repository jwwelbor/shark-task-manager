package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestGetRejectionCounts tests counting rejections for tasks
func TestGetRejectionCounts(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Cleanup before test
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-F99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E99-F99'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	// Seed epic and feature
	epicResult, err := database.ExecContext(ctx, `
		INSERT INTO epics (key, slug, title, description, status)
		VALUES ('E99', 'test-epic-99', 'Test Epic 99', 'Test', 'active')
	`)
	if err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}
	epicID, err := epicResult.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get epic ID: %v", err)
	}

	featureResult, err := database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, slug, title, description)
		VALUES (?, 'E99-F99', 'test-feature-99', 'Test Feature 99', 'Test')
	`, epicID)
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}
	featureID, err := featureResult.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get feature ID: %v", err)
	}

	// Create test task
	task := &models.Task{
		FeatureID: featureID,
		Key:       "T-E99-F99-001",
		Title:     "Test task with rejections",
		Status:    models.TaskStatusInProgress,
		Priority:  5,
		AgentType: nil,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = taskRepo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)

	tests := []struct {
		name             string
		rejectionCount   int
		hasLastRejection bool
		expectedLastTime *time.Time
	}{
		{
			name:             "task with no rejections",
			rejectionCount:   0,
			hasLastRejection: false,
			expectedLastTime: nil,
		},
		{
			name:             "task with one rejection",
			rejectionCount:   1,
			hasLastRejection: true,
			expectedLastTime: nil, // Will be set dynamically
		},
		{
			name:             "task with multiple rejections",
			rejectionCount:   3,
			hasLastRejection: true,
			expectedLastTime: nil, // Will be set dynamically
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any existing rejections
			_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE task_id = ? AND note_type = 'rejection'", task.ID)

			// Create rejection notes
			now := time.Now()
			for j := 0; j < tt.rejectionCount; j++ {
				noteTime := now.Add(time.Duration(j) * time.Hour)
				_, err := database.ExecContext(ctx, `
					INSERT INTO task_notes (task_id, note_type, content, created_at, created_by)
					VALUES (?, 'rejection', ?, ?, 'test-rejector')
				`, task.ID, "Rejection reason "+string(rune('0'+j)), noteTime)
				if err != nil {
					t.Fatalf("Failed to create rejection note: %v", err)
				}
			}

			// Get rejection counts
			counts, lastTimes, err := taskRepo.GetRejectionCounts(ctx, []int64{task.ID})
			if err != nil {
				t.Fatalf("Failed to get rejection counts: %v", err)
			}

			// Verify count
			if counts[task.ID] != tt.rejectionCount {
				t.Errorf("Expected %d rejections, got %d", tt.rejectionCount, counts[task.ID])
			}

			// Verify last rejection time
			if tt.hasLastRejection {
				if lastTimes[task.ID] == nil {
					t.Errorf("Expected last rejection time, got nil")
				}
			} else {
				if lastTimes[task.ID] != nil {
					t.Errorf("Expected no last rejection time, got %v", lastTimes[task.ID])
				}
			}

			_ = i // Use i variable
		})
	}
}

// TestGetRejectionCountsMultipleTasks tests getting rejection counts for multiple tasks at once
func TestGetRejectionCountsMultipleTasks(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E88-F88-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E88-F88'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E88'")

	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	// Seed epic and feature
	epicResult, err := database.ExecContext(ctx, `
		INSERT INTO epics (key, slug, title, description, status)
		VALUES ('E88', 'test-epic-88', 'Test Epic 88', 'Test', 'active')
	`)
	if err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}
	epicID, _ := epicResult.LastInsertId()

	featureResult, err := database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, slug, title, description)
		VALUES (?, 'E88-F88', 'test-feature-88', 'Test Feature 88', 'Test')
	`, epicID)
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}
	featureID, _ := featureResult.LastInsertId()

	// Create multiple test tasks
	var taskIDs []int64
	for i := 1; i <= 3; i++ {
		task := &models.Task{
			FeatureID: featureID,
			Key:       "T-E88-F88-00" + string(rune('0'+i)),
			Title:     "Test task " + string(rune('0'+i)),
			Status:    models.TaskStatusInProgress,
			Priority:  5,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
		taskIDs = append(taskIDs, task.ID)
	}

	// Cleanup
	defer func() {
		for _, taskID := range taskIDs {
			_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskID)
		}
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID)
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID)
	}()

	// Add rejections: task1=0, task2=1, task3=2
	for i, taskID := range taskIDs {
		for j := 0; j < i; j++ {
			_, _ = database.ExecContext(ctx, `
				INSERT INTO task_notes (task_id, note_type, content, created_at, created_by)
				VALUES (?, 'rejection', ?, ?, 'test-rejector')
			`, taskID, "Rejection", time.Now())
		}
	}

	// Get counts for all tasks
	counts, _, err := taskRepo.GetRejectionCounts(ctx, taskIDs)
	if err != nil {
		t.Fatalf("Failed to get rejection counts: %v", err)
	}

	// Verify
	expectedCounts := map[int64]int{
		taskIDs[0]: 0,
		taskIDs[1]: 1,
		taskIDs[2]: 2,
	}

	for taskID, expectedCount := range expectedCounts {
		if counts[taskID] != expectedCount {
			t.Errorf("Task %d: expected %d rejections, got %d", taskID, expectedCount, counts[taskID])
		}
	}
}
