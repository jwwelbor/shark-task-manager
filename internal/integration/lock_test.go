package integration

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// Task T-E34-F08-012 covers test-plan.md TC-016's run-scoped-lock half:
// concurrent registration attempts for the same epicRunID must serialize
// through AcquireRegistrationLock/Release rather than interleave. Per
// TC-016's Caller-Path Contract, these tests drive real goroutines racing
// the real lock file — no test-side mutex serializes the callers, and the
// filesystem is never mocked.

// TestAcquireRegistrationLock_SerializesConcurrentGoroutines covers
// TC-016: several goroutines race AcquireRegistrationLock for the same
// epicRunID; a shared counter (protected only by its own bookkeeping
// mutex, not used to serialize the callers) must never observe more than
// one holder inside the critical section at once.
func TestAcquireRegistrationLock_SerializesConcurrentGoroutines(t *testing.T) {
	dir := t.TempDir()
	const epicRunID = "run-lock-serialize"

	var (
		bookkeeping sync.Mutex
		inFlight    int
		maxInFlight int
	)

	const goroutines = 8
	var start, done sync.WaitGroup
	errs := make([]error, goroutines)
	start.Add(1)
	done.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer done.Done()
			start.Wait() // barrier: release all goroutines together

			lock, err := AcquireRegistrationLock(dir, epicRunID)
			if err != nil {
				errs[i] = err
				return
			}

			bookkeeping.Lock()
			inFlight++
			if inFlight > maxInFlight {
				maxInFlight = inFlight
			}
			bookkeeping.Unlock()

			time.Sleep(5 * time.Millisecond) // widen the race window

			bookkeeping.Lock()
			inFlight--
			bookkeeping.Unlock()

			errs[i] = lock.Release()
		}()
	}
	start.Done() // release the barrier
	done.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
	if maxInFlight > 1 {
		t.Fatalf("lock did not serialize concurrent holders: observed %d concurrent holders, want at most 1", maxInFlight)
	}
}

// TestAcquireRegistrationLock_ReleaseUnblocksWaiter covers TC-016's
// blocking-contract half directly: a second Acquire for the same
// epicRunID blocks while the first holder holds the lock, and proceeds
// only after Release.
func TestAcquireRegistrationLock_ReleaseUnblocksWaiter(t *testing.T) {
	dir := t.TempDir()
	const epicRunID = "run-lock-unblock"

	first, err := AcquireRegistrationLock(dir, epicRunID)
	if err != nil {
		t.Fatalf("first AcquireRegistrationLock: %v", err)
	}

	acquired := make(chan struct{})
	waiterErr := make(chan error, 1)
	go func() {
		second, err := AcquireRegistrationLock(dir, epicRunID)
		if err != nil {
			waiterErr <- err
			return
		}
		close(acquired)
		waiterErr <- second.Release()
	}()

	select {
	case <-acquired:
		t.Fatal("second AcquireRegistrationLock returned before the first lock was released")
	case <-time.After(50 * time.Millisecond):
		// Expected: still blocked while the first holder holds the lock.
	}

	if err := first.Release(); err != nil {
		t.Fatalf("release first lock: %v", err)
	}

	select {
	case <-acquired:
		// Expected: the waiter proceeded once the lock was released.
	case <-time.After(2 * time.Second):
		t.Fatal("second AcquireRegistrationLock never unblocked after Release")
	}

	if err := <-waiterErr; err != nil {
		t.Fatalf("waiter goroutine: %v", err)
	}
}

// TestAcquireRegistrationLock_TimesOutOnWedgedLock supports TC-016
// (T-E34-F08-012) by covering the acquire-side timeout: a lock file left
// behind (simulating a crashed holder — see
// RegistrationLockTimeoutError's doc comment on why this task does not
// reclaim it) causes AcquireRegistrationLock to fail with a typed
// *RegistrationLockTimeoutError rather than blocking forever.
func TestAcquireRegistrationLock_TimesOutOnWedgedLock(t *testing.T) {
	origInterval, origTimeout := registrationLockPollInterval, registrationLockTimeout
	registrationLockPollInterval = time.Millisecond
	registrationLockTimeout = 20 * time.Millisecond
	t.Cleanup(func() {
		registrationLockPollInterval = origInterval
		registrationLockTimeout = origTimeout
	})

	dir := t.TempDir()
	const epicRunID = "run-lock-wedged"

	held, err := AcquireRegistrationLock(dir, epicRunID)
	if err != nil {
		t.Fatalf("acquire the lock to simulate a wedged holder: %v", err)
	}
	t.Cleanup(func() { _ = held.Release() })

	_, err = AcquireRegistrationLock(dir, epicRunID)
	if err == nil {
		t.Fatal("expected a timeout error acquiring an already-held lock, got nil")
	}
	var timeoutErr *RegistrationLockTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected *RegistrationLockTimeoutError, got %T: %v", err, err)
	}
}
