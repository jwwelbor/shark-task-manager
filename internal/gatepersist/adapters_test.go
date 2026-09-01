package gatepersist

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

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

func TestEntityServiceTransitioner_TransitionAndIdempotency(t *testing.T) {
	svc := testWorkflowService(t)
	entitySvc := services.NewEntityService(svc)
	registry := services.NewEntityRegistry()

	repo := &fakeTaskRepo{task: &models.Task{
		BaseEntity: models.BaseEntity{ID: 1, Key: "T-E01-F01-001", Title: "t"},
		Status:     models.TaskStatus("todo"),
	}}
	registry.Register(models.EntityTypeTask, repo)

	transitioner := NewEntityServiceTransitioner(entitySvc, registry)

	from, transitioned, err := transitioner.Transition(context.Background(), models.EntityTypeTask, "T-E01-F01-001", "in_review", "moving to review", "agent-1")
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
	from2, transitioned2, err := transitioner.Transition(context.Background(), models.EntityTypeTask, "T-E01-F01-001", "in_review", "moving to review again", "agent-1")
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
