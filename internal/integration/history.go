// Package integration — see run.go's package doc.
package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// UnreachableBaseError indicates a recorded base commit does not exist, or
// exists but is no longer reachable from the reviewed head — history was
// rewritten past it (e.g. the branch carrying it was discarded, or the
// repository was reset). integration_review's closure check and
// Backfill's own --base validation (backfill.go's verifyCommitReachable)
// both fail closed on this rather than ever falling back to inferring
// scope from `git merge-base HEAD main` (spec.md REQ-F-004 AC-T1; task
// T-E34-F08-013 AC-T1).
type UnreachableBaseError struct {
	Base string
	Head string
	Err  error
}

// Error implements the error interface.
func (e *UnreachableBaseError) Error() string {
	if e.Head == "" {
		return fmt.Sprintf("integration: base %s is missing or unreachable: %v", e.Base, e.Err)
	}
	return fmt.Sprintf("integration: base %s is missing or unreachable from head %s: %v", e.Base, e.Head, e.Err)
}

// Unwrap exposes the underlying resolution error for errors.Is/errors.As.
func (e *UnreachableBaseError) Unwrap() error {
	return e.Err
}

// VerifyBaseReachable checks that base exists as a real commit object in
// projectRoot's git repository and — whenever head is non-empty — that
// base is an ancestor of head. It never falls back to `git merge-base HEAD
// main`: a missing base, or one that no longer sits in head's ancestry,
// always returns a typed *UnreachableBaseError rather than silently
// narrowing the reviewed diff to a guessed common ancestor (spec.md
// REQ-F-004 AC-T1).
//
// This is the one function both Backfill's --base validation and
// AnalyzeHistory's base check call — "one correctness argument" for what
// counts as a reachable base (spec.md "Key technical decisions" #2),
// applied here to base-reachability rather than the write path that
// decision was originally written about.
func VerifyBaseReachable(projectRoot, base, head string) error {
	if strings.TrimSpace(base) == "" {
		return &UnreachableBaseError{Base: base, Head: head, Err: fmt.Errorf("base commit is empty")}
	}
	if err := verifyCommitObjectExists(projectRoot, base); err != nil {
		return &UnreachableBaseError{Base: base, Head: head, Err: err}
	}
	if strings.TrimSpace(head) == "" {
		return nil
	}
	if err := verifyCommitObjectExists(projectRoot, head); err != nil {
		return &UnreachableBaseError{Base: base, Head: head, Err: fmt.Errorf("head commit %s: %w", head, err)}
	}
	ancestor, err := isAncestor(projectRoot, base, head)
	if err != nil {
		return &UnreachableBaseError{Base: base, Head: head, Err: err}
	}
	if !ancestor {
		return &UnreachableBaseError{Base: base, Head: head, Err: fmt.Errorf("base is not an ancestor of head (history rewritten past the recorded base)")}
	}
	return nil
}

// verifyCommitObjectExists checks that commit resolves to a real commit
// object in projectRoot's repository (`git cat-file -e <commit>^{commit}`,
// which checks object existence and type without materializing content).
func verifyCommitObjectExists(projectRoot, commit string) error {
	cmd := exec.Command("git", "cat-file", "-e", commit+"^{commit}")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("commit %q does not resolve to a real commit object: %v: %s", commit, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// isAncestor reports whether ancestor is an ancestor of (or identical to)
// descendant, via `git merge-base --is-ancestor` — an exact reachability
// check, never `git merge-base <a> <b>` (which computes a *different*
// thing, a nearest-common-ancestor guess, and would silently substitute
// that guess for the exact check this package requires).
func isAncestor(projectRoot, ancestor, descendant string) (bool, error) {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", ancestor, descendant)
	cmd.Dir = projectRoot
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		// Exit status 1 is git's documented "not an ancestor" result, not a
		// failure to run the check.
		return false, nil
	}
	return false, fmt.Errorf("git merge-base --is-ancestor %s %s: %w", ancestor, descendant, err)
}

// ReplacementRecord is the explicit, immutable record created when a
// recorded feature completion's commit hash is no longer literally present
// in the base-to-head commit range but its tree content is found under a
// different, later commit in that same range — a rebase or squash-merge.
// architecture.md "Epic integration candidate identity": "Rebase and
// squash operations create an explicit replacement record linked by
// prior_record_digest; they do not silently rewrite the base." One
// immutable, content-addressed file per detected edge:
// .shark/runs/<epic-run-id>/integration-history/<record-digest>.json
// (spec.md REQ-F-004; task T-E34-F08-013 AC-T2).
type ReplacementRecord struct {
	EpicRunID         string    `json:"epic_run_id"`
	FeatureKey        string    `json:"feature_key"`
	OriginalCommit    string    `json:"original_commit"`
	ReplacementCommit string    `json:"replacement_commit"`
	PriorRecordDigest string    `json:"prior_record_digest"`
	DetectedAt        time.Time `json:"detected_at"`
	RecordDigest      string    `json:"record_digest"`
}

// HistoryInventory is the full base-to-head commit inventory
// integration_review's closure check reads: every commit accounted for by
// a recorded feature completion — directly, or via an explicit
// ReplacementRecord when its commit hash was rewritten by a rebase or
// squash-merge (AC-T2) — every remaining commit in the range that no event
// accounts for, reported as Interleaved (AC-T3), and every recorded event
// whose own FeatureCommit could not be placed anywhere in the range or
// resolved to a replacement, reported as Unaccounted rather than silently
// dropped — feature.md REQ-F-005 requires every completed/staged feature
// be included in the review inventory, and integration_review can only
// reject a genuinely missing feature event if this inventory exposes it.
type HistoryInventory struct {
	Base         string
	Head         string
	Replacements []*ReplacementRecord
	Interleaved  []string
	Unaccounted  []IntegrationEvent
}

// AnalyzeHistory computes epicRunID's base-to-head history inventory.
// head is required (unlike VerifyBaseReachable's own, more permissive
// contract, which Backfill's pre-candidate call relies on): `git rev-list
// base..` treats an omitted upper bound as the current HEAD, so an empty
// head here would silently infer scope from whatever HEAD happens to be at
// call time — the same class of silent scope-inference TC-018 forbids for
// `merge-base HEAD main`, just reached through a different empty string.
//
// For each of events' own FeatureCommit: a commit identical to base (a
// no-op feature) or already in the base..head range is accounted for
// directly. Otherwise AnalyzeHistory looks for a commit in the range whose
// introduced change is content-identical to the recorded commit's own
// cumulative change since base (via findRewrittenReplacement's patch-id
// comparison) — a rebase or squash-merge changes the commit hash and
// parent while preserving the change itself — and, on a match, persists an
// explicit ReplacementRecord linking the two (AC-T2) rather than rewriting
// the original event or the run's base. When neither placement succeeds,
// the event is reported on Unaccounted rather than silently dropped.
//
// Every commit in the range not attributed to an event's own commit or its
// replacement is reported as Interleaved (AC-T3) — visible in the
// inventory, not silently folded in or dropped; integration_review's own
// closure check is where a disposition for either list is required, not
// this function.
func AnalyzeHistory(projectRoot, epicRunID, base, head string, events []IntegrationEvent) (*HistoryInventory, error) {
	if strings.TrimSpace(head) == "" {
		return nil, fmt.Errorf("integration: AnalyzeHistory requires a non-empty head — an empty upper bound would let `git rev-list` silently resolve scope from the current HEAD instead of the recorded candidate head")
	}
	if err := VerifyBaseReachable(projectRoot, base, head); err != nil {
		return nil, err
	}

	rangeCommits, err := commitsInRange(projectRoot, base, head)
	if err != nil {
		return nil, fmt.Errorf("integration: list commits in range %s..%s: %w", base, head, err)
	}
	inRange := make(map[string]struct{}, len(rangeCommits))
	for _, c := range rangeCommits {
		inRange[c] = struct{}{}
	}

	inventory := &HistoryInventory{Base: base, Head: head}
	accounted := make(map[string]struct{}, len(events))

	for _, event := range events {
		commit := strings.TrimSpace(event.FeatureCommit)
		if commit == "" {
			continue
		}
		if commit == base {
			// A no-op feature: its recorded commit is the base itself, with
			// nothing to diff and nothing to replace.
			accounted[commit] = struct{}{}
			continue
		}
		if _, ok := inRange[commit]; ok {
			accounted[commit] = struct{}{}
			continue
		}

		replacement, err := findRewrittenReplacement(projectRoot, base, commit, rangeCommits)
		if err != nil {
			return nil, err
		}
		if replacement == "" {
			// Neither present in the range nor resolvable to a rewritten
			// replacement: surfaced so integration_review's own
			// closure-check can reject a genuinely missing feature event
			// (architecture.md) rather than never seeing it.
			inventory.Unaccounted = append(inventory.Unaccounted, event)
			continue
		}

		persisted, err := buildAndPersistReplacementRecord(projectRoot, epicRunID, event, replacement)
		if err != nil {
			return nil, err
		}
		inventory.Replacements = append(inventory.Replacements, persisted)
		accounted[replacement] = struct{}{}
	}

	var interleaved []string
	for _, c := range rangeCommits {
		if _, ok := accounted[c]; !ok {
			interleaved = append(interleaved, c)
		}
	}
	sort.Strings(interleaved)
	inventory.Interleaved = interleaved

	return inventory, nil
}

// commitsInRange lists every commit reachable from head but not from base
// (`git rev-list base..head`) — the exact base-to-candidate range
// architecture.md's full-diff inventory reviews. Returns (nil, nil) for an
// empty range (base and head identical, or head has nothing beyond base).
func commitsInRange(projectRoot, base, head string) ([]string, error) {
	cmd := exec.Command("git", "rev-list", base+".."+head)
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

// findRewrittenReplacement looks among candidates for a commit whose
// introduced change is content-identical to commit's own cumulative change
// since base, and returns it. It returns ("", nil) only for a positively
// identified "no rewrite to find" outcome — commit's own object no longer
// resolves (e.g. pruned), base and commit share no common history, or no
// candidate's patch matches — and a non-nil error for anything else (a git
// invocation failing for a reason other than "no such relationship"),
// so a git-invocation failure is never silently indistinguishable from a
// genuine "not found" and misreported as a missing feature event.
//
// A plain tree-equality check does not work here: a rebase commonly
// replays a feature's commit onto a *new* base that itself carries other,
// unrelated changes (exactly what makes a rebase worth testing), so the
// rewritten commit's resulting tree differs from the original commit's
// tree even though the feature's own change is unchanged. `git patch-id`
// is git's own tool for this exact problem — a content-stable identity for
// a diff that survives being replayed onto a different parent — so this
// compares commit's cumulative diff since its fork point off base against
// each candidate's own diff from its immediate parent. A squash-merge is
// handled by the same comparison: forkPoint is found by walking commit's
// own parent chain (still resolvable from the recorded commit hash even
// after the feature branch pointer itself is gone), so the cumulative
// diff compared is the feature's *whole* squashed change, not just its
// final commit's own patch.
func findRewrittenReplacement(projectRoot, base, commit string, candidates []string) (string, error) {
	if err := verifyCommitObjectExists(projectRoot, commit); err != nil {
		return "", nil
	}
	forkPoint, ok, err := mergeBase(projectRoot, base, commit)
	if err != nil {
		return "", fmt.Errorf("integration: resolve fork point for %s off base %s: %w", commit, base, err)
	}
	if !ok {
		// base and commit share no common history at all — not a rewrite.
		return "", nil
	}
	if forkPoint == commit {
		// commit is itself an ancestor of base (predates the epic's own
		// recorded base) — there is no diff to compare a replacement
		// against, so this is not a rewrite either.
		return "", nil
	}
	wantPatch, err := diffPatchID(projectRoot, forkPoint, commit)
	if err != nil {
		return "", fmt.Errorf("integration: compute patch identity for %s: %w", commit, err)
	}

	for _, candidate := range candidates {
		parent, err := firstParent(projectRoot, candidate)
		if err != nil {
			// A parentless (root) candidate commit has nothing to diff
			// against for this comparison — not comparable, not an error.
			continue
		}
		candidatePatch, err := diffPatchID(projectRoot, parent, candidate)
		if err != nil {
			return "", fmt.Errorf("integration: compute patch identity for candidate %s: %w", candidate, err)
		}
		if candidatePatch == wantPatch {
			return candidate, nil
		}
	}
	return "", nil
}

// mergeBase resolves the best common ancestor of a and b (`git merge-base
// a b`) — used here to find a recorded commit's own fork point off the
// epic's base, regardless of what has happened to the branch pointer that
// used to name it. ok is false, with a nil error, specifically when git
// reports no common ancestor exists (exit status 1) — a legitimate "these
// two share no history" outcome, not a failure to run the check.
func mergeBase(projectRoot, a, b string) (string, bool, error) {
	cmd := exec.Command("git", "merge-base", a, b)
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return "", false, nil
	}
	return "", false, fmt.Errorf("git merge-base %s %s: %w", a, b, err)
}

// firstParent resolves commit's first parent (`git rev-parse commit^`).
func firstParent(projectRoot, commit string) (string, error) {
	cmd := exec.Command("git", "rev-parse", commit+"^")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// diffPatchID computes `git patch-id --stable`'s content-stable identity
// for the diff from..to: the same identity survives a rebase or squash
// that changes the commit hash and parent but preserves the change itself.
func diffPatchID(projectRoot, from, to string) (string, error) {
	diffCmd := exec.Command("git", "diff", from, to)
	diffCmd.Dir = projectRoot
	diffOut, err := diffCmd.Output()
	if err != nil {
		return "", err
	}
	if len(bytes.TrimSpace(diffOut)) == 0 {
		return "", fmt.Errorf("integration: empty diff between %s and %s", from, to)
	}

	patchIDCmd := exec.Command("git", "patch-id", "--stable")
	patchIDCmd.Dir = projectRoot
	patchIDCmd.Stdin = bytes.NewReader(diffOut)
	patchOut, err := patchIDCmd.Output()
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(patchOut))
	if len(fields) == 0 {
		return "", fmt.Errorf("integration: git patch-id produced no output for diff %s..%s", from, to)
	}
	return fields[0], nil
}

// buildAndPersistReplacementRecord builds the ReplacementRecord linking
// event's original, now-rewritten commit to replacementCommit, and
// durably persists it to its content-addressed path — returning the
// record that now exists on disk. If an identical edge was already
// detected and persisted by an earlier call (this AnalyzeHistory run, an
// earlier one, or a concurrent one), the previously-persisted bytes are
// read back and returned rather than the freshly-built (different
// DetectedAt) in-memory copy, so a caller can never observe an inventory
// entry whose DetectedAt disagrees with what is actually on disk.
func buildAndPersistReplacementRecord(projectRoot, epicRunID string, event IntegrationEvent, replacementCommit string) (*ReplacementRecord, error) {
	priorDigest, err := eventDigest(event)
	if err != nil {
		return nil, err
	}

	record := &ReplacementRecord{
		EpicRunID:         epicRunID,
		FeatureKey:        event.FeatureKey,
		OriginalCommit:    event.FeatureCommit,
		ReplacementCommit: replacementCommit,
		PriorRecordDigest: priorDigest,
		DetectedAt:        time.Now().UTC(),
	}
	digest, err := computeReplacementDigest(*record)
	if err != nil {
		return nil, err
	}
	record.RecordDigest = digest

	return persistReplacementRecord(projectRoot, epicRunID, record)
}

// eventDigest computes the sha256 hex digest of event's canonical JSON —
// the "prior record" identity a ReplacementRecord's PriorRecordDigest
// links back to.
func eventDigest(event IntegrationEvent) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("integration: marshal event for digest: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// computeReplacementDigest computes record's own digest: sha256 hex of its
// canonical JSON with RecordDigest *and* DetectedAt cleared first.
// RecordDigest is excluded matching candidate.go's exclude-own-digest-field
// convention; DetectedAt is excluded matching event.go's deriveEventID,
// which deliberately excludes RecordedAt so a retried write of the same
// logical edge is idempotent (same bytes, same content-addressed path)
// rather than minting a new file on every call purely because wall-clock
// time advanced between calls.
func computeReplacementDigest(record ReplacementRecord) (string, error) {
	record.RecordDigest = ""
	record.DetectedAt = time.Time{}
	data, err := json.Marshal(record)
	if err != nil {
		return "", fmt.Errorf("integration: marshal replacement record for digest: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// replacementRecordPath is the content-addressed path for epicRunID's
// replacement record identified by recordDigest.
func replacementRecordPath(projectRoot, epicRunID, recordDigest string) string {
	return filepath.Join(projectRoot, ".shark", "runs", epicRunID, "integration-history", recordDigest+".json")
}

// persistReplacementRecord durably writes record to its content-addressed
// path, atomically (temp file + os.Link, matching this package's other
// publish-once patterns), and returns the record that now exists on disk.
// Idempotent: record's path is derived from its own RecordDigest — which
// excludes DetectedAt — so a repeat detection of the identical edge always
// derives the identical path. If the file already exists (this exact edge
// was already detected and recorded, whether by an earlier AnalyzeHistory
// call or a concurrent one), persistReplacementRecord reads back and
// returns the winner's already-persisted record instead of writing again,
// mirroring publishEvent/publishRun's own "another caller already
// published, read back the winner" pattern.
func persistReplacementRecord(projectRoot, epicRunID string, record *ReplacementRecord) (*ReplacementRecord, error) {
	path := replacementRecordPath(projectRoot, epicRunID, record.RecordDigest)
	if existing, err := readReplacementRecord(path); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, runDirMode); err != nil {
		return nil, fmt.Errorf("integration: create history directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("integration: marshal replacement record: %w", err)
	}

	tmpPath := fmt.Sprintf("%s.%d-%d.tmp", path, os.Getpid(), time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, data, runFileMode); err != nil {
		return nil, fmt.Errorf("integration: write temp replacement record: %w", err)
	}
	defer os.Remove(tmpPath)

	if err := os.Link(tmpPath, path); err != nil {
		if os.IsExist(err) {
			// Published concurrently by another caller — read back its
			// record, exactly like a "not found" retry above.
			winner, readErr := readReplacementRecord(path)
			if readErr != nil {
				return nil, readErr
			}
			if winner == nil {
				return nil, fmt.Errorf("integration: replacement record at %s vanished after a concurrent publish", path)
			}
			return winner, nil
		}
		return nil, fmt.Errorf("integration: publish replacement record at %s: %w", path, err)
	}
	return record, nil
}

// readReplacementRecord reads and parses the replacement record at path.
// It returns (nil, nil) when no file exists yet, mirroring this package's
// other readX helpers (readRun/readEvent/readCandidate).
func readReplacementRecord(path string) (*ReplacementRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("integration: read replacement record at %s: %w", path, err)
	}
	var record ReplacementRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("integration: replacement record at %s is corrupt: %w", path, err)
	}
	return &record, nil
}
