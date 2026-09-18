package gaterun

import (
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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

// lockNonceBytes is the size of the random token AcquireRunLock writes into
// the lock file and Release later verifies before unlinking (TD-186). 16
// bytes of crypto/rand is far more than enough to make an accidental
// collision with a different holder's token practically impossible.
const lockNonceBytes = 16

// RunLock is a held per-run advisory lock. Release must be called exactly
// once to free it. It holds the same no-follow-verified run-directory
// handle AcquireRunLock derived the lock file from, so Release unlinks the
// lock file relative to that descriptor rather than re-deriving and
// separately reusing the run directory's path. It also holds the random
// token AcquireRunLock wrote into the lock file it created, so Release can
// fence its unlink against a different holder's lock file of the same name
// (see the fencing note on Release).
type RunLock struct {
	dh    *os.File
	nonce string
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
			nonce, nerr := writeLockNonce(f)
			closeErr := f.Close()
			if nerr == nil {
				nerr = closeErr
			}
			if nerr != nil {
				_ = removeAt(dh, lockFileName)
				_ = dh.Close()
				return nil, fmt.Errorf("gaterun: acquire run lock %s/%s: write ownership token: %w", dir, lockFileName, nerr)
			}
			return &RunLock{dh: dh, nonce: nonce}, nil
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

// writeLockNonce generates a random hex token and writes it to the
// just-created lock file f, returning the token so the caller can compare
// against it later.
func writeLockNonce(f *os.File) (string, error) {
	var buf [lockNonceBytes]byte
	if _, err := crand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("gaterun: generate lock ownership token: %w", err)
	}
	nonce := hex.EncodeToString(buf[:])
	if _, err := f.WriteString(nonce); err != nil {
		return "", fmt.Errorf("gaterun: write lock ownership token: %w", err)
	}
	return nonce, nil
}

// Release frees the lock. It is safe to call once; a second call returns an
// error rather than silently succeeding, so a double-release bug surfaces.
//
// Before unlinking, Release re-opens the lock file relative to l.dh (never
// by a re-joined path — that would reopen exactly the TOCTOU this package's
// no-follow helpers exist to close, via openRegularNoFollowAt, the same
// no-follow-then-fstat-verified read this package already uses for
// identity.json) and compares its content against the random token
// AcquireRunLock wrote when it created the file. This guards the
// cross-holder case TD-186 tracked: an operator declares this holder's lock
// stale and removes it (the documented recovery path above) while this
// holder is merely hung, not crashed; a second process then acquires a fresh
// lock file of the same name; and this holder's Release finally fires.
// Without the check it would unlink the second holder's lock file. With it,
// a mismatch is refused and reported rather than silently deleting another
// holder's lock.
//
// This narrows, but cannot fully close, the race: neither POSIX nor Win32
// offers an atomic "delete this file only if its content still equals X," so
// a delete-then-recreate landing in the few-microsecond gap between the read
// below and the unlink call would still be removed by this call. Closing
// that residual window would require switching from a presence-marker file
// to `flock`-style locking tied to the fd itself, which is out of scope here
// (see TD-186's research report). An inode/device-identity fencing scheme
// was tried first and rejected: on this repository's ext4 checkout, a plain
// remove-then-create in the same directory deterministically reuses the
// freed inode number, so identity fencing gave zero protection in exactly
// the scenario this fix targets. A random content nonce does not have that
// failure mode.
func (l *RunLock) Release() error {
	if l.dh == nil {
		return fmt.Errorf("gaterun: release run lock: already released")
	}
	dh := l.dh
	wantNonce := l.nonce
	l.dh = nil
	defer func() { _ = dh.Close() }()

	f, err := openRegularNoFollowAt(dh, lockFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("gaterun: release run lock %s/%s: lock file no longer exists (already removed)", dh.Name(), lockFileName)
		}
		return fmt.Errorf("gaterun: release run lock %s/%s: read ownership token: %w", dh.Name(), lockFileName, err)
	}
	data, readErr := io.ReadAll(f)
	_ = f.Close()
	if readErr != nil {
		return fmt.Errorf("gaterun: release run lock %s/%s: read ownership token: %w", dh.Name(), lockFileName, readErr)
	}
	if string(data) != wantNonce {
		return fmt.Errorf("gaterun: release run lock %s/%s: lock file was replaced by another holder; refusing to unlink it", dh.Name(), lockFileName)
	}

	if err := removeAt(dh, lockFileName); err != nil {
		return fmt.Errorf("gaterun: release run lock %s/%s: %w", dh.Name(), lockFileName, err)
	}
	return nil
}
