package runner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var bugNegatedRefutation = regexp.MustCompile(`(?i)\bnot\s+refuted\b`)

type bugFindingKey struct {
	File      reviewGitPath
	Line      int
	Rule      string
	Summary   string
	Diagnosis string
}
type bugReviewDecision struct {
	FindingID   string        `json:"finding_id"`
	File        reviewGitPath `json:"file"`
	Line        int           `json:"line"`
	Severity    string        `json:"severity"`
	Rule        string        `json:"rule"`
	Summary     string        `json:"summary"`
	Diagnosis   string        `json:"diagnosis"`
	Disposition string        `json:"disposition"`
	Reason      string        `json:"reason"`
}
type bugReviewDecisions struct {
	CandidateIdentity bugReviewIdentity   `json:"candidate_identity"`
	ReviewedFiles     reviewGitPaths      `json:"reviewed_files"`
	Findings          []bugReviewDecision `json:"findings"`
}

func validateBugReviewDecisions(data []byte, scope bugCandidateScope, verdict string, requireReviewedFiles bool) (map[bugFindingKey]bugReviewDecision, error) {
	var packet bugReviewDecisions
	if err := json.Unmarshal(data, &packet); err != nil {
		return nil, fmt.Errorf("invalid review decisions: %w", err)
	}
	id := packet.CandidateIdentity
	if !id.matches(scope) {
		return nil, fmt.Errorf("review decisions candidate or plan mismatch")
	}
	if requireReviewedFiles {
		if err := validateBugReviewedFiles(packet.ReviewedFiles, scope.ChangedFiles); err != nil {
			return nil, err
		}
	}
	return indexBugReviewFindings(packet.Findings, verdict)
}

func indexBugReviewFindings(findings []bugReviewDecision, verdict string) (map[bugFindingKey]bugReviewDecision, error) {
	if findings == nil {
		return nil, fmt.Errorf("review decisions findings missing")
	}
	found := make(map[bugFindingKey]bugReviewDecision, len(findings))
	retainedFinding := false
	for _, finding := range findings {
		if err := validateBugReviewDecision(finding, verdict); err != nil {
			return nil, err
		}
		retainedFinding = retainedFinding || finding.Disposition != "refuted"
		key := bugFindingKey{finding.File, finding.Line, finding.Rule, finding.Summary, finding.Diagnosis}
		if _, exists := found[key]; exists {
			return nil, fmt.Errorf("duplicate review decision")
		}
		found[key] = finding
	}
	if strings.EqualFold(verdict, "PASS-WITH-TRIAGE") && !retainedFinding {
		return nil, fmt.Errorf("PASS-with-triage requires at least one retained finding")
	}
	return found, nil
}

func validateBugReviewedFiles(reviewedFiles, changedFiles []string) error {
	if reviewedFiles == nil {
		return fmt.Errorf("review decisions reviewed_files missing")
	}
	inspected := make(map[string]bool, len(reviewedFiles))
	for _, file := range reviewedFiles {
		inspected[file] = true
	}
	for _, file := range changedFiles {
		if !inspected[file] {
			return fmt.Errorf("review decisions coverage incomplete: missing %s", quoteReviewGitPath(file))
		}
	}
	return nil
}

func validateBugReviewDecision(f bugReviewDecision, verdict string) error {
	if !validBugDecisionShape(f) {
		return fmt.Errorf("malformed review decision")
	}
	switch f.Severity {
	case "blocker":
		if f.Disposition != "refuted" {
			return fmt.Errorf("review contains unresolved blocker")
		}
		if bugNegatedRefutation.MatchString(f.Reason) {
			return fmt.Errorf("refutation reason contradicts its disposition")
		}
	case "non-blocker":
	case "nit":
		if f.Disposition != "report-only" {
			return fmt.Errorf("nit disposition must be report-only")
		}
	default:
		return fmt.Errorf("invalid review decision severity")
	}
	if strings.EqualFold(verdict, "PASS") && f.Disposition != "refuted" {
		return fmt.Errorf("clean pass contains an undispositioned finding")
	}
	return nil
}

func validBugDecisionShape(f bugReviewDecision) bool {
	return f.File != "" && f.Line > 0 && f.Rule != "" && f.Summary != "" && f.Diagnosis != "" && strings.TrimSpace(f.Disposition) != "" && strings.TrimSpace(f.Reason) != ""
}
