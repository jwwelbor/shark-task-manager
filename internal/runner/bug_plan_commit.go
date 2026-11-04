package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
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
	if req.PlanRoot != "" {
		roots = []string{req.PlanRoot}
	}
	var last error
	for _, root := range roots {
		plan, err := committedBugPlanAt(ctx, root, ref)
		if err == nil {
			canonical, err := canonicalBugProject(root)
			return ref, plan, canonical, err
		}
		last = err
	}
	return ref, nil, "", fmt.Errorf("plan must be tracked, not ignored, and committed in a selected checkout: %w", last)
}

func verifyBugPlanProvenance(ctx context.Context, parent, dir string, envelope []byte) error {
	identity, exists, err := gaterun.ReadIdentity(dir)
	if err != nil || !exists || identity.PlanRoot == "" || identity.PlanCommit == "" || identity.PlanBlob == "" {
		return fmt.Errorf("approved plan provenance is unavailable")
	}
	if err := verifyBugPlanRef(ctx, parent, identity.RunID, identity.PlanCommit); err != nil {
		return err
	}
	for _, hash := range []string{identity.PlanCommit, identity.PlanBlob} {
		if len(hash) != 40 && len(hash) != 64 {
			return fmt.Errorf("approved plan provenance contains an invalid Git object ID")
		}
		if _, err := hex.DecodeString(hash); err != nil {
			return fmt.Errorf("approved plan provenance contains an invalid Git object ID")
		}
	}
	ref, err := findBugEvidence(gatepersist.Request{EnvelopeJSON: envelope}, "bug_repair_plan")
	if err != nil {
		return err
	}
	_, relative, err := candidateRepository(ctx, parent)
	if err != nil {
		return err
	}
	commitType := exec.CommandContext(ctx, "git", "-c", "core.fsmonitor=false", "-C", parent, "cat-file", "-t", identity.PlanCommit)
	kind, err := commitType.Output()
	if err != nil || strings.TrimSpace(string(kind)) != "commit" {
		return fmt.Errorf("approved plan commit is unavailable")
	}
	commitID := exec.CommandContext(ctx, "git", "-c", "core.fsmonitor=false", "-C", parent, "rev-parse", "--verify", "--end-of-options", identity.PlanCommit)
	fullCommit, err := commitID.Output()
	if err != nil || !strings.EqualFold(strings.TrimSpace(string(fullCommit)), identity.PlanCommit) {
		return fmt.Errorf("approved plan commit identity is invalid")
	}
	object := identity.PlanCommit + ":" + filepath.ToSlash(filepath.Join(relative, ref.Pointer))
	cmd := exec.CommandContext(ctx, "git", "-c", "core.fsmonitor=false", "-C", parent, "rev-parse", "--verify", "--end-of-options", object)
	blob, err := cmd.Output()
	if err != nil || !strings.EqualFold(strings.TrimSpace(string(blob)), identity.PlanBlob) {
		return fmt.Errorf("approved plan commit or blob is unavailable")
	}
	if _, err := os.Stat(identity.PlanRoot); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect approved plan checkout: %w", err)
	}
	root, err := BugWorktreeProjectRoot(ctx, parent, identity.PlanRoot)
	if err != nil || root != identity.PlanRoot {
		return fmt.Errorf("approved plan checkout is invalid")
	}
	if _, err := committedBugPlanAt(ctx, root, ref); err != nil {
		return fmt.Errorf("approved project plan changed: %w", err)
	}
	return nil
}

func bugPlanRef(ctx context.Context, parent, runID, commit string) (string, error) {
	_, project, err := candidateRepository(ctx, parent)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(filepath.ToSlash(project) + "\x00" + runID + "\x00" + strings.ToLower(commit)))
	return fmt.Sprintf("refs/shark/approved-plans/%x", hash), nil
}

func verifyBugPlanRef(ctx context.Context, parent, runID, commit string) error {
	ref, err := bugPlanRef(ctx, parent, runID, commit)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "git", "-c", "core.fsmonitor=false", "-C", parent, "rev-parse", "--verify", "--end-of-options", ref)
	out, err := cmd.Output()
	if err != nil || !strings.EqualFold(strings.TrimSpace(string(out)), commit) {
		return fmt.Errorf("approved plan retention reference is unavailable or mismatched")
	}
	return nil
}

func retainBugPlanCommit(ctx context.Context, req gatepersist.Request) error {
	parent := filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir)))
	ref, err := bugPlanRef(ctx, parent, req.RunID, req.PlanCommit)
	if err != nil {
		return err
	}
	zero := strings.Repeat("0", len(req.PlanCommit))
	cmd := exec.CommandContext(ctx, "git", "-c", "core.fsmonitor=false", "-C", parent, "update-ref", ref, req.PlanCommit, zero)
	if err := cmd.Run(); err != nil {
		if verifyErr := verifyBugPlanRef(ctx, parent, req.RunID, req.PlanCommit); verifyErr != nil {
			return fmt.Errorf("retain approved plan commit: %w", verifyErr)
		}
	}
	return nil
}

// verifyCurrentBugPlan rejects an unreviewed edit in the active checkout. A
// clean older committed copy is allowed: the approved snapshot remains the
// authority when a parent checkout has not yet received the reviewed commit.
func verifyCurrentBugPlan(ctx context.Context, current, dir string, envelope []byte) error {
	identity, exists, err := gaterun.ReadIdentity(dir)
	if err != nil || !exists {
		return fmt.Errorf("approved plan identity is unavailable")
	}
	ref, err := findBugEvidence(gatepersist.Request{EnvelopeJSON: envelope}, "bug_repair_plan")
	if err != nil {
		return err
	}
	expected, err := validateBugEvidenceRef(ref)
	if err != nil {
		return err
	}
	run := func(args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "git", append([]string{"--literal-pathspecs", "-c", "core.fsmonitor=false", "-C", current}, args...)...).Output()
	}
	repository, relative, err := candidateRepository(ctx, current)
	if err != nil {
		return err
	}
	path := filepath.ToSlash(filepath.Join(relative, ref.Pointer))
	plan, err := readBoundedBugEvidence(current, ref.Pointer, bugEvidenceLimit("bug_repair_plan"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if _, trackedErr := run("ls-files", "--error-unmatch", "--", ref.Pointer); trackedErr == nil {
				return fmt.Errorf("active project plan changed without approval")
			}
			head, headErr := run("rev-parse", "--verify", "HEAD")
			if headErr != nil {
				return headErr
			}
			headPlan, headPlanErr := run("ls-tree", "--full-tree", "-z", strings.TrimSpace(string(head)), "--", path)
			if headPlanErr != nil || len(headPlan) != 0 {
				return fmt.Errorf("active project plan changed without approval")
			}
			base, baseErr := run("merge-base", strings.TrimSpace(string(head)), identity.PlanCommit)
			if baseErr != nil {
				return fmt.Errorf("active project plan changed without approval")
			}
			basePlan, basePlanErr := run("ls-tree", "--full-tree", "-z", strings.TrimSpace(string(base)), "--", path)
			if basePlanErr != nil || len(basePlan) != 0 {
				return fmt.Errorf("active project plan changed without approval")
			}
			return nil
		}
		return err
	}
	if _, err := run("ls-files", "--error-unmatch", "--", ref.Pointer); err != nil {
		return fmt.Errorf("active project plan changed without approval")
	}
	status, err := run("status", "--porcelain=v1", "-z", "--", ref.Pointer)
	if err != nil || len(status) != 0 {
		return fmt.Errorf("active project plan changed without approval")
	}
	head, err := run("rev-parse", "--verify", "HEAD")
	if err != nil {
		return fmt.Errorf("active project plan changed without approval")
	}
	rootGit := func(args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "git", append([]string{"-c", "core.fsmonitor=false", "-C", repository}, args...)...).Output()
	}
	headBlob, headErr := rootGit("rev-parse", "--verify", "--end-of-options", strings.TrimSpace(string(head))+":"+path)
	indexBlob, indexErr := rootGit("rev-parse", "--verify", "--end-of-options", ":"+path)
	working := exec.CommandContext(ctx, "git", "-c", "core.fsmonitor=false", "-C", current, "hash-object", "--path", ref.Pointer, "--stdin")
	working.Stdin = bytes.NewReader(plan)
	workingBlob, workingErr := working.Output()
	if headErr != nil || indexErr != nil || workingErr != nil || !bytes.Equal(bytes.TrimSpace(headBlob), bytes.TrimSpace(indexBlob)) || !bytes.Equal(bytes.TrimSpace(headBlob), bytes.TrimSpace(workingBlob)) {
		return fmt.Errorf("active project plan changed without approval")
	}
	sum := sha256.Sum256(plan)
	if bytes.Equal(sum[:], expected) {
		return nil
	}
	base, err := run("merge-base", strings.TrimSpace(string(head)), identity.PlanCommit)
	if err != nil {
		return fmt.Errorf("active project plan changed without approval")
	}
	oldBlob, oldErr := run("rev-parse", "--verify", "--end-of-options", strings.TrimSpace(string(base))+":"+path)
	currentBlob, currentErr := run("rev-parse", "--verify", "--end-of-options", strings.TrimSpace(string(head))+":"+path)
	if oldErr != nil || currentErr != nil || !bytes.Equal(bytes.TrimSpace(oldBlob), bytes.TrimSpace(currentBlob)) {
		return fmt.Errorf("active project plan changed without approval")
	}
	return nil
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
	repository, relative, err := candidateRepository(ctx, root)
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
	indexPath := ":" + filepath.ToSlash(filepath.Join(relative, ref.Pointer))
	index := exec.CommandContext(ctx, "git", "-c", "core.fsmonitor=false", "-C", repository, "rev-parse", "--verify", "--end-of-options", indexPath)
	staged, err := index.Output()
	if err != nil || !bytes.Equal(bytes.TrimSpace(staged), bytes.TrimSpace(committed)) {
		return nil, fmt.Errorf("plan index differs from the current commit")
	}
	cmd := exec.CommandContext(ctx, "git", "-c", "core.fsmonitor=false", "-C", root, "hash-object", "--path", ref.Pointer, "--stdin")
	cmd.Stdin = bytes.NewReader(plan)
	actual, err := cmd.Output()
	if err != nil || !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(committed)) {
		return nil, fmt.Errorf("plan differs from the current commit")
	}
	return plan, nil
}

func bugPlanProvenance(ctx context.Context, req gatepersist.Request) (string, string, string, error) {
	ref, plan, root, err := committedBugPlan(ctx, req)
	if err != nil {
		return "", "", "", err
	}
	_, relative, err := candidateRepository(ctx, root)
	if err != nil {
		return "", "", "", err
	}
	run := func(args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "git", append([]string{"-c", "core.fsmonitor=false", "-C", root}, args...)...).Output()
	}
	commit, err := run("rev-parse", "--verify", "HEAD")
	if err != nil {
		return "", "", "", err
	}
	object := strings.TrimSpace(string(commit)) + ":" + filepath.ToSlash(filepath.Join(relative, ref.Pointer))
	blob, err := run("rev-parse", "--verify", "--end-of-options", object)
	if err != nil {
		return "", "", "", err
	}
	hash := exec.CommandContext(ctx, "git", "-c", "core.fsmonitor=false", "-C", root, "hash-object", "--path", ref.Pointer, "--stdin")
	hash.Stdin = bytes.NewReader(plan)
	actual, err := hash.Output()
	if err != nil || !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(blob)) {
		return "", "", "", fmt.Errorf("committed plan changed while resolving provenance")
	}
	// Recheck after resolving the commit so a concurrent checkout change is
	// rejected rather than recorded as evidence for different bytes.
	if _, err := committedBugPlanAt(ctx, root, ref); err != nil {
		return "", "", "", err
	}
	return root, strings.TrimSpace(string(commit)), strings.TrimSpace(string(blob)), nil
}
