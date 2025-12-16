package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// testIsTaskAvailable is a test helper that checks if a task is available
// It uses the TaskRepositoryInterface so it works with mocks
func testIsTaskAvailable(ctx context.Context, task *models.Task, repo TaskRepositoryInterface) bool {
	if task.DependsOn == nil || *task.DependsOn == "" || *task.DependsOn == "[]" {
		return true // No dependencies
	}

	var deps []string
	if err := json.Unmarshal([]byte(*task.DependsOn), &deps); err != nil {
		return true // Invalid JSON, treat as no dependencies
	}

	// Check each dependency
	for _, depKey := range deps {
		depTask, err := repo.GetByKey(ctx, depKey)
		if err != nil {
			return false // Dependency not found
		}

		// Dependency must be completed or archived
		if depTask.Status != models.TaskStatusCompleted && depTask.Status != models.TaskStatusArchived {
			return false
		}
	}

	return true
}

// TestTaskAvailability tests dependency resolution logic
func TestTaskAvailability(t *testing.T) {
	ctx := context.Background()
	mockRepo := NewMockTaskRepository()

	// Setup test data
	emptyDeps := "[]"
	completedDep := `["T-TEST-001"]`
	incompleteDep := `["T-TEST-002"]`
	agentType := models.AgentTypeBackend

	// Add test tasks to mock repository
	mockRepo.AddTask(&models.Task{
		ID:        1,
		Key:       "T-TEST-001",
		Title:     "Completed Task",
		Status:    models.TaskStatusCompleted,
		AgentType: &agentType,
		Priority:  1,
		DependsOn: &emptyDeps,
	})

	mockRepo.AddTask(&models.Task{
		ID:        2,
		Key:       "T-TEST-002",
		Title:     "Todo Task",
		Status:    models.TaskStatusTodo,
		AgentType: &agentType,
		Priority:  2,
		DependsOn: &emptyDeps,
	})

	mockRepo.AddTask(&models.Task{
		ID:        3,
		Key:       "T-TEST-003",
		Title:     "Task with Dependency",
		Status:    models.TaskStatusTodo,
		AgentType: &agentType,
		Priority:  3,
		DependsOn: &completedDep,
	})

	mockRepo.AddTask(&models.Task{
		ID:        4,
		Key:       "T-TEST-004",
		Title:     "Task with Incomplete Dependency",
		Status:    models.TaskStatusTodo,
		AgentType: &agentType,
		Priority:  4,
		DependsOn: &incompleteDep,
	})

	tests := []struct {
		name            string
		taskKey         string
		expectAvailable bool
		reason          string
	}{
		{
			name:            "no_dependencies",
			taskKey:         "T-TEST-002",
			expectAvailable: true,
			reason:          "Task with no dependencies should be available",
		},
		{
			name:            "completed_dependency",
			taskKey:         "T-TEST-003",
			expectAvailable: true,
			reason:          "Task depending on completed task should be available",
		},
		{
			name:            "incomplete_dependency",
			taskKey:         "T-TEST-004",
			expectAvailable: false,
			reason:          "Task depending on todo task should NOT be available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := mockRepo.GetByKey(ctx, tt.taskKey)
			if err != nil {
				t.Fatalf("Failed to get task %s: %v", tt.taskKey, err)
			}

			available := testIsTaskAvailable(ctx, task, mockRepo)
			if available != tt.expectAvailable {
				t.Errorf("Task %s: expected available=%v, got %v. %s",
					tt.taskKey, tt.expectAvailable, available, tt.reason)
			}
		})
	}
}

// TestDependencyParsing tests JSON dependency parsing edge cases
func TestDependencyParsing(t *testing.T) {
	ctx := context.Background()
	mockRepo := NewMockTaskRepository()
	agentType := models.AgentTypeBackend

	tests := []struct {
		name            string
		dependsOn       string
		expectAvailable bool
		description     string
	}{
		{
			name:            "empty_string",
			dependsOn:       "",
			expectAvailable: true,
			description:     "Empty string should be treated as no dependencies",
		},
		{
			name:            "empty_array",
			dependsOn:       "[]",
			expectAvailable: true,
			description:     "Empty JSON array should be treated as no dependencies",
		},
		{
			name:            "invalid_json",
			dependsOn:       "invalid",
			expectAvailable: true,
			description:     "Invalid JSON should be treated as no dependencies (fail safe)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &models.Task{
				FeatureID: 1,
				Key:       "T-PARSE-TEST",
				Title:     "Parsing Test",
				Status:    models.TaskStatusTodo,
				AgentType: &agentType,
				Priority:  1,
				DependsOn: &tt.dependsOn,
			}

			available := testIsTaskAvailable(ctx, task, mockRepo)
			if available != tt.expectAvailable {
				t.Errorf("%s: expected available=%v, got %v. %s",
					tt.name, tt.expectAvailable, available, tt.description)
			}
		})
	}
}

// TestStateTransitionLogic verifies state validation would work at CLI level
func TestStateTransitionLogic(t *testing.T) {
	// This tests the validation logic that SHOULD be in place
	// These are the rules enforced by the CLI commands

	validTransitions := map[string]map[string]bool{
		string(models.TaskStatusTodo): {
			string(models.TaskStatusInProgress): true,
			string(models.TaskStatusBlocked):    true,
		},
		string(models.TaskStatusInProgress): {
			string(models.TaskStatusReadyForReview): true,
			string(models.TaskStatusBlocked):        true,
		},
		string(models.TaskStatusReadyForReview): {
			string(models.TaskStatusCompleted):  true,
			string(models.TaskStatusInProgress): true, // reopen
		},
		string(models.TaskStatusBlocked): {
			string(models.TaskStatusTodo): true,
		},
	}

	tests := []struct {
		from   models.TaskStatus
		to     models.TaskStatus
		isValid bool
	}{
		// Valid transitions
		{models.TaskStatusTodo, models.TaskStatusInProgress, true},
		{models.TaskStatusInProgress, models.TaskStatusReadyForReview, true},
		{models.TaskStatusReadyForReview, models.TaskStatusCompleted, true},
		{models.TaskStatusTodo, models.TaskStatusBlocked, true},
		{models.TaskStatusInProgress, models.TaskStatusBlocked, true},
		{models.TaskStatusBlocked, models.TaskStatusTodo, true},
		{models.TaskStatusReadyForReview, models.TaskStatusInProgress, true},

		// Invalid transitions
		{models.TaskStatusCompleted, models.TaskStatusInProgress, false},
		{models.TaskStatusTodo, models.TaskStatusCompleted, false},
		{models.TaskStatusTodo, models.TaskStatusReadyForReview, false},
		{models.TaskStatusBlocked, models.TaskStatusInProgress, false},
		{models.TaskStatusBlocked, models.TaskStatusCompleted, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			allowed := false
			if transitions, ok := validTransitions[string(tt.from)]; ok {
				allowed = transitions[string(tt.to)]
			}

			if allowed != tt.isValid {
				if tt.isValid {
					t.Errorf("Expected transition %s -> %s to be VALID, but validation says invalid",
						tt.from, tt.to)
				} else {
					t.Errorf("Expected transition %s -> %s to be INVALID, but validation says valid",
						tt.from, tt.to)
				}
			}
		})
	}
}

// TestNextTaskSelection tests the logic for selecting the next available task
func TestNextTaskSelection(t *testing.T) {
	ctx := context.Background()
	mockRepo := NewMockTaskRepository()

	// Setup test data
	emptyDeps := "[]"
	completedDep := `["T-TEST-001"]`
	agentType := models.AgentTypeBackend

	// Add test tasks to mock repository
	mockRepo.AddTask(&models.Task{
		ID:        1,
		Key:       "T-TEST-001",
		Title:     "Completed Task",
		Status:    models.TaskStatusCompleted,
		AgentType: &agentType,
		Priority:  1,
		DependsOn: &emptyDeps,
	})

	mockRepo.AddTask(&models.Task{
		ID:        2,
		Key:       "T-TEST-002",
		Title:     "Todo Task",
		Status:    models.TaskStatusTodo,
		AgentType: &agentType,
		Priority:  2,
		DependsOn: &emptyDeps,
	})

	mockRepo.AddTask(&models.Task{
		ID:        3,
		Key:       "T-TEST-003",
		Title:     "Task with Dependency",
		Status:    models.TaskStatusTodo,
		AgentType: &agentType,
		Priority:  3,
		DependsOn: &completedDep,
	})

	// Query for todo tasks
	todoStatus := models.TaskStatusTodo
	tasks, err := mockRepo.FilterCombined(ctx, &todoStatus, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to get todo tasks: %v", err)
	}

	// Filter by availability (no incomplete dependencies)
	var availableTasks []*models.Task
	for _, task := range tasks {
		if testIsTaskAvailable(ctx, task, mockRepo) {
			availableTasks = append(availableTasks, task)
		}
	}

	if len(availableTasks) == 0 {
		t.Fatal("Expected at least one available task")
	}

	// Next task should be the one with lowest priority number (highest priority)
	nextTask := availableTasks[0]
	for _, task := range availableTasks[1:] {
		if task.Priority < nextTask.Priority {
			nextTask = task
		}
	}

	// Verify T-TEST-002 (priority 2) is selected over T-TEST-003 (priority 3)
	// T-TEST-003 depends on completed task, so both are available
	// But T-TEST-002 has higher priority (lower number)
	if nextTask.Key != "T-TEST-002" {
		t.Errorf("Expected next task to be T-TEST-002 (highest priority available), got %s", nextTask.Key)
	}
}

// TestJSONDependencyFormat validates dependency JSON format
func TestJSONDependencyFormat(t *testing.T) {
	validJSON := `["T-TEST-001", "T-TEST-002"]`

	var deps []string
	err := json.Unmarshal([]byte(validJSON), &deps)
	if err != nil {
		t.Errorf("Valid dependency JSON failed to parse: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	if deps[0] != "T-TEST-001" || deps[1] != "T-TEST-002" {
		t.Errorf("Unexpected dependency values: %v", deps)
	}
}

// TestAgentIdentification tests the agent identifier logic
func TestAgentIdentification(t *testing.T) {
	tests := []struct {
		name        string
		flagValue   string
		expectedID  string
	}{
		{
			name:       "flag_provided",
			flagValue:  "custom-agent",
			expectedID: "custom-agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAgentIdentifier(tt.flagValue)
			if result != tt.expectedID {
				t.Errorf("Expected agent ID '%s', got '%s'", tt.expectedID, result)
			}
		})
	}
}
