package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

func TestValidateWorkflowActions_AllValid(t *testing.T) {
	workflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation", "test-driven-development"},
					InstructionTemplate: "Implement task {task_id}",
				},
			},
			"completed": {
				Phase: "done",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionArchive,
					InstructionTemplate: "Task {task_id} complete",
				},
			},
		},
	}

	report := validateWorkflowActions(workflow, false)

	if !report.Valid {
		t.Errorf("Expected valid report, got invalid")
	}
	if report.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", report.ErrorCount)
	}
	if report.ValidCount != 2 {
		t.Errorf("Expected 2 valid statuses, got %d", report.ValidCount)
	}
}

func TestValidateWorkflowActions_MissingActionable(t *testing.T) {
	workflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				// No orchestrator action
			},
			"todo": {
				Phase: "planning",
				// No orchestrator action (not actionable)
			},
		},
	}

	report := validateWorkflowActions(workflow, false)

	if report.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", report.ErrorCount)
	}
	if report.WarningCount != 1 {
		t.Errorf("Expected 1 warning (ready_for_development), got %d", report.WarningCount)
	}
	// Report is valid if no errors and (not strict or no warnings)
	// Since we have a warning but not strict mode, it's valid
	if !report.Valid {
		t.Errorf("Report should be valid in non-strict mode with only warnings")
	}
}

func TestValidateWorkflowActions_StrictMode(t *testing.T) {
	workflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				// No orchestrator action
			},
		},
	}

	report := validateWorkflowActions(workflow, true)

	if report.Valid {
		t.Errorf("Expected invalid report in strict mode with warnings")
	}
	if report.WarningCount != 1 {
		t.Errorf("Expected 1 warning, got %d", report.WarningCount)
	}
}

func TestValidateWorkflowActions_InvalidSchema(t *testing.T) {
	workflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:    config.ActionSpawnAgent,
					AgentType: "developer",
					// Missing skills array
					InstructionTemplate: "Implement {task_id}",
				},
			},
		},
	}

	report := validateWorkflowActions(workflow, false)

	if report.Valid {
		t.Errorf("Expected invalid report due to missing skills")
	}
	if report.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", report.ErrorCount)
	}
	if report.Results[0].Severity != "error" {
		t.Errorf("Expected error severity, got %s", report.Results[0].Severity)
	}
}

func TestValidateStatusAction_ValidSpawnAgent(t *testing.T) {
	metadata := &config.StatusMetadata{
		Phase: "development",
		OrchestratorAction: &config.OrchestratorAction{
			Action:              config.ActionSpawnAgent,
			AgentType:           "developer",
			Skills:              []string{"implementation"},
			InstructionTemplate: "Implement {task_id}",
		},
	}

	result := validateStatusAction("ready_for_development", metadata, false)

	if !result.Valid {
		t.Errorf("Expected valid result")
	}
	if result.Severity != "" {
		t.Errorf("Expected no severity, got %s", result.Severity)
	}
	if result.ActionType != config.ActionSpawnAgent {
		t.Errorf("Expected ActionSpawnAgent, got %s", result.ActionType)
	}
}

func TestValidateStatusAction_MissingActionable(t *testing.T) {
	metadata := &config.StatusMetadata{
		Phase: "development",
		// No action
	}

	result := validateStatusAction("ready_for_development", metadata, false)

	if result.Valid {
		t.Errorf("Expected invalid result for missing actionable action")
	}
	if result.Severity != "warning" {
		t.Errorf("Expected warning severity, got %s", result.Severity)
	}
}

func TestValidateStatusAction_NonActionableMissingAction(t *testing.T) {
	metadata := &config.StatusMetadata{
		Phase: "planning",
		// No action
	}

	result := validateStatusAction("todo", metadata, false)

	if !result.Valid {
		t.Errorf("Expected valid for non-actionable status without action in non-strict mode")
	}
}

func TestValidateStatusAction_MissingAgentType(t *testing.T) {
	metadata := &config.StatusMetadata{
		Phase: "development",
		OrchestratorAction: &config.OrchestratorAction{
			Action: config.ActionSpawnAgent,
			// Missing agent_type
			Skills:              []string{"implementation"},
			InstructionTemplate: "Implement {task_id}",
		},
	}

	result := validateStatusAction("ready_for_development", metadata, false)

	if result.Valid {
		t.Errorf("Expected invalid result due to missing agent_type")
	}
	if result.Severity != "error" {
		t.Errorf("Expected error severity, got %s", result.Severity)
	}
}

func TestValidateWorkflowActions_MixedValidity(t *testing.T) {
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
				// Missing action (warning)
			},
			"ready_for_approval": {
				Phase: "approval",
				OrchestratorAction: &config.OrchestratorAction{
					Action: config.ActionSpawnAgent,
					// Missing agent_type (error)
					Skills:              []string{"quality"},
					InstructionTemplate: "Review {task_id}",
				},
			},
			"completed": {
				Phase: "done",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionArchive,
					InstructionTemplate: "Archive {task_id}",
				},
			},
		},
	}

	report := validateWorkflowActions(workflow, false)

	if report.ValidCount != 2 {
		t.Errorf("Expected 2 valid, got %d", report.ValidCount)
	}
	if report.WarningCount != 1 {
		t.Errorf("Expected 1 warning, got %d", report.WarningCount)
	}
	if report.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", report.ErrorCount)
	}
	if report.Valid {
		t.Errorf("Expected invalid due to error")
	}
}

func TestValidateWorkflowActions_NoActions(t *testing.T) {
	workflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"todo": {
				Phase: "planning",
				// No action
			},
			"in_progress": {
				Phase: "development",
				// No action
			},
		},
	}

	report := validateWorkflowActions(workflow, false)

	// No errors, no warnings (non-actionable statuses without actions are OK), so report should be valid
	if !report.Valid {
		t.Errorf("Expected valid (non-actionable statuses without actions are allowed)")
	}
	if report.ValidCount != 2 {
		t.Errorf("Expected 2 valid (non-actionable), got %d", report.ValidCount)
	}
}

func TestValidationReport_JSON(t *testing.T) {
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
		},
	}

	report := validateWorkflowActions(workflow, false)

	if report.TotalStatuses != 1 {
		t.Errorf("Expected 1 total status, got %d", report.TotalStatuses)
	}
	if len(report.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(report.Results))
	}

	// Check result has all expected fields
	result := report.Results[0]
	if result.Status != "ready_for_development" {
		t.Errorf("Expected status ready_for_development, got %s", result.Status)
	}
	if result.ActionType != config.ActionSpawnAgent {
		t.Errorf("Expected ActionSpawnAgent, got %s", result.ActionType)
	}
	if result.AgentType != "developer" {
		t.Errorf("Expected developer agent type, got %s", result.AgentType)
	}
}
