package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// PrepareBugApprovalSnapshot retains the hash-verified plan under the bound
// plan-review run. Downstream consumers use these bytes as the approved plan.
func PrepareBugApprovalSnapshot(ctx context.Context, req gatepersist.Request) error {
	if req.EntityType != models.EntityTypeBug || req.Gate != "plan_review" || req.Role != gateresult.RoleSuccess {
		return nil
	}
	_, plan, _, err := committedBugPlan(ctx, req)
	if err != nil {
		return fmt.Errorf("read approved plan: %w", err)
	}
	_, _, blob, err := bugPlanProvenance(ctx, req)
	if err != nil || blob != req.PlanBlob {
		return fmt.Errorf("approved plan commit binding changed before transition")
	}
	_, err = gaterun.CreateApprovedPlan(req.RunDir, plan)
	return err
}

// BugDocumentReader resolves documents linked to the bug in the parent store.
type BugDocumentReader interface {
	ListRelatedDocumentsByKey(context.Context, string) ([]*models.Document, error)
}

// PrepareBugPlanApproval requires a linked, digest-bound independent review
// report before retaining the reviewed plan revision.
func PrepareBugPlanApproval(ctx context.Context, req gatepersist.Request, documents BugDocumentReader) error {
	if err := ValidateBugPlanReviewReport(ctx, req, documents); err != nil {
		return err
	}
	return PrepareBugApprovalSnapshot(ctx, req)
}

// ValidateBugPlanReviewReport checks the registered report before persistence.
func ValidateBugPlanReviewReport(ctx context.Context, req gatepersist.Request, documents BugDocumentReader) error {
	if req.EntityType != models.EntityTypeBug || req.Gate != "plan_review" || req.Role != gateresult.RoleSuccess {
		return nil
	}
	if documents == nil {
		return fmt.Errorf("plan review document reader is required")
	}
	ref, report, err := readBugEvidence(req, "plan_review_report")
	if err != nil {
		return fmt.Errorf("plan review report: %w", err)
	}
	if len(report) == 0 {
		return fmt.Errorf("plan review report is empty")
	}
	planRef, err := findBugEvidence(req, "bug_repair_plan")
	if err != nil {
		return err
	}
	fields := map[string][]string{}
	for _, line := range strings.Split(string(report), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && (strings.EqualFold(key, "plan_path") || strings.EqualFold(key, "plan_sha256")) {
			fields[strings.ToLower(key)] = append(fields[strings.ToLower(key)], strings.TrimSpace(value))
		}
	}
	if len(fields["plan_path"]) != 1 || len(fields["plan_sha256"]) != 1 || fields["plan_path"][0] != planRef.Pointer || !strings.EqualFold(fields["plan_sha256"][0], planRef.Summary) {
		return fmt.Errorf("plan review report must identify exactly the approved plan path and sha256 digest")
	}
	linked, err := documents.ListRelatedDocumentsByKey(ctx, req.EntityKey)
	if err != nil {
		return fmt.Errorf("list plan review documents: %w", err)
	}
	for _, doc := range linked {
		if doc != nil && filepath.Clean(doc.FilePath) == filepath.Clean(ref.Pointer) {
			return nil
		}
	}
	return fmt.Errorf("plan review report %q is not linked to bug %s", ref.Pointer, req.EntityKey)
}

func verifyApprovedBugPlan(dir string, envelope []byte) ([]byte, error) {
	req := gatepersist.Request{EnvelopeJSON: envelope}
	ref, err := findBugEvidence(req, "bug_repair_plan")
	if err != nil {
		return nil, err
	}
	expected, err := validateBugEvidenceRef(ref)
	if err != nil {
		return nil, err
	}
	plan, exists, err := gaterun.ReadApprovedPlan(dir)
	if err != nil || !exists {
		return nil, fmt.Errorf("approved plan snapshot unavailable")
	}
	hash := sha256.Sum256(plan)
	if !strings.EqualFold(hex.EncodeToString(expected), hex.EncodeToString(hash[:])) {
		return nil, fmt.Errorf("approved plan snapshot digest mismatch")
	}
	return plan, nil
}

// ApprovedBugPlan resolves the latest applied, uninvalidated plan approval and
// returns the digest-verified snapshot for dispatch to downstream workers.
type ApprovedPlan struct {
	Pointer string
	Bytes   []byte
}

func ApprovedBugPlan(ctx context.Context, projectRoot, key string, notes gatepersist.NoteReader, history gatepersist.HistoryReader, normalize func(string) string) (ApprovedPlan, error) {
	if notes == nil || history == nil {
		return ApprovedPlan{}, fmt.Errorf("plan approval readers are required")
	}
	if normalize == nil {
		normalize = func(value string) string { return value }
	}
	req := gatepersist.Request{EntityType: models.EntityTypeBug, EntityKey: key, RunDir: filepath.Join(projectRoot, ".shark", "runs", "dispatch")}
	spec := bugPriorSpec{"plan_review", "bug_repair_plan", "development"}
	entry, err := latestRelevantBugHistory(ctx, req, history, spec, normalize)
	if err != nil {
		return ApprovedPlan{}, err
	}
	meta, err := latestBugApproval(ctx, req, notes, spec.gate, entry)
	if err != nil {
		return ApprovedPlan{}, err
	}
	if !matchesBugApprovalHistory(entry, spec, meta, normalize) {
		return ApprovedPlan{}, fmt.Errorf("prior gate approval invalidated or not applied in history")
	}
	data, _, err := readBugApprovalResult(ctx, req, meta, spec.gate)
	if err != nil {
		return ApprovedPlan{}, err
	}
	ref, err := findBugEvidence(gatepersist.Request{EnvelopeJSON: data}, "bug_repair_plan")
	if err != nil {
		return ApprovedPlan{}, err
	}
	dir, err := gaterun.RunDir(projectRoot, meta.RunID)
	if err != nil {
		return ApprovedPlan{}, err
	}
	plan, err := verifyApprovedBugPlan(dir, data)
	return ApprovedPlan{Pointer: ref.Pointer, Bytes: plan}, err
}

// AppendApprovedBugPlan places the approved bytes in the dispatched prompt.
func AppendApprovedBugPlan(prompt string, plan ApprovedPlan) string {
	hash := sha256.Sum256(plan.Bytes)
	return fmt.Sprintf("%s\n\nThe following is the authoritative approved repair plan at %s (sha256:%x). Use these exact bytes for the approved approach and carry that path and digest in your evidence; a material change requires a new plan review.\n\n%s\n", prompt, plan.Pointer, hash, plan.Bytes)
}

// BugNeedsApprovedPlan identifies worker stages downstream of plan approval.
func BugNeedsApprovedPlan(status string) bool {
	switch strings.ToLower(status) {
	case "development", "code_review", "qa":
		return true
	default:
		return false
	}
}

// NewBugGateAdmission binds downstream evidence to completed prior gates.
// Production supplies the parent workflow's bug-status normalizer; callers
// without one retain canonical-only matching for existing integrations.
func NewBugGateAdmission(notes gatepersist.NoteReader, history gatepersist.HistoryReader, normalize ...func(string) string) func(context.Context, gatepersist.Request) error {
	status := func(value string) string { return value }
	if len(normalize) > 0 && normalize[0] != nil {
		status = normalize[0]
	}
	return func(ctx context.Context, req gatepersist.Request) error {
		if err := BugGateAdmission(ctx, req); err != nil {
			return err
		}
		if req.EntityType != models.EntityTypeBug || req.Role != gateresult.RoleSuccess {
			return nil
		}
		return admitBugDownstream(ctx, req, notes, history, status)
	}
}

func admitBugDownstream(ctx context.Context, req gatepersist.Request, notes gatepersist.NoteReader, history gatepersist.HistoryReader, normalize func(string) string) error {
	switch req.Gate {
	case "development", "code_review", "qa":
	default:
		return nil
	}
	if notes == nil || history == nil {
		return fmt.Errorf("plan approval readers are required")
	}
	if err := verifyBugPriorGate(ctx, req, notes, history, bugPriorSpec{"plan_review", "bug_repair_plan", "development"}, normalize); err != nil {
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
	return verifyBugPriorGate(ctx, req, notes, history, bugPriorSpec{"code_review", "bug_review_scope", "qa"}, normalize)
}

type bugApprovalNote struct {
	Gate    string `json:"gate"`
	Role    string `json:"role"`
	Outcome string `json:"outcome_key"`
	RunID   string `json:"run_id"`
	Digest  string `json:"operation_digest"`
}
type bugPriorSpec struct{ gate, evidenceKind, target string }

func verifyBugPriorGate(ctx context.Context, req gatepersist.Request, notes gatepersist.NoteReader, history gatepersist.HistoryReader, spec bugPriorSpec, normalize func(string) string) error {
	entry, err := latestRelevantBugHistory(ctx, req, history, spec, normalize)
	if err != nil {
		return err
	}
	meta, err := latestBugApproval(ctx, req, notes, spec.gate, entry)
	if err != nil {
		return err
	}
	if !matchesBugApprovalHistory(entry, spec, meta, normalize) {
		return fmt.Errorf("prior gate approval invalidated or not applied in history")
	}
	data, evidenceRoot, err := readBugApprovalResult(ctx, req, meta, spec.gate)
	if err != nil {
		return err
	}
	if err := compareBugApprovalEvidence(req, data, evidenceRoot, spec); err != nil {
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

func readBugApprovalResult(ctx context.Context, req gatepersist.Request, meta bugApprovalNote, gate string) ([]byte, string, error) {
	root := filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir)))
	dir, err := gaterun.RunDir(root, meta.RunID)
	if err != nil {
		return nil, "", fmt.Errorf("resolve approval run: %w", err)
	}
	state, exists, err := gaterun.LoadOperationState(dir)
	if err != nil || !exists {
		return nil, "", fmt.Errorf("approval operation state unavailable")
	}
	if !matchesBugApprovalState(state, req, meta, gate) {
		return nil, "", fmt.Errorf("approval operation is not a committed matching review")
	}
	evidenceRoot, err := verifyBugApprovalIdentity(ctx, dir, req, meta, gate)
	if err != nil {
		return nil, "", err
	}
	data, err := decodeBugApprovalResult(dir, req, meta, gate)
	if err == nil && gate == "plan_review" {
		_, err = verifyApprovedBugPlan(dir, data)
		if err == nil {
			err = verifyBugPlanProvenance(ctx, filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir))), dir, data)
		}
	}
	return data, evidenceRoot, err
}

func matchesBugApprovalState(state *gaterun.OperationState, req gatepersist.Request, meta bugApprovalNote, gate string) bool {
	return state.AdmissionVersion == gatepersist.BoundAdmissionVersion && state.PersistenceState == gaterun.PersistenceStateTransitioned && state.EntityType == string(req.EntityType) && strings.EqualFold(state.EntityKey, req.EntityKey) && state.Gate == gate && state.SourceStatus == gate && state.OperationDigest == meta.Digest
}
func verifyBugApprovalIdentity(ctx context.Context, dir string, req gatepersist.Request, meta bugApprovalNote, gate string) (string, error) {
	identity, exists, err := gaterun.ReadIdentity(dir)
	if err != nil || !exists {
		return "", fmt.Errorf("approval run identity unavailable")
	}
	want := gaterun.RunIdentity{RunID: meta.RunID, EntityKey: req.EntityKey, EntityType: string(req.EntityType), SourceStatus: gate, Gate: gate, OperationDigest: meta.Digest, EvidenceRoot: identity.EvidenceRoot, PlanRoot: identity.PlanRoot, PlanCommit: identity.PlanCommit, PlanBlob: identity.PlanBlob}
	if identity.RunID != meta.RunID || gaterun.VerifyRunIdentity(identity, want) != nil {
		return "", fmt.Errorf("approval run identity mismatch")
	}
	if identity.EvidenceRoot == "" {
		return "", nil
	}
	if gate == "plan_review" {
		return identity.EvidenceRoot, nil
	}
	parent := filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir)))
	verified, err := resolveBugEvidenceRoot(ctx, parent, identity.EvidenceRoot)
	if err != nil || verified != identity.EvidenceRoot {
		return "", fmt.Errorf("approval evidence root is unavailable or invalid")
	}
	return verified, nil
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

func compareBugApprovalEvidence(req gatepersist.Request, data []byte, evidenceRoot string, spec bugPriorSpec) error {
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
	priorRequest.EvidenceRoot = evidenceRoot
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

func latestRelevantBugHistory(ctx context.Context, req gatepersist.Request, history gatepersist.HistoryReader, spec bugPriorSpec, normalize func(string) string) (*models.EntityHistory, error) {
	entries, err := history.GetHistory(ctx, req.EntityType, req.EntityKey)
	if err != nil {
		return nil, fmt.Errorf("read bug rework history: %w", err)
	}
	var latest *models.EntityHistory
	for _, entry := range entries {
		if relevantBugHistory(entry, spec, normalize) && (latest == nil || entry.ID > latest.ID) {
			latest = entry
		}
	}
	return latest, nil
}
func relevantBugHistory(entry *models.EntityHistory, spec bugPriorSpec, normalize func(string) string) bool {
	if entry == nil {
		return false
	}
	target := strings.ToLower(normalize(entry.ToStatus))
	switch target {
	case "research", "repair_plan", "plan_review":
		return true
	}
	if spec.gate == "code_review" && (target == "development" || target == "code_review") {
		return true
	}
	return entry.FromStatus != nil && strings.EqualFold(normalize(*entry.FromStatus), spec.gate) && strings.EqualFold(target, spec.target)
}
func matchesBugApprovalHistory(entry *models.EntityHistory, spec bugPriorSpec, meta bugApprovalNote, normalize func(string) string) bool {
	if entry == nil || entry.Forced || entry.FromStatus == nil || entry.Notes == nil {
		return false
	}
	return strings.EqualFold(normalize(*entry.FromStatus), spec.gate) && strings.EqualFold(normalize(entry.ToStatus), spec.target) && *entry.Notes == gatepersist.BoundTransitionReason(spec.gate, meta.Outcome, meta.RunID, meta.Digest)
}
