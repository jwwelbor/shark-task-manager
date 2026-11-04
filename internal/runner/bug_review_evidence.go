package runner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
)

var bugReviewVerdict = regexp.MustCompile(`(?i)\bVERDICT\s*:?\s*\*{0,2}\s*:?\s*(PASS(?:-WITH-TRIAGE)?|FAIL)\b`)

type bugAngleFinding struct {
	Severity  string `json:"severity"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Rule      string `json:"rule"`
	Summary   string `json:"summary"`
	Diagnosis string `json:"diagnosis"`
}
type bugAngle struct {
	CandidateIdentity struct {
		Commit string `json:"commit"`
		Tree   string `json:"tree"`
		Diff   string `json:"diff_sha256"`
	} `json:"candidate_identity"`
	ReviewedFiles []string          `json:"reviewed_files"`
	Findings      []bugAngleFinding `json:"findings"`
}

// verifyBugReviewEvidence validates the report and actual per-angle output
// retained by deep-review; it does not judge the substance of the review.
func verifyBugReviewEvidence(req gatepersist.Request) error {
	_, scopeData, err := readBugEvidence(req, "bug_review_scope")
	if err != nil {
		return err
	}
	var scope bugCandidateScope
	if err = json.Unmarshal(scopeData, &scope); err != nil {
		return err
	}
	_, data, err := readBugEvidence(req, "bug_review_report")
	if err != nil {
		return err
	}
	report := string(data)
	header, body, ok := strings.Cut(report, "\n---\n")
	if !ok {
		return fmt.Errorf("review report header is missing")
	}
	if !strings.Contains(header, "**Candidate:** `"+scope.CandidateCommit+"`") || !strings.Contains(header, "**Tree:** `"+scope.CandidateTree+"`") || !strings.Contains(header, "**Diff SHA-256:** `"+scope.DiffSHA256+"`") {
		return fmt.Errorf("review report candidate mismatch")
	}
	if strings.Contains(strings.ToUpper(body), "INCOMPLETE") || len(strings.TrimSpace(body)) < 80 {
		return fmt.Errorf("review report is incomplete")
	}
	headerVerdict := bugReviewVerdict.FindStringSubmatch(header)
	verdicts := bugReviewVerdict.FindAllStringSubmatch(body, -1)
	if len(headerVerdict) == 0 || len(verdicts) == 0 {
		return fmt.Errorf("review verdict is missing")
	}
	for _, v := range verdicts {
		if !strings.EqualFold(v[1], headerVerdict[1]) || strings.EqualFold(v[1], "FAIL") {
			return fmt.Errorf("review verdict is not a consistent pass")
		}
	}
	lower := strings.ToLower(body)
	if !(strings.Contains(lower, "scope") || strings.Contains(lower, "coverage")) || !(strings.Contains(lower, "findings") || strings.Contains(lower, "defects")) {
		return fmt.Errorf("review report lacks scope or findings")
	}
	six := strings.Contains(header, "`runner_mode=canonical-six-angle`") || strings.Contains(header, "`runner_mode=dispatched-six-angle`")
	if !strings.Contains(header, "`consolidator_completed=True`") {
		return fmt.Errorf("review consolidation is incomplete")
	}
	if !six {
		if !strings.Contains(header, "`runner_mode=adversarial-cli`") || !strings.Contains(header, "`specialists_completed=0`") || strings.Contains(header, "**Adversarial model:** `none`") || !strings.Contains(header, "**Adversarial model:** `") {
			return fmt.Errorf("review runner is incomplete")
		}
		return nil
	}
	if !strings.Contains(header, "`specialists_completed=6`") {
		return fmt.Errorf("six-angle review is incomplete")
	}
	_, angleData, err := readBugEvidence(req, "bug_review_angles")
	if err != nil {
		return err
	}
	return validateBugReviewAngles(angleData, scope, body)
}

func validateBugReviewAngles(angleData []byte, scope bugCandidateScope, body string) error {
	var angles map[string]bugAngle
	if err := json.Unmarshal(angleData, &angles); err != nil || len(angles) != 6 {
		return fmt.Errorf("six angle results are required")
	}
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		angle, found := angles[name]
		if !found || angle.CandidateIdentity.Commit != scope.CandidateCommit || angle.CandidateIdentity.Tree != scope.CandidateTree || angle.CandidateIdentity.Diff != scope.DiffSHA256 {
			return fmt.Errorf("angle %s candidate mismatch", name)
		}
		got := map[string]bool{}
		for _, file := range angle.ReviewedFiles {
			got[file] = true
		}
		if len(got) != len(scope.ChangedFiles) {
			return fmt.Errorf("angle %s coverage incomplete", name)
		}
		for _, file := range scope.ChangedFiles {
			if !got[file] {
				return fmt.Errorf("angle %s coverage incomplete", name)
			}
		}
		if angle.Findings == nil {
			return fmt.Errorf("angle %s findings missing", name)
		}
		for _, f := range angle.Findings {
			if (f.Severity != "blocker" && f.Severity != "non-blocker" && f.Severity != "nit") || f.File == "" || f.Line < 1 || f.Rule == "" || f.Summary == "" || f.Diagnosis == "" {
				return fmt.Errorf("angle %s finding malformed", name)
			}
			if f.Severity == "blocker" {
				pos := strings.Index(body, fmt.Sprintf("%s:%d", f.File, f.Line))
				if pos < 0 {
					return fmt.Errorf("unresolved blocker in angle %s", name)
				}
				lo := pos - 150
				if lo < 0 {
					lo = 0
				}
				hi := pos + 1000
				if hi > len(body) {
					hi = len(body)
				}
				excerpt := body[lo:hi]
				if !strings.Contains(excerpt, f.Summary) || !strings.Contains(strings.ToUpper(excerpt), "REFUTED") {
					return fmt.Errorf("unresolved blocker in angle %s", name)
				}
			}
		}
	}
	return nil
}
