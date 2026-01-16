package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// TestWorkflowValidateActions_Integration tests the validate-actions command with a real workflow
func TestWorkflowValidateActions_Integration(t *testing.T) {
	// Load workflow config
	configPath, err := cli.GetConfigPath()
	if err != nil {
		t.Skipf("Skipping integration test: cannot get config path - %v", err)
	}

	workflow, err := config.LoadWorkflowConfig(configPath)
	if err != nil {
		t.Skipf("Skipping integration test: cannot load workflow - %v", err)
	}

	if workflow == nil {
		workflow = config.DefaultWorkflow()
	}

	// Validate
	report := validateWorkflowActions(workflow, false)

	if report == nil {
		t.Fatal("Expected report, got nil")
	}

	// Should have some results
	if len(report.Results) == 0 {
		t.Fatal("Expected results in report")
	}

	// Should be valid (no errors in default workflow)
	if report.ErrorCount > 0 {
		t.Errorf("Unexpected errors in workflow: %d", report.ErrorCount)
	}
}

// TestWorkflowShowActions_Integration tests the show-actions command with a real workflow
func TestWorkflowShowActions_Integration(t *testing.T) {
	// Load workflow config
	configPath, err := cli.GetConfigPath()
	if err != nil {
		t.Skipf("Skipping integration test: cannot get config path - %v", err)
	}

	workflow, err := config.LoadWorkflowConfig(configPath)
	if err != nil {
		t.Skipf("Skipping integration test: cannot load workflow - %v", err)
	}

	if workflow == nil {
		workflow = config.DefaultWorkflow()
	}

	// Build display with no filters
	display := buildActionsDisplay(workflow, "", "")

	if display == nil {
		t.Fatal("Expected display, got nil")
	}

	// Check summary
	if display.Summary.TotalStatuses == 0 {
		t.Error("Expected statuses in workflow")
	}

	// Display should have results if actions are defined
	// (might be empty if no actions configured)
}

// TestWorkflowShowActions_StatusFilter tests filtering by status
func TestWorkflowShowActions_StatusFilter(t *testing.T) {
	workflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation"},
					InstructionTemplate: "Implement {task_id}",
				},
			},
			"ready_for_qa": {
				Phase: "qa",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "qa",
					Skills:              []string{"testing"},
					InstructionTemplate: "Test {task_id}",
				},
			},
		},
	}

	display := buildActionsDisplay(workflow, "ready_for_development", "")

	if len(display.WorkflowActions) != 1 {
		t.Errorf("Expected 1 result with status filter, got %d", len(display.WorkflowActions))
	}

	if display.WorkflowActions[0].Status != "ready_for_development" {
		t.Errorf("Expected ready_for_development, got %s", display.WorkflowActions[0].Status)
	}
}

// TestWorkflowShowActions_ActionTypeFilter tests filtering by action type
func TestWorkflowShowActions_ActionTypeFilter(t *testing.T) {
	workflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation"},
					InstructionTemplate: "Implement {task_id}",
				},
			},
			"completed": {
				Phase: "done",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionArchive,
					InstructionTemplate: "Archive {task_id}",
				},
			},
			"blocked": {
				Phase: "any",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionPause,
					InstructionTemplate: "Task blocked {task_id}",
				},
			},
		},
	}

	// Filter for spawn_agent
	display := buildActionsDisplay(workflow, "", config.ActionSpawnAgent)

	if len(display.WorkflowActions) != 1 {
		t.Errorf("Expected 1 spawn_agent action, got %d", len(display.WorkflowActions))
	}

	// Filter for archive
	display = buildActionsDisplay(workflow, "", config.ActionArchive)

	if len(display.WorkflowActions) != 1 {
		t.Errorf("Expected 1 archive action, got %d", len(display.WorkflowActions))
	}

	if display.WorkflowActions[0].Status != "completed" {
		t.Errorf("Expected completed, got %s", display.WorkflowActions[0].Status)
	}

	// Filter for pause
	display = buildActionsDisplay(workflow, "", config.ActionPause)

	if len(display.WorkflowActions) != 1 {
		t.Errorf("Expected 1 pause action, got %d", len(display.WorkflowActions))
	}

	if display.WorkflowActions[0].Status != "blocked" {
		t.Errorf("Expected blocked, got %s", display.WorkflowActions[0].Status)
	}
}

// TestValidateWorkflowActions_ValidActionWithAllFields validates spawn_agent with all required fields
func TestValidateWorkflowActions_ValidActionWithAllFields(t *testing.T) {
	workflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"test-driven-development", "implementation"},
					InstructionTemplate: "Implement task {task_id} following TDD",
				},
			},
		},
	}

	report := validateWorkflowActions(workflow, false)

	if !report.Valid {
		t.Errorf("Expected valid report")
	}

	if report.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", report.ErrorCount)
	}

	if len(report.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(report.Results))
	}

	result := report.Results[0]
	if result.ActionType != config.ActionSpawnAgent {
		t.Errorf("Expected ActionSpawnAgent, got %s", result.ActionType)
	}

	if result.AgentType != "developer" {
		t.Errorf("Expected developer agent, got %s", result.AgentType)
	}

	if len(result.Skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(result.Skills))
	}
}

// TestValidateWorkflowActions_EmptyTemplate checks empty template validation
func TestValidateWorkflowActions_EmptyTemplate(t *testing.T) {
	workflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation"},
					InstructionTemplate: "", // Empty template
				},
			},
		},
	}

	report := validateWorkflowActions(workflow, false)

	if report.Valid {
		t.Errorf("Expected invalid report due to empty template")
	}

	if report.ErrorCount == 0 {
		t.Errorf("Expected error for empty template")
	}
}
