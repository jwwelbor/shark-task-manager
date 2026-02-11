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

func TestBuildMultiLevelSummary(t *testing.T) {
	display := &MultiLevelActionsDisplay{
		EpicActions: &WorkflowActionsDisplay{
			Summary: ActionsSummary{
				TotalStatuses:       4,
				StatusesWithActions: 2,
				ActionCounts:        map[string]int{"spawn_agent": 2},
			},
		},
		FeatureActions: &WorkflowActionsDisplay{
			Summary: ActionsSummary{
				TotalStatuses:       5,
				StatusesWithActions: 3,
				ActionCounts:        map[string]int{"spawn_agent": 2, "archive": 1},
			},
		},
		TaskActions: &WorkflowActionsDisplay{
			Summary: ActionsSummary{
				TotalStatuses:       19,
				StatusesWithActions: 8,
				ActionCounts:        map[string]int{"spawn_agent": 6, "archive": 1, "pause": 1},
			},
		},
	}

	summary := buildMultiLevelSummary(display)

	if summary.EpicTotal != 4 {
		t.Errorf("Expected EpicTotal=4, got %d", summary.EpicTotal)
	}
	if summary.EpicWithActions != 2 {
		t.Errorf("Expected EpicWithActions=2, got %d", summary.EpicWithActions)
	}
	if summary.FeatureTotal != 5 {
		t.Errorf("Expected FeatureTotal=5, got %d", summary.FeatureTotal)
	}
	if summary.FeatureWithActions != 3 {
		t.Errorf("Expected FeatureWithActions=3, got %d", summary.FeatureWithActions)
	}
	if summary.TaskTotal != 19 {
		t.Errorf("Expected TaskTotal=19, got %d", summary.TaskTotal)
	}
	if summary.TaskWithActions != 8 {
		t.Errorf("Expected TaskWithActions=8, got %d", summary.TaskWithActions)
	}
}

func TestBuildMultiLevelSummary_NilLevels(t *testing.T) {
	display := &MultiLevelActionsDisplay{
		EpicActions: &WorkflowActionsDisplay{
			Summary: ActionsSummary{
				TotalStatuses:       4,
				StatusesWithActions: 1,
				ActionCounts:        map[string]int{"spawn_agent": 1},
			},
		},
		// FeatureActions is nil
		// TaskActions is nil
	}

	summary := buildMultiLevelSummary(display)

	if summary.EpicTotal != 4 {
		t.Errorf("Expected EpicTotal=4, got %d", summary.EpicTotal)
	}
	if summary.FeatureTotal != 0 {
		t.Errorf("Expected FeatureTotal=0, got %d", summary.FeatureTotal)
	}
	if summary.TaskTotal != 0 {
		t.Errorf("Expected TaskTotal=0, got %d", summary.TaskTotal)
	}
}

func TestShowActions_MultiLevel_AllLevels(t *testing.T) {
	epicWf := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"draft": {Phase: "planning"},
			"active": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "researcher",
					Skills:              []string{"research"},
					InstructionTemplate: "Research {id}",
				},
			},
		},
	}

	featureWf := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"draft":  {Phase: "planning"},
			"active": {Phase: "development"},
		},
	}

	taskWf := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"todo": {Phase: "planning"},
			"ready_for_development": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"implementation"},
					InstructionTemplate: "Implement {task_id}",
				},
			},
			"completed": {Phase: "done"},
		},
	}

	display := &MultiLevelActionsDisplay{}
	display.EpicActions = buildActionsDisplay(epicWf, "", "")
	display.FeatureActions = buildActionsDisplay(featureWf, "", "")
	display.TaskActions = buildActionsDisplay(taskWf, "", "")
	display.Summary = buildMultiLevelSummary(display)

	// Epic: 1 of 2 statuses have actions
	if display.Summary.EpicTotal != 2 {
		t.Errorf("Expected EpicTotal=2, got %d", display.Summary.EpicTotal)
	}
	if display.Summary.EpicWithActions != 1 {
		t.Errorf("Expected EpicWithActions=1, got %d", display.Summary.EpicWithActions)
	}

	// Feature: 0 of 2 statuses have actions
	if display.Summary.FeatureTotal != 2 {
		t.Errorf("Expected FeatureTotal=2, got %d", display.Summary.FeatureTotal)
	}
	if display.Summary.FeatureWithActions != 0 {
		t.Errorf("Expected FeatureWithActions=0, got %d", display.Summary.FeatureWithActions)
	}

	// Task: 1 of 3 statuses have actions
	if display.Summary.TaskTotal != 3 {
		t.Errorf("Expected TaskTotal=3, got %d", display.Summary.TaskTotal)
	}
	if display.Summary.TaskWithActions != 1 {
		t.Errorf("Expected TaskWithActions=1, got %d", display.Summary.TaskWithActions)
	}
}

func TestShowActions_LevelFilter_Epic(t *testing.T) {
	epicWf := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"active": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "researcher",
					Skills:              []string{"research"},
					InstructionTemplate: "Research {id}",
				},
			},
		},
	}

	// Simulate --level=epic: only epic actions populated
	display := &MultiLevelActionsDisplay{}
	display.EpicActions = buildActionsDisplay(epicWf, "", "")
	display.Summary = buildMultiLevelSummary(display)

	if display.EpicActions == nil {
		t.Fatal("Expected EpicActions to be non-nil")
	}
	if display.FeatureActions != nil {
		t.Error("Expected FeatureActions to be nil")
	}
	if display.TaskActions != nil {
		t.Error("Expected TaskActions to be nil")
	}
	if display.Summary.EpicWithActions != 1 {
		t.Errorf("Expected EpicWithActions=1, got %d", display.Summary.EpicWithActions)
	}
}

func TestShowActions_LevelFilter_Feature(t *testing.T) {
	featureWf := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"active": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "architect",
					Skills:              []string{"architecture"},
					InstructionTemplate: "Review {id}",
				},
			},
			"draft": {Phase: "planning"},
		},
	}

	// Simulate --level=feature
	display := &MultiLevelActionsDisplay{}
	display.FeatureActions = buildActionsDisplay(featureWf, "", "")
	display.Summary = buildMultiLevelSummary(display)

	if display.EpicActions != nil {
		t.Error("Expected EpicActions to be nil")
	}
	if display.FeatureActions == nil {
		t.Fatal("Expected FeatureActions to be non-nil")
	}
	if display.TaskActions != nil {
		t.Error("Expected TaskActions to be nil")
	}
	if display.Summary.FeatureWithActions != 1 {
		t.Errorf("Expected FeatureWithActions=1, got %d", display.Summary.FeatureWithActions)
	}
}

func TestShowActions_LevelFilter_Task(t *testing.T) {
	taskWf := &config.WorkflowConfig{
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
			"todo": {Phase: "planning"},
		},
	}

	// Simulate --level=task
	display := &MultiLevelActionsDisplay{}
	display.TaskActions = buildActionsDisplay(taskWf, "", "")
	display.Summary = buildMultiLevelSummary(display)

	if display.EpicActions != nil {
		t.Error("Expected EpicActions to be nil")
	}
	if display.FeatureActions != nil {
		t.Error("Expected FeatureActions to be nil")
	}
	if display.TaskActions == nil {
		t.Fatal("Expected TaskActions to be non-nil")
	}
	if display.Summary.TaskWithActions != 2 {
		t.Errorf("Expected TaskWithActions=2, got %d", display.Summary.TaskWithActions)
	}
	if display.Summary.TaskTotal != 3 {
		t.Errorf("Expected TaskTotal=3, got %d", display.Summary.TaskTotal)
	}
}

func TestShowActions_StatusFilter_MultiLevel(t *testing.T) {
	epicWf := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "researcher",
					InstructionTemplate: "Research {id}",
				},
			},
			"active": {Phase: "development"},
		},
	}

	taskWf := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					InstructionTemplate: "Implement {task_id}",
				},
			},
			"todo": {Phase: "planning"},
		},
	}

	// Filter by status "ready_for_development" - should match in both levels
	epicDisplay := buildActionsDisplay(epicWf, "ready_for_development", "")
	taskDisplay := buildActionsDisplay(taskWf, "ready_for_development", "")

	if len(epicDisplay.WorkflowActions) != 1 {
		t.Errorf("Expected 1 epic action for ready_for_development, got %d", len(epicDisplay.WorkflowActions))
	}
	if len(taskDisplay.WorkflowActions) != 1 {
		t.Errorf("Expected 1 task action for ready_for_development, got %d", len(taskDisplay.WorkflowActions))
	}
}

func TestShowActions_ActionTypeFilter_MultiLevel(t *testing.T) {
	epicWf := &config.WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]config.StatusMetadata{
			"completed": {
				Phase: "done",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionArchive,
					InstructionTemplate: "Archive {id}",
				},
			},
			"active": {
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "researcher",
					InstructionTemplate: "Research {id}",
				},
			},
		},
	}

	// Filter for spawn_agent only
	display := buildActionsDisplay(epicWf, "", config.ActionSpawnAgent)

	if len(display.WorkflowActions) != 1 {
		t.Errorf("Expected 1 spawn_agent action, got %d", len(display.WorkflowActions))
	}
	if display.WorkflowActions[0].Status != "active" {
		t.Errorf("Expected active status, got %s", display.WorkflowActions[0].Status)
	}
}
