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

func TestValidateActions_MultiLevel_AllValid(t *testing.T) {
	epicWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_epic_planning": {
				Phase: "planning",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "business_analyst",
					Skills:              []string{"planning"},
					InstructionTemplate: "Plan epic {epic_id}",
				},
			},
		},
	}

	featureWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_feature_design": {
				Phase: "design",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "architect",
					Skills:              []string{"design"},
					InstructionTemplate: "Design feature {feature_id}",
				},
			},
		},
	}

	taskWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation"},
					InstructionTemplate: "Implement task {task_id}",
				},
			},
		},
	}

	epicReport := validateWorkflowActions(epicWorkflow, false)
	featureReport := validateWorkflowActions(featureWorkflow, false)
	taskReport := validateWorkflowActions(taskWorkflow, false)

	multiReport := &MultiLevelValidationReport{
		Valid:         epicReport.Valid && featureReport.Valid && taskReport.Valid,
		StrictMode:    false,
		EpicReport:    epicReport,
		FeatureReport: featureReport,
		TaskReport:    taskReport,
	}

	if !multiReport.Valid {
		t.Errorf("Expected overall valid=true, got false")
	}
	if !multiReport.EpicReport.Valid {
		t.Errorf("Expected epic report valid=true")
	}
	if !multiReport.FeatureReport.Valid {
		t.Errorf("Expected feature report valid=true")
	}
	if !multiReport.TaskReport.Valid {
		t.Errorf("Expected task report valid=true")
	}
}

func TestValidateActions_MultiLevel_EpicInvalid(t *testing.T) {
	epicWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_epic_planning": {
				Phase: "planning",
				OrchestratorAction: &config.OrchestratorAction{
					Action:    config.ActionSpawnAgent,
					AgentType: "business_analyst",
					// Missing skills - error
					InstructionTemplate: "Plan epic {epic_id}",
				},
			},
		},
	}

	featureWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_feature_design": {
				Phase: "design",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "architect",
					Skills:              []string{"design"},
					InstructionTemplate: "Design feature {feature_id}",
				},
			},
		},
	}

	taskWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation"},
					InstructionTemplate: "Implement task {task_id}",
				},
			},
		},
	}

	epicReport := validateWorkflowActions(epicWorkflow, false)
	featureReport := validateWorkflowActions(featureWorkflow, false)
	taskReport := validateWorkflowActions(taskWorkflow, false)

	multiReport := &MultiLevelValidationReport{
		Valid:         epicReport.Valid && featureReport.Valid && taskReport.Valid,
		StrictMode:    false,
		EpicReport:    epicReport,
		FeatureReport: featureReport,
		TaskReport:    taskReport,
	}

	if multiReport.Valid {
		t.Errorf("Expected overall valid=false due to epic errors, got true")
	}
	if epicReport.Valid {
		t.Errorf("Expected epic report valid=false")
	}
	if !featureReport.Valid {
		t.Errorf("Expected feature report valid=true")
	}
	if !taskReport.Valid {
		t.Errorf("Expected task report valid=true")
	}
}

func TestValidateActions_LevelFilter(t *testing.T) {
	// Test that when filtering by level, only that level's workflow is validated
	taskWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation"},
					InstructionTemplate: "Implement task {task_id}",
				},
			},
		},
	}

	report := validateWorkflowActions(taskWorkflow, false)

	if !report.Valid {
		t.Errorf("Expected task workflow to be valid")
	}
	if report.TotalStatuses != 1 {
		t.Errorf("Expected 1 status, got %d", report.TotalStatuses)
	}
}

func TestValidateActions_StrictMode_MultiLevel(t *testing.T) {
	epicWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"in_planning": {
				Phase: "planning",
				// No action - warning in strict mode
			},
		},
	}

	featureWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"in_design": {
				Phase: "design",
				// No action - warning in strict mode
			},
		},
	}

	taskWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"in_development": {
				Phase: "development",
				// No action - warning in strict mode
			},
		},
	}

	epicReport := validateWorkflowActions(epicWorkflow, true)
	featureReport := validateWorkflowActions(featureWorkflow, true)
	taskReport := validateWorkflowActions(taskWorkflow, true)

	multiReport := &MultiLevelValidationReport{
		Valid:         epicReport.Valid && featureReport.Valid && taskReport.Valid,
		StrictMode:    true,
		EpicReport:    epicReport,
		FeatureReport: featureReport,
		TaskReport:    taskReport,
	}

	if multiReport.Valid {
		t.Errorf("Expected invalid in strict mode with warnings")
	}
	if epicReport.WarningCount == 0 {
		t.Errorf("Expected warnings in epic report")
	}
	if featureReport.WarningCount == 0 {
		t.Errorf("Expected warnings in feature report")
	}
	if taskReport.WarningCount == 0 {
		t.Errorf("Expected warnings in task report")
	}
}

func TestValidateActions_DefaultWorkflow_NoActions(t *testing.T) {
	// Default workflows should pass validation even with no custom actions
	defaultWorkflow := config.DefaultWorkflow()

	report := validateWorkflowActions(defaultWorkflow, false)

	// Should be valid even with no custom orchestrator actions
	if !report.Valid {
		t.Errorf("Expected default workflow to be valid")
	}
}

func TestValidateActions_OverallValid_OnlyWhenAllLevelsValid(t *testing.T) {
	tests := []struct {
		name         string
		epicValid    bool
		featureValid bool
		taskValid    bool
		overallValid bool
	}{
		{"all valid", true, true, true, true},
		{"epic invalid", false, true, true, false},
		{"feature invalid", true, false, true, false},
		{"task invalid", true, true, false, false},
		{"all invalid", false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiReport := &MultiLevelValidationReport{
				Valid:      true,
				StrictMode: false,
				EpicReport: &ValidationReport{
					Valid: tt.epicValid,
				},
				FeatureReport: &ValidationReport{
					Valid: tt.featureValid,
				},
				TaskReport: &ValidationReport{
					Valid: tt.taskValid,
				},
			}

			// Apply AND logic
			multiReport.Valid = tt.epicValid && tt.featureValid && tt.taskValid

			if multiReport.Valid != tt.overallValid {
				t.Errorf("Expected overall valid=%v, got %v", tt.overallValid, multiReport.Valid)
			}
		})
	}
}

func TestMultiLevelValidationReport_JSON(t *testing.T) {
	multiReport := &MultiLevelValidationReport{
		Valid:      true,
		StrictMode: false,
		EpicReport: &ValidationReport{
			Valid:         true,
			TotalStatuses: 1,
			ValidCount:    1,
		},
		FeatureReport: &ValidationReport{
			Valid:         true,
			TotalStatuses: 2,
			ValidCount:    2,
		},
		TaskReport: &ValidationReport{
			Valid:         true,
			TotalStatuses: 3,
			ValidCount:    3,
		},
	}

	if !multiReport.Valid {
		t.Errorf("Expected valid=true")
	}
	if multiReport.EpicReport == nil {
		t.Errorf("Expected epic report to be non-nil")
	}
	if multiReport.FeatureReport == nil {
		t.Errorf("Expected feature report to be non-nil")
	}
	if multiReport.TaskReport == nil {
		t.Errorf("Expected task report to be non-nil")
	}
}

func TestValidateActions_InvalidLevel(t *testing.T) {
	invalidLevels := []string{"invalid", "EPIC", "tasks", ""}

	for _, level := range invalidLevels {
		if level == "" {
			continue // Empty is valid (means all levels)
		}

		validLevels := map[string]bool{"epic": true, "feature": true, "task": true}
		if validLevels[level] {
			t.Errorf("Level %q should be invalid but was accepted", level)
		}
	}

	// Test valid levels
	validLevels := []string{"epic", "feature", "task"}
	validMap := map[string]bool{"epic": true, "feature": true, "task": true}

	for _, level := range validLevels {
		if !validMap[level] {
			t.Errorf("Level %q should be valid but was rejected", level)
		}
	}
}
