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
	status   map[string]string
	outcomes map[string]string
}

func (t *fakeParityTransitioner) GetNextStatus(_ context.Context, key string) (*services.NextStatusInfo, error) {
	return &services.NextStatusInfo{CurrentStatus: t.status[key], Outcomes: t.outcomes}, nil
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

func (t *fakeParityTransition) Transition(_ context.Context, _ models.EntityType, entityKey, targetStatus, _, _ string) (string, bool, error) {
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
	}, malformed)
	if riderErr == nil {
		t.Fatal("expected rider path to reject malformed envelope")
	}
}
