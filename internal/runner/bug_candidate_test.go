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

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

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
	git("commit", "-m", "unaccepted predecessor")
	laterBase := strings.TrimSpace(git("rev-parse", "HEAD"))
	require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("candidate\n"), 0600))
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
	cmd.Env = append(os.Environ(), "REVIEW_BASE="+base, "REVIEW_SOURCE_METADATA="+metadata)
	scope, err := cmd.Output()
	require.NoError(t, err)
	require.NoError(t, verifyBugCandidate(context.Background(), root, scope))
	require.NoError(t, os.WriteFile(filepath.Join(snapshot, "untracked"), []byte("changed snapshot"), 0600))
	require.ErrorContains(t, verifyBugCandidate(context.Background(), root, scope), "snapshot worktree changed")
	require.NoError(t, os.Remove(filepath.Join(snapshot, "untracked")))
	require.NoError(t, os.Rename(snapshot, snapshot+"-missing"))
	require.ErrorContains(t, verifyBugCandidate(context.Background(), root, scope), "snapshot commit changed or unavailable")
	require.NoError(t, os.Rename(snapshot+"-missing", snapshot))

	// Drive the same scope through the production ingestion coordinator. Store
	// test artifacts as ignored files so they do not alter the candidate.
	exclude := filepath.Join(root, ".git", "info", "exclude")
	f, err := os.OpenFile(exclude, os.O_APPEND|os.O_WRONLY, 0)
	require.NoError(t, err)
	_, err = f.WriteString("\n.shark/\nplan.md\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".shark"), 0700))
	plan := []byte("approved plan\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "plan.md"), plan, 0600))
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
	scopeHash := sha256.Sum256(scope)
	planRef := fmt.Sprintf(`{"kind":"bug_repair_plan","pointer":"plan.md","summary":"sha256:%x"},{"kind":"bug_review_baseline","pointer":"%s"}`, planHash, base)
	scopeRef := fmt.Sprintf(`{"kind":"bug_review_scope","pointer":".shark/scope.json","summary":"sha256:%x"}`, scopeHash)
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
	require.NoError(t, os.WriteFile(bodyPath, []byte(nativeResult.Report), 0600))
	reportPath := filepath.Join(root, ".shark", "report.md")
	cmd = exec.Command("python3", filepath.Join(scripts, "adaptive_review.py"), "write-report", "--body-file", bodyPath, "--review-output-path", reportPath, "--runner-mode", "canonical-six-angle", "--specialists-completed", "6", "--consolidator-completed", "true", "--base-commit", base, "--diff-path", rawScope["diff_path"].(string), "--verdict", "PASS", "--scope-file", filepath.Join(root, ".shark", "scope.json"), "--angle-results", anglePath, "--review-decisions", decisionPath)
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	reportBytes, err := os.ReadFile(reportPath)
	require.NoError(t, err)
	retainedDecisions, err := os.ReadFile(reportPath + ".decisions.json")
	require.NoError(t, err)
	decisionHash := sha256.Sum256(retainedDecisions)
	decisionRef := fmt.Sprintf(`{"kind":"bug_review_decisions","pointer":".shark/report.md.decisions.json","summary":"sha256:%x"}`, decisionHash)
	reportHash, anglesHash := sha256.Sum256(reportBytes), sha256.Sum256(angleBytes)
	reportRef := fmt.Sprintf(`{"kind":"bug_review_report","pointer":".shark/report.md","summary":"sha256:%x"}`, reportHash)
	anglesRef := fmt.Sprintf(`{"kind":"bug_review_angles","pointer":".shark/angles.json","summary":"sha256:%x"}`, anglesHash)
	envelope := func(refs string) []byte {
		return []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[%s],"gate_result":{"schema_version":1,"summary":"approved"}}`, refs))
	}
	_, err = readBugReviewScope(context.Background(), gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "narrow"), EnvelopeJSON: envelope(planRef + "," + narrowRef)})
	require.ErrorContains(t, err, "baseline differs")
	changedPlan := []byte("newly approved plan B\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "plan.md"), changedPlan, 0600))
	changedHash := sha256.Sum256(changedPlan)
	changedRef := fmt.Sprintf(`{"kind":"bug_repair_plan","pointer":"plan.md","summary":"sha256:%x"},{"kind":"bug_review_baseline","pointer":"%s"}`, changedHash, base)
	_, err = readBugReviewScope(context.Background(), gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "replan"), EnvelopeJSON: envelope(changedRef + "," + scopeRef)})
	require.ErrorContains(t, err, "plan revision differs")
	require.NoError(t, os.WriteFile(filepath.Join(root, "plan.md"), plan, 0600))
	// The comparable CLI path must also interoperate with the report writer.
	cliReportPath := filepath.Join(root, ".shark", "cli-report.md")
	cmd = exec.Command("python3", filepath.Join(scripts, "adaptive_review.py"), "write-report", "--body-file", bodyPath, "--review-output-path", cliReportPath, "--runner-mode", "adversarial-cli", "--specialists-completed", "0", "--consolidator-completed", "true", "--adversarial-model", "codex", "--fallback-reason", "native and dispatched review unavailable", "--base-commit", base, "--diff-path", rawScope["diff_path"].(string), "--verdict", "PASS", "--scope-file", filepath.Join(root, ".shark", "scope.json"), "--review-decisions", decisionPath)
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	cliReport, err := os.ReadFile(cliReportPath)
	require.NoError(t, err)
	cliHash := sha256.Sum256(cliReport)
	cliRef := fmt.Sprintf(`{"kind":"bug_review_report","pointer":".shark/cli-report.md","summary":"sha256:%x"}`, cliHash)
	require.NoError(t, verifyBugReviewEvidence(gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "fallback"), EnvelopeJSON: envelope(scopeRef + "," + cliRef + "," + decisionRef)}))
	prior := envelope(planRef)
	dir, err := gaterun.RunDir(root, "approved-plan")
	require.NoError(t, err)
	lock, err := gaterun.AcquireRunLock(dir, 0)
	require.NoError(t, err)
	require.NoError(t, lock.Release())
	digest, err := gaterun.ComputeOperationDigest("B001", "bug", "plan_review", "plan_review", prior)
	require.NoError(t, err)
	_, err = gaterun.CreateIdentity(dir, gaterun.RunIdentity{RunID: "approved-plan", EntityKey: "B001", EntityType: "bug", SourceStatus: "plan_review", Gate: "plan_review", OperationDigest: digest})
	require.NoError(t, err)
	_, err = gaterun.CreateResult(dir, prior)
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
	history.items = append(history.items, &models.EntityHistory{ID: 2, FromStatus: &reviewFrom, ToStatus: "qa", Notes: &reviewReason})
	coord.Admission = NewBugGateAdmission(notes, history)
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
