package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// TestDisplayOrchestratorAction_Nil verifies that calling displayOrchestratorAction
// with nil does not panic and outputs the "None configured" fallback.
func TestDisplayOrchestratorAction_Nil(t *testing.T) {
	// Should not panic when called with nil
	displayOrchestratorAction(nil)
}

// TestDisplayOrchestratorAction_WithAction verifies that a fully populated action
// is displayed without panicking.
func TestDisplayOrchestratorAction_WithAction(t *testing.T) {
	action := &config.PopulatedAction{
		Action:      config.ActionSpawnAgent,
		AgentType:   "developer",
		Skills:      []string{"backend", "testing"},
		Instruction: "Implement T-E99-F01-001 following the specification",
	}

	// Should not panic with a full action
	displayOrchestratorAction(action)
}

// TestDisplayOrchestratorAction_PartialAction verifies that a partial action
// (empty AgentType, no Skills) is displayed without panicking.
func TestDisplayOrchestratorAction_PartialAction(t *testing.T) {
	action := &config.PopulatedAction{
		Action:      config.ActionPause,
		AgentType:   "",
		Skills:      nil,
		Instruction: "Wait for manual review before proceeding",
	}

	// Should not panic with partial fields
	displayOrchestratorAction(action)
}

// TestResolveAction_WithOrchestratorAction tests that resolveAction returns a PopulatedAction when status has an orchestrator_action configured
func TestResolveAction_WithOrchestratorAction(t *testing.T) {
	// Construct test workflow config with orchestrator action
	wf := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Color:       "yellow",
				Description: "Ready for development",
				Phase:       "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation", "test-driven-development"},
					InstructionTemplate: "Implement task {id}: {title}",
				},
			},
		},
	}

	placeholders := map[string]string{
		"id":    "T-E07-F28-001",
		"title": "Add orchestrator action to task get output",
	}

	// Call resolveAction
	action := resolveAction(wf, "ready_for_development", placeholders)

	// Verify action is populated
	if action == nil {
		t.Fatal("expected PopulatedAction, got nil")
	}

	if action.Action != config.ActionSpawnAgent {
		t.Errorf("expected action=%s, got %s", config.ActionSpawnAgent, action.Action)
	}

	if action.AgentType != "developer" {
		t.Errorf("expected agent_type=developer, got %s", action.AgentType)
	}

	if len(action.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(action.Skills))
	}

	expectedInstruction := "Implement task T-E07-F28-001: Add orchestrator action to task get output"
	if action.Instruction != expectedInstruction {
		t.Errorf("expected instruction=%s, got %s", expectedInstruction, action.Instruction)
	}
}

// TestResolveAction_NoAction tests that resolveAction returns nil when status has no action
func TestResolveAction_NoAction(t *testing.T) {
	wf := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"todo": {
				Color:              "gray",
				Description:        "Task not started",
				Phase:              "planning",
				OrchestratorAction: nil, // No action configured
			},
		},
	}

	placeholders := map[string]string{
		"id":    "T-E07-F28-001",
		"title": "Some task",
	}

	action := resolveAction(wf, "todo", placeholders)

	if action != nil {
		t.Errorf("expected nil action, got %+v", action)
	}
}

// TestResolveAction_NilWorkflow tests that resolveAction returns nil when workflow is nil
func TestResolveAction_NilWorkflow(t *testing.T) {
	action := resolveAction(nil, "ready_for_development", map[string]string{})

	if action != nil {
		t.Errorf("expected nil action for nil workflow, got %+v", action)
	}
}

// TestResolveAction_StatusNotFound tests that resolveAction returns nil when status is not in metadata
func TestResolveAction_StatusNotFound(t *testing.T) {
	wf := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"todo": {
				Color: "gray",
			},
		},
	}

	action := resolveAction(wf, "nonexistent_status", map[string]string{})

	if action != nil {
		t.Errorf("expected nil action for nonexistent status, got %+v", action)
	}
}

// TestResolveAction_MultipleStatuses tests resolveAction with multiple statuses
func TestResolveAction_MultipleStatuses(t *testing.T) {
	wf := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation"},
					InstructionTemplate: "Start work on {id}",
				},
			},
			"ready_for_review": {
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "tech-lead",
					Skills:              []string{"code-review"},
					InstructionTemplate: "Review task {id}",
				},
			},
			"completed": {
				OrchestratorAction: nil, // No action for completed
			},
		},
	}

	testCases := []struct {
		status            string
		expectedAction    string
		expectedAgentType string
		expectedNil       bool
	}{
		{"ready_for_development", config.ActionSpawnAgent, "developer", false},
		{"ready_for_review", config.ActionSpawnAgent, "tech-lead", false},
		{"completed", "", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.status, func(t *testing.T) {
			placeholders := map[string]string{"id": "T-E07-F28-001"}
			action := resolveAction(wf, tc.status, placeholders)

			if tc.expectedNil {
				if action != nil {
					t.Errorf("expected nil action for status %s, got %+v", tc.status, action)
				}
				return
			}

			if action == nil {
				t.Fatalf("expected action for status %s, got nil", tc.status)
			}

			if action.Action != tc.expectedAction {
				t.Errorf("expected action=%s, got %s", tc.expectedAction, action.Action)
			}

			if action.AgentType != tc.expectedAgentType {
				t.Errorf("expected agent_type=%s, got %s", tc.expectedAgentType, action.AgentType)
			}
		})
	}
}

// TestResolveAction_PauseAction tests non-spawn_agent actions
func TestResolveAction_PauseAction(t *testing.T) {
	wf := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"blocked": {
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionPause,
					InstructionTemplate: "Task {id} is blocked. Wait for dependencies.",
				},
			},
		},
	}

	placeholders := map[string]string{"id": "T-E07-F28-001"}
	action := resolveAction(wf, "blocked", placeholders)

	if action == nil {
		t.Fatal("expected action, got nil")
	}

	if action.Action != config.ActionPause {
		t.Errorf("expected action=pause, got %s", action.Action)
	}

	// For pause action, AgentType and Skills should be empty
	if action.AgentType != "" {
		t.Errorf("expected empty agent_type for pause action, got %s", action.AgentType)
	}

	if len(action.Skills) != 0 {
		t.Errorf("expected empty skills for pause action, got %v", action.Skills)
	}

	expectedInstruction := "Task T-E07-F28-001 is blocked. Wait for dependencies."
	if action.Instruction != expectedInstruction {
		t.Errorf("expected instruction=%s, got %s", expectedInstruction, action.Instruction)
	}
}

// TestResolveAction_WithRelatedPlaceholders tests that resolveAction works with related_docs and related_tasks placeholders
func TestResolveAction_WithRelatedPlaceholders(t *testing.T) {
	wf := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation", "test-driven-development"},
					InstructionTemplate: "Implement {id}: {title}. Read: {related_docs}. Dependencies: {related_tasks}",
				},
			},
		},
	}

	placeholders := map[string]string{
		"id":            "E07-F29-001",
		"title":         "Add template variables feature",
		"related_docs":  "docs/spec.md,docs/design.md",
		"related_tasks": "E07-F05-001,E10-F05-002",
	}

	action := resolveAction(wf, "ready_for_development", placeholders)

	if action == nil {
		t.Fatal("expected PopulatedAction, got nil")
	}

	// Verify that placeholders with related docs and tasks are populated correctly
	expectedInstruction := "Implement E07-F29-001: Add template variables feature. Read: docs/spec.md,docs/design.md. Dependencies: E07-F05-001,E10-F05-002"
	if action.Instruction != expectedInstruction {
		t.Errorf("expected instruction=%s, got %s", expectedInstruction, action.Instruction)
	}
}

// TestResolveAction_WithEmptyRelatedPlaceholders tests graceful handling of empty related placeholders
func TestResolveAction_WithEmptyRelatedPlaceholders(t *testing.T) {
	wf := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation"},
					InstructionTemplate: "Implement {id}. Docs: {related_docs}. Tasks: {related_tasks}",
				},
			},
		},
	}

	placeholders := map[string]string{
		"id":            "E07-F29-002",
		"related_docs":  "",
		"related_tasks": "",
	}

	action := resolveAction(wf, "ready_for_development", placeholders)

	if action == nil {
		t.Fatal("expected PopulatedAction, got nil")
	}

	// Empty placeholders should be replaced with empty strings, resulting in valid but sparse instruction
	expectedInstruction := "Implement E07-F29-002. Docs: . Tasks: "
	if action.Instruction != expectedInstruction {
		t.Errorf("expected instruction=%s, got %s", expectedInstruction, action.Instruction)
	}
}
