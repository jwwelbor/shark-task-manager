package commands

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestEpicUpdateCommand_Exists tests that the epic update command exists
func TestEpicUpdateCommand_Exists(t *testing.T) {
	// Verify the epic update command is registered
	var found bool
	for _, cmd := range epicCmd.Commands() {
		if cmd.Use == "update <epic-key>" {
			found = true

			// Verify it has the expected flags
			if cmd.Flags().Lookup("title") == nil {
				t.Error("epic update command missing --title flag")
			}
			if cmd.Flags().Lookup("description") == nil {
				t.Error("epic update command missing --description flag")
			}
			if cmd.Flags().Lookup("status") == nil {
				t.Error("epic update command missing --status flag")
			}
			if cmd.Flags().Lookup("priority") == nil {
				t.Error("epic update command missing --priority flag")
			}
			if cmd.Flags().Lookup("filename") == nil {
				t.Error("epic update command missing --filename flag")
			}
			if cmd.Flags().Lookup("path") == nil {
				t.Error("epic update command missing --path flag")
			}
			if cmd.Flags().Lookup("key") == nil {
				t.Error("epic update command missing --key flag")
			}
			if cmd.Flags().Lookup("force") == nil {
				t.Error("epic update command missing --force flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("epic update command not found in epic subcommands")
	}
}

// TestEpicUpdate_StatusCascadeWithForce tests that updating an epic to completed
// with --force cascades the status change to all child features and tasks
func TestEpicUpdate_StatusCascadeWithForce(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	database := test.GetTestDB()
	db := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E97-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E97-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")

	// Create test epic
	epic := &models.Epic{
		Key:         "E97",
		Title:       "Test Epic",
		Description: test.StringPtr("Test"),
		Status:      models.EpicStatusActive,
		Priority:    models.PriorityHigh,
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create test features with tasks
	for i := 1; i <= 2; i++ {
		feature := &models.Feature{
			EpicID:      epic.ID,
			Key:         test.GenerateUniqueKey("E97", i)[2:], // Remove "T-" prefix, yields "E97-001" format
			Title:       "Test Feature",
			Description: test.StringPtr("Test"),
			Status:      models.FeatureStatusActive,
			ProgressPct: 0.0,
		}
		// Fix the key format to be E97-F0X
		feature.Key = "E97-F0" + string(rune('0'+i))

		err = featureRepo.Create(ctx, feature)
		if err != nil {
			t.Fatalf("Failed to create feature: %v", err)
		}

		// Create tasks for this feature in various statuses
		statuses := []models.TaskStatus{
			models.TaskStatusTodo,
			models.TaskStatusInProgress,
			models.TaskStatusBlocked,
		}

		for j, status := range statuses {
			task := &models.Task{
				FeatureID: feature.ID,
				Key:       test.GenerateUniqueKey(feature.Key, j+1),
				Title:     "Test Task " + string(status),
				Status:    status,
				Priority:  5,
			}
			err = taskRepo.Create(ctx, task)
			if err != nil {
				t.Fatalf("Failed to create task: %v", err)
			}
		}
	}

	// Update epic status to completed with force=true
	// This should cascade to all child features and tasks
	epic.Status = models.EpicStatusCompleted
	err = epicRepo.Update(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to update epic: %v", err)
	}

	// Call cascade method (simulating --force behavior)
	err = epicRepo.CascadeStatusToFeaturesAndTasks(ctx, epic.ID, models.FeatureStatusCompleted, models.TaskStatusCompleted)
	if err != nil {
		t.Fatalf("Failed to cascade status to features and tasks: %v", err)
	}

	// Verify all features are now completed
	features, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to list features: %v", err)
	}

	for _, feature := range features {
		if feature.Status != models.FeatureStatusCompleted {
			t.Errorf("Feature %s status = %s, expected completed (force cascade should have updated it)", feature.Key, feature.Status)
		}

		// Verify all tasks in this feature are completed
		tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}

		for _, task := range tasks {
			if task.Status != models.TaskStatusCompleted {
				t.Errorf("Task %s status = %s, expected completed (force cascade should have updated it)", task.Key, task.Status)
			}
		}
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E97-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E97-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")
}

// TestEpicCreateCommand_KeyFlag tests that the epic create command has a --key flag
func TestEpicCreateCommand_KeyFlag(t *testing.T) {
	// Verify the epic create command is registered
	var found bool
	for _, cmd := range epicCmd.Commands() {
		if cmd.Use == "create <title>" {
			found = true

			// Verify it has the --key flag
			if cmd.Flags().Lookup("key") == nil {
				t.Error("epic create command missing --key flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("epic create command not found in epic subcommands")
	}
}
