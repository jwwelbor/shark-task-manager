package runner

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/stretchr/testify/require"
)

func TestBugPlanMustBeCommittedAndNotIgnored(t *testing.T) {
	root := t.TempDir()
	path := "docs/plan/plan.md"
	plan := []byte("reviewed plan")
	commitTestBugPlan(t, root, path, plan)
	hash := sha256.Sum256(plan)
	req := gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "test"), EnvelopeJSON: []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_repair_plan","pointer":"%s","summary":"sha256:%x"}]}`, path, hash))}
	_, _, _, err := committedBugPlan(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, path), []byte("unapproved edit"), 0600))
	_, _, _, err = committedBugPlan(context.Background(), req)
	require.Error(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, path), plan, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "info", "exclude"), []byte("docs/plan/plan.md\n"), 0600))
	_, _, _, err = committedBugPlan(context.Background(), req)
	require.ErrorContains(t, err, "ignored")
	untracked := "docs/plan/untracked.md"
	require.NoError(t, os.WriteFile(filepath.Join(root, untracked), plan, 0600))
	req.EnvelopeJSON = []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_repair_plan","pointer":"%s","summary":"sha256:%x"}]}`, untracked, hash))
	_, _, _, err = committedBugPlan(context.Background(), req)
	require.ErrorContains(t, err, "not tracked")
}

func TestApprovedPlanProvenanceSurvivesWorktreeRemoval(t *testing.T) {
	parent := t.TempDir()
	path := "docs/plan/plan.md"
	commitTestBugPlan(t, parent, path, []byte("older plan\n"))
	worker := filepath.Join(t.TempDir(), "worker")
	cmd := exec.Command("git", "-C", parent, "worktree", "add", "--detach", worker, "HEAD")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	plan := []byte("approved worker plan\n")
	commitTestBugPlan(t, worker, path, plan)
	require.NoError(t, os.WriteFile(filepath.Join(parent, "unrelated.txt"), []byte("parallel change"), 0600))
	cmd = exec.Command("git", "-C", parent, "add", "unrelated.txt")
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	cmd = exec.Command("git", "-C", parent, "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "-m", "unrelated parent change")
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	dir, err := gaterun.RunDir(parent, "approved")
	require.NoError(t, err)
	hash := sha256.Sum256(plan)
	envelope := []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_repair_plan","pointer":"%s","summary":"sha256:%x"}]}`, path, hash))
	req := gatepersist.Request{RunDir: dir, EvidenceRoot: worker, EnvelopeJSON: envelope}
	root, commit, blob, err := bugPlanProvenance(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, retainBugPlanCommit(context.Background(), gatepersist.Request{RunDir: dir, RunID: "approved", PlanCommit: commit}))
	_, err = gaterun.CreateIdentity(dir, gaterun.RunIdentity{RunID: "approved", EntityKey: "B001", EntityType: "bug", Gate: "plan_review", OperationDigest: "operation", EvidenceRoot: worker, PlanRoot: root, PlanCommit: commit, PlanBlob: blob})
	require.NoError(t, err)
	_, err = gaterun.CreateApprovedPlan(dir, plan)
	require.NoError(t, err)
	require.NoError(t, verifyBugPlanProvenance(context.Background(), parent, dir, envelope))
	require.NoError(t, os.WriteFile(filepath.Join(worker, path), []byte("unreviewed edit"), 0600))
	require.ErrorContains(t, verifyBugPlanProvenance(context.Background(), parent, dir, envelope), "approved project plan changed")
	require.NoError(t, os.WriteFile(filepath.Join(worker, path), plan, 0600))
	cmd = exec.Command("git", "-C", parent, "worktree", "remove", worker)
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	cmd = exec.Command("git", "-C", parent, "reflog", "expire", "--expire=now", "--all")
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	cmd = exec.Command("git", "-C", parent, "gc", "--prune=now")
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	require.NoError(t, verifyBugPlanProvenance(context.Background(), parent, dir, envelope))
	require.NoError(t, verifyCurrentBugPlan(context.Background(), parent, dir, envelope), "a clean older parent copy is not an edited approved plan")
	require.NoError(t, os.WriteFile(filepath.Join(parent, path), []byte("unreviewed active edit"), 0600))
	require.ErrorContains(t, verifyCurrentBugPlan(context.Background(), parent, dir, envelope), "changed without approval")
	require.NoError(t, os.WriteFile(filepath.Join(parent, path), []byte("older plan\n"), 0600))
	require.NoError(t, os.Remove(filepath.Join(parent, path)))
	require.ErrorContains(t, verifyCurrentBugPlan(context.Background(), parent, dir, envelope), "changed without approval")
	require.NoError(t, os.WriteFile(filepath.Join(parent, path), []byte("staged plan"), 0600))
	cmd = exec.Command("git", "-C", parent, "add", path)
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	require.NoError(t, os.WriteFile(filepath.Join(parent, path), plan, 0600))
	require.ErrorContains(t, verifyCurrentBugPlan(context.Background(), parent, dir, envelope), "changed without approval")
	cmd = exec.Command("git", "-C", parent, "reset", "--", path)
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	require.NoError(t, os.WriteFile(filepath.Join(parent, path), []byte("older plan\n"), 0600))
	symlink := filepath.Join(t.TempDir(), "project-link")
	if err := os.Symlink(parent, symlink); err == nil {
		require.NoError(t, verifyBugPlanRef(context.Background(), symlink, "approved", commit))
	}
	moved := filepath.Join(t.TempDir(), "moved-project")
	require.NoError(t, os.Rename(parent, moved))
	movedDir := filepath.Join(moved, ".shark", "runs", "approved")
	require.NoError(t, verifyBugPlanProvenance(context.Background(), moved, movedDir, envelope))
}

func TestCommittedBugPlanRespectsCheckoutLineEndings(t *testing.T) {
	root := t.TempDir()
	path := "docs/plan/plan.md"
	plan := []byte("reviewed line\r\n")
	commitTestBugPlan(t, root, path, plan)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("*.md text eol=crlf\n"), 0600))
	cmd := exec.Command("git", "-C", root, "add", ".gitattributes", path)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	cmd = exec.Command("git", "-C", root, "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "-m", "normalize plan")
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	hash := sha256.Sum256(plan)
	req := gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "test"), EnvelopeJSON: []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_repair_plan","pointer":"%s","summary":"sha256:%x"}]}`, path, hash))}
	_, got, _, err := committedBugPlan(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, plan, got)
}

func TestCommittedBugPlanChoosesMatchingCheckout(t *testing.T) {
	parent := t.TempDir()
	path := "docs/plan/plan.md"
	old := []byte("old committed plan\n")
	commitTestBugPlan(t, parent, path, old)
	worker := filepath.Join(t.TempDir(), "worker")
	cmd := exec.Command("git", "-C", parent, "worktree", "add", "--detach", worker, "HEAD")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	newPlan := []byte("new committed plan\n")
	commitTestBugPlan(t, worker, path, newPlan)
	for _, tc := range []struct {
		plan []byte
		root string
	}{{newPlan, worker}, {old, parent}} {
		hash := sha256.Sum256(tc.plan)
		req := gatepersist.Request{RunDir: filepath.Join(parent, ".shark", "runs", "test"), EvidenceRoot: worker, EnvelopeJSON: []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_repair_plan","pointer":"%s","summary":"sha256:%x"}]}`, path, hash))}
		_, got, root, err := committedBugPlan(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, tc.plan, got)
		require.Equal(t, tc.root, root)
	}
}

func TestActiveCheckoutMayPredatePlanButNotDeleteIt(t *testing.T) {
	parent := t.TempDir()
	run := func(root string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	run(parent, "init")
	run(parent, "config", "user.name", "Test")
	run(parent, "config", "user.email", "test@example.invalid")
	run(parent, "commit", "--allow-empty", "-m", "before plan")
	worker := filepath.Join(t.TempDir(), "worker")
	run(parent, "worktree", "add", "--detach", worker, "HEAD")
	path := "docs/plan/plan.md"
	plan := []byte("reviewed plan")
	commitTestBugPlan(t, worker, path, plan)
	dir, err := gaterun.RunDir(parent, "approved")
	require.NoError(t, err)
	hash := sha256.Sum256(plan)
	envelope := []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_repair_plan","pointer":"%s","summary":"sha256:%x"}]}`, path, hash))
	root, commit, blob, err := bugPlanProvenance(context.Background(), gatepersist.Request{RunDir: dir, EvidenceRoot: worker, EnvelopeJSON: envelope})
	require.NoError(t, err)
	_, err = gaterun.CreateIdentity(dir, gaterun.RunIdentity{RunID: "approved", EntityKey: "B001", EntityType: "bug", Gate: "plan_review", OperationDigest: "operation", PlanRoot: root, PlanCommit: commit, PlanBlob: blob})
	require.NoError(t, err)
	require.NoError(t, verifyCurrentBugPlan(context.Background(), parent, dir, envelope), "the parent predates the plan")
	run(parent, "merge", "--ff-only", commit)
	run(parent, "rm", path)
	run(parent, "commit", "-m", "delete plan")
	require.ErrorContains(t, verifyCurrentBugPlan(context.Background(), parent, dir, envelope), "changed without approval")
}
