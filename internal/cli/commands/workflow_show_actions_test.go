package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

func TestBuildActionsDisplay_AllActions(t *testing.T) {
	workflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				Color: "yellow",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation"},
					InstructionTemplate: "Implement {task_id}",
				},
			},
			"ready_for_qa": {
				Phase: "qa",
				Color: "green",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "qa",
					Skills:              []string{"quality"},
					InstructionTemplate: "Test {task_id}",
				},
			},
			"completed": {
				Phase: "done",
				Color: "blue",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionArchive,
					InstructionTemplate: "Archive {task_id}",
				},
			},
			"todo": {
				Phase: "planning",
				// No action
			},
		},
	}

	display := buildActionsDisplay(workflow, "", "")

	if len(display.WorkflowActions) != 3 {
		t.Errorf("Expected 3 actions, got %d", len(display.WorkflowActions))
	}
	if display.Summary.TotalStatuses != 4 {
		t.Errorf("Expected 4 total statuses, got %d", display.Summary.TotalStatuses)
	}
	if display.Summary.StatusesWithActions != 3 {
		t.Errorf("Expected 3 statuses with actions, got %d", display.Summary.StatusesWithActions)
	}
}

func TestBuildActionsDisplay_FilterByStatus(t *testing.T) {
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
					Skills:              []string{"quality"},
					InstructionTemplate: "Test {task_id}",
				},
			},
		},
	}

	display := buildActionsDisplay(workflow, "ready_for_development", "")

	if len(display.WorkflowActions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(display.WorkflowActions))
	}
	if display.WorkflowActions[0].Status != "ready_for_development" {
		t.Errorf("Expected ready_for_development, got %s", display.WorkflowActions[0].Status)
	}
}

func TestBuildActionsDisplay_FilterByActionType(t *testing.T) {
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

	// Filter for spawn_agent actions
	display := buildActionsDisplay(workflow, "", config.ActionSpawnAgent)

	if len(display.WorkflowActions) != 1 {
		t.Errorf("Expected 1 spawn_agent action, got %d", len(display.WorkflowActions))
	}
	if display.WorkflowActions[0].OrchestratorAction.Action != config.ActionSpawnAgent {
		t.Errorf("Expected ActionSpawnAgent, got %s", display.WorkflowActions[0].OrchestratorAction.Action)
	}

	// Filter for archive actions
	display = buildActionsDisplay(workflow, "", config.ActionArchive)

	if len(display.WorkflowActions) != 1 {
		t.Errorf("Expected 1 archive action, got %d", len(display.WorkflowActions))
	}
	if display.WorkflowActions[0].Status != "completed" {
		t.Errorf("Expected completed status, got %s", display.WorkflowActions[0].Status)
	}
}

func TestBuildActionsDisplay_NoActions(t *testing.T) {
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

	display := buildActionsDisplay(workflow, "", "")

	if len(display.WorkflowActions) != 0 {
		t.Errorf("Expected 0 actions, got %d", len(display.WorkflowActions))
	}
	if display.Summary.StatusesWithActions != 0 {
		t.Errorf("Expected 0 statuses with actions, got %d", display.Summary.StatusesWithActions)
	}
}

func TestBuildActionsDisplay_ActionCounts(t *testing.T) {
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
					Skills:              []string{"quality"},
					InstructionTemplate: "Test {task_id}",
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

	display := buildActionsDisplay(workflow, "", "")

	if display.Summary.ActionCounts[config.ActionSpawnAgent] != 2 {
		t.Errorf("Expected 2 spawn_agent actions, got %d", display.Summary.ActionCounts[config.ActionSpawnAgent])
	}
	if display.Summary.ActionCounts[config.ActionArchive] != 1 {
		t.Errorf("Expected 1 archive action, got %d", display.Summary.ActionCounts[config.ActionArchive])
	}
}

func TestGroupByPhase(t *testing.T) {
	actions := []StatusActionDisplay{
		{
			Status: "ready_for_development",
			Phase:  "development",
		},
		{
			Status: "ready_for_qa",
			Phase:  "qa",
		},
		{
			Status: "draft",
			Phase:  "planning",
		},
		{
			Status: "completed",
			Phase:  "done",
		},
	}

	grouped := groupByPhase(actions)

	// Check all phases are present
	if len(grouped) != 4 {
		t.Errorf("Expected 4 phases, got %d", len(grouped))
	}

	// Check each phase has correct count
	if len(grouped["planning"]) != 1 {
		t.Errorf("Expected 1 planning status, got %d", len(grouped["planning"]))
	}
	if len(grouped["development"]) != 1 {
		t.Errorf("Expected 1 development status, got %d", len(grouped["development"]))
	}
	if len(grouped["qa"]) != 1 {
		t.Errorf("Expected 1 qa status, got %d", len(grouped["qa"]))
	}
	if len(grouped["done"]) != 1 {
		t.Errorf("Expected 1 done status, got %d", len(grouped["done"]))
	}

	// Check sorting within groups
	multiPhaseActions := []StatusActionDisplay{
		{Status: "z_status", Phase: "development"},
		{Status: "a_status", Phase: "development"},
		{Status: "m_status", Phase: "development"},
	}

	grouped = groupByPhase(multiPhaseActions)
	devStatuses := grouped["development"]

	if devStatuses[0].Status != "a_status" {
		t.Errorf("Expected a_status first, got %s", devStatuses[0].Status)
	}
	if devStatuses[1].Status != "m_status" {
		t.Errorf("Expected m_status second, got %s", devStatuses[1].Status)
	}
	if devStatuses[2].Status != "z_status" {
		t.Errorf("Expected z_status third, got %s", devStatuses[2].Status)
	}
}

func TestGroupByPhase_UnspecifiedPhase(t *testing.T) {
	actions := []StatusActionDisplay{
		{
			Status: "status_no_phase",
			Phase:  "", // Empty phase
		},
	}

	grouped := groupByPhase(actions)

	// Should group into "any" when phase is empty
	if len(grouped["any"]) != 1 {
		t.Errorf("Expected 1 status in 'any' phase, got %d", len(grouped["any"]))
	}
	if grouped["any"][0].Status != "status_no_phase" {
		t.Errorf("Expected status_no_phase, got %s", grouped["any"][0].Status)
	}
}
