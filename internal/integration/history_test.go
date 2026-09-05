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
