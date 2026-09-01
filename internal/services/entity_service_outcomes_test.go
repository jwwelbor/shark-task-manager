package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cfgworkflow "github.com/jwwelbor/shark-task-manager/internal/config/workflow"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// newRouteBasedTaskEntityService builds an EntityService backed by a route-based
// (steps:) task workflow loaded from a temp project root.
func newRouteBasedTaskEntityService(t *testing.T) *EntityService {
	t.Helper()

	projectRoot := t.TempDir()
	workflowDir := filepath.Join(projectRoot, "shark-data", "workflow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workflow dir: %v", err)
	}

	taskYAML := `version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    is_planning: true
    action: advance_status
    outcomes:
      pass: development
      fail: draft
      blocked: blocked
  development:
    phase: development
    action: spawn_agent
    agent: developer
    result_contract: gate_result_v1
    outcomes:
      pass: completed
      fail: draft
      blocked: blocked
    outcome_roles:
      pass: success
      fail: route_rework
      blocked: blocked
  blocked:
    phase: blocked
    parking: true
  completed:
    phase: done
    terminal: true
`
	if err := os.WriteFile(filepath.Join(workflowDir, "task.yaml"), []byte(taskYAML), 0o644); err != nil {
		t.Fatalf("write task.yaml: %v", err)
	}
	configJSON := `{"workflow_config": "shark-data/workflow"}`
	if err := os.WriteFile(filepath.Join(projectRoot, ".sharkconfig.json"), []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfgworkflow.ClearWorkflowCache()
	t.Cleanup(cfgworkflow.ClearWorkflowCache)

	wfSvc := workflow.NewService(projectRoot)
	return NewEntityService(wfSvc).ForLevel(workflow.LevelTask)
}

// TestEntityService_NextStatus_OutcomesPopulated asserts that NextStatusInfo
// carries the route-based outcome map for a workable step and leaves it empty
// for terminal steps (B3-#8).
func TestEntityService_NextStatus_OutcomesPopulated(t *testing.T) {
	svc := newRouteBasedTaskEntityService(t)

	t.Run("workable step has outcomes", func(t *testing.T) {
		task := &models.Task{Status: "development"}
		info := svc.GetNextStatusForEntity(models.EntityTypeTask, "E01-F01-001", task, nil)
		if len(info.Outcomes) == 0 {
			t.Fatalf("expected non-empty Outcomes for workable step, got %v", info.Outcomes)
		}
		if info.Outcomes["pass"] != "completed" || info.Outcomes["blocked"] != "blocked" {
			t.Errorf("unexpected outcomes: %v", info.Outcomes)
		}
	})

	t.Run("terminal step has no outcomes", func(t *testing.T) {
		task := &models.Task{Status: "completed"}
		info := svc.GetNextStatusForEntity(models.EntityTypeTask, "E01-F01-001", task, nil)
		if len(info.Outcomes) != 0 {
			t.Errorf("expected empty Outcomes for terminal step, got %v", info.Outcomes)
		}
		if !info.IsTerminal {
			t.Error("expected IsTerminal=true for completed step")
		}
	})
}

// TestEntityService_NextStatus_ResultContractAndOutcomeRoles asserts
// REQ-F-006: a step configured with result_contract: gate_result_v1
// resolves both ResultContract and a complete OutcomeRoles map on
// NextStatusInfo, a step with no result_contract resolves "legacy" with a
// nil role map, and both GetNextStatus and GetNextStatusForEntity (the two
// construction sites `shark next` and the core runner ultimately read from)
// agree.
func TestEntityService_NextStatus_ResultContractAndOutcomeRoles(t *testing.T) {
	svc := newRouteBasedTaskEntityService(t)

	t.Run("gate_result_v1 step resolves contract and roles", func(t *testing.T) {
		task := &models.Task{Status: "development"}
		info := svc.GetNextStatusForEntity(models.EntityTypeTask, "E01-F01-001", task, nil)
		if info.ResultContract != "gate_result_v1" {
			t.Errorf("expected result_contract=gate_result_v1, got %q", info.ResultContract)
		}
		if len(info.OutcomeRoles) != 3 {
			t.Fatalf("expected 3 outcome roles, got %#v", info.OutcomeRoles)
		}
		if info.OutcomeRoles["pass"] != "success" || info.OutcomeRoles["fail"] != "route_rework" || info.OutcomeRoles["blocked"] != "blocked" {
			t.Errorf("unexpected outcome roles: %#v", info.OutcomeRoles)
		}
	})

	t.Run("legacy step resolves legacy with no roles", func(t *testing.T) {
		task := &models.Task{Status: "draft"}
		info := svc.GetNextStatusForEntity(models.EntityTypeTask, "E01-F01-001", task, nil)
		if info.ResultContract != "legacy" {
			t.Errorf("expected result_contract=legacy for an unconfigured step, got %q", info.ResultContract)
		}
		if len(info.OutcomeRoles) != 0 {
			t.Errorf("expected no outcome roles for a legacy step, got %#v", info.OutcomeRoles)
		}
	})

	t.Run("GetNextStatus (DB-backed path) agrees with GetNextStatusForEntity", func(t *testing.T) {
		info, err := svc.GetNextStatus(context.Background(), &stubEntityRepo{status: "development"}, models.EntityTypeTask, "E01-F01-001", nil)
		if err != nil {
			t.Fatalf("GetNextStatus: %v", err)
		}
		if info.ResultContract != "gate_result_v1" {
			t.Errorf("expected result_contract=gate_result_v1 via GetNextStatus, got %q", info.ResultContract)
		}
		if len(info.OutcomeRoles) != 3 {
			t.Errorf("expected 3 outcome roles via GetNextStatus, got %#v", info.OutcomeRoles)
		}
	})
}

// stubEntityRepo is a minimal EntityRepository stub (embedding the
// package's existing noopEntityRepo for the methods this test doesn't
// exercise) so TestEntityService_NextStatus_ResultContractAndOutcomeRoles
// can exercise GetNextStatus (the DB-backed path `shark next`/the core
// runner actually call) without a real database.
type stubEntityRepo struct {
	noopEntityRepo
	status string
}

func (r *stubEntityRepo) GetByKey(_ context.Context, _ string) (models.Entity, error) {
	return &models.Task{Status: models.TaskStatus(r.status)}, nil
}

// TestEntityService_NextStatus_LegacyHasNoOutcomes asserts a legacy
// (status_flow) workflow yields an empty Outcomes map.
func TestEntityService_NextStatus_LegacyHasNoOutcomes(t *testing.T) {
	projectRoot := t.TempDir()
	configJSON := `{
		"task_workflow": {
			"status_flow_version": "1.0",
			"status_flow": {"todo": ["in_progress"], "in_progress": ["done"], "done": []},
			"status_metadata": {
				"todo": {"phase": "planning"},
				"in_progress": {"phase": "development"},
				"done": {"phase": "done"}
			},
			"special_statuses": {"_start_": ["todo"], "_complete_": ["done"]}
		}
	}`
	if err := os.WriteFile(filepath.Join(projectRoot, ".sharkconfig.json"), []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfgworkflow.ClearWorkflowCache()
	t.Cleanup(cfgworkflow.ClearWorkflowCache)

	svc := NewEntityService(workflow.NewService(projectRoot)).ForLevel(workflow.LevelTask)
	task := &models.Task{Status: "in_progress"}
	info := svc.GetNextStatusForEntity(models.EntityTypeTask, "E01-F01-001", task, nil)
	if len(info.Outcomes) != 0 {
		t.Errorf("expected no outcomes for legacy workflow, got %v", info.Outcomes)
	}
}
