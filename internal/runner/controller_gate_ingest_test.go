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
// calls. It is invoked directly against ingestGateResultForDispatch (a
// focused unit test of that one method's contract); for a test that drives
// the FULL controller.Run() dispatch loop end-to-end — proving the branch
// selection in handleSpawnAgent actually reaches this method for a
// gate_result_v1 step, and that a legacy step still doesn't — see
// controller_gate_e2e_test.go.
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

	toStatus, _, err := controller.ingestGateResultForDispatch(
		context.Background(), "E01-F01-001", "todo", nextInfo, action, opts, dispatchResult, &disabled, 1,
	)
	if err != nil {
		t.Fatalf("expected successful gate ingestion, got error: %v", err)
	}
	if toStatus != "in_review" {
		t.Fatalf("expected transition to in_review, got %q", toStatus)
	}
}

// TestIngestGateResultForDispatch_NonTerminalTargetDoesNotReleaseLease is the
// T-E34-F05-004 rework's regression guard for UAT CRITICAL finding #2 (the
// cross-stage lease-lifetime bug): Run()'s main loop (see controller.go's
// `for { ... currentStatus = outcome.nextStatus }`) keeps dispatching further
// stages for the SAME entity/session under the SAME lease whenever the
// resolved target status is non-terminal. Before this fix,
// ingestGateResultForDispatch passed RetirementConfirmed: true
// unconditionally, so gatepersist.Coordinator released the lease after the
// very first gate stage even though the run was about to dispatch another —
// this test proves a non-terminal target (in_review, not one of the default
// workflow's terminal statuses) leaves the lease held.
func TestIngestGateResultForDispatch_NonTerminalTargetDoesNotReleaseLease(t *testing.T) {
	transitioner := &fakeTransitioner{status: map[string]string{"E01-F01-001": "todo"}}
	releaser := &fakeLeaseReleaser{}
	coordinator := gatepersist.NewCoordinator(
		&fakeNoteWriter{}, fakeNoteReader{}, fakeHistoryReader{},
		fakeStatusValidator{valid: map[string]bool{"todo": true, "in_review": true}},
		transitioner, transitioner, releaser,
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
	opts := RunOptions{ProjectRoot: t.TempDir(), RunID: "run-nonterm123def456abc123def456ab", EntityType: "task", SessionID: "sess-1"}
	disabled := false

	toStatus, _, err := controller.ingestGateResultForDispatch(
		context.Background(), "E01-F01-001", "todo", nextInfo, action, opts, dispatchResult, &disabled, 1,
	)
	if err != nil {
		t.Fatalf("expected successful gate ingestion, got error: %v", err)
	}
	if toStatus != "in_review" {
		t.Fatalf("expected transition to in_review, got %q", toStatus)
	}
	if releaser.released {
		t.Fatal("expected the lease to remain held after a non-terminal gate stage (Run()'s loop is about to dispatch another stage for this entity), but it was released")
	}
}

// TestIngestGateResultForDispatch_TerminalTargetReleasesLease is the sibling
// of the above: when the resolved target status IS terminal (Run()'s loop is
// about to stop dispatching this entity in this invocation), the lease must
// actually be released — the fix must not simply always withhold retirement.
// gatepersist.Coordinator (T-E34-F05-003 rework) gates release on BOTH
// RetirementConfirmed AND the distinct RunConcluded signal; this controller's
// terminal branch sets both on its second (retire) call.
func TestIngestGateResultForDispatch_TerminalTargetReleasesLease(t *testing.T) {
	transitioner := &fakeTransitioner{status: map[string]string{"E01-F01-001": "in_review"}}
	releaser := &fakeLeaseReleaser{}
	coordinator := gatepersist.NewCoordinator(
		&fakeNoteWriter{}, fakeNoteReader{}, fakeHistoryReader{},
		fakeStatusValidator{valid: map[string]bool{"in_review": true, "completed": true}},
		transitioner, transitioner, releaser,
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
		AvailableTransitions: []services.TransitionInfoWithAction{{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}}},
		Outcomes:             map[string]string{"pass": "completed"},
	}
	action := &config.PopulatedAction{Provider: "anthropic"}
	dispatchResult := &DispatchResult{
		ExitCode: 0,
		Stdout: `{"kind": "final", "recommended_outcome": "pass", "evidence": [],` +
			` "gate_result": {"schema_version": 1, "summary": "all checks passed"}}`,
		Duration: time.Millisecond,
	}
	opts := RunOptions{ProjectRoot: t.TempDir(), RunID: "run-term123def456abc123def456abcd", EntityType: "task", SessionID: "sess-1"}
	disabled := false

	toStatus, _, err := controller.ingestGateResultForDispatch(
		context.Background(), "E01-F01-001", "in_review", nextInfo, action, opts, dispatchResult, &disabled, 1,
	)
	if err != nil {
		t.Fatalf("expected successful gate ingestion, got error: %v", err)
	}
	if toStatus != "completed" {
		t.Fatalf("expected transition to completed, got %q", toStatus)
	}
	if !releaser.released {
		t.Fatal("expected the lease to be released once the resolved target status is terminal (Run()'s loop is about to stop for this entity), but it was not")
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

	toStatus, _, err := controller.ingestGateResultForDispatch(
		context.Background(), "E01-F01-001", "todo", nextInfo, action, opts, dispatchResult, &disabled, 1,
	)
	if err == nil {
		t.Fatalf("expected malformed envelope to fail closed, got toStatus=%q", toStatus)
	}
	if transitioner.status["E01-F01-001"] != "todo" {
		t.Fatalf("expected no transition to have occurred, entity is now %q", transitioner.status["E01-F01-001"])
	}
}
