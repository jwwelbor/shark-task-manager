package contracts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/stretchr/testify/require"
)

// TestBugWorktreeDispatchRetainsEvidence runs the real CLI and Git worktree
// path with a deterministic local worker. It does not claim provider coverage.
func TestBugWorktreeDispatchRetainsEvidence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture worker uses a POSIX shell")
	}
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/shark")
	build.Dir = repo
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	for _, relativeProject := range []string{".", "nested/project"} {
		t.Run(relativeProject, func(t *testing.T) {
			testBugWorktreeDispatch(t, binary, relativeProject)
		})
	}
}

func testBugWorktreeDispatch(t *testing.T, binary, relativeProject string) {
	t.Helper()
	repository := t.TempDir()
	project := filepath.Join(repository, relativeProject)
	require.NoError(t, os.MkdirAll(project, 0700))
	runIn := func(dir, name string, args ...string) string {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		data, e := cmd.CombinedOutput()
		require.NoError(t, e, string(data))
		return strings.TrimSpace(string(data))
	}
	run := func(name string, args ...string) string {
		t.Helper()
		return runIn(project, name, args...)
	}
	git := func(args ...string) string { t.Helper(); return runIn(repository, "git", args...) }
	git("init", "-b", "main")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.invalid")
	git("config", "commit.gpgsign", "false")
	git("config", "core.hooksPath", t.TempDir())
	require.NoError(t, os.WriteFile(filepath.Join(project, "app"), []byte("base"), 0600))
	git("add", filepath.Join(relativeProject, "app"))
	git("commit", "-m", "base")
	baseline := git("rev-parse", "HEAD")
	dbPath := filepath.Join(project, "fixture.db")
	shark := func(args ...string) string {
		t.Helper()
		return run(binary, append(args, "--db", dbPath, "--json")...)
	}
	shark("admin", "init", "--non-interactive")
	shark("bug", "create", "Worktree dispatch fixture")
	shark("status", "set", "B001", "plan_review", "--force", "--reason", "fixture setup")
	plan := []byte("fixture repair plan\n")
	hash := sha256.Sum256(plan)
	approve, err := json.Marshal(map[string]any{
		"kind": "final", "recommended_outcome": "pass",
		"evidence":    []map[string]string{{"kind": "bug_repair_plan", "pointer": ".shark/runs/worker-fixture/plan.md", "summary": fmt.Sprintf("sha256:%x", hash)}, {"kind": "bug_review_baseline", "pointer": baseline}},
		"gate_result": map[string]any{"schema_version": 1, "summary": "Fixture approval"},
	})
	require.NoError(t, err)
	blocked := `{"kind":"final","recommended_outcome":"blocked","evidence":[],"gate_result":{"schema_version":1,"summary":"Fixture stops after approval","no_kickback_reason":"Fixture stops after approval"}}`
	script := fmt.Sprintf("#!/bin/sh\nset -eu\nfound=0\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = --output-format ]; then shift; [ \"$1\" = text ]; found=1; break; fi\n  shift\ndone\n[ \"$found\" = 1 ]\nplan=.shark/runs/worker-fixture/plan.md\nif [ -f \"$plan\" ]; then\n  printf '%%s\\n' '%s'\n  exit 0\nfi\nmkdir -p .shark/runs/worker-fixture\nprintf 'fixture repair plan\\n' > \"$plan\"\nprintf '%%s\\n' '%s'\n", blocked, approve)
	fakeDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(fakeDir, "claude"), []byte(script), 0700))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "run", "B001", "--worktree", "--db", dbPath, "--json")
	cmd.Dir = project
	cmd.Env = append(os.Environ(), "PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	require.NoError(t, err, stderr.String())
	var result struct {
		Outcome     string `json:"outcome"`
		FinalStatus string `json:"final_status"`
		Stages      []struct {
			Status     string `json:"status"`
			GateStatus struct {
				RunID string `json:"run_id"`
			} `json:"gate_status"`
		} `json:"stages"`
	}
	require.NoError(t, json.Unmarshal(out, &result), string(out))
	require.Equal(t, "paused", result.Outcome)
	require.Equal(t, "blocked", result.FinalStatus)
	require.GreaterOrEqual(t, len(result.Stages), 2)
	require.Equal(t, "plan_review", result.Stages[0].Status)
	runDir, err := gaterun.RunDir(project, result.Stages[0].GateStatus.RunID)
	require.NoError(t, err)
	identity, exists, err := gaterun.ReadIdentity(runDir)
	require.NoError(t, err)
	require.True(t, exists)
	require.NotEmpty(t, identity.EvidenceRoot)
	retainedPlan, err := os.ReadFile(filepath.Join(identity.EvidenceRoot, ".shark", "runs", "worker-fixture", "plan.md"))
	require.NoError(t, err)
	require.Equal(t, plan, retainedPlan)
}
