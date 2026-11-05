package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func commitTestBugPlan(t *testing.T, root, path string, plan []byte) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); os.IsNotExist(err) {
		run("init")
		run("config", "user.name", "Test")
		run("config", "user.email", "test@example.invalid")
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, path), plan, 0600))
	run("add", "--", path)
	run("commit", "-m", "record plan")
}
