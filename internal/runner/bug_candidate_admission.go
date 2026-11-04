package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

func bugEvidenceProject(req gatepersist.Request) string {
	if req.EvidenceRoot != "" {
		return req.EvidenceRoot
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir)))
}

// readBugEvidence resolves one digest-bound, project-local regular file.
func readBugEvidence(req gatepersist.Request, kind string) (workercontrol.EvidenceRef, []byte, error) {
	ref, err := findBugEvidence(req, kind)
	if err != nil {
		return ref, nil, err
	}
	expected, err := validateBugEvidenceRef(ref)
	if err != nil {
		return ref, nil, err
	}
	root := bugEvidenceProject(req)
	data, err := readBoundedBugEvidence(root, ref.Pointer)
	if err != nil {
		return ref, nil, err
	}
	digest := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), hex.EncodeToString(expected)) {
		return ref, nil, fmt.Errorf("evidence digest is stale")
	}
	return ref, data, nil
}
func findBugEvidence(req gatepersist.Request, kind string) (workercontrol.EvidenceRef, error) {
	envelope, err := workercontrol.Decode(req.EnvelopeJSON)
	if err != nil {
		return workercontrol.EvidenceRef{}, err
	}
	var found *workercontrol.EvidenceRef
	for i := range envelope.Evidence {
		if envelope.Evidence[i].Kind == kind {
			if found != nil {
				return workercontrol.EvidenceRef{}, fmt.Errorf("exactly one %s reference is required", kind)
			}
			found = &envelope.Evidence[i]
		}
	}
	if found == nil {
		return workercontrol.EvidenceRef{}, fmt.Errorf("%s evidence is required", kind)
	}
	return *found, nil
}
func validateBugEvidenceRef(ref workercontrol.EvidenceRef) ([]byte, error) {
	if !filepath.IsLocal(ref.Pointer) || ref.Pointer == "." {
		return nil, fmt.Errorf("evidence path must be local to project")
	}
	if !strings.HasPrefix(ref.Summary, "sha256:") {
		return nil, fmt.Errorf("evidence digest is required")
	}
	expected, err := hex.DecodeString(strings.TrimPrefix(ref.Summary, "sha256:"))
	if err != nil || len(expected) != sha256.Size {
		return nil, fmt.Errorf("invalid evidence digest")
	}
	return expected, nil
}
func readBoundedBugEvidence(projectRoot, path string) ([]byte, error) {
	root, err := os.OpenRoot(projectRoot)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }() // Read-only handle; close does not affect verified bytes.
	file, err := root.OpenFile(path, os.O_RDONLY|bugPlanOpenFlags, 0)
	if err != nil {
		return nil, fmt.Errorf("open evidence: %w", err)
	}
	defer func() { _ = file.Close() }() // Read-only handle; close does not affect verified bytes.
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 2*1024*1024 {
		return nil, fmt.Errorf("evidence must be a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, 2*1024*1024+1))
	if err != nil || len(data) > 2*1024*1024 {
		return nil, fmt.Errorf("cannot read bounded evidence")
	}
	return data, nil
}

// readBugReviewScope binds a gate result to the exact saved scope and verifies
// that the source still produces the reviewed tree before any transition.
func readBugReviewScope(ctx context.Context, req gatepersist.Request) (workercontrol.EvidenceRef, error) {
	ref, data, err := readBugEvidence(req, "bug_review_scope")
	if err != nil {
		return ref, err
	}
	baseline, err := readBugBaseline(req)
	if err != nil {
		return ref, err
	}
	var scope bugCandidateScope
	if err := json.Unmarshal(data, &scope); err != nil {
		return ref, err
	}
	if scope.BaseCommit != baseline {
		return ref, fmt.Errorf("review scope baseline differs from approved baseline")
	}
	planRef, _, err := readBugEvidence(req, "bug_repair_plan")
	if err != nil {
		return ref, err
	}
	if !strings.EqualFold("sha256:"+scope.RepairPlanSHA256, planRef.Summary) {
		return ref, fmt.Errorf("review scope plan revision differs from approved plan")
	}
	root := bugEvidenceProject(req)
	if err := verifyBugCandidate(ctx, root, data); err != nil {
		return ref, fmt.Errorf("review candidate is stale or invalid: %w", err)
	}
	return ref, nil
}
