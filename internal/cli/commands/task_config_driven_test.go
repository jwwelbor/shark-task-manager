package commands

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TestTaskBlockCommand_HardcodedStatusValidation demonstrates the hardcoded validation issue
// in the task block command (lines 1280-1281 in task.go).
//
// PROBLEM: The block command only allows blocking from "todo" or "in_progress" statuses.
// This is hardcoded and doesn't respect workflow configuration.
//
// This test SHOULD FAIL initially to demonstrate the limitation.
func TestTaskBlockCommand_HardcodedStatusValidation(t *testing.T) {
	tests := []struct {
		name           string
		currentStatus  models.TaskStatus
		shouldBlock    bool
		workflowConfig *config.WorkflowConfig
		description    string
	}{
		{
			name:          "block from in_development should work",
			currentStatus: models.TaskStatus("in_development"),
			shouldBlock:   true,
			workflowConfig: &config.WorkflowConfig{
				StatusFlow: map[string][]string{
					"in_development": {"blocked", "ready_for_code_review"},
					"blocked":        {"in_development"},
				},
			},
			description: "Workflow config allows blocking from in_development, but hardcoded validation rejects it",
		},
		{
			name:          "block from in_refinement should work",
			currentStatus: models.TaskStatus("in_refinement"),
			shouldBlock:   true,
			workflowConfig: &config.WorkflowConfig{
				StatusFlow: map[string][]string{
					"in_refinement": {"blocked", "ready_for_development"},
					"blocked":       {"in_refinement"},
				},
			},
			description: "Workflow config allows blocking from in_refinement, but hardcoded validation rejects it",
		},
		{
			name:          "block from in_qa should work",
			currentStatus: models.TaskStatus("in_qa"),
			shouldBlock:   true,
			workflowConfig: &config.WorkflowConfig{
				StatusFlow: map[string][]string{
					"in_qa":  {"blocked", "ready_for_approval"},
					"blocked": {"in_qa"},
				},
			},
			description: "Workflow config allows blocking from in_qa, but hardcoded validation rejects it",
		},
		{
			name:          "hardcoded todo status still works",
			currentStatus: models.TaskStatusTodo,
			shouldBlock:   true,
			workflowConfig: &config.WorkflowConfig{
				StatusFlow: map[string][]string{
					"todo":    {"blocked", "in_progress"},
					"blocked": {"todo"},
				},
			},
			description: "Old hardcoded status should still work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The current implementation has this hardcoded check:
			// if !force && task.Status != models.TaskStatusTodo && task.Status != models.TaskStatusInProgress {
			//     return error
			// }
			//
			// This check should instead consult the workflow config to see if
			// transition from current status to "blocked" is allowed.

			canBlockAccordingToWorkflow := false
			if tt.workflowConfig != nil && tt.workflowConfig.StatusFlow != nil {
				allowedTransitions := tt.workflowConfig.StatusFlow[string(tt.currentStatus)]
				for _, nextStatus := range allowedTransitions {
					if nextStatus == "blocked" {
						canBlockAccordingToWorkflow = true
						break
					}
				}
			}

			// The hardcoded check in task.go line 1280
			canBlockAccordingToHardcodedLogic := (tt.currentStatus == models.TaskStatusTodo ||
				tt.currentStatus == models.TaskStatusInProgress)

			if tt.shouldBlock {
				if !canBlockAccordingToWorkflow {
					t.Fatalf("Test setup error: workflow doesn't allow blocking from %s", tt.currentStatus)
				}

				if !canBlockAccordingToHardcodedLogic {
					t.Errorf("EXPECTED FAILURE (demonstrates hardcoded limitation):\n"+
						"  %s\n"+
						"  Current status: %s\n"+
						"  Workflow config allows blocking: %v\n"+
						"  Hardcoded validation allows blocking: %v\n"+
						"  Problem: Hardcoded check on line 1280 prevents valid workflow transitions",
						tt.description,
						tt.currentStatus,
						canBlockAccordingToWorkflow,
						canBlockAccordingToHardcodedLogic,
					)
				}
			}
		})
	}
}

// TestTaskReopenCommand_HardcodedStatusValidation demonstrates the hardcoded validation issue
// in the task reopen command (lines 1404-1405 in task.go).
//
// PROBLEM: The reopen command only allows reopening from "ready_for_review" status.
// This is hardcoded and doesn't respect workflow configuration.
func TestTaskReopenCommand_HardcodedStatusValidation(t *testing.T) {
	tests := []struct {
		name           string
		currentStatus  models.TaskStatus
		shouldReopen   bool
		workflowConfig *config.WorkflowConfig
		description    string
	}{
		{
			name:          "reopen from in_code_review should work",
			currentStatus: models.TaskStatus("in_code_review"),
			shouldReopen:  true,
			workflowConfig: &config.WorkflowConfig{
				StatusFlow: map[string][]string{
					"in_code_review": {"in_development", "ready_for_qa"},
					"in_development": {"ready_for_code_review"},
				},
			},
			description: "Workflow config allows transitioning from in_code_review to in_development, but hardcoded validation rejects it",
		},
		{
			name:          "reopen from in_qa should work",
			currentStatus: models.TaskStatus("in_qa"),
			shouldReopen:  true,
			workflowConfig: &config.WorkflowConfig{
				StatusFlow: map[string][]string{
					"in_qa":          {"in_development", "ready_for_approval"},
					"in_development": {"ready_for_code_review"},
				},
			},
			description: "Workflow config allows transitioning from in_qa back to in_development, but hardcoded validation rejects it",
		},
		{
			name:          "reopen from in_approval should work",
			currentStatus: models.TaskStatus("in_approval"),
			shouldReopen:  true,
			workflowConfig: &config.WorkflowConfig{
				StatusFlow: map[string][]string{
					"in_approval":    {"ready_for_qa", "ready_for_development"},
					"ready_for_qa":   {"in_qa"},
					"in_development": {"ready_for_code_review"},
				},
			},
			description: "Workflow config allows transitioning from in_approval back to earlier stages, but hardcoded validation rejects it",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The current implementation has this hardcoded check:
			// if !force && task.Status != models.TaskStatusReadyForReview {
			//     return error
			// }
			//
			// This check should instead consult the workflow config to see if
			// the reopen operation (transition to in_progress or other target status)
			// is allowed from the current status.

			// Check if workflow allows any backward transition (which is what reopen means)
			canReopenAccordingToWorkflow := false
			if tt.workflowConfig != nil && tt.workflowConfig.StatusFlow != nil {
				allowedTransitions := tt.workflowConfig.StatusFlow[string(tt.currentStatus)]
				// Reopen typically means going back to an earlier status
				for _, nextStatus := range allowedTransitions {
					if nextStatus == "in_development" || nextStatus == "in_progress" ||
						nextStatus == "ready_for_development" || nextStatus == "ready_for_refinement" {
						canReopenAccordingToWorkflow = true
						break
					}
				}
			}

			// The hardcoded check in task.go line 1404
			canReopenAccordingToHardcodedLogic := (tt.currentStatus == models.TaskStatusReadyForReview)

			if tt.shouldReopen {
				if !canReopenAccordingToWorkflow {
					t.Fatalf("Test setup error: workflow doesn't allow reopening from %s", tt.currentStatus)
				}

				if !canReopenAccordingToHardcodedLogic {
					t.Errorf("EXPECTED FAILURE (demonstrates hardcoded limitation):\n"+
						"  %s\n"+
						"  Current status: %s\n"+
						"  Workflow config allows reopening: %v\n"+
						"  Hardcoded validation allows reopening: %v\n"+
						"  Problem: Hardcoded check on line 1404 prevents valid workflow transitions",
						tt.description,
						tt.currentStatus,
						canReopenAccordingToWorkflow,
						canReopenAccordingToHardcodedLogic,
					)
				}
			}
		})
	}
}

// TestTaskStartCommand_ShouldUseWorkflowConfig demonstrates that task start should
// check workflow config for allowed starting statuses instead of hardcoded logic.
func TestTaskStartCommand_ShouldUseWorkflowConfig(t *testing.T) {
	tests := []struct {
		name           string
		currentStatus  models.TaskStatus
		shouldStart    bool
		workflowConfig *config.WorkflowConfig
		description    string
	}{
		{
			name:          "start from draft status",
			currentStatus: models.TaskStatus("draft"),
			shouldStart:   true,
			workflowConfig: &config.WorkflowConfig{
				StatusFlow: map[string][]string{
					"draft":                 {"ready_for_refinement"},
					"ready_for_development": {"in_development"},
				},
				SpecialStatuses: map[string][]string{
					"_start_": {"draft", "ready_for_development"},
				},
			},
			description: "Draft should be a valid starting status according to workflow config",
		},
		{
			name:          "start from ready_for_development status",
			currentStatus: models.TaskStatus("ready_for_development"),
			shouldStart:   true,
			workflowConfig: &config.WorkflowConfig{
				StatusFlow: map[string][]string{
					"ready_for_development": {"in_development"},
				},
				SpecialStatuses: map[string][]string{
					"_start_": {"draft", "ready_for_development"},
				},
			},
			description: "ready_for_development should be a valid starting status according to workflow config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Current implementation likely checks for TaskStatusTodo
			// It should instead check workflow's special_statuses._start_ array

			isValidStartStatus := false
			if tt.workflowConfig != nil && tt.workflowConfig.SpecialStatuses != nil {
				startStatuses := tt.workflowConfig.SpecialStatuses["_start_"]
				for _, status := range startStatuses {
					if status == string(tt.currentStatus) {
						isValidStartStatus = true
						break
					}
				}
			}

			if tt.shouldStart && !isValidStartStatus {
				t.Errorf("Workflow config should define %s as a start status", tt.currentStatus)
			}

			t.Logf("Status '%s' should be allowed as starting status per workflow config", tt.currentStatus)
		})
	}
}

// TestRepositoryFallbackTransitions_ShouldBeRemoved documents that the fallback
// hardcoded transitions in task_repository.go (lines 506-530) should be removed.
func TestRepositoryFallbackTransitions_ShouldBeRemoved(t *testing.T) {
	t.Log("DESIGN ISSUE: task_repository.go has fallback hardcoded transitions")
	t.Log("Location: lines 506-530 in isValidTransition method")
	t.Log("Problem: If workflow config is nil, falls back to hardcoded map of transitions")
	t.Log("Solution: Always require workflow config, use default workflow if config file missing")
	t.Log("")
	t.Log("Current fallback code:")
	t.Log("  validTransitions := map[models.TaskStatus][]models.TaskStatus{")
	t.Log("      models.TaskStatusTodo: {models.TaskStatusInProgress, models.TaskStatusBlocked},")
	t.Log("      models.TaskStatusInProgress: {models.TaskStatusReadyForReview, models.TaskStatusBlocked},")
	t.Log("      ...")
	t.Log("  }")
	t.Log("")
	t.Log("This fallback should be removed because:")
	t.Log("  1. Creates dual source of truth (config file + code)")
	t.Log("  2. Prevents using custom workflow statuses")
	t.Log("  3. config.DefaultWorkflow() already provides safe default")
}

// Mock repository for testing command logic without database
type MockTaskRepositoryConfigDriven struct {
	tasks    map[string]*models.Task
	workflow *config.WorkflowConfig
}

func NewMockTaskRepositoryConfigDriven(workflow *config.WorkflowConfig) *MockTaskRepositoryConfigDriven {
	return &MockTaskRepositoryConfigDriven{
		tasks:    make(map[string]*models.Task),
		workflow: workflow,
	}
}

func (m *MockTaskRepositoryConfigDriven) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	task, exists := m.tasks[key]
	if !exists {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func (m *MockTaskRepositoryConfigDriven) AddTask(task *models.Task) {
	m.tasks[task.Key] = task
}

func (m *MockTaskRepositoryConfigDriven) CanTransition(from, to string) bool {
	if m.workflow == nil || m.workflow.StatusFlow == nil {
		return false
	}
	allowedTransitions := m.workflow.StatusFlow[from]
	for _, allowed := range allowedTransitions {
		if allowed == to {
			return true
		}
	}
	return false
}
