package gaterun

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestReadBoundedRegularFile_RejectsSymlinkTarget guards
// ReadBoundedRegularFile — the bare-path counterpart of readRegularBounded
// used by CLI-flag-supplied files (--apply-result, --impact-file) — against
// transparently following a symlink to a file outside the caller's intended
// scope.
func TestReadBoundedRegularFile_RejectsSymlinkTarget(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret.json")
	if err := os.WriteFile(outside, []byte(`{"secret":true}`), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(t.TempDir(), "envelope.json")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if _, err := ReadBoundedRegularFile(link, 1024); err == nil {
		t.Fatal("ReadBoundedRegularFile over a symlinked target: want error, got nil")
	} else {
		var unsafeErr *UnsafePathError
		if !errors.As(err, &unsafeErr) {
			t.Errorf("ReadBoundedRegularFile over a symlinked target error = %v, want *UnsafePathError", err)
		}
	}
}

// TestReadBoundedRegularFile_OrdinaryFileRoundTrips is the sanity check that
// a well-formed regular file is read back byte-for-byte within the bound.
func TestReadBoundedRegularFile_OrdinaryFileRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "envelope.json")
	want := []byte(`{"a":1}`)
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	got, err := ReadBoundedRegularFile(path, 1024)
	if err != nil {
		t.Fatalf("ReadBoundedRegularFile: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestReadBoundedRegularFile_RejectsOversized proves the maxBytes bound is
// enforced by ReadBoundedRegularFile itself.
func TestReadBoundedRegularFile_RejectsOversized(t *testing.T) {
	path := filepath.Join(t.TempDir(), "envelope.json")
	if err := os.WriteFile(path, make([]byte, 11), 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	if _, err := ReadBoundedRegularFile(path, 10); err == nil {
		t.Fatal("ReadBoundedRegularFile with an oversized file: want error, got nil")
	}
}
