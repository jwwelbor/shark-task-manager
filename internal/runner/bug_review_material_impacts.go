package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gitutil"
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
	Edits         int    `json:"edits"`
	TrackerWrites int    `json:"tracker_writes"`
}

func validateBugReviewMaterialImpacts(req gatepersist.Request, scope bugCandidateScope, header, body string) error {
	envelope, err := workercontrol.Decode(req.EnvelopeJSON)
	if err != nil {
		return err
	}
	expectedCount, dispositionCount := 0, 0
	for _, ref := range envelope.Evidence {
		switch ref.Kind {
		case "bug_review_material_impacts":
			expectedCount++
		case "bug_review_material_dispositions":
			dispositionCount++
		}
	}
	if expectedCount == 0 && dispositionCount == 0 {
		// Reviews with no material decisions need no separate impact evidence.
		if len(materialDecisionMarkers(body)) != 0 || strings.Contains(header, "Expected material impacts SHA-256") || strings.Contains(header, "Material dispositions SHA-256") {
			return fmt.Errorf("material decision evidence requires both expected impacts and dispositions")
		}
		return nil
	}
	if expectedCount != 1 || dispositionCount != 1 {
		return fmt.Errorf("material decision evidence requires both expected impacts and dispositions")
	}
	_, expectedData, err := readBugEvidence(req, "bug_review_material_impacts")
	if err != nil {
		return err
	}
	_, dispositionData, err := readBugEvidence(req, "bug_review_material_dispositions")
	if err != nil {
		return err
	}
	if err := verifyReviewArtifactHash(header, "Expected material impacts", expectedData); err != nil {
		return err
	}
	if err := verifyReviewArtifactHash(header, "Material dispositions", dispositionData); err != nil {
		return err
	}
	var expected bugReviewMaterialImpacts
	if err := decodeStrict(expectedData, &expected); err != nil {
		return fmt.Errorf("invalid expected material impacts: %w", err)
	}
	var submitted bugReviewMaterialDispositions
	if err := decodeStrict(dispositionData, &submitted); err != nil {
		return fmt.Errorf("invalid material dispositions: %w", err)
	}
	if !expected.CandidateIdentity.matches(scope) || !submitted.CandidateIdentity.matches(scope) {
		return fmt.Errorf("material decision evidence candidate mismatch")
	}
	if len(expected.Decisions) == 0 || len(expected.Decisions) != len(submitted.Decisions) {
		return fmt.Errorf("material decision disposition set is incomplete")
	}
	dispositions := make(map[string]bugReviewMaterialDisposition, len(submitted.Decisions))
	for _, item := range submitted.Decisions {
		if strings.TrimSpace(item.DecisionID) == "" {
			return fmt.Errorf("material disposition decision identity is missing")
		}
		if _, ok := dispositions[item.DecisionID]; ok {
			return fmt.Errorf("duplicate material disposition decision")
		}
		dispositions[item.DecisionID] = item
	}
	seenDecisions := make(map[string]bool, len(expected.Decisions))
	for _, decision := range expected.Decisions {
		if err := validateBugReviewMaterialDecision(decision); err != nil {
			return err
		}
		if err := verifyBugMaterialDecisionSource(scope, decision); err != nil {
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
	}
	markers := materialDecisionMarkers(body)
	if len(markers) != len(seenDecisions) {
		return fmt.Errorf("frozen report material decisions do not match expected impact set")
	}
	for id := range markers {
		if !seenDecisions[id] {
			return fmt.Errorf("frozen report material decisions do not match expected impact set")
		}
	}
	// Reject worker-supplied dispositions for decisions outside the frozen set.
	for id := range dispositions {
		if !seenDecisions[id] {
			return fmt.Errorf("material disposition has no frozen decision")
		}
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

func verifyBugMaterialDecisionSource(scope bugCandidateScope, decision bugReviewMaterialDecision) error {
	root, _, err := candidateRepository(context.Background(), scope.ProjectRoot)
	if err != nil {
		return fmt.Errorf("resolve material decision source repository: %w", err)
	}
	path := filepath.ToSlash(decision.SourcePointer)
	data, err := gitutil.Command(context.Background(), "-C", root, "show", scope.CandidateCommit+":"+path).Output()
	if err != nil {
		return fmt.Errorf("read material decision source from frozen candidate: %w", err)
	}
	digest := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), decision.SourceSHA256) {
		return fmt.Errorf("material decision source digest does not match frozen candidate")
	}
	return nil
}

func validateBugParentTriageObservation(req gatepersist.Request, scope bugCandidateScope, decisions map[bugFindingKey]bugReviewDecision) error {
	required := make(map[string]bool)
	for _, decision := range decisions {
		if decision.Severity == "nit" && decision.Disposition == "report_only" {
			required[decision.FindingID] = true
		}
	}
	if len(required) == 0 {
		return nil
	}
	if !req.ParentObservationInjected {
		return fmt.Errorf("parent-observed report-only triage receipt is required")
	}
	_, data, err := readBugEvidence(req, "bug_parent_triage_observation")
	if err != nil {
		return fmt.Errorf("parent-observed report-only triage receipt is required")
	}
	var observation bugParentTriageObservation
	if err := decodeStrict(data, &observation); err != nil {
		return fmt.Errorf("invalid parent triage observation: %w", err)
	}
	if !observation.CandidateIdentity.matches(scope) || observation.ObservedBy != "parent" || len(observation.Actions) != len(required) {
		return fmt.Errorf("parent triage observation does not match frozen findings")
	}
	seen := make(map[string]bool, len(observation.Actions))
	for _, action := range observation.Actions {
		if !required[action.FindingID] || seen[action.FindingID] || action.Action != "report_only" || action.Edits != 0 || action.TrackerWrites != 0 {
			return fmt.Errorf("parent triage observation does not prove report-only disposition")
		}
		seen[action.FindingID] = true
	}
	return nil
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
	if strings.TrimSpace(decision.DecisionID) == "" || !filepath.IsLocal(decision.SourcePointer) || decision.SourcePointer == "" {
		return fmt.Errorf("material decision source identity is invalid")
	}
	digest, err := hex.DecodeString(decision.SourceSHA256)
	if err != nil || len(digest) != sha256.Size || len(decision.AffectedArtifacts) == 0 && len(decision.AffectedCriteria) == 0 {
		return fmt.Errorf("material decision source or expected members are invalid")
	}
	artifacts := make(map[string]bool, len(decision.AffectedArtifacts))
	for _, artifact := range decision.AffectedArtifacts {
		if !filepath.IsLocal(artifact.Path) || artifact.Path == "" || strings.TrimSpace(artifact.InvalidatedText) == "" || artifacts[artifact.Path] {
			return fmt.Errorf("material decision artifact set is invalid")
		}
		artifacts[artifact.Path] = true
	}
	criteria := make(map[string]bool, len(decision.AffectedCriteria))
	for _, criterion := range decision.AffectedCriteria {
		key := criterion.EntityKey + "\x00" + criterion.ProductionCallerPath + "\x00" + criterion.AcceptanceCriterionID
		if criterion.EntityKey == "" || !filepath.IsLocal(criterion.ProductionCallerPath) || criterion.ProductionCallerPath == "" || criterion.AcceptanceCriterionID == "" || strings.TrimSpace(criterion.Assertion) == "" || criteria[key] {
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
