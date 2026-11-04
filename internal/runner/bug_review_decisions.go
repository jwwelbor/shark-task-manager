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
	File        string `json:"file"`
	Line        int    `json:"line"`
	Severity    string `json:"severity"`
	Rule        string `json:"rule"`
	Summary     string `json:"summary"`
	Diagnosis   string `json:"diagnosis"`
	Disposition string `json:"disposition"`
	Reason      string `json:"reason"`
}
type bugReviewDecisions struct {
	CandidateIdentity bugReviewIdentity   `json:"candidate_identity"`
	Findings          []bugReviewDecision `json:"findings"`
}

func validateBugReviewDecisions(data []byte, scope bugCandidateScope, verdict string) (map[bugFindingKey]bugReviewDecision, error) {
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
	found := make(map[bugFindingKey]bugReviewDecision, len(packet.Findings))
	for _, finding := range packet.Findings {
		if err := validateBugReviewDecision(finding, verdict); err != nil {
			return nil, err
		}
		key := bugFindingKey{finding.File, finding.Line, finding.Rule, finding.Summary, finding.Diagnosis}
		if _, exists := found[key]; exists {
			return nil, fmt.Errorf("duplicate review decision")
		}
		found[key] = finding
	}
	return found, nil
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
	return f.File != "" && f.Line > 0 && f.Rule != "" && f.Summary != "" && f.Diagnosis != "" && f.Disposition != "" && f.Reason != ""
}
