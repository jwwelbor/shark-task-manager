package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetainedBugWorktreeRequiresUnambiguousCheckout(t *testing.T) {
	root := t.TempDir()
	commitTestBugPlan(t, root, "docs/plan/plan.md", []byte("plan"))
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	path, err := RetainedBugWorktree(context.Background(), root, "B001")
	require.NoError(t, err)
	require.Empty(t, path)
	base := filepath.Join(root, DefaultWorktreeBaseDir)
	require.NoError(t, os.MkdirAll(base, 0700))
	first := filepath.Join(base, "first")
	git("worktree", "add", "-b", "shark-run-B001-1", first)
	path, err = RetainedBugWorktree(context.Background(), root, "B001")
	require.NoError(t, err)
	require.Equal(t, first, path)
	otherProject := filepath.Join(root, "other-project")
	require.NoError(t, os.MkdirAll(filepath.Join(otherProject, DefaultWorktreeBaseDir), 0700))
	other := filepath.Join(otherProject, DefaultWorktreeBaseDir, "other")
	git("worktree", "add", "-b", "shark-run-B001-other", other)
	path, err = RetainedBugWorktree(context.Background(), root, "B001")
	require.NoError(t, err)
	require.Equal(t, first, path, "a sibling project worktree must not be reused")
	second := filepath.Join(base, "second")
	git("worktree", "add", "-b", "shark-run-B001-2", second)
	_, err = RetainedBugWorktree(context.Background(), root, "B001")
	require.ErrorContains(t, err, "multiple retained worktrees")
}
