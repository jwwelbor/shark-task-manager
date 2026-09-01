package runner

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
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

// TestIngestGateResultForDispatch_NonTaskEntityTerminalStatusReleasesLease is
// the code-review round-8 regression guard for the task-level-vs-entity-level
// IsTerminalStatus gap: ingestGateResultForDispatch called
// c.workflowSvc.IsTerminalStatus (the controller's UNSCOPED, task-level-
// default workflow.Service) instead of scoping it to the dispatched entity's
// own type via .ForLevel(opts.EntityType). This was masked for task/feature/
// epic/bug/change entities because their terminal status names all happen to
// be "completed"/"cancelled" — but tech-debt's own default workflow
// (shark-data/workflow/tech-debt.yaml, wired by this same feature's
// T-E34-F05-005) uses "resolved"/"wont_fix" as its terminal names instead.
// Before the fix, a gate_result_v1 stage resolving a tech_debt entity to
// "resolved" was never recognized as terminal by the unscoped task-level
// service (which only knows "completed"/"cancelled"), so the lease was never
// released here — reproducing round 7's Finding 2 defect class for a
// different EntityType. This test constructs the controller's workflowSvc
// with the task-level default (level="") — exactly like every other test in
// this file — and drives a tech_debt entity resolving to "resolved" to prove
// the lease-release decision now consults the tech_debt-scoped service.
func TestIngestGateResultForDispatch_NonTaskEntityTerminalStatusReleasesLease(t *testing.T) {
	transitioner := &fakeTransitioner{status: map[string]string{"TD-001": "in_progress"}}
	releaser := &fakeLeaseReleaser{}
	coordinator := gatepersist.NewCoordinator(
		&fakeNoteWriter{}, fakeNoteReader{}, fakeHistoryReader{},
		fakeStatusValidator{valid: map[string]bool{"in_progress": true, "resolved": true}},
		transitioner, transitioner, releaser,
	)

	controller, err := NewRunController(RunControllerDeps{
		Transitioner: &oneShotStatusTransitioner{},
		ActionSvc:    &MockActionService{},
		// Task-level default (level=""), matching every other test in this
		// file — the bug is that ingestGateResultForDispatch used this
		// unscoped service directly instead of scoping it per-entity.
		WorkflowSvc: defaultWorkflowSvc(),
		Dispatchers: map[string]AgentDispatcher{"anthropic": &MockDispatcher{}},
		GateIngest: &GateIngestDeps{
			Coordinator:  coordinator,
			OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
		},
	})
	if err != nil {
		t.Fatalf("NewRunController: %v", err)
	}

	nextInfo := &services.NextStatusInfo{
		AvailableTransitions: []services.TransitionInfoWithAction{{TransitionInfo: workflow.TransitionInfo{TargetStatus: "resolved"}}},
		Outcomes:             map[string]string{"pass": "resolved"},
	}
	action := &config.PopulatedAction{Provider: "anthropic"}
	dispatchResult := &DispatchResult{
		ExitCode: 0,
		Stdout: `{"kind": "final", "recommended_outcome": "pass", "evidence": [],` +
			` "gate_result": {"schema_version": 1, "summary": "all checks passed"}}`,
		Duration: time.Millisecond,
	}
	opts := RunOptions{ProjectRoot: t.TempDir(), RunID: "run-techdebt123def456abc123def456", EntityType: "tech_debt", SessionID: "sess-1"}
	disabled := false

	toStatus, _, err := controller.ingestGateResultForDispatch(
		context.Background(), "TD-001", "in_progress", nextInfo, action, opts, dispatchResult, &disabled, 1,
	)
	if err != nil {
		t.Fatalf("expected successful gate ingestion, got error: %v", err)
	}
	if toStatus != "resolved" {
		t.Fatalf("expected transition to resolved, got %q", toStatus)
	}
	if !releaser.released {
		t.Fatal("expected the lease to be released once the tech_debt-scoped terminal status (resolved) is reached, but it was not — IsTerminalStatus was evaluated against the unscoped task-level default, which does not know 'resolved' is terminal")
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

// TestIngestGateResultForDispatch_MultiStageDispatchDoesNotReuseRunID is the
// code-review round-7 Finding 1 regression guard: `shark run` generates a
// single opts.RunID once per invocation and (before this fix) threaded it
// unchanged into gaterun.RunDir/CreateResult for EVERY gate_result_v1 stage
// dispatched for the same entity within that invocation. But
// gaterun.CreateResult's create-once contract treats run_id as identifying
// exactly ONE persisted result — a second, differently-digested envelope
// under the same run_id returns a *gaterun.ConflictError. Any workflow with
// two or more consecutive gate_result_v1 steps for the same entity in one
// `shark run` invocation (e.g. code_review -> qa) failed on the second gate
// stage. This drives ingestGateResultForDispatch twice with the SAME
// opts.RunID (mirroring Run()'s loop, which keeps opts.RunID constant across
// iterations) but two different stages (different currentStatus, different
// stageN, different envelope content) and asserts the second stage persists
// successfully rather than colliding with the first.
func TestIngestGateResultForDispatch_MultiStageDispatchDoesNotReuseRunID(t *testing.T) {
	key := "E01-F01-001"
	transitioner := &fakeTransitioner{status: map[string]string{key: "code_review"}}
	coordinator := gatepersist.NewCoordinator(
		&fakeNoteWriter{}, fakeNoteReader{}, fakeHistoryReader{},
		fakeStatusValidator{valid: map[string]bool{"code_review": true, "qa": true, "completed": true}},
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

	action := &config.PopulatedAction{Provider: "anthropic"}
	opts := RunOptions{ProjectRoot: t.TempDir(), RunID: "run-multi-stage-same-id", EntityType: "task", SessionID: "sess-1"}
	disabled := false

	// Stage 1: code_review -> qa.
	stage1NextInfo := &services.NextStatusInfo{
		AvailableTransitions: []services.TransitionInfoWithAction{{TransitionInfo: workflow.TransitionInfo{TargetStatus: "qa"}}},
		Outcomes:             map[string]string{"pass": "qa"},
	}
	stage1Dispatch := &DispatchResult{
		ExitCode: 0,
		Stdout: `{"kind": "final", "recommended_outcome": "pass", "evidence": [],` +
			` "gate_result": {"schema_version": 1, "summary": "code review passed"}}`,
		Duration: time.Millisecond,
	}
	toStatus1, _, err := controller.ingestGateResultForDispatch(
		context.Background(), key, "code_review", stage1NextInfo, action, opts, stage1Dispatch, &disabled, 1,
	)
	if err != nil {
		t.Fatalf("stage 1 (code_review) gate ingestion failed: %v", err)
	}
	if toStatus1 != "qa" {
		t.Fatalf("expected stage 1 to transition to qa, got %q", toStatus1)
	}

	// Stage 2: qa -> completed, dispatched under the SAME opts.RunID
	// (mirroring Run()'s loop, which never changes opts.RunID between
	// iterations) but a different stage. Before the fix this collided with
	// stage 1's already-accepted result.json under the same run directory.
	stage2NextInfo := &services.NextStatusInfo{
		AvailableTransitions: []services.TransitionInfoWithAction{{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}}},
		Outcomes:             map[string]string{"pass": "completed"},
	}
	stage2Dispatch := &DispatchResult{
		ExitCode: 0,
		Stdout: `{"kind": "final", "recommended_outcome": "pass", "evidence": [],` +
			` "gate_result": {"schema_version": 1, "summary": "qa passed"}}`,
		Duration: time.Millisecond,
	}
	toStatus2, _, err := controller.ingestGateResultForDispatch(
		context.Background(), key, "qa", stage2NextInfo, action, opts, stage2Dispatch, &disabled, 2,
	)
	if err != nil {
		t.Fatalf("stage 2 (qa) gate ingestion failed (expected no collision with stage 1's persisted result under the same run_id): %v", err)
	}
	if toStatus2 != "completed" {
		t.Fatalf("expected stage 2 to transition to completed, got %q", toStatus2)
	}
}

// TestIngestGateResultForDispatch_SameStageConflictingReplayStillFailsClosed
// is the discriminating counterpart of the multi-stage test above: it proves
// gateStageRunID's per-stage scoping did NOT weaken gaterun's create-once
// contract for retries WITHIN a single stage (the rework brief's explicit
// constraint — "must not break gaterun's existing create-once/idempotent-
// replay guarantee for a SINGLE stage's own retries"). Two calls with the
// SAME stageN and SAME opts.RunID but DIFFERENT envelope content must still
// collide with a *gaterun.ConflictError, exactly as before this fix.
func TestIngestGateResultForDispatch_SameStageConflictingReplayStillFailsClosed(t *testing.T) {
	key := "E01-F01-001"
	transitioner := &fakeTransitioner{status: map[string]string{key: "code_review"}}
	coordinator := gatepersist.NewCoordinator(
		&fakeNoteWriter{}, fakeNoteReader{}, fakeHistoryReader{},
		fakeStatusValidator{valid: map[string]bool{"code_review": true, "qa": true}},
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
		AvailableTransitions: []services.TransitionInfoWithAction{{TransitionInfo: workflow.TransitionInfo{TargetStatus: "qa"}}},
		Outcomes:             map[string]string{"pass": "qa"},
	}
	action := &config.PopulatedAction{Provider: "anthropic"}
	opts := RunOptions{ProjectRoot: t.TempDir(), RunID: "run-same-stage-conflict", EntityType: "task", SessionID: "sess-1"}
	disabled := false

	first := &DispatchResult{
		ExitCode: 0,
		Stdout: `{"kind": "final", "recommended_outcome": "pass", "evidence": [],` +
			` "gate_result": {"schema_version": 1, "summary": "first attempt"}}`,
		Duration: time.Millisecond,
	}
	if _, _, err := controller.ingestGateResultForDispatch(
		context.Background(), key, "code_review", nextInfo, action, opts, first, &disabled, 1,
	); err != nil {
		t.Fatalf("expected the first attempt at stage 1 to succeed: %v", err)
	}

	conflicting := &DispatchResult{
		ExitCode: 0,
		Stdout: `{"kind": "final", "recommended_outcome": "pass", "evidence": [],` +
			` "gate_result": {"schema_version": 1, "summary": "a DIFFERENT attempt at the SAME stage"}}`,
		Duration: time.Millisecond,
	}
	_, _, err = controller.ingestGateResultForDispatch(
		context.Background(), key, "code_review", nextInfo, action, opts, conflicting, &disabled, 1,
	)
	if err == nil {
		t.Fatal("expected a conflicting replay at the SAME stage (same stageN, same opts.RunID) to fail closed")
	}
	if !gaterun.IsConflict(err) {
		t.Fatalf("expected a *gaterun.ConflictError (create-once contract preserved for same-stage retries), got: %v", err)
	}
}

// TestIngestGateResultForDispatch_SameStageIdenticalReplayIsIdempotent is the
// idempotent-replay sibling of the conflict test above: a second call with
// the SAME stageN/opts.RunID and BYTE-IDENTICAL envelope content must
// succeed without error and without writing a second gate-summary note —
// gaterun's create-once contract treats a byte-identical replay as
// idempotent success, and gateStageRunID must preserve that for a single
// stage's own retries.
func TestIngestGateResultForDispatch_SameStageIdenticalReplayIsIdempotent(t *testing.T) {
	key := "E01-F01-001"
	transitioner := &fakeTransitioner{status: map[string]string{key: "code_review"}}
	notes := &fakeNoteWriter{}
	coordinator := gatepersist.NewCoordinator(
		notes, fakeNoteReader{}, fakeHistoryReader{},
		fakeStatusValidator{valid: map[string]bool{"code_review": true, "qa": true}},
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
		AvailableTransitions: []services.TransitionInfoWithAction{{TransitionInfo: workflow.TransitionInfo{TargetStatus: "qa"}}},
		Outcomes:             map[string]string{"pass": "qa"},
	}
	action := &config.PopulatedAction{Provider: "anthropic"}
	opts := RunOptions{ProjectRoot: t.TempDir(), RunID: "run-same-stage-idempotent", EntityType: "task", SessionID: "sess-1"}
	disabled := false
	identical := &DispatchResult{
		ExitCode: 0,
		Stdout: `{"kind": "final", "recommended_outcome": "pass", "evidence": [],` +
			` "gate_result": {"schema_version": 1, "summary": "identical every time"}}`,
		Duration: time.Millisecond,
	}

	toStatus1, _, err := controller.ingestGateResultForDispatch(
		context.Background(), key, "code_review", nextInfo, action, opts, identical, &disabled, 1,
	)
	if err != nil {
		t.Fatalf("expected the first attempt at stage 1 to succeed: %v", err)
	}
	notesAfterFirst := len(notes.notes)
	if notesAfterFirst == 0 {
		t.Fatal("expected the first attempt to write at least one gate-summary note")
	}

	toStatus2, _, err := controller.ingestGateResultForDispatch(
		context.Background(), key, "code_review", nextInfo, action, opts, identical, &disabled, 1,
	)
	if err != nil {
		t.Fatalf("expected a byte-identical replay at the SAME stage to succeed idempotently, got error: %v", err)
	}
	if toStatus1 != toStatus2 {
		t.Fatalf("expected both attempts to resolve the same target status, got %q then %q", toStatus1, toStatus2)
	}
	if len(notes.notes) != notesAfterFirst {
		t.Fatalf("expected the idempotent replay to write no additional notes, had %d notes after first attempt, %d after replay", notesAfterFirst, len(notes.notes))
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
