package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

type impactNoteWriterStub struct {
	calls int
	meta  string
	err   error
}

func (s *impactNoteWriterStub) AddNoteWithMetadata(_ context.Context, _ models.EntityType, _ string, _ string, _ string, _ string, metadata string) (*models.EntityNote, error) {
	s.calls++
	s.meta = metadata
	if s.err != nil {
		return nil, s.err
	}
	return &models.EntityNote{ID: 42}, nil
}

func TestImpactServiceRecord_PropagatesWriterFailure(t *testing.T) {
	service, _ := NewImpactService(&impactNoteWriterStub{err: errors.New("database unavailable")})
	record, err := service.Record(context.Background(), validImpactInput())
	if err == nil || !strings.Contains(err.Error(), "persist change-impact note") {
		t.Fatalf("Record error = %v, want contextual writer failure", err)
	}
	if record != nil {
		t.Fatalf("Record returned a successful record after writer failure: %+v", record)
	}
}

func TestReconcileAndValidateImpact_RejectsNil(t *testing.T) {
	if err := ReconcileAndValidateImpact(nil, "adr", "ADR-0007", "docs/adr/0007.md"); err == nil {
		t.Fatal("ReconcileAndValidateImpact accepted nil impact")
	}
}

func validImpactInput() RecordImpactInput {
	return RecordImpactInput{
		EntityType: models.EntityTypeTask, EntityKey: "E01-F01-001",
		SourceKind: "adr", SourceKey: "ADR-0007", SourcePointer: "docs/adr/0007.md",
		Impact: gateresult.ChangeImpactSet{ChangeSummary: "A bounded change", Status: "accounted"},
	}
}

func TestImpactServiceRecord_ReconcilesValidatesAndPersists(t *testing.T) {
	writer := &impactNoteWriterStub{}
	service, err := NewImpactService(writer)
	if err != nil {
		t.Fatalf("NewImpactService: %v", err)
	}
	record, err := service.Record(context.Background(), validImpactInput())
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if writer.calls != 1 || record.NoteID != 42 || record.SourceKey != "ADR-0007" {
		t.Fatalf("unexpected record result: calls=%d record=%+v", writer.calls, record)
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(writer.meta), &metadata); err != nil {
		t.Fatalf("decode persisted metadata: %v", err)
	}
	if metadata["record_kind"] != impactRecordKind || metadata["source_pointer"] != "docs/adr/0007.md" {
		t.Fatalf("unexpected metadata: %v", metadata)
	}
}

func TestImpactServiceRecord_RejectsParentIdentityConflict(t *testing.T) {
	writer := &impactNoteWriterStub{}
	service, _ := NewImpactService(writer)
	input := validImpactInput()
	input.Impact.SourceKey = "ADR-9999"
	if _, err := service.Record(context.Background(), input); err == nil {
		t.Fatal("Record accepted a conflicting source key")
	}
	if writer.calls != 0 {
		t.Fatalf("writer called %d times after conflict", writer.calls)
	}
}
