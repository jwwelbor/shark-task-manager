package commands

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// The fakes below are local, minimal implementations of
// runner.EntityTransitioner and gatepersist's injected interfaces, built for
// exercising applyResultIngest/runner.IngestGateResult without a database
// (the CLI-tests golden rule: never a real database in a CLI-command test —
// see .claude/rules/testing/cli-tests.md).

type fakeParityTransitioner struct {
	status         map[string]string
	outcomes       map[string]string
	resultContract string
	outcomeRoles   map[string]gateresult.OutcomeRole
}

func (t *fakeParityTransitioner) GetNextStatus(_ context.Context, key string) (*services.NextStatusInfo, error) {
	return &services.NextStatusInfo{
		CurrentStatus:  t.status[key],
		Outcomes:       t.outcomes,
		ResultContract: t.resultContract,
		OutcomeRoles:   t.outcomeRoles,
	}, nil
}

func (t *fakeParityTransitioner) TransitionStatus(_ context.Context, key string, targetStatus string, _ services.TransitionOptions) (*services.TransitionResult, error) {
	from := t.status[key]
	t.status[key] = targetStatus
	return &services.TransitionResult{FromStatus: from, ToStatus: targetStatus, Transitioned: from != targetStatus}, nil
}

type fakeParityNoteWriter struct{ count int }

func (f *fakeParityNoteWriter) AddNoteWithMetadata(context.Context, models.EntityType, string, string, string, string, string) (*models.EntityNote, error) {
	f.count++
	return &models.EntityNote{}, nil
}

type fakeParityNoteReader struct{}

func (fakeParityNoteReader) ListNotes(context.Context, models.EntityType, string, []string) ([]*models.EntityNote, error) {
	return nil, nil
}

type fakeParityHistoryReader struct{}

func (fakeParityHistoryReader) GetHistory(context.Context, models.EntityType, string) ([]*models.EntityHistory, error) {
	return nil, nil
}

type fakeParityStatusValidator struct{}

func (fakeParityStatusValidator) IsValidStatus(models.EntityType, string) bool { return true }

type fakeParityTransition struct {
	status map[string]string
}

func (t *fakeParityTransition) Transition(_ context.Context, _ models.EntityType, entityKey, targetStatus, _, _ string, _ gatepersist.TransitionGuard) (string, bool, error) {
	from := t.status[entityKey]
	t.status[entityKey] = targetStatus
	return from, from != targetStatus, nil
}

func (t *fakeParityTransition) CurrentStatus(_ context.Context, _ models.EntityType, entityKey string) (string, error) {
	return t.status[entityKey], nil
}

type fakeParityLeaseReleaser struct{}

func (fakeParityLeaseReleaser) Release(context.Context, string, string, string, string, bool) (bool, error) {
	return true, nil
}

// trackingLeaseReleaser records whether Release was called, so a test can
// assert on release behavior (fakeParityLeaseReleaser above always reports
// success but never records the call).
type trackingLeaseReleaser struct{ released bool }

func (r *trackingLeaseReleaser) Release(context.Context, string, string, string, string, bool) (bool, error) {
	r.released = true
	return true, nil
}

// fakeTerminalStatusChecker is a minimal terminalStatusChecker stub: it
// reports every status in terminal as terminal, everything else as not.
type fakeTerminalStatusChecker struct{ terminal map[string]bool }

func (c fakeTerminalStatusChecker) IsTerminalStatus(status string) bool {
	return c.terminal[status]
}

const parityEnvelope = `{
	"kind": "final",
	"recommended_outcome": "pass",
	"evidence": [{"kind": "test_run", "pointer": "artifacts/test.log"}],
	"gate_result": {"schema_version": 1, "summary": "all checks passed"}
}`

// TestApplyResultIngest_ParityWithDirectCoreIngestion is T-E34-F05-004's
// REQ-F-005 parity test: it runs the identical fixture through the core
// ingestion call (runner.IngestGateResult, called directly — the same way
// internal/runner/controller.go's gate_result_v1 branch calls it) and
// Rider's --apply-result path (applyResultIngest) against two independent,
// identically-seeded coordinators, then compares the normalized results.
func TestApplyResultIngest_ParityWithDirectCoreIngestion(t *testing.T) {
	entityKey := "E01-F01-001"
	runID := "run-parity1234567890abcdef1234567890"
	outcomeRoles := map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}
	outcomes := map[string]string{"pass": "in_review"}

	newCoordinator := func() *gatepersist.Coordinator {
		transition := &fakeParityTransition{status: map[string]string{entityKey: "todo"}}
		return gatepersist.NewCoordinator(
			&fakeParityNoteWriter{}, fakeParityNoteReader{}, fakeParityHistoryReader{},
			fakeParityStatusValidator{}, transition, transition, fakeParityLeaseReleaser{},
		)
	}

	// Core path: called the same way controller.go's gate_result_v1 branch
	// calls it, with the same run/entity/route context.
	coreCoordinator := newCoordinator()
	coreResult, coreErr := runner.IngestGateResult(context.Background(), runner.GateIngestRequest{
		EnvelopeBytes: []byte(parityEnvelope),
		Coordinator:   coreCoordinator,
		ProjectRoot:   t.TempDir(),
		RunID:         runID,
		EntityKey:     entityKey,
		EntityType:    models.EntityTypeTask,
		SourceStatus:  "todo",
		Gate:          "todo",
		Session:       gatepersist.Session{ID: "sess-core"},
		OutcomeRoles:  outcomeRoles,
		Outcomes:      outcomes,
	})

	// Rider path: applyResultIngest, exactly what --apply-result calls.
	riderCoordinator := newCoordinator()
	riderResult, riderErr := applyResultIngest(context.Background(), applyResultDeps{
		Transitioner: &fakeParityTransitioner{status: map[string]string{entityKey: "todo"}, outcomes: outcomes},
		Coordinator:  riderCoordinator,
		ProjectRoot:  t.TempDir(),
		RunID:        runID,
		EntityType:   "task",
		EntityKey:    entityKey,
		SessionID:    "sess-rider",
		OutcomeRoles: outcomeRoles,
		WorkflowSvc:  fakeTerminalStatusChecker{},
	}, []byte(parityEnvelope))

	if coreErr != nil || riderErr != nil {
		t.Fatalf("expected both paths to succeed, core err=%v rider err=%v", coreErr, riderErr)
	}
	if coreResult.OutcomeKey != riderResult.OutcomeKey {
		t.Errorf("outcome key mismatch: core=%q rider=%q", coreResult.OutcomeKey, riderResult.OutcomeKey)
	}
	if coreResult.Role != riderResult.Role {
		t.Errorf("role mismatch: core=%q rider=%q", coreResult.Role, riderResult.Role)
	}
	if coreResult.ToStatus != riderResult.ToStatus {
		t.Errorf("to_status mismatch: core=%q rider=%q", coreResult.ToStatus, riderResult.ToStatus)
	}
	if coreResult.Transitioned != riderResult.Transitioned {
		t.Errorf("transitioned mismatch: core=%v rider=%v", coreResult.Transitioned, riderResult.Transitioned)
	}
	if coreResult.OperationDigest != riderResult.OperationDigest {
		t.Errorf("operation digest mismatch: core=%q rider=%q", coreResult.OperationDigest, riderResult.OperationDigest)
	}
}

// TestApplyResultIngest_MalformedEnvelopeFailsClosedSameAsCore asserts both
// paths reject an identical malformed fixture with the same rejection class
// (a decode error), never a silent legacy fallback.
func TestApplyResultIngest_MalformedEnvelopeFailsClosedSameAsCore(t *testing.T) {
	entityKey := "E01-F01-001"
	malformed := []byte(`{"kind": "final", "recommended_outcome":`)

	_, coreErr := runner.IngestGateResult(context.Background(), runner.GateIngestRequest{
		EnvelopeBytes: malformed,
		Coordinator:   gatepersist.NewCoordinator(&fakeParityNoteWriter{}, fakeParityNoteReader{}, fakeParityHistoryReader{}, fakeParityStatusValidator{}, &fakeParityTransition{status: map[string]string{}}, &fakeParityTransition{status: map[string]string{}}, fakeParityLeaseReleaser{}),
		ProjectRoot:   t.TempDir(),
		RunID:         "run-parity1234567890abcdef1234567891",
		EntityKey:     entityKey,
		EntityType:    models.EntityTypeTask,
		SourceStatus:  "todo",
		Gate:          "todo",
		Session:       gatepersist.Session{ID: "sess-core"},
		OutcomeRoles:  map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
		Outcomes:      map[string]string{"pass": "in_review"},
	})
	if coreErr == nil {
		t.Fatal("expected core path to reject malformed envelope")
	}

	_, riderErr := applyResultIngest(context.Background(), applyResultDeps{
		Transitioner: &fakeParityTransitioner{status: map[string]string{entityKey: "todo"}, outcomes: map[string]string{"pass": "in_review"}},
		Coordinator:  gatepersist.NewCoordinator(&fakeParityNoteWriter{}, fakeParityNoteReader{}, fakeParityHistoryReader{}, fakeParityStatusValidator{}, &fakeParityTransition{status: map[string]string{}}, &fakeParityTransition{status: map[string]string{}}, fakeParityLeaseReleaser{}),
		ProjectRoot:  t.TempDir(),
		RunID:        "run-parity1234567890abcdef1234567891",
		EntityType:   "task",
		EntityKey:    entityKey,
		SessionID:    "sess-rider",
		OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
		WorkflowSvc:  fakeTerminalStatusChecker{},
	}, malformed)
	if riderErr == nil {
		t.Fatal("expected rider path to reject malformed envelope")
	}
}

// TestApplyResultIngest_ConflictingReplayFailsClosedSameAsCore is the Rider
// side of the conflicting-replay fixture (internal/runner's
// TestIngestGateResult_ConflictingReplayFailsClosed): a second
// --apply-result call under the SAME run_id/entity as an already-persisted
// first call, but with a DIFFERENT envelope, must fail closed on both paths
// rather than silently accepting the newer content.
func TestApplyResultIngest_ConflictingReplayFailsClosedSameAsCore(t *testing.T) {
	entityKey := "E01-F01-001"
	runID := "run-conflict1234567890abcdef1234567891"
	outcomeRoles := map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}
	outcomes := map[string]string{"pass": "in_review"}
	conflicting := []byte(`{
		"kind": "final",
		"recommended_outcome": "pass",
		"evidence": [{"kind": "test_run", "pointer": "artifacts/test.log"}],
		"gate_result": {"schema_version": 1, "summary": "a DIFFERENT summary than the first call"}
	}`)

	newCoordinator := func() *gatepersist.Coordinator {
		transition := &fakeParityTransition{status: map[string]string{entityKey: "todo"}}
		return gatepersist.NewCoordinator(
			&fakeParityNoteWriter{}, fakeParityNoteReader{}, fakeParityHistoryReader{},
			fakeParityStatusValidator{}, transition, transition, fakeParityLeaseReleaser{},
		)
	}

	coreCoordinator := newCoordinator()
	coreProjectRoot := t.TempDir()
	if _, err := runner.IngestGateResult(context.Background(), runner.GateIngestRequest{
		EnvelopeBytes: []byte(parityEnvelope), Coordinator: coreCoordinator, ProjectRoot: coreProjectRoot,
		RunID: runID, EntityKey: entityKey, EntityType: models.EntityTypeTask,
		SourceStatus: "todo", Gate: "todo", Session: gatepersist.Session{ID: "sess-core"},
		OutcomeRoles: outcomeRoles, Outcomes: outcomes,
	}); err != nil {
		t.Fatalf("expected core path's first ingestion to succeed: %v", err)
	}
	if _, err := runner.IngestGateResult(context.Background(), runner.GateIngestRequest{
		EnvelopeBytes: conflicting, Coordinator: coreCoordinator, ProjectRoot: coreProjectRoot,
		RunID: runID, EntityKey: entityKey, EntityType: models.EntityTypeTask,
		SourceStatus: "todo", Gate: "todo", Session: gatepersist.Session{ID: "sess-core"},
		OutcomeRoles: outcomeRoles, Outcomes: outcomes,
	}); err == nil {
		t.Fatal("expected core path's conflicting replay to fail closed")
	}

	riderCoordinator := newCoordinator()
	riderProjectRoot := t.TempDir()
	if _, err := applyResultIngest(context.Background(), applyResultDeps{
		Transitioner: &fakeParityTransitioner{status: map[string]string{entityKey: "todo"}, outcomes: outcomes},
		Coordinator:  riderCoordinator, ProjectRoot: riderProjectRoot, RunID: runID,
		EntityType: "task", EntityKey: entityKey, SessionID: "sess-rider", OutcomeRoles: outcomeRoles,
		WorkflowSvc: fakeTerminalStatusChecker{},
	}, []byte(parityEnvelope)); err != nil {
		t.Fatalf("expected rider path's first ingestion to succeed: %v", err)
	}
	if _, err := applyResultIngest(context.Background(), applyResultDeps{
		Transitioner: &fakeParityTransitioner{status: map[string]string{entityKey: "in_review"}, outcomes: outcomes},
		Coordinator:  riderCoordinator, ProjectRoot: riderProjectRoot, RunID: runID,
		EntityType: "task", EntityKey: entityKey, SessionID: "sess-rider", OutcomeRoles: outcomeRoles,
		WorkflowSvc: fakeTerminalStatusChecker{},
	}, conflicting); err == nil {
		t.Fatal("expected rider path's conflicting replay to fail closed, same as core")
	}
}

// TestApplyResultIngest_ReleasesLeaseOnTerminalOutcome is the code-review
// round-7 Finding 2 regression guard (recorded as a note on
// T-E34-F05-004): applyResultIngest built runner.GateIngestRequest without
// ever setting RetirementConfirmed or RunConcluded, unlike
// internal/runner/controller.go's ingestGateResultForDispatch, which
// resolves IsTerminalStatus and re-ingests with both flags true before
// releasing the lease. Every `shark run --apply-result` invocation left the
// claim/lease held even on a terminal outcome, until TTL expiry. This
// asserts the lease IS released once the resolved target status is
// terminal.
func TestApplyResultIngest_ReleasesLeaseOnTerminalOutcome(t *testing.T) {
	entityKey := "E01-F01-001"
	outcomeRoles := map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}
	outcomes := map[string]string{"pass": "completed"}

	transition := &fakeParityTransition{status: map[string]string{entityKey: "in_review"}}
	releaser := &trackingLeaseReleaser{}
	coordinator := gatepersist.NewCoordinator(
		&fakeParityNoteWriter{}, fakeParityNoteReader{}, fakeParityHistoryReader{},
		fakeParityStatusValidator{}, transition, transition, releaser,
	)

	result, err := applyResultIngest(context.Background(), applyResultDeps{
		Transitioner: &fakeParityTransitioner{status: map[string]string{entityKey: "in_review"}, outcomes: outcomes},
		Coordinator:  coordinator,
		ProjectRoot:  t.TempDir(),
		RunID:        "run-terminal1234567890abcdef1234567891",
		EntityType:   "task",
		EntityKey:    entityKey,
		SessionID:    "sess-rider",
		OutcomeRoles: outcomeRoles,
		WorkflowSvc:  fakeTerminalStatusChecker{terminal: map[string]bool{"completed": true}},
	}, []byte(parityEnvelope))
	if err != nil {
		t.Fatalf("expected ingestion to succeed: %v", err)
	}
	if result.ToStatus != "completed" {
		t.Fatalf("expected ToStatus=completed, got %q", result.ToStatus)
	}
	if !releaser.released {
		t.Fatal("expected --apply-result to release the lease on a terminal outcome, but it was not released")
	}
}

// TestApplyResultIngest_DoesNotReleaseLeaseOnNonTerminalOutcome is the
// sibling of the above: a non-terminal target status must NOT release the
// lease (mirrors ingestGateResultForDispatch's own non-terminal guard).
func TestApplyResultIngest_DoesNotReleaseLeaseOnNonTerminalOutcome(t *testing.T) {
	entityKey := "E01-F01-001"
	outcomeRoles := map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}
	outcomes := map[string]string{"pass": "qa"}

	transition := &fakeParityTransition{status: map[string]string{entityKey: "code_review"}}
	releaser := &trackingLeaseReleaser{}
	coordinator := gatepersist.NewCoordinator(
		&fakeParityNoteWriter{}, fakeParityNoteReader{}, fakeParityHistoryReader{},
		fakeParityStatusValidator{}, transition, transition, releaser,
	)

	result, err := applyResultIngest(context.Background(), applyResultDeps{
		Transitioner: &fakeParityTransitioner{status: map[string]string{entityKey: "code_review"}, outcomes: outcomes},
		Coordinator:  coordinator,
		ProjectRoot:  t.TempDir(),
		RunID:        "run-nonterminal1234567890abcdef123456",
		EntityType:   "task",
		EntityKey:    entityKey,
		SessionID:    "sess-rider",
		OutcomeRoles: outcomeRoles,
		WorkflowSvc:  fakeTerminalStatusChecker{terminal: map[string]bool{"completed": true}},
	}, []byte(parityEnvelope))
	if err != nil {
		t.Fatalf("expected ingestion to succeed: %v", err)
	}
	if result.ToStatus != "qa" {
		t.Fatalf("expected ToStatus=qa, got %q", result.ToStatus)
	}
	if releaser.released {
		t.Fatal("expected --apply-result to leave the lease held on a non-terminal outcome, but it was released")
	}
}
