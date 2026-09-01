package commands

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
)

func TestResolveResumeStatus_NoDurableResult(t *testing.T) {
	root := t.TempDir()
	_, err := resolveResumeStatus(root, "run-1", "task", "E01-F01-001", time.Now())
	if err == nil {
		t.Fatal("resolveResumeStatus with no result.json: want error, got nil")
	}
}

func TestResolveResumeStatus_ResultWithoutStateReportsResumeNextOperation(t *testing.T) {
	root := t.TempDir()
	dir, err := gaterun.RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	if _, err := gaterun.CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}

	out, err := resolveResumeStatus(root, "run-1", "task", "E01-F01-001", time.Now())
	if err != nil {
		t.Fatalf("resolveResumeStatus: %v", err)
	}
	if out.Action != gaterun.ResumeActionResumeNextOperation {
		t.Errorf("Action = %s, want resume_next_operation", out.Action)
	}
	if out.Status != nil {
		t.Errorf("Status = %+v, want nil (no operation-state.json yet)", out.Status)
	}
	if out.RunID != "run-1" || out.EntityKey != "E01-F01-001" || out.EntityType != "task" {
		t.Errorf("identity echo mismatch: %+v", out)
	}
}

func TestResolveResumeStatus_MatchingIdentityNoWarning(t *testing.T) {
	root := t.TempDir()
	dir, err := gaterun.RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	if _, err := gaterun.CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	s := gaterun.NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	if err := s.MarkPersistenceComplete(); err != nil {
		t.Fatalf("mark: %v", err)
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	out, err := resolveResumeStatus(root, "run-1", "task", "E01-F01-001", time.Now())
	if err != nil {
		t.Fatalf("resolveResumeStatus: %v", err)
	}
	if out.Action != gaterun.ResumeActionResumeTransition {
		t.Errorf("Action = %s, want resume_transition", out.Action)
	}
	if out.Status == nil {
		t.Fatal("Status = nil, want a projection once operation-state.json exists")
	}
}

func TestResolveResumeStatus_MismatchedIdentityFailsClosed(t *testing.T) {
	root := t.TempDir()
	dir, err := gaterun.RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	if _, err := gaterun.CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	s := gaterun.NewOperationState("run-1", "E01-F01-999", "task", "in_review", "code_review", "digest")
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := resolveResumeStatus(root, "run-1", "task", "E01-F01-001", time.Now()); err == nil {
		t.Fatal("resolveResumeStatus with mismatched recorded entity_key: want error, got nil")
	}
}

func TestResolveResumeStatus_AlreadyTransitioned(t *testing.T) {
	root := t.TempDir()
	dir, err := gaterun.RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	if _, err := gaterun.CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	s := gaterun.NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	if err := s.MarkPersistenceComplete(); err != nil {
		t.Fatalf("mark persistence: %v", err)
	}
	if err := s.MarkTransitionApplied(); err != nil {
		t.Fatalf("mark transition: %v", err)
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	out, err := resolveResumeStatus(root, "run-1", "task", "E01-F01-001", time.Now())
	if err != nil {
		t.Fatalf("resolveResumeStatus: %v", err)
	}
	if out.Action != gaterun.ResumeActionAlreadyTransitioned {
		t.Errorf("Action = %s, want already_transitioned", out.Action)
	}
}

// fakeGateStepResolver is a minimal, no-database gateStepResolver fake:
// resumeGateIngestIfConfigured resolves a step's result_contract/outcomes/
// outcome_roles by an explicit status argument (the durably recorded gate
// step, decision.State.Gate), so this fake ignores the status parameter and
// always returns its fixed configuration — matching how the production
// *workflow.Service is used here (one resolved step per test fixture).
type fakeGateStepResolver struct {
	resultContract string
	outcomes       map[string]string
	outcomeRoles   map[string]gateresult.OutcomeRole
}

func (f *fakeGateStepResolver) GetResultContract(string) string      { return f.resultContract }
func (f *fakeGateStepResolver) GetOutcomes(string) map[string]string { return f.outcomes }
func (f *fakeGateStepResolver) GetOutcomeRoles(string) map[string]gateresult.OutcomeRole {
	return f.outcomeRoles
}

func TestRunResumeRun_RequiresSession(t *testing.T) {
	origResumeID, origSession := runResumeID, runSession
	defer func() { runResumeID, runSession = origResumeID, origSession }()

	runResumeID = "run-1"
	runSession = ""
	if err := runResumeRun(context.Background(), "task", "E01-F01-001"); err == nil {
		t.Fatal("runResumeRun without --session: want error, got nil")
	}
}

// TestRunResumeRun_GateResultV1ResumeTransitionReIngestsStoredEnvelope is
// T-E34-F05-004's REQ-F-005/item-4 wiring proof: for a gate_result_v1 step
// whose sidecar records persistence_complete (resume_transition), --resume-run
// must re-ingest the durably stored envelope through the same
// runner.IngestGateResult boundary the core dispatch loop and Rider's
// --apply-result surface call, applying the transition — not merely report
// status.
func TestRunResumeRun_GateResultV1ResumeTransitionReIngestsStoredEnvelope(t *testing.T) {
	origResumeID, origSession := runResumeID, runSession
	origResolver, origCoordinator := runResumeWorkflowServiceOverride, runResumeCoordinatorOverride
	defer func() {
		runResumeID, runSession = origResumeID, origSession
		runResumeWorkflowServiceOverride, runResumeCoordinatorOverride = origResolver, origCoordinator
	}()

	entityKey := "E01-F01-001"
	runID := "run-resumegate1234567890abcdef123456"
	root := t.TempDir()
	dir, err := gaterun.RunDir(root, runID)
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	if _, err := gaterun.CreateResult(dir, []byte(parityEnvelope)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	digest, err := gaterun.ComputeOperationDigest(entityKey, "task", "todo", "todo", []byte(parityEnvelope))
	if err != nil {
		t.Fatalf("ComputeOperationDigest: %v", err)
	}
	s := gaterun.NewOperationState(runID, entityKey, "task", "todo", "todo", digest)
	if err := s.MarkPersistenceComplete(); err != nil {
		t.Fatalf("mark persistence complete: %v", err)
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("save operation state: %v", err)
	}

	transition := &fakeParityTransition{status: map[string]string{entityKey: "todo"}}
	runResumeWorkflowServiceOverride = &fakeGateStepResolver{
		resultContract: "gate_result_v1",
		outcomes:       map[string]string{"pass": "in_review"},
		outcomeRoles:   map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
	}
	runResumeCoordinatorOverride = gatepersist.NewCoordinator(
		&fakeParityNoteWriter{}, fakeParityNoteReader{}, fakeParityHistoryReader{},
		fakeParityStatusValidator{}, transition, transition, fakeParityLeaseReleaser{},
	)

	runResumeID = runID
	runSession = "sess-resume"

	out, decision, err := resolveResumeStatusAndDecision(root, runID, "task", entityKey, time.Now())
	if err != nil {
		t.Fatalf("resolveResumeStatusAndDecision: %v", err)
	}
	if decision.Action != gaterun.ResumeActionResumeTransition {
		t.Fatalf("expected resume_transition, got %s", decision.Action)
	}

	if err := resumeGateIngestIfConfigured(context.Background(), root, "task", entityKey, decision, out); err != nil {
		t.Fatalf("resumeGateIngestIfConfigured: %v", err)
	}

	if !out.Ingested {
		t.Fatal("expected Ingested=true")
	}
	if out.ToStatus != "in_review" {
		t.Fatalf("expected ToStatus=in_review, got %q", out.ToStatus)
	}
	if transition.status[entityKey] != "in_review" {
		t.Fatalf("expected the coordinator's transitioner to record in_review, got %q", transition.status[entityKey])
	}
}

// TestRunResumeRun_PartialPersistenceResumeCompletesTransition is the
// partial-resume fixture: operation-state.json records PersistenceStatePending
// (resume_next_operation — persistence never completed, e.g. a crash right
// after result.json was durably written). --resume-run must still complete
// the ingestion via the stored envelope, producing the same final result as
// a fresh ingestion of that same envelope would.
func TestRunResumeRun_PartialPersistenceResumeCompletesTransition(t *testing.T) {
	origResumeID, origSession := runResumeID, runSession
	origResolver, origCoordinator := runResumeWorkflowServiceOverride, runResumeCoordinatorOverride
	defer func() {
		runResumeID, runSession = origResumeID, origSession
		runResumeWorkflowServiceOverride, runResumeCoordinatorOverride = origResolver, origCoordinator
	}()

	entityKey := "E01-F01-001"
	runID := "run-partial1234567890abcdef1234567890"
	root := t.TempDir()
	dir, err := gaterun.RunDir(root, runID)
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	if _, err := gaterun.CreateResult(dir, []byte(parityEnvelope)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	digest, err := gaterun.ComputeOperationDigest(entityKey, "task", "todo", "todo", []byte(parityEnvelope))
	if err != nil {
		t.Fatalf("ComputeOperationDigest: %v", err)
	}
	// PersistenceStatePending (the NewOperationState default) is the
	// resume_next_operation case — no MarkPersistenceComplete call.
	s := gaterun.NewOperationState(runID, entityKey, "task", "todo", "todo", digest)
	if err := s.Save(dir); err != nil {
		t.Fatalf("save operation state: %v", err)
	}

	transition := &fakeParityTransition{status: map[string]string{entityKey: "todo"}}
	runResumeWorkflowServiceOverride = &fakeGateStepResolver{
		resultContract: "gate_result_v1",
		outcomes:       map[string]string{"pass": "in_review"},
		outcomeRoles:   map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
	}
	runResumeCoordinatorOverride = gatepersist.NewCoordinator(
		&fakeParityNoteWriter{}, fakeParityNoteReader{}, fakeParityHistoryReader{},
		fakeParityStatusValidator{}, transition, transition, fakeParityLeaseReleaser{},
	)

	runResumeID = runID
	runSession = "sess-resume"

	out, decision, err := resolveResumeStatusAndDecision(root, runID, "task", entityKey, time.Now())
	if err != nil {
		t.Fatalf("resolveResumeStatusAndDecision: %v", err)
	}
	if decision.Action != gaterun.ResumeActionResumeNextOperation {
		t.Fatalf("expected resume_next_operation, got %s", decision.Action)
	}

	if err := resumeGateIngestIfConfigured(context.Background(), root, "task", entityKey, decision, out); err != nil {
		t.Fatalf("resumeGateIngestIfConfigured: %v", err)
	}

	if !out.Ingested || out.ToStatus != "in_review" || !out.Transitioned {
		t.Fatalf("expected a completed transition to in_review from the partial-resume path, got %+v", out)
	}
	if transition.status[entityKey] != "in_review" {
		t.Fatalf("expected the coordinator's transitioner to record in_review, got %q", transition.status[entityKey])
	}
}

// fakeCountingLeaseReleaser records every Release call so a test can assert
// the lease was released exactly once — never zero times (F-2's defect) and
// never more than once (a double-release would be its own bug).
type fakeCountingLeaseReleaser struct {
	calls []runReleaseCall
}

func (f *fakeCountingLeaseReleaser) Release(_ context.Context, entityType, entityKey, sessionID, outcome string, force bool) (bool, error) {
	f.calls = append(f.calls, runReleaseCall{entityType: entityType, entityKey: entityKey, sessionID: sessionID, outcome: outcome, force: force})
	return true, nil
}

func setUpAlreadyTransitionedFixture(t *testing.T, entityKey, runID, root string) *gaterun.ResumeDecision {
	t.Helper()
	dir, err := gaterun.RunDir(root, runID)
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	if _, err := gaterun.CreateResult(dir, []byte(parityEnvelope)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	digest, err := gaterun.ComputeOperationDigest(entityKey, "task", "todo", "todo", []byte(parityEnvelope))
	if err != nil {
		t.Fatalf("ComputeOperationDigest: %v", err)
	}
	s := gaterun.NewOperationState(runID, entityKey, "task", "todo", "todo", digest)
	if err := s.MarkPersistenceComplete(); err != nil {
		t.Fatalf("mark persistence complete: %v", err)
	}
	if err := s.MarkTransitionApplied(); err != nil {
		t.Fatalf("mark transition applied: %v", err)
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("save operation state: %v", err)
	}

	out, decision, err := resolveResumeStatusAndDecision(root, runID, "task", entityKey, time.Now())
	if err != nil {
		t.Fatalf("resolveResumeStatusAndDecision: %v", err)
	}
	if decision.Action != gaterun.ResumeActionAlreadyTransitioned {
		t.Fatalf("expected already_transitioned, got %s", decision.Action)
	}
	_ = out
	return decision
}

// TestRunResumeRun_AlreadyTransitionedVerifiesAndReleasesLease is F-2
// (T-E34-F05-004 rework): gaterun/resume.go documents
// ResumeActionAlreadyTransitioned's contract as "verify the expected live
// target state and release the lease; it must not repeat the transition."
// Before this fix, --resume-run skipped gate ingestion entirely for this
// action, so gatepersist.Coordinator — which owns lease release per
// run_resume.go's own doc comment — was never invoked, leaving the parent's
// lease held forever in the "transition succeeded, crashed before release"
// crash window feature.md's "Replay a committed result" acceptance scenario
// names.
func TestRunResumeRun_AlreadyTransitionedVerifiesAndReleasesLease(t *testing.T) {
	origResumeID, origSession := runResumeID, runSession
	origResolver, origCoordinator := runResumeWorkflowServiceOverride, runResumeCoordinatorOverride
	defer func() {
		runResumeID, runSession = origResumeID, origSession
		runResumeWorkflowServiceOverride, runResumeCoordinatorOverride = origResolver, origCoordinator
	}()

	entityKey := "E01-F01-001"
	runID := "run-alreadydone1234567890abcdef123456"
	root := t.TempDir()
	decision := setUpAlreadyTransitionedFixture(t, entityKey, runID, root)

	// The entity already sits at the recorded target ("in_review",
	// parityEnvelope's "pass" outcome) — the expected live state for a
	// resume landing after the transition succeeded but before the lease
	// was released.
	transition := &fakeParityTransition{status: map[string]string{entityKey: "in_review"}}
	releaser := &fakeCountingLeaseReleaser{}
	runResumeWorkflowServiceOverride = &fakeGateStepResolver{
		resultContract: "gate_result_v1",
		outcomes:       map[string]string{"pass": "in_review"},
		outcomeRoles:   map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
	}
	runResumeCoordinatorOverride = gatepersist.NewCoordinator(
		&fakeParityNoteWriter{}, fakeParityNoteReader{}, fakeParityHistoryReader{},
		fakeParityStatusValidator{}, transition, transition, releaser,
	)

	runResumeID = runID
	runSession = "sess-resume"

	out := &resumeRunOutput{}
	if err := resumeGateIngestIfConfigured(context.Background(), root, "task", entityKey, decision, out); err != nil {
		t.Fatalf("resumeGateIngestIfConfigured: %v", err)
	}

	if out.ToStatus != "in_review" {
		t.Fatalf("expected ToStatus=in_review, got %q", out.ToStatus)
	}
	if !out.LeaseReleased {
		t.Fatal("expected LeaseReleased=true")
	}
	if len(releaser.calls) != 1 {
		t.Fatalf("expected exactly 1 lease release call, got %d: %+v", len(releaser.calls), releaser.calls)
	}
	if releaser.calls[0].sessionID != "sess-resume" {
		t.Fatalf("expected release for session sess-resume, got %+v", releaser.calls[0])
	}
}

// TestRunResumeRun_AlreadyTransitionedFailsClosedOnDivergedStatus proves the
// other half of F-2's contract: an already_transitioned resume must verify
// the live status before releasing anything — if something else moved the
// entity away from the recorded target between the crashed run and this
// resume, the lease must NOT be released, mirroring
// gatepersist.Coordinator's own divergence check.
func TestRunResumeRun_AlreadyTransitionedFailsClosedOnDivergedStatus(t *testing.T) {
	origResumeID, origSession := runResumeID, runSession
	origResolver, origCoordinator := runResumeWorkflowServiceOverride, runResumeCoordinatorOverride
	defer func() {
		runResumeID, runSession = origResumeID, origSession
		runResumeWorkflowServiceOverride, runResumeCoordinatorOverride = origResolver, origCoordinator
	}()

	entityKey := "E01-F01-001"
	runID := "run-alreadydiverged1234567890abcdef12"
	root := t.TempDir()
	decision := setUpAlreadyTransitionedFixture(t, entityKey, runID, root)

	// Diverged: recorded target is "in_review", but the entity now sits at
	// "blocked" (e.g. a human `status set --force`).
	transition := &fakeParityTransition{status: map[string]string{entityKey: "blocked"}}
	releaser := &fakeCountingLeaseReleaser{}
	runResumeWorkflowServiceOverride = &fakeGateStepResolver{
		resultContract: "gate_result_v1",
		outcomes:       map[string]string{"pass": "in_review"},
		outcomeRoles:   map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
	}
	runResumeCoordinatorOverride = gatepersist.NewCoordinator(
		&fakeParityNoteWriter{}, fakeParityNoteReader{}, fakeParityHistoryReader{},
		fakeParityStatusValidator{}, transition, transition, releaser,
	)

	runResumeID = runID
	runSession = "sess-resume"

	out := &resumeRunOutput{}
	if err := resumeGateIngestIfConfigured(context.Background(), root, "task", entityKey, decision, out); err == nil {
		t.Fatal("expected resumeGateIngestIfConfigured to fail closed on a diverged live status")
	}
	if len(releaser.calls) != 0 {
		t.Fatalf("expected no lease release when verification fails closed, got %d calls", len(releaser.calls))
	}
}

// TestRunResumeRun_UninitializedStateReIngestsUsingLiveStatus is the sibling
// of F-2, swept in the same pass (T-E34-F05-004 rework): the
// create-once-result/before-state-init crash window (result.json committed,
// operation-state.json never written — gaterun.DecideResume's nil-State
// resume_next_operation case) must also reach the coordinator, or a crash
// in this narrower window leaves the parent's lease held forever exactly
// like F-2. Unlike the already_transitioned case, the entity has NOT
// transitioned here, so its live current status is the correct source
// status/gate to key off of.
func TestRunResumeRun_UninitializedStateReIngestsUsingLiveStatus(t *testing.T) {
	origResumeID, origSession := runResumeID, runSession
	origTransitioner, origCoordinator := runResumeTransitionerOverride, runResumeCoordinatorOverride
	defer func() {
		runResumeID, runSession = origResumeID, origSession
		runResumeTransitionerOverride, runResumeCoordinatorOverride = origTransitioner, origCoordinator
	}()

	entityKey := "E01-F01-001"
	runID := "run-uninit1234567890abcdef1234567890"
	root := t.TempDir()
	dir, err := gaterun.RunDir(root, runID)
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	// result.json only — operation-state.json is never written, matching
	// the create-once-result/before-state-init crash window.
	if _, err := gaterun.CreateResult(dir, []byte(parityEnvelope)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}

	out, decision, err := resolveResumeStatusAndDecision(root, runID, "task", entityKey, time.Now())
	if err != nil {
		t.Fatalf("resolveResumeStatusAndDecision: %v", err)
	}
	if decision.Action != gaterun.ResumeActionResumeNextOperation || decision.State != nil {
		t.Fatalf("expected resume_next_operation with nil State, got action=%s state=%+v", decision.Action, decision.State)
	}

	transition := &fakeParityTransition{status: map[string]string{entityKey: "todo"}}
	releaser := &fakeCountingLeaseReleaser{}
	runResumeTransitionerOverride = &fakeParityTransitioner{
		status:         map[string]string{entityKey: "todo"},
		outcomes:       map[string]string{"pass": "in_review"},
		resultContract: "gate_result_v1",
		outcomeRoles:   map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
	}
	runResumeCoordinatorOverride = gatepersist.NewCoordinator(
		&fakeParityNoteWriter{}, fakeParityNoteReader{}, fakeParityHistoryReader{},
		fakeParityStatusValidator{}, transition, transition, releaser,
	)

	runResumeID = runID
	runSession = "sess-resume"

	if err := resumeGateIngestForUninitializedState(context.Background(), root, "task", entityKey, decision, out); err != nil {
		t.Fatalf("resumeGateIngestForUninitializedState: %v", err)
	}

	if !out.Ingested || out.ToStatus != "in_review" || !out.Transitioned {
		t.Fatalf("expected a completed transition to in_review, got %+v", out)
	}
	if !out.LeaseReleased {
		t.Fatal("expected LeaseReleased=true")
	}
	if len(releaser.calls) != 1 {
		t.Fatalf("expected exactly 1 lease release call, got %d", len(releaser.calls))
	}
	if transition.status[entityKey] != "in_review" {
		t.Fatalf("expected the coordinator's transitioner to record in_review, got %q", transition.status[entityKey])
	}
}
