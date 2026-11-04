package contracts

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/require"
)

type failAfterRealTransition struct {
	delegate gatepersist.Transitioner
	fail     bool
}

func initCC044Baseline(t *testing.T, project string) string {
	t.Helper()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", project, "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return strings.TrimSpace(string(out))
	}
	git("init", "-b", "main")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.invalid")
	git("commit", "--allow-empty", "-m", "baseline")
	return git("rev-parse", "HEAD")
}

func commitCC044Files(t *testing.T, project string, paths ...string) {
	t.Helper()
	for _, args := range [][]string{append([]string{"add", "--"}, paths...), {"-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "-m", "record project plan"}} {
		cmd := exec.Command("git", append([]string{"-C", project}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
}

func TestFeatureReworkPreservesExistingReasonPolicy(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/shark")
	build.Dir = root
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	project := t.TempDir()
	dbPath := filepath.Join(project, "fixture.db")
	shark := func(args ...string) {
		t.Helper()
		cmd := exec.Command(binary, append(args, "--db", dbPath, "--json")...)
		cmd.Dir = project
		data, e := cmd.CombinedOutput()
		require.NoError(t, e, string(data))
	}
	shark("admin", "init", "--non-interactive")
	shark("epic", "create", "Review regression")
	shark("feature", "create", "E01", "Planning regression")
	shark("status", "set", "E01-F01", "test_planning", "--force", "--reason", "fixture setup")
	shark("status", "set", "E01-F01", "specification")
	shark("status", "set", "E01-F01", "research")
}

// TestBugSuccessRoutesRequireAdmission exercises both public status commands;
// explicit force is used only to position each disposable fixture.
func TestBugSuccessRoutesRequireAdmission(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/shark")
	build.Dir = root
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	project := t.TempDir()
	dbPath := filepath.Join(project, "fixture.db")
	run := func(args ...string) ([]byte, error) {
		t.Helper()
		cmd := exec.Command(binary, append(args, "--db", dbPath, "--json")...)
		cmd.Dir = project
		return cmd.CombinedOutput()
	}
	for _, args := range [][]string{{"admin", "init", "--non-interactive"}, {"bug", "create", "Gate bypass fixture"}} {
		out, err = run(args...)
		require.NoError(t, err, string(out))
	}
	for _, route := range [][2]string{{"repair_plan", "plan_review"}, {"plan_review", "development"}, {"development", "code_review"}, {"code_review", "qa"}, {"qa", "completed"}} {
		t.Run(route[0], func(t *testing.T) {
			for _, advance := range [][]string{{"status", "set", "B001", route[1]}, {"status", "advance", "B001"}} {
				out, err = run("status", "set", "B001", route[0], "--force", "--reason", "fixture setup")
				require.NoError(t, err, string(out))
				beforeHistory, historyErr := run("history", "B001")
				require.NoError(t, historyErr, string(beforeHistory))
				out, err = run(advance...)
				require.Error(t, err, string(out))
				require.Contains(t, string(out), "validated gate result is required")
				status, statusErr := run("get", "B001")
				require.NoError(t, statusErr, string(status))
				var current struct {
					Status string `json:"status"`
				}
				require.NoError(t, json.Unmarshal(status, &current))
				require.Equal(t, route[0], current.Status)
				afterHistory, historyErr := run("history", "B001")
				require.NoError(t, historyErr, string(afterHistory))
				require.JSONEq(t, string(beforeHistory), string(afterHistory))
			}
		})
	}
	for _, route := range [][2]string{{"plan_review", "repair_plan"}, {"qa", "blocked"}} {
		out, err = run("status", "set", "B001", route[0], "--force", "--reason", "fixture setup")
		require.NoError(t, err, string(out))
		out, err = run("status", "set", "B001", route[1])
		require.NoError(t, err, string(out))
	}
}

func TestBugTriageCannotBypassConfiguredSuccessGate(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/shark")
	build.Dir = root
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	project := t.TempDir()
	dbPath := filepath.Join(project, "fixture.db")
	run := func(args ...string) ([]byte, error) {
		t.Helper()
		cmd := exec.Command(binary, append(args, "--db", dbPath, "--json")...)
		cmd.Dir = project
		return cmd.CombinedOutput()
	}
	out, err = run("admin", "init", "--non-interactive")
	require.NoError(t, err, string(out))
	source := filepath.Join(root, "internal", "sharkdata", "default_data", "workflow")
	workflowDir := filepath.Join(project, "workflow")
	require.NoError(t, os.Mkdir(workflowDir, 0700))
	entries, err := os.ReadDir(source)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(source, entry.Name()))
		require.NoError(t, readErr)
		if entry.Name() == "bug.yaml" {
			text := string(data)
			start, end := strings.Index(text, "  development:\n"), strings.Index(text, "  draft:\n")
			require.True(t, start >= 0 && end > start)
			triaged := strings.Replace(text[start:end], "  development:\n", "  triaged:\n", 1)
			text = strings.Replace(text, "  plan_review:\n", triaged+"  plan_review:\n", 1)
			text = strings.Replace(text, "      pass: development\n", "      pass: triaged\n", 1)
			data = []byte(text)
		}
		require.NoError(t, os.WriteFile(filepath.Join(workflowDir, entry.Name()), data, 0600))
	}
	configPath := filepath.Join(project, ".sharkconfig.json")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var config map[string]any
	require.NoError(t, json.Unmarshal(data, &config))
	config["workflow_config"] = workflowDir
	data, err = json.Marshal(config)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0600))
	out, err = run("bug", "create", "Configured triage gate")
	require.NoError(t, err, string(out))
	out, err = run("status", "set", "B001", "plan_review", "--force", "--reason", "fixture setup")
	require.NoError(t, err, string(out))
	out, err = run("bug", "triage", "B001", "--severity=high")
	require.Error(t, err, string(out))
	require.Contains(t, string(out), "validated gate result is required")
	out, err = run("get", "B001")
	require.NoError(t, err, string(out))
	var bug struct {
		Status   string `json:"status"`
		Severity string `json:"severity"`
	}
	require.NoError(t, json.Unmarshal(out, &bug))
	require.Equal(t, "plan_review", bug.Status)
	require.Equal(t, "medium", bug.Severity)
	workflowPath := filepath.Join(workflowDir, "bug.yaml")
	data, err = os.ReadFile(workflowPath)
	require.NoError(t, err)
	before, tail, ok := strings.Cut(string(data), "  plan_review:\n")
	require.True(t, ok)
	block, after, ok := strings.Cut(tail, "  research:\n")
	require.True(t, ok)
	block = strings.Replace(block, "      pass: triaged", "      pass: development", 1)
	block = strings.Replace(block, "      fail: repair_plan", "      fail: triaged", 1)
	require.NoError(t, os.WriteFile(workflowPath, []byte(before+"  plan_review:\n"+block+"  research:\n"+after), 0600))
	out, err = run("bug", "triage", "B001", "--severity=high")
	require.NoError(t, err, string(out))
	out, err = run("history", "B001")
	require.NoError(t, err, string(out))
	var history struct {
		Entries []struct {
			Old string `json:"old_status"`
			New string `json:"new_status"`
		} `json:"history"`
	}
	require.NoError(t, json.Unmarshal(out, &history))
	require.NotEmpty(t, history.Entries)
	require.Equal(t, "plan_review", history.Entries[0].Old)
	require.Equal(t, "triaged", history.Entries[0].New)
}

func (f *failAfterRealTransition) Transition(ctx context.Context, typ models.EntityType, key, target, reason, agent string, guard gatepersist.TransitionGuard) (string, bool, error) {
	from, changed, err := f.delegate.Transition(ctx, typ, key, target, reason, agent, guard)
	if err == nil && changed && f.fail {
		f.fail = false
		return "", false, errors.New("simulated crash after transition")
	}
	return from, changed, err
}

// TestBugReviewGateProductionPath uses the real CLI, SQLite database, claim,
// bundled workflow, report writer, and persisted bug history. The review
// artifacts are deliberately synthetic and do not constitute a real review.
func TestBugReviewGateProductionPath(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	scripts := filepath.Join(root, "skills", "shark-rider", "skills", "deep-review", "scripts")
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/shark")
	build.Dir = root
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	for _, useWorktree := range []bool{false, true} {
		t.Run(fmt.Sprintf("worktree-%t", useWorktree), func(t *testing.T) {
			project := t.TempDir()
			dbPath := filepath.Join(project, "fixture.db")
			run := func(program string, args ...string) []byte {
				t.Helper()
				cmd := exec.Command(program, args...)
				cmd.Dir = project
				output, e := cmd.CombinedOutput()
				require.NoError(t, e, string(output))
				return output
			}
			git := func(args ...string) string { t.Helper(); return strings.TrimSpace(string(run("git", args...))) }
			shark := func(args ...string) []byte { t.Helper(); return run(binary, append(args, "--db", dbPath, "--json")...) }
			git("init", "-b", "main")
			git("config", "user.name", "Test")
			git("config", "user.email", "test@example.invalid")
			git("config", "commit.gpgsign", "false")
			git("config", "core.hooksPath", t.TempDir())
			require.NoError(t, os.WriteFile(filepath.Join(project, ".gitignore"), []byte(".sharkconfig.json\nfixture.db*\ndocs/*\n!docs/plan/\ndocs/plan/*\n!docs/plan/bugs/\ndocs/plan/bugs/*\n!docs/plan/bugs/*.repair-plan.md\nshark-data/\n"), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(project, "app.txt"), []byte("base\n"), 0600))
			git("add", ".gitignore", "app.txt")
			git("commit", "-m", "baseline")
			baseline := git("rev-parse", "HEAD")
			shark("admin", "init", "--non-interactive")
			// The portable Go helper and descriptor-relative Python inventory must
			// compute the same bytes, including untracked content.
			untracked := filepath.Join(project, "inventory.txt")
			require.NoError(t, os.WriteFile(untracked, []byte("untracked fixture"), 0600))
			inventory := func(force bool, relativeProject string) string {
				t.Helper()
				cmd := exec.Command("python3", "-c", "import source_inventory,sys; print(source_inventory.source_inventory(sys.argv[1], sys.argv[2]))", project, relativeProject)
				cmd.Dir = project
				cmd.Env = append(os.Environ(), "PYTHONPATH="+scripts)
				if force {
					cmd.Env = append(cmd.Env, "SHARK_REVIEW_INVENTORY_FORCE=1", "SHARK_REVIEW_INVENTORY_BIN="+binary)
				}
				data, err := cmd.CombinedOutput()
				require.NoError(t, err, string(data))
				return strings.TrimSpace(string(data))
			}
			expectedInventory, err := runner.ReviewSourceInventory(context.Background(), project)
			require.NoError(t, err)
			require.Equal(t, expectedInventory, inventory(true, "."))
			if runtime.GOOS != "windows" {
				require.Equal(t, expectedInventory, inventory(false, "."))
			}
			nestedRuntime := filepath.Join(project, "nested", ".shark", "runs", "run")
			require.NoError(t, os.MkdirAll(nestedRuntime, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(nestedRuntime, "result"), []byte("runtime"), 0600))
			nestedInventory, err := runner.ReviewSourceInventoryForProject(context.Background(), project, "nested")
			require.NoError(t, err)
			require.Equal(t, nestedInventory, inventory(true, "nested"))
			if runtime.GOOS != "windows" {
				require.Equal(t, nestedInventory, inventory(false, "nested"))
			}
			require.NoError(t, os.RemoveAll(filepath.Join(project, "nested"))) // Entirely test-owned fixture.
			// Exercise the same portable read path used on hosts without Python no-follow opens.
			readCmd := exec.Command("python3", "-c", "import safe_review_io,sys; sys.stdout.buffer.write(safe_review_io.read_review_file(sys.argv[1], 64))", untracked)
			readCmd.Dir = project
			readCmd.Env = append(os.Environ(), "PYTHONPATH="+scripts, "SHARK_REVIEW_IO_FORCE=1", "SHARK_REVIEW_INVENTORY_BIN="+binary)
			readOutput, err := readCmd.CombinedOutput()
			require.NoError(t, err, string(readOutput))
			require.Equal(t, []byte("untracked fixture"), readOutput)
			require.NoError(t, os.Remove(untracked))
			shark("admin", "install-shark-data")
			workflowPath := filepath.Join(project, "shark-data", "workflow", "bug.yaml")
			workflowData, err := os.ReadFile(workflowPath)
			require.NoError(t, err)
			beforeReview, reviewTail, ok := strings.Cut(string(workflowData), "  code_review:\n")
			require.True(t, ok)
			reviewBlock, afterReview, ok := strings.Cut(reviewTail, "  completed:\n")
			require.True(t, ok)
			require.Contains(t, reviewBlock, "      pass: qa")
			require.Contains(t, reviewBlock, "      pass: success")
			reviewBlock = strings.Replace(reviewBlock, "      pass: qa", "      approved: qa", 1)
			reviewBlock = strings.Replace(reviewBlock, "      pass: success", "      approved: success", 1)
			require.NoError(t, os.WriteFile(workflowPath, []byte(beforeReview+"  code_review:\n"+reviewBlock+"  completed:\n"+afterReview), 0600))
			shark("bug", "create", "Admission integration")
			shark("status", "set", "B001", "plan_review", "--force", "--reason", "fixture setup")
			shark("claim", "B001", "--session", "fixture-session", "--by", "test")
			planRel := "docs/plan/bugs/B001.repair-plan.md"
			plan := []byte("# Plan\nBaseline: " + baseline + "\nPreserve cost and model behavior.\n")
			require.NoError(t, os.WriteFile(filepath.Join(project, planRel), plan, 0600))
			commitCC044Files(t, project, planRel)
			planHash := sha256.Sum256(plan)
			refs := []map[string]string{{"kind": "bug_repair_plan", "pointer": planRel, "summary": fmt.Sprintf("sha256:%x", planHash)}, {"kind": "bug_review_baseline", "pointer": baseline}}
			reportRel := "docs/review/B001.plan-review.md"
			report := []byte(fmt.Sprintf("plan_path: %s\nplan_sha256: sha256:%x\nIndependent review approved the plan.\n", planRel, planHash))
			require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(project, reportRel)), 0700))
			require.NoError(t, os.WriteFile(filepath.Join(project, reportRel), report, 0600))
			shark("related-docs", "add", "Plan review", reportRel, "--bug=B001")
			reportHash := sha256.Sum256(report)
			refs = append(refs, map[string]string{"kind": "plan_review_report", "pointer": reportRel, "summary": fmt.Sprintf("sha256:%x", reportHash)})
			resultPath := filepath.Join(project, ".shark", "runs", "fixture-result", "fixture-result.json")
			require.NoError(t, os.MkdirAll(filepath.Dir(resultPath), 0700))
			writeEnvelope := func(path string, evidence []map[string]string, outcome string) {
				t.Helper()
				data, e := json.Marshal(map[string]any{"kind": "final", "recommended_outcome": outcome, "evidence": evidence, "gate_result": map[string]any{"schema_version": 1, "summary": "fixture pass"}})
				require.NoError(t, e)
				require.NoError(t, os.WriteFile(path, data, 0600))
			}
			writeEnvelope(resultPath, refs, "pass")
			next := shark("next", "B001")
			require.Contains(t, string(next), "gate_result_v1")
			shark("run", "B001", "--apply-result", resultPath, "--run-id", "fixture-plan", "--session", "fixture-session")
			shark("run", "B001", "--apply-result", resultPath, "--run-id", "fixture-development", "--session", "fixture-session")
			require.Contains(t, string(shark("next", "B001")), "approved")
			require.NoError(t, os.WriteFile(filepath.Join(project, "app.txt"), []byte("fixed; cost and model preserved\n"), 0600))
			git("add", "app.txt")
			git("commit", "-m", "repair")
			worker := project
			if useWorktree {
				worker = filepath.Join(t.TempDir(), "worker")
				git("worktree", "add", "--detach", worker, "HEAD")
				require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(worker, planRel)), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(worker, planRel), plan, 0600))
			}
			runGate := func(args ...string) []byte {
				t.Helper()
				if useWorktree {
					args = append(args, "--workdir", worker)
				}
				return shark(args...)
			}
			capture := exec.Command("bash", filepath.Join(scripts, "get_diff.sh"))
			capture.Dir = worker
			capture.Env = append(os.Environ(), "REVIEW_BASE="+baseline, fmt.Sprintf("REVIEW_PLAN_SHA256=%x", planHash))
			scopeData, err := capture.Output()
			require.NoError(t, err)
			var scope struct {
				CandidateCommit string   `json:"candidate_commit"`
				CandidateTree   string   `json:"candidate_tree"`
				DiffSHA256      string   `json:"diff_sha256"`
				DiffPath        string   `json:"diff_path"`
				ChangedFiles    []string `json:"changed_files"`
			}
			require.NoError(t, json.Unmarshal(scopeData, &scope))
			fixtureDir := filepath.Join(worker, ".shark", "runs", "fixture")
			require.NoError(t, os.MkdirAll(fixtureDir, 0700))
			write := func(name string, data []byte) string {
				t.Helper()
				path := filepath.Join(fixtureDir, name)
				require.NoError(t, os.WriteFile(path, data, 0600))
				return path
			}
			scopePath := write("scope.json", scopeData)
			identity := map[string]string{"commit": scope.CandidateCommit, "tree": scope.CandidateTree, "diff_sha256": scope.DiffSHA256, "repair_plan_sha256": fmt.Sprintf("%x", planHash)}
			angles := map[string]any{}
			for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
				angles[name] = map[string]any{"candidate_identity": identity, "reviewed_files": scope.ChangedFiles, "findings": []any{}}
			}
			angleBytes, err := json.Marshal(angles)
			require.NoError(t, err)
			anglesPath := write("angles.json", angleBytes)
			decisions, err := json.Marshal(map[string]any{"candidate_identity": identity, "findings": []any{}})
			require.NoError(t, err)
			decisionsPath := write("decisions.json", decisions)
			bodyPath := write("body.md", []byte("Scope: reviewed all candidate files and callers. Findings: none. Cost and model behavior preserved. VERDICT: PASS\n"))
			reportPath := filepath.Join(fixtureDir, "report.md")
			run("python3", filepath.Join(scripts, "adaptive_review.py"), "write-report", "--project-root", worker, "--source-root", worker, "--body-file", bodyPath, "--review-output-path", reportPath, "--runner-mode", "dispatched-six-angle", "--specialists-completed", "6", "--consolidator-completed", "true", "--base-commit", baseline, "--diff-path", scope.DiffPath, "--scope-file", scopePath, "--angle-results", anglesPath, "--review-decisions", decisionsPath, "--verdict", "PASS")
			for kind, path := range map[string]string{"bug_review_scope": reportPath + ".scope.json", "bug_review_report": reportPath, "bug_review_angles": anglesPath, "bug_review_decisions": reportPath + ".decisions.json"} {
				data, e := os.ReadFile(path)
				require.NoError(t, e)
				rel, e := filepath.Rel(worker, path)
				require.NoError(t, e)
				hash := sha256.Sum256(data)
				refs = append(refs, map[string]string{"kind": kind, "pointer": rel, "summary": fmt.Sprintf("sha256:%x", hash)})
			}
			writeEnvelope(resultPath, refs, "approved")
			runGate("run", "B001", "--apply-result", resultPath, "--run-id", "fixture-review", "--session", "fixture-session")
			writeEnvelope(resultPath, refs, "pass")
			require.NoError(t, os.WriteFile(filepath.Join(worker, "app.txt"), []byte("stale\n"), 0600))
			staleArgs := []string{"run", "B001", "--apply-result", resultPath, "--run-id", "fixture-stale", "--session", "fixture-session", "--db", dbPath, "--json"}
			if useWorktree {
				staleArgs = append(staleArgs, "--workdir", worker)
			}
			stale := exec.Command(binary, staleArgs...)
			stale.Dir = project
			staleOutput, err := stale.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(staleOutput), "candidate worktree changed")
			require.NoError(t, os.WriteFile(filepath.Join(worker, "app.txt"), []byte("fixed; cost and model preserved\n"), 0600))
			runGate("run", "B001", "--apply-result", resultPath, "--run-id", "fixture-qa", "--session", "fixture-session")
			var result struct {
				Status string `json:"status"`
			}
			require.NoError(t, json.Unmarshal(shark("get", "B001"), &result))
			require.Equal(t, "completed", result.Status)
			history := string(shark("history", "B001"))
			require.Contains(t, history, "run fixture-plan operation")
			require.Contains(t, history, "outcome approved run fixture-review operation")
			require.Contains(t, history, "run fixture-qa operation")
		})
	}
}

// TestBugAdmissionReplayBindsActualHistory reproduces the competing-run
// interleaving using real SQLite notes, claims, status and entity history.
func TestBugAdmissionReplayBindsActualHistory(t *testing.T) {
	ctx := context.Background()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/shark")
	build.Dir = root
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	project := t.TempDir()
	baseline := initCC044Baseline(t, project)
	dbPath := filepath.Join(project, "fixture.db")
	shark := func(args ...string) {
		t.Helper()
		cmd := exec.Command(binary, append(args, "--db", dbPath, "--json")...)
		cmd.Dir = project
		data, e := cmd.CombinedOutput()
		require.NoError(t, e, string(data))
	}
	shark("admin", "init", "--non-interactive")
	shark("admin", "install-shark-data")
	workflowPath := filepath.Join(project, "shark-data", "workflow", "bug.yaml")
	workflowData, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	workflowData = []byte(strings.Replace(string(workflowData), "  plan_review:\n", "  plan_review:\n    aliases: [plan_review_old]\n", 1))
	require.NoError(t, os.WriteFile(workflowPath, workflowData, 0600))
	shark("bug", "create", "Replay provenance")
	shark("status", "set", "B001", "plan_review", "--force", "--reason", "fixture setup")
	shark("claim", "B001", "--session", "fixture-session", "--by", "test")
	sqlDB, err := db.InitDB(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	storage := repository.NewDB(sqlDB)
	registry := services.NewEntityRegistry()
	registry.Register(models.EntityTypeBug, services.NewBugRepositoryAdapter(repository.NewBugRepository(storage)))
	notes, err := services.NewNoteService(repository.NewEntityNoteRepository(storage), registry)
	require.NoError(t, err)
	historyRepo := repository.NewEntityHistoryRepository(storage)
	history := services.NewEntityHistoryService(historyRepo, registry)
	flow := workflow.NewService(project)
	entity := services.NewEntityService(flow)
	entity.SetHistoryRepo(historyRepo)
	transition := gatepersist.NewEntityServiceTransitioner(entity, registry, flow)
	claims := services.NewClaimService(claimrepo.NewRepository(storage), nil)
	coordinator := gatepersist.NewCoordinator(notes, notes, history, gatepersist.NewWorkflowStatusValidator(flow), transition, transition, claims, transition)
	coordinator.ClaimVerifier = claims
	coordinator.BeforeTransition = runner.PrepareBugApprovalSnapshot
	realAdmission := runner.NewBugGateAdmission(notes, history, flow.ForLevel("bug").NormalizeStatus)
	calls := map[string]int{}
	var callsMu sync.Mutex
	entered, proceed := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(proceed) }) }
	t.Cleanup(release)
	coordinator.Admission = func(ctx context.Context, req gatepersist.Request) error {
		callsMu.Lock()
		calls[req.RunID]++
		count := calls[req.RunID]
		callsMu.Unlock()
		if req.RunID == "run-c" && count == 1 {
			close(entered)
			<-proceed
		}
		if req.RunID == "run-b" && count == 2 {
			return errors.New("plan B changed")
		}
		return realAdmission(ctx, req)
	}
	makeEnvelope := func(name, content string) []byte {
		t.Helper()
		path := filepath.Join(project, "docs", "plan", "bugs", name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0600))
		commitCC044Files(t, project, filepath.ToSlash(filepath.Join("docs", "plan", "bugs", name)))
		hash := sha256.Sum256([]byte(content))
		data, e := json.Marshal(map[string]any{"kind": "final", "recommended_outcome": "pass", "evidence": []map[string]string{{"kind": "bug_repair_plan", "pointer": filepath.ToSlash(filepath.Join("docs", "plan", "bugs", name)), "summary": fmt.Sprintf("sha256:%x", hash)}, {"kind": "bug_review_baseline", "pointer": baseline}}, "gate_result": map[string]any{"schema_version": 1, "summary": content}})
		require.NoError(t, e)
		return data
	}
	a, b := makeEnvelope("plan-a.md", "plan A"), makeEnvelope("plan-b.md", "plan B")
	ingest := func(run string, data []byte) error {
		_, e := runner.IngestGateResult(ctx, runner.GateIngestRequest{EnvelopeBytes: data, Coordinator: coordinator, ProjectRoot: project, RunID: run, EntityKey: "B001", EntityType: models.EntityTypeBug, SourceStatus: "plan_review", Gate: "plan_review", Session: gatepersist.Session{ID: "fixture-session"}, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}, Outcomes: map[string]string{"pass": "development"}})
		return e
	}
	require.ErrorContains(t, ingest("run-b", b), "plan B changed")
	competing := make(chan error, 1)
	go func() { competing <- ingest("run-c", b) }()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		require.FailNow(t, "competing run did not reach admission")
	}
	require.NoError(t, ingest("run-a", a))
	release()
	require.ErrorContains(t, <-competing, "does not belong to this run")
	require.ErrorContains(t, ingest("run-b", b), "does not belong to this run")
	entries, err := history.GetHistory(ctx, models.EntityTypeBug, "B001")
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	require.Contains(t, *entries[0].Notes, "run run-a operation")
	coordinator.Admission = realAdmission
	_, err = runner.IngestGateResult(ctx, runner.GateIngestRequest{EnvelopeBytes: a, Coordinator: coordinator, ProjectRoot: project, RunID: "development-after-competition", EntityKey: "B001", EntityType: models.EntityTypeBug, SourceStatus: "development", Gate: "development", Session: gatepersist.Session{ID: "fixture-session"}, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}, Outcomes: map[string]string{"pass": "code_review"}})
	require.NoError(t, err, "an uncommitted later note must not hide the valid approval")

	// A crash after this run's own database transition may retire even if the
	// artifact later changes, because its exact transition is already durable.
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", project, "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
		data, e := cmd.CombinedOutput()
		require.NoError(t, e, string(data))
	}
	git("init", "-b", "main")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.invalid")
	require.NoError(t, os.WriteFile(filepath.Join(project, "tracked"), []byte("base"), 0600))
	git("add", "tracked")
	git("commit", "-m", "base")
	worker := filepath.Join(t.TempDir(), "worker")
	other := filepath.Join(t.TempDir(), "other")
	git("worktree", "add", "--detach", worker, "HEAD")
	git("worktree", "add", "--detach", other, "HEAD")
	shark("bug", "create", "Crash recovery")
	shark("status", "set", "B002", "plan_review", "--force", "--reason", "fixture setup")
	_, err = sqlDB.ExecContext(ctx, "UPDATE bugs SET status = ? WHERE key = ?", "plan_review_old", "B002")
	require.NoError(t, err)
	shark("claim", "B002", "--session", "fixture-session", "--by", "test")
	crashTransition := &failAfterRealTransition{delegate: transition, fail: true}
	crashCoordinator := gatepersist.NewCoordinator(notes, notes, history, gatepersist.NewWorkflowStatusValidator(flow), crashTransition, transition, claims, transition)
	crashCoordinator.ClaimVerifier = claims
	crashCoordinator.BeforeTransition = runner.PrepareBugApprovalSnapshot
	crashCoordinator.Admission = realAdmission
	crashResult := makeEnvelope("plan-crash.md", "crash plan")
	workerPlan := filepath.Join(worker, "docs", "plan", "bugs", "plan-crash.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(workerPlan), 0700))
	require.NoError(t, os.WriteFile(workerPlan, []byte("crash plan"), 0600))
	commitCC044Files(t, worker, "docs/plan/bugs/plan-crash.md")
	crashRequest := runner.GateIngestRequest{EnvelopeBytes: crashResult, Coordinator: crashCoordinator, ProjectRoot: project, WorkingDir: worker, RunID: "crash-run", EntityKey: "B002", EntityType: models.EntityTypeBug, SourceStatus: "plan_review", Gate: "plan_review", Session: gatepersist.Session{ID: "fixture-session"}, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}, Outcomes: map[string]string{"pass": "development"}}
	_, err = runner.IngestGateResult(ctx, crashRequest)
	require.ErrorContains(t, err, "simulated crash after transition")
	conflict := crashRequest
	conflict.WorkingDir = other
	_, err = runner.IngestGateResult(ctx, conflict)
	require.ErrorContains(t, err, "evidence root conflicts")
	require.NoError(t, os.Rename(worker, worker+"-missing"))
	crashRequest.WorkingDir = "" // Replay uses its durable identity, even after removal.
	crashCoordinator.Admission = func(context.Context, gatepersist.Request) error {
		return errors.New("evidence changed after transition")
	}
	resumed, err := runner.IngestGateResult(ctx, crashRequest)
	require.NoError(t, err)
	require.True(t, resumed.TransitionApplied)

	// Reconstruct a crash after create-once identity/result but before state.
	require.NoError(t, os.Rename(worker+"-missing", worker))
	shark("bug", "create", "Uninitialized replay")
	shark("status", "set", "B003", "plan_review", "--force", "--reason", "fixture setup")
	shark("claim", "B003", "--session", "fixture-session", "--by", "test")
	uninitialized := makeEnvelope("plan-uninitialized.md", "uninitialized plan")
	require.NoError(t, os.WriteFile(filepath.Join(worker, "docs", "plan", "bugs", "plan-uninitialized.md"), []byte("uninitialized plan"), 0600))
	commitCC044Files(t, worker, "docs/plan/bugs/plan-uninitialized.md")
	runDir, err := gaterun.RunDir(project, "uninitialized-run")
	require.NoError(t, err)
	digest, err := gaterun.ComputeOperationDigest("B003", "bug", "plan_review", "plan_review", uninitialized)
	require.NoError(t, err)
	planGit := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", worker}, args...)...)
		data, e := cmd.CombinedOutput()
		require.NoError(t, e, string(data))
		return strings.TrimSpace(string(data))
	}
	_, err = gaterun.CreateIdentity(runDir, gaterun.RunIdentity{RunID: "uninitialized-run", EntityKey: "B003", EntityType: "bug", SourceStatus: "plan_review", Gate: "plan_review", OperationDigest: digest, EvidenceRoot: worker, PlanRoot: worker, PlanCommit: planGit("rev-parse", "HEAD"), PlanBlob: planGit("rev-parse", "HEAD:docs/plan/bugs/plan-uninitialized.md")})
	require.NoError(t, err)
	_, err = gaterun.CreateResult(runDir, uninitialized)
	require.NoError(t, err)
	coordinator.Admission = realAdmission
	uninitializedRequest := runner.GateIngestRequest{EnvelopeBytes: uninitialized, Coordinator: coordinator, ProjectRoot: project, RunID: "uninitialized-run", EntityKey: "B003", EntityType: models.EntityTypeBug, SourceStatus: "plan_review", Gate: "plan_review", Session: gatepersist.Session{ID: "fixture-session"}, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}, Outcomes: map[string]string{"pass": "development"}}
	recovered, err := runner.IngestGateResult(ctx, uninitializedRequest)
	require.NoError(t, err)
	require.True(t, recovered.TransitionApplied)
}

func TestBugPlanApprovalRespectsConfiguredSuccessOutcome(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/shark")
	build.Dir = root
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	project := t.TempDir()
	baseline := initCC044Baseline(t, project)
	dbPath := filepath.Join(project, "fixture.db")
	shark := func(args ...string) []byte {
		t.Helper()
		cmd := exec.Command(binary, append(args, "--db", dbPath, "--json")...)
		cmd.Dir = project
		data, e := cmd.CombinedOutput()
		require.NoError(t, e, string(data))
		return data
	}
	shark("admin", "init", "--non-interactive")
	shark("admin", "install-shark-data")
	path := filepath.Join(project, "shark-data", "workflow", "bug.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	before, tail, ok := strings.Cut(string(data), "  plan_review:\n")
	require.True(t, ok)
	block, after, ok := strings.Cut(tail, "  research:\n")
	require.True(t, ok)
	require.Contains(t, block, "      pass: development")
	require.Contains(t, block, "      pass: success")
	block = strings.Replace(block, "      pass: development", "      approved: development", 1)
	block = strings.Replace(block, "      pass: success", "      approved: success", 1)
	block = "    aliases: [plan_review_old]\n" + block
	require.NoError(t, os.WriteFile(path, []byte(before+"  plan_review:\n"+block+"  research:\n"+after), 0600))
	shark("bug", "create", "Custom success outcome")
	shark("status", "set", "B001", "plan_review", "--force", "--reason", "fixture setup")
	fixtureDB, err := db.InitDB(dbPath)
	require.NoError(t, err)
	_, err = fixtureDB.ExecContext(context.Background(), "UPDATE bugs SET status = ? WHERE key = ?", "plan_review_old", "B001")
	require.NoError(t, err)
	require.NoError(t, fixtureDB.Close())
	shark("claim", "B001", "--session", "fixture-session", "--by", "test")
	dispatch := shark("next", "B001")
	require.Contains(t, string(dispatch), "approved")
	require.Contains(t, string(dispatch), "success")
	planRel := "docs/plan/bugs/B001.repair-plan.md"
	plan := []byte("# Approved fixture plan\n")
	require.NoError(t, os.WriteFile(filepath.Join(project, planRel), plan, 0600))
	commitCC044Files(t, project, planRel)
	hash := sha256.Sum256(plan)
	refs := []map[string]string{{"kind": "bug_repair_plan", "pointer": planRel, "summary": fmt.Sprintf("sha256:%x", hash)}, {"kind": "bug_review_baseline", "pointer": baseline}}
	reportRel := "docs/review/B001.plan-review.md"
	report := []byte(fmt.Sprintf("plan_path: %s\nplan_sha256: sha256:%x\nIndependent review approved the plan.\n", planRel, hash))
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(project, reportRel)), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(project, reportRel), report, 0600))
	shark("related-docs", "add", "Plan review", reportRel, "--bug=B001")
	reportHash := sha256.Sum256(report)
	refs = append(refs, map[string]string{"kind": "plan_review_report", "pointer": reportRel, "summary": fmt.Sprintf("sha256:%x", reportHash)})
	resultPath := filepath.Join(project, "result.json")
	write := func(outcome string) {
		t.Helper()
		envelope, e := json.Marshal(map[string]any{"kind": "final", "recommended_outcome": outcome, "evidence": refs, "gate_result": map[string]any{"schema_version": 1, "summary": "fixture"}})
		require.NoError(t, e)
		require.NoError(t, os.WriteFile(resultPath, envelope, 0600))
	}
	write("approved")
	shark("run", "B001", "--apply-result", resultPath, "--run-id", "custom-approval", "--session", "fixture-session")
	require.NoError(t, os.WriteFile(filepath.Join(project, planRel), []byte("unreviewed edit\n"), 0600))
	stale := exec.Command(binary, "next", "B001", "--db", dbPath, "--json")
	stale.Dir = project
	staleOutput, staleErr := stale.CombinedOutput()
	require.Error(t, staleErr)
	require.Contains(t, string(staleOutput), "approved project plan changed")
	require.NoError(t, os.WriteFile(filepath.Join(project, planRel), plan, 0600))
	var downstream struct {
		Prompt       string `json:"prompt"`
		PromptSHA256 string `json:"prompt_sha256"`
	}
	downstreamJSON := shark("next", "B001")
	require.NoError(t, json.Unmarshal(downstreamJSON, &downstream), string(downstreamJSON))
	require.Contains(t, downstream.Prompt, string(plan))
	require.NotContains(t, downstream.Prompt, "unreviewed edit")
	promptHash := sha256.Sum256([]byte(downstream.Prompt))
	require.Equal(t, fmt.Sprintf("%x", promptHash), downstream.PromptSHA256)
	write("pass")
	shark("run", "B001", "--apply-result", resultPath, "--run-id", "custom-development", "--session", "fixture-session")
	var result struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(shark("get", "B001"), &result))
	require.Equal(t, "code_review", result.Status)
	require.Contains(t, string(shark("history", "B001")), "outcome approved run custom-approval")
	require.Contains(t, string(shark("history", "B001")), "plan_review_old")
}
