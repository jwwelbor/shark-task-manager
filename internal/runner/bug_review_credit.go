package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gitutil"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Credit contract v1. A six-angle packet may omit a changed path from its
// fresh reviewed_files list only by declaring a carried unit for it. A carried
// unit is a file-local obligation with a closed input list. It is eligible
// only when the parent can independently verify the predecessor receipt, the
// predecessor angle's own direct inspection, unchanged Git tree entries for the
// path and every input, and an uninterrupted review history.
const bugCreditVersion = 1

var bugCreditOperationDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var bugGateTransitionNote = regexp.MustCompile(`^gate (\S+) outcome (\S+) run \S+ operation [0-9a-f]{64}$`)

type bugCarriedUnit struct {
	Version              int            `json:"version"`
	Path                 reviewGitPath  `json:"path"`
	Obligation           string         `json:"obligation"`
	Inputs               reviewGitPaths `json:"inputs"`
	DynamicLookup        *bool          `json:"dynamic_lookup"`
	PredecessorCommit    string         `json:"predecessor_commit"`
	PredecessorRun       string         `json:"predecessor_run_id"`
	PredecessorOperation string         `json:"predecessor_operation_digest"`
}

// bugUnitVerifier performs the parent-side checks for one carried unit.
type bugUnitVerifier func(angle string, unit bugCarriedUnit) error

// bugCredit carries the parent-held readers needed to verify predecessor
// receipts. A nil value means no credit is available and every angle must list
// the full manifest.
type bugCredit struct {
	notes     gatepersist.NoteReader
	history   gatepersist.HistoryReader
	normalize func(string) string
	resolve   bugSuccessTarget
	// endID bounds the review interval at historical QA replay: the accepted
	// successor's own code-review transition. Zero means the interval runs to
	// the latest history entry.
	endID int64
}

func decodeBugSuccessorDelta(scope bugCandidateScope) (*bugSuccessorDelta, error) {
	if len(scope.SuccessorDelta) == 0 {
		return nil, nil
	}
	var delta bugSuccessorDelta
	if err := decodeStrict(scope.SuccessorDelta, &delta); err != nil {
		return nil, fmt.Errorf("decode predecessor comparison: %w", err)
	}
	return &delta, nil
}

func pathSet(paths []string) map[string]bool {
	set := make(map[string]bool, len(paths))
	for _, path := range paths {
		set[path] = true
	}
	return set
}

// checkBugAngleCoverage keeps the legacy full-manifest rule for packets without
// carried units and otherwise reconciles fresh and verified carried paths.
func checkBugAngleCoverage(name string, angle bugAngle, scope bugCandidateScope, verify bugUnitVerifier) error {
	if len(angle.CarriedUnits) == 0 {
		return verifyDeclaredReviewPaths(angle.ReviewedFiles, scope.ChangedFiles)
	}
	delta, err := decodeBugSuccessorDelta(scope)
	if err != nil {
		return err
	}
	if delta == nil {
		return fmt.Errorf("carried units require a captured successor delta")
	}
	carried, err := checkCarriedShapes(angle, scope, delta)
	if err != nil {
		return err
	}
	if err := requireFreshOrCarried(angle, scope, carried); err != nil {
		return err
	}
	if verify == nil {
		return fmt.Errorf("carried credit is unavailable without parent review history")
	}
	for _, unit := range angle.CarriedUnits {
		if err := verify(name, unit); err != nil {
			return fmt.Errorf("carried unit %q: %w", string(unit.Path), err)
		}
	}
	return nil
}

func requireFreshOrCarried(angle bugAngle, scope bugCandidateScope, carried map[string]bool) error {
	if err := verifyDeclaredReviewPaths(angle.ReviewedFiles, nil); err != nil {
		return err
	}
	fresh := pathSet(angle.ReviewedFiles)
	for path := range carried {
		if fresh[path] {
			return fmt.Errorf("path %q is both fresh and carried", path)
		}
	}
	for _, path := range scope.ChangedFiles {
		if !fresh[path] && !carried[path] {
			return fmt.Errorf("coverage incomplete: missing %q", path)
		}
	}
	return nil
}

func checkCarriedShapes(angle bugAngle, scope bugCandidateScope, delta *bugSuccessorDelta) (map[string]bool, error) {
	if !angle.CrossFileFresh {
		return nil, fmt.Errorf("carried units require a fresh cross-file inspection declaration")
	}
	manifest, changed := pathSet(scope.ChangedFiles), pathSet(delta.ChangedFiles)
	carried := make(map[string]bool, len(angle.CarriedUnits))
	for _, unit := range angle.CarriedUnits {
		if err := checkCarriedShape(unit, delta, manifest, changed); err != nil {
			return nil, fmt.Errorf("carried unit %q: %w", string(unit.Path), err)
		}
		if carried[string(unit.Path)] {
			return nil, fmt.Errorf("carried unit %q is duplicated", string(unit.Path))
		}
		carried[string(unit.Path)] = true
	}
	return carried, nil
}

func checkCarriedShape(unit bugCarriedUnit, delta *bugSuccessorDelta, manifest, changed map[string]bool) error {
	if unit.Version != bugCreditVersion || unit.Obligation != "file_local" {
		return fmt.Errorf("unsupported credit version or obligation")
	}
	if unit.DynamicLookup == nil || *unit.DynamicLookup || unit.Inputs == nil {
		return fmt.Errorf("a closed input list without dynamic lookup is required")
	}
	if unit.PredecessorCommit != delta.PredecessorCommit || unit.PredecessorRun == "" || !bugCreditOperationDigest.MatchString(unit.PredecessorOperation) {
		return fmt.Errorf("predecessor identity does not match the captured successor delta")
	}
	if !manifest[string(unit.Path)] || changed[string(unit.Path)] {
		return fmt.Errorf("path is outside the manifest or changed since the predecessor")
	}
	for _, input := range unit.Inputs {
		if changed[input] {
			return fmt.Errorf("declared input %q changed since the predecessor", input)
		}
	}
	return nil
}

// verifier returns the parent-side unit verifier for one scope, or nil when no
// parent readers are available.
func (c *bugCredit) verifier(ctx context.Context, req gatepersist.Request, scope bugCandidateScope) bugUnitVerifier {
	if c == nil || c.notes == nil || c.history == nil || c.resolve == nil {
		return nil
	}
	delta, err := decodeBugSuccessorDelta(scope)
	if err != nil || delta == nil {
		return nil
	}
	memo := make(map[string]*bugPredecessor)
	return func(angle string, unit bugCarriedUnit) error {
		key := unit.PredecessorRun + "/" + unit.PredecessorOperation
		pred, ok := memo[key]
		if !ok {
			loaded, err := c.loadPredecessor(ctx, req, unit)
			if err != nil {
				return err
			}
			pred, memo[key] = loaded, loaded
		}
		if err := pred.matchesDelta(scope, delta); err != nil {
			return err
		}
		if err := pred.inspectedDirectly(angle, unit); err != nil {
			return err
		}
		return sameBugTreeEntries(ctx, req, delta.PredecessorTree, scope.CandidateTree, append([]string{string(unit.Path)}, unit.Inputs...))
	}
}

type bugPredecessor struct {
	scope  bugCandidateScope
	angles map[string]bugAngle
}

func (p *bugPredecessor) matchesDelta(scope bugCandidateScope, delta *bugSuccessorDelta) error {
	if p.scope.CandidateCommit != delta.PredecessorCommit || p.scope.CandidateTree != delta.PredecessorTree || p.scope.BaseCommit != scope.BaseCommit {
		return fmt.Errorf("predecessor receipt does not bind the captured predecessor commit and baseline")
	}
	return nil
}

func (p *bugPredecessor) inspectedDirectly(name string, unit bugCarriedUnit) error {
	angle, found := p.angles[name]
	if !found || !angle.CandidateIdentity.matches(p.scope) {
		return fmt.Errorf("predecessor angle %s is unavailable", name)
	}
	for _, carried := range angle.CarriedUnits {
		if carried.Path == unit.Path {
			return fmt.Errorf("recursive credit is not allowed")
		}
	}
	for _, path := range append([]string{string(unit.Path)}, unit.Inputs...) {
		if !pathSet(angle.ReviewedFiles)[path] {
			return fmt.Errorf("predecessor angle %s has no direct inspection of %q", name, path)
		}
	}
	return nil
}

func (c *bugCredit) loadPredecessor(ctx context.Context, req gatepersist.Request, unit bugCarriedUnit) (*bugPredecessor, error) {
	meta, entry, err := c.findPredecessorReceipt(ctx, req, unit)
	if err != nil {
		return nil, err
	}
	if err := c.requireUninterrupted(ctx, req, entry); err != nil {
		return nil, err
	}
	data, evidenceRoot, err := readBugApprovalResult(ctx, req, meta, "code_review")
	if err != nil {
		return nil, fmt.Errorf("predecessor receipt: %w", err)
	}
	parent := filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir)))
	return readBugPredecessorEvidence(ctx, req, parent, meta.RunID, evidenceRoot, data)
}

func readBugPredecessorEvidence(_ context.Context, req gatepersist.Request, parent, run, evidenceRoot string, data []byte) (*bugPredecessor, error) {
	prior := gatepersist.Request{
		EntityKey: req.EntityKey, EntityType: req.EntityType, Gate: "code_review", SourceStatus: "code_review",
		RunDir: filepath.Join(parent, ".shark", "runs", run), RunID: run, EvidenceRoot: evidenceRoot, EnvelopeJSON: data,
	}
	_, scopeData, err := readBugEvidence(prior, "bug_review_scope")
	if err != nil {
		return nil, fmt.Errorf("predecessor scope: %w", err)
	}
	var scope bugCandidateScope
	if err := json.Unmarshal(scopeData, &scope); err != nil {
		return nil, err
	}
	_, angleData, err := readBugEvidence(prior, "bug_review_angles")
	if err != nil {
		return nil, fmt.Errorf("predecessor angles: %w", err)
	}
	var angles map[string]bugAngle
	if err := json.Unmarshal(angleData, &angles); err != nil || len(angles) != 6 {
		return nil, fmt.Errorf("predecessor angle packet is invalid")
	}
	return &bugPredecessor{scope: scope, angles: angles}, nil
}

// findPredecessorReceipt resolves the unit's run and operation digest against
// parent-held approval notes and applied history, never by filename order.
func (c *bugCredit) findPredecessorReceipt(ctx context.Context, req gatepersist.Request, unit bugCarriedUnit) (bugApprovalNote, *models.EntityHistory, error) {
	entries, err := c.history.GetHistory(ctx, req.EntityType, req.EntityKey)
	if err != nil {
		return bugApprovalNote{}, nil, fmt.Errorf("predecessor history unavailable: %w", err)
	}
	notes, err := c.notes.ListNotes(ctx, req.EntityType, req.EntityKey, []string{"review"})
	if err != nil {
		return bugApprovalNote{}, nil, fmt.Errorf("predecessor notes unavailable: %w", err)
	}
	spec := bugPriorSpec{"code_review", "bug_review_scope"}
	for _, note := range notes {
		meta, ok := approvalNoteFor(note, unit)
		if !ok {
			continue
		}
		for _, entry := range entries {
			if matchesBugApprovalHistory(entry, spec, meta, c.normalize, c.resolve) {
				return meta, entry, nil
			}
		}
	}
	return bugApprovalNote{}, nil, fmt.Errorf("predecessor receipt is not a parent-applied code review pass for this entity")
}

func approvalNoteFor(note *models.EntityNote, unit bugCarriedUnit) (bugApprovalNote, bool) {
	var meta bugApprovalNote
	if note == nil || note.Metadata == nil || json.Unmarshal([]byte(*note.Metadata), &meta) != nil {
		return meta, false
	}
	return meta, meta.matches("code_review") && meta.RunID == unit.PredecessorRun && meta.Digest == unit.PredecessorOperation
}

// requireUninterrupted rejects a predecessor when a later gate result with a
// non-success outcome role, or a forced transition, occurred before the end of
// the review interval. Roles come from workflow configuration.
func (c *bugCredit) requireUninterrupted(ctx context.Context, req gatepersist.Request, pred *models.EntityHistory) error {
	entries, err := c.history.GetHistory(ctx, req.EntityType, req.EntityKey)
	if err != nil {
		return fmt.Errorf("review interval is unverifiable: %w", err)
	}
	for _, entry := range entries {
		if entry == nil || entry.ID <= pred.ID || (c.endID != 0 && entry.ID > c.endID) {
			continue
		}
		if c.interrupts(entry) {
			return fmt.Errorf("a rejection or forced transition occurred after the predecessor receipt")
		}
	}
	return nil
}

func (c *bugCredit) interrupts(entry *models.EntityHistory) bool {
	if entry.Forced {
		return true
	}
	if entry.Notes == nil {
		return false
	}
	match := bugGateTransitionNote.FindStringSubmatch(*entry.Notes)
	if match == nil {
		return false
	}
	_, success := c.resolve(strings.ToLower(match[1]), match[2])
	return !success
}

// sameBugTreeEntries compares mode, type and object of each path between two
// trees using literal pathspecs.
func sameBugTreeEntries(ctx context.Context, req gatepersist.Request, treeA, treeB string, paths []string) error {
	repo, _, err := candidateRepository(ctx, bugEvidenceProject(req))
	if err != nil {
		return err
	}
	for _, path := range paths {
		a, err := bugTreeEntry(ctx, repo, treeA, path)
		if err != nil {
			return err
		}
		b, err := bugTreeEntry(ctx, repo, treeB, path)
		if err != nil {
			return err
		}
		if a != b {
			return fmt.Errorf("%q differs between the predecessor and candidate trees", path)
		}
	}
	return nil
}

func bugTreeEntry(ctx context.Context, repo, tree, path string) (string, error) {
	if !validGitHash(tree) {
		return "", fmt.Errorf("invalid tree identity")
	}
	out, err := gitutil.Command(ctx, "-c", "core.fsmonitor=false", "--literal-pathspecs", "-C", repo, "ls-tree", "-z", tree, "--", path).Output()
	if err != nil {
		return "", fmt.Errorf("read tree entry: %w", err)
	}
	return string(out), nil
}
