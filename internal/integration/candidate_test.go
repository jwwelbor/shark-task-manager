package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

// Task T-E34-F08-006 covers test-plan.md TC-007's candidate-side assertion
// (final EventIDs contains both concurrent writers' IDs) and TC-008 (stale-
// digest CAS rejection via the test-only synchronization hook). Per
// TC-008's Caller-Path Contract, these tests drive UpdateCandidate's real
// production path — never a lower-level unexported function fed a
// hand-constructed stale digest.

// TestUpdateCandidate_ConcurrentDifferentFeatures covers TC-007's
// candidate-side assertion and spec.md AC-4: two goroutines racing a real
// RecordEvent + UpdateCandidate sequence for two different feature keys
// under the same epicRunID both survive — two distinct event files exist,
// and the final IntegrationCandidate.EventIDs contains both IDs.
func TestUpdateCandidate_ConcurrentDifferentFeatures(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-candidate-concurrent"

	type call struct {
		featureKey    string
		featureCommit string
	}
	calls := []call{
		{featureKey: "E34-F20", featureCommit: "commit-x"},
		{featureKey: "E34-F21", featureCommit: "commit-y"},
	}

	var (
		start    sync.WaitGroup
		done     sync.WaitGroup
		mu       sync.Mutex
		errs     = make([]error, len(calls))
		eventIDs = make([]string, len(calls))
	)
	start.Add(1)
	done.Add(len(calls))

	for i, c := range calls {
		i, c := i, c
		go func() {
			defer done.Done()
			start.Wait() // barrier: release both goroutines together

			event, err := RecordEvent(epicRunID, c.featureKey, c.featureCommit, nil, nil)
			if err != nil {
				mu.Lock()
				errs[i] = err
				mu.Unlock()
				return
			}
			mu.Lock()
			eventIDs[i] = event.EventID
			mu.Unlock()

			_, err = UpdateCandidate(epicRunID, event)
			mu.Lock()
			errs[i] = err
			mu.Unlock()
		}()
	}
	start.Done() // release the barrier
	done.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: error: %v", i, err)
		}
	}

	final, _, err := readCandidate(candidatePath(dir, epicRunID))
	if err != nil {
		t.Fatalf("read final candidate: %v", err)
	}
	if final == nil {
		t.Fatal("no candidate file exists after both updates")
	}

	// Negative case (literal AC-4 wording): "neither overwrites the
	// other" — both events' IDs must survive in the final candidate.
	if len(final.EventIDs) != len(calls) {
		t.Fatalf("expected %d event IDs in final candidate, got %d: %v", len(calls), len(final.EventIDs), final.EventIDs)
	}
	for i, id := range eventIDs {
		found := false
		for _, gotID := range final.EventIDs {
			if gotID == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("goroutine %d's event %s missing from final EventIDs %v", i, id, final.EventIDs)
		}
	}
}

// TestUpdateCandidate_CreatesCandidateFile covers the base case underlying
// TC-007: a single UpdateCandidate call for a brand-new epicRunID creates
// the candidate file with that event's ID and a non-empty Digest.
func TestUpdateCandidate_CreatesCandidateFile(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-candidate-create"
	event, err := RecordEvent(epicRunID, "E34-F22", "commit-z", nil, nil)
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}

	candidate, err := UpdateCandidate(epicRunID, event)
	if err != nil {
		t.Fatalf("UpdateCandidate: %v", err)
	}
	if candidate.Digest == "" {
		t.Error("Digest is empty")
	}
	if len(candidate.EventIDs) != 1 || candidate.EventIDs[0] != event.EventID {
		t.Fatalf("EventIDs = %v, want [%s]", candidate.EventIDs, event.EventID)
	}

	onDisk, _, err := readCandidate(candidatePath(dir, epicRunID))
	if err != nil {
		t.Fatalf("read candidate file: %v", err)
	}
	if onDisk == nil {
		t.Fatal("candidate file does not exist on disk")
	}
	if onDisk.Digest != candidate.Digest {
		t.Errorf("on-disk Digest = %q, want %q", onDisk.Digest, candidate.Digest)
	}
}

// TestUpdateCandidate_DigestExcludesItself covers task AC-T1: recomputing
// the digest over the persisted candidate (with Digest cleared) must
// reproduce the recorded Digest — the digest is computed over the
// candidate's canonical JSON with the Digest field itself excluded.
func TestUpdateCandidate_DigestExcludesItself(t *testing.T) {
	_, _ = chdirProjectRoot(t)

	const epicRunID = "run-candidate-digest"
	event, err := RecordEvent(epicRunID, "E34-F23", "commit-w", nil, nil)
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}

	candidate, err := UpdateCandidate(epicRunID, event)
	if err != nil {
		t.Fatalf("UpdateCandidate: %v", err)
	}

	recomputed, err := computeDigest(*candidate)
	if err != nil {
		t.Fatalf("computeDigest: %v", err)
	}
	if recomputed != candidate.Digest {
		t.Errorf("recomputed digest %q != recorded digest %q", recomputed, candidate.Digest)
	}
}

// TestUpdateCandidate_StaleDigestRetrySucceeds covers TC-008 and task
// AC-T3/AC-T4's first half: a write against a stale expected-prior digest
// is rejected, but a single automatic retry closes the race when the
// second attempt's expected digest is current. The concurrent change is
// published deterministically via updateCandidateTestHook, exercising
// UpdateCandidate's real read-verify-write-retry path rather than a
// lower-level function fed a hand-constructed stale digest (TC-008's
// Caller-Path Contract).
func TestUpdateCandidate_StaleDigestRetrySucceeds(t *testing.T) {
	_, _ = chdirProjectRoot(t)

	const epicRunID = "run-candidate-cas-retry"

	seedEvent, err := RecordEvent(epicRunID, "E34-F30", "commit-seed", nil, nil)
	if err != nil {
		t.Fatalf("seed RecordEvent: %v", err)
	}
	if _, err := UpdateCandidate(epicRunID, seedEvent); err != nil {
		t.Fatalf("seed UpdateCandidate: %v", err)
	}

	concurrentEvent, err := RecordEvent(epicRunID, "E34-F31", "commit-concurrent", nil, nil)
	if err != nil {
		t.Fatalf("concurrent RecordEvent: %v", err)
	}

	var busy, triggered bool
	updateCandidateTestHook = func() {
		// Guard against two kinds of re-entry into this same hook: the
		// nested UpdateCandidate call below re-enters attemptUpdateCandidate
		// (and therefore this hook) on its own attempt, and the outer
		// call's own retry (attempt 2) re-enters it a second time. Both
		// must be no-ops — only the outer call's *first* attempt should
		// ever trigger the concurrent write.
		if busy || triggered {
			return
		}
		triggered = true
		busy = true
		defer func() { busy = false }()

		// Publish a fully-completed concurrent write while the outer
		// attempt is "paused" here, between its digest read and its
		// rename — this is what forces the outer attempt's first try to
		// observe a stale expected digest.
		if _, err := UpdateCandidate(epicRunID, concurrentEvent); err != nil {
			t.Errorf("concurrent UpdateCandidate during pause: %v", err)
		}
	}
	t.Cleanup(func() { updateCandidateTestHook = nil })

	targetEvent, err := RecordEvent(epicRunID, "E34-F32", "commit-target", nil, nil)
	if err != nil {
		t.Fatalf("target RecordEvent: %v", err)
	}

	result, err := UpdateCandidate(epicRunID, targetEvent)
	if err != nil {
		t.Fatalf("UpdateCandidate expected to succeed via its single retry: %v", err)
	}

	want := []string{seedEvent.EventID, concurrentEvent.EventID, targetEvent.EventID}
	sort.Strings(want)
	got := append([]string(nil), result.EventIDs...)
	sort.Strings(got)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("final EventIDs = %v, want %v (retry must fold in the concurrent write, not lose it)", got, want)
	}
}

// TestUpdateCandidate_PersistentConflictReportsTypedError covers TC-008's
// core assertion and task AC-T3/AC-T4's second half: when the conflict
// persists through the single retry, UpdateCandidate returns a typed
// *CandidateConflictError rather than looping a third time, and the file on
// disk after the rejected write is byte-identical to what the last
// successful (concurrent) writer published — the failed writer's own
// content never lands on disk.
func TestUpdateCandidate_PersistentConflictReportsTypedError(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-candidate-cas-persistent"

	seedEvent, err := RecordEvent(epicRunID, "E34-F40", "commit-seed", nil, nil)
	if err != nil {
		t.Fatalf("seed RecordEvent: %v", err)
	}
	if _, err := UpdateCandidate(epicRunID, seedEvent); err != nil {
		t.Fatalf("seed UpdateCandidate: %v", err)
	}

	var (
		busy         bool
		hookCalls    int
		lastOnDisk   []byte
		concurrentID int
	)
	path := candidatePath(dir, epicRunID)

	updateCandidateTestHook = func() {
		if busy {
			// Guards against the nested UpdateCandidate call below
			// re-entering this same hook — a single-threaded,
			// deterministic stand-in for "goroutine B", not a
			// test-side mutex serializing two real racers. Only the
			// outer call's own top-level attempts should ever count
			// below.
			return
		}
		hookCalls++
		busy = true
		defer func() { busy = false }()

		concurrentID++
		ev, err := RecordEvent(epicRunID, fmt.Sprintf("E34-conflict-%d", concurrentID), fmt.Sprintf("commit-conflict-%d", concurrentID), nil, nil)
		if err != nil {
			t.Errorf("conflict RecordEvent #%d: %v", concurrentID, err)
			return
		}
		if _, err := UpdateCandidate(epicRunID, ev); err != nil {
			t.Errorf("conflict UpdateCandidate #%d: %v", concurrentID, err)
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read candidate after conflict write #%d: %v", concurrentID, err)
			return
		}
		lastOnDisk = data
	}
	t.Cleanup(func() { updateCandidateTestHook = nil })

	targetEvent, err := RecordEvent(epicRunID, "E34-F41", "commit-target", nil, nil)
	if err != nil {
		t.Fatalf("target RecordEvent: %v", err)
	}

	_, err = UpdateCandidate(epicRunID, targetEvent)
	if err == nil {
		t.Fatal("expected a persistent-conflict error, got nil")
	}
	var conflict *CandidateConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected *CandidateConflictError, got %T: %v", err, err)
	}

	if hookCalls != maxUpdateCandidateAttempts {
		t.Fatalf("hook fired %d times, want exactly %d (one write attempt + one retry, no third attempt)", hookCalls, maxUpdateCandidateAttempts)
	}

	// AC-T3: the file on disk after the rejected write is byte-identical
	// to what the last successful concurrent writer published — the
	// failed writer's own content never landed on disk.
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read candidate after rejected write: %v", err)
	}
	if string(after) != string(lastOnDisk) {
		t.Fatalf("candidate file changed after the rejected write:\nbefore (last good write): %s\nafter:  %s", lastOnDisk, after)
	}

	// The failed writer's own event must not appear anywhere in the final
	// on-disk candidate.
	final, _, err := readCandidate(path)
	if err != nil {
		t.Fatalf("parse final candidate: %v", err)
	}
	for _, id := range final.EventIDs {
		if id == targetEvent.EventID {
			t.Fatalf("rejected writer's event %s leaked into on-disk candidate %v", targetEvent.EventID, final.EventIDs)
		}
	}
}

// Task T-E34-F08-012 covers test-plan.md TC-016's archived-candidate-head
// half: before UpdateCandidate replaces integration-candidate.json, the
// prior head's exact bytes must already be durably retained at
// integration-heads/<record-digest>.json (AC-T1). Per TC-016's Caller-Path
// Contract, these tests drive the real UpdateCandidate path — never
// attemptUpdateCandidate directly — and observe ordering via the
// production archiveTestHook seam rather than a test-side mutex.

// TestUpdateCandidate_FirstCandidateArchivesNothing covers TC-016 / AC-T1's
// "the very first candidate for a run has no prior head to retain" case: a
// brand-new candidate (current == nil) must not create an
// integration-heads/ directory at all.
func TestUpdateCandidate_FirstCandidateArchivesNothing(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-candidate-archive-first"
	event, err := RecordEvent(epicRunID, "E34-F50", "commit-only", nil, nil)
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if _, err := UpdateCandidate(epicRunID, event); err != nil {
		t.Fatalf("UpdateCandidate: %v", err)
	}

	headsDir := candidateHeadsDir(candidatePath(dir, epicRunID))
	if _, statErr := os.Stat(headsDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected no integration-heads directory for a brand-new candidate, stat returned err=%v", statErr)
	}
}

// TestUpdateCandidate_ArchivesPriorHeadBeforeReplacing covers TC-016's core
// ordering assertion: when a second event folds into an existing
// candidate, the prior head's exact bytes are archived to
// integration-heads/<priorDigest>.json *before* the live candidate file is
// replaced. archiveTestHook fires between those two steps in production
// code, so observing both files at that instant proves the ordering — a
// single end-state check could not distinguish archive-then-replace from
// replace-then-archive (test-plan.md TC-016's Caller-Path Contract note).
func TestUpdateCandidate_ArchivesPriorHeadBeforeReplacing(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-candidate-archive-order"

	firstEvent, err := RecordEvent(epicRunID, "E34-F51", "commit-first", nil, nil)
	if err != nil {
		t.Fatalf("first RecordEvent: %v", err)
	}
	first, err := UpdateCandidate(epicRunID, firstEvent)
	if err != nil {
		t.Fatalf("first UpdateCandidate: %v", err)
	}

	path := candidatePath(dir, epicRunID)
	priorBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read prior candidate file: %v", err)
	}
	archivePath := filepath.Join(candidateHeadsDir(path), first.Digest+".json")

	secondEvent, err := RecordEvent(epicRunID, "E34-F52", "commit-second", nil, nil)
	if err != nil {
		t.Fatalf("second RecordEvent: %v", err)
	}

	var (
		hookFired          bool
		archiveBytesAtHook []byte
		liveBytesAtHook    []byte
	)
	archiveTestHook = func() {
		hookFired = true
		archiveBytesAtHook, _ = os.ReadFile(archivePath)
		liveBytesAtHook, _ = os.ReadFile(path)
	}
	t.Cleanup(func() { archiveTestHook = nil })

	second, err := UpdateCandidate(epicRunID, secondEvent)
	if err != nil {
		t.Fatalf("second UpdateCandidate: %v", err)
	}

	if !hookFired {
		t.Fatal("archiveTestHook never fired — archival did not happen on this transition")
	}
	if len(archiveBytesAtHook) == 0 {
		t.Fatal("archived head file did not exist yet when the hook fired (archive-before-replace ordering violated)")
	}
	if string(archiveBytesAtHook) != string(priorBytes) {
		t.Fatalf("archived head bytes differ from the original prior head bytes:\narchived: %s\noriginal: %s", archiveBytesAtHook, priorBytes)
	}
	if string(liveBytesAtHook) != string(priorBytes) {
		t.Fatalf("live candidate file was already replaced when the hook fired (write-then-replace ordering violated):\ngot:  %s\nwant (unchanged prior bytes): %s", liveBytesAtHook, priorBytes)
	}

	finalArchiveBytes, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("archived head file missing after update: %v", err)
	}
	if string(finalArchiveBytes) != string(priorBytes) {
		t.Fatal("archived head bytes changed after the update completed")
	}
	if second.Digest == first.Digest {
		t.Fatal("digest did not change after folding in a second event")
	}
}

// TestUpdateCandidate_ArchivedHeadDigestRecomputable covers AC-T1's literal
// wording: "every prior_record_digest is recomputable from retained
// bytes" — recomputing computeDigest over the archived file's parsed
// content must reproduce the exact digest the file is named for.
func TestUpdateCandidate_ArchivedHeadDigestRecomputable(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-candidate-archive-recompute"

	firstEvent, err := RecordEvent(epicRunID, "E34-F53", "commit-a", nil, nil)
	if err != nil {
		t.Fatalf("first RecordEvent: %v", err)
	}
	first, err := UpdateCandidate(epicRunID, firstEvent)
	if err != nil {
		t.Fatalf("first UpdateCandidate: %v", err)
	}

	secondEvent, err := RecordEvent(epicRunID, "E34-F54", "commit-b", nil, nil)
	if err != nil {
		t.Fatalf("second RecordEvent: %v", err)
	}
	if _, err := UpdateCandidate(epicRunID, secondEvent); err != nil {
		t.Fatalf("second UpdateCandidate: %v", err)
	}

	archivePath := filepath.Join(candidateHeadsDir(candidatePath(dir, epicRunID)), first.Digest+".json")
	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archived head: %v", err)
	}

	var archived IntegrationCandidate
	if err := json.Unmarshal(data, &archived); err != nil {
		t.Fatalf("archived head is not valid JSON: %v", err)
	}
	recomputed, err := computeDigest(archived)
	if err != nil {
		t.Fatalf("computeDigest over archived head: %v", err)
	}
	if recomputed != first.Digest {
		t.Fatalf("recomputed digest from archived bytes = %q, want %q (the record_digest the archive file is named for)", recomputed, first.Digest)
	}
}

// TestArchiveCandidateHead_IdempotentWhenAlreadyArchived supports TC-016
// (T-E34-F08-012) by covering archiveCandidateHead's own idempotent-retry
// guarantee directly: writing the identical headDigest a second time is a
// no-op, not an error.
func TestArchiveCandidateHead_IdempotentWhenAlreadyArchived(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	path := candidatePath(dir, "run-archive-idempotent")
	data := []byte(`{"epic_run_id":"run-archive-idempotent"}`)

	if err := archiveCandidateHead(path, "digest-x", data); err != nil {
		t.Fatalf("first archiveCandidateHead: %v", err)
	}
	if err := archiveCandidateHead(path, "digest-x", data); err != nil {
		t.Fatalf("second (idempotent) archiveCandidateHead: %v", err)
	}

	archived, err := os.ReadFile(filepath.Join(candidateHeadsDir(path), "digest-x.json"))
	if err != nil {
		t.Fatalf("read archived head: %v", err)
	}
	if string(archived) != string(data) {
		t.Fatalf("archived content = %s, want %s", archived, data)
	}
}
