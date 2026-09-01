package runner

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// fakeNoteWriter/fakeNoteReader/fakeHistoryReader/fakeStatusValidator/
// fakeTransitioner/fakeLeaseReleaser are minimal in-memory implementations
// of gatepersist's injected interfaces, purpose-built for exercising
// IngestGateResult end-to-end without a database. gatepersist's own fakes
// (fakes_test.go) are unexported to its _test package and cannot be reused
// here.
type fakeNoteWriter struct {
	notes []fakeNote
}

type fakeNote struct {
	entityType models.EntityType
	entityKey  string
	noteType   string
	content    string
	metadata   string
}

func (f *fakeNoteWriter) AddNoteWithMetadata(_ context.Context, entityType models.EntityType, entityKey, noteType, content, _ string, metadata string) (*models.EntityNote, error) {
	f.notes = append(f.notes, fakeNote{entityType: entityType, entityKey: entityKey, noteType: noteType, content: content, metadata: metadata})
	return &models.EntityNote{}, nil
}

type fakeNoteReader struct{}

func (fakeNoteReader) ListNotes(context.Context, models.EntityType, string, []string) ([]*models.EntityNote, error) {
	return nil, nil
}

type fakeHistoryReader struct{}

func (fakeHistoryReader) GetHistory(context.Context, models.EntityType, string) ([]*models.EntityHistory, error) {
	return nil, nil
}

type fakeStatusValidator struct{ valid map[string]bool }

func (v fakeStatusValidator) IsValidStatus(_ models.EntityType, status string) bool {
	return v.valid[status]
}

type fakeTransitioner struct {
	status map[string]string
}

func (t *fakeTransitioner) Transition(_ context.Context, _ models.EntityType, entityKey, targetStatus, _, _ string) (string, bool, error) {
	from := t.status[entityKey]
	if t.status == nil {
		t.status = map[string]string{}
	}
	t.status[entityKey] = targetStatus
	return from, from != targetStatus, nil
}

func (t *fakeTransitioner) CurrentStatus(_ context.Context, _ models.EntityType, entityKey string) (string, error) {
	return t.status[entityKey], nil
}

type fakeLeaseReleaser struct{ released bool }

func (l *fakeLeaseReleaser) Release(context.Context, string, string, string, string, bool) (bool, error) {
	l.released = true
	return true, nil
}

func newFakeCoordinator(mainEntity string) *gatepersist.Coordinator {
	notes := &fakeNoteWriter{}
	transitioner := &fakeTransitioner{status: map[string]string{mainEntity: "todo"}}
	return gatepersist.NewCoordinator(
		notes,
		fakeNoteReader{},
		fakeHistoryReader{},
		fakeStatusValidator{valid: map[string]bool{"todo": true, "in_review": true}},
		transitioner,
		transitioner,
		&fakeLeaseReleaser{},
	)
}

func validGateEnvelope() []byte {
	return []byte(`{
		"kind": "final",
		"recommended_outcome": "pass",
		"evidence": [{"kind": "test_run", "pointer": "artifacts/test.log"}],
		"gate_result": {"schema_version": 1, "summary": "all checks passed"}
	}`)
}

func baseGateIngestRequest(t *testing.T, envelope []byte) GateIngestRequest {
	t.Helper()
	return GateIngestRequest{
		EnvelopeBytes: envelope,
		Coordinator:   newFakeCoordinator("E01-F01-001"),
		ProjectRoot:   t.TempDir(),
		RunID:         "run-1234567890abcdef1234567890abcdef",
		EntityKey:     "E01-F01-001",
		EntityType:    models.EntityTypeTask,
		SourceStatus:  "todo",
		Gate:          "code_review",
		Session:       gatepersist.Session{ID: "sess-1", Agent: "dev-agent"},
		OutcomeRoles:  map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
		Outcomes:      map[string]string{"pass": "in_review"},
	}
}

func TestIngestGateResult_ValidEnvelopePersistsAndTransitions(t *testing.T) {
	req := baseGateIngestRequest(t, validGateEnvelope())
	result, err := IngestGateResult(context.Background(), req)
	if err != nil {
		t.Fatalf("expected successful ingestion, got error: %v", err)
	}
	if result.OutcomeKey != "pass" {
		t.Fatalf("expected outcome key pass, got %q", result.OutcomeKey)
	}
	if result.Role != gateresult.RoleSuccess {
		t.Fatalf("expected role success, got %q", result.Role)
	}
	if !result.Transitioned || result.ToStatus != "in_review" {
		t.Fatalf("expected transition to in_review, got %+v", result.Result)
	}
}

func TestIngestGateResult_MalformedEnvelopeFailsClosed(t *testing.T) {
	req := baseGateIngestRequest(t, []byte(`{"kind": "final", "recommended_outcome":`))
	_, err := IngestGateResult(context.Background(), req)
	if err == nil {
		t.Fatal("expected malformed envelope to fail closed")
	}
}

// TestIngestGateResult_ConflictingReplayFailsClosed is one of the task
// spec's Testing Strategy fixtures: a second ingestion call under the SAME
// run_id/entity as an already-persisted first call, but with a DIFFERENT
// envelope, must fail closed rather than silently accepting the newer
// content or re-persisting over the durable record. This is
// gaterun.VerifyResumeIdentity's operation-digest check, reached through the
// full IngestGateResult boundary.
func TestIngestGateResult_ConflictingReplayFailsClosed(t *testing.T) {
	coordinator := newFakeCoordinator("E01-F01-001")
	projectRoot := t.TempDir()
	base := func(envelope []byte) GateIngestRequest {
		return GateIngestRequest{
			EnvelopeBytes: envelope,
			Coordinator:   coordinator,
			ProjectRoot:   projectRoot,
			RunID:         "run-conflict1234567890abcdef1234567890",
			EntityKey:     "E01-F01-001",
			EntityType:    models.EntityTypeTask,
			SourceStatus:  "todo",
			Gate:          "code_review",
			Session:       gatepersist.Session{ID: "sess-1", Agent: "dev-agent"},
			OutcomeRoles:  map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
			Outcomes:      map[string]string{"pass": "in_review"},
		}
	}

	first, err := IngestGateResult(context.Background(), base(validGateEnvelope()))
	if err != nil {
		t.Fatalf("expected the first ingestion to succeed, got: %v", err)
	}
	if !first.Transitioned {
		t.Fatal("expected the first ingestion to transition")
	}

	conflicting := []byte(`{
		"kind": "final",
		"recommended_outcome": "pass",
		"evidence": [{"kind": "test_run", "pointer": "artifacts/test.log"}],
		"gate_result": {"schema_version": 1, "summary": "a DIFFERENT summary than the first call"}
	}`)
	if _, err := IngestGateResult(context.Background(), base(conflicting)); err == nil {
		t.Fatal("expected a conflicting replay (same run_id/entity, different envelope) to fail closed")
	}
}

func TestIngestGateResult_WrongKindFailsClosed(t *testing.T) {
	req := baseGateIngestRequest(t, []byte(`{"kind": "failed", "evidence": []}`))
	_, err := IngestGateResult(context.Background(), req)
	if err == nil {
		t.Fatal("expected a non-final kind to fail closed")
	}
}

func TestIngestGateResult_AbsentGateResultFailsClosed(t *testing.T) {
	req := baseGateIngestRequest(t, []byte(`{"kind": "final", "recommended_outcome": "pass", "evidence": []}`))
	_, err := IngestGateResult(context.Background(), req)
	if err == nil {
		t.Fatal("expected an absent gate_result payload to fail closed")
	}
}

func TestIngestGateResult_MalformedGateResultFailsClosed(t *testing.T) {
	req := baseGateIngestRequest(t, []byte(`{
		"kind": "final",
		"recommended_outcome": "pass",
		"evidence": [],
		"gate_result": {"schema_version": 2, "summary": "bad version"}
	}`))
	_, err := IngestGateResult(context.Background(), req)
	if err == nil {
		t.Fatal("expected a malformed nested gate_result to fail closed")
	}
}

func TestIngestGateResult_UnknownOutcomeRoleFailsClosed(t *testing.T) {
	req := baseGateIngestRequest(t, validGateEnvelope())
	req.OutcomeRoles = map[string]gateresult.OutcomeRole{} // no role configured for "pass"
	_, err := IngestGateResult(context.Background(), req)
	if err == nil {
		t.Fatal("expected an unconfigured outcome role to fail closed")
	}
}

func TestIngestGateResult_RoleViolationFailsClosed(t *testing.T) {
	// A "success" role result must contain no kickback; this one does.
	envelope := []byte(`{
		"kind": "final",
		"recommended_outcome": "pass",
		"evidence": [],
		"gate_result": {
			"schema_version": 1,
			"summary": "has a kickback but role is success",
			"kickbacks": [{"entity_key": "E01-F01-002", "target_status": "todo", "reason": "rework needed"}]
		}
	}`)
	req := baseGateIngestRequest(t, envelope)
	_, err := IngestGateResult(context.Background(), req)
	if err == nil {
		t.Fatal("expected a role-violating gate_result to fail closed")
	}
}

func TestIngestGateResult_UnconfiguredTargetStatusFailsClosed(t *testing.T) {
	req := baseGateIngestRequest(t, validGateEnvelope())
	req.Outcomes = map[string]string{} // no target status configured for "pass"
	_, err := IngestGateResult(context.Background(), req)
	if err == nil {
		t.Fatal("expected an unconfigured target status to fail closed")
	}
}

func TestIngestGateResult_NilCoordinatorFailsClosed(t *testing.T) {
	req := baseGateIngestRequest(t, validGateEnvelope())
	req.Coordinator = nil
	_, err := IngestGateResult(context.Background(), req)
	if err == nil {
		t.Fatal("expected a nil coordinator to fail closed")
	}
}
