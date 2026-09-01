// TestRunController_Run_GateResultV1StepRoutesThroughGateIngest and its
// legacy sibling below are the T-E34-F05-004 REQ-F-005 discriminating
// evidence controller_gate_ingest_test.go's own doc comment flagged as
// missing: they drive the FULL controller.Run() loop (dispatch →
// resultContractFor → branch selection → gate ingest/legacy outcome
// parsing → transition), rather than calling
// ingestGateResultForDispatch/resultContractFor directly. A dispatch-level
// bug in the branch selection itself (e.g. reading the wrong stepInfo, or
// never reaching the gate branch at all) would pass the existing
// unit-level tests but fail these.
package runner

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// e2eGateTransitioner is a controller-facing EntityTransitioner AND a
// gatepersist Transitioner/StatusReader over the same shared status map, so
// a transition applied by the gate coordinator (inside
// ingestGateResultForDispatch) is visible to the controller's own next
// GetNextStatus call in Run()'s loop — proving the two paths are actually
// wired together, not just independently mockable.
type e2eGateTransitioner struct {
	status         map[string]string
	resultContract string
	outcomeRoles   map[string]gateresult.OutcomeRole
	outcomes       map[string]string
	transitionErr  error // set to prove the legacy TransitionStatus path is never called for a gate step
}

func (t *e2eGateTransitioner) GetNextStatus(_ context.Context, key string) (*services.NextStatusInfo, error) {
	status := t.status[key]
	info := &services.NextStatusInfo{
		CurrentStatus:  status,
		IsTerminal:     status == "in_review",
		ResultContract: t.resultContract,
		OutcomeRoles:   t.outcomeRoles,
		Outcomes:       t.outcomes,
	}
	if status == "todo" {
		// Non-empty AvailableTransitions is required for handleSpawnAgent's
		// post-dispatch read to reach the result_contract branch at all — an
		// empty list short-circuits into the "no transitions available"
		// implicit-status path before ever resolving resultContractFor.
		info.AvailableTransitions = []services.TransitionInfoWithAction{
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "in_review"}},
		}
	}
	return info, nil
}

func (t *e2eGateTransitioner) TransitionStatus(_ context.Context, _, _ string, _ services.TransitionOptions) (*services.TransitionResult, error) {
	return nil, t.transitionErr
}

// gatepersist.Transitioner
func (t *e2eGateTransitioner) Transition(_ context.Context, _ models.EntityType, entityKey, targetStatus, _, _ string, _ gatepersist.TransitionGuard) (string, bool, error) {
	from := t.status[entityKey]
	t.status[entityKey] = targetStatus
	return from, from != targetStatus, nil
}

// gatepersist.StatusReader
func (t *e2eGateTransitioner) CurrentStatus(_ context.Context, _ models.EntityType, entityKey string) (string, error) {
	return t.status[entityKey], nil
}

// TestRunController_Run_GateResultV1StepRoutesThroughGateIngest drives a
// full controller.Run() loop for a step whose resolved result_contract is
// gate_result_v1: the dispatcher returns a worker-control envelope on
// stdout, and the assertion is that the entity actually transitions to the
// envelope's configured target status via the gate coordinator, with the
// legacy TransitionStatus path never invoked.
func TestRunController_Run_GateResultV1StepRoutesThroughGateIngest(t *testing.T) {
	key := "E01-F01-001"
	shared := &e2eGateTransitioner{
		status:         map[string]string{key: "todo"},
		resultContract: "gate_result_v1",
		outcomeRoles:   map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
		outcomes:       map[string]string{"pass": "in_review"},
		transitionErr:  errAssertLegacyTransitionNotCalled,
	}
	coordinator := gatepersist.NewCoordinator(
		&fakeNoteWriter{}, fakeNoteReader{}, fakeHistoryReader{},
		fakeStatusValidator{valid: map[string]bool{"todo": true, "in_review": true}},
		shared, shared, &fakeLeaseReleaser{},
	)

	envelope := `{"kind": "final", "recommended_outcome": "pass", "evidence": [],` +
		` "gate_result": {"schema_version": 1, "summary": "all checks passed"}}`

	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, AgentType: "developer", Provider: "anthropic", Instruction: "do work"}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{DispatchFunc: func(_ context.Context, _ DispatchInput) (*DispatchResult, error) {
			return &DispatchResult{ExitCode: 0, Stdout: envelope}, nil
		}},
	}

	ctrl, err := NewRunController(RunControllerDeps{
		Transitioner: shared,
		ActionSvc:    actionSvc,
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  dispatchers,
		GateIngest:   &GateIngestDeps{Coordinator: coordinator},
	})
	if err != nil {
		t.Fatalf("NewRunController: %v", err)
	}

	result, err := ctrl.Run(context.Background(), key, RunOptions{ProjectRoot: t.TempDir(), RunID: "run-e2e-gate-result-v1", SessionID: "sess-1", EntityType: "task"})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "completed" {
		t.Fatalf("expected Outcome=completed, got %s (error=%s)", result.Outcome, result.Error)
	}
	if result.FinalStatus != "in_review" {
		t.Fatalf("expected FinalStatus=in_review (via gate coordinator), got %q", result.FinalStatus)
	}
	if shared.status[key] != "in_review" {
		t.Fatalf("expected shared status to be updated by the gate coordinator, got %q", shared.status[key])
	}
	if len(result.Stages) == 0 || result.Stages[0].GateStatus == nil {
		t.Fatal("expected the gate stage to carry a GateStatus operator projection (T-E34-F05-004 REQ-F-005 item 5)")
	}
	if result.Stages[0].GateStatus.PersistenceState != "transition_applied" {
		t.Fatalf("expected GateStatus.PersistenceState=transition_applied, got %q", result.Stages[0].GateStatus.PersistenceState)
	}
}

// TestRunController_Run_LegacyStepStillRoutesThroughRecommendedOutcome pins
// the companion acceptance criterion: a non-gate (legacy/omitted
// result_contract) step must continue to transition via the existing
// recommendedOutcome/TransitionStatus path, unaffected by the gate_result_v1
// branch this task added.
func TestRunController_Run_LegacyStepStillRoutesThroughRecommendedOutcome(t *testing.T) {
	key := "E01-F01-002"
	transitioned := false
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(_ context.Context, k string) (*services.NextStatusInfo, error) {
			if transitioned {
				return &services.NextStatusInfo{CurrentStatus: "in_review", IsTerminal: true}, nil
			}
			return &services.NextStatusInfo{
				CurrentStatus: "todo",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "in_review"}},
				},
				Outcomes: map[string]string{"pass": "in_review"},
			}, nil
		},
		TransitionStatusFunc: func(_ context.Context, _, target string, _ services.TransitionOptions) (*services.TransitionResult, error) {
			transitioned = true
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, _ string, _ map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, AgentType: "developer", Provider: "anthropic", Instruction: "do work"}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{DispatchFunc: func(_ context.Context, _ DispatchInput) (*DispatchResult, error) {
			return &DispatchResult{ExitCode: 0, Stdout: "recommended outcome: pass"}, nil
		}},
	}

	ctrl := makeController(t, transitioner, actionSvc, dispatchers)
	result, err := ctrl.Run(context.Background(), key, RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if !transitioned {
		t.Fatal("expected the legacy TransitionStatus path to be called for a step with no result_contract")
	}
	if result.FinalStatus != "in_review" {
		t.Fatalf("expected FinalStatus=in_review, got %q", result.FinalStatus)
	}
}

// multiStageE2EGateTransitioner is a controller-facing EntityTransitioner
// AND a gatepersist Transitioner/StatusReader driving a two-gate-stage chain
// (code_review -> qa -> completed), both steps resolving result_contract
// gate_result_v1. Unlike e2eGateTransitioner (a single-stage chain), this
// lets a full Run() loop exercise two consecutive gate_result_v1 dispatches
// for the same entity within one invocation — the code-review round-7
// Finding 1 scenario (gaterun.CreateResult's create-once contract colliding
// on a reused run_id across stages).
type multiStageE2EGateTransitioner struct {
	status map[string]string
}

func (t *multiStageE2EGateTransitioner) GetNextStatus(_ context.Context, key string) (*services.NextStatusInfo, error) {
	status := t.status[key]
	info := &services.NextStatusInfo{
		CurrentStatus:  status,
		IsTerminal:     status == "completed",
		ResultContract: "gate_result_v1",
		OutcomeRoles:   map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
	}
	switch status {
	case "code_review":
		info.AvailableTransitions = []services.TransitionInfoWithAction{
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "qa"}},
		}
		info.Outcomes = map[string]string{"pass": "qa"}
	case "qa":
		info.AvailableTransitions = []services.TransitionInfoWithAction{
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
		}
		info.Outcomes = map[string]string{"pass": "completed"}
	}
	return info, nil
}

func (t *multiStageE2EGateTransitioner) TransitionStatus(_ context.Context, _, _ string, _ services.TransitionOptions) (*services.TransitionResult, error) {
	return nil, errAssertLegacyTransitionNotCalled
}

// gatepersist.Transitioner
func (t *multiStageE2EGateTransitioner) Transition(_ context.Context, _ models.EntityType, entityKey, targetStatus, _, _ string, _ gatepersist.TransitionGuard) (string, bool, error) {
	from := t.status[entityKey]
	t.status[entityKey] = targetStatus
	return from, from != targetStatus, nil
}

// gatepersist.StatusReader
func (t *multiStageE2EGateTransitioner) CurrentStatus(_ context.Context, _ models.EntityType, entityKey string) (string, error) {
	return t.status[entityKey], nil
}

// TestRunController_Run_MultiStageGateResultV1DispatchDoesNotReuseRunID is
// the full-loop (controller.Run()) counterpart of
// TestIngestGateResultForDispatch_MultiStageDispatchDoesNotReuseRunID: it
// drives TWO consecutive gate_result_v1 stages (code_review -> qa ->
// completed) for the same entity through one Run() invocation with a SINGLE
// opts.RunID, proving the branch selection and its per-stage RunID scoping
// are actually wired together end-to-end, not just correct at the unit level.
func TestRunController_Run_MultiStageGateResultV1DispatchDoesNotReuseRunID(t *testing.T) {
	key := "E01-F01-001"
	shared := &multiStageE2EGateTransitioner{status: map[string]string{key: "code_review"}}
	coordinator := gatepersist.NewCoordinator(
		&fakeNoteWriter{}, fakeNoteReader{}, fakeHistoryReader{},
		fakeStatusValidator{valid: map[string]bool{"code_review": true, "qa": true, "completed": true}},
		shared, shared, &fakeLeaseReleaser{},
	)

	callCount := 0
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{DispatchFunc: func(_ context.Context, _ DispatchInput) (*DispatchResult, error) {
			callCount++
			envelope := fmt.Sprintf(
				`{"kind": "final", "recommended_outcome": "pass", "evidence": [],`+
					` "gate_result": {"schema_version": 1, "summary": "stage %d passed"}}`,
				callCount,
			)
			return &DispatchResult{ExitCode: 0, Stdout: envelope}, nil
		}},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, _ string, _ map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, AgentType: "developer", Provider: "anthropic", Instruction: "do work"}, nil
		},
	}

	ctrl, err := NewRunController(RunControllerDeps{
		Transitioner: shared,
		ActionSvc:    actionSvc,
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  dispatchers,
		GateIngest:   &GateIngestDeps{Coordinator: coordinator},
	})
	if err != nil {
		t.Fatalf("NewRunController: %v", err)
	}

	result, err := ctrl.Run(context.Background(), key, RunOptions{ProjectRoot: t.TempDir(), RunID: "run-multi-stage-e2e", SessionID: "sess-1", EntityType: "task"})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.Outcome != "completed" {
		t.Fatalf("expected Outcome=completed, got %s (error=%s)", result.Outcome, result.Error)
	}
	if shared.status[key] != "completed" {
		t.Fatalf("expected both gate stages to apply, got final status %q", shared.status[key])
	}
	if callCount != 2 {
		t.Fatalf("expected exactly 2 agent dispatches (code_review, qa), got %d", callCount)
	}
	if len(result.Stages) != 2 {
		t.Fatalf("expected 2 recorded stages, got %d", len(result.Stages))
	}
}

var errAssertLegacyTransitionNotCalled = &legacyTransitionCalledError{}

type legacyTransitionCalledError struct{}

func (e *legacyTransitionCalledError) Error() string {
	return "legacy TransitionStatus was called for a gate_result_v1 step; expected the gate coordinator to own the transition"
}
