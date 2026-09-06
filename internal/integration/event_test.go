package integration

import (
	"encoding/json"
	"os"
	"sync"
	"testing"
)

// Task T-E34-F08-005 covers only test-plan.md TC-007's package-level
// subtest — two goroutines racing real RecordEvent writes for two different
// feature keys under the same EpicRunID, against a real temp directory (no
// filesystem mock, matching TC-007's Caller-Path Contract). The
// feature-completion call-site wiring subtest is a separate task
// (T-E34-F08-008) and is not covered here.

// TestRecordEvent_ConcurrentDifferentFeatures covers the task's own Test
// Cases text and spec.md AC-4: two goroutines racing RecordEvent for two
// different feature keys under the same epicRunID both survive — two
// distinct, readable event files exist on disk, and each event's EventID
// matches the deterministic sha256-derivation rule (AC-T1).
func TestRecordEvent_ConcurrentDifferentFeatures(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-concurrent"

	type call struct {
		featureKey    string
		featureCommit string
	}
	calls := []call{
		{featureKey: "E34-F01", featureCommit: "commit-a"},
		{featureKey: "E34-F02", featureCommit: "commit-b"},
	}

	var (
		start  sync.WaitGroup
		done   sync.WaitGroup
		mu     sync.Mutex
		events = make([]*IntegrationEvent, len(calls))
		errs   = make([]error, len(calls))
	)
	start.Add(1)
	done.Add(len(calls))

	for i, c := range calls {
		i, c := i, c
		go func() {
			defer done.Done()
			start.Wait() // barrier: release both goroutines together
			event, err := RecordEvent(epicRunID, c.featureKey, c.featureCommit, nil, nil)
			mu.Lock()
			events[i] = event
			errs[i] = err
			mu.Unlock()
		}()
	}
	start.Done() // release the barrier
	done.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: RecordEvent error: %v", i, err)
		}
	}

	seenIDs := make(map[string]bool)
	for i, event := range events {
		want := deriveEventID(epicRunID, calls[i].featureKey, calls[i].featureCommit)
		if event.EventID != want {
			t.Errorf("goroutine %d: EventID = %q, want %q (deterministic derivation)", i, event.EventID, want)
		}
		seenIDs[event.EventID] = true

		path := eventRecordPath(dir, epicRunID, event.EventID)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("goroutine %d: event file not readable at %s: %v", i, path, err)
		}
		var onDisk IntegrationEvent
		if err := json.Unmarshal(data, &onDisk); err != nil {
			t.Fatalf("goroutine %d: event file at %s is not valid JSON: %v", i, path, err)
		}
		if onDisk.FeatureKey != calls[i].featureKey {
			t.Errorf("goroutine %d: on-disk FeatureKey = %q, want %q", i, onDisk.FeatureKey, calls[i].featureKey)
		}
		if onDisk.FeatureCommit != calls[i].featureCommit {
			t.Errorf("goroutine %d: on-disk FeatureCommit = %q, want %q", i, onDisk.FeatureCommit, calls[i].featureCommit)
		}
	}

	// Negative case (literal AC-4 wording): "neither overwrites the other" —
	// both event files must exist, and they must be distinct files.
	if len(seenIDs) != len(calls) {
		t.Fatalf("expected %d distinct event files, found %d distinct EventIDs: %v", len(calls), len(seenIDs), seenIDs)
	}
}

// TestRecordEvent_IdempotentRetry covers task AC-T1: a retried RecordEvent
// call for the identical completion (same epicRunID/featureKey/
// featureCommit) is idempotent — same EventID, same file, byte-identical
// contents on disk (the second call returns the already-persisted record
// rather than recomputing/overwriting it).
func TestRecordEvent_IdempotentRetry(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-retry"
	const featureKey = "E34-F03"
	const featureCommit = "commit-c"

	first, err := RecordEvent(epicRunID, featureKey, featureCommit, []string{"a.go"}, []string{"b.txt"})
	if err != nil {
		t.Fatalf("first RecordEvent: %v", err)
	}

	path := eventRecordPath(dir, epicRunID, first.EventID)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read event file after first call: %v", err)
	}

	second, err := RecordEvent(epicRunID, featureKey, featureCommit, []string{"a.go"}, []string{"b.txt"})
	if err != nil {
		t.Fatalf("second (retried) RecordEvent: %v", err)
	}

	if second.EventID != first.EventID {
		t.Fatalf("retried EventID = %q, want %q (deterministic derivation must be stable)", second.EventID, first.EventID)
	}
	if !second.RecordedAt.Equal(first.RecordedAt) {
		t.Fatalf("retried RecordedAt = %v, want %v (retry must return the persisted record, not recompute it)", second.RecordedAt, first.RecordedAt)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read event file after retried call: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("event file changed after a retried write:\nbefore: %s\nafter:  %s", before, after)
	}
}

// TestRecordEvent_DifferentCommitProducesDistinctEventID covers task AC-T2:
// a different featureCommit for a re-opened, re-completed feature (same
// epicRunID and featureKey) produces a new, distinct EventID and a separate
// event file — the earlier event file is untouched.
func TestRecordEvent_DifferentCommitProducesDistinctEventID(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-reopen"
	const featureKey = "E34-F04"

	first, err := RecordEvent(epicRunID, featureKey, "commit-first", nil, nil)
	if err != nil {
		t.Fatalf("first RecordEvent: %v", err)
	}

	second, err := RecordEvent(epicRunID, featureKey, "commit-second", nil, nil)
	if err != nil {
		t.Fatalf("second RecordEvent (re-opened/re-completed): %v", err)
	}

	if second.EventID == first.EventID {
		t.Fatalf("EventID unchanged across different featureCommit values: both are %q", first.EventID)
	}

	firstPath := eventRecordPath(dir, epicRunID, first.EventID)
	if _, err := os.Stat(firstPath); err != nil {
		t.Fatalf("first event file missing after second completion: %v", err)
	}
	secondPath := eventRecordPath(dir, epicRunID, second.EventID)
	if _, err := os.Stat(secondPath); err != nil {
		t.Fatalf("second event file missing: %v", err)
	}
}
