package commands

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jwwelbor/shark-task-manager/internal/runner"
)

func gitCheckout(t *testing.T, branch string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", branch},
		{"-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "base"},
	} {
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		require.NoError(t, err, string(out))
	}
	return dir
}

// A bug worker must be told which checkout and branch to edit, identically on
// the keyed-next and in-process run paths.
func TestBugDispatchNamesWorkerDirectoryOnBothPaths(t *testing.T) {
	ctx := context.Background()
	dir := gitCheckout(t, "bug/B123-fix")

	next, err := bugParentProjectInstructions(ctx, "prompt", dir)
	require.NoError(t, err)
	want := "Keep source work in the assigned worker working directory '" + dir + `' (branch "bug/B123-fix").`
	wantParentCmd := "For Shark record, related-document, and bundled skill/agent commands, run shark"
	assert.Contains(t, next, want)
	assert.Contains(t, next, wantParentCmd)

	run, err := dispatchPromptAssembler("bug").AssemblePrompt(ctx, runner.PromptAssemblyInput{Instruction: "prompt", WorkingDir: dir})
	require.NoError(t, err)
	assert.Contains(t, run, want)
	assert.Contains(t, run, wantParentCmd)
}

func TestBugDispatchSaysWhenNoWorkerDirectoryWasAssigned(t *testing.T) {
	ctx := context.Background()
	out, err := bugParentProjectInstructions(ctx, "prompt", "")
	require.NoError(t, err)
	assert.Contains(t, out, "No separate worker working directory was assigned")
	assert.NotContains(t, out, "assigned worker working directory '")

	projectRoot := gitCheckout(t, "bug/B124-root")
	t.Chdir(projectRoot)
	runRoot, err := dispatchPromptAssembler("bug").AssemblePrompt(ctx, runner.PromptAssemblyInput{Instruction: "prompt"})
	require.NoError(t, err)
	assert.Contains(t, runRoot, "No separate worker working directory was assigned")

	subDir := filepath.Join(projectRoot, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	t.Chdir(subDir)
	runInherited, err := dispatchPromptAssembler("bug").AssemblePrompt(ctx, runner.PromptAssemblyInput{Instruction: "prompt"})
	require.NoError(t, err)
	assert.Contains(t, runInherited, "Keep source work in the assigned worker working directory '"+subDir+`' (branch "bug/B124-root").`)
}

func TestGitBranchClauseReportsDetachedHead(t *testing.T) {
	dir := gitCheckout(t, "main")
	out, err := exec.Command("git", "-C", dir, "checkout", "-q", "--detach").CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Equal(t, " (detached HEAD)", gitBranchClause(context.Background(), dir))
	assert.Empty(t, gitBranchClause(context.Background(), t.TempDir()))
}
