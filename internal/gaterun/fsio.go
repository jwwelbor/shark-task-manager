package gaterun

import (
	"bytes"
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

// resultFileName, operationStateFileName, and identityFileName are the
// sidecar files this package owns under a run directory.
const (
	resultFileName         = "result.json"
	operationStateFileName = "operation-state.json"
	// identityFileName holds the create-once RunIdentity binding (identity.go,
	// UAT-3-1 fix): it is created before result.json ever becomes durable, so
	// any run directory that has a result.json also, by construction, already
	// has its owning entity/run identity durably and immutably bound.
	identityFileName = "identity.json"
)

// Every exported function below re-derives a fresh no-follow-verified run
// directory handle via openRunDirNoFollow and performs all of its file
// operations relative to that one handle (openRegularNoFollowAt,
// createExclAt, linkAt, renameAt, removeAt, existingTargetKindAt — see
// fsio_nofollow_unix.go / fsio_nofollow_windows.go). No operation in this
// file ever stats or checks a path and then separately reopens or
// reconstructs that same path by string for the actual read/write/rename: on
// Unix, every step from the ancestor directories down through the leaf
// operation shares one openat-derived descriptor chain, closing the
// ancestor-directory TOCTOU a plain Lstat-then-path-reopen would leave open.
//
// Precondition: dir must be a value returned by RunDir (or an equal string
// derived from the same projectRoot/runID pair) — i.e.
// <projectRoot>/.shark/runs/<runID>. openRunDirNoFollow re-derives the
// ancestor chain from dir's own path components (see dirhandle_unix.go /
// dirhandle_windows.go), so a dir whose last three components are not
// exactly ".shark", "runs", and a ValidateRunID-accepted run ID is rejected
// before any file operation is attempted, rather than silently operating
// against an unrecognized location. Both of this package's real callers
// (internal/gatepersist's Request.RunDir and internal/runner's ingest path)
// obtain dir exclusively via RunDir, so this precondition holds in
// production; it is new relative to the pre-rework code, which accepted any
// existing directory here.

// readRegularBounded reads name (relative to dh) after verifying, via a
// no-follow open plus an fstat on the resulting descriptor, that it is a
// real (non-symlink) regular file, bounding the read at maxBytes+1 so an
// oversized target is rejected rather than exhausting memory. Because the
// safety check and the read share one already-open descriptor, there is no
// TOCTOU window between verifying the target and reading it.
func readRegularBounded(dh *os.File, name string, maxBytes int) ([]byte, error) {
	f, err := openRegularNoFollowAt(dh, name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("gaterun: read %s: %w", name, err)
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("gaterun: %s exceeds the %d byte bound", name, maxBytes)
	}
	return data, nil
}

// rejectUnsafeExistingTarget verifies that, if name (relative to dh) already
// exists, it is a real regular file (not a symlink, directory, FIFO,
// socket, or device). A non-existent target is not an error here — the
// caller is about to create it.
func rejectUnsafeExistingTarget(dh *os.File, name string) error {
	exists, isSymlink, isRegular, err := existingTargetKindAt(dh, name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if isSymlink {
		return &UnsafePathError{Path: name, Reason: "refusing to follow symlink"}
	}
	if !isRegular {
		return &UnsafePathError{Path: name, Reason: "existing target is not a regular file"}
	}
	return nil
}

// createTempAt creates a new, exclusively-owned temp file relative to dh
// with the given prefix/suffix and a random hex name component, mirroring
// os.CreateTemp's collision-retry behavior but relative to dh instead of a
// path.
func createTempAt(dh *os.File, prefix, suffix string) (name string, f *os.File, err error) {
	var buf [12]byte
	for attempt := 0; attempt < 10_000; attempt++ {
		if _, rerr := crand.Read(buf[:]); rerr != nil {
			return "", nil, fmt.Errorf("gaterun: generate temp file name: %w", rerr)
		}
		candidate := prefix + hex.EncodeToString(buf[:]) + suffix
		f, err = createExclAt(dh, candidate, 0o600)
		if err == nil {
			return candidate, f, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", nil, err
		}
	}
	return "", nil, fmt.Errorf("gaterun: failed to create a unique temp file after repeated collisions")
}

// fsyncDir fsyncs dh, which is what makes a preceding rename or file-create
// within it durable across a crash. Called after every sidecar write in
// this package.
func fsyncDir(dh *os.File) error {
	if err := dh.Sync(); err != nil {
		return fmt.Errorf("gaterun: fsync run dir %s: %w", dh.Name(), err)
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
// The link fails with an existing-target error precisely when another
// writer already won, giving the same exclusive first-writer-wins guarantee
// O_CREATE|O_EXCL provides for file creation, across both goroutines and
// processes sharing the run directory.
func CreateResult(dir string, data []byte) (created bool, err error) {
	if len(data) == 0 {
		return false, fmt.Errorf("gaterun: result data must not be empty")
	}
	return createOnceSidecar(dir, resultFileName, ".result-", data)
}

// createOnceSidecar implements the shared create-once, no-replace,
// first-writer-wins protocol every immutable sidecar file in this package
// uses (result.json via CreateResult, identity.json via CreateIdentity):
// see CreateResult's doc comment for the full write-complete-temp-then-
// hardlink rationale, which applies identically here regardless of which
// fileName is being created.
func createOnceSidecar(dir, fileName, tempPrefix string, data []byte) (created bool, err error) {
	if len(data) > maxSidecarBytes {
		return false, fmt.Errorf("gaterun: %s data exceeds the %d byte bound", fileName, maxSidecarBytes)
	}

	dh, err := openRunDirNoFollow(dir)
	if err != nil {
		return false, err
	}
	defer func() { _ = dh.Close() }()

	tmpName, tmp, err := createTempAt(dh, tempPrefix, ".tmp")
	if err != nil {
		return false, fmt.Errorf("gaterun: create temp %s file: %w", fileName, err)
	}
	defer func() { _ = removeAt(dh, tmpName) }() // always cleaned up: linked content keeps its own inode.

	if cherr := tmp.Chmod(0o600); cherr != nil {
		_ = tmp.Close()
		return false, fmt.Errorf("gaterun: chmod temp %s file: %w", fileName, cherr)
	}
	if werr := writeSyncClose(tmp, data); werr != nil {
		return false, werr
	}

	linkErr := linkAt(dh, tmpName, fileName)
	if linkErr == nil {
		if dErr := fsyncDir(dh); dErr != nil {
			return false, dErr
		}
		return true, nil
	}
	if !errors.Is(linkErr, fs.ErrExist) {
		return false, linkErr
	}

	// Another writer already won the race (or fileName pre-dates this
	// call). Its content is guaranteed complete by construction — verify it
	// is safe, then decide idempotent-success vs conflict.
	existing, readErr := readRegularBounded(dh, fileName, maxSidecarBytes)
	if readErr != nil {
		return false, readErr
	}
	if bytes.Equal(existing, data) {
		return false, nil
	}
	return false, &ConflictError{Path: dir + string(os.PathSeparator) + fileName}
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
	return readOnceSidecar(dir, resultFileName)
}

// readOnceSidecar is ReadResult/ReadIdentity's shared implementation: read
// dir/fileName, or report (nil, false, nil) if it does not exist yet.
func readOnceSidecar(dir, fileName string) (data []byte, exists bool, err error) {
	dh, err := openRunDirNoFollow(dir)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = dh.Close() }()

	data, err = readRegularBounded(dh, fileName, maxSidecarBytes)
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

	dh, err := openRunDirNoFollow(dir)
	if err != nil {
		return err
	}
	defer func() { _ = dh.Close() }()

	if err := rejectUnsafeExistingTarget(dh, operationStateFileName); err != nil {
		return err
	}

	tmpName, tmp, err := createTempAt(dh, ".operation-state-", ".tmp")
	if err != nil {
		return fmt.Errorf("gaterun: create temp operation-state file: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = removeAt(dh, tmpName)
		}
	}()

	if cherr := tmp.Chmod(0o600); cherr != nil {
		_ = tmp.Close()
		return fmt.Errorf("gaterun: chmod temp operation-state file: %w", cherr)
	}
	if werr := writeSyncClose(tmp, data); werr != nil {
		return werr
	}
	if rerr := renameAt(dh, tmpName, operationStateFileName); rerr != nil {
		return rerr
	}
	committed = true
	if derr := fsyncDir(dh); derr != nil {
		return derr
	}
	return nil
}

// ReadOperationState reads and returns the bytes of dir/operation-state.json,
// or (nil, false, nil) if it does not exist yet (e.g. before the first
// operation-state write in a fresh run).
func ReadOperationState(dir string) (data []byte, exists bool, err error) {
	dh, err := openRunDirNoFollow(dir)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = dh.Close() }()

	data, err = readRegularBounded(dh, operationStateFileName, maxSidecarBytes)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}
