// This file covers the T-E34-F05-004 rework round-4 UAT finding (Finding 2,
// HIGH): run_apply_result.go's runApplyResult used an unconditional
// os.ReadFile on --apply-result BEFORE workercontrol.Decode's own
// MaxEnvelopeBytes bound was ever evaluated -- so a maliciously oversized
// --apply-result file was still fully buffered into memory first, defeating
// the point of that bound (REQ-NF-001/AC2 "must reject oversized content").
// readBoundedEnvelopeFile is the fix: it bounds the read itself at
// MaxEnvelopeBytes+1 bytes via io.LimitReader, mirroring the same pattern
// internal/gaterun's own readRegularBounded already uses for the sidecar
// transport's file reads (fsio.go) -- this is the CLI-supplied-path mirror
// of that pattern, not a second bound.
package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

func writeFileOfSize(t *testing.T, size int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "envelope.json")
	data := make([]byte, size)
	for i := range data {
		data[i] = 'a'
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	return path
}

// TestReadBoundedEnvelopeFile_RejectsOversizedFileAtReadBoundary is the
// direct regression test: a file one byte over MaxEnvelopeBytes must be
// rejected by the bounded reader itself, with a message naming the size
// bound -- not left to workercontrol.Decode to reject after the fact.
func TestReadBoundedEnvelopeFile_RejectsOversizedFileAtReadBoundary(t *testing.T) {
	path := writeFileOfSize(t, workercontrol.MaxEnvelopeBytes+1)

	_, err := readBoundedEnvelopeFile(path)
	if err == nil {
		t.Fatal("readBoundedEnvelopeFile with an oversized file: want error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds the maximum envelope size") {
		t.Fatalf("error = %v, want a message naming the size bound", err)
	}
}

// TestReadBoundedEnvelopeFile_AcceptsFileAtExactLimit proves the bound is
// MaxEnvelopeBytes (inclusive), not off-by-one in the other direction: a
// file exactly at the limit must be read in full.
func TestReadBoundedEnvelopeFile_AcceptsFileAtExactLimit(t *testing.T) {
	path := writeFileOfSize(t, workercontrol.MaxEnvelopeBytes)

	got, err := readBoundedEnvelopeFile(path)
	if err != nil {
		t.Fatalf("readBoundedEnvelopeFile at exactly the limit: %v", err)
	}
	if len(got) != workercontrol.MaxEnvelopeBytes {
		t.Fatalf("read %d bytes, want %d", len(got), workercontrol.MaxEnvelopeBytes)
	}
}

// TestReadBoundedEnvelopeFile_SmallFileRoundTrips is the ordinary-path
// sanity check: a small, well-formed file's exact bytes come back
// unmodified.
func TestReadBoundedEnvelopeFile_SmallFileRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "envelope.json")
	want := []byte(`{"kind": "final", "recommended_outcome": "pass", "gate_result": {"schema_version": 1, "summary": "ok"}}`)
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	got, err := readBoundedEnvelopeFile(path)
	if err != nil {
		t.Fatalf("readBoundedEnvelopeFile: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestRunApplyResult_OversizedEnvelopeFileRejectedAfterAuthorization is the
// end-to-end regression test through runApplyResult itself: authorization
// (verifyClaimSession) passes first -- proving this is genuinely the
// bounded-read fix, not a byproduct of the authorization gate rejecting the
// call -- and the oversized file is then rejected by readBoundedEnvelopeFile
// before cli.FindProjectRoot/buildTransitioner/buildGateCoordinator (all of
// which require a real database and are therefore never reached in this
// mock-only CLI test).
func TestRunApplyResult_OversizedEnvelopeFileRejectedAfterAuthorization(t *testing.T) {
	origRunID, origPath, origSession := runApplyRunID, runApplyResultPath, runSession
	t.Cleanup(func() {
		runApplyRunID = origRunID
		runApplyResultPath = origPath
		runSession = origSession
	})

	runApplyRunID = "run-bounded-read-1"
	runApplyResultPath = writeFileOfSize(t, workercontrol.MaxEnvelopeBytes+1)
	runSession = "the-real-owning-session"

	withRunClaimSvcOverride(t, &mockRunClaimService{
		getClaim: &models.EntityClaim{
			EntityType:    "task",
			EntityKey:     "E01-F01-001",
			SessionID:     "the-real-owning-session",
			LastHeartbeat: time.Now().UTC(),
		},
	})

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runApplyResult(cmd, "task", "E01-F01-001")
	if err == nil {
		t.Fatal("runApplyResult with an oversized --apply-result file: want error, got nil")
	}
	if strings.Contains(err.Error(), "authorization failed") {
		t.Fatalf("error = %v, want the bounded-read rejection, not an authorization failure (authorization was configured to pass)", err)
	}
	if !strings.Contains(err.Error(), "exceeds the maximum envelope size") {
		t.Fatalf("error = %v, want a message naming the size bound", err)
	}
}
