//go:build !windows

package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCreateWorktreeCancelsBlockingSmudgeFilter(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	git("init", "-b", "main")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.invalid")
	require.NoError(t, os.WriteFile(filepath.Join(root, "source.txt"), []byte("base"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("source.txt filter=slow\n"), 0600))
	git("add", "--", "source.txt", ".gitattributes")
	git("commit", "-m", "base")
	git("config", "filter.slow.smudge", "exec /bin/sleep 5")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := NewGitWorktreeCreator().CreateWorktree(ctx, root, filepath.Join(t.TempDir(), "worker"), "worker-test")
	require.Error(t, err)
	require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
	require.Less(t, time.Since(start), 3*time.Second)
}
