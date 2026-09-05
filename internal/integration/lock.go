// Package integration — see run.go's package doc.
package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// registrationLockPollInterval / registrationLockTimeout bound how long
// AcquireRegistrationLock polls a contended lock before giving up, so a
// wedged lock fails fast with a diagnosable error instead of hanging the
// caller — and, in a test, the whole test binary — forever. Both are vars
// (not consts) so tests can shrink them, mirroring
// candidate.go's updateCandidateTestHook seam convention: production code
// never touches them, and a test that does restores the defaults via
// t.Cleanup.
var (
	registrationLockPollInterval = 5 * time.Millisecond
	registrationLockTimeout      = 10 * time.Second
)

// RegistrationLockTimeoutError indicates AcquireRegistrationLock gave up
// waiting for a contended lock file after registrationLockTimeout. It is a
// distinct type (not a plain fmt.Errorf) so a caller can distinguish
// "still held by someone else" from any other acquisition failure (e.g. a
// permissions error creating the lock directory).
//
// This task deliberately does not attempt to reclaim a lock left behind by
// a crashed holder — no staleness/TTL logic here, unlike the entity-claim
// system's claim_ttl_seconds. That is a real, named gap: task
// T-E34-F08-012 covers only the happy-path registration sequence, and
// T-E34-F08-014's "crash-restart repair" scope is run.go only, not this
// file — so nothing currently reclaims a lock file left behind by a
// process that died mid-registration. A wedged lock will keep timing out
// future registration attempts for that epicRunID until the lock file is
// removed by hand.
type RegistrationLockTimeoutError struct {
	Path string
}

// Error implements the error interface.
func (e *RegistrationLockTimeoutError) Error() string {
	return fmt.Sprintf("integration: timed out waiting for registration lock at %s", e.Path)
}

// RegistrationLock represents one held run-scoped advisory lock guarding
// an epic run's registration-note write sequence (spec.md "CAS over a
// mutex": CAS remains candidate.go's UpdateCandidate mechanism for
// concurrent candidate updates; this lock's sole job is serializing the
// *multi-step* registration sequence — fsync events/head, then insert the
// note — which a single CAS write cannot express, per architecture.md
// "Epic integration candidate identity"). Call Release to unblock the next
// waiter.
type RegistrationLock struct {
	path string
}

// AcquireRegistrationLock blocks — polling with a short, fixed backoff —
// until it exclusively creates the lock file for epicRunID's registration
// sequence, or returns a *RegistrationLockTimeoutError once
// registrationLockTimeout elapses without acquiring it. Exactly one
// caller — a goroutine in this process or a competing process — ever
// holds the lock for a given epicRunID at a time; every other concurrent
// caller for that same epicRunID blocks until the holder calls Release.
//
// Like run.go/event.go/candidate.go's publish patterns, mutual exclusion
// rests on os.OpenFile's O_EXCL create-if-absent guarantee — but unlike
// those "try once, then read the winner's state" patterns, a lock waiter
// must actually wait its turn rather than walk away, so this polls instead
// of failing immediately on contention. Works identically for goroutines
// racing within one process and for independent processes, since it never
// relies on an in-process mutex (spec.md "Key technical decisions" #3: the
// parent-loop processes this guards are not threads in one process).
func AcquireRegistrationLock(projectRoot, epicRunID string) (*RegistrationLock, error) {
	path := registrationLockPath(projectRoot, epicRunID)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, runDirMode); err != nil {
		return nil, fmt.Errorf("integration: create registration-lock directory %s: %w", dir, err)
	}

	deadline := time.Now().Add(registrationLockTimeout)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, runFileMode)
		if err == nil {
			_ = f.Close()
			return &RegistrationLock{path: path}, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("integration: acquire registration lock at %s: %w", path, err)
		}
		if time.Now().After(deadline) {
			return nil, &RegistrationLockTimeoutError{Path: path}
		}
		time.Sleep(registrationLockPollInterval)
	}
}

// Release releases l, deleting its lock file so the next blocked caller's
// AcquireRegistrationLock can proceed. Safe to call once per successful
// Acquire; removing an already-removed lock file is not an error.
func (l *RegistrationLock) Release() error {
	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("integration: release registration lock at %s: %w", l.path, err)
	}
	return nil
}

// registrationLockPath is the per-epic-run lock-file path guarding
// epicRunID's registration-note write sequence.
func registrationLockPath(projectRoot, epicRunID string) string {
	return filepath.Join(projectRoot, ".shark", "runs", epicRunID, "registration.lock")
}
