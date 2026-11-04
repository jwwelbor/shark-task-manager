package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

// committedBugPlan selects a checkout in which the referenced plan is tracked,
// not ignored, and byte-identical to its current committed revision.
func committedBugPlan(ctx context.Context, req gatepersist.Request) (workercontrol.EvidenceRef, []byte, string, error) {
	ref, err := findBugEvidence(req, "bug_repair_plan")
	if err != nil {
		return ref, nil, "", err
	}
	if _, err := validateBugEvidenceRef(ref); err != nil {
		return ref, nil, "", err
	}
	clean := filepath.Clean(ref.Pointer)
	if !strings.HasPrefix(filepath.ToSlash(clean), "docs/plan/") {
		return ref, nil, "", fmt.Errorf("bug repair plan must be a project document under docs/plan")
	}
	roots := []string{bugEvidenceProject(req)}
	if req.EvidenceRoot != "" {
		roots = append(roots, filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir))))
	}
	var selected []byte
	var selectedRoot string
	var last error
	for _, root := range roots {
		plan, err := committedBugPlanAt(ctx, root, ref)
		if err == nil {
			if selected == nil {
				selected = plan
				selectedRoot, err = canonicalBugProject(root)
				if err != nil {
					return ref, nil, "", err
				}
			}
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return ref, nil, "", fmt.Errorf("plan must be tracked, not ignored, and committed in each checkout containing it: %w", err)
		}
		last = err
	}
	if selected != nil {
		return ref, selected, selectedRoot, nil
	}
	return ref, nil, "", fmt.Errorf("plan must be tracked, not ignored, and committed in a selected checkout: %w", last)
}

func committedBugPlanAt(ctx context.Context, root string, ref workercontrol.EvidenceRef) ([]byte, error) {
	expected, err := validateBugEvidenceRef(ref)
	if err != nil {
		return nil, err
	}
	plan, err := readBoundedBugEvidence(root, ref.Pointer, bugEvidenceLimit("bug_repair_plan"))
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(plan)
	if !bytes.Equal(hash[:], expected) {
		return nil, fmt.Errorf("plan digest is stale")
	}
	_, relative, err := candidateRepository(ctx, root)
	if err != nil {
		return nil, err
	}
	run := func(args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "git", append([]string{"-c", "core.fsmonitor=false", "-C", root}, args...)...).Output()
	}
	if _, err := run("ls-files", "--error-unmatch", "--", ref.Pointer); err != nil {
		return nil, fmt.Errorf("plan is not tracked")
	}
	_, err = run("check-ignore", "--no-index", "-q", "--", ref.Pointer)
	if err == nil {
		return nil, fmt.Errorf("plan is ignored")
	}
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
		return nil, fmt.Errorf("cannot check plan ignore status: %w", err)
	}
	object := "HEAD:" + filepath.ToSlash(filepath.Join(relative, ref.Pointer))
	kind, err := run("cat-file", "-t", object)
	if err != nil || strings.TrimSpace(string(kind)) != "blob" {
		return nil, fmt.Errorf("plan is not in the current commit")
	}
	committed, err := run("rev-parse", "--verify", "--end-of-options", object)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve committed plan")
	}
	cmd := exec.CommandContext(ctx, "git", "-c", "core.fsmonitor=false", "-C", root, "hash-object", "--stdin")
	cmd.Stdin = bytes.NewReader(plan)
	actual, err := cmd.Output()
	if err != nil || !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(committed)) {
		return nil, fmt.Errorf("plan differs from the current commit")
	}
	return plan, nil
}
