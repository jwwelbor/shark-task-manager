package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// gateEvidenceRoot keeps run sidecars in the parent while resolving bug
// evidence in the parent-selected worktree. A stored identity owns replay.
func gateEvidenceRoot(ctx context.Context, req GateIngestRequest, runDir string, role gateresult.OutcomeRole) (string, error) {
	if req.EntityType != models.EntityTypeBug || role != gateresult.RoleSuccess || !bugEvidenceGate(req.Gate) {
		return "", nil
	}
	identity, exists, err := gaterun.ReadIdentity(runDir)
	if err != nil {
		return "", err
	}
	return selectedEvidenceRoot(ctx, req, identity, exists)
}

func selectedEvidenceRoot(ctx context.Context, req GateIngestRequest, identity *gaterun.RunIdentity, exists bool) (string, error) {
	if exists {
		if err := verifyEvidenceRootIdentity(req, identity); err != nil {
			return "", err
		}
		if req.WorkingDir == "" {
			// A completed transition can retire even after worktree removal.
			// Pending admission revalidates the root before reading it.
			return identity.EvidenceRoot, nil
		}
	}
	if req.WorkingDir == "" {
		return "", nil
	}
	root, err := resolveBugEvidenceRoot(ctx, req.ProjectRoot, req.WorkingDir)
	if err != nil {
		return "", err
	}
	if exists && root != identity.EvidenceRoot {
		return "", fmt.Errorf("worker evidence root conflicts with run identity")
	}
	return root, nil
}

func verifyEvidenceRootIdentity(req GateIngestRequest, identity *gaterun.RunIdentity) error {
	digest, err := gaterun.ComputeOperationDigest(req.EntityKey, string(req.EntityType), req.SourceStatus, req.Gate, req.EnvelopeBytes)
	if err != nil {
		return err
	}
	want := gaterun.RunIdentity{EntityKey: req.EntityKey, EntityType: string(req.EntityType), SourceStatus: req.SourceStatus, Gate: req.Gate, OperationDigest: digest, EvidenceRoot: identity.EvidenceRoot}
	if identity.RunID != req.RunID {
		return fmt.Errorf("worker evidence run identity mismatch")
	}
	return gaterun.VerifyRunIdentity(identity, want)
}

func bugEvidenceGate(gate string) bool {
	switch gate {
	case "repair_plan", "plan_review", "development", "code_review", "qa":
		return true
	default:
		return false
	}
}

// BugWorktreeProjectRoot maps the parent Shark project into its linked worktree.
// Workers and evidence admission must use the same project-relative paths.
func BugWorktreeProjectRoot(ctx context.Context, parentProject, worktree string) (string, error) {
	root, err := resolveBugEvidenceRoot(ctx, parentProject, worktree)
	if err != nil {
		return "", err
	}
	if root == "" {
		return canonicalBugProject(parentProject)
	}
	return root, nil
}

// ValidateBugWorkingDir checks the worker's actual cwd before dispatch. Bug
// evidence pointers are project-relative, so a different cwd would otherwise
// cause workers and the parent to interpret the same pointer differently.
func ValidateBugWorkingDir(ctx context.Context, parentProject, workingDir string) error {
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve inherited bug working directory: %w", err)
		}
	}
	project, err := BugWorktreeProjectRoot(ctx, parentProject, workingDir)
	if err != nil {
		return fmt.Errorf("bug workdir %q cannot be bound to project %q: %w", workingDir, parentProject, err)
	}
	actual, err := canonicalBugProject(workingDir)
	if err != nil {
		return fmt.Errorf("resolve bug workdir %q: %w", workingDir, err)
	}
	if actual != project {
		return fmt.Errorf("bug workdir %q must be the corresponding Shark project root %q so evidence paths agree", actual, project)
	}
	return nil
}

func resolveBugEvidenceRoot(ctx context.Context, parentProject, workingDir string) (string, error) {
	parentRepo, relative, err := candidateRepository(ctx, parentProject)
	if err != nil {
		return "", err
	}
	if !filepath.IsLocal(relative) {
		return "", fmt.Errorf("Shark project is outside its repository")
	}
	workerRepo, _, err := candidateRepository(ctx, workingDir)
	if err != nil {
		return "", err
	}
	run := func(root string, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "git", append([]string{"-c", "core.fsmonitor=false", "-C", root}, args...)...).Output()
	}
	if err := validateSnapshotRepository(parentRepo, workerRepo, run); err != nil {
		return "", err
	}
	workerProject, err := filepath.EvalSymlinks(filepath.Join(workerRepo, relative))
	if err != nil {
		return "", fmt.Errorf("resolve worker project: %w", err)
	}
	actualRelative, err := filepath.Rel(workerRepo, workerProject)
	if err != nil || actualRelative != relative {
		return "", fmt.Errorf("worker project does not match repository location")
	}
	parentProject, err = canonicalBugProject(parentProject)
	if err != nil {
		return "", err
	}
	if workerProject == parentProject {
		return "", nil
	}
	return workerProject, nil
}

func canonicalBugProject(project string) (string, error) {
	absolute, err := filepath.Abs(project)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}

// verifyBugEvidenceRoot rechecks the parent-owned root before a pending
// admission reads files; completed transitions may resume without this read.
func verifyBugEvidenceRoot(ctx context.Context, req gatepersist.Request) error {
	if req.EvidenceRoot == "" {
		return nil
	}
	parent := filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir)))
	verified, err := resolveBugEvidenceRoot(ctx, parent, req.EvidenceRoot)
	if err != nil || verified != req.EvidenceRoot {
		return fmt.Errorf("bug evidence root is unavailable or invalid")
	}
	return nil
}
