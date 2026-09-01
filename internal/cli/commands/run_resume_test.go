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
	origTransitioner, origCoordinator := runResumeTransitionerOverride, runResumeCoordinatorOverride
	defer func() {
		runResumeID, runSession = origResumeID, origSession
		runResumeTransitionerOverride, runResumeCoordinatorOverride = origTransitioner, origCoordinator
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
	runResumeTransitionerOverride = &fakeParityTransitioner{
		status:         map[string]string{entityKey: "todo"},
		outcomes:       map[string]string{"pass": "in_review"},
		resultContract: "gate_result_v1",
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

// TestRunResumeRun_AlreadyTransitionedSkipsGateIngest asserts an
// already_transitioned decision performs no re-ingestion — the transition is
// already durably applied and must not be repeated.
func TestRunResumeRun_AlreadyTransitionedSkipsGateIngest(t *testing.T) {
	entityKey := "E01-F01-001"
	runID := "run-alreadydone1234567890abcdef123456"
	root := t.TempDir()
	dir, err := gaterun.RunDir(root, runID)
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	if _, err := gaterun.CreateResult(dir, []byte(parityEnvelope)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	s := gaterun.NewOperationState(runID, entityKey, "task", "todo", "todo", "digest")
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
	if out.Ingested {
		t.Fatal("expected no ingestion to have been attempted for an already_transitioned decision")
	}
}
