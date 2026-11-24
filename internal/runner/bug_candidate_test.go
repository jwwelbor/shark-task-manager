package runner

import (
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

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

func TestBugEvidenceLimitsDistinguishPlansAndReviewArtifacts(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "evidence")
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, file.Truncate(2*1024*1024+1))
	require.NoError(t, file.Close())
	_, err = readBoundedBugEvidence(root, "evidence", bugEvidenceLimit("bug_repair_plan"))
	require.Error(t, err)
	for _, kind := range []string{"bug_review_scope", "bug_review_report", "bug_review_decisions", "bug_review_angles", "bug_review_material_impacts", "bug_review_material_dispositions"} {
		_, err = readBoundedBugEvidence(root, "evidence", bugEvidenceLimit(kind))
		require.NoError(t, err, kind)
	}
	require.NoError(t, os.Truncate(path, 128*1024*1024+1))
	_, err = readBoundedBugEvidence(root, "evidence", bugEvidenceLimit("bug_review_report"))
	require.Error(t, err)
}

func TestBugCandidateScopePreservesGitNamesBeforeUnicodeDecoding(t *testing.T) {
	// The same path tokens occur in the candidate, successor delta and snapshot marker.
	const names = `["caf\u00e9","caf\udce9","caf\ufffd","caf\\udce9","caf\ud800\udce9"]`
	expected := []string{"café", "caf\xe9", "caf\uFFFD", `caf\udce9`, "caf\U000100E9"}
	var scope bugCandidateScope
	data := []byte(`{"changed_files":` + names + `,"source_snapshot":{"selected_paths":` + names + `}}`)
	require.NoError(t, json.Unmarshal(data, &scope))
	require.Equal(t, expected, []string(scope.ChangedFiles))
	require.Equal(t, expected, []string(scope.Source.SelectedPaths))
	for _, invalid := range []string{`"\ud800"`, `"\udc01"`, `"\u0000"`, `""`, `"../outside"`, string([]byte{'"', 0xe9, '"'})} {
		require.Error(t, json.Unmarshal([]byte(`{"changed_files":[`+invalid+`]}`), &scope), "%q", invalid)
	}
}

func TestBugCandidateScopeTracksSourceChanges(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return string(out)
	}
	git("init", "-b", "main")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.com")
	git("config", "commit.gpgsign", "false")
	git("config", "core.hooksPath", t.TempDir())
	require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("base\n"), 0600))
	git("add", "change.txt")
	git("commit", "-m", "base")
	base := strings.TrimSpace(git("rev-parse", "HEAD"))
	require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("unaccepted predecessor\n"), 0600))
	git("add", "change.txt")
	require.NoError(t, os.WriteFile(filepath.Join(root, "leaf.txt"), []byte("independent leaf"), 0600))
	git("add", "leaf.txt")
	git("commit", "-m", "unaccepted predecessor")
	laterBase := strings.TrimSpace(git("rev-parse", "HEAD"))
	require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("candidate\n"), 0600))
	plan := []byte("approved plan\n")
	commitTestBugPlan(t, root, "docs/plan/plan.md", plan)
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	scripts := filepath.Join(filepath.Dir(file), "..", "..", "skills", "shark-rider", "skills", "deep-review", "scripts")
	parent := t.TempDir()
	snapshot := filepath.Join(parent, "snapshot")
	metadata := filepath.Join(parent, "source.json")
	cmd := exec.Command("python3", filepath.Join(scripts, "prepare_candidate.py"), "--project-root", root, "--destination", snapshot, "--metadata-file", metadata, "change.txt")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	cmd = exec.Command("bash", filepath.Join(scripts, "get_diff.sh"))
	cmd.Dir = snapshot
	cmd.Env = append(os.Environ(), "REVIEW_BASE="+base, "REVIEW_PREDECESSOR_COMMIT="+laterBase, "REVIEW_SOURCE_METADATA="+metadata)
	scope, err := cmd.Output()
	require.NoError(t, err)
	require.NoError(t, verifyBugCandidate(context.Background(), root, scope))
	for _, flag := range []string{"assume-unchanged", "skip-worktree"} {
		git("update-index", "--"+flag, "--", "change.txt")
		require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("hidden change\n"), 0600))
		require.ErrorContains(t, verifyBugCandidate(context.Background(), root, scope), "assume-unchanged or skip-worktree")
		git("update-index", "--no-"+flag, "--", "change.txt")
		require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("candidate\n"), 0600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs/plan/plan.md"), []byte("changed plan"), 0600))
	require.ErrorContains(t, verifyBugCandidate(context.Background(), root, scope), "source change inventory changed")
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs/plan/plan.md"), plan, 0600))
	runArtifacts := filepath.Join(root, ".shark", "runs", "new-run")
	require.NoError(t, os.MkdirAll(runArtifacts, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(runArtifacts, "result.json"), []byte("new run"), 0600))
	require.NoError(t, verifyBugCandidate(context.Background(), root, scope))
	require.NoError(t, os.WriteFile(filepath.Join(snapshot, "untracked"), []byte("changed snapshot"), 0600))
	require.ErrorContains(t, verifyBugCandidate(context.Background(), root, scope), "snapshot worktree changed")
	require.NoError(t, os.Remove(filepath.Join(snapshot, "untracked")))
	require.NoError(t, os.Rename(snapshot, snapshot+"-missing"))
	require.Error(t, verifyBugCandidate(context.Background(), root, scope))
	require.NoError(t, os.Rename(snapshot+"-missing", snapshot))
	if runtime.GOOS != "windows" {
		gitDirCmd := exec.Command("git", "-C", snapshot, "rev-parse", "--absolute-git-dir")
		gitDir, err := gitDirCmd.Output()
		require.NoError(t, err)
		marker := filepath.Join(strings.TrimSpace(string(gitDir)), "shark-review-source.json")
		require.NoError(t, os.Rename(marker, marker+".saved"))
		fifo := exec.Command("mkfifo", marker)
		require.NoError(t, fifo.Run())
		done := make(chan error, 1)
		go func() { done <- verifyBugCandidate(context.Background(), root, scope) }()
		select {
		case err := <-done:
			require.ErrorContains(t, err, "snapshot provenance marker")
		case <-time.After(time.Second):
			require.FailNow(t, "snapshot marker read blocked on FIFO")
		}
		require.NoError(t, os.Remove(marker))
		require.NoError(t, os.Rename(marker+".saved", marker))
	}

	// Drive the same scope through the production ingestion coordinator. Store
	// test artifacts as ignored files so they do not alter the candidate.
	exclude := filepath.Join(root, ".git", "info", "exclude")
	f, err := os.OpenFile(exclude, os.O_APPEND|os.O_WRONLY, 0)
	require.NoError(t, err)
	_, err = f.WriteString("\n.shark/\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".shark"), 0700))
	planHash := sha256.Sum256(plan)
	bindPlan := func(data []byte) []byte {
		var doc map[string]any
		require.NoError(t, json.Unmarshal(data, &doc))
		doc["repair_plan_sha256"] = fmt.Sprintf("%x", planHash)
		bound, err := json.Marshal(doc)
		require.NoError(t, err)
		return bound
	}
	scope = bindPlan(scope)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".shark", "scope.json"), scope, 0600))
	planRef := fmt.Sprintf(`{"kind":"bug_repair_plan","pointer":"docs/plan/plan.md","summary":"sha256:%x"},{"kind":"bug_review_baseline","pointer":"%s"}`, planHash, base)
	cmd = exec.Command("bash", filepath.Join(scripts, "get_diff.sh"))
	cmd.Dir = snapshot
	cmd.Env = append(os.Environ(), "REVIEW_BASE="+laterBase, "REVIEW_SOURCE_METADATA="+metadata)
	narrowScope, err := cmd.Output()
	require.NoError(t, err)
	narrowScope = bindPlan(narrowScope)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".shark", "narrow-scope.json"), narrowScope, 0600))
	narrowHash := sha256.Sum256(narrowScope)
	narrowRef := fmt.Sprintf(`{"kind":"bug_review_scope","pointer":".shark/narrow-scope.json","summary":"sha256:%x"}`, narrowHash)

	var captured bugCandidateScope
	require.NoError(t, json.Unmarshal(scope, &captured))
	var rawScope map[string]any
	require.NoError(t, json.Unmarshal(scope, &rawScope))
	envelope := func(refs string) []byte {
		return []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[%s],"gate_result":{"schema_version":1,"summary":"approved"}}`, refs))
	}
	// Challenge the parent reader directly with a newly hash-bound scope. The
	// full baseline manifest, Git objects, plan identity and snapshot stay valid.
	delta, ok := rawScope["successor_delta"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, laterBase, delta["predecessor_commit"])
	git("tag", "-a", "predecessor-alias", "-m", "same commit, different object", laterBase)
	tagObject := strings.TrimSpace(git("rev-parse", "predecessor-alias"))
	for field, changed := range map[string]any{
		"changed_files": []string{}, "predecessor_commit": tagObject,
		"predecessor_tree": captured.CandidateTree, "diff_sha256": strings.Repeat("0", 64),
	} {
		t.Run("successor_delta_"+field, func(t *testing.T) {
			original := delta[field]
			delta[field] = changed
			altered, err := json.Marshal(rawScope)
			delta[field] = original
			require.NoError(t, err)
			path := ".shark/delta-" + field + ".json"
			require.NoError(t, os.WriteFile(filepath.Join(root, path), altered, 0600))
			ref := fmt.Sprintf(`{"kind":"bug_review_scope","pointer":%q,"summary":"sha256:%x"}`, path, sha256.Sum256(altered))
			_, err = readBugReviewScope(context.Background(), gatepersist.Request{
				RunDir: filepath.Join(root, ".shark", "runs", "delta-"+field), EnvelopeJSON: envelope(planRef + "," + ref),
			})
			require.ErrorContains(t, err, "successor delta")
		})
	}
	for _, invalid := range []any{nil, map[string]any{"unexpected": "member"}} {
		rawScope["successor_delta"] = invalid
		challenge, err := json.Marshal(rawScope)
		require.NoError(t, err)
		require.ErrorContains(t, verifyBugCandidate(context.Background(), root, challenge), "successor delta")
	}
	rawScope["successor_delta"] = delta
	// Execute the native Workflow with deterministic specialist stubs, then
	// feed its actual result through the Python writer and Go admission.
	nativeHarness := `
const fs = require('fs');
const src = fs.readFileSync(process.argv[1], 'utf8');
const scope = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
const run = new (Object.getPrototypeOf(async function(){}).constructor)('args','phase','parallel','agent','log', src.slice(src.indexOf('// ─── Args')));
const identity = {commit:scope.candidate_commit, tree:scope.candidate_tree, diff_sha256:scope.diff_sha256, repair_plan_sha256:scope.repair_plan_sha256};
const angle = {candidate_identity:identity, reviewed_files:[...scope.changed_files,'supplemental-context'], findings:[]};
const parallel = async () => Array(6).fill(angle);
const agent = async () => ({report:'Scope: reviewed every changed file and associated callers. Findings: no blocking defects. **Verdict:** PASS', decisions:{candidate_identity:identity,findings:[]}});
const args = {...scope, skill_dir:'fixture', entity_type:'bug', repair_context:{plan:'plan',plan_revision:'sha256:'+scope.repair_plan_sha256,research:'research',coverage:'coverage',prior_findings:[]}};
run(args,()=>{},parallel,agent,()=>{}).then(r=>process.stdout.write(JSON.stringify(r))).catch(e=>{console.error(e);process.exit(1)});
`
	native := exec.Command("node", "-e", nativeHarness, filepath.Join(scripts, "review_workflow.js"), filepath.Join(root, ".shark", "scope.json"))
	nativeOutput, err := native.CombinedOutput()
	require.NoError(t, err, string(nativeOutput))
	var nativeResult struct {
		Report     string          `json:"report"`
		RunnerMode string          `json:"runner_mode"`
		Angles     json.RawMessage `json:"angle_results"`
		Decisions  json.RawMessage `json:"decisions"`
	}
	require.NoError(t, json.Unmarshal(nativeOutput, &nativeResult))
	require.Equal(t, "canonical-six-angle", nativeResult.RunnerMode)
	angleBytes := []byte(nativeResult.Angles)
	anglePath := filepath.Join(root, ".shark", "angles.json")
	require.NoError(t, os.WriteFile(anglePath, angleBytes, 0600))
	decisionPath := filepath.Join(root, ".shark", "decisions.json")
	require.NoError(t, os.WriteFile(decisionPath, nativeResult.Decisions, 0600))
	bodyPath := filepath.Join(root, ".shark", "body.md")
	// The writer permits reports larger than the plan-evidence limit; admission
	// must accept the same bounded review artifacts that the writer produces.
	require.NoError(t, os.WriteFile(bodyPath, []byte(nativeResult.Report+"\n"+strings.Repeat("x", 2*1024*1024)), 0600))
	reportPath := filepath.Join(root, ".shark", "report.md")
	cmd = exec.Command("python3", filepath.Join(scripts, "adaptive_review.py"), "write-report", "--project-root", snapshot, "--source-root", root, "--body-file", bodyPath, "--review-output-path", reportPath, "--runner-mode", "canonical-six-angle", "--specialists-completed", "6", "--consolidator-completed", "true", "--base-commit", base, "--diff-path", rawScope["diff_path"].(string), "--verdict", "PASS", "--scope-file", filepath.Join(root, ".shark", "scope.json"), "--angle-results", anglePath, "--review-decisions", decisionPath)
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	reportBytes, err := os.ReadFile(reportPath)
	require.NoError(t, err)
	retainedScope, err := os.ReadFile(reportPath + ".scope.json")
	require.NoError(t, err)
	retainedScopeHash := sha256.Sum256(retainedScope)
	scopeRef := fmt.Sprintf(`{"kind":"bug_review_scope","pointer":".shark/report.md.scope.json","summary":"sha256:%x"}`, retainedScopeHash)
	retainedDecisions, err := os.ReadFile(reportPath + ".decisions.json")
	require.NoError(t, err)
	decisionHash := sha256.Sum256(retainedDecisions)
	decisionRef := fmt.Sprintf(`{"kind":"bug_review_decisions","pointer":".shark/report.md.decisions.json","summary":"sha256:%x"}`, decisionHash)
	reportHash, anglesHash := sha256.Sum256(reportBytes), sha256.Sum256(angleBytes)
	reportRef := fmt.Sprintf(`{"kind":"bug_review_report","pointer":".shark/report.md","summary":"sha256:%x"}`, reportHash)
	anglesRef := fmt.Sprintf(`{"kind":"bug_review_angles","pointer":".shark/angles.json","summary":"sha256:%x"}`, anglesHash)

	for i, extra := range []string{
		"**Verdict:** FAIL",
		"**Runner metadata:** `runner_mode=incomplete` `specialists_completed=2` `consolidator_completed=False`",
		"**Candidate:** `" + strings.Repeat("0", 40) + "`",
		"**Retained scope SHA-256:** `" + strings.Repeat("0", 64) + "`",
	} {
		conflicting := strings.Replace(string(reportBytes), "\n---\n", "\n"+extra+"\n\n---\n", 1)
		path := fmt.Sprintf(".shark/conflicting-%d.md", i)
		require.NoError(t, os.WriteFile(filepath.Join(root, path), []byte(conflicting), 0600))
		hash := sha256.Sum256([]byte(conflicting))
		ref := fmt.Sprintf(`{"kind":"bug_review_report","pointer":"%s","summary":"sha256:%x"}`, path, hash)
		err := verifyBugReviewEvidence(gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "conflicting"), EnvelopeJSON: envelope(scopeRef + "," + ref + "," + anglesRef + "," + decisionRef)})
		require.Error(t, err, "conflicting header %d must fail closed", i)
	}
	dispatched := strings.Replace(string(reportBytes), "`runner_mode=canonical-six-angle`", "`runner_mode=dispatched-six-angle`", 1)
	dispatched = strings.Replace(dispatched, "**Fallback reason:** none", "**Fallback reason:** native Workflow unavailable", 1)
	for _, tc := range []struct {
		name, content string
		valid         bool
	}{
		{"dispatched", dispatched, true},
		{"missing-fallback", strings.Replace(dispatched, "**Fallback reason:** native Workflow unavailable", "**Fallback reason:** none", 1), false},
	} {
		path := ".shark/" + tc.name + ".md"
		require.NoError(t, os.WriteFile(filepath.Join(root, path), []byte(tc.content), 0600))
		hash := sha256.Sum256([]byte(tc.content))
		ref := fmt.Sprintf(`{"kind":"bug_review_report","pointer":"%s","summary":"sha256:%x"}`, path, hash)
		err := verifyBugReviewEvidence(gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", tc.name), EnvelopeJSON: envelope(scopeRef + "," + ref + "," + anglesRef + "," + decisionRef)})
		if tc.valid {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, "review runner is incomplete")
		}
	}
	upperRef := strings.Replace(planRef, base, strings.ToUpper(base), 1)
	_, upperErr := readBugReviewScope(context.Background(), gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "uppercase"), EnvelopeJSON: envelope(upperRef + "," + scopeRef)})
	require.NoError(t, upperErr)
	_, err = readBugReviewScope(context.Background(), gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "narrow"), EnvelopeJSON: envelope(planRef + "," + narrowRef)})
	require.ErrorContains(t, err, "baseline differs")
	changedPlan := []byte("newly approved plan B\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs/plan/plan.md"), changedPlan, 0600))
	changedHash := sha256.Sum256(changedPlan)
	changedRef := fmt.Sprintf(`{"kind":"bug_repair_plan","pointer":"docs/plan/plan.md","summary":"sha256:%x"},{"kind":"bug_review_baseline","pointer":"%s"}`, changedHash, base)
	_, err = readBugReviewScope(context.Background(), gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "replan"), EnvelopeJSON: envelope(changedRef + "," + scopeRef)})
	require.ErrorContains(t, err, "plan revision differs")
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs/plan/plan.md"), plan, 0600))
	// The comparable CLI path must also interoperate with the report writer.
	var cliPacket map[string]any
	require.NoError(t, json.Unmarshal(nativeResult.Decisions, &cliPacket))
	var inspected []string
	for _, name := range []string{"change.txt", "docs/plan/plan.md", "leaf.txt"} {
		_, err := os.ReadFile(filepath.Join(snapshot, name))
		require.NoError(t, err)
		inspected = append(inspected, name)
	}
	cliPacket["reviewed_files"] = inspected
	cliDecisionBytes, err := json.Marshal(cliPacket)
	require.NoError(t, err)
	cliDecisionPath := filepath.Join(root, ".shark", "cli-decisions.json")
	require.NoError(t, os.WriteFile(cliDecisionPath, cliDecisionBytes, 0600))
	cliReportPath := filepath.Join(root, ".shark", "cli-report.md")
	cmd = exec.Command("python3", filepath.Join(scripts, "adaptive_review.py"), "write-report", "--project-root", snapshot, "--source-root", root, "--body-file", bodyPath, "--review-output-path", cliReportPath, "--runner-mode", "adversarial-cli", "--specialists-completed", "0", "--consolidator-completed", "true", "--adversarial-model", "codex", "--fallback-reason", "native and dispatched review unavailable", "--base-commit", base, "--diff-path", rawScope["diff_path"].(string), "--verdict", "PASS", "--scope-file", filepath.Join(root, ".shark", "scope.json"), "--review-decisions", cliDecisionPath)
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	cliReport, err := os.ReadFile(cliReportPath)
	require.NoError(t, err)
	cliHash := sha256.Sum256(cliReport)
	cliRef := fmt.Sprintf(`{"kind":"bug_review_report","pointer":".shark/cli-report.md","summary":"sha256:%x"}`, cliHash)
	cliScope, err := os.ReadFile(cliReportPath + ".scope.json")
	require.NoError(t, err)
	cliScopeHash := sha256.Sum256(cliScope)
	cliScopeRef := fmt.Sprintf(`{"kind":"bug_review_scope","pointer":".shark/cli-report.md.scope.json","summary":"sha256:%x"}`, cliScopeHash)
	cliDecisionBytes, err = os.ReadFile(cliReportPath + ".decisions.json")
	require.NoError(t, err)
	cliDecisionRef := fmt.Sprintf(`{"kind":"bug_review_decisions","pointer":".shark/cli-report.md.decisions.json","summary":"sha256:%x"}`, sha256.Sum256(cliDecisionBytes))
	require.NoError(t, verifyBugReviewEvidence(gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "fallback"), EnvelopeJSON: envelope(cliScopeRef + "," + cliRef + "," + cliDecisionRef)}))
	t.Run("cli_declaration_omits_predecessor_file", func(t *testing.T) {
		cliPacket["reviewed_files"] = []string{"docs/plan/plan.md", "change.txt"}
		partial, err := json.Marshal(cliPacket)
		require.NoError(t, err)
		badReport := strings.Replace(string(cliReport), fmt.Sprintf("%x", sha256.Sum256(cliDecisionBytes)), fmt.Sprintf("%x", sha256.Sum256(partial)), 1)
		require.NoError(t, os.WriteFile(filepath.Join(root, ".shark", "partial-cli-decisions.json"), partial, 0600))
		require.NoError(t, os.WriteFile(filepath.Join(root, ".shark", "partial-cli-report.md"), []byte(badReport), 0600))
		partialRef := fmt.Sprintf(`{"kind":"bug_review_decisions","pointer":".shark/partial-cli-decisions.json","summary":"sha256:%x"}`, sha256.Sum256(partial))
		reportRef := fmt.Sprintf(`{"kind":"bug_review_report","pointer":".shark/partial-cli-report.md","summary":"sha256:%x"}`, sha256.Sum256([]byte(badReport)))
		err = verifyBugReviewEvidence(gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "partial-cli"), EnvelopeJSON: envelope(cliScopeRef + "," + reportRef + "," + partialRef)})
		require.ErrorContains(t, err, "leaf.txt")
	})
	for _, tc := range []struct{ old, replacement string }{
		{"**Adversarial model:** `codex`", "**Adversarial model:** ``"},
		{"**Adversarial model:** `codex`", "**Adversarial model:** `unknown`"},
		{"**Fallback reason:** native and dispatched review unavailable", "**Fallback reason:** none"},
		{"**Fallback reason:** native and dispatched review unavailable", "**Fallback reason:** "},
	} {
		mutated := []byte(strings.Replace(string(cliReport), tc.old, tc.replacement, 1))
		require.NotEqual(t, string(cliReport), string(mutated))
		require.NoError(t, os.WriteFile(cliReportPath, mutated, 0600))
		mutatedHash := sha256.Sum256(mutated)
		mutatedRef := fmt.Sprintf(`{"kind":"bug_review_report","pointer":".shark/cli-report.md","summary":"sha256:%x"}`, mutatedHash)
		err := verifyBugReviewEvidence(gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "fallback"), EnvelopeJSON: envelope(cliScopeRef + "," + mutatedRef + "," + cliDecisionRef)})
		require.ErrorContains(t, err, "review runner is incomplete")
	}
	require.NoError(t, os.WriteFile(cliReportPath, cliReport, 0600))
	var refreshed map[string]any
	require.NoError(t, json.Unmarshal(retainedScope, &refreshed))
	refreshed["source_snapshot"].(map[string]any)["source_inventory_sha256"] = strings.Repeat("0", 64)
	refreshedBytes, err := json.Marshal(refreshed)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".shark", "refreshed-scope.json"), refreshedBytes, 0600))
	refreshedHash := sha256.Sum256(refreshedBytes)
	refreshedRef := fmt.Sprintf(`{"kind":"bug_review_scope","pointer":".shark/refreshed-scope.json","summary":"sha256:%x"}`, refreshedHash)
	err = verifyBugReviewEvidence(gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "old-report"), EnvelopeJSON: envelope(refreshedRef + "," + reportRef + "," + anglesRef + "," + decisionRef)})
	require.ErrorContains(t, err, "Retained scope digest does not match report")
	prior := envelope(planRef)
	dir, err := gaterun.RunDir(root, "approved-plan")
	require.NoError(t, err)
	lock, err := gaterun.AcquireRunLock(dir, 0)
	require.NoError(t, err)
	require.NoError(t, lock.Release())
	planRoot, planCommit, planBlob, err := bugPlanProvenance(context.Background(), gatepersist.Request{RunDir: dir, EnvelopeJSON: prior})
	require.NoError(t, err)
	digest, err := gaterun.ComputeBoundOperationDigest("B001", "bug", "plan_review", "plan_review", prior, planRoot, planCommit, planBlob)
	require.NoError(t, err)
	require.NoError(t, retainBugPlanCommit(context.Background(), gatepersist.Request{RunDir: dir, RunID: "approved-plan", PlanCommit: planCommit}))
	_, err = gaterun.CreateIdentity(dir, gaterun.RunIdentity{RunID: "approved-plan", EntityKey: "B001", EntityType: "bug", SourceStatus: "plan_review", Gate: "plan_review", OperationDigest: digest, PlanRoot: planRoot, PlanCommit: planCommit, PlanBlob: planBlob})
	require.NoError(t, err)
	_, err = gaterun.CreateResult(dir, prior)
	require.NoError(t, err)
	_, err = gaterun.CreateApprovedPlan(dir, plan)
	require.NoError(t, err)
	state := gaterun.NewOperationState("approved-plan", "B001", "bug", "plan_review", "plan_review", digest)
	state.AdmissionVersion = gatepersist.BoundAdmissionVersion
	require.NoError(t, state.MarkPersistenceComplete())
	require.NoError(t, state.MarkTransitionApplied())
	require.NoError(t, state.Save(dir))
	metaBytes, err := json.Marshal(map[string]string{"gate": "plan_review", "role": "success", "outcome_key": "pass", "run_id": "approved-plan", "operation_digest": digest})
	require.NoError(t, err)
	meta := string(metaBytes)
	notes := bugApprovalNotes{items: []*models.EntityNote{{ID: 1, Metadata: &meta}}}
	from, reason := "plan_review", gatepersist.BoundTransitionReason("plan_review", "pass", "approved-plan", digest)
	history := bugApprovalHistory{items: []*models.EntityHistory{{ID: 1, FromStatus: &from, ToStatus: "development", Notes: &reason}}}
	status := &fakeTransitioner{status: map[string]string{"B001": "code_review"}}
	written := &fakeNoteWriter{}
	coord := gatepersist.NewCoordinator(written, fakeNoteReader{}, fakeHistoryReader{}, fakeStatusValidator{valid: map[string]bool{"code_review": true, "qa": true, "resolved": true}}, status, status, &fakeLeaseReleaser{}, status)
	coord.Admission = NewBugGateAdmission(notes, history)
	ingest := func(run, gate, target string, result []byte) error {
		_, e := IngestGateResult(context.Background(), GateIngestRequest{EnvelopeBytes: result, Coordinator: coord, ProjectRoot: root, RunID: run, EntityKey: "B001", EntityType: models.EntityTypeBug, SourceStatus: gate, Gate: gate, Session: gatepersist.Session{ID: "parent"}, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}, Outcomes: map[string]string{"pass": target}})
		return e
	}
	require.Error(t, ingest("missing-scope", "code_review", "qa", envelope(planRef)))
	require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("stale\n"), 0600))
	require.Error(t, ingest("stale-scope", "code_review", "qa", envelope(planRef+","+scopeRef)))
	require.Equal(t, "code_review", status.status["B001"])
	require.Empty(t, written.notes)
	require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("candidate\n"), 0600))
	require.Error(t, ingest("missing-report", "code_review", "qa", envelope(planRef+","+scopeRef)))
	require.Error(t, ingest("missing-angles", "code_review", "qa", envelope(planRef+","+scopeRef+","+reportRef)))
	require.NoError(t, ingest("reviewed", "code_review", "qa", envelope(planRef+","+scopeRef+","+reportRef+","+anglesRef+","+decisionRef)))
	// A transition into QA is not sufficient without its completed review receipt.
	require.Error(t, ingest("missing-review-receipt", "qa", "resolved", envelope(planRef+","+scopeRef)))
	require.Equal(t, "qa", status.status["B001"])
	reviewMeta := written.notes[0].metadata
	notes.items = append(notes.items, &models.EntityNote{ID: 2, Metadata: &reviewMeta})
	reviewDigest, err := gaterun.ComputeOperationDigest("B001", "bug", "code_review", "code_review", envelope(planRef+","+scopeRef+","+reportRef+","+anglesRef+","+decisionRef))
	require.NoError(t, err)
	reviewFrom, reviewReason := "code_review", gatepersist.BoundTransitionReason("code_review", "pass", "reviewed", reviewDigest)
	history.items = append(history.items, &models.EntityHistory{ID: 2, FromStatus: &reviewFrom, ToStatus: "reviewed", Notes: &reviewReason})
	intermediate := "reviewed"
	history.items = append(history.items, &models.EntityHistory{ID: 3, FromStatus: &intermediate, ToStatus: "qa"})
	resolve := func(gate, outcome string) (string, bool) {
		if gate == "code_review" && outcome == "pass" {
			return "reviewed", true
		}
		return defaultBugSuccessTarget(gate, outcome)
	}
	coord.Admission = newBugGateAdmission(notes, history, func(value string) string { return value }, resolve)
	require.NoError(t, os.WriteFile(reportPath, []byte("changed report"), 0600))
	require.Error(t, ingest("tampered-report", "qa", "resolved", envelope(planRef+","+scopeRef)))
	require.NoError(t, os.WriteFile(reportPath, reportBytes, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("stale\n"), 0600))
	require.Error(t, ingest("stale-qa", "qa", "resolved", envelope(planRef+","+scopeRef)))
	require.Equal(t, "qa", status.status["B001"])
	require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("candidate\n"), 0600))
	require.NoError(t, ingest("valid-qa", "qa", "resolved", envelope(planRef+","+scopeRef)))
	require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("changed after review\n"), 0600))
	require.Error(t, verifyBugCandidate(context.Background(), root, scope))
	require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("candidate\n"), 0600))
	var data map[string]any
	require.NoError(t, json.Unmarshal(scope, &data))
	delete(data, "source_snapshot")
	altered, err := json.Marshal(data)
	require.NoError(t, err)
	require.Error(t, verifyBugCandidate(context.Background(), root, altered))
}

func TestBugCandidateScopesNestedProjectRuntimeWithoutHidingSource(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return strings.TrimSpace(string(out))
	}
	git("init", "-b", "main")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.invalid")
	project := filepath.Join(root, "nested")
	runtimeDir := filepath.Join(project, ".shark", "runs")
	require.NoError(t, os.MkdirAll(runtimeDir, 0700))
	tracked := filepath.Join(runtimeDir, "tracked.txt")
	require.NoError(t, os.WriteFile(tracked, []byte("base"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "app"), []byte("base"), 0600))
	git("add", "app", "nested/.shark/runs/tracked.txt")
	git("commit", "-m", "base")
	base := git("rev-parse", "HEAD")
	require.NoError(t, os.WriteFile(filepath.Join(root, "app"), []byte("candidate"), 0600))
	git("commit", "-am", "candidate")
	require.NoError(t, os.Mkdir(filepath.Join(runtimeDir, "run"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(runtimeDir, "run", "result.json"), []byte("runtime"), 0600))
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	script := filepath.Join(filepath.Dir(file), "..", "..", "skills", "shark-rider", "skills", "deep-review", "scripts", "get_diff.sh")
	capture := exec.Command("bash", script)
	capture.Dir = root
	capture.Env = append(os.Environ(), "REVIEW_BASE="+base, "REVIEW_SHARK_PROJECT_ROOT="+project)
	scope, err := capture.CombinedOutput()
	require.NoError(t, err, string(scope))
	require.NoError(t, verifyBugCandidate(context.Background(), project, scope))
	scopeFile := filepath.Join(t.TempDir(), "scope.json")
	require.NoError(t, os.WriteFile(scopeFile, scope, 0600))
	validator := filepath.Join(filepath.Dir(script), "adaptive_review.py")
	validate := func(sharkRoot string) error {
		t.Helper()
		cmd := exec.Command("python3", validator, "validate-scope", "--project-root", root, "--source-root", root, "--shark-project-root", sharkRoot, "--scope-file", scopeFile)
		return cmd.Run()
	}
	require.NoError(t, validate(project))
	require.Error(t, validate(root))
	if runtime.GOOS != "windows" {
		marker := filepath.Join(git("rev-parse", "--absolute-git-dir"), "shark-review-source.json")
		require.NoError(t, os.Symlink("missing-marker", marker))
		require.ErrorContains(t, verifyBugCandidate(context.Background(), project, scope), "snapshot source metadata missing")
		require.Error(t, validate(project))
		require.NoError(t, os.Remove(marker))
	}
	require.NoError(t, os.WriteFile(tracked, []byte("tracked change"), 0600))
	require.Error(t, validate(project))
	require.ErrorContains(t, verifyBugCandidate(context.Background(), project, scope), "candidate worktree changed")
	require.NoError(t, os.WriteFile(tracked, []byte("base"), 0600))
	otherRuntime := filepath.Join(root, ".shark", "runs")
	require.NoError(t, os.MkdirAll(otherRuntime, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(otherRuntime, "other"), []byte("unrelated"), 0600))
	require.ErrorContains(t, verifyBugCandidate(context.Background(), project, scope), "candidate worktree changed")
	require.Error(t, validate(project))
}
