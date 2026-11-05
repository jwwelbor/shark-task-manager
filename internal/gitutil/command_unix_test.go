//go:build !windows

package gitutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCommandCancelsGitAndCleanFilterChildren(t *testing.T) {
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
	git("add", "--", "source.txt")
	git("commit", "-m", "base")
	git("config", "filter.slow.clean", "exec /bin/sleep 5")
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("source.txt filter=slow\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "source.txt"), []byte("edit"), 0600))
	for _, args := range [][]string{{"status", "--porcelain=v1", "-z"}, {"hash-object", "--path", "source.txt", "--stdin"}, {"add", "--", "source.txt"}} {
		t.Run(args[0], func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			cmd := Command(ctx, append([]string{"-c", "core.fsmonitor=false", "-C", root}, args...)...)
			cmd.Stdin = strings.NewReader("edit")
			start := time.Now()
			_, err := cmd.Output()
			require.Error(t, err)
			require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
			require.Less(t, time.Since(start), 3*time.Second)
		})
	}
}

func TestCommandCancellationRetiresFilterDescendant(t *testing.T) {
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
	git("add", "source.txt")
	git("commit", "-m", "base")
	script := filepath.Join(root, "filter.sh")
	started, leaked := filepath.Join(root, "started"), filepath.Join(root, "leaked")
	// Git executes the script in the repository root; all names are fixed literals.
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nprintf started > started\nsleep 2\nprintf leaked > leaked\n"), 0700))
	git("config", "filter.slow.clean", "./filter.sh")
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("source.txt filter=slow\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "source.txt"), []byte("edit"), 0600))
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := Command(ctx, "-C", root, "status", "--porcelain").Output()
	require.Error(t, err)
	require.FileExists(t, started)
	require.Never(t, func() bool {
		_, err := os.Stat(leaked)
		return !os.IsNotExist(err)
	}, 2200*time.Millisecond, 20*time.Millisecond, "cancelled filter descendant continued running")
}
