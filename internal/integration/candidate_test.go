package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
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

			_, err = UpdateCandidate(context.Background(), epicRunID, event)
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

// TestUpdateCandidate_WaitsForInFlightClaim proves that an observed claim is
// not immediately charged against the caller's one stale-digest retry. The
// first writer holds its real claim after acquiring it; the second writer
// observes that claim, waits for the first publish, then folds its own event
// onto the new candidate. This is deterministic rather than relying on the
// scheduler to make TC-007 collide at the narrow claim-to-rename window.
func TestUpdateCandidate_WaitsForInFlightClaim(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-candidate-inflight-claim"
	firstEvent, err := RecordEvent(epicRunID, "E34-F22", "commit-first", nil, nil)
	if err != nil {
		t.Fatalf("first RecordEvent: %v", err)
	}
	secondEvent, err := RecordEvent(epicRunID, "E34-F23", "commit-second", nil, nil)
	if err != nil {
		t.Fatalf("second RecordEvent: %v", err)
	}

	claimed, contended, release := installCandidateClaimPause(t)

	firstResult := make(chan error, 1)
	go func() {
		_, err := UpdateCandidate(context.Background(), epicRunID, firstEvent)
		firstResult <- err
	}()
	<-claimed

	secondResult := make(chan error, 1)
	go func() {
		_, err := UpdateCandidate(context.Background(), epicRunID, secondEvent)
		secondResult <- err
	}()
	<-contended
	release()

	if err := <-firstResult; err != nil {
		t.Fatalf("first UpdateCandidate: %v", err)
	}
	if err := <-secondResult; err != nil {
		t.Fatalf("second UpdateCandidate waited for the in-flight claim instead of exhausting retries: %v", err)
	}

	final, _, err := readCandidate(candidatePath(dir, epicRunID))
	if err != nil {
		t.Fatalf("read final candidate: %v", err)
	}
	want := []string{firstEvent.EventID, secondEvent.EventID}
	sort.Strings(want)
	got := append([]string(nil), final.EventIDs...)
	sort.Strings(got)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("final EventIDs = %v, want %v", got, want)
	}
}

func installCandidateClaimPause(t *testing.T) (<-chan struct{}, <-chan struct{}, func()) {
	t.Helper()
	claimed := make(chan struct{})
	contended := make(chan struct{})
	release := make(chan struct{})
	var claimOnce, contentionOnce sync.Once
	candidateClaimAcquiredTestHook = func() {
		claimOnce.Do(func() {
			close(claimed)
			<-release
		})
	}
	candidateClaimContendedTestHook = func() {
		contentionOnce.Do(func() { close(contended) })
	}
	t.Cleanup(func() {
		candidateClaimAcquiredTestHook = nil
		candidateClaimContendedTestHook = nil
	})
	return claimed, contended, func() { close(release) }
}

// TestUpdateCandidate_DuplicateEventIsIdempotent covers the no-op sibling of
// the in-flight-claim path. A duplicate EventID returns the existing
// candidate without consuming a claim slot, leaving a distinct next event
// free to advance the candidate. Pre-idempotency no-op claims remain
// fail-closed for TD-212's crash-safe recovery work.
func TestUpdateCandidate_DuplicateEventIsIdempotent(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-candidate-duplicate-claim"
	event, err := RecordEvent(epicRunID, "E34-F24", "commit-duplicate", nil, nil)
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if _, err := UpdateCandidate(context.Background(), epicRunID, event); err != nil {
		t.Fatalf("seed UpdateCandidate: %v", err)
	}

	seed, _, err := readCandidate(candidatePath(dir, epicRunID))
	if err != nil {
		t.Fatalf("read seed candidate: %v", err)
	}
	duplicate, err := UpdateCandidate(context.Background(), epicRunID, event)
	if err != nil {
		t.Fatalf("duplicate UpdateCandidate must be idempotent: %v", err)
	}
	if duplicate.Digest != seed.Digest {
		t.Fatalf("duplicate candidate digest = %s, want unchanged %s", duplicate.Digest, seed.Digest)
	}

	path := candidatePath(dir, epicRunID)
	claimPath := filepath.Join(candidateClaimsDir(path), claimFileName(seed.Digest))
	if _, err := os.Stat(claimPath); !os.IsNotExist(err) {
		t.Fatalf("duplicate UpdateCandidate consumed a claim slot at %s: %v", claimPath, err)
	}

	distinctEvent, err := RecordEvent(epicRunID, "E34-F25", "commit-distinct", nil, nil)
	if err != nil {
		t.Fatalf("distinct RecordEvent: %v", err)
	}
	advanced, err := UpdateCandidate(context.Background(), epicRunID, distinctEvent)
	if err != nil {
		t.Fatalf("distinct UpdateCandidate after duplicate no-op: %v", err)
	}

	want := []string{event.EventID, distinctEvent.EventID}
	sort.Strings(want)
	got := append([]string(nil), advanced.EventIDs...)
	sort.Strings(got)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("advanced EventIDs = %v, want %v", got, want)
	}

	// Retries need the same guarantee after another event has advanced the
	// head: retrying the older event must not republish it as HeadCommit.
	retried, err := UpdateCandidate(context.Background(), epicRunID, event)
	if err != nil {
		t.Fatalf("older duplicate UpdateCandidate: %v", err)
	}
	if retried.Digest != advanced.Digest {
		t.Fatalf("older duplicate digest = %s, want unchanged %s", retried.Digest, advanced.Digest)
	}
	if retried.HeadCommit != distinctEvent.FeatureCommit {
		t.Fatalf("older duplicate HeadCommit = %s, want %s", retried.HeadCommit, distinctEvent.FeatureCommit)
	}
}

func TestUpdateCandidate_CanceledContextDoesNotPublishCandidate(t *testing.T) {
	dir, _ := chdirProjectRoot(t)
	const epicRunID = "run-cancelled-candidate"
	event, err := RecordEvent(epicRunID, "E34-F99", "commit-cancelled", nil, nil)
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = UpdateCandidate(ctx, epicRunID, event)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("UpdateCandidate() error = %v, want context cancellation", err)
	}
	if _, err := os.Stat(candidatePath(dir, epicRunID)); !os.IsNotExist(err) {
		t.Fatalf("UpdateCandidate published a candidate after cancellation: %v", err)
	}
}

func TestGetCandidate_CanceledContext(t *testing.T) {
	chdirProjectRoot(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := GetCandidate(ctx, "run-cancelled-read")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetCandidate() error = %v, want context cancellation", err)
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

	candidate, err := UpdateCandidate(context.Background(), epicRunID, event)
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
	if onDisk.PathDigestSchemaVersion != currentPathDigestSchemaVersion {
		t.Errorf("on-disk PathDigestSchemaVersion = %d, want %d", onDisk.PathDigestSchemaVersion, currentPathDigestSchemaVersion)
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

	candidate, err := UpdateCandidate(context.Background(), epicRunID, event)
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

func TestMigrateCandidatePathDigestsAuthorized_RejectsNilAuthorizerForWrites(t *testing.T) {
	if _, err := MigrateCandidatePathDigestsAuthorized(context.Background(), "E99", "run-nil-authorizer", false, nil); err == nil || !strings.Contains(err.Error(), "authorization callback") {
		t.Fatalf("error = %v, want required authorization callback", err)
	}
}

// TestMigrateCandidatePathDigests_RepairsLegacyCandidate covers B076: a
// candidate written before candidate-level path-digest capture must be
// explicitly migrated before integration_review can trust it. The migration
// preserves the accumulated identity, captures the current clean-tree
// inventory, and retains the exact legacy bytes as an archived predecessor.
func TestMigrateCandidatePathDigests_RepairsLegacyCandidate(t *testing.T) {
	dir, headCommit := chdirProjectRoot(t)

	const epicKey = "E99"
	run, err := CaptureBase(context.Background(), epicKey)
	if err != nil {
		t.Fatalf("CaptureBase: %v", err)
	}

	legacy := IntegrationCandidate{
		EpicRunID:  run.EpicRunID,
		BaseCommit: headCommit,
		HeadCommit: "legacy-head",
		EventIDs:   []string{"legacy-event"},
		Digest:     "",
	}
	legacy.Digest, err = computeDigest(legacy)
	if err != nil {
		t.Fatalf("compute legacy digest: %v", err)
	}
	legacyBytes, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy candidate: %v", err)
	}
	path := candidatePath(dir, run.EpicRunID)
	if err := os.MkdirAll(filepath.Dir(path), runDirMode); err != nil {
		t.Fatalf("create candidate directory: %v", err)
	}
	if err := os.WriteFile(path, legacyBytes, runFileMode); err != nil {
		t.Fatalf("write legacy candidate: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("dirty during migration"), 0o644); err != nil {
		t.Fatalf("write dirty tracked file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "migration-untracked.txt"), []byte("untracked during migration"), 0o644); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}

	migrated, err := MigrateCandidatePathDigestsAuthorized(context.Background(), epicKey, run.EpicRunID, false, func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("MigrateCandidatePathDigests: %v", err)
	}
	if migrated.PathDigestSchemaVersion != currentPathDigestSchemaVersion {
		t.Fatalf("PathDigestSchemaVersion = %d, want %d", migrated.PathDigestSchemaVersion, currentPathDigestSchemaVersion)
	}
	if migrated.BaseCommit != legacy.BaseCommit || migrated.HeadCommit != legacy.HeadCommit || !slices.Equal(migrated.EventIDs, legacy.EventIDs) {
		t.Fatalf("migration changed candidate identity: got %+v, want base/head/events from %+v", migrated, legacy)
	}
	if migrated.TrackedPathDigests == nil || migrated.UntrackedPathDigests == nil {
		t.Fatalf("migration must persist verified empty path-digest maps: got tracked=%v untracked=%v", migrated.TrackedPathDigests, migrated.UntrackedPathDigests)
	}
	if _, ok := migrated.TrackedPathDigests["seed.txt"]; !ok {
		t.Fatalf("migration omitted dirty tracked path digest: %#v", migrated.TrackedPathDigests)
	}
	if _, ok := migrated.UntrackedPathDigests["migration-untracked.txt"]; !ok {
		t.Fatalf("migration omitted untracked path digest: %#v", migrated.UntrackedPathDigests)
	}

	archived, err := os.ReadFile(filepath.Join(candidateHeadsDir(path), legacy.Digest+".json"))
	if err != nil {
		t.Fatalf("read archived legacy candidate: %v", err)
	}
	if !bytes.Equal(archived, legacyBytes) {
		t.Fatalf("archived legacy candidate bytes changed during migration")
	}
}

func TestMigrateCandidatePathDigests_DryRunWritesNothing(t *testing.T) {
	dir, headCommit := chdirProjectRoot(t)

	const epicKey = "E99"
	run, err := CaptureBase(context.Background(), epicKey)
	if err != nil {
		t.Fatalf("CaptureBase: %v", err)
	}
	legacy := IntegrationCandidate{EpicRunID: run.EpicRunID, BaseCommit: headCommit, HeadCommit: "legacy-head", EventIDs: []string{"legacy-event"}}
	legacy.Digest, err = computeDigest(legacy)
	if err != nil {
		t.Fatalf("compute legacy digest: %v", err)
	}
	legacyBytes, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy candidate: %v", err)
	}
	path := candidatePath(dir, run.EpicRunID)
	if err := os.MkdirAll(filepath.Dir(path), runDirMode); err != nil {
		t.Fatalf("create candidate directory: %v", err)
	}
	if err := os.WriteFile(path, legacyBytes, runFileMode); err != nil {
		t.Fatalf("write legacy candidate: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("dirty during dry run"), 0o644); err != nil {
		t.Fatalf("write dirty tracked file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "migration-dry-run-untracked.txt"), []byte("untracked during dry run"), 0o644); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}

	beforeCandidate, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read candidate before dry run: %v", err)
	}
	beforeFiles := countFilesUnder(t, filepath.Join(dir, ".shark"))
	migrated, err := MigrateCandidatePathDigestsAuthorized(context.Background(), epicKey, run.EpicRunID, true, nil)
	if err != nil {
		t.Fatalf("dry-run migration: %v", err)
	}
	if migrated.PathDigestSchemaVersion != currentPathDigestSchemaVersion {
		t.Fatalf("dry-run schema version = %d, want %d", migrated.PathDigestSchemaVersion, currentPathDigestSchemaVersion)
	}
	if _, ok := migrated.TrackedPathDigests["seed.txt"]; !ok {
		t.Fatalf("dry-run migration omitted dirty tracked path digest: %#v", migrated.TrackedPathDigests)
	}
	if _, ok := migrated.UntrackedPathDigests["migration-dry-run-untracked.txt"]; !ok {
		t.Fatalf("dry-run migration omitted untracked path digest: %#v", migrated.UntrackedPathDigests)
	}
	afterCandidate, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read candidate after dry run: %v", err)
	}
	if !bytes.Equal(afterCandidate, beforeCandidate) {
		t.Fatal("dry-run migration mutated the candidate")
	}
	if afterFiles := countFilesUnder(t, filepath.Join(dir, ".shark")); afterFiles != beforeFiles {
		t.Fatalf("dry-run migration changed .shark files: before=%d after=%d", beforeFiles, afterFiles)
	}
}

func TestMigrateCandidatePathDigests_AuthorizationRecheckAbortsPublish(t *testing.T) {
	dir, headCommit := chdirProjectRoot(t)
	const epicKey = "E99"
	run, err := CaptureBase(context.Background(), epicKey)
	if err != nil {
		t.Fatalf("CaptureBase: %v", err)
	}
	legacy := IntegrationCandidate{EpicRunID: run.EpicRunID, BaseCommit: headCommit, HeadCommit: "legacy-head", EventIDs: []string{"legacy-event"}}
	legacy.Digest, err = computeDigest(legacy)
	if err != nil {
		t.Fatalf("compute legacy digest: %v", err)
	}
	legacyBytes, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy candidate: %v", err)
	}
	path := candidatePath(dir, run.EpicRunID)
	if err := os.MkdirAll(filepath.Dir(path), runDirMode); err != nil {
		t.Fatalf("create candidate directory: %v", err)
	}
	if err := os.WriteFile(path, legacyBytes, runFileMode); err != nil {
		t.Fatalf("write legacy candidate: %v", err)
	}
	if _, err := MigrateCandidatePathDigestsAuthorized(context.Background(), epicKey, run.EpicRunID, false, func(context.Context) error {
		return errors.New("lease expired during migration")
	}); err == nil {
		t.Fatal("expected authorization recheck to abort migration")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read candidate after authorization rejection: %v", err)
	}
	if !bytes.Equal(after, legacyBytes) {
		t.Fatal("authorization rejection mutated the candidate")
	}
	if files := countFilesUnder(t, candidateHeadsDir(path)); files != 0 {
		t.Fatalf("authorization rejection archived %d predecessor files", files)
	}
}

func TestMigrateCandidatePathDigests_RejectsMismatchedRunEpic(t *testing.T) {
	dir, headCommit := chdirProjectRoot(t)

	const epicKey = "E99"
	run, err := CaptureBase(context.Background(), epicKey)
	if err != nil {
		t.Fatalf("CaptureBase: %v", err)
	}
	run.EpicKey = "E98"
	runBytes, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatalf("marshal mismatched run: %v", err)
	}
	if err := os.WriteFile(runRecordPath(dir, epicKey), runBytes, runFileMode); err != nil {
		t.Fatalf("write mismatched run: %v", err)
	}

	legacy := IntegrationCandidate{EpicRunID: run.EpicRunID, BaseCommit: headCommit, HeadCommit: "legacy-head", EventIDs: []string{"legacy-event"}}
	legacy.Digest, err = computeDigest(legacy)
	if err != nil {
		t.Fatalf("compute legacy digest: %v", err)
	}
	legacyBytes, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy candidate: %v", err)
	}
	path := candidatePath(dir, run.EpicRunID)
	if err := os.MkdirAll(filepath.Dir(path), runDirMode); err != nil {
		t.Fatalf("create candidate directory: %v", err)
	}
	if err := os.WriteFile(path, legacyBytes, runFileMode); err != nil {
		t.Fatalf("write legacy candidate: %v", err)
	}

	if _, err := MigrateCandidatePathDigestsAuthorized(context.Background(), epicKey, run.EpicRunID, false, func(context.Context) error { return nil }); err == nil {
		t.Fatal("expected mismatched run epic key to be rejected")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read candidate after rejection: %v", err)
	}
	if !bytes.Equal(got, legacyBytes) {
		t.Fatal("mismatched run rejection mutated the candidate")
	}
}

func TestMigrateCandidatePathDigests_RejectsInvalidDigest(t *testing.T) {
	dir, headCommit := chdirProjectRoot(t)
	const epicKey = "E99"
	run, err := CaptureBase(context.Background(), epicKey)
	if err != nil {
		t.Fatalf("CaptureBase: %v", err)
	}
	legacy := IntegrationCandidate{EpicRunID: run.EpicRunID, BaseCommit: headCommit, HeadCommit: "legacy-head", EventIDs: []string{"legacy-event"}, Digest: "not-the-real-digest"}
	path := candidatePath(dir, run.EpicRunID)
	if err := os.MkdirAll(filepath.Dir(path), runDirMode); err != nil {
		t.Fatalf("create candidate directory: %v", err)
	}
	legacyBytes, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("marshal candidate: %v", err)
	}
	if err := os.WriteFile(path, legacyBytes, runFileMode); err != nil {
		t.Fatalf("write candidate: %v", err)
	}
	if _, err := MigrateCandidatePathDigestsAuthorized(context.Background(), epicKey, run.EpicRunID, false, func(context.Context) error { return nil }); err == nil {
		t.Fatal("expected invalid candidate digest to be rejected")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read candidate after rejection: %v", err)
	}
	if !bytes.Equal(got, legacyBytes) {
		t.Fatal("invalid-digest rejection mutated the candidate")
	}
}

func TestMigrateCandidatePathDigests_RejectsFutureSchema(t *testing.T) {
	dir, headCommit := chdirProjectRoot(t)
	const epicKey = "E99"
	run, err := CaptureBase(context.Background(), epicKey)
	if err != nil {
		t.Fatalf("CaptureBase: %v", err)
	}
	future := IntegrationCandidate{EpicRunID: run.EpicRunID, BaseCommit: headCommit, HeadCommit: "future-head", EventIDs: []string{"future-event"}, PathDigestSchemaVersion: currentPathDigestSchemaVersion + 1, TrackedPathDigests: map[string]string{}, UntrackedPathDigests: map[string]string{}}
	future.Digest, err = computeDigest(future)
	if err != nil {
		t.Fatalf("compute future digest: %v", err)
	}
	path := candidatePath(dir, run.EpicRunID)
	if err := os.MkdirAll(filepath.Dir(path), runDirMode); err != nil {
		t.Fatalf("create candidate directory: %v", err)
	}
	data, err := json.MarshalIndent(future, "", "  ")
	if err != nil {
		t.Fatalf("marshal future candidate: %v", err)
	}
	if err := os.WriteFile(path, data, runFileMode); err != nil {
		t.Fatalf("write future candidate: %v", err)
	}
	if _, err := MigrateCandidatePathDigestsAuthorized(context.Background(), epicKey, run.EpicRunID, false, func(context.Context) error { return nil }); err == nil {
		t.Fatal("expected future schema to be rejected")
	}
}

func TestMigrateCandidatePathDigests_AlreadyMigratedIsIdempotent(t *testing.T) {
	dir, headCommit := chdirProjectRoot(t)
	const epicKey = "E99"
	run, err := CaptureBase(context.Background(), epicKey)
	if err != nil {
		t.Fatalf("CaptureBase: %v", err)
	}
	migrated := IntegrationCandidate{EpicRunID: run.EpicRunID, BaseCommit: headCommit, HeadCommit: "current-head", EventIDs: []string{"current-event"}, PathDigestSchemaVersion: currentPathDigestSchemaVersion, TrackedPathDigests: map[string]string{}, UntrackedPathDigests: map[string]string{}}
	migrated.Digest, err = computeDigest(migrated)
	if err != nil {
		t.Fatalf("compute migrated digest: %v", err)
	}
	path := candidatePath(dir, run.EpicRunID)
	if err := os.MkdirAll(filepath.Dir(path), runDirMode); err != nil {
		t.Fatalf("create candidate directory: %v", err)
	}
	data, err := json.MarshalIndent(migrated, "", "  ")
	if err != nil {
		t.Fatalf("marshal migrated candidate: %v", err)
	}
	if err := os.WriteFile(path, data, runFileMode); err != nil {
		t.Fatalf("write migrated candidate: %v", err)
	}
	got, err := MigrateCandidatePathDigestsAuthorized(context.Background(), epicKey, run.EpicRunID, false, func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("idempotent migration: %v", err)
	}
	if got.Digest != migrated.Digest || got.PathDigestSchemaVersion != currentPathDigestSchemaVersion {
		t.Fatalf("idempotent migration changed candidate: got %+v, want %+v", got, migrated)
	}
	if _, err := os.Stat(filepath.Join(candidateHeadsDir(path), migrated.Digest+".json")); !os.IsNotExist(err) {
		t.Fatalf("idempotent migration unexpectedly archived current candidate: err=%v", err)
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
	if _, err := UpdateCandidate(context.Background(), epicRunID, seedEvent); err != nil {
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
		if _, err := UpdateCandidate(context.Background(), epicRunID, concurrentEvent); err != nil {
			t.Errorf("concurrent UpdateCandidate during pause: %v", err)
		}
	}
	t.Cleanup(func() { updateCandidateTestHook = nil })

	targetEvent, err := RecordEvent(epicRunID, "E34-F32", "commit-target", nil, nil)
	if err != nil {
		t.Fatalf("target RecordEvent: %v", err)
	}

	result, err := UpdateCandidate(context.Background(), epicRunID, targetEvent)
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
	if _, err := UpdateCandidate(context.Background(), epicRunID, seedEvent); err != nil {
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
		if _, err := UpdateCandidate(context.Background(), epicRunID, ev); err != nil {
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

	_, err = UpdateCandidate(context.Background(), epicRunID, targetEvent)
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
	if _, err := UpdateCandidate(context.Background(), epicRunID, event); err != nil {
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
	first, err := UpdateCandidate(context.Background(), epicRunID, firstEvent)
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

	second, err := UpdateCandidate(context.Background(), epicRunID, secondEvent)
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
	first, err := UpdateCandidate(context.Background(), epicRunID, firstEvent)
	if err != nil {
		t.Fatalf("first UpdateCandidate: %v", err)
	}

	secondEvent, err := RecordEvent(epicRunID, "E34-F54", "commit-b", nil, nil)
	if err != nil {
		t.Fatalf("second RecordEvent: %v", err)
	}
	if _, err := UpdateCandidate(context.Background(), epicRunID, secondEvent); err != nil {
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

func TestArchiveCandidateHead_RejectsConflictingExistingBytes(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	path := candidatePath(dir, "run-archive-conflict")
	want := []byte(`{"epic_run_id":"run-archive-conflict"}`)
	conflicting := []byte(`{"epic_run_id":"different"}`)
	if err := archiveCandidateHead(path, "digest-x", want); err != nil {
		t.Fatalf("first archiveCandidateHead: %v", err)
	}
	archivePath := filepath.Join(candidateHeadsDir(path), "digest-x.json")
	if err := os.WriteFile(archivePath, conflicting, runFileMode); err != nil {
		t.Fatalf("tamper archived bytes: %v", err)
	}
	if err := archiveCandidateHead(path, "digest-x", want); err == nil {
		t.Fatal("expected conflicting archived bytes to be rejected")
	}
	got, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read conflicting archive: %v", err)
	}
	if !bytes.Equal(got, conflicting) {
		t.Fatal("conflicting archive bytes were overwritten")
	}
}

// TestUpdateCandidate_DirtyPathDigests covers TC-019 (T-E34-F08-016): a
// dirty tracked file and an untracked candidate path each get a recorded
// digest in the resulting candidate, and a fixture consumer (mirroring
// integration_review's own read) retrieves both correctly. Per TC-019's
// Caller-Path Contract, this drives UpdateCandidate's real production path
// against a real temp working tree with staged, dirty, and untracked
// files — never a hand-built path/digest list.
func TestUpdateCandidate_DirtyPathDigests(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-candidate-dirty-digests"

	// Dirty tracked path: seed.txt was committed by initTestGitRepo, then
	// staged here with new content — a real worktree change `git status`
	// reports as a tracked modification, not untracked.
	dirtyContent := []byte("seed-modified-and-staged")
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), dirtyContent, 0o644); err != nil {
		t.Fatalf("write dirty tracked file: %v", err)
	}
	runGit(t, dir, "add", "seed.txt")

	// Untracked candidate path: a brand-new file never added to git.
	untrackedContent := []byte("brand-new-untracked-content")
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), untrackedContent, 0o644); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}

	event, err := RecordEvent(epicRunID, "E34-F60", "commit-dirty", nil, nil)
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	candidate, err := UpdateCandidate(context.Background(), epicRunID, event)
	if err != nil {
		t.Fatalf("UpdateCandidate: %v", err)
	}

	wantDirtyDigest := sha256Hex(dirtyContent)
	wantUntrackedDigest := sha256Hex(untrackedContent)
	if len(candidate.TrackedPathDigests) != 1 {
		t.Fatalf("TrackedPathDigests = %v, want exactly seed.txt (runtime .shark files must be excluded)", candidate.TrackedPathDigests)
	}
	if len(candidate.UntrackedPathDigests) != 1 {
		t.Fatalf("UntrackedPathDigests = %v, want exactly untracked.txt (runtime .shark files must be excluded)", candidate.UntrackedPathDigests)
	}

	t.Run("dirty tracked file", func(t *testing.T) {
		got, ok := candidate.TrackedPathDigests["seed.txt"]
		if !ok {
			t.Fatalf("candidate.TrackedPathDigests missing seed.txt; got %v", candidate.TrackedPathDigests)
		}
		if got != wantDirtyDigest {
			t.Fatalf("TrackedPathDigests[seed.txt] = %q, want %q", got, wantDirtyDigest)
		}
	})

	t.Run("untracked candidate path", func(t *testing.T) {
		got, ok := candidate.UntrackedPathDigests["untracked.txt"]
		if !ok {
			t.Fatalf("candidate.UntrackedPathDigests missing untracked.txt; got %v", candidate.UntrackedPathDigests)
		}
		if got != wantUntrackedDigest {
			t.Fatalf("UntrackedPathDigests[untracked.txt] = %q, want %q", got, wantUntrackedDigest)
		}
	})
}

func TestComputeDirtyPathDigests_RenameDeleteAndCleanTree(t *testing.T) {
	t.Run("clean tree", func(t *testing.T) {
		dir, _ := chdirProjectRoot(t)
		requireDirtyPathDigests(t, dir, nil, nil)
	})

	t.Run("rename records only the current path", func(t *testing.T) {
		dir, _ := chdirProjectRoot(t)
		runGit(t, dir, "mv", "seed.txt", "renamed.txt")
		requireDirtyPathDigests(t, dir, map[string]string{"renamed.txt": sha256Hex([]byte("seed"))}, nil)
	})

	t.Run("deleted tracked file is omitted", func(t *testing.T) {
		dir, _ := chdirProjectRoot(t)
		if err := os.Remove(filepath.Join(dir, "seed.txt")); err != nil {
			t.Fatalf("remove seed.txt: %v", err)
		}
		requireDirtyPathDigests(t, dir, nil, nil)
	})
}

func requireDirtyPathDigests(t *testing.T, dir string, wantTracked, wantUntracked map[string]string) {
	t.Helper()
	tracked, untracked, err := computeDirtyPathDigests(context.Background(), dir)
	if err != nil {
		t.Fatalf("computeDirtyPathDigests: %v", err)
	}
	if !equalPathDigests(tracked, wantTracked) || !equalPathDigests(untracked, wantUntracked) {
		t.Fatalf("dirty path digests = tracked %v, untracked %v; want tracked %v, untracked %v", tracked, untracked, wantTracked, wantUntracked)
	}
}

func equalPathDigests(got, want map[string]string) bool {
	if len(got) != len(want) {
		return false
	}
	for path, digest := range want {
		if got[path] != digest {
			return false
		}
	}
	return true
}

// sha256Hex computes data's sha256 hex digest — the same convention
// computeDirtyPathDigests uses, applied here to the test's own expected
// content so the assertion never hand-codes a digest literal.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// TestUpdateCandidate_TransientPublishFailureDoesNotPermanentlyBlock covers
// the code-review kickback on task T-E34-F08-006 (defect-class-sweep): a
// claim file that wins attemptUpdateCandidate's compare-and-swap was, before
// this fix, retained forever unconditionally — including when a later,
// genuinely fallible step (archiving or the final os.Rename) failed after
// the claim was won. Since a failed attempt never advances the on-disk
// candidate's digest, every later attempt (this call's own retry, and every
// independent call after it) recomputes the identical expectedDigest and
// loses the same claim's os.Link forever, reporting a spurious
// *CandidateConflictError even though no other writer ever won or published
// under that claim — a transient I/O error permanently deadlocking the
// candidate transition.
//
// This test forces a real (not mocked) os.Rename failure by stripping write
// permission from the run directory at the archiveTestHook seam — after
// archiving has already completed, exactly where the class defect bites —
// matching this repo's existing chmod-based fault-injection convention
// (internal/runner/liveness_test.go TC-023, internal/services/edit_service_test.go).
// It then restores permissions and proves a fresh attempt against the
// identical, unchanged expected digest succeeds rather than being
// permanently blocked by the abandoned claim.
func TestUpdateCandidate_TransientPublishFailureDoesNotPermanentlyBlock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-bit fault injection is not meaningful on windows")
	}
	if os.Getuid() == 0 {
		t.Skip("running as root; permission checks are not enforced")
	}

	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-candidate-transient-failure"

	firstEvent, err := RecordEvent(epicRunID, "E34-F70", "commit-first", nil, nil)
	if err != nil {
		t.Fatalf("first RecordEvent: %v", err)
	}
	if _, err := UpdateCandidate(context.Background(), epicRunID, firstEvent); err != nil {
		t.Fatalf("first UpdateCandidate: %v", err)
	}

	secondEvent, err := RecordEvent(epicRunID, "E34-F71", "commit-second", nil, nil)
	if err != nil {
		t.Fatalf("second RecordEvent: %v", err)
	}

	runDir := filepath.Join(dir, ".shark", "runs", epicRunID)

	archiveTestHook = func() {
		// Archiving of the first candidate's head has already completed by
		// this point (it fires after archiveCandidateHead returns); strip
		// write permission on the run directory so the rename that follows
		// fails with a genuine, transient-shaped I/O error.
		if err := os.Chmod(runDir, 0o555); err != nil {
			t.Fatalf("chmod run dir read-only: %v", err)
		}
	}
	t.Cleanup(func() {
		archiveTestHook = nil
		_ = os.Chmod(runDir, 0o755)
	})

	_, err = UpdateCandidate(context.Background(), epicRunID, secondEvent)
	if err == nil {
		t.Fatal("expected the induced rename failure to surface as an error")
	}
	var conflict *CandidateConflictError
	if errors.As(err, &conflict) {
		t.Fatalf("expected a non-conflict publish error (the induced rename failure), got *CandidateConflictError: %v", err)
	}
	// Pin the assertion to the rename branch specifically (attemptUpdateCandidate
	// wraps os.Rename's failure as "integration: publish candidate at ...").
	if !strings.Contains(err.Error(), "publish candidate") {
		t.Fatalf("expected the rename-failure error to mention publishing the candidate, got: %v", err)
	}

	// The transient condition is over: restore permissions before retrying.
	archiveTestHook = nil
	if err := os.Chmod(runDir, 0o755); err != nil {
		t.Fatalf("restore run dir permissions: %v", err)
	}

	// A fresh attempt against the identical, unchanged expected digest
	// (the first attempt never advanced the on-disk candidate) must now
	// succeed: the abandoned claim from the failed attempt above must not
	// permanently occupy that digest's transition slot.
	result, err := UpdateCandidate(context.Background(), epicRunID, secondEvent)
	if err != nil {
		t.Fatalf("retry after transient failure cleared: expected success, got permanently blocked: %v", err)
	}
	want := []string{firstEvent.EventID, secondEvent.EventID}
	sort.Strings(want)
	got := append([]string(nil), result.EventIDs...)
	sort.Strings(got)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("final EventIDs = %v, want %v", got, want)
	}
}

// TestUpdateCandidate_TransientArchiveFailureDoesNotPermanentlyBlock is the
// sibling of TestUpdateCandidate_TransientPublishFailureDoesNotPermanentlyBlock
// covering the *other* fallible step the claim-rollback guards: a failure in
// archiveCandidateHead itself (before the rename is ever attempted). The
// fault is injected by pre-chmodding integration-heads/ read-only —
// os.MkdirAll is a no-op on an already-existing directory regardless of its
// mode, so archiveCandidateHead's own os.OpenFile(O_CREATE) for its temp file
// is what fails, a real permission error rather than a mocked one.
func TestUpdateCandidate_TransientArchiveFailureDoesNotPermanentlyBlock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-bit fault injection is not meaningful on windows")
	}
	if os.Getuid() == 0 {
		t.Skip("running as root; permission checks are not enforced")
	}

	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-candidate-transient-archive-failure"

	firstEvent, err := RecordEvent(epicRunID, "E34-F72", "commit-first", nil, nil)
	if err != nil {
		t.Fatalf("first RecordEvent: %v", err)
	}
	if _, err := UpdateCandidate(context.Background(), epicRunID, firstEvent); err != nil {
		t.Fatalf("first UpdateCandidate: %v", err)
	}

	secondEvent, err := RecordEvent(epicRunID, "E34-F73", "commit-second", nil, nil)
	if err != nil {
		t.Fatalf("second RecordEvent: %v", err)
	}

	headsDir := candidateHeadsDir(candidatePath(dir, epicRunID))
	if err := os.MkdirAll(headsDir, 0o755); err != nil {
		t.Fatalf("pre-create integration-heads dir: %v", err)
	}
	if err := os.Chmod(headsDir, 0o555); err != nil {
		t.Fatalf("chmod integration-heads dir read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(headsDir, 0o755) })

	_, err = UpdateCandidate(context.Background(), epicRunID, secondEvent)
	if err == nil {
		t.Fatal("expected the induced archive failure to surface as an error")
	}
	var conflict *CandidateConflictError
	if errors.As(err, &conflict) {
		t.Fatalf("expected a non-conflict archive error (the induced archive failure), got *CandidateConflictError: %v", err)
	}
	if !strings.Contains(err.Error(), "archived-head") {
		t.Fatalf("expected the archive-failure error to mention the archived-head file, got: %v", err)
	}

	// The transient condition is over: restore permissions before retrying.
	if err := os.Chmod(headsDir, 0o755); err != nil {
		t.Fatalf("restore integration-heads dir permissions: %v", err)
	}

	// A fresh attempt against the identical, unchanged expected digest must
	// now succeed: the abandoned claim from the failed archive attempt above
	// must not permanently occupy that digest's transition slot.
	result, err := UpdateCandidate(context.Background(), epicRunID, secondEvent)
	if err != nil {
		t.Fatalf("retry after transient failure cleared: expected success, got permanently blocked: %v", err)
	}
	want := []string{firstEvent.EventID, secondEvent.EventID}
	sort.Strings(want)
	got := append([]string(nil), result.EventIDs...)
	sort.Strings(got)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("final EventIDs = %v, want %v", got, want)
	}
}
