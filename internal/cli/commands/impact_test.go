package commands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// mockImpactNoteWriter is a gatepersist.NoteWriter test double recording
// every call, so tests can assert both "was it written" and "with what
// content/metadata" without a real database (CLI-tests golden rule).
type mockImpactNoteWriter struct {
	calls []mockImpactNoteWriterCall
	err   error
}

type mockImpactNoteWriterCall struct {
	entityType models.EntityType
	entityKey  string
	noteType   string
	content    string
	metadata   string
}

func (m *mockImpactNoteWriter) AddNoteWithMetadata(_ context.Context, entityType models.EntityType, entityKey, noteType, content, _, metadata string) (*models.EntityNote, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.calls = append(m.calls, mockImpactNoteWriterCall{entityType, entityKey, noteType, content, metadata})
	return &models.EntityNote{ID: int64(len(m.calls))}, nil
}

func writeImpactFile(t *testing.T, body map[string]interface{}) string {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal impact fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "impact.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write impact fixture: %v", err)
	}
	return path
}

func resetImpactFlags() {
	impactSourceKind = ""
	impactSourceKey = ""
	impactSourcePointer = ""
	impactFile = ""
}

func validImpactBody() map[string]interface{} {
	return map[string]interface{}{
		"change_summary": "Bounded behavioral change from ADR-0007",
		"status":         "accounted",
	}
}

func TestRunImpactRecord_ValidatesAndPersists(t *testing.T) {
	defer resetImpactFlags()
	mock := &mockImpactNoteWriter{}
	impactNoteWriterOverride = mock
	defer func() { impactNoteWriterOverride = nil }()

	impactSourceKind = "adr"
	impactSourceKey = "ADR-0007"
	impactSourcePointer = "docs/adr/0007-something.md"
	impactFile = writeImpactFile(t, validImpactBody())

	cmd := newChangeCmdWithCtx()
	if err := runImpactRecord(cmd, []string{"E01-F01-001"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.calls) != 1 {
		t.Fatalf("expected exactly one note write, got %d", len(mock.calls))
	}
	call := mock.calls[0]
	if call.entityType != models.EntityTypeTask {
		t.Errorf("expected entity_type=task for E01-F01-001, got %q", call.entityType)
	}
	if call.entityKey != "E01-F01-001" {
		t.Errorf("expected entity_key=E01-F01-001, got %q", call.entityKey)
	}
	if call.noteType != noteTypeReferenceImpact {
		t.Errorf("expected note_type=%q, got %q", noteTypeReferenceImpact, call.noteType)
	}

	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(call.metadata), &meta); err != nil {
		t.Fatalf("failed to parse persisted metadata: %v", err)
	}
	if meta["record_kind"] != recordKindChangeImpact {
		t.Errorf("expected record_kind=%q, got %v", recordKindChangeImpact, meta["record_kind"])
	}
	if meta["source_kind"] != "adr" || meta["source_key"] != "ADR-0007" {
		t.Errorf("unexpected source identity in metadata: %v", meta)
	}
}

func TestRunImpactRecord_FillsIdentityFromFlagsWhenFileOmitsThem(t *testing.T) {
	defer resetImpactFlags()
	mock := &mockImpactNoteWriter{}
	impactNoteWriterOverride = mock
	defer func() { impactNoteWriterOverride = nil }()

	impactSourceKind = "adr"
	impactSourceKey = "ADR-0009"
	impactSourcePointer = "docs/adr/0009-something.md"
	// The file itself has no source_kind/source_key/source_pointer fields —
	// they must be filled from the flags, not rejected as missing.
	impactFile = writeImpactFile(t, validImpactBody())

	cmd := newChangeCmdWithCtx()
	if err := runImpactRecord(cmd, []string{"E01-F01-001"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected exactly one note write, got %d", len(mock.calls))
	}
}

func TestRunImpactRecord_RejectsConflictingIdentity(t *testing.T) {
	defer resetImpactFlags()
	mock := &mockImpactNoteWriter{}
	impactNoteWriterOverride = mock
	defer func() { impactNoteWriterOverride = nil }()

	impactSourceKind = "adr"
	impactSourceKey = "ADR-0001" // flag asserts ADR-0001
	impactSourcePointer = "docs/adr/0001-something.md"

	body := validImpactBody()
	body["source_kind"] = "adr"
	body["source_key"] = "ADR-9999" // file claims a DIFFERENT ADR
	body["source_pointer"] = "docs/adr/0001-something.md"
	impactFile = writeImpactFile(t, body)

	cmd := newChangeCmdWithCtx()
	err := runImpactRecord(cmd, []string{"E01-F01-001"})
	if err == nil {
		t.Fatal("expected a conflict error, got nil")
	}
	if len(mock.calls) != 0 {
		t.Fatalf("expected no note write on a rejected conflict, got %d", len(mock.calls))
	}
}

func TestRunImpactRecord_RejectsInvalidChangeImpactSet(t *testing.T) {
	defer resetImpactFlags()
	mock := &mockImpactNoteWriter{}
	impactNoteWriterOverride = mock
	defer func() { impactNoteWriterOverride = nil }()

	impactSourceKind = "adr"
	impactSourceKey = "ADR-0007"
	impactSourcePointer = "docs/adr/0007-something.md"
	// Missing change_summary and status — must fail gateresult validation.
	impactFile = writeImpactFile(t, map[string]interface{}{})

	cmd := newChangeCmdWithCtx()
	err := runImpactRecord(cmd, []string{"E01-F01-001"})
	if err == nil {
		t.Fatal("expected a validation error for an incomplete ChangeImpactSet")
	}
	if len(mock.calls) != 0 {
		t.Fatalf("expected no note write on a validation failure, got %d", len(mock.calls))
	}
}

func TestRunImpactRecord_UnknownEntityKeyFailsClosed(t *testing.T) {
	defer resetImpactFlags()
	mock := &mockImpactNoteWriter{}
	impactNoteWriterOverride = mock
	defer func() { impactNoteWriterOverride = nil }()

	impactSourceKind = "adr"
	impactSourceKey = "ADR-0007"
	impactSourcePointer = "docs/adr/0007-something.md"
	impactFile = writeImpactFile(t, validImpactBody())

	cmd := newChangeCmdWithCtx()
	err := runImpactRecord(cmd, []string{"not-a-real-key-format-!!!"})
	if err == nil {
		t.Fatal("expected an error for an undetectable entity key")
	}
	if len(mock.calls) != 0 {
		t.Fatalf("expected no note write for an undetectable entity key, got %d", len(mock.calls))
	}
}
