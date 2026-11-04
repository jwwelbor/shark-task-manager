package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

func TestBugEvidenceRootBindsWorkerAndRecoversFromIdentity(t *testing.T) {
	ctx := context.Background()
	parent := t.TempDir()
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	git(parent, "init", "-b", "main")
	git(parent, "config", "user.name", "Test")
	git(parent, "config", "user.email", "test@example.invalid")
	require.NoError(t, os.WriteFile(filepath.Join(parent, "file"), []byte("base"), 0600))
	git(parent, "add", "file")
	git(parent, "commit", "-m", "base")
	worker := filepath.Join(t.TempDir(), "worker")
	git(parent, "worktree", "add", "--detach", worker, "HEAD")
	runDir, err := gaterun.RunDir(parent, "root-test")
	require.NoError(t, err)
	req := GateIngestRequest{ProjectRoot: parent, WorkingDir: worker, RunID: "root-test", EntityKey: "B001", EntityType: models.EntityTypeBug, SourceStatus: "plan_review", Gate: "plan_review", EnvelopeBytes: []byte(`{"kind":"final"}`)}
	bound, err := gateEvidenceRoot(ctx, req, runDir, gateresult.RoleSuccess)
	require.NoError(t, err)
	require.Equal(t, worker, bound)
	digest, err := gaterun.ComputeOperationDigest(req.EntityKey, string(req.EntityType), req.SourceStatus, req.Gate, req.EnvelopeBytes)
	require.NoError(t, err)
	identity := gaterun.RunIdentity{RunID: "root-test", EntityKey: "B001", EntityType: "bug", SourceStatus: "plan_review", Gate: "plan_review", OperationDigest: digest, EvidenceRoot: bound}
	_, err = gaterun.CreateIdentity(runDir, identity)
	require.NoError(t, err)
	req.WorkingDir = ""
	replayed, err := gateEvidenceRoot(ctx, req, runDir, gateresult.RoleSuccess)
	require.NoError(t, err)
	require.Equal(t, bound, replayed)
	changed := req
	changed.EnvelopeBytes = []byte(`{"kind":"final","changed":true}`)
	_, err = gateEvidenceRoot(ctx, changed, runDir, gateresult.RoleSuccess)
	require.ErrorContains(t, err, "operation_digest mismatch")
	foreign := t.TempDir()
	git(foreign, "init", "-b", "main")
	req.WorkingDir = foreign
	_, err = gateEvidenceRoot(ctx, req, runDir, gateresult.RoleSuccess)
	require.Error(t, err)
	// Identity lookup for an applied transition does not require the old
	// worktree to remain present; pending admission separately rechecks it.
	require.NoError(t, os.Rename(worker, worker+"-missing"))
	req.WorkingDir = ""
	replayed, err = gateEvidenceRoot(ctx, req, runDir, gateresult.RoleSuccess)
	require.NoError(t, err)
	require.Equal(t, bound, replayed)
}
