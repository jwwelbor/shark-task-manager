package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// Integration tests for task list command with positional arguments

// TestParseTaskListArgsIntegration verifies positional argument parsing for task list
func TestParseTaskListArgsIntegration(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantEpic    *string
		wantFeature *string
		wantErr     bool
		errMessage  string
	}{
		// No arguments - list all tasks
		{
			name:        "No args lists all tasks",
			args:        []string{},
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     false,
		},

		// Single epic argument
		{
			name:        "Single epic E04",
			args:        []string{"E04"},
			wantEpic:    strPtr("E04"),
			wantFeature: nil,
			wantErr:     false,
		},

		// Single feature key (combined format)
		{
			name:        "Combined feature E04-F01",
			args:        []string{"E04-F01"},
			wantEpic:    strPtr("E04"),
			wantFeature: strPtr("F01"),
			wantErr:     false,
		},

		// Two arguments - epic and feature suffix
		{
			name:        "Two args E04 F01",
			args:        []string{"E04", "F01"},
			wantEpic:    strPtr("E04"),
			wantFeature: strPtr("F01"),
			wantErr:     false,
		},

		// Two arguments - epic and full feature key
		{
			name:        "Two args E04 E04-F01",
			args:        []string{"E04", "E04-F01"},
			wantEpic:    strPtr("E04"),
			wantFeature: strPtr("F01"),
			wantErr:     false,
		},

		// Invalid single argument
		{
			name:        "Invalid format E1",
			args:        []string{"E1"},
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
			errMessage:  "invalid",
		},

		{
			name:        "Invalid format e04",
			args:        []string{"e04"},
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
			errMessage:  "invalid",
		},

		{
			name:        "Feature suffix only F01",
			args:        []string{"F01"},
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
			errMessage:  "invalid",
		},

		// Invalid two arguments
		{
			name:        "Invalid second arg",
			args:        []string{"E04", "invalid"},
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
			errMessage:  "invalid",
		},

		// Too many arguments
		{
			name:        "Three arguments not allowed",
			args:        []string{"E04", "F01", "extra"},
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
			errMessage:  "too many",
		},

		{
			name:        "Four arguments not allowed",
			args:        []string{"E04", "F01", "extra", "more"},
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
			errMessage:  "too many",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epic, feature, err := ParseTaskListArgs(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTaskListArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMessage != "" {
				if err == nil || !containsIgnoreCase(err.Error(), tt.errMessage) {
					t.Errorf("ParseTaskListArgs(%v) error should contain %q, got %v", tt.args, tt.errMessage, err)
				}
				return
			}

			if (epic == nil) != (tt.wantEpic == nil) {
				t.Errorf("ParseTaskListArgs(%v) epic = %v, want %v", tt.args, epic, tt.wantEpic)
				return
			}

			if epic != nil && tt.wantEpic != nil && *epic != *tt.wantEpic {
				t.Errorf("ParseTaskListArgs(%v) epic = %q, want %q", tt.args, *epic, *tt.wantEpic)
			}

			if (feature == nil) != (tt.wantFeature == nil) {
				t.Errorf("ParseTaskListArgs(%v) feature = %v, want %v", tt.args, feature, tt.wantFeature)
				return
			}

			if feature != nil && tt.wantFeature != nil && *feature != *tt.wantFeature {
				t.Errorf("ParseTaskListArgs(%v) feature = %q, want %q", tt.args, *feature, *tt.wantFeature)
			}
		})
	}
}

// TestTaskListQueryWithDatabase verifies task list with real database
func TestTaskListQueryWithDatabase(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	// Create test epic
	testEpicKey := "E71"
	epic := &models.Epic{
		Key:           testEpicKey,
		Title:         "Task List Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: ptrPriority(models.PriorityHigh),
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	createdEpic, err := epicRepo.GetByKey(ctx, testEpicKey)
	if err != nil || createdEpic == nil {
		t.Fatalf("Failed to retrieve created epic: %v", err)
	}

	// Create test feature
	testFeatureKey := fmt.Sprintf("%s-F01", testEpicKey)
	featureFilePath := fmt.Sprintf("docs/plan/%s/F01/feature.md", testEpicKey)
	execOrder := 1
	feature := &models.Feature{
		Key:            testFeatureKey,
		EpicID:         createdEpic.ID,
		Title:          "Task List Test Feature",
		Status:         models.FeatureStatusDraft,
		FilePath:       &featureFilePath,
		ExecutionOrder: &execOrder,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	createdFeature, err := featureRepo.GetByKey(ctx, testFeatureKey)
	if err != nil || createdFeature == nil {
		t.Fatalf("Failed to retrieve created feature: %v", err)
	}

	// Create tasks under the feature
	for i := 1; i <= 3; i++ {
		taskKey := fmt.Sprintf("T-%s-%03d", testFeatureKey, i)
		agentType := models.AgentTypeGeneral
		taskFilePath := fmt.Sprintf("docs/plan/%s/tasks/%s.md", testFeatureKey, taskKey)
		task := &models.Task{
			Key:         taskKey,
			FeatureID:   createdFeature.ID,
			Title:       fmt.Sprintf("Test Task %d", i),
			Description: strPtr("Task for integration testing"),
			Status:      models.TaskStatusTodo,
			AgentType:   &agentType,
			Priority:    i,
			FilePath:    &taskFilePath,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create test task: %v", err)
		}
	}

	// Test: List all tasks (no filter)
	allTasks, err := taskRepo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list all tasks: %v", err)
	}
	if len(allTasks) == 0 {
		t.Error("Expected tasks in database but found none")
	}

	// Test: Filter by epic
	epicTasks, err := taskRepo.ListByEpic(ctx, testEpicKey)
	if err != nil {
		t.Fatalf("Failed to list tasks by epic: %v", err)
	}
	if len(epicTasks) != 3 {
		t.Errorf("Expected 3 tasks for epic, got %d", len(epicTasks))
	}

	// Test: Filter by feature
	featureTasks, err := taskRepo.ListByFeature(ctx, createdFeature.ID)
	if err != nil {
		t.Fatalf("Failed to list tasks by feature: %v", err)
	}
	if len(featureTasks) != 3 {
		t.Errorf("Expected 3 tasks for feature, got %d", len(featureTasks))
	}
}

// Helper function for case-insensitive substring check
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr)
}
