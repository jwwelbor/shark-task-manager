package runner

// Covers T-E34-F01-005 (docs/plan/E34-prompt-and-skill-improvements/E34-F01-harness-aware-prompt-rendering/tasks/T-E34-F01-005.md):
// AC-T1 — RunController.Run() resolves harness identity via an optional
// HarnessResolver dependency and merges the three harness* keys into vars
// before GetStatusActionPopulated, injecting the zero identity's three empty
// keys (never an absent key, D-F01-07) when the resolver is nil.
//
// AC-T2 and the AC-08/TC-009-011 next/run parity assertions live in
// internal/cli/commands/next_run_harness_parity_test.go, the only place both
// runNext and RunController.Run are reachable from the same test binary.

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// mockHarnessClaimReader implements services.ClaimReader for controller-level
// harness resolver tests, mirroring harnessMockClaimReader in
// internal/cli/commands/next_harness_test.go (unexported there, so this
// package needs its own copy — same shape, per the Caller-Path Contract's
// "mock ClaimReader" seam, never HarnessResolver.Resolve itself).
type mockHarnessClaimReader struct {
	claim *models.EntityClaim
}

func (m *mockHarnessClaimReader) Get(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
	return m.claim, nil
}

var _ services.ClaimReader = (*mockHarnessClaimReader)(nil)

// terminalAfterFirstIterationTransitioner returns a live (non-terminal)
// status on the first GetNextStatus call and a terminal status thereafter,
// so a controller.Run() driven with a "pause" action naturally stops after
// exactly one Step-3/4 pass without needing dispatch/transition mocking.
type oneShotStatusTransitioner struct {
	status string
}

func (o *oneShotStatusTransitioner) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	return &services.NextStatusInfo{CurrentStatus: o.status}, nil
}

func (o *oneShotStatusTransitioner) TransitionStatus(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	return nil, errors.New("oneShotStatusTransitioner: TransitionStatus must not be called by a pause action")
}

// pauseActionService always returns a "pause" action and records the vars
// map it was called with, so tests can assert on the harness* keys merged
// into vars by the controller's harness resolution step (AC-T1).
type pauseActionService struct {
	capturedVars map[string]string
}

func (p *pauseActionService) GetStatusAction(ctx context.Context, status string) (*config.OrchestratorAction, error) {
	return nil, nil
}

func (p *pauseActionService) GetStatusActionPopulated(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
	p.capturedVars = vars
	return &config.PopulatedAction{Action: "pause"}, nil
}

func (p *pauseActionService) GetAllActions(ctx context.Context) (map[string]*config.OrchestratorAction, error) {
	return map[string]*config.OrchestratorAction{}, nil
}

func (p *pauseActionService) ValidateActions(ctx context.Context) (*config.ValidationResult, error) {
	return &config.ValidationResult{Valid: true}, nil
}

func (p *pauseActionService) Reload(ctx context.Context) error { return nil }

func (p *pauseActionService) ForEntity(entityType string) config.ActionService { return p }

var _ config.ActionService = (*pauseActionService)(nil)

// TestRunController_HarnessResolverNil_InjectsZeroIdentityVars covers AC-T1's
// "when nil, the controller injects the zero identity's three empty keys —
// never an absent key" clause (D-F01-07).
func TestRunController_HarnessResolverNil_InjectsZeroIdentityVars(t *testing.T) {
	actionSvc := &pauseActionService{}
	ctrl, err := NewRunController(RunControllerDeps{
		Transitioner: &oneShotStatusTransitioner{status: "in_progress"},
		ActionSvc:    actionSvc,
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  map[string]AgentDispatcher{"": &MockDispatcher{}},
		// HarnessResolver intentionally omitted (nil).
	})
	if err != nil {
		t.Fatalf("NewRunController() error = %v", err)
	}

	_, err = ctrl.Run(context.Background(), "E07-F01-001", RunOptions{EntityType: "task"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	for _, key := range []string{"harness", "harness_version", "harness_model"} {
		v, ok := actionSvc.capturedVars[key]
		if !ok {
			t.Fatalf("vars missing key %q with nil HarnessResolver; D-F01-07 requires the key to always be present", key)
		}
		if v != "" {
			t.Fatalf("vars[%q] = %q, want empty string with nil HarnessResolver", key, v)
		}
	}
}

// TestRunController_HarnessResolverResolvesClaimIntoVars covers AC-T1's
// resolver-merge clause: when a HarnessResolver is wired, the resolved
// identity (here sourced from a claim) is merged into vars before
// GetStatusActionPopulated runs.
func TestRunController_HarnessResolverResolvesClaimIntoVars(t *testing.T) {
	actionSvc := &pauseActionService{}
	claims := &mockHarnessClaimReader{claim: &models.EntityClaim{Harness: "claude", HarnessVersion: "2.1.0", HarnessModel: "opus"}}
	ctrl, err := NewRunController(RunControllerDeps{
		Transitioner:    &oneShotStatusTransitioner{status: "in_progress"},
		ActionSvc:       actionSvc,
		WorkflowSvc:     defaultWorkflowSvc(),
		Dispatchers:     map[string]AgentDispatcher{"": &MockDispatcher{}},
		HarnessResolver: services.NewHarnessResolver(claims),
	})
	if err != nil {
		t.Fatalf("NewRunController() error = %v", err)
	}

	_, err = ctrl.Run(context.Background(), "E07-F01-001", RunOptions{EntityType: "task"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := map[string]string{"harness": "claude", "harness_version": "2.1.0", "harness_model": "opus"}
	for key, wantVal := range want {
		if got := actionSvc.capturedVars[key]; got != wantVal {
			t.Errorf("vars[%q] = %q, want %q", key, got, wantVal)
		}
	}
}

// TestRunController_HarnessOverrideBeatsClaim covers AC-T1's flag-override
// entry point at the controller level: RunOptions.HarnessOverride outranks
// the claim per field, matching REQ-F-002/D-F01-04 precedence.
func TestRunController_HarnessOverrideBeatsClaim(t *testing.T) {
	actionSvc := &pauseActionService{}
	claims := &mockHarnessClaimReader{claim: &models.EntityClaim{Harness: "codex"}}
	ctrl, err := NewRunController(RunControllerDeps{
		Transitioner:    &oneShotStatusTransitioner{status: "in_progress"},
		ActionSvc:       actionSvc,
		WorkflowSvc:     defaultWorkflowSvc(),
		Dispatchers:     map[string]AgentDispatcher{"": &MockDispatcher{}},
		HarnessResolver: services.NewHarnessResolver(claims),
	})
	if err != nil {
		t.Fatalf("NewRunController() error = %v", err)
	}

	_, err = ctrl.Run(context.Background(), "E07-F01-001", RunOptions{
		EntityType:      "task",
		HarnessOverride: services.HarnessIdentity{Type: "claude"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := actionSvc.capturedVars["harness"]; got != "claude" {
		t.Errorf("vars[\"harness\"] = %q, want %q (flag must beat claim)", got, "claude")
	}
}
