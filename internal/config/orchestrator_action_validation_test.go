package config

import (
	"strings"
	"testing"
)

// Tests in this file verify backward-compatible ValidateAllOrchestratorActions
// which accepts map[string]StatusMetadata (the original API).
// Pure unit tests for OrchestratorAction validation have moved to internal/config/action/.

// TestValidateAllOrchestratorActions tests the backward-compatible wrapper
func TestValidateAllOrchestratorActions(t *testing.T) {
	statusMetadata := map[string]StatusMetadata{
		"ready_for_development": {
			OrchestratorAction: &OrchestratorAction{
				Action:              ActionSpawnAgent,
				AgentType:           "developer",
				Skills:              []string{"implementation"},
				InstructionTemplate: "Implement task {task_id}",
			},
		},
		"invalid_status": {
			OrchestratorAction: &OrchestratorAction{
				Action:              "invalid_action",
				InstructionTemplate: "Test",
			},
		},
	}

	errors := ValidateAllOrchestratorActions(statusMetadata)

	if len(errors) == 0 {
		t.Fatal("Expected validation error for invalid_action, got none")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Error(), "invalid_status") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected error mentioning invalid_status, got: %v", errors)
	}
}

// TestValidateAllOrchestratorActions_AllValid tests with all valid actions
func TestValidateAllOrchestratorActions_AllValid(t *testing.T) {
	statusMetadata := map[string]StatusMetadata{
		"ready_for_development": {
			OrchestratorAction: &OrchestratorAction{
				Action:              ActionSpawnAgent,
				AgentType:           "developer",
				Skills:              []string{"implementation"},
				InstructionTemplate: "Implement task {task_id}",
			},
		},
		"on_hold": {
			OrchestratorAction: &OrchestratorAction{
				Action:              ActionPause,
				InstructionTemplate: "Task paused",
			},
		},
	}

	errors := ValidateAllOrchestratorActions(statusMetadata)

	if len(errors) > 0 {
		t.Errorf("Expected no errors for all valid actions, got: %v", errors)
	}
}

// TestValidateAllOrchestratorActions_NoActions tests with no orchestrator actions
func TestValidateAllOrchestratorActions_NoActions(t *testing.T) {
	statusMetadata := map[string]StatusMetadata{
		"todo": {
			Color:       "gray",
			Description: "Task ready to start",
		},
		"in_progress": {
			Color:       "blue",
			Description: "Task in progress",
		},
	}

	errors := ValidateAllOrchestratorActions(statusMetadata)

	if len(errors) > 0 {
		t.Errorf("Expected no errors when no orchestrator actions, got: %v", errors)
	}
}
