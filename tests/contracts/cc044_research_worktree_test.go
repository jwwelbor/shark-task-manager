package contracts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBugResearchWorktreeUsesParentRecordsAndWorkerReport(t *testing.T) {
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
	project := t.TempDir()
	initCC044Baseline(t, project)
	db := filepath.Join(project, "fixture.db")
	shark := func(args ...string) []byte {
		t.Helper()
		cmd := exec.Command(binary, append(args, "--db", db, "--json")...)
		cmd.Dir = project
		data, err := cmd.CombinedOutput()
		require.NoError(t, err, string(data))
		return data
	}
	shark("admin", "init", "--non-interactive")
	shark("bug", "create", "Research in a worktree")
	shark("status", "set", "B001", "research", "--force", "--reason", "fixture setup")
	blocked := `{"kind":"final","recommended_outcome":"blocked","evidence":[],"gate_result":{"schema_version":1,"summary":"Fixture stops after research","no_kickback_reason":"Fixture stops after research"}}`
	script := fmt.Sprintf("#!/bin/sh\nset -eu\nreport=docs/plan/bugs/B001.research-report.md\nif [ -f \"$report\" ]; then printf '%%s\\n' '%s'; exit 0; fi\nmkdir -p docs/plan/bugs\ncat > \"$report\" <<'REPORT'\n---\nresearch_schema: 2\nrigor: simple\ncategories: [backend]\nrelated_work: false\n---\n# Research report\n## Scope\nBug diagnosis.\n## Research checklist\n- [x] `scope_vocabulary` — Evidence: `app`.\n- [x] `affected_implementation_or_contract` — Evidence: `app`.\n## Findings\nThe cause is isolated.\n## Decisions\nRepair the existing boundary.\n## Sources\n- `app`\nREPORT\ngit add \"$report\"\ngit -c commit.gpgsign=false -c core.hooksPath=/dev/null commit -m 'Record research' >/dev/null\n(cd %q && %q --db %q related-docs add 'Research report' docs/plan/bugs/B001.research-report.md --bug=B001 >/dev/null)\nprintf 'RECOMMENDED OUTCOME: pass\\n'\n", blocked, project, binary, db)
	fakeDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(fakeDir, "claude"), []byte(script), 0700))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "run", "B001", "--worktree", "--db", db, "--json")
	cmd.Dir = project
	cmd.Env = append(os.Environ(), "PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err = cmd.Output()
	require.NoError(t, err, stderr.String())
	var result struct {
		FinalStatus string `json:"final_status"`
		Stages      []struct {
			Status string `json:"status"`
		} `json:"stages"`
	}
	require.NoError(t, json.Unmarshal(out, &result), string(out))
	require.Equal(t, "blocked", result.FinalStatus, stderr.String())
	require.GreaterOrEqual(t, len(result.Stages), 2)
	require.Equal(t, "research", result.Stages[0].Status)
	require.Contains(t, string(shark("related-docs", "list", "--bug=B001")), "B001.research-report.md")
	list := func() []byte {
		t.Helper()
		cmd := exec.Command("git", "-C", project, "worktree", "list", "--porcelain")
		data, err := cmd.Output()
		require.NoError(t, err)
		return data
	}
	before := list()
	shark("run", "B001", "--worktree")
	require.Equal(t, before, list(), "resume must reuse the retained bug checkout")
}

func TestBugRejectsIncompatibleResearchAndRenamedGateBeforeAdvancement(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/shark")
	build.Dir = repo
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	project := t.TempDir()
	initCC044Baseline(t, project)
	db := filepath.Join(project, "fixture.db")
	run := func(args ...string) ([]byte, error) {
		t.Helper()
		cmd := exec.Command(binary, append(args, "--db", db, "--json")...)
		cmd.Dir = project
		return cmd.CombinedOutput()
	}
	for _, args := range [][]string{{"admin", "init", "--non-interactive"}, {"admin", "install-shark-data"}, {"bug", "create", "Workflow guard"}, {"status", "set", "B001", "research", "--force", "--reason", "setup"}} {
		out, err = run(args...)
		require.NoError(t, err, string(out))
	}
	path := filepath.Join(project, "shark-data", "workflow", "bug.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	before, tail, ok := strings.Cut(string(data), "  research:\n")
	require.True(t, ok)
	changed := strings.Replace(tail, "      pass: repair_plan", "      approved: repair_plan", 1)
	require.NotEqual(t, tail, changed)
	require.NoError(t, os.WriteFile(path, []byte(before+"  research:\n"+changed), 0600))
	out, err = run("status", "advance", "B001", "--outcome", "approved")
	require.Error(t, err)
	require.Contains(t, string(out), "legacy bug research has no pass")
	status, err := run("get", "B001")
	require.NoError(t, err)
	var bug struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(status, &bug))
	require.Equal(t, "research", bug.Status)
	aliased := strings.Replace(string(data), "  plan_review:\n", "  bespoke_plan_review:\n    aliases: [plan_review]\n", 1)
	require.NoError(t, os.WriteFile(path, []byte(aliased), 0600))
	out, err = run("next", "B001")
	require.Error(t, err)
	require.Contains(t, string(out), "evidence gate")
	upper := strings.Replace(string(data), "  plan_review:\n", "  PLAN_REVIEW:\n", 1)
	require.NotEqual(t, string(data), upper)
	require.NoError(t, os.WriteFile(path, []byte(upper), 0600))
	out, err = run("next", "B001")
	require.Error(t, err)
	require.Contains(t, string(out), "unsupported capitalization")
}

func TestBugResearchRiderAdvanceUsesWorkerWorkdir(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/shark")
	build.Dir = repo
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	project := t.TempDir()
	initCC044Baseline(t, project)
	db := filepath.Join(project, "fixture.db")
	shark := func(args ...string) []byte {
		t.Helper()
		cmd := exec.Command(binary, append(args, "--db", db, "--json")...)
		cmd.Dir = project
		data, err := cmd.CombinedOutput()
		require.NoError(t, err, string(data))
		return data
	}
	shark("admin", "init", "--non-interactive")
	shark("bug", "create", "Rider research")
	shark("status", "set", "B001", "research", "--force", "--reason", "fixture setup")
	nested := filepath.Join(project, "nested")
	require.NoError(t, os.Mkdir(nested, 0700))
	for _, action := range []string{"next", "run"} {
		cmd := exec.Command(binary, action, "B001", "--db", db, "--json")
		cmd.Dir = nested
		output, err := cmd.CombinedOutput()
		require.Error(t, err)
		require.Contains(t, string(output), "corresponding Shark project root")
	}
	worker := filepath.Join(t.TempDir(), "worker")
	git := exec.Command("git", "-C", project, "worktree", "add", "--detach", worker, "HEAD")
	out, err = git.CombinedOutput()
	require.NoError(t, err, string(out))
	conflict := exec.Command(binary, "run", "B001", "--worktree", "--workdir", worker, "--db", db, "--json")
	conflict.Dir = project
	conflictOut, conflictErr := conflict.CombinedOutput()
	require.Error(t, conflictErr)
	require.Contains(t, string(conflictOut), "cannot be combined")
	shark("next", "B001", "--workdir", worker)
	foreign := t.TempDir()
	bad := exec.Command(binary, "next", "B001", "--workdir", foreign, "--db", db, "--json")
	bad.Dir = project
	out, err = bad.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), "cannot be bound")
	report := "---\nresearch_schema: 2\nrigor: simple\ncategories: [backend]\nrelated_work: false\n---\n# Research report\n## Scope\nDiagnose the bug.\n## Research checklist\n- [x] `scope_vocabulary` — Evidence: `app`.\n- [x] `affected_implementation_or_contract` — Evidence: `app`.\n## Findings\nCause located.\n## Decisions\nRepair the boundary.\n## Sources\n- `app`\n"
	path := filepath.Join(worker, "docs", "plan", "bugs", "B001.research-report.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, []byte(report), 0600))
	for _, args := range [][]string{{"add", "docs/plan/bugs/B001.research-report.md"}, {"-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "-m", "Record research"}} {
		cmd := exec.Command("git", append([]string{"-C", worker}, args...)...)
		out, err = cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	shark("related-docs", "add", "Research report", "docs/plan/bugs/B001.research-report.md", "--bug=B001")
	shark("status", "advance", "B001", "--outcome", "pass", "--workdir", worker)
	var bug struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(shark("get", "B001"), &bug))
	require.Equal(t, "repair_plan", bug.Status)
}

func TestBugSelfReworkAliasStopsAndRetires(t *testing.T) {
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
	project := t.TempDir()
	initCC044Baseline(t, project)
	db := filepath.Join(project, "fixture.db")
	shark := func(args ...string) {
		t.Helper()
		cmd := exec.Command(binary, append(args, "--db", db, "--json")...)
		cmd.Dir = project
		data, err := cmd.CombinedOutput()
		require.NoError(t, err, string(data))
	}
	shark("admin", "init", "--non-interactive")
	shark("admin", "install-shark-data")
	path := filepath.Join(project, "shark-data", "workflow", "bug.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	before, rest, ok := strings.Cut(string(data), "  repair_plan:\n")
	require.True(t, ok)
	block, after, ok := strings.Cut(rest, "  research:\n")
	require.True(t, ok)
	block = "    aliases: [repair_plan_old]\n" + strings.Replace(block, "      fail: repair_plan\n", "      fail: repair_plan_old\n", 1)
	require.NoError(t, os.WriteFile(path, []byte(before+"  repair_plan:\n"+block+"  research:\n"+after), 0600))
	shark("bug", "create", "Self route alias")
	shark("status", "set", "B001", "repair_plan", "--force", "--reason", "setup")
	fake := t.TempDir()
	script := "#!/bin/sh\nprintf '%s\\n' '{\"kind\":\"final\",\"recommended_outcome\":\"fail\",\"evidence\":[],\"gate_result\":{\"schema_version\":1,\"summary\":\"retry\"}}'\n"
	require.NoError(t, os.WriteFile(filepath.Join(fake, "claude"), []byte(script), 0700))
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "run", "B001", "--db", db, "--json")
	cmd.Dir = project
	cmd.Env = append(os.Environ(), "PATH="+fake+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err = cmd.Output()
	require.NoError(t, err, stderr.String())
	var result struct {
		Outcome     string `json:"outcome"`
		FinalStatus string `json:"final_status"`
		Stages      []struct {
			GateStatus struct {
				RetirementState string `json:"retirement_state"`
			} `json:"gate_status"`
		} `json:"stages"`
	}
	require.NoError(t, json.Unmarshal(out, &result), string(out))
	require.Equal(t, "paused", result.Outcome)
	require.Equal(t, "repair_plan", result.FinalStatus)
	require.Len(t, result.Stages, 1)
	require.NotEqual(t, "pending", result.Stages[0].GateStatus.RetirementState)
}

func TestNonBugWorktreeAndWorkdirRetainsExistingBehavior(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/shark")
	build.Dir = repo
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	project := t.TempDir()
	initCC044Baseline(t, project)
	db := filepath.Join(project, "fixture.db")
	for _, args := range [][]string{{"admin", "init", "--non-interactive"}, {"epic", "create", "Feature test"}, {"feature", "create", "E01", "Feature test"}, {"run", "E01-F01", "--dry-run", "--worktree", "--workdir", project}} {
		cmd := exec.Command(binary, append(args, "--db", db, "--json")...)
		cmd.Dir = project
		out, err = cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
}
