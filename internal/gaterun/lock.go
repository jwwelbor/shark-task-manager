package gaterun

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
// once to free it.
type RunLock struct {
	path string
}

// AcquireRunLock acquires the advisory per-run lock for dir (a directory
// returned by RunDir), blocking and retrying until either the lock is
// acquired or timeout elapses. The lock is implemented as an O_EXCL-created
// sentinel file: creation is atomic at the syscall level, so it is safe
// across both goroutines within one process and separate processes sharing
// the same run directory (e.g. a restarted parent).
//
// A held lock is never silently stolen: on timeout, AcquireRunLock returns
// an error rather than breaking another holder's lock. Operators recovering
// from a genuinely crashed holder must remove the stale lock file directly.
func AcquireRunLock(dir string, timeout time.Duration) (*RunLock, error) {
	path := filepath.Join(dir, lockFileName)
	deadline := time.Now().Add(timeout)

	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304 -- dir is validated by RunDir.
		if err == nil {
			_ = f.Close()
			return &RunLock{path: path}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("gaterun: acquire run lock %s: %w", path, err)
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("gaterun: timed out waiting for run lock %s", path)
		}
		time.Sleep(lockPollInterval)
	}
}

// Release frees the lock. It is safe to call once; a second call returns an
// error rather than silently succeeding, so a double-release bug surfaces.
func (l *RunLock) Release() error {
	if err := os.Remove(l.path); err != nil {
		return fmt.Errorf("gaterun: release run lock %s: %w", l.path, err)
	}
	return nil
}
