package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/gitutil"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

// AttachParentTriageObservation appends the parent-owned observation file for
// runID to the worker envelope. It is the single injection step shared by the
// apply-result adapter and the core controller. The worker cannot supply the
// observation itself.
func AttachParentTriageObservation(projectRoot, runID, path string, envelopeBytes []byte) ([]byte, error) {
	runDir, err := gaterun.RunDir(projectRoot, runID)
	if err != nil {
		return nil, err
	}
	expectedPath := filepath.Join(runDir, "parent-triage-observation.json")
	actualPath, err := filepath.Abs(path)
	if err != nil || actualPath != expectedPath {
		return nil, fmt.Errorf("parent observation must be at the run-scoped parent-triage-observation.json path")
	}
	data, err := gaterun.ReadBoundedRegularFile(actualPath, 64*1024)
	if err != nil {
		return nil, err
	}
	envelope, err := workercontrol.Decode(envelopeBytes)
	if err != nil {
		return nil, err
	}
	if hasBugEvidence(envelope.Evidence, "bug_parent_triage_observation") {
		return nil, fmt.Errorf("worker envelope cannot supply parent triage observation")
	}
	digest := sha256.Sum256(data)
	relative, err := filepath.Rel(projectRoot, actualPath)
	if err != nil {
		return nil, err
	}
	envelope.Evidence = append(envelope.Evidence, workercontrol.EvidenceRef{Kind: "bug_parent_triage_observation", Pointer: filepath.ToSlash(relative), Summary: "sha256:" + hex.EncodeToString(digest[:])})
	return json.Marshal(envelope)
}

// RunOwnParentObservationInjected reports whether the envelope carries exactly
// one bug_parent_triage_observation reference that points at this run's own
// parent-triage-observation.json and whose digest matches that file. Crash
// recovery (--resume-run) uses it to recognize the controller's own injection
// in a committed result; any other pointer, digest or file state is false.
func RunOwnParentObservationInjected(projectRoot, runID string, envelopeBytes []byte) bool {
	runDir, err := gaterun.RunDir(projectRoot, runID)
	if err != nil {
		return false
	}
	envelope, err := workercontrol.Decode(envelopeBytes)
	if err != nil {
		return false
	}
	ref, err := selectBugEvidence(envelope.Evidence, "bug_parent_triage_observation")
	if err != nil {
		return false
	}
	path := filepath.Join(runDir, "parent-triage-observation.json")
	relative, err := filepath.Rel(projectRoot, path)
	if err != nil || ref.Pointer != filepath.ToSlash(relative) {
		return false
	}
	data, err := gaterun.ReadBoundedRegularFile(path, 64*1024)
	if err != nil {
		return false
	}
	digest := sha256.Sum256(data)
	return ref.Summary == "sha256:"+hex.EncodeToString(digest[:])
}

// BugTriageWindow identifies the parent-measured interval around one dispatch.
type BugTriageWindow struct {
	ProjectRoot, WorkingDir, RunID string
	// TrackerWrites is the number of entity notes created during the dispatch,
	// counted by the parent from the tracker, not reported by the worker.
	TrackerWrites int
}

// ObserveBugReportOnlyTriage measures what the host did while a bug code review
// withdrew findings as report-only, writes the observation exclusively into
// the parent run directory and attaches it. A pre-existing file is treated as
// planted and rejected. It returns the original envelope when no report-only
// finding needs an observation.
func ObserveBugReportOnlyTriage(ctx context.Context, w BugTriageWindow, envelopeBytes []byte) ([]byte, bool, error) {
	envelope, err := workercontrol.Decode(envelopeBytes)
	if err != nil {
		// Malformed worker output is reported by IngestGateResult itself.
		return envelopeBytes, false, nil
	}
	ids, identity, err := measuredReportOnlyFindings(w.WorkingDir, envelope.Evidence)
	if err != nil || len(ids) == 0 {
		return envelopeBytes, false, err
	}
	edits, err := measureCheckoutEdits(ctx, w.WorkingDir, envelope.Evidence)
	if err != nil {
		return nil, false, err
	}
	actions := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		actions = append(actions, map[string]any{"finding_id": id.id, "action": id.action, "edits": edits, "tracker_writes": w.TrackerWrites})
	}
	data, err := json.Marshal(map[string]any{"candidate_identity": identity, "observed_by": "parent", "actions": actions})
	if err != nil {
		return nil, false, err
	}
	path, err := writeExclusiveObservation(w.ProjectRoot, w.RunID, data)
	if err != nil {
		return nil, false, err
	}
	attached, err := AttachParentTriageObservation(w.ProjectRoot, w.RunID, path, envelopeBytes)
	return attached, err == nil, err
}

func readDigestBoundEvidence(root string, refs []workercontrol.EvidenceRef, kind string) ([]byte, error) {
	ref, err := selectBugEvidence(refs, kind)
	if err != nil {
		return nil, err
	}
	expected, err := validateBugEvidenceRef(ref)
	if err != nil {
		return nil, err
	}
	data, err := readBoundedBugEvidence(root, ref.Pointer, bugEvidenceLimit(kind))
	if err != nil {
		return nil, err
	}
	if digest := sha256.Sum256(data); !bytes.Equal(digest[:], expected) {
		return nil, fmt.Errorf("%s digest is stale", kind)
	}
	return data, nil
}

type measuredHostAction struct{ id, action string }

func measuredReportOnlyFindings(root string, refs []workercontrol.EvidenceRef) ([]measuredHostAction, bugReviewIdentity, error) {
	if !hasBugEvidence(refs, "bug_review_decisions") {
		return nil, bugReviewIdentity{}, nil
	}
	data, err := readDigestBoundEvidence(root, refs, "bug_review_decisions")
	if err != nil {
		return nil, bugReviewIdentity{}, err
	}
	var packet bugReviewDecisions
	if err := json.Unmarshal(data, &packet); err != nil {
		return nil, bugReviewIdentity{}, fmt.Errorf("invalid review decisions: %w", err)
	}
	decisions := make(map[bugFindingKey]bugReviewDecision, len(packet.Findings))
	for _, finding := range packet.Findings {
		decisions[bugFindingKey{finding.File, finding.Line, finding.Rule, finding.Summary, finding.Diagnosis}] = finding
	}
	raw, err := measuredRawSeverities(root, refs)
	if err != nil {
		return nil, bugReviewIdentity{}, err
	}
	byID, err := expectedBugHostActions(decisions, raw)
	if err != nil {
		return nil, bugReviewIdentity{}, err
	}
	ids := make([]measuredHostAction, 0, len(byID))
	for _, finding := range packet.Findings {
		if action, ok := byID[finding.FindingID]; ok {
			ids = append(ids, measuredHostAction{finding.FindingID, action})
		}
	}
	return ids, packet.CandidateIdentity, nil
}

// measuredRawSeverities reads the raw angle findings when the worker retained them.
func measuredRawSeverities(root string, refs []workercontrol.EvidenceRef) (map[bugFindingKey]string, error) {
	if !hasBugEvidence(refs, "bug_review_angles") {
		return nil, nil
	}
	data, err := readDigestBoundEvidence(root, refs, "bug_review_angles")
	if err != nil {
		return nil, err
	}
	return rawSeveritiesFromAngles(data)
}

// measureCheckoutEdits counts changed or untracked source entries relative to
// the frozen candidate commit, ignoring Shark runtime artifacts.
func measureCheckoutEdits(ctx context.Context, workingDir string, refs []workercontrol.EvidenceRef) (int, error) {
	data, err := readDigestBoundEvidence(workingDir, refs, "bug_review_scope")
	if err != nil {
		return 0, err
	}
	var scope bugCandidateScope
	if err := json.Unmarshal(data, &scope); err != nil {
		return 0, err
	}
	run := func(args ...string) ([]byte, error) {
		return gitutil.Command(ctx, append([]string{"-c", "core.fsmonitor=false", "-C", workingDir}, args...)...).Output()
	}
	if err := validateCleanBugHeadCommit(scope, run); err != nil {
		return 0, err
	}
	status, err := cleanReviewStatus(normalizedRuntimeProject(scope.RuntimeProject), run)
	if err != nil {
		return 0, err
	}
	return len(bytes.FieldsFunc(status, func(r rune) bool { return r == 0 })), nil
}

func validateCleanBugHeadCommit(scope bugCandidateScope, run bugGitRunner) error {
	head, err := run("rev-parse", "HEAD")
	if err != nil || string(bytes.TrimSpace(head)) != scope.CandidateCommit {
		return fmt.Errorf("checkout moved off the frozen candidate commit")
	}
	return nil
}

func writeExclusiveObservation(projectRoot, runID string, data []byte) (string, error) {
	runDir, err := gaterun.RunDir(projectRoot, runID)
	if err != nil {
		return "", err
	}
	path := filepath.Join(runDir, "parent-triage-observation.json")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("parent triage observation already exists or cannot be created: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return "", err
	}
	return path, file.Close()
}

// countBugEntityNotes counts all tracker notes for an entity.
func countBugEntityNotes(ctx context.Context, reader gatepersist.NoteReader, entityType models.EntityType, key string) (int, error) {
	if reader == nil {
		return 0, fmt.Errorf("tracker note reader unavailable")
	}
	notes, err := reader.ListNotes(ctx, entityType, key, nil)
	return len(notes), err
}
