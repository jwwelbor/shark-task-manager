package integration

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Task T-E34-F08-013 covers test-plan.md TC-018: pinned-base fixture
// histories for an independently squash-merged feature, unrelated
// interleaved commits, a rebase, and a missing/unreachable base (one
// subtest each) — real git commands against a temp repository, no mock,
// asserting no case infers scope from `merge-base HEAD main`.

// runGit runs a git command in dir and returns its trimmed combined
// output, failing the test on error.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// writeAndCommit writes filename/content in dir and commits it, returning
// the new commit's full hash.
func writeAndCommit(t *testing.T, dir, filename, content, message string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
	runGit(t, dir, "add", filename)
	runGit(t, dir, "commit", "-q", "-m", message)
	return runGit(t, dir, "rev-parse", "HEAD")
}

// TestAnalyzeHistory_MissingBaseFailsClosed covers TC-018's missing/
// unreachable-base subtest and spec.md AC-T1: a base commit that does not
// exist in the repository fails closed with a typed *UnreachableBaseError,
// never silently inferring scope from `git merge-base HEAD main`.
func TestAnalyzeHistory_MissingBaseFailsClosed(t *testing.T) {
	dir, head := chdirProjectRoot(t)

	const missingBase = "0000000000000000000000000000000000dead"
	_, err := AnalyzeHistory(dir, "run-missing-base", missingBase, head, nil)
	if err == nil {
		t.Fatal("expected an error for a missing base commit")
	}
	var unreachable *UnreachableBaseError
	if !errors.As(err, &unreachable) {
		t.Fatalf("expected *UnreachableBaseError, got %T: %v", err, err)
	}
	if unreachable.Base != missingBase {
		t.Errorf("Base = %q, want %q", unreachable.Base, missingBase)
	}
}

// TestAnalyzeHistory_UnreachableBaseFailsClosed covers the "exists but not
// an ancestor of head" half of AC-T1: a real commit that is not on head's
// ancestry (a discarded branch) still fails closed rather than being
// treated as reachable.
func TestAnalyzeHistory_UnreachableBaseFailsClosed(t *testing.T) {
	dir, head := chdirProjectRoot(t)

	// A real commit, but on a branch never merged into head.
	runGit(t, dir, "checkout", "-q", "-b", "discarded")
	discarded := writeAndCommit(t, dir, "discarded.txt", "discarded work", "discarded work")
	runGit(t, dir, "checkout", "-q", "-")

	_, err := AnalyzeHistory(dir, "run-unreachable-base", discarded, head, nil)
	if err == nil {
		t.Fatal("expected an error for a base that is not an ancestor of head")
	}
	var unreachable *UnreachableBaseError
	if !errors.As(err, &unreachable) {
		t.Fatalf("expected *UnreachableBaseError, got %T: %v", err, err)
	}
}

// TestAnalyzeHistory_CleanHistoryAllAccounted is the clean-history baseline
// every other subtest is diffed against (test-plan.md TC-018's Caller-Path
// Contract counter-factual): a feature commit directly in the base..head
// range is accounted for, with no replacement and nothing interleaved.
func TestAnalyzeHistory_CleanHistoryAllAccounted(t *testing.T) {
	dir, base := chdirProjectRoot(t)

	featureCommit := writeAndCommit(t, dir, "feature.txt", "feature work", "feature work")
	head := runGit(t, dir, "rev-parse", "HEAD")

	events := []IntegrationEvent{
		{EpicRunID: "run-clean", FeatureKey: "E90-F00", FeatureCommit: featureCommit},
	}

	inv, err := AnalyzeHistory(dir, "run-clean", base, head, events)
	if err != nil {
		t.Fatalf("AnalyzeHistory: %v", err)
	}
	if len(inv.Replacements) != 0 {
		t.Errorf("expected no replacements, got %d", len(inv.Replacements))
	}
	if len(inv.Interleaved) != 0 {
		t.Errorf("expected no interleaved commits, got %v", inv.Interleaved)
	}
}

// TestAnalyzeHistory_InterleavedCommitsRemainVisible covers TC-018's
// unrelated-interleaved-commits subtest and spec.md AC-T3: a commit in the
// base..head range not attributable to any recorded feature event remains
// visible in Interleaved rather than being silently included in or
// excluded from the inventory.
func TestAnalyzeHistory_InterleavedCommitsRemainVisible(t *testing.T) {
	dir, base := chdirProjectRoot(t)

	featureCommit := writeAndCommit(t, dir, "feature.txt", "feature work", "feature work")
	unrelated := writeAndCommit(t, dir, "unrelated.txt", "unrelated change", "unrelated change")
	head := runGit(t, dir, "rev-parse", "HEAD")

	events := []IntegrationEvent{
		{EpicRunID: "run-interleaved", FeatureKey: "E90-F01", FeatureCommit: featureCommit},
	}

	inv, err := AnalyzeHistory(dir, "run-interleaved", base, head, events)
	if err != nil {
		t.Fatalf("AnalyzeHistory: %v", err)
	}
	if len(inv.Replacements) != 0 {
		t.Fatalf("expected no replacements, got %d", len(inv.Replacements))
	}
	if len(inv.Interleaved) != 1 || inv.Interleaved[0] != unrelated {
		t.Fatalf("Interleaved = %v, want [%s]", inv.Interleaved, unrelated)
	}
}

// TestAnalyzeHistory_SquashMergedFeatureGetsReplacementRecord covers
// TC-018's independently-squash-merged-feature subtest and spec.md AC-T2:
// a feature squash-merged into the base branch no longer has its original
// (pre-squash) commit hash anywhere in the base..head range, but its
// change is content-identical to the squash commit's own diff — detected
// and recorded as an explicit ReplacementRecord linked by
// PriorRecordDigest, never a silent base rewrite.
func TestAnalyzeHistory_SquashMergedFeatureGetsReplacementRecord(t *testing.T) {
	dir, base := chdirProjectRoot(t)
	mainBranch := runGit(t, dir, "rev-parse", "--abbrev-ref", "HEAD")

	runGit(t, dir, "checkout", "-q", "-b", "feature/squash")
	writeAndCommit(t, dir, "f1.txt", "part one", "feature part one")
	featureTip := writeAndCommit(t, dir, "f2.txt", "part two", "feature part two")

	runGit(t, dir, "checkout", "-q", mainBranch)
	runGit(t, dir, "merge", "--squash", "feature/squash")
	runGit(t, dir, "commit", "-q", "-m", "squash-merge feature/squash")
	head := runGit(t, dir, "rev-parse", "HEAD")

	events := []IntegrationEvent{
		{EpicRunID: "run-squash", FeatureKey: "E90-F02", FeatureCommit: featureTip},
	}

	inv, err := AnalyzeHistory(dir, "run-squash", base, head, events)
	if err != nil {
		t.Fatalf("AnalyzeHistory: %v", err)
	}
	if len(inv.Interleaved) != 0 {
		t.Fatalf("expected no interleaved commits, got %v", inv.Interleaved)
	}
	if len(inv.Replacements) != 1 {
		t.Fatalf("expected exactly one replacement record, got %d", len(inv.Replacements))
	}

	rec := inv.Replacements[0]
	if rec.OriginalCommit != featureTip {
		t.Errorf("OriginalCommit = %q, want %q", rec.OriginalCommit, featureTip)
	}
	if rec.ReplacementCommit != head {
		t.Errorf("ReplacementCommit = %q, want %q", rec.ReplacementCommit, head)
	}
	if rec.PriorRecordDigest == "" {
		t.Error("PriorRecordDigest is empty")
	}
	if rec.RecordDigest == "" {
		t.Error("RecordDigest is empty")
	}
	if base != inv.Base || head != inv.Head {
		t.Errorf("inventory Base/Head = %q/%q, want %q/%q", inv.Base, inv.Head, base, head)
	}

	assertReplacementPersisted(t, dir, "run-squash", rec)
}

// TestAnalyzeHistory_RebaseOntoUnrelatedWorkGetsReplacementRecord covers
// TC-018's rebase subtest and spec.md AC-T2: a feature branch rebased onto
// unrelated intervening work (a realistic rebase, not a no-op) changes the
// feature commit's hash and parent, but AnalyzeHistory still identifies it
// by content and records an explicit ReplacementRecord — the base itself
// is never silently rewritten to the new commit.
func TestAnalyzeHistory_RebaseOntoUnrelatedWorkGetsReplacementRecord(t *testing.T) {
	dir, base := chdirProjectRoot(t)
	mainBranch := runGit(t, dir, "rev-parse", "--abbrev-ref", "HEAD")

	// Feature branch created off base, before any unrelated work lands.
	runGit(t, dir, "checkout", "-q", "-b", "feature/rebase")
	originalFeatureCommit := writeAndCommit(t, dir, "feature.txt", "feature work", "feature work")

	// Unrelated work lands on main, independent of the feature.
	runGit(t, dir, "checkout", "-q", mainBranch)
	unrelated := writeAndCommit(t, dir, "unrelated.txt", "unrelated change", "unrelated change")

	// The feature branch is rebased onto that unrelated work, rewriting its
	// commit hash and parent.
	runGit(t, dir, "checkout", "-q", "feature/rebase")
	runGit(t, dir, "rebase", mainBranch)
	rebasedFeatureCommit := runGit(t, dir, "rev-parse", "HEAD")
	if rebasedFeatureCommit == originalFeatureCommit {
		t.Fatal("rebase did not change the feature commit's hash — fixture is not exercising a real rewrite")
	}

	runGit(t, dir, "checkout", "-q", mainBranch)
	runGit(t, dir, "merge", "-q", "--ff-only", "feature/rebase")
	head := runGit(t, dir, "rev-parse", "HEAD")
	if head != rebasedFeatureCommit {
		t.Fatalf("head = %q, want %q", head, rebasedFeatureCommit)
	}

	events := []IntegrationEvent{
		{EpicRunID: "run-rebase", FeatureKey: "E90-F03", FeatureCommit: originalFeatureCommit},
	}

	inv, err := AnalyzeHistory(dir, "run-rebase", base, head, events)
	if err != nil {
		t.Fatalf("AnalyzeHistory: %v", err)
	}
	// The unrelated commit the feature was rebased onto is itself now part
	// of the base..head range (that is what a rebase onto intervening work
	// means) but is not attributable to this feature's own event — it must
	// remain visible as interleaved, not silently absorbed into the
	// feature's replacement record (AC-T3), and not silently dropped.
	if len(inv.Interleaved) != 1 || inv.Interleaved[0] != unrelated {
		t.Fatalf("Interleaved = %v, want [%s]", inv.Interleaved, unrelated)
	}
	if len(inv.Replacements) != 1 {
		t.Fatalf("expected exactly one replacement record, got %d", len(inv.Replacements))
	}

	rec := inv.Replacements[0]
	if rec.OriginalCommit != originalFeatureCommit {
		t.Errorf("OriginalCommit = %q, want %q", rec.OriginalCommit, originalFeatureCommit)
	}
	if rec.ReplacementCommit != rebasedFeatureCommit {
		t.Errorf("ReplacementCommit = %q, want %q", rec.ReplacementCommit, rebasedFeatureCommit)
	}
	if rec.ReplacementCommit == unrelated {
		t.Error("replacement incorrectly resolved to the unrelated commit, not the rebased feature commit")
	}

	assertReplacementPersisted(t, dir, "run-rebase", rec)
}

// assertReplacementPersisted verifies rec was durably written to its
// content-addressed path under epicRunID's integration-history directory,
// with matching bytes on disk.
func assertReplacementPersisted(t *testing.T, projectRoot, epicRunID string, rec *ReplacementRecord) {
	t.Helper()
	data, err := os.ReadFile(replacementRecordPath(projectRoot, epicRunID, rec.RecordDigest))
	if err != nil {
		t.Fatalf("read persisted replacement record: %v", err)
	}
	var persisted ReplacementRecord
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("unmarshal persisted replacement record: %v", err)
	}
	if persisted.RecordDigest != rec.RecordDigest {
		t.Errorf("persisted RecordDigest = %q, want %q", persisted.RecordDigest, rec.RecordDigest)
	}
	if persisted.OriginalCommit != rec.OriginalCommit {
		t.Errorf("persisted OriginalCommit = %q, want %q", persisted.OriginalCommit, rec.OriginalCommit)
	}
}

// TestVerifyBaseReachable_EmptyBase covers VerifyBaseReachable's own
// direct contract (used by both AnalyzeHistory and, via backfill.go's
// verifyCommitReachable, Backfill's --base validation): an empty base
// fails closed rather than being treated as "no base to check."
func TestVerifyBaseReachable_EmptyBase(t *testing.T) {
	dir, head := chdirProjectRoot(t)

	err := VerifyBaseReachable(dir, "", head)
	var unreachable *UnreachableBaseError
	if !errors.As(err, &unreachable) {
		t.Fatalf("expected *UnreachableBaseError for an empty base, got %T: %v", err, err)
	}
}

// TestVerifyBaseReachable_NoHeadOnlyChecksExistence covers the
// backfill-time shape of the shared check (head == ""): a real commit
// passes even though no head is supplied to compare ancestry against —
// this is exactly Backfill's own call shape, which has no head yet.
func TestVerifyBaseReachable_NoHeadOnlyChecksExistence(t *testing.T) {
	dir, commit := chdirProjectRoot(t)

	if err := VerifyBaseReachable(dir, commit, ""); err != nil {
		t.Fatalf("VerifyBaseReachable with no head: %v", err)
	}
}

// TestAnalyzeHistory_EmptyHeadRejected: unlike VerifyBaseReachable (which
// legitimately allows an empty head for Backfill's pre-candidate call
// shape), AnalyzeHistory always requires an explicit head. `git rev-list
// base..` (an omitted upper bound) resolves to the current HEAD, so a
// caller that failed to bind a candidate head would otherwise silently get
// a review scope inferred from whatever HEAD happens to be at call time —
// exactly the "inferred scope" TC-018 forbids, just sourced from a bug
// rather than from `merge-base HEAD main`.
func TestAnalyzeHistory_EmptyHeadRejected(t *testing.T) {
	dir, base := chdirProjectRoot(t)

	// If AnalyzeHistory ever silently fell back to `rev-list base..`
	// resolving "" to HEAD, this extra commit — unrelated to any recorded
	// event — would appear in the inventory even though no head was ever
	// supplied.
	writeAndCommit(t, dir, "after.txt", "commit made after the call", "after")

	_, err := AnalyzeHistory(dir, "run-empty-head", base, "", nil)
	if err == nil {
		t.Fatal("expected an error for an empty head")
	}
}

// TestAnalyzeHistory_ReplacementDetectionIsIdempotent covers the
// idempotency of the persisted ReplacementRecord: calling AnalyzeHistory
// twice for the identical squash-merged fixture must produce the identical
// RecordDigest and write exactly one file, not one per call. RecordDigest
// must not depend on DetectedAt — mirroring event.go's deriveEventID,
// which deliberately excludes RecordedAt so a retried write is idempotent.
func TestAnalyzeHistory_ReplacementDetectionIsIdempotent(t *testing.T) {
	dir, base := chdirProjectRoot(t)
	mainBranch := runGit(t, dir, "rev-parse", "--abbrev-ref", "HEAD")

	runGit(t, dir, "checkout", "-q", "-b", "feature/squash-idempotent")
	featureTip := writeAndCommit(t, dir, "f1.txt", "part one", "feature part one")

	runGit(t, dir, "checkout", "-q", mainBranch)
	runGit(t, dir, "merge", "--squash", "feature/squash-idempotent")
	runGit(t, dir, "commit", "-q", "-m", "squash-merge feature/squash-idempotent")
	head := runGit(t, dir, "rev-parse", "HEAD")

	events := []IntegrationEvent{
		{EpicRunID: "run-idempotent", FeatureKey: "E90-F09", FeatureCommit: featureTip},
	}

	first, err := AnalyzeHistory(dir, "run-idempotent", base, head, events)
	if err != nil {
		t.Fatalf("first AnalyzeHistory: %v", err)
	}
	second, err := AnalyzeHistory(dir, "run-idempotent", base, head, events)
	if err != nil {
		t.Fatalf("second AnalyzeHistory: %v", err)
	}

	if len(first.Replacements) != 1 || len(second.Replacements) != 1 {
		t.Fatalf("expected exactly one replacement each call, got %d and %d", len(first.Replacements), len(second.Replacements))
	}
	if first.Replacements[0].RecordDigest != second.Replacements[0].RecordDigest {
		t.Fatalf("RecordDigest differs across calls: %q vs %q", first.Replacements[0].RecordDigest, second.Replacements[0].RecordDigest)
	}

	historyDir := filepath.Join(dir, ".shark", "runs", "run-idempotent", "integration-history")
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		t.Fatalf("read history dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one persisted replacement record, got %d", len(entries))
	}
}

// TestAnalyzeHistory_UnaccountedFeatureCommitSurfaces covers the third
// silent-drop path: a recorded event whose FeatureCommit resolves to
// nothing findable — not in range, not an ancestor, and no content-
// identical replacement anywhere in the range — must appear in
// Unaccounted rather than vanishing from the inventory. feature.md
// REQ-F-005 requires every completed/staged feature be included in the
// review inventory; a silently-dropped entry here is exactly the "missing
// feature event" architecture.md says integration_review must reject, and
// it can only reject what the inventory exposes.
func TestAnalyzeHistory_UnaccountedFeatureCommitSurfaces(t *testing.T) {
	dir, base := chdirProjectRoot(t)

	otherCommit := writeAndCommit(t, dir, "other.txt", "other work", "other work")
	head := runGit(t, dir, "rev-parse", "HEAD")

	const neverExisted = "0000000000000000000000000000000000beef"
	events := []IntegrationEvent{
		{EpicRunID: "run-unaccounted", FeatureKey: "E90-F04", FeatureCommit: neverExisted},
	}

	inv, err := AnalyzeHistory(dir, "run-unaccounted", base, head, events)
	if err != nil {
		t.Fatalf("AnalyzeHistory: %v", err)
	}
	if len(inv.Replacements) != 0 {
		t.Fatalf("expected no replacements, got %d", len(inv.Replacements))
	}
	if len(inv.Unaccounted) != 1 || inv.Unaccounted[0].FeatureCommit != neverExisted {
		t.Fatalf("Unaccounted = %+v, want one entry for %s", inv.Unaccounted, neverExisted)
	}
	if len(inv.Interleaved) != 1 || inv.Interleaved[0] != otherCommit {
		t.Fatalf("Interleaved = %v, want [%s]", inv.Interleaved, otherCommit)
	}
}

// TestAnalyzeHistory_CommitEqualToBaseIsAccounted covers the defensive
// edge case of a recorded event whose FeatureCommit is literally the
// epic's own base (a no-op feature): accounted directly, no error, no
// spurious replacement lookup against its own unchanged state.
func TestAnalyzeHistory_CommitEqualToBaseIsAccounted(t *testing.T) {
	dir, base := chdirProjectRoot(t)
	head := writeAndCommit(t, dir, "other.txt", "other work", "other work")

	events := []IntegrationEvent{
		{EpicRunID: "run-noop-feature", FeatureKey: "E90-F05", FeatureCommit: base},
	}

	inv, err := AnalyzeHistory(dir, "run-noop-feature", base, head, events)
	if err != nil {
		t.Fatalf("AnalyzeHistory: %v", err)
	}
	if len(inv.Unaccounted) != 0 {
		t.Fatalf("expected no unaccounted events, got %+v", inv.Unaccounted)
	}
	if len(inv.Replacements) != 0 {
		t.Fatalf("expected no replacements, got %d", len(inv.Replacements))
	}
}

// TestAnalyzeHistory_EmptyCommitInRangeDoesNotBreakReplacementDetection
// covers the empty-diff edge findRewrittenReplacement's patch-id
// comparison must not mishandle: an --allow-empty commit anywhere in the
// base..head range introduces an empty diff, which is "not comparable,"
// not a git invocation failure — it must neither abort the whole
// AnalyzeHistory call nor spuriously match another empty-diff commit via
// two empty patch-id strings comparing equal. The squash-merged feature's
// own replacement must still be found.
func TestAnalyzeHistory_EmptyCommitInRangeDoesNotBreakReplacementDetection(t *testing.T) {
	dir, base := chdirProjectRoot(t)
	mainBranch := runGit(t, dir, "rev-parse", "--abbrev-ref", "HEAD")

	runGit(t, dir, "checkout", "-q", "-b", "feature/squash-empty")
	featureTip := writeAndCommit(t, dir, "f1.txt", "part one", "feature part one")

	runGit(t, dir, "checkout", "-q", mainBranch)
	runGit(t, dir, "merge", "--squash", "feature/squash-empty")
	runGit(t, dir, "commit", "-q", "-m", "squash-merge feature/squash-empty")
	squashCommit := runGit(t, dir, "rev-parse", "HEAD")

	runGit(t, dir, "commit", "-q", "--allow-empty", "-m", "empty marker")
	head := runGit(t, dir, "rev-parse", "HEAD")

	events := []IntegrationEvent{
		{EpicRunID: "run-squash-empty", FeatureKey: "E90-F06", FeatureCommit: featureTip},
	}

	inv, err := AnalyzeHistory(dir, "run-squash-empty", base, head, events)
	if err != nil {
		t.Fatalf("AnalyzeHistory: %v", err)
	}
	if len(inv.Replacements) != 1 {
		t.Fatalf("expected exactly one replacement record, got %d", len(inv.Replacements))
	}
	if inv.Replacements[0].OriginalCommit != featureTip {
		t.Errorf("OriginalCommit = %q, want %q", inv.Replacements[0].OriginalCommit, featureTip)
	}
	if inv.Replacements[0].ReplacementCommit != squashCommit {
		t.Errorf("ReplacementCommit = %q, want %q (not the empty marker commit)", inv.Replacements[0].ReplacementCommit, squashCommit)
	}
	if len(inv.Interleaved) != 1 || inv.Interleaved[0] != head {
		t.Fatalf("Interleaved = %v, want [%s] (the empty marker commit)", inv.Interleaved, head)
	}
}
