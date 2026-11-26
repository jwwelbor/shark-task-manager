package runner

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// bugGuardExecution is the retained record of running a sweep's guard against
// the deliberately restored defect and against the corrected candidate.
type bugGuardExecution struct {
	GuardPointer string `json:"guard_pointer"`
	ClassKey     string `json:"class_key"`
	Site         string `json:"site"`
	Candidate    string `json:"candidate"`
	Enabled      bool   `json:"enabled"`
	Restored     struct {
		ExitCode    *int   `json:"exit_code"`
		FlaggedSite string `json:"flagged_site"`
		Diagnostic  string `json:"diagnostic"`
	} `json:"restored"`
	Corrected struct {
		ExitCode *int `json:"exit_code"`
	} `json:"corrected"`
}

// verifyBugSweepGuards resolves the paired guard evidence behind every
// completed remediation sweep before the result is written, whichever outcome
// carries it. A claimed "verified" status is not evidence.
func verifyBugSweepGuards(req gatepersist.Request) error {
	if req.EntityType != models.EntityTypeBug || req.Result == nil {
		return nil
	}
	for _, sweep := range req.Result.RemediationSweeps {
		if sweep.Status != "complete" {
			continue
		}
		if err := verifyBugSweepGuard(req, sweep); err != nil {
			return fmt.Errorf("sweep %s: unsupported closure: %w", sweep.ClassKey, err)
		}
	}
	return nil
}

func verifyBugSweepGuard(req gatepersist.Request, sweep gateresult.DefectClassSweep) error {
	guard := sweep.Guard
	if !filepath.IsLocal(guard.ImplementationPointer) || !filepath.IsLocal(guard.CounterfactualPointer) {
		return fmt.Errorf("guard evidence pointers must be project-local")
	}
	root := bugEvidenceProject(req)
	if _, err := readBoundedBugEvidence(root, guard.ImplementationPointer, bugEvidenceLimit("")); err != nil {
		return fmt.Errorf("guard implementation is unreadable: %w", err)
	}
	data, err := readBoundedBugEvidence(root, guard.CounterfactualPointer, bugEvidenceLimit(""))
	if err != nil {
		return fmt.Errorf("guard execution record is unreadable: %w", err)
	}
	var run bugGuardExecution
	if err := decodeStrict(data, &run); err != nil {
		return fmt.Errorf("invalid guard execution record: %w", err)
	}
	return checkBugGuardExecution(req, sweep, run)
}

func checkBugGuardExecution(req gatepersist.Request, sweep gateresult.DefectClassSweep, run bugGuardExecution) error {
	if filepath.ToSlash(filepath.Clean(run.GuardPointer)) != filepath.ToSlash(filepath.Clean(sweep.Guard.ImplementationPointer)) || run.ClassKey != sweep.ClassKey {
		return fmt.Errorf("execution record belongs to a different guard or class")
	}
	if !run.Enabled {
		return fmt.Errorf("guard was skipped or disabled")
	}
	site := false
	for _, instance := range sweep.Instances {
		site = site || instance.SitePointer == run.Site
	}
	if !site {
		return fmt.Errorf("execution record exercises a site outside the sweep")
	}
	if run.Restored.ExitCode == nil || *run.Restored.ExitCode == 0 || run.Restored.FlaggedSite != run.Site || !strings.Contains(run.Restored.Diagnostic, run.Site) {
		return fmt.Errorf("guard did not flag the restored defect at its own site")
	}
	if run.Corrected.ExitCode == nil || *run.Corrected.ExitCode != 0 {
		return fmt.Errorf("guard did not pass on the corrected candidate")
	}
	return checkBugGuardCandidate(req, run.Candidate)
}

// checkBugGuardCandidate binds the corrected-code observation to the reviewed
// candidate whenever the result retains its scope.
func checkBugGuardCandidate(req gatepersist.Request, candidate string) error {
	if strings.TrimSpace(candidate) == "" {
		return fmt.Errorf("execution record does not name its corrected candidate")
	}
	if _, data, err := readBugEvidence(req, "bug_review_scope"); err == nil {
		var scope bugCandidateScope
		if err := decodeJSONLoose(data, &scope); err != nil {
			return err
		}
		if scope.CandidateCommit != candidate {
			return fmt.Errorf("execution record is for a different candidate")
		}
	}
	return nil
}

func decodeJSONLoose(data []byte, target any) error {
	return json.Unmarshal(data, target)
}
