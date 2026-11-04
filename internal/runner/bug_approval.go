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

// NewBugGateAdmission binds downstream plan evidence to an actual completed
// independent plan-review result and to the entity's subsequent history.
func NewBugGateAdmission(notes gatepersist.NoteReader, history gatepersist.HistoryReader) func(context.Context, gatepersist.Request) error {
	return func(ctx context.Context, req gatepersist.Request) error {
		if err := BugGateAdmission(ctx, req); err != nil {
			return err
		}
		if req.EntityType != models.EntityTypeBug || req.Role != gateresult.RoleSuccess {
			return nil
		}
		switch req.Gate {
		case "development", "code_review", "qa":
		default:
			return nil
		}
		if notes == nil || history == nil {
			return fmt.Errorf("plan approval readers are required")
		}
		if err := verifyBugPriorGate(ctx, req, notes, history, "plan_review", "bug_repair_plan", "development"); err != nil {
			return err
		}
		if req.Gate == "code_review" || req.Gate == "qa" {
			if _, err := readBugReviewScope(ctx, req); err != nil {
				return err
			}
		}
		if req.Gate == "code_review" {
			if err := verifyBugReviewEvidence(req); err != nil {
				return err
			}
		}
		if req.Gate == "qa" {
			return verifyBugPriorGate(ctx, req, notes, history, "code_review", "bug_review_scope", "qa")
		}
		return nil
	}
}

type bugApprovalNote struct {
	Gate    string `json:"gate"`
	Role    string `json:"role"`
	Outcome string `json:"outcome_key"`
	RunID   string `json:"run_id"`
	Digest  string `json:"operation_digest"`
}

func verifyBugPriorGate(ctx context.Context, req gatepersist.Request, notes gatepersist.NoteReader, history gatepersist.HistoryReader, priorGate, evidenceKind, target string) error {
	current, err := workercontrol.Decode(req.EnvelopeJSON)
	if err != nil {
		return err
	}
	var plan workercontrol.EvidenceRef
	for _, e := range current.Evidence {
		if e.Kind == evidenceKind {
			plan = e
		}
	}
	items, err := notes.ListNotes(ctx, req.EntityType, req.EntityKey, []string{"review"})
	if err != nil {
		return fmt.Errorf("read plan approval notes: %w", err)
	}
	var selected *models.EntityNote
	var meta bugApprovalNote
	for _, note := range items {
		if note == nil || note.Metadata == nil {
			continue
		}
		var m bugApprovalNote
		if json.Unmarshal([]byte(*note.Metadata), &m) != nil || m.Gate != priorGate || m.Role != "success" || m.Outcome != "pass" {
			continue
		}
		if selected == nil || note.ID > selected.ID {
			selected, meta = note, m
		}
	}
	if selected == nil {
		return fmt.Errorf("required prior gate approval is missing")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir)))
	dir, err := gaterun.RunDir(root, meta.RunID)
	if err != nil {
		return fmt.Errorf("resolve approval run: %w", err)
	}
	state, exists, err := gaterun.LoadOperationState(dir)
	if err != nil || !exists {
		return fmt.Errorf("plan approval operation state unavailable")
	}
	if state.PersistenceState != gaterun.PersistenceStateTransitioned || state.EntityType != string(req.EntityType) || !strings.EqualFold(state.EntityKey, req.EntityKey) || state.Gate != priorGate || state.SourceStatus != priorGate || state.OperationDigest != meta.Digest {
		return fmt.Errorf("plan approval operation is not a committed matching review")
	}
	identity, exists, err := gaterun.ReadIdentity(dir)
	if err != nil || !exists {
		return fmt.Errorf("plan approval run identity unavailable")
	}
	wantIdentity := gaterun.RunIdentity{RunID: meta.RunID, EntityKey: req.EntityKey, EntityType: string(req.EntityType), SourceStatus: priorGate, Gate: priorGate, OperationDigest: meta.Digest}
	if identity.RunID != meta.RunID || gaterun.VerifyRunIdentity(identity, wantIdentity) != nil {
		return fmt.Errorf("plan approval run identity mismatch")
	}
	data, exists, err := gaterun.ReadResult(dir)
	if err != nil || !exists {
		return fmt.Errorf("plan approval result unavailable")
	}
	digest, err := gaterun.ComputeOperationDigest(req.EntityKey, string(req.EntityType), priorGate, priorGate, data)
	if err != nil || digest != meta.Digest {
		return fmt.Errorf("plan approval result identity mismatch")
	}
	review, err := workercontrol.Decode(data)
	if err != nil || review.Kind != workercontrol.KindFinal || review.RecommendedOutcome != "pass" {
		return fmt.Errorf("plan approval result is invalid")
	}
	gateResult, err := gateresult.Decode(review.GateResult)
	if err != nil || gateresult.ValidateRole(gateresult.RoleSuccess, gateResult, req.EntityKey) != nil {
		return fmt.Errorf("plan approval gate result is invalid")
	}
	matched := false
	for _, e := range review.Evidence {
		if e.Kind == evidenceKind && e.Pointer == plan.Pointer && e.Summary == plan.Summary {
			matched = true
		}
	}
	if !matched {
		return fmt.Errorf("current evidence was not approved at prior gate")
	}
	if priorGate == "code_review" {
		priorRequest := req
		priorRequest.EnvelopeJSON = data
		if err := verifyBugReviewEvidence(priorRequest); err != nil {
			return fmt.Errorf("prior code review evidence: %w", err)
		}
	}
	entries, err := history.GetHistory(ctx, req.EntityType, req.EntityKey)
	if err != nil {
		return fmt.Errorf("read bug rework history: %w", err)
	}
	var latest *models.EntityHistory
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		relevant := entry.ToStatus == "research" || entry.ToStatus == "repair_plan" || entry.ToStatus == "plan_review"
		if priorGate == "code_review" {
			relevant = relevant || entry.ToStatus == "development" || entry.ToStatus == "code_review"
		}
		if !relevant && !(entry.FromStatus != nil && *entry.FromStatus == priorGate && entry.ToStatus == target) {
			continue
		}
		if latest == nil || entry.ID > latest.ID {
			latest = entry
		}
	}
	if latest == nil || latest.Forced || latest.FromStatus == nil || *latest.FromStatus != priorGate || latest.ToStatus != target || latest.Notes == nil || *latest.Notes != "gate "+priorGate+" outcome pass" {
		return fmt.Errorf("prior gate approval invalidated or not applied in history")
	}
	return nil
}
