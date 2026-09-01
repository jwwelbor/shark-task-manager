package gaterun

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// resultFileName and operationStateFileName are the two sidecar files this
// package owns under a run directory.
const (
	resultFileName         = "result.json"
	operationStateFileName = "operation-state.json"
)

// readRegularBounded reads path after verifying, via a no-follow open plus
// an fstat on the resulting descriptor (see openRegularNoFollow), that it is
// a real (non-symlink) regular file, bounding the read at maxBytes+1 so an
// oversized target is rejected rather than exhausting memory. Because the
// safety check and the read share one already-open descriptor, there is no
// TOCTOU window between verifying the target and reading it.
func readRegularBounded(path string, maxBytes int) ([]byte, error) {
	f, err := openRegularNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("gaterun: read %s: %w", path, err)
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("gaterun: %s exceeds the %d byte bound", path, maxBytes)
	}
	return data, nil
}

// rejectUnsafeExistingTarget verifies that, if path already exists, it is a
// real regular file (not a symlink, directory, FIFO, socket, or device). A
// non-existent path is not an error here — the caller is about to create it.
func rejectUnsafeExistingTarget(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("gaterun: stat %s: %w", path, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return &UnsafePathError{Path: path, Reason: "refusing to follow symlink"}
	}
	if !fi.Mode().IsRegular() {
		return &UnsafePathError{Path: path, Reason: "existing target is not a regular file"}
	}
	return nil
}

// fsyncDir opens dir and fsyncs it, which is what makes a preceding rename
// or file-create within dir durable across a crash. Called after every
// sidecar write in this package.
func fsyncDir(dir string) error {
	d, err := os.Open(dir) // #nosec G304 -- dir is the validated, Lstat-verified run directory.
	if err != nil {
		return fmt.Errorf("gaterun: open run dir %s for fsync: %w", dir, err)
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return fmt.Errorf("gaterun: fsync run dir %s: %w", dir, err)
	}
	return nil
}

// CreateResult atomically creates dir/result.json exactly once, no-replace:
//
//   - If the file does not yet exist, it is created with O_EXCL (mode 0600,
//     chmod'd explicitly to defeat umask), the data is written, the file is
//     fsync'd, and the run directory is fsync'd to make the create durable.
//     Returns created=true.
//   - If the file already exists as a real regular file with byte-identical
//     content, this is idempotent success. Returns created=false, err=nil.
//   - If it exists with different content, a *ConflictError is returned —
//     the accepted bytes are never overwritten.
//   - If it exists as a symlink or non-regular target, an *UnsafePathError
//     is returned.
//
// CreateResult uses a write-complete-temp-file-then-hardlink protocol rather
// than a direct O_CREATE|O_EXCL on the final path: a plain O_EXCL create
// would make the (initially empty) destination inode visible to concurrent
// readers *before* its content is written, so a racing reader could observe
// a partial file and misreport a still-in-flight identical write as
// conflicting. Writing the complete, fsync'd content to a same-directory
// temp file first, then atomically hard-linking it into place, guarantees
// that whenever the destination becomes visible under its final name it is
// already fully written — no reader can ever observe a partial result.json.
// os.Link fails with an existing-target error precisely when another writer
// already won, giving the same exclusive first-writer-wins guarantee
// O_CREATE|O_EXCL provides for file creation, across both goroutines and
// processes sharing the run directory.
func CreateResult(dir string, data []byte) (created bool, err error) {
	if len(data) == 0 {
		return false, fmt.Errorf("gaterun: result data must not be empty")
	}
	if len(data) > maxSidecarBytes {
		return false, fmt.Errorf("gaterun: result data exceeds the %d byte bound", maxSidecarBytes)
	}

	path := filepath.Join(dir, resultFileName)

	tmp, err := os.CreateTemp(dir, ".result-*.tmp") // #nosec G304 -- dir is validated by RunDir.
	if err != nil {
		return false, fmt.Errorf("gaterun: create temp result file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // always cleaned up: linked content keeps its own inode.

	if werr := writeSyncClose(tmp, data); werr != nil {
		return false, werr
	}
	if cherr := os.Chmod(tmpPath, 0o600); cherr != nil {
		return false, fmt.Errorf("gaterun: chmod temp result file: %w", cherr)
	}

	linkErr := os.Link(tmpPath, path)
	if linkErr == nil {
		if dErr := fsyncDir(dir); dErr != nil {
			return false, dErr
		}
		return true, nil
	}
	if !errors.Is(linkErr, fs.ErrExist) {
		return false, fmt.Errorf("gaterun: link %s into place: %w", path, linkErr)
	}

	// Another writer already won the race (or result.json pre-dates this
	// call). Its content is guaranteed complete by construction — verify it
	// is safe, then decide idempotent-success vs conflict.
	existing, readErr := readRegularBounded(path, maxSidecarBytes)
	if readErr != nil {
		return false, readErr
	}
	if bytes.Equal(existing, data) {
		return false, nil
	}
	return false, &ConflictError{Path: path}
}

func writeSyncClose(f *os.File, data []byte) (err error) {
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("gaterun: close %s: %w", f.Name(), cerr)
		}
	}()
	if _, werr := f.Write(data); werr != nil {
		return fmt.Errorf("gaterun: write %s: %w", f.Name(), werr)
	}
	if serr := f.Sync(); serr != nil {
		return fmt.Errorf("gaterun: fsync %s: %w", f.Name(), serr)
	}
	return nil
}

// ReadResult reads and returns the bytes of dir/result.json, or (nil, false,
// nil) if it does not exist yet. A single stat (inside readRegularBounded)
// is the sole existence/safety check — no separate pre-stat — so a
// not-exist, a symlink, and a non-regular target each report through
// exactly one error taxonomy regardless of which of the two calls in this
// package's flow first observes the path.
func ReadResult(dir string) (data []byte, exists bool, err error) {
	path := filepath.Join(dir, resultFileName)
	data, err = readRegularBounded(path, maxSidecarBytes)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

// WriteOperationState atomically replaces dir/operation-state.json via a
// same-directory temp-file write, fsync, rename, and directory fsync. Unlike
// CreateResult, this file may be replaced any number of times — it is the
// mutable replay journal, not the immutable accepted result.
func WriteOperationState(dir string, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("gaterun: operation state data must not be empty")
	}
	if len(data) > maxSidecarBytes {
		return fmt.Errorf("gaterun: operation state data exceeds the %d byte bound", maxSidecarBytes)
	}

	path := filepath.Join(dir, operationStateFileName)
	if err := rejectUnsafeExistingTarget(path); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".operation-state-*.tmp") // #nosec G304 -- dir is validated by RunDir.
	if err != nil {
		return fmt.Errorf("gaterun: create temp operation-state file: %w", err)
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	if werr := writeSyncClose(tmp, data); werr != nil {
		return werr
	}
	if cherr := os.Chmod(tmpPath, 0o600); cherr != nil {
		return fmt.Errorf("gaterun: chmod temp operation-state file: %w", cherr)
	}
	if rerr := os.Rename(tmpPath, path); rerr != nil {
		return fmt.Errorf("gaterun: rename operation-state file into place: %w", rerr)
	}
	committed = true
	if derr := fsyncDir(dir); derr != nil {
		return derr
	}
	return nil
}

// ReadOperationState reads and returns the bytes of dir/operation-state.json,
// or (nil, false, nil) if it does not exist yet (e.g. before the first
// operation-state write in a fresh run).
func ReadOperationState(dir string) (data []byte, exists bool, err error) {
	path := filepath.Join(dir, operationStateFileName)
	data, err = readRegularBounded(path, maxSidecarBytes)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}
