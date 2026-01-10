package commands

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestFeatureUpdateCommand_Exists tests that the feature update command exists
func TestFeatureUpdateCommand_Exists(t *testing.T) {
	// Verify the feature update command is registered
	var found bool
	for _, cmd := range featureCmd.Commands() {
		if cmd.Use == "update <feature-key>" {
			found = true

			// Verify it has the expected flags
			if cmd.Flags().Lookup("title") == nil {
				t.Error("feature update command missing --title flag")
			}
			if cmd.Flags().Lookup("description") == nil {
				t.Error("feature update command missing --description flag")
			}
			if cmd.Flags().Lookup("status") == nil {
				t.Error("feature update command missing --status flag")
			}
			if cmd.Flags().Lookup("execution-order") == nil {
				t.Error("feature update command missing --execution-order flag")
			}
			if cmd.Flags().Lookup("filename") == nil {
				t.Error("feature update command missing --filename flag")
			}
			if cmd.Flags().Lookup("path") == nil {
				t.Error("feature update command missing --path flag")
			}
			if cmd.Flags().Lookup("key") == nil {
				t.Error("feature update command missing --key flag")
			}
			if cmd.Flags().Lookup("force") == nil {
				t.Error("feature update command missing --force flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("feature update command not found in feature subcommands")
	}
}

// TestFeatureUpdate_StatusCascadeWithForce tests that updating a feature to completed
// with --force cascades the status change to all child tasks
func TestFeatureUpdate_StatusCascadeWithForce(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	database := test.GetTestDB()
	db := repository.NewDB(database)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create test epic
	_, err := database.ExecContext(ctx,
		"INSERT INTO epics (key, title, description, status, priority) VALUES ('E98', 'Test Epic', 'Test', 'active', 'high')")
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	var epicID int64
	err = database.QueryRowContext(ctx, "SELECT id FROM epics WHERE key = 'E98'").Scan(&epicID)
	if err != nil {
		t.Fatalf("Failed to get epic ID: %v", err)
	}

	// Create test feature
	feature := &models.Feature{
		EpicID:      epicID,
		Key:         "E98-F01",
		Title:       "Test Feature",
		Description: test.StringPtr("Test"),
		Status:      models.FeatureStatusActive,
		ProgressPct: 0.0,
	}
	err = featureRepo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create tasks in various statuses
	statuses := []models.TaskStatus{
		models.TaskStatusTodo,
		models.TaskStatusInProgress,
		models.TaskStatusReadyForReview,
		models.TaskStatusBlocked,
		models.TaskStatusArchived,
	}

	for i, status := range statuses {
		task := &models.Task{
			FeatureID: feature.ID,
			Key:       test.GenerateUniqueKey("E98-F01", i+1),
			Title:     "Test Task " + string(status),
			Status:    status,
			Priority:  5,
		}
		err = taskRepo.Create(ctx, task)
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Update feature status to completed with force=true
	// This should cascade to all child tasks
	feature.Status = models.FeatureStatusCompleted
	feature.StatusOverride = true // Set override flag
	err = featureRepo.Update(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to update feature: %v", err)
	}

	// Enable status override
	err = featureRepo.SetStatusOverride(ctx, feature.ID, true)
	if err != nil {
		t.Fatalf("Failed to set status override: %v", err)
	}

	// Call cascade method (simulating --force behavior)
	err = featureRepo.CascadeStatusToTasks(ctx, feature.ID, models.TaskStatusCompleted)
	if err != nil {
		t.Fatalf("Failed to cascade status to tasks: %v", err)
	}

	// Verify all tasks are now completed
	tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	for _, task := range tasks {
		if task.Status != models.TaskStatusCompleted {
			t.Errorf("Task %s status = %s, expected completed (force cascade should have updated it)", task.Key, task.Status)
		}
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")
}

// TestFeatureCreateCommand_KeyFlag tests that the feature create command has a --key flag
func TestFeatureCreateCommand_KeyFlag(t *testing.T) {
	// Verify the feature create command is registered
	var found bool
	for _, cmd := range featureCmd.Commands() {
		// Updated to match new positional argument syntax
		if cmd.Use == "create [EPIC] <title> [flags]" {
			found = true

			// Verify it has the --key flag
			if cmd.Flags().Lookup("key") == nil {
				t.Error("feature create command missing --key flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("feature create command not found in feature subcommands")
	}
}
