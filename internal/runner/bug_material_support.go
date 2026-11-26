package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gitutil"
)

// materialRunObservation is one recorded execution of a production caller
// against an acceptance criterion at a candidate.
type materialRunObservation struct {
	Owner     string `json:"owner"`
	Caller    string `json:"caller"`
	Criterion string `json:"criterion"`
	Candidate string `json:"candidate"`
	Outcome   string `json:"outcome"`
}

// verifyMaterialSupport resolves the evidence behind each disposition instead
// of trusting its pointer spelling: an amended artifact must no longer contain
// the invalidated text in the frozen candidate, and a regression_pass must be
// backed by recorded passing runs of that exact caller and criterion at the
// candidate. tracked_follow_up is owner-authorized outside the frozen tree and
// is not resolved here.
func verifyMaterialSupport(ctx context.Context, req gatepersist.Request, scope bugCandidateScope, expected bugReviewMaterialDecision, actual bugReviewMaterialDisposition) error {
	if err := verifyAmendedArtifacts(ctx, scope, expected, actual.Artifacts); err != nil {
		return err
	}
	return verifyPassingCriteria(req, scope, expected, actual.Criteria)
}

func verifyAmendedArtifacts(ctx context.Context, scope bugCandidateScope, expected bugReviewMaterialDecision, submitted []bugReviewArtifactDisposition) error {
	invalidated := make(map[string]string, len(expected.AffectedArtifacts))
	for _, artifact := range expected.AffectedArtifacts {
		invalidated[artifact.Path] = artifact.InvalidatedText
	}
	for _, item := range submitted {
		if item.Disposition != "amended" {
			continue
		}
		if filepath.ToSlash(filepath.Clean(item.EvidencePointer)) != item.Path {
			return fmt.Errorf("%s: amendment evidence for %s must be the frozen artifact itself", expected.DecisionID, item.Path)
		}
		content, err := readFrozenMaterialBlob(ctx, scope, item.Path)
		if err != nil {
			return fmt.Errorf("%s: amendment of %s: %w", expected.DecisionID, item.Path, err)
		}
		if bytes.Contains(content, []byte(invalidated[item.Path])) {
			return fmt.Errorf("%s: %s still contains the invalidated text", expected.DecisionID, item.Path)
		}
	}
	return nil
}

func verifyPassingCriteria(req gatepersist.Request, scope bugCandidateScope, expected bugReviewMaterialDecision, submitted []bugReviewCriterionDisposition) error {
	runs := make(map[string][]materialRunObservation)
	for _, item := range submitted {
		if item.Disposition != "regression_pass" {
			continue
		}
		key := materialCriterionKey{item.EntityKey, item.ProductionCallerPath, item.AcceptanceCriterionID}
		if _, loaded := runs[item.EvidencePointer]; !loaded {
			observed, err := readMaterialRuns(req, item.EvidencePointer)
			if err != nil {
				return fmt.Errorf("%s: run evidence for %s: %w", expected.DecisionID, key, err)
			}
			runs[item.EvidencePointer] = observed
		}
		if err := requirePassingRun(runs[item.EvidencePointer], key, scope.CandidateCommit); err != nil {
			return fmt.Errorf("%s: %s: %w", expected.DecisionID, key, err)
		}
	}
	return nil
}

func requirePassingRun(observed []materialRunObservation, key materialCriterionKey, candidate string) error {
	passed := false
	for _, run := range observed {
		if run.Owner != key.owner || run.Caller != key.caller || run.Criterion != key.criterion || run.Candidate != candidate {
			continue
		}
		if run.Outcome != "pass" {
			return fmt.Errorf("recorded run at the candidate did not pass")
		}
		passed = true
	}
	if !passed {
		return fmt.Errorf("no recorded passing run at the candidate")
	}
	return nil
}

func readMaterialRuns(req gatepersist.Request, pointer string) ([]materialRunObservation, error) {
	data, err := readBoundedBugEvidence(bugEvidenceProject(req), pointer, bugEvidenceLimit(""))
	if err != nil {
		return nil, err
	}
	var observed []materialRunObservation
	if err := json.Unmarshal(data, &observed); err != nil {
		return nil, fmt.Errorf("decode run observations: %w", err)
	}
	return observed, nil
}

// readFrozenMaterialBlob reads a regular, size-bounded blob from the candidate tree.
func readFrozenMaterialBlob(ctx context.Context, scope bugCandidateScope, path string) ([]byte, error) {
	root, _, err := candidateRepository(ctx, scope.ProjectRoot)
	if err != nil {
		return nil, err
	}
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." || !filepath.IsLocal(path) {
		return nil, fmt.Errorf("invalid artifact path %q", path)
	}
	entry, err := gitutil.Command(ctx, "--literal-pathspecs", "-C", root, "ls-tree", "-l", "-z", "--full-tree", scope.CandidateCommit, "--", path).Output()
	if err != nil {
		return nil, err
	}
	oid, size, err := materialBlobEntry(string(entry), path)
	if err != nil {
		return nil, err
	}
	cmd := gitutil.Command(ctx, "-C", root, "cat-file", "blob", oid)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(out, size+1))
	waitErr := cmd.Wait()
	if readErr != nil || int64(len(data)) != size {
		return nil, fmt.Errorf("blob size differs from frozen entry")
	}
	return data, waitErr
}
