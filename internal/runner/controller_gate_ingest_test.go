package runner

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TestResultContractFor_DefaultsToLegacy pins REQ-F-006's compatibility
// default: a nil stepInfo or a step whose ResultContract is unset/omitted
// resolves to "legacy".
func TestResultContractFor_DefaultsToLegacy(t *testing.T) {
	contract, err := resultContractFor(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contract != resultContractLegacy {
		t.Fatalf("expected nil stepInfo to resolve to %q, got %q", resultContractLegacy, contract)
	}

	contract, err = resultContractFor(&services.NextStatusInfo{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contract != resultContractLegacy {
		t.Fatalf("expected an empty ResultContract to resolve to %q, got %q", resultContractLegacy, contract)
	}
}

// TestResultContractFor_ResolvesGateResultV1 proves resultContractFor reads
// the workflow-resolved ResultContract field once populated (T-E34-F05-005).
func TestResultContractFor_ResolvesGateResultV1(t *testing.T) {
	contract, err := resultContractFor(&services.NextStatusInfo{ResultContract: resultContractGateResultV1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contract != resultContractGateResultV1 {
		t.Fatalf("expected %q, got %q", resultContractGateResultV1, contract)
	}
}

// TestResultContractFor_RejectsUnknownValue proves an unrecognized
// result_contract value fails closed rather than silently defaulting.
func TestResultContractFor_RejectsUnknownValue(t *testing.T) {
	_, err := resultContractFor(&services.NextStatusInfo{ResultContract: "some_future_contract"})
	if err == nil {
		t.Fatalf("expected an error for an unknown result_contract value")
	}
}

// TestIngestGateResultForDispatch_ValidEnvelopeTransitions exercises the
// controller-level wiring (GateIngestDeps → IngestGateResult →
// gatepersist.Coordinator) that handleSpawnAgent's gate_result_v1 branch
// calls. It is invoked directly (rather than through the full Run() loop)
// because resultContractFor is hard-stubbed to "legacy" until
// T-E34-F05-005 lands the config field it will read — see that function's
// doc comment.
func TestIngestGateResultForDispatch_ValidEnvelopeTransitions(t *testing.T) {
	transitioner := &fakeTransitioner{status: map[string]string{"E01-F01-001": "todo"}}
	coordinator := gatepersist.NewCoordinator(
		&fakeNoteWriter{},
		fakeNoteReader{},
		fakeHistoryReader{},
		fakeStatusValidator{valid: map[string]bool{"todo": true, "in_review": true}},
		transitioner,
		transitioner,
		&fakeLeaseReleaser{},
	)

	controller, err := NewRunController(RunControllerDeps{
		Transitioner: &oneShotStatusTransitioner{},
		ActionSvc:    &MockActionService{},
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  map[string]AgentDispatcher{"anthropic": &MockDispatcher{}},
		GateIngest: &GateIngestDeps{
			Coordinator:  coordinator,
			OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
		},
	})
	if err != nil {
		t.Fatalf("NewRunController: %v", err)
	}

	nextInfo := &services.NextStatusInfo{
		AvailableTransitions: []services.TransitionInfoWithAction{{TransitionInfo: workflow.TransitionInfo{TargetStatus: "in_review"}}},
		Outcomes:             map[string]string{"pass": "in_review"},
	}
	action := &config.PopulatedAction{Provider: "anthropic"}
	dispatchResult := &DispatchResult{
		ExitCode: 0,
		Stdout: `{"kind": "final", "recommended_outcome": "pass", "evidence": [],` +
			` "gate_result": {"schema_version": 1, "summary": "all checks passed"}}`,
		Duration: time.Millisecond,
	}
	opts := RunOptions{ProjectRoot: t.TempDir(), RunID: "run-abc123def456abc123def456abc123", EntityType: "task", SessionID: "sess-1"}
	disabled := false

	toStatus, err := controller.ingestGateResultForDispatch(
		context.Background(), "E01-F01-001", "todo", nextInfo, action, opts, dispatchResult, &disabled, 1,
	)
	if err != nil {
		t.Fatalf("expected successful gate ingestion, got error: %v", err)
	}
	if toStatus != "in_review" {
		t.Fatalf("expected transition to in_review, got %q", toStatus)
	}
}

// TestIngestGateResultForDispatch_MalformedEnvelopeFailsClosed asserts the
// gate_result_v1 path never falls through to the legacy recommendedOutcome
// parser on a malformed envelope — it must fail with no transition. Proven
// here by never calling targetStatusForDispatch/legacy transition at all in
// this code path; malformed stdout that would otherwise be a legacy
// pass-first fallback (an unrecognized string) still errors.
func TestIngestGateResultForDispatch_MalformedEnvelopeFailsClosed(t *testing.T) {
	transitioner := &fakeTransitioner{status: map[string]string{"E01-F01-001": "todo"}}
	coordinator := gatepersist.NewCoordinator(
		&fakeNoteWriter{}, fakeNoteReader{}, fakeHistoryReader{},
		fakeStatusValidator{valid: map[string]bool{"todo": true, "in_review": true}},
		transitioner, transitioner, &fakeLeaseReleaser{},
	)
	controller, err := NewRunController(RunControllerDeps{
		Transitioner: &oneShotStatusTransitioner{},
		ActionSvc:    &MockActionService{},
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  map[string]AgentDispatcher{"anthropic": &MockDispatcher{}},
		GateIngest: &GateIngestDeps{
			Coordinator:  coordinator,
			OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
		},
	})
	if err != nil {
		t.Fatalf("NewRunController: %v", err)
	}

	nextInfo := &services.NextStatusInfo{
		AvailableTransitions: []services.TransitionInfoWithAction{{TransitionInfo: workflow.TransitionInfo{TargetStatus: "in_review"}}},
		Outcomes:             map[string]string{"pass": "in_review"},
	}
	action := &config.PopulatedAction{Provider: "anthropic"}
	// Not a valid envelope at all — this is the exact shape the legacy
	// pass-first path would silently accept (no "recommended outcome:"
	// line, not bare outcome JSON) and fall through to
	// nextInfo.AvailableTransitions[0]. The gate path must reject it
	// instead.
	dispatchResult := &DispatchResult{ExitCode: 0, Stdout: "worker finished successfully", Duration: time.Millisecond}
	opts := RunOptions{ProjectRoot: t.TempDir(), RunID: "run-abc123def456abc123def456abc123", EntityType: "task", SessionID: "sess-1"}
	disabled := false

	toStatus, err := controller.ingestGateResultForDispatch(
		context.Background(), "E01-F01-001", "todo", nextInfo, action, opts, dispatchResult, &disabled, 1,
	)
	if err == nil {
		t.Fatalf("expected malformed envelope to fail closed, got toStatus=%q", toStatus)
	}
	if transitioner.status["E01-F01-001"] != "todo" {
		t.Fatalf("expected no transition to have occurred, entity is now %q", transitioner.status["E01-F01-001"])
	}
}
