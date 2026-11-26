package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
)

var bugReviewVerdict = regexp.MustCompile(`(?i)\bVERDICT\s*:?\s*\*{0,2}\s*:?\s*(PASS(?:-WITH-TRIAGE)?|FAIL)\b`)
var bugContradictoryBlocker = regexp.MustCompile(`\bnot\s+refuted\b|\bblockers?\s+remain\b|\bunresolved\s+blockers?\b`)
var bugIncompleteReport = regexp.MustCompile(`(?i)\bINCOMPLETE\b`)
var bugReviewScopeWord = regexp.MustCompile(`(?i)\b(scope|coverage)\b`)
var bugReviewFindingsWord = regexp.MustCompile(`(?i)\b(findings|defects)\b`)
var bugReviewHeaderField = regexp.MustCompile(`\*\*([^*\n]+):\*\*`)
var bugCLIModel = regexp.MustCompile("\\*\\*Adversarial model:\\*\\*[ \\t]*`([^`\\r\\n]*)`")
var bugCLIFallback = regexp.MustCompile(`\*\*Fallback reason:\*\*[ \t]*([^·\r\n]*)`)

type bugAngleFinding struct {
	Severity  string        `json:"severity"`
	File      reviewGitPath `json:"file"`
	Line      int           `json:"line"`
	Rule      string        `json:"rule"`
	Summary   string        `json:"summary"`
	Diagnosis string        `json:"diagnosis"`
	Impact    string        `json:"impact"`
	Evidence  string        `json:"evidence"`
}
type bugReviewIdentity struct {
	Commit string `json:"commit"`
	Tree   string `json:"tree"`
	Diff   string `json:"diff_sha256"`
	Plan   string `json:"repair_plan_sha256"`
}
type bugAngle struct {
	ReviewContractVersion int                 `json:"review_contract_version"`
	CandidateIdentity     bugReviewIdentity   `json:"candidate_identity"`
	ReviewedFiles         reviewGitPaths      `json:"reviewed_files"`
	Findings              []bugAngleFinding   `json:"findings"`
	CarriedUnits          []bugCarriedUnit    `json:"carried_units,omitempty"`
	CrossFileFresh        bool                `json:"cross_file_fresh,omitempty"`
	SiteInspections       []bugSiteInspection `json:"site_inspections,omitempty"`
}

func (id bugReviewIdentity) matches(scope bugCandidateScope) bool {
	return id.Commit == scope.CandidateCommit && id.Tree == scope.CandidateTree && id.Diff == scope.DiffSHA256 && id.Plan == scope.RepairPlanSHA256
}

// verifyBugReviewEvidence validates retained review artifacts without judging their substantive quality.
func verifyBugReviewEvidence(ctx context.Context, req gatepersist.Request, credit *bugCredit) error {
	_, scopeData, err := readBugEvidence(req, "bug_review_scope")
	if err != nil {
		return err
	}
	var scope bugCandidateScope
	if err := json.Unmarshal(scopeData, &scope); err != nil {
		return err
	}
	_, data, err := readBugEvidence(req, "bug_review_report")
	if err != nil {
		return err
	}
	header, body, ok := strings.Cut(string(data), "\n---\n")
	if !ok {
		return fmt.Errorf("review report header is missing")
	}
	if err := validateBugReviewHeader(header, scope); err != nil {
		return err
	}
	if err := verifyReviewArtifactHash(header, "Retained scope", scopeData); err != nil {
		return err
	}
	if err := validateBugReviewBody(header, body); err != nil {
		return err
	}
	return validateBugReviewArtifacts(ctx, req, scope, header, body, credit)
}

func validateBugReviewArtifacts(ctx context.Context, req gatepersist.Request, scope bugCandidateScope, header, body string, credit *bugCredit) error {
	_, decisionData, err := readBugEvidence(req, "bug_review_decisions")
	if err != nil {
		return err
	}
	if err := verifyReviewArtifactHash(header, "Review decisions", decisionData); err != nil {
		return err
	}
	verdict := bugReviewVerdict.FindStringSubmatch(header)[1]
	six, err := bugReviewRunnerIsSix(header)
	if err != nil {
		return err
	}
	decisions, err := validateBugReviewDecisions(decisionData, scope, verdict)
	if err != nil {
		return err
	}
	if decisions.version == 0 {
		if err := requireHistoricalReview(req); err != nil {
			return err
		}
	}
	if err := requireBugFindingDecisions(body, decisions.byFinding); err != nil {
		return err
	}
	var raw map[bugFindingKey]string
	if !six {
		if err := decisions.validateCLI(scope); err != nil {
			return err
		}
	} else {
		if err := verifyBugAngleArtifact(ctx, req, scope, header, decisions, decisionData, verdict, credit); err != nil {
			return err
		}
		if raw, err = rawBugFindingSeverities(req); err != nil {
			return err
		}
	}
	if err := validateBugReviewMaterialImpacts(ctx, req, scope, header, body); err != nil {
		return err
	}
	return validateBugParentTriageObservationRaw(req, scope, decisions.byFinding, raw)
}

func verifyBugAngleArtifact(ctx context.Context, req gatepersist.Request, scope bugCandidateScope, header string, decisions *bugReviewDecisionIndex, decisionData []byte, verdict string, credit *bugCredit) error {
	_, data, err := readBugEvidence(req, "bug_review_angles")
	if err != nil {
		return err
	}
	if err := verifyReviewArtifactHash(header, "Angle results", data); err != nil {
		return err
	}
	if err := validateBugReviewAnglesWith(data, scope, decisions, verdict, credit.verifier(ctx, req, scope)); err != nil {
		return err
	}
	var angles map[string]bugAngle
	if err := json.Unmarshal(data, &angles); err != nil {
		return err
	}
	return verifyBugDiscoveries(decisionData, angles, scope)
}

func requireBugFindingDecisions(body string, decisions map[bugFindingKey]bugReviewDecision) error {
	seen := make(map[string]bool)
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		marker := bugFindingDetail.FindStringSubmatch(line)
		if marker == nil {
			continue
		}
		if seen[marker[2]] {
			return fmt.Errorf("report finding IDs must be unique")
		}
		seen[marker[2]] = true
		matches := 0
		for _, decision := range decisions {
			if decision.FindingID == marker[2] {
				matches++
				if decision.Severity != strings.ToLower(marker[3]) || decision.Disposition != strings.ToLower(marker[4]) || decision.Disposition == "refuted" {
					return fmt.Errorf("report finding decision does not match")
				}
			}
		}
		if matches != 1 {
			return fmt.Errorf("report finding requires its matching retained decision")
		}
	}
	return nil
}

func verifyReviewArtifactHash(header, label string, data []byte) error {
	digest := sha256.Sum256(data)
	if !strings.Contains(header, fmt.Sprintf("**%s SHA-256:** `%x`", label, digest)) {
		return fmt.Errorf("%s digest does not match report", label)
	}
	return nil
}
func validateBugReviewHeader(header string, scope bugCandidateScope) error {
	if err := validateUniqueBugReviewHeader(header); err != nil {
		return err
	}
	for label, value := range map[string]string{"Base commit": scope.BaseCommit, "Candidate": scope.CandidateCommit, "Tree": scope.CandidateTree, "Diff SHA-256": scope.DiffSHA256, "Repair plan SHA-256": scope.RepairPlanSHA256} {
		if value == "" || !strings.Contains(header, "**"+label+":** `"+value+"`") {
			return fmt.Errorf("review report %s mismatch", label)
		}
	}
	return nil
}

func validateUniqueBugReviewHeader(header string) error {
	seen := map[string]bool{}
	for _, match := range bugReviewHeaderField.FindAllStringSubmatch(header, -1) {
		name := strings.ToLower(match[1])
		if seen[name] {
			return fmt.Errorf("review report has duplicate %s field", name)
		}
		seen[name] = true
	}
	for _, key := range []string{"runner_mode", "specialists_completed", "consolidator_completed"} {
		if strings.Count(strings.ToLower(header), "`"+key+"=") != 1 {
			return fmt.Errorf("review report has missing or duplicate %s metadata", key)
		}
	}
	if strings.Contains(header, "**Review contract:** `2`") {
		for _, key := range []string{"Requested model", "Verified effective model"} {
			if !seen[strings.ToLower(key)] {
				return fmt.Errorf("impact-based review report is missing %s metadata", key)
			}
		}
	}
	if len(bugReviewVerdict.FindAllStringSubmatch(header, -1)) != 1 {
		return fmt.Errorf("review report must have one header verdict")
	}
	return nil
}

var bugCurrentComplete = regexp.MustCompile(`(?i)^CURRENT_REVIEW_STATUS:[ \t]*COMPLETE[ \t]*$`)
var bugCurrentField = regexp.MustCompile(`(?i)\bCURRENT_REVIEW_STATUS\b`)
var bugHistoricalStatus = regexp.MustCompile(`(?i)^HISTORICAL_REVIEW_STATUS:[ \t]*DID_NOT_FINISH[ \t]*$`)
var bugFindingDetail = regexp.MustCompile(`(?i)^FINDING_DETAIL:[ \t]*INCOMPLETE (VALIDATION|PRODUCT BEHAVIOR)[ \t]*\|[ \t]*(F-[1-9][0-9]*)[ \t]*\|[ \t]*(blocker|non-blocker|nit)[ \t]*\|[ \t]*([a-z0-9_-]+)[ \t]*$`)

var bugExplicitUnfinished = regexp.MustCompile(`(?i)\b(current|this|the[ \t]+current)[ \t]+(review|report|assessment|inspection)[ \t]+((is|was|did|has|remains)[ \t]+)*((still[ \t]+)?unfinished|not[ \t]+(yet[ \t]+)?(finish|finished|complete|completed)|failed[ \t]+to[ \t]+finish|only[ \t]+partially[ \t]+complete|did_not_finish)\b`)

func statusSafeBody(body string) bool {
	if bugExplicitUnfinished.MatchString(body) {
		return false
	}
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	current, history := 0, 0
	footer := false
	for _, line := range lines {
		if bugCurrentField.MatchString(line) {
			current++
			if !bugCurrentComplete.MatchString(line) {
				return false
			}
		}
		historical, finding := bugHistoricalStatus.MatchString(line), bugFindingDetail.MatchString(line)
		if historical {
			history++
		}
		if historical || finding {
			footer = true
		} else if footer && strings.TrimSpace(line) != "" {
			return false
		}
		if bugIncompleteReport.MatchString(line) && !finding {
			return false
		}
	}
	return current <= 1 && history <= 1 && (!footer || current == 1)
}
func validateBugReviewBody(header, body string) error {
	if !utf8.ValidString(body) || !statusSafeBody(body) || utf8.RuneCountInString(strings.TrimSpace(body)) < 80 {
		return fmt.Errorf("review report is incomplete")
	}
	verdict := bugReviewVerdict.FindStringSubmatch(header)
	verdicts := bugReviewVerdict.FindAllStringSubmatch(body, -1)
	if len(verdict) == 0 || len(verdicts) == 0 {
		return fmt.Errorf("review verdict is missing")
	}
	if strings.EqualFold(verdict[1], "PASS") {
		for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
			if bugFindingDetail.MatchString(line) {
				return fmt.Errorf("clean review pass contains a finding")
			}
		}
	}
	for _, v := range verdicts {
		if !strings.EqualFold(v[1], verdict[1]) || strings.EqualFold(v[1], "FAIL") {
			return fmt.Errorf("review verdict is not a consistent pass")
		}
	}
	return validateBugReviewBodyCoverage(body)
}
func validateBugReviewBodyCoverage(body string) error {
	lower := strings.ToLower(body)
	checked := strings.ReplaceAll(strings.ReplaceAll(lower, "no blockers remain", ""), "no unresolved blockers", "")
	if bugContradictoryBlocker.MatchString(checked) {
		return fmt.Errorf("review report describes an unresolved blocker")
	}
	if !bugReviewScopeWord.MatchString(body) || !bugReviewFindingsWord.MatchString(body) {
		return fmt.Errorf("review report lacks scope or findings")
	}
	return nil
}
func bugReviewRunnerIsSix(header string) (bool, error) {
	if !strings.Contains(header, "`consolidator_completed=True`") {
		return false, fmt.Errorf("review consolidation is incomplete")
	}
	six := strings.Contains(header, "`runner_mode=canonical-six-angle`") || strings.Contains(header, "`runner_mode=dispatched-six-angle`")
	if six {
		if !strings.Contains(header, "`specialists_completed=6`") {
			return false, fmt.Errorf("six-angle review is incomplete")
		}
		if strings.Contains(header, "`runner_mode=dispatched-six-angle`") && !validBugFallback(header) {
			return false, fmt.Errorf("review runner is incomplete")
		}
		return true, nil
	}
	if !validBugCLIHeader(header) {
		return false, fmt.Errorf("review runner is incomplete")
	}
	return false, nil
}
func validBugCLIHeader(header string) bool {
	if !strings.Contains(header, "`runner_mode=adversarial-cli`") || !strings.Contains(header, "`specialists_completed=0`") {
		return false
	}
	model := bugCLIModel.FindStringSubmatch(header)
	if len(model) != 2 {
		return false
	}
	return meaningfulBugReviewMetadata(model[1]) && validBugFallback(header)
}

func validBugFallback(header string) bool {
	reason := bugCLIFallback.FindStringSubmatch(header)
	return len(reason) == 2 && meaningfulBugReviewMetadata(reason[1])
}

func meaningfulBugReviewMetadata(value string) bool {
	value = strings.ToLower(strings.TrimSpace(strings.Trim(strings.TrimSpace(value), "`")))
	return value != "" && value != "none" && value != "unknown"
}

func validateBugReviewAngles(data []byte, scope bugCandidateScope, decisions *bugReviewDecisionIndex, verdict string) error {
	return validateBugReviewAnglesWith(data, scope, decisions, verdict, nil)
}

func validateBugReviewAnglesWith(data []byte, scope bugCandidateScope, decisions *bugReviewDecisionIndex, verdict string, verify bugUnitVerifier) error {
	var angles map[string]bugAngle
	if err := json.Unmarshal(data, &angles); err != nil || len(angles) != 6 {
		return fmt.Errorf("six angle results are required")
	}
	version := angles["a"].ReviewContractVersion
	if version != 0 && version != 2 {
		return fmt.Errorf("unsupported angle review contract version")
	}
	if version != decisions.version {
		return fmt.Errorf("angle and decision review contract versions differ")
	}
	matches := newBugAngleMatcher(angles, decisions)
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		angle, found := angles[name]
		if !found {
			return fmt.Errorf("angle %s missing", name)
		}
		if angle.ReviewContractVersion != version {
			return fmt.Errorf("angle review contract versions differ")
		}
		if err := validateBugReviewAngle(name, angle, scope, matches, verdict, verify); err != nil {
			return fmt.Errorf("angle %s: %w", name, err)
		}
	}
	return matches.checkRemaining()
}

func (k bugFindingKey) site() bugFindingKey {
	k.Origin, k.Evidence = "", ""
	return k
}

func (f bugAngleFinding) site() bugFindingKey {
	return bugFindingKey{File: f.File, Line: f.Line, Rule: f.Rule, Summary: f.Summary, Diagnosis: f.Diagnosis}
}

type bugAngleMatcher struct {
	decisions  *bugReviewDecisionIndex
	siteCounts map[bugFindingKey]int
	matched    map[bugFindingKey]bool
}

func newBugAngleMatcher(angles map[string]bugAngle, decisions *bugReviewDecisionIndex) *bugAngleMatcher {
	counts := make(map[bugFindingKey]int)
	for _, angle := range angles {
		for _, finding := range angle.Findings {
			counts[finding.site()]++
		}
	}
	return &bugAngleMatcher{decisions: decisions, siteCounts: counts, matched: make(map[bugFindingKey]bool)}
}

func (m *bugAngleMatcher) match(origin string, f bugAngleFinding) (bugReviewDecision, bool) {
	site := f.site()
	key := site
	if m.decisions.version == 0 {
		decision, ok := m.decisions.byFinding[key]
		return decision, ok
	}
	key.Origin, key.Evidence = origin, f.Evidence
	decision, ok := m.decisions.byFinding[key]
	if !ok && m.siteCounts[site] == 1 {
		key = site
		decision, ok = m.decisions.byFinding[key]
	}
	if !ok || m.matched[key] || decision.Evidence != f.Evidence {
		return bugReviewDecision{}, false
	}
	m.matched[key] = true
	return decision, true
}

func (m *bugAngleMatcher) checkRemaining() error {
	if m.decisions.version == 0 {
		return nil
	}
	for key, decision := range m.decisions.byFinding {
		if key.Origin == "" && !m.matched[key] && m.siteCounts[key] > 0 {
			return fmt.Errorf("ambiguous extra decision %s at %s:%d", decision.FindingID, quoteReviewGitPath(string(key.File)), key.Line)
		}
		if key.Origin == "single-reviewer" || (isReviewAngle(key.Origin) && !m.matched[key]) {
			return fmt.Errorf("angle %s has no raw observation for %s at %s:%d", key.Origin, decision.FindingID, quoteReviewGitPath(string(key.File)), key.Line)
		}
	}
	return nil
}

func validateBugReviewAngle(name string, angle bugAngle, scope bugCandidateScope, matches *bugAngleMatcher, verdict string, verify bugUnitVerifier) error {
	if !angle.CandidateIdentity.matches(scope) {
		return fmt.Errorf("candidate mismatch")
	}
	if err := checkBugAngleCoverage(name, angle, scope, verify); err != nil {
		return err
	}
	if angle.Findings == nil {
		return fmt.Errorf("findings missing")
	}
	for _, finding := range angle.Findings {
		if err := validateFindingSupport(angle.ReviewContractVersion, finding.Severity, finding.Impact, finding.Evidence); err != nil {
			return err
		}
		if err := validateBugAngleFinding(name, finding, matches, verdict); err != nil {
			return err
		}
	}
	return nil
}
func verifyDeclaredReviewPaths(inspected, required []string) error {
	if inspected == nil {
		return fmt.Errorf("reviewed_files missing")
	}
	seen := make(map[string]bool, len(inspected))
	for _, path := range inspected {
		if !validReviewGitPath(path) {
			return fmt.Errorf("invalid reviewed_files path %s", quoteReviewGitPath(path))
		}
		seen[path] = true
	}
	for _, path := range required {
		if !seen[path] {
			return fmt.Errorf("coverage incomplete: missing %s", quoteReviewGitPath(path))
		}
	}
	return nil
}

func validateBugAngleFinding(name string, f bugAngleFinding, matches *bugAngleMatcher, verdict string) error {
	if !validBugAngleFinding(f) {
		return fmt.Errorf("finding malformed")
	}
	decision, found := matches.match(name, f)
	if !found {
		if f.Severity == "blocker" {
			return fmt.Errorf("unresolved blocker")
		}
		return fmt.Errorf("finding has no explicit disposition at %s:%d", quoteReviewGitPath(string(f.File)), f.Line)
	}
	if f.Severity == "blocker" && !resolvedBugRawBlocker(decision, verdict) {
		return fmt.Errorf("unresolved blocker")
	}
	return nil
}

func resolvedBugRawBlocker(decision bugReviewDecision, verdict string) bool {
	if decision.Severity == "blocker" {
		return decision.Disposition == "refuted"
	}
	return strings.EqualFold(verdict, "PASS-WITH-TRIAGE") && decision.Disposition != "refuted"
}
func validBugAngleFinding(f bugAngleFinding) bool {
	if f.Severity != "blocker" && f.Severity != "non-blocker" && f.Severity != "nit" {
		return false
	}
	return f.File != "" && f.Line > 0 && f.Rule != "" && f.Summary != "" && f.Diagnosis != ""
}

// requireHistoricalReview admits an unversioned packet only for an operation
// that the parent already applied under the pre-v2 provenance contract. A
// pending journal, a missing journal or a run admitted under the current
// contract grants no such eligibility.
func requireHistoricalReview(req gatepersist.Request) error {
	state, exists, err := gaterun.LoadOperationState(req.RunDir)
	if err != nil || !exists || state.PersistenceState != gaterun.PersistenceStateTransitioned || state.AdmissionVersion != gatepersist.BoundAdmissionVersion {
		return fmt.Errorf("unversioned review packet requires review_contract_version 2: only a review applied before that contract is eligible")
	}
	return nil
}
