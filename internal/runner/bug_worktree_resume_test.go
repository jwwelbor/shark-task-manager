package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	before, err := ReviewSourceInventory(context.Background(), root)
	require.NoError(t, err)
	base := filepath.Join(root, DefaultWorktreeBaseDir)
	require.NoError(t, os.MkdirAll(base, 0700))
	first := filepath.Join(base, "first")
	git("worktree", "add", "-b", "shark-run-B001-1", first)
	path, err = RetainedBugWorktree(context.Background(), root, "B001")
	require.NoError(t, err)
	require.Equal(t, first, path)
	after, err := ReviewSourceInventory(context.Background(), root)
	require.NoError(t, err)
	require.Equal(t, before, after)
	link := filepath.Join(t.TempDir(), "project-link")
	if err := os.Symlink(root, link); err == nil {
		linkedInventory, err := ReviewSourceInventory(context.Background(), link)
		require.NoError(t, err)
		require.Equal(t, after, linkedInventory)
		_, file, _, ok := runtime.Caller(0)
		require.True(t, ok)
		scripts := filepath.Join(filepath.Dir(file), "..", "..", "skills", "shark-rider", "skills", "deep-review", "scripts")
		cmd := exec.Command("python3", "-c", "import source_inventory,sys; print(source_inventory.source_inventory(sys.argv[1]))", link)
		cmd.Env = append(os.Environ(), "PYTHONPATH="+scripts)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		require.Equal(t, linkedInventory, strings.TrimSpace(string(out)))
	}
	require.NoError(t, os.WriteFile(filepath.Join(base, "unrelated.txt"), []byte("must remain visible"), 0600))
	changed, err := ReviewSourceInventory(context.Background(), root)
	require.NoError(t, err)
	require.NotEqual(t, before, changed)
	require.NoError(t, os.Remove(filepath.Join(base, "unrelated.txt")))
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
