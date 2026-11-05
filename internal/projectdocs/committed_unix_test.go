//go:build !windows

package projectdocs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVerifyCommittedCancelsBlockedGit(t *testing.T) {
	tools := t.TempDir()
	git := filepath.Join(tools, "git")
	require.NoError(t, os.WriteFile(git, []byte("#!/bin/sh\nexec /bin/sleep 30\n"), 0700))
	t.Setenv("PATH", tools)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := VerifyCommitted(ctx, t.TempDir(), "docs/plan/report.md", []byte("report"))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), 3*time.Second)
}

func TestReadBoundedRejectsOversizedFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "report.md")
	require.NoError(t, os.WriteFile(path, []byte("123456789"), 0600))
	_, err := ReadBounded(root, "report.md", 8)
	require.ErrorContains(t, err, "bounded regular file")
}

func TestVerifyCommittedCancelsBlockedCleanFilter(t *testing.T) {
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
	path := "docs/plan/report.md"
	require.NoError(t, os.MkdirAll(filepath.Join(root, "docs", "plan"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, path), []byte("report"), 0600))
	git("add", "--", path)
	git("commit", "-m", "Commit report")
	git("config", "filter.slow.clean", "exec /bin/sleep 5")
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("docs/plan/report.md filter=slow\n"), 0600))
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := VerifyCommitted(ctx, root, path, []byte("report"))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), 3*time.Second)
}
