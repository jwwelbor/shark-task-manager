package runner

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
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
	Severity  string `json:"severity"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Rule      string `json:"rule"`
	Summary   string `json:"summary"`
	Diagnosis string `json:"diagnosis"`
}
type bugReviewIdentity struct {
	Commit string `json:"commit"`
	Tree   string `json:"tree"`
	Diff   string `json:"diff_sha256"`
	Plan   string `json:"repair_plan_sha256"`
}
type bugAngle struct {
	CandidateIdentity bugReviewIdentity `json:"candidate_identity"`
	ReviewedFiles     []string          `json:"reviewed_files"`
	Findings          []bugAngleFinding `json:"findings"`
}

func (id bugReviewIdentity) matches(scope bugCandidateScope) bool {
	return id.Commit == scope.CandidateCommit && id.Tree == scope.CandidateTree && id.Diff == scope.DiffSHA256 && id.Plan == scope.RepairPlanSHA256
}

// verifyBugReviewEvidence validates retained review artifacts without judging their substantive quality.
func verifyBugReviewEvidence(req gatepersist.Request) error {
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
	return validateBugReviewArtifacts(req, scope, header, body)
}

func validateBugReviewArtifacts(req gatepersist.Request, scope bugCandidateScope, header, body string) error {
	_, decisionData, err := readBugEvidence(req, "bug_review_decisions")
	if err != nil {
		return err
	}
	if err := verifyReviewArtifactHash(header, "Review decisions", decisionData); err != nil {
		return err
	}
	verdict := bugReviewVerdict.FindStringSubmatch(header)[1]
	decisions, err := validateBugReviewDecisions(decisionData, scope, verdict)
	if err != nil {
		return err
	}
	if err := requireBugFindingDecisions(body, decisions); err != nil {
		return err
	}
	six, err := bugReviewRunnerIsSix(header)
	if err != nil {
		return err
	}
	if !six {
		return nil
	}
	_, angleData, err := readBugEvidence(req, "bug_review_angles")
	if err != nil {
		return err
	}
	if err := verifyReviewArtifactHash(header, "Angle results", angleData); err != nil {
		return err
	}
	return validateBugReviewAngles(angleData, scope, decisions, verdict)
}

func requireBugFindingDecisions(body string, decisions map[bugFindingKey]bugReviewDecision) error {
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		marker := bugFindingDetail.FindStringSubmatch(line)
		if marker == nil {
			continue
		}
		matches := 0
		for _, decision := range decisions {
			if decision.FindingID == marker[2] {
				matches++
				if !strings.EqualFold(decision.Severity, marker[3]) || !strings.EqualFold(decision.Disposition, marker[4]) || strings.EqualFold(decision.Disposition, "refuted") {
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
	if len(bugReviewVerdict.FindAllStringSubmatch(header, -1)) != 1 {
		return fmt.Errorf("review report must have one header verdict")
	}
	return nil
}

var bugCurrentComplete = regexp.MustCompile(`(?i)^CURRENT_REVIEW_STATUS:[ \t]*COMPLETE[ \t]*$`)
var bugCurrentField = regexp.MustCompile(`(?i)\bCURRENT_REVIEW_STATUS\b`)
var bugHistoricalStatus = regexp.MustCompile(`(?i)^HISTORICAL_REVIEW_STATUS:[ \t]*DID_NOT_FINISH[ \t]*$`)
var bugFindingDetail = regexp.MustCompile(`(?i)^FINDING_DETAIL:[ \t]*INCOMPLETE (VALIDATION|PRODUCT BEHAVIOR)[ \t]*\|[ \t]*(F-[1-9][0-9]*)[ \t]*\|[ \t]*(blocker|non-blocker|nit)[ \t]*\|[ \t]*([a-z0-9_-]+)[ \t]*$`)

var bugExplicitUnfinished = regexp.MustCompile(`(?i)\b(current|this|the)[ \t]+(current[ \t]+)?(review|report|assessment|inspection)[ \t]+((is|was|did|has|remains)[ \t]+)*(not[ \t]+(finish|finished|complete|completed)|unfinished|did_not_finish)\b`)

func statusSafeBody(body string) bool {
	if bugExplicitUnfinished.MatchString(body) {
		return false
	}
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	current, history, findings := 0, 0, 0
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
		if finding {
			findings++
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
	return current <= 1 && history <= 1 && findings <= 1 && (!footer || current == 1)
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

func validateBugReviewAngles(data []byte, scope bugCandidateScope, decisions map[bugFindingKey]bugReviewDecision, verdict string) error {
	var angles map[string]bugAngle
	if err := json.Unmarshal(data, &angles); err != nil || len(angles) != 6 {
		return fmt.Errorf("six angle results are required")
	}
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		angle, found := angles[name]
		if !found {
			return fmt.Errorf("angle %s missing", name)
		}
		if err := validateBugReviewAngle(angle, scope, decisions, verdict); err != nil {
			return fmt.Errorf("angle %s: %w", name, err)
		}
	}
	return nil
}
func validateBugReviewAngle(angle bugAngle, scope bugCandidateScope, decisions map[bugFindingKey]bugReviewDecision, verdict string) error {
	if !angle.CandidateIdentity.matches(scope) {
		return fmt.Errorf("candidate mismatch")
	}
	got := map[string]bool{}
	for _, file := range angle.ReviewedFiles {
		got[file] = true
	}
	for _, file := range scope.ChangedFiles {
		if !got[file] {
			return fmt.Errorf("coverage incomplete")
		}
	}
	if angle.Findings == nil {
		return fmt.Errorf("findings missing")
	}
	for _, finding := range angle.Findings {
		if err := validateBugAngleFinding(finding, decisions, verdict); err != nil {
			return err
		}
	}
	return nil
}
func validateBugAngleFinding(f bugAngleFinding, decisions map[bugFindingKey]bugReviewDecision, verdict string) error {
	if !validBugAngleFinding(f) {
		return fmt.Errorf("finding malformed")
	}
	decision, found := decisions[bugFindingKey{f.File, f.Line, f.Rule, f.Summary, f.Diagnosis}]
	if !found {
		if f.Severity == "blocker" {
			return fmt.Errorf("unresolved blocker")
		}
		return fmt.Errorf("finding has no explicit disposition")
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
