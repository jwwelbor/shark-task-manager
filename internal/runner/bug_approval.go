package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

// NewBugGateAdmission binds downstream evidence to completed prior gates.
func NewBugGateAdmission(notes gatepersist.NoteReader, history gatepersist.HistoryReader) func(context.Context, gatepersist.Request) error {
	return func(ctx context.Context, req gatepersist.Request) error {
		if err := BugGateAdmission(ctx, req); err != nil {
			return err
		}
		if req.EntityType != models.EntityTypeBug || req.Role != gateresult.RoleSuccess {
			return nil
		}
		return admitBugDownstream(ctx, req, notes, history)
	}
}

func admitBugDownstream(ctx context.Context, req gatepersist.Request, notes gatepersist.NoteReader, history gatepersist.HistoryReader) error {
	switch req.Gate {
	case "development", "code_review", "qa":
	default:
		return nil
	}
	if notes == nil || history == nil {
		return fmt.Errorf("plan approval readers are required")
	}
	if err := verifyBugPriorGate(ctx, req, notes, history, bugPriorSpec{"plan_review", "bug_repair_plan", "development"}); err != nil {
		return err
	}
	if req.Gate == "development" {
		return nil
	}
	if _, err := readBugReviewScope(ctx, req); err != nil {
		return err
	}
	if req.Gate == "code_review" {
		return verifyBugReviewEvidence(req)
	}
	return verifyBugPriorGate(ctx, req, notes, history, bugPriorSpec{"code_review", "bug_review_scope", "qa"})
}

type bugApprovalNote struct {
	Gate    string `json:"gate"`
	Role    string `json:"role"`
	Outcome string `json:"outcome_key"`
	RunID   string `json:"run_id"`
	Digest  string `json:"operation_digest"`
}
type bugPriorSpec struct{ gate, evidenceKind, target string }

func verifyBugPriorGate(ctx context.Context, req gatepersist.Request, notes gatepersist.NoteReader, history gatepersist.HistoryReader, spec bugPriorSpec) error {
	entry, err := latestRelevantBugHistory(ctx, req, history, spec)
	if err != nil {
		return err
	}
	meta, err := latestBugApproval(ctx, req, notes, spec.gate, entry)
	if err != nil {
		return err
	}
	if !matchesBugApprovalHistory(entry, spec, meta) {
		return fmt.Errorf("prior gate approval invalidated or not applied in history")
	}
	data, err := readBugApprovalResult(req, meta, spec.gate)
	if err != nil {
		return err
	}
	if err := compareBugApprovalEvidence(req, data, spec); err != nil {
		return err
	}
	return nil
}

func latestBugApproval(ctx context.Context, req gatepersist.Request, notes gatepersist.NoteReader, gate string, entry *models.EntityHistory) (bugApprovalNote, error) {
	if entry == nil || entry.Notes == nil {
		return bugApprovalNote{}, fmt.Errorf("prior gate transition is missing")
	}
	items, err := notes.ListNotes(ctx, req.EntityType, req.EntityKey, []string{"review"})
	if err != nil {
		return bugApprovalNote{}, fmt.Errorf("read approval notes: %w", err)
	}
	var selected *models.EntityNote
	var meta bugApprovalNote
	for _, note := range items {
		candidate, ok := matchingApprovalNote(note, gate, *entry.Notes)
		if !ok {
			continue
		}
		if selected == nil || note.ID > selected.ID {
			selected, meta = note, candidate
		}
	}
	if selected == nil {
		return meta, fmt.Errorf("required prior gate approval is missing")
	}
	return meta, nil
}

func matchingApprovalNote(note *models.EntityNote, gate, reason string) (bugApprovalNote, bool) {
	var candidate bugApprovalNote
	if note == nil || note.Metadata == nil {
		return candidate, false
	}
	if json.Unmarshal([]byte(*note.Metadata), &candidate) != nil {
		return candidate, false
	}
	return candidate, candidate.matches(gate) && candidate.matchesTransition(reason)
}
func (n bugApprovalNote) matches(gate string) bool {
	return n.Gate == gate && n.Role == "success" && n.Outcome != ""
}
func (n bugApprovalNote) matchesTransition(reason string) bool {
	return reason == gatepersist.BoundTransitionReason(n.Gate, n.Outcome, n.RunID, n.Digest)
}

func readBugApprovalResult(req gatepersist.Request, meta bugApprovalNote, gate string) ([]byte, error) {
	root := filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir)))
	dir, err := gaterun.RunDir(root, meta.RunID)
	if err != nil {
		return nil, fmt.Errorf("resolve approval run: %w", err)
	}
	state, exists, err := gaterun.LoadOperationState(dir)
	if err != nil || !exists {
		return nil, fmt.Errorf("approval operation state unavailable")
	}
	if !matchesBugApprovalState(state, req, meta, gate) {
		return nil, fmt.Errorf("approval operation is not a committed matching review")
	}
	if err := verifyBugApprovalIdentity(dir, req, meta, gate); err != nil {
		return nil, err
	}
	return decodeBugApprovalResult(dir, req, meta, gate)
}
func matchesBugApprovalState(state *gaterun.OperationState, req gatepersist.Request, meta bugApprovalNote, gate string) bool {
	return state.AdmissionVersion == gatepersist.BoundAdmissionVersion && state.PersistenceState == gaterun.PersistenceStateTransitioned && state.EntityType == string(req.EntityType) && strings.EqualFold(state.EntityKey, req.EntityKey) && state.Gate == gate && state.SourceStatus == gate && state.OperationDigest == meta.Digest
}
func verifyBugApprovalIdentity(dir string, req gatepersist.Request, meta bugApprovalNote, gate string) error {
	identity, exists, err := gaterun.ReadIdentity(dir)
	if err != nil || !exists {
		return fmt.Errorf("approval run identity unavailable")
	}
	want := gaterun.RunIdentity{RunID: meta.RunID, EntityKey: req.EntityKey, EntityType: string(req.EntityType), SourceStatus: gate, Gate: gate, OperationDigest: meta.Digest}
	if identity.RunID != meta.RunID || gaterun.VerifyRunIdentity(identity, want) != nil {
		return fmt.Errorf("approval run identity mismatch")
	}
	return nil
}
func decodeBugApprovalResult(dir string, req gatepersist.Request, meta bugApprovalNote, gate string) ([]byte, error) {
	data, exists, err := gaterun.ReadResult(dir)
	if err != nil || !exists {
		return nil, fmt.Errorf("approval result unavailable")
	}
	digest, err := gaterun.ComputeOperationDigest(req.EntityKey, string(req.EntityType), gate, gate, data)
	if err != nil || digest != meta.Digest {
		return nil, fmt.Errorf("approval result identity mismatch")
	}
	envelope, err := workercontrol.Decode(data)
	if err != nil || envelope.Kind != workercontrol.KindFinal || envelope.RecommendedOutcome != meta.Outcome {
		return nil, fmt.Errorf("approval result is invalid")
	}
	result, err := gateresult.Decode(envelope.GateResult)
	if err != nil || gateresult.ValidateRole(gateresult.RoleSuccess, result, req.EntityKey) != nil {
		return nil, fmt.Errorf("approval gate result is invalid")
	}
	return data, nil
}

func compareBugApprovalEvidence(req gatepersist.Request, data []byte, spec bugPriorSpec) error {
	current, err := workercontrol.Decode(req.EnvelopeJSON)
	if err != nil {
		return err
	}
	prior, err := workercontrol.Decode(data)
	if err != nil {
		return err
	}
	if !matchingBugEvidence(current.Evidence, prior.Evidence, spec.evidenceKind) {
		return fmt.Errorf("current evidence was not approved at prior gate")
	}
	priorRequest := req
	priorRequest.EnvelopeJSON = data
	if spec.gate == "plan_review" {
		return compareBugBaselines(req, priorRequest)
	}
	if spec.gate == "code_review" {
		if err := verifyBugReviewEvidence(priorRequest); err != nil {
			return fmt.Errorf("prior code review evidence: %w", err)
		}
	}
	return nil
}
func matchingBugEvidence(current, prior []workercontrol.EvidenceRef, kind string) bool {
	for _, a := range current {
		if a.Kind != kind {
			continue
		}
		for _, b := range prior {
			if b.Kind == kind && a.Pointer == b.Pointer && strings.EqualFold(a.Summary, b.Summary) {
				return true
			}
		}
	}
	return false
}
func compareBugBaselines(current, prior gatepersist.Request) error {
	a, err := readBugBaseline(current)
	if err != nil {
		return err
	}
	b, err := readBugBaseline(prior)
	if err != nil {
		return err
	}
	if a != b {
		return fmt.Errorf("current baseline was not approved")
	}
	return nil
}

func latestRelevantBugHistory(ctx context.Context, req gatepersist.Request, history gatepersist.HistoryReader, spec bugPriorSpec) (*models.EntityHistory, error) {
	entries, err := history.GetHistory(ctx, req.EntityType, req.EntityKey)
	if err != nil {
		return nil, fmt.Errorf("read bug rework history: %w", err)
	}
	var latest *models.EntityHistory
	for _, entry := range entries {
		if relevantBugHistory(entry, spec) && (latest == nil || entry.ID > latest.ID) {
			latest = entry
		}
	}
	return latest, nil
}
func relevantBugHistory(entry *models.EntityHistory, spec bugPriorSpec) bool {
	if entry == nil {
		return false
	}
	switch entry.ToStatus {
	case "research", "repair_plan", "plan_review":
		return true
	}
	if spec.gate == "code_review" && (entry.ToStatus == "development" || entry.ToStatus == "code_review") {
		return true
	}
	return entry.FromStatus != nil && *entry.FromStatus == spec.gate && entry.ToStatus == spec.target
}
func matchesBugApprovalHistory(entry *models.EntityHistory, spec bugPriorSpec, meta bugApprovalNote) bool {
	if entry == nil || entry.Forced || entry.FromStatus == nil || entry.Notes == nil {
		return false
	}
	return *entry.FromStatus == spec.gate && entry.ToStatus == spec.target && *entry.Notes == gatepersist.BoundTransitionReason(spec.gate, meta.Outcome, meta.RunID, meta.Digest)
}
