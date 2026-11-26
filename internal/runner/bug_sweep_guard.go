package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// BugWorkItemResolver resolves a tracked work item by key, so an open sweep's
// link is checked against the real entity rather than its spelling.
type BugWorkItemResolver func(ctx context.Context, key string) (models.Entity, error)

// bugOpenSweepLink is the retained record that links an open sweep, which has
// no feasible executable guard, to the work item that covers its class.
type bugOpenSweepLink struct {
	Kind        string `json:"kind"`
	ClassKey    string `json:"class_key"`
	WorkItemKey string `json:"work_item_key"`
	Reason      string `json:"reason"`
}

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
func verifyBugSweepGuards(ctx context.Context, req gatepersist.Request, items BugWorkItemResolver) error {
	if req.EntityType != models.EntityTypeBug || req.Result == nil {
		return nil
	}
	for _, sweep := range req.Result.RemediationSweeps {
		var err error
		if sweep.Status == "complete" {
			err = verifyBugSweepGuard(req, sweep, true)
		} else {
			err = verifyOpenSweepLink(ctx, req, sweep, items)
		}
		if err != nil {
			return fmt.Errorf("sweep %s: unsupported closure: %w", sweep.ClassKey, err)
		}
	}
	return nil
}

// verifyOpenSweepLink requires an open sweep to retain its reason and a link
// to an existing work item whose title or description covers the class.
func verifyOpenSweepLink(ctx context.Context, req gatepersist.Request, sweep gateresult.DefectClassSweep, items BugWorkItemResolver) error {
	pointer := sweep.Guard.CounterfactualPointer
	if !filepath.IsLocal(pointer) {
		return fmt.Errorf("open sweep link pointer must be project-local")
	}
	data, err := readBoundedBugEvidence(bugEvidenceProject(req), pointer, bugEvidenceLimit(""))
	if err != nil {
		return fmt.Errorf("open sweep link record is unreadable: %w", err)
	}
	var link bugOpenSweepLink
	if err := decodeStrict(data, &link); err != nil {
		return fmt.Errorf("invalid open sweep link record: %w", err)
	}
	if link.Kind != "open_sweep_link" || link.ClassKey != sweep.ClassKey || strings.TrimSpace(link.Reason) == "" || strings.TrimSpace(link.WorkItemKey) == "" {
		return fmt.Errorf("open sweep link must name its class, reason and work item")
	}
	if items == nil {
		return fmt.Errorf("work item %s cannot be resolved: no resolver is configured", link.WorkItemKey)
	}
	item, err := items(ctx, link.WorkItemKey)
	if err != nil || item == nil {
		return fmt.Errorf("linked work item %s does not resolve", link.WorkItemKey)
	}
	if !strings.Contains(item.GetTitle()+"\n"+item.GetDescription(), sweep.ClassKey) {
		return fmt.Errorf("linked work item %s does not cover class %s", link.WorkItemKey, sweep.ClassKey)
	}
	return nil
}

// verifyBugSweepGuard checks a completed sweep's guard pair. bindCandidate
// binds the corrected run to the candidate under review; historical sweeps
// were reviewed at an earlier candidate, so they only need a named one.
func verifyBugSweepGuard(req gatepersist.Request, sweep gateresult.DefectClassSweep, bindCandidate bool) error {
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
	return checkBugGuardExecution(req, sweep, run, bindCandidate)
}

func checkBugGuardExecution(req gatepersist.Request, sweep gateresult.DefectClassSweep, run bugGuardExecution, bindCandidate bool) error {
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
	if !bindCandidate {
		if strings.TrimSpace(run.Candidate) == "" {
			return fmt.Errorf("execution record does not name its corrected candidate")
		}
		return nil
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
