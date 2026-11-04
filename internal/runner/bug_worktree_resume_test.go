package runner

import (
	"context"
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
	first := filepath.Join(t.TempDir(), "first")
	git("worktree", "add", "-b", "shark-run-B001-1", first)
	path, err = RetainedBugWorktree(context.Background(), root, "B001")
	require.NoError(t, err)
	require.Equal(t, first, path)
	second := filepath.Join(t.TempDir(), "second")
	git("worktree", "add", "-b", "shark-run-B001-2", second)
	_, err = RetainedBugWorktree(context.Background(), root, "B001")
	require.ErrorContains(t, err, "multiple retained worktrees")
}
