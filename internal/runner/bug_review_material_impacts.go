package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

// bugReviewMaterialImpacts is frozen with the candidate-bound review report.
// The separate disposition packet is checked against this set at admission;
// it is not represented as GateResult.change_impacts (which is planning-only).
type bugReviewMaterialImpacts struct {
	CandidateIdentity bugReviewIdentity           `json:"candidate_identity"`
	Decisions         []bugReviewMaterialDecision `json:"decisions"`
}

type bugReviewMaterialDecision struct {
	DecisionID        string                       `json:"decision_id"`
	SourcePointer     string                       `json:"source_pointer"`
	SourceSHA256      string                       `json:"source_sha256"`
	AffectedArtifacts []bugReviewMaterialArtifact  `json:"affected_artifacts"`
	AffectedCriteria  []bugReviewMaterialCriterion `json:"affected_criteria"`
}

type bugReviewMaterialArtifact struct {
	Path            string `json:"path"`
	InvalidatedText string `json:"invalidated_text"`
}

type bugReviewMaterialCriterion struct {
	EntityKey             string `json:"entity_key"`
	ProductionCallerPath  string `json:"production_caller_path"`
	AcceptanceCriterionID string `json:"acceptance_criterion_id"`
	Assertion             string `json:"assertion"`
}

type bugReviewMaterialDispositions struct {
	CandidateIdentity bugReviewIdentity              `json:"candidate_identity"`
	Decisions         []bugReviewMaterialDisposition `json:"decisions"`
}

type bugReviewMaterialDisposition struct {
	DecisionID string                          `json:"decision_id"`
	Artifacts  []bugReviewArtifactDisposition  `json:"artifacts"`
	Criteria   []bugReviewCriterionDisposition `json:"criteria"`
}

type bugReviewArtifactDisposition struct {
	Path            string `json:"path"`
	Disposition     string `json:"disposition"`
	EvidencePointer string `json:"evidence_pointer"`
}

type bugReviewCriterionDisposition struct {
	EntityKey             string `json:"entity_key"`
	ProductionCallerPath  string `json:"production_caller_path"`
	AcceptanceCriterionID string `json:"acceptance_criterion_id"`
	Disposition           string `json:"disposition"`
	EvidencePointer       string `json:"evidence_pointer"`
}

// bugParentTriageObservation is injected by the parent after observing the
// host triage action. It is carried as envelope evidence by the host adapter,
// never accepted from the dispatched worker's original envelope.
type bugParentTriageObservation struct {
	CandidateIdentity bugReviewIdentity       `json:"candidate_identity"`
	ObservedBy        string                  `json:"observed_by"`
	Actions           []bugParentTriageAction `json:"actions"`
}

type bugParentTriageAction struct {
	FindingID     string `json:"finding_id"`
	Action        string `json:"action"`
	Edits         *int   `json:"edits"`
	TrackerWrites *int   `json:"tracker_writes"`
}

func validateBugReviewMaterialImpacts(ctx context.Context, req gatepersist.Request, scope bugCandidateScope, header, body string) error {
	markers := materialDecisionMarkers(body)
	present, err := materialPacketsPresent(req, header, markers)
	if err != nil {
		return err
	}
	var enumerated map[string]*frozenMaterialDecision
	independent := scope.ProjectRoot != ""
	if independent {
		if enumerated, err = enumerateFrozenMaterialDecisions(ctx, scope); err != nil {
			return err
		}
	}
	if !present {
		if len(enumerated) > 0 {
			return fmt.Errorf("frozen source declares material decisions; expected impacts and dispositions are required")
		}
		return nil
	}
	var expected bugReviewMaterialImpacts
	if err := readMaterialPacket(req, header, "bug_review_material_impacts", "Expected material impacts", &expected); err != nil {
		return err
	}
	var submitted bugReviewMaterialDispositions
	if err := readMaterialPacket(req, header, "bug_review_material_dispositions", "Material dispositions", &submitted); err != nil {
		return err
	}
	if !expected.CandidateIdentity.matches(scope) || !submitted.CandidateIdentity.matches(scope) {
		return fmt.Errorf("material decision evidence candidate mismatch")
	}
	if len(expected.Decisions) == 0 || len(expected.Decisions) != len(submitted.Decisions) {
		return fmt.Errorf("material decision disposition set is incomplete")
	}
	if independent {
		if err := compareFrozenMaterialDecisions(enumerated, expected.Decisions); err != nil {
			return err
		}
		markers = nil // the frozen source, not the report, controls the set
	}
	dispositions, err := indexMaterialDispositions(submitted.Decisions)
	if err != nil {
		return err
	}
	return verifyMaterialDecisionSet(ctx, req, scope, expected.Decisions, dispositions, markers)
}

func materialPacketsPresent(req gatepersist.Request, header string, markers map[string]bool) (bool, error) {
	envelope, err := workercontrol.Decode(req.EnvelopeJSON)
	if err != nil {
		return false, err
	}
	counts := make(map[string]int)
	for _, ref := range envelope.Evidence {
		counts[ref.Kind]++
	}
	impacts, dispositions := counts["bug_review_material_impacts"], counts["bug_review_material_dispositions"]
	if impacts == 1 && dispositions == 1 {
		return true, nil
	}
	declared := len(markers) > 0 || strings.Contains(header, "Expected material impacts SHA-256") || strings.Contains(header, "Material dispositions SHA-256")
	if impacts != 0 || dispositions != 0 || declared {
		return false, fmt.Errorf("material decision evidence requires both expected impacts and dispositions")
	}
	return false, nil
}

func readMaterialPacket(req gatepersist.Request, header, kind, label string, target any) error {
	_, data, err := readBugEvidence(req, kind)
	if err != nil {
		return err
	}
	if err := verifyReviewArtifactHash(header, label, data); err != nil {
		return err
	}
	if err := decodeStrict(data, target); err != nil {
		return fmt.Errorf("invalid %s: %w", strings.ToLower(label), err)
	}
	return nil
}

func indexMaterialDispositions(submitted []bugReviewMaterialDisposition) (map[string]bugReviewMaterialDisposition, error) {
	dispositions := make(map[string]bugReviewMaterialDisposition, len(submitted))
	for _, item := range submitted {
		if strings.TrimSpace(item.DecisionID) == "" {
			return nil, fmt.Errorf("material disposition decision identity is missing")
		}
		if _, ok := dispositions[item.DecisionID]; ok {
			return nil, fmt.Errorf("duplicate material disposition decision")
		}
		dispositions[item.DecisionID] = item
	}
	return dispositions, nil
}

func verifyMaterialDecisionSet(ctx context.Context, req gatepersist.Request, scope bugCandidateScope, expected []bugReviewMaterialDecision, dispositions map[string]bugReviewMaterialDisposition, markers map[string]bool) error {
	seenDecisions := make(map[string]bool, len(expected))
	for _, decision := range expected {
		if err := validateBugReviewMaterialDecision(decision); err != nil {
			return err
		}
		if err := verifyBugMaterialDecisionSource(ctx, scope, decision); err != nil {
			return err
		}
		if seenDecisions[decision.DecisionID] {
			return fmt.Errorf("duplicate expected material decision")
		}
		seenDecisions[decision.DecisionID] = true
		submittedDecision, ok := dispositions[decision.DecisionID]
		if !ok {
			return fmt.Errorf("material decision disposition is missing")
		}
		if err := compareBugReviewMaterialMembers(decision, submittedDecision); err != nil {
			return err
		}
		if err := verifyMaterialSupport(ctx, req, scope, decision, submittedDecision); err != nil {
			return err
		}
	}
	if markers != nil && !maps.Equal(markers, seenDecisions) {
		return fmt.Errorf("frozen report material decisions do not match expected impact set")
	}
	return nil
}

func materialDecisionMarkers(body string) map[string]bool {
	markers := make(map[string]bool)
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(line, "MATERIAL_DECISION: ") {
			continue
		}
		id := strings.TrimSpace(strings.TrimPrefix(line, "MATERIAL_DECISION: "))
		if id != "" {
			markers[id] = true
		}
	}
	return markers
}

func validateBugParentTriageObservation(req gatepersist.Request, scope bugCandidateScope, decisions map[bugFindingKey]bugReviewDecision) error {
	return validateBugParentTriageObservationRaw(req, scope, decisions, nil)
}

// validateBugParentTriageObservationRaw derives the expected host actions from
// the raw angle findings together with their decisions, so relabeling a raw
// blocker cannot suppress the host action it requires.
func validateBugParentTriageObservationRaw(req gatepersist.Request, scope bugCandidateScope, decisions map[bugFindingKey]bugReviewDecision, raw map[bugFindingKey]string) error {
	expected, err := expectedBugHostActions(decisions, raw)
	if err != nil || len(expected) == 0 {
		return err
	}
	if !req.ParentObservationInjected {
		return fmt.Errorf("parent-observed report-only triage receipt is required")
	}
	_, data, err := readBugEvidence(req, "bug_parent_triage_observation")
	if err != nil {
		return fmt.Errorf("parent-observed report-only triage receipt is required: %w", err)
	}
	var observation bugParentTriageObservation
	if err := decodeStrict(data, &observation); err != nil {
		return fmt.Errorf("invalid parent triage observation: %w", err)
	}
	if !observation.CandidateIdentity.matches(scope) || observation.ObservedBy != "parent" {
		return fmt.Errorf("parent triage observation does not match frozen findings")
	}
	return checkHostActions(expected, observation.Actions)
}

const (
	bugHostActionReportOnly = "report_only"
	bugHostActionTriage     = "triage"
)

// expectedBugHostActions maps each finding that needs a host action to that
// action. A report-only nit is observed as report_only; a raw blocker that was
// reclassified without refutation or a report-only withdrawal requires triage.
func expectedBugHostActions(decisions map[bugFindingKey]bugReviewDecision, raw map[bugFindingKey]string) (map[string]string, error) {
	expected := make(map[string]string)
	for key, decision := range decisions {
		action := ""
		switch {
		// The Rider writer uses report-only. Earlier B087 packets used report_only.
		case decision.Severity != "blocker" && (decision.Disposition == "report-only" || decision.Disposition == "report_only"):
			action = bugHostActionReportOnly
		case raw[key.site()] == "blocker" && decision.Severity != "blocker" && decision.Disposition != "refuted":
			action = bugHostActionTriage
		default:
			continue
		}
		if strings.TrimSpace(decision.FindingID) == "" {
			return nil, fmt.Errorf("report-only finding ID is required")
		}
		if _, exists := expected[decision.FindingID]; exists {
			return nil, fmt.Errorf("duplicate report-only finding ID %s", decision.FindingID)
		}
		expected[decision.FindingID] = action
	}
	return expected, nil
}

func checkHostActions(expected map[string]string, actions []bugParentTriageAction) error {
	observed := make(map[string]bool, len(expected))
	for _, action := range actions {
		want, ok := expected[action.FindingID]
		if !ok {
			return fmt.Errorf("unexpected report-only observation for %q", action.FindingID)
		}
		if observed[action.FindingID] {
			return fmt.Errorf("duplicate report-only observation for %s", action.FindingID)
		}
		if err := action.verify(want); err != nil {
			return err
		}
		observed[action.FindingID] = true
	}
	for id, want := range expected {
		if !observed[id] {
			if want == bugHostActionTriage {
				return fmt.Errorf("missing triage observation for %s", id)
			}
			return fmt.Errorf("missing report-only observation for %s", id)
		}
	}
	return nil
}

func (action bugParentTriageAction) verify(want string) error {
	if action.Action != want {
		return fmt.Errorf("%s: expected %s host action", action.FindingID, want)
	}
	if want == bugHostActionTriage {
		if action.Edits == nil || action.TrackerWrites == nil || *action.Edits < 0 || *action.TrackerWrites < 0 {
			return fmt.Errorf("%s: triage requires observed edits and tracker_writes counters", action.FindingID)
		}
		return nil
	}
	if action.Edits == nil || *action.Edits != 0 {
		return fmt.Errorf("%s: report-only requires observed edits of zero", action.FindingID)
	}
	if action.TrackerWrites == nil || *action.TrackerWrites != 0 {
		return fmt.Errorf("%s: report-only requires observed tracker_writes of zero", action.FindingID)
	}
	return nil
}

// rawBugFindingSeverities reads the six-angle raw findings, keeping the most
// severe label when angles repeat a finding.
func rawBugFindingSeverities(req gatepersist.Request) (map[bugFindingKey]string, error) {
	_, data, err := readBugEvidence(req, "bug_review_angles")
	if err != nil {
		return nil, err
	}
	return rawSeveritiesFromAngles(data)
}

func rawSeveritiesFromAngles(data []byte) (map[bugFindingKey]string, error) {
	var angles map[string]bugAngle
	if err := json.Unmarshal(data, &angles); err != nil {
		return nil, err
	}
	rank := map[string]int{"nit": 1, "non-blocker": 2, "blocker": 3}
	raw := make(map[bugFindingKey]string)
	for _, angle := range angles {
		for _, f := range angle.Findings {
			key := f.site()
			if rank[f.Severity] > rank[raw[key]] {
				raw[key] = f.Severity
			}
		}
	}
	return raw, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	return nil
}

func validateBugReviewMaterialDecision(decision bugReviewMaterialDecision) error {
	if strings.TrimSpace(decision.DecisionID) == "" || !filepath.IsLocal(decision.SourcePointer) {
		return fmt.Errorf("material decision source identity is invalid")
	}
	digest, err := hex.DecodeString(decision.SourceSHA256)
	if err != nil || len(digest) != sha256.Size || len(decision.AffectedArtifacts) == 0 && len(decision.AffectedCriteria) == 0 {
		return fmt.Errorf("material decision source or expected members are invalid")
	}
	if err := validateMaterialArtifacts(decision.AffectedArtifacts); err != nil {
		return err
	}
	return validateMaterialCriteria(decision.AffectedCriteria)
}

func validateMaterialArtifacts(affected []bugReviewMaterialArtifact) error {
	artifacts := make(map[string]bool, len(affected))
	for _, artifact := range affected {
		if !filepath.IsLocal(artifact.Path) || strings.TrimSpace(artifact.InvalidatedText) == "" || artifacts[artifact.Path] {
			return fmt.Errorf("material decision artifact set is invalid")
		}
		artifacts[artifact.Path] = true
	}
	return nil
}

func validateMaterialCriteria(affected []bugReviewMaterialCriterion) error {
	criteria := make(map[materialCriterionKey]bool, len(affected))
	for _, criterion := range affected {
		key := materialCriterionKey{criterion.EntityKey, criterion.ProductionCallerPath, criterion.AcceptanceCriterionID}
		if criterion.EntityKey == "" || !filepath.IsLocal(criterion.ProductionCallerPath) || criterion.AcceptanceCriterionID == "" || strings.TrimSpace(criterion.Assertion) == "" || criteria[key] {
			return fmt.Errorf("material decision acceptance criterion set is invalid")
		}
		criteria[key] = true
	}
	return nil
}

func compareBugReviewMaterialMembers(expected bugReviewMaterialDecision, actual bugReviewMaterialDisposition) error {
	if err := compareMaterialArtifacts(expected, actual.Artifacts); err != nil {
		return err
	}
	return compareMaterialCriteria(expected, actual.Criteria)
}

func compareMaterialArtifacts(expected bugReviewMaterialDecision, submitted []bugReviewArtifactDisposition) error {
	seen := make(map[string]bool, len(submitted))
	for _, item := range submitted {
		if item.Path == "" || seen[item.Path] || !filepath.IsLocal(item.EvidencePointer) {
			return fmt.Errorf("%s: invalid or duplicate artifact disposition for %s", expected.DecisionID, item.Path)
		}
		if item.Disposition != "amended" && item.Disposition != "tracked_follow_up" {
			return fmt.Errorf("%s: invalid artifact disposition for %s", expected.DecisionID, item.Path)
		}
		seen[item.Path] = true
	}
	for _, item := range expected.AffectedArtifacts {
		if !seen[item.Path] {
			return fmt.Errorf("%s: missing artifact disposition for %s", expected.DecisionID, item.Path)
		}
		delete(seen, item.Path)
	}
	for path := range seen {
		return fmt.Errorf("%s: unexpected artifact disposition for %s", expected.DecisionID, path)
	}
	return nil
}

type materialCriterionKey struct {
	owner, caller, criterion string
}

func (k materialCriterionKey) String() string {
	return k.owner + " " + k.caller + " " + k.criterion
}

func compareMaterialCriteria(expected bugReviewMaterialDecision, submitted []bugReviewCriterionDisposition) error {
	seen := make(map[materialCriterionKey]bool, len(submitted))
	for _, item := range submitted {
		key := materialCriterionKey{item.EntityKey, item.ProductionCallerPath, item.AcceptanceCriterionID}
		if seen[key] || !filepath.IsLocal(item.EvidencePointer) {
			return fmt.Errorf("%s: invalid or duplicate criterion disposition for %s", expected.DecisionID, key)
		}
		if item.Disposition != "regression_pass" && item.Disposition != "tracked_follow_up" {
			return fmt.Errorf("%s: invalid criterion disposition for %s", expected.DecisionID, key)
		}
		seen[key] = true
	}
	for _, item := range expected.AffectedCriteria {
		key := materialCriterionKey{item.EntityKey, item.ProductionCallerPath, item.AcceptanceCriterionID}
		if !seen[key] {
			return fmt.Errorf("%s: missing criterion disposition for %s", expected.DecisionID, key)
		}
		delete(seen, key)
	}
	for key := range seen {
		return fmt.Errorf("%s: unexpected criterion disposition for %s", expected.DecisionID, key)
	}
	return nil
}
