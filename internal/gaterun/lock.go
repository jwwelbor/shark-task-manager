package gaterun

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// lockFileName is the per-run advisory lock this package uses to serialize
// CreateResult/WriteOperationState sequences (REQ-F-003's "under a per-run
// lock"). It sits alongside, not inside, the two sidecar files it guards.
const lockFileName = ".run.lock"

// lockPollInterval is the retry cadence AcquireRunLock uses while spin-
// waiting for a held lock. Short enough to keep AcquireRunLock's timeout
// parameter meaningful for both fast unit tests and real dispatch use.
const lockPollInterval = 5 * time.Millisecond

// DefaultLockTimeout is a sensible default AcquireRunLock timeout for
// callers coordinating a broader guarded sequence (e.g. the persistence
// coordinator holding the lock across its own target writes and sidecar
// update). CreateResult and WriteOperationState in this package do not use
// it themselves — their own atomicity comes from the hardlink-create and
// rename-replace protocols, not from this lock — so a caller that already
// holds the run lock via AcquireRunLock can safely call them without risking
// a self-deadlock.
const DefaultLockTimeout = 30 * time.Second

// RunLock is a held per-run advisory lock. Release must be called exactly
// once to free it. It holds the same no-follow-verified run-directory
// handle AcquireRunLock derived the lock file from, so Release unlinks the
// lock file relative to that descriptor rather than re-deriving and
// separately reusing the run directory's path.
type RunLock struct {
	dh *os.File
}

// AcquireRunLock acquires the advisory per-run lock for dir (a directory
// returned by RunDir), blocking and retrying until either the lock is
// acquired or timeout elapses. The lock is implemented as an O_EXCL-created
// sentinel file relative to a no-follow-verified run-directory handle:
// creation is atomic at the syscall level, so it is safe across both
// goroutines within one process and separate processes sharing the same run
// directory (e.g. a restarted parent).
//
// A held lock is never silently stolen: on timeout, AcquireRunLock returns
// an error rather than breaking another holder's lock. Operators recovering
// from a genuinely crashed holder must remove the stale lock file directly.
func AcquireRunLock(dir string, timeout time.Duration) (*RunLock, error) {
	dh, err := openRunDirNoFollow(dir)
	if err != nil {
		return nil, fmt.Errorf("gaterun: acquire run lock: %w", err)
	}
	deadline := time.Now().Add(timeout)

	for {
		f, err := createExclAt(dh, lockFileName, 0o600)
		if err == nil {
			_ = f.Close()
			return &RunLock{dh: dh}, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			_ = dh.Close()
			return nil, fmt.Errorf("gaterun: acquire run lock %s/%s: %w", dir, lockFileName, err)
		}
		if time.Now().After(deadline) {
			_ = dh.Close()
			return nil, fmt.Errorf("gaterun: timed out waiting for run lock %s/%s", dir, lockFileName)
		}
		time.Sleep(lockPollInterval)
	}
}

// Release frees the lock. It is safe to call once; a second call returns an
// error rather than silently succeeding, so a double-release bug surfaces.
func (l *RunLock) Release() error {
	if l.dh == nil {
		return fmt.Errorf("gaterun: release run lock: already released")
	}
	dh := l.dh
	l.dh = nil
	defer func() { _ = dh.Close() }()

	if err := removeAt(dh, lockFileName); err != nil {
		return fmt.Errorf("gaterun: release run lock %s/%s: %w", dh.Name(), lockFileName, err)
	}
	return nil
}
