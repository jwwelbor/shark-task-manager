package gatepersist

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// fakeAdvanceGuardRecorder is a minimal in-memory services.AdvanceGuardRecorder,
// enough to prove EntityServiceTransitioner actually wires a
// TransitionGuard through to services.EntityService's replay ledger.
type fakeAdvanceGuardRecorder struct {
	consumed map[string]bool
}

func newFakeAdvanceGuardRecorder() *fakeAdvanceGuardRecorder {
	return &fakeAdvanceGuardRecorder{consumed: make(map[string]bool)}
}

func (r *fakeAdvanceGuardRecorder) key(entityType string, entityID int64, sessionID, fromStatus, outcome string) string {
	return entityType + "|" + sessionID + "|" + fromStatus + "|" + outcome
}

func (r *fakeAdvanceGuardRecorder) WasConsumed(_ context.Context, entityType string, entityID int64, sessionID, fromStatus, outcome string) (bool, error) {
	return r.consumed[r.key(entityType, entityID, sessionID, fromStatus, outcome)], nil
}

func (r *fakeAdvanceGuardRecorder) RecordConsumed(_ context.Context, entityType string, entityID int64, sessionID, fromStatus, outcome string) error {
	r.consumed[r.key(entityType, entityID, sessionID, fromStatus, outcome)] = true
	return nil
}

func (r *fakeAdvanceGuardRecorder) DeleteConsumed(_ context.Context, entityType string, entityID int64, sessionID, fromStatus, outcome string) error {
	delete(r.consumed, r.key(entityType, entityID, sessionID, fromStatus, outcome))
	return nil
}

// fakeTaskRepo is a minimal services.EntityRepository backed by one
// in-memory *models.Task, enough to exercise EntityServiceTransitioner's
// wiring (repository lookup -> ForLevel -> TransitionStatus) without a real
// database, matching this project's "service tests use mocks" convention.
type fakeTaskRepo struct {
	task *models.Task
}

func (r *fakeTaskRepo) GetByKey(_ context.Context, key string) (models.Entity, error) {
	if key != r.task.Key {
		return nil, os.ErrNotExist
	}
	return r.task, nil
}
func (r *fakeTaskRepo) GetByID(_ context.Context, id int64) (models.Entity, error) {
	if id != r.task.ID {
		return nil, os.ErrNotExist
	}
	return r.task, nil
}
func (r *fakeTaskRepo) UpdateStatus(_ context.Context, _ int64, status string) error {
	r.task.SetStatus(status)
	return nil
}
func (r *fakeTaskRepo) UpdateStatusIfCurrent(_ context.Context, _ int64, expected, status string) (bool, error) {
	if r.task.GetStatus() != expected {
		return false, nil
	}
	r.task.SetStatus(status)
	return true, nil
}
func (r *fakeTaskRepo) Update(_ context.Context, _ models.Entity) error { return nil }
func (r *fakeTaskRepo) GetContextData(_ context.Context, _ int64) (*string, error) {
	return nil, nil
}
func (r *fakeTaskRepo) UpdateContextData(_ context.Context, _ int64, _ *string) error { return nil }

func testWorkflowService(t *testing.T) *workflow.Service {
	t.Helper()
	config.ClearWorkflowCache()
	t.Cleanup(config.ClearWorkflowCache)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	configJSON := `{
		"task_workflow": {
			"status_flow_version": "1.0",
			"special_statuses": {"_start_": ["todo"], "_complete_": ["completed"]},
			"status_flow": {
				"todo": ["in_review"],
				"in_review": ["ready_for_qa", "todo"],
				"ready_for_qa": ["completed"],
				"completed": []
			},
			"status_metadata": {
				"todo": {"phase": "planning"},
				"in_review": {"phase": "review"},
				"ready_for_qa": {"phase": "qa"},
				"completed": {"phase": "done"}
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return workflow.NewService(tmpDir)
}

func TestWorkflowStatusValidator_IsValidStatus(t *testing.T) {
	svc := testWorkflowService(t)
	v := NewWorkflowStatusValidator(svc)

	if !v.IsValidStatus(models.EntityTypeTask, "ready_for_qa") {
		t.Errorf("expected ready_for_qa to be a valid task status")
	}
	if v.IsValidStatus(models.EntityTypeTask, "not_a_real_status") {
		t.Errorf("expected an unconfigured status to be invalid")
	}
}

// testRouteBasedWorkflowServiceWithAlias builds a route-based (steps:) task
// workflow whose "qa" step (result_contract: gate_result_v1, matching the
// shipped change.qa/change.code_review shape) carries a pre-migration
// status alias — route-based-workflow.md §5's "resolve on read" case an
// entity can be parked under.
func testRouteBasedWorkflowServiceWithAlias(t *testing.T) *workflow.Service {
	t.Helper()
	config.ClearWorkflowCache()
	t.Cleanup(config.ClearWorkflowCache)

	tmpDir := t.TempDir()
	workflowDir := filepath.Join(tmpDir, "workflow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workflow dir: %v", err)
	}
	taskYAML := `version: "1.0"
start: todo
steps:
  todo:
    phase: planning
    action: advance_status
    outcomes: { pass: in_review }
  in_review:
    phase: development
    action: advance_status
    outcomes: { pass: qa }
  qa:
    phase: qa
    action: spawn_agent
    agent: qa
    prompt: task/qa.md
    result_contract: gate_result_v1
    aliases: [ready_for_qa]
    outcomes: { pass: completed, fail: in_review, blocked: todo }
  completed:
    phase: done
    terminal: true
`
	if err := os.WriteFile(filepath.Join(workflowDir, "task.yaml"), []byte(taskYAML), 0o644); err != nil {
		t.Fatalf("write task.yaml: %v", err)
	}
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(`{"workflow_config": "workflow"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return workflow.NewService(tmpDir)
}

func TestEntityServiceTransitioner_TransitionAndIdempotency(t *testing.T) {
	svc := testWorkflowService(t)
	entitySvc := services.NewEntityService(svc)
	registry := services.NewEntityRegistry()

	repo := &fakeTaskRepo{task: &models.Task{
		BaseEntity: models.BaseEntity{ID: 1, Key: "T-E01-F01-001", Title: "t"},
		Status:     models.TaskStatus("todo"),
	}}
	registry.Register(models.EntityTypeTask, repo)

	transitioner := NewEntityServiceTransitioner(entitySvc, registry, svc)

	from, transitioned, err := transitioner.Transition(context.Background(), models.EntityTypeTask, "T-E01-F01-001", "in_review", "moving to review", "agent-1", TransitionGuard{})
	if err != nil {
		t.Fatalf("Transition: %v", err)
	}
	if !transitioned || from != "todo" {
		t.Fatalf("expected transitioned=true from=todo, got transitioned=%v from=%q", transitioned, from)
	}
	if repo.task.GetStatus() != "in_review" {
		t.Fatalf("expected task status in_review, got %q", repo.task.GetStatus())
	}

	// Idempotent re-call at the same target must no-op rather than error or
	// re-write, matching the Transitioner interface's documented contract
	// that Coordinator.Persist relies on for its "verify already-applied
	// identical target" resume behavior.
	from2, transitioned2, err := transitioner.Transition(context.Background(), models.EntityTypeTask, "T-E01-F01-001", "in_review", "moving to review again", "agent-1", TransitionGuard{})
	if err != nil {
		t.Fatalf("idempotent Transition: %v", err)
	}
	if transitioned2 {
		t.Fatalf("expected the idempotent re-call to report transitioned=false")
	}
	if from2 != "in_review" {
		t.Fatalf("expected from=in_review on the idempotent re-call, got %q", from2)
	}
}

// TestEntityServiceTransitioner_CurrentStatusResolvesAlias is F-3's alias
// hardening (found by advisor review during T-E34-F05-004 rework):
// gatepersist.Coordinator's pre-transition and already-transitioned
// verification branches both compare CurrentStatus's return value against
// SourceStatus/TargetStatus, which callers populate from
// NextStatusInfo.CurrentStatus — always alias-resolved (canonical) per
// EntityService.GetNextStatus / route-based-workflow.md §5 "resolve on
// read". Before this fix, CurrentStatus returned the entity's raw stored
// status; an entity still parked under a pre-migration alias (e.g.
// "ready_for_qa" for the "qa" step) would compare its raw alias against the
// canonical name and fail closed on an otherwise-healthy entity on every
// foreground gate run.
func TestEntityServiceTransitioner_CurrentStatusResolvesAlias(t *testing.T) {
	svc := testRouteBasedWorkflowServiceWithAlias(t)
	entitySvc := services.NewEntityService(svc)
	registry := services.NewEntityRegistry()

	repo := &fakeTaskRepo{task: &models.Task{
		BaseEntity: models.BaseEntity{ID: 1, Key: "T-E01-F01-001", Title: "t"},
		// Stored under the pre-migration alias, not the canonical step name.
		Status: models.TaskStatus("ready_for_qa"),
	}}
	registry.Register(models.EntityTypeTask, repo)

	transitioner := NewEntityServiceTransitioner(entitySvc, registry, svc)

	got, err := transitioner.CurrentStatus(context.Background(), models.EntityTypeTask, "T-E01-F01-001")
	if err != nil {
		t.Fatalf("CurrentStatus: %v", err)
	}
	if got != "qa" {
		t.Fatalf("CurrentStatus = %q, want canonical %q (alias-resolved from stored %q)", got, "qa", "ready_for_qa")
	}
}

// TestEntityServiceTransitioner_GuardedAdvance_RejectsSessionFromStatusMismatch
// is this rework round's regression test: code-review round 4 found that
// EntityServiceTransitioner.Transition dropped guard.SessionID/FromStatus/
// Outcome/GuardAdvance on the floor, so advance_guard.enabled never actually
// engaged replay protection for any gatepersist-driven transition, even
// though the config was on. Before the fix in adapters.go, this test failed
// with transitioned=true (the guard silently no-op'd); it must now fail
// closed with services.ErrAdvanceGuardStaleFromStatus, mirroring
// controller.go's TestGuardedTransitionOptions_BindsRunLeaseAndSourceStatus
// intent (that test only checks the pure options builder — this one proves
// the wiring reaches services.EntityService's actual enforcement).
func TestEntityServiceTransitioner_GuardedAdvance_RejectsSessionFromStatusMismatch(t *testing.T) {
	svc := testWorkflowService(t)
	entitySvc := services.NewEntityService(svc)
	entitySvc.SetAdvanceGuard(config.AdvanceGuardConfig{Enabled: true}, newFakeAdvanceGuardRecorder())
	registry := services.NewEntityRegistry()

	repo := &fakeTaskRepo{task: &models.Task{
		BaseEntity: models.BaseEntity{ID: 1, Key: "T-E01-F01-001", Title: "t"},
		Status:     models.TaskStatus("todo"),
	}}
	registry.Register(models.EntityTypeTask, repo)

	transitioner := NewEntityServiceTransitioner(entitySvc, registry, svc)

	// The task is actually at "todo", but the guard claims it observed
	// "in_review" as the pre-transition status — exactly the stale/replayed
	// session scenario advance_guard exists to reject.
	guard := TransitionGuard{SessionID: "sess-1", FromStatus: "in_review", Outcome: "pass"}
	_, transitioned, err := transitioner.Transition(context.Background(), models.EntityTypeTask, "T-E01-F01-001", "in_review", "moving to review", "agent-1", guard)
	if !errors.Is(err, services.ErrAdvanceGuardStaleFromStatus) {
		t.Fatalf("expected ErrAdvanceGuardStaleFromStatus, got transitioned=%v err=%v", transitioned, err)
	}
	if repo.task.GetStatus() != "todo" {
		t.Fatalf("expected the rejected guarded advance to leave status untouched, got %q", repo.task.GetStatus())
	}
}

// TestEntityServiceTransitioner_GuardedAdvance_SucceedsAndRejectsReplay proves
// the positive path: a correctly-wired guard (matching session/from-status/
// outcome) is accepted once, and services.EntityService's replay ledger then
// rejects an identical guarded re-call — the actual protection
// advance_guard.enabled is supposed to buy gatepersist's parent-owned
// transitions.
func TestEntityServiceTransitioner_GuardedAdvance_SucceedsAndRejectsReplay(t *testing.T) {
	svc := testWorkflowService(t)
	entitySvc := services.NewEntityService(svc)
	entitySvc.SetAdvanceGuard(config.AdvanceGuardConfig{Enabled: true}, newFakeAdvanceGuardRecorder())
	registry := services.NewEntityRegistry()

	repo := &fakeTaskRepo{task: &models.Task{
		BaseEntity: models.BaseEntity{ID: 1, Key: "T-E01-F01-001", Title: "t"},
		Status:     models.TaskStatus("todo"),
	}}
	registry.Register(models.EntityTypeTask, repo)

	transitioner := NewEntityServiceTransitioner(entitySvc, registry, svc)

	guard := TransitionGuard{SessionID: "sess-1", FromStatus: "todo", Outcome: "pass"}
	from, transitioned, err := transitioner.Transition(context.Background(), models.EntityTypeTask, "T-E01-F01-001", "in_review", "moving to review", "agent-1", guard)
	if err != nil {
		t.Fatalf("Transition: %v", err)
	}
	if !transitioned || from != "todo" {
		t.Fatalf("expected transitioned=true from=todo, got transitioned=%v from=%q", transitioned, from)
	}

	// Same guard tuple replayed against a fresh task back at "todo" (a
	// concurrent/duplicate worker submission) must be rejected by the ledger
	// rather than silently re-applied.
	repo.task.SetStatus("todo")
	_, _, err = transitioner.Transition(context.Background(), models.EntityTypeTask, "T-E01-F01-001", "in_review", "moving to review", "agent-1", guard)
	if !errors.Is(err, services.ErrAdvanceGuardRepeatRejected) {
		t.Fatalf("expected ErrAdvanceGuardRepeatRejected on replay, got %v", err)
	}
}
