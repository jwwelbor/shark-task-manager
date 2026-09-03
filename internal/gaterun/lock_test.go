package gaterun

import (
	"sync"
	"testing"
	"time"
)

func TestRunLock_ExcludesConcurrentHolders(t *testing.T) {
	dir := newRunDir(t)

	lock, err := AcquireRunLock(dir, time.Second)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	acquired := make(chan struct{})
	go func() {
		l2, err := AcquireRunLock(dir, 5*time.Second)
		if err != nil {
			t.Errorf("second acquire: %v", err)
			return
		}
		close(acquired)
		_ = l2.Release()
	}()

	select {
	case <-acquired:
		t.Fatal("second acquire succeeded while first lock was still held")
	case <-time.After(100 * time.Millisecond):
		// expected: still blocked
	}

	if err := lock.Release(); err != nil {
		t.Fatalf("release: %v", err)
	}

	select {
	case <-acquired:
		// expected: second acquire now succeeds
	case <-time.After(2 * time.Second):
		t.Fatal("second acquire did not succeed after release")
	}
}

func TestRunLock_TimesOutWhenHeld(t *testing.T) {
	dir := newRunDir(t)
	lock, err := AcquireRunLock(dir, time.Second)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer lock.Release()

	_, err = AcquireRunLock(dir, 50*time.Millisecond)
	if err == nil {
		t.Fatal("second acquire with short timeout: want error, got nil")
	}
}

func TestRunLock_ReleaseAllowsReacquire(t *testing.T) {
	dir := newRunDir(t)
	lock, err := AcquireRunLock(dir, time.Second)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	lock2, err := AcquireRunLock(dir, time.Second)
	if err != nil {
		t.Fatalf("reacquire after release: %v", err)
	}
	_ = lock2.Release()
}

func TestRunLock_SerializesManyGoroutines(t *testing.T) {
	dir := newRunDir(t)
	const n = 8
	var mu sync.Mutex
	var order []int
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			l, err := AcquireRunLock(dir, 5*time.Second)
			if err != nil {
				t.Errorf("goroutine %d acquire: %v", i, err)
				return
			}
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
			time.Sleep(2 * time.Millisecond)
			_ = l.Release()
		}(i)
	}
	wg.Wait()
	if len(order) != n {
		t.Fatalf("len(order) = %d, want %d", len(order), n)
	}
}
