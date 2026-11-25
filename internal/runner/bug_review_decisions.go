package runner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var bugNegatedRefutation = regexp.MustCompile(`(?i)\bnot\s+refuted\b`)

type bugFindingKey struct {
	File      string
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
	Impact      string        `json:"impact"`
	Evidence    string        `json:"evidence"`
	Disposition string        `json:"disposition"`
	Reason      string        `json:"reason"`
}
type bugReviewDecisions struct {
	ReviewContractVersion int                 `json:"review_contract_version"`
	CandidateIdentity     bugReviewIdentity   `json:"candidate_identity"`
	ReviewedFiles         reviewGitPaths      `json:"reviewed_files"`
	Findings              []bugReviewDecision `json:"findings"`
}

type bugReviewDecisionIndex struct {
	version       int
	byFinding     map[bugFindingKey]bugReviewDecision
	reviewedFiles reviewGitPaths
}

func validateBugReviewDecisions(data []byte, scope bugCandidateScope, verdict string) (*bugReviewDecisionIndex, error) {
	var packet bugReviewDecisions
	if err := json.Unmarshal(data, &packet); err != nil {
		return nil, fmt.Errorf("invalid review decisions: %w", err)
	}
	id := packet.CandidateIdentity
	if !id.matches(scope) {
		return nil, fmt.Errorf("review decisions candidate or plan mismatch")
	}
	if packet.Findings == nil {
		return nil, fmt.Errorf("review decisions findings missing")
	}
	if packet.ReviewContractVersion != 0 && packet.ReviewContractVersion != 2 {
		return nil, fmt.Errorf("unsupported review contract version")
	}
	found, err := indexBugReviewDecisions(packet, verdict)
	if err != nil {
		return nil, err
	}
	if packet.ReviewContractVersion == 2 && strings.EqualFold(verdict, "PASS-WITH-TRIAGE") && !hasBugTriageBasis(found) {
		return nil, fmt.Errorf("PASS-with-triage requires at least one retained finding with an actionable or report-only disposition")
	}
	return &bugReviewDecisionIndex{version: packet.ReviewContractVersion, byFinding: found, reviewedFiles: packet.ReviewedFiles}, nil
}

func hasBugTriageBasis(found map[bugFindingKey]bugReviewDecision) bool {
	for _, finding := range found {
		if finding.Disposition != "refuted" && finding.Disposition != "excluded" {
			return true
		}
	}
	return false
}

func indexBugReviewDecisions(packet bugReviewDecisions, verdict string) (map[bugFindingKey]bugReviewDecision, error) {
	found := make(map[bugFindingKey]bugReviewDecision, len(packet.Findings))
	for _, finding := range packet.Findings {
		if err := validateBugReviewDecision(finding, verdict); err != nil {
			return nil, err
		}
		if err := validateFindingSupport(packet.ReviewContractVersion, finding.Severity, finding.Impact, finding.Evidence); err != nil {
			return nil, err
		}
		key := bugFindingKey{string(finding.File), finding.Line, finding.Rule, finding.Summary, finding.Diagnosis}
		if _, exists := found[key]; exists {
			return nil, fmt.Errorf("duplicate review decision")
		}
		found[key] = finding
	}
	return found, nil
}

// Previously applied legacy packets predate the impact and evidence fields.
func validateFindingSupport(version int, severity, impact, evidence string) error {
	if version != 2 {
		return nil
	}
	if severity == "nit" {
		return fmt.Errorf("current finding uses legacy nit severity")
	}
	if strings.TrimSpace(impact) == "" {
		return fmt.Errorf("current finding is missing impact")
	}
	if strings.TrimSpace(evidence) == "" {
		return fmt.Errorf("current finding is missing supporting evidence")
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
	case "non-blocker", "nit":
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
