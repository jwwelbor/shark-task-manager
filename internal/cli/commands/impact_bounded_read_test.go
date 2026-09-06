// This file covers code-review round-6's finding: impact.go's
// runImpactRecord used an unconditional os.ReadFile on --impact-file BEFORE
// gateresult.ValidateChangeImpactSet's field-level bounds were ever
// evaluated -- so a maliciously oversized --impact-file was still fully
// buffered into memory first. readBoundedImpactFile is the fix: it bounds
// the read itself via gaterun.ReadBoundedRegularFile at
// workercontrol.MaxEnvelopeBytes+1 bytes, mirroring
// run_apply_result.go's readBoundedEnvelopeFile (run_apply_result_bounded_read_test.go)
// -- the sibling gap this rework closes, not a second implementation.
package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

func writeImpactFileOfSize(t *testing.T, size int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "impact.json")
	data := make([]byte, size)
	for i := range data {
		data[i] = 'a'
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	return path
}

// TestReadBoundedImpactFile_RejectsOversizedFileAtReadBoundary is the direct
// regression test: a file one byte over MaxEnvelopeBytes must be rejected by
// the bounded reader itself, with a message naming the size bound.
func TestReadBoundedImpactFile_RejectsOversizedFileAtReadBoundary(t *testing.T) {
	path := writeImpactFileOfSize(t, workercontrol.MaxEnvelopeBytes+1)

	_, err := readBoundedImpactFile(path)
	if err == nil {
		t.Fatal("readBoundedImpactFile with an oversized file: want error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds the maximum size") {
		t.Fatalf("error = %v, want a message naming the size bound", err)
	}
}

// TestReadBoundedImpactFile_AcceptsFileAtExactLimit proves the bound is
// MaxEnvelopeBytes (inclusive), not off-by-one in the other direction.
func TestReadBoundedImpactFile_AcceptsFileAtExactLimit(t *testing.T) {
	path := writeImpactFileOfSize(t, workercontrol.MaxEnvelopeBytes)

	got, err := readBoundedImpactFile(path)
	if err != nil {
		t.Fatalf("readBoundedImpactFile at exactly the limit: %v", err)
	}
	if len(got) != workercontrol.MaxEnvelopeBytes {
		t.Fatalf("read %d bytes, want %d", len(got), workercontrol.MaxEnvelopeBytes)
	}
}

// TestReadBoundedImpactFile_SmallFileRoundTrips is the ordinary-path sanity
// check: a small, well-formed file's exact bytes come back unmodified.
func TestReadBoundedImpactFile_SmallFileRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "impact.json")
	want := []byte(`{"change_summary": "ok", "status": "accounted"}`)
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	got, err := readBoundedImpactFile(path)
	if err != nil {
		t.Fatalf("readBoundedImpactFile: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestRunImpactRecord_OversizedImpactFileRejected is the end-to-end
// regression test through runImpactRecord itself: the oversized file is
// rejected by readBoundedImpactFile before json.Unmarshal or
// gateresult.ValidateChangeImpactSet ever see it, and well before any note
// is persisted.
func TestRunImpactRecord_OversizedImpactFileRejected(t *testing.T) {
	defer resetImpactFlags()
	mock := &mockImpactNoteWriter{}
	impactNoteWriterOverride = mock
	defer func() { impactNoteWriterOverride = nil }()

	impactSourceKind = "adr"
	impactSourceKey = "ADR-0007"
	impactSourcePointer = "docs/adr/0007-something.md"
	impactFile = writeImpactFileOfSize(t, workercontrol.MaxEnvelopeBytes+1)

	cmd := newChangeCmdWithCtx()
	err := runImpactRecord(cmd, []string{"E01-F01-001"})
	if err == nil {
		t.Fatal("runImpactRecord with an oversized --impact-file: want error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds the maximum size") {
		t.Fatalf("error = %v, want a message naming the size bound", err)
	}
	if len(mock.calls) != 0 {
		t.Fatalf("expected no note to be persisted, got %d calls", len(mock.calls))
	}
}
