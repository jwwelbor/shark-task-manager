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
	require.NoError(t, os.WriteFile(filepath.Join(root, "change.txt"), []byte("base\n"), 0600))
	git("add", "change.txt")
	git("commit", "-m", "base")
	base := strings.TrimSpace(git("rev-parse", "HEAD"))
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

	// Drive the same scope through the production ingestion coordinator. Store
	// test artifacts as ignored files so they do not alter the candidate.
	exclude := filepath.Join(root, ".git", "info", "exclude")
	f, err := os.OpenFile(exclude, os.O_APPEND|os.O_WRONLY, 0)
	require.NoError(t, err)
	_, err = f.WriteString("\n.shark/\nplan.md\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".shark"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".shark", "scope.json"), scope, 0600))
	plan := []byte("approved plan\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "plan.md"), plan, 0600))
	planHash, scopeHash := sha256.Sum256(plan), sha256.Sum256(scope)
	planRef := fmt.Sprintf(`{"kind":"bug_repair_plan","pointer":"plan.md","summary":"sha256:%x"}`, planHash)
	scopeRef := fmt.Sprintf(`{"kind":"bug_review_scope","pointer":".shark/scope.json","summary":"sha256:%x"}`, scopeHash)

	var captured bugCandidateScope
	require.NoError(t, json.Unmarshal(scope, &captured))
	var rawScope map[string]any
	require.NoError(t, json.Unmarshal(scope, &rawScope))
	identity := map[string]string{"commit": captured.CandidateCommit, "tree": captured.CandidateTree, "diff_sha256": captured.DiffSHA256}
	angles := map[string]any{}
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		angles[name] = map[string]any{"candidate_identity": identity, "reviewed_files": captured.ChangedFiles, "findings": []any{}}
	}
	angleBytes, err := json.Marshal(angles)
	require.NoError(t, err)
	anglePath := filepath.Join(root, ".shark", "angles.json")
	require.NoError(t, os.WriteFile(anglePath, angleBytes, 0600))
	bodyPath := filepath.Join(root, ".shark", "body.md")
	require.NoError(t, os.WriteFile(bodyPath, []byte("VERDICT: PASS\nScope: reviewed every changed file and the associated production callers. Findings: no blocking defects found.\n"), 0600))
	reportPath := filepath.Join(root, ".shark", "report.md")
	cmd = exec.Command("python3", filepath.Join(scripts, "adaptive_review.py"), "write-report", "--body-file", bodyPath, "--review-output-path", reportPath, "--runner-mode", "dispatched-six-angle", "--specialists-completed", "6", "--consolidator-completed", "true", "--base-commit", base, "--diff-path", rawScope["diff_path"].(string), "--verdict", "PASS", "--scope-file", filepath.Join(root, ".shark", "scope.json"), "--angle-results", anglePath)
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	reportBytes, err := os.ReadFile(reportPath)
	require.NoError(t, err)
	reportHash, anglesHash := sha256.Sum256(reportBytes), sha256.Sum256(angleBytes)
	reportRef := fmt.Sprintf(`{"kind":"bug_review_report","pointer":".shark/report.md","summary":"sha256:%x"}`, reportHash)
	anglesRef := fmt.Sprintf(`{"kind":"bug_review_angles","pointer":".shark/angles.json","summary":"sha256:%x"}`, anglesHash)
	envelope := func(refs string) []byte {
		return []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[%s],"gate_result":{"schema_version":1,"summary":"approved"}}`, refs))
	}
	// The comparable CLI path must also interoperate with the report writer.
	cliReportPath := filepath.Join(root, ".shark", "cli-report.md")
	cmd = exec.Command("python3", filepath.Join(scripts, "adaptive_review.py"), "write-report", "--body-file", bodyPath, "--review-output-path", cliReportPath, "--runner-mode", "adversarial-cli", "--specialists-completed", "0", "--consolidator-completed", "true", "--adversarial-model", "codex", "--fallback-reason", "native and dispatched review unavailable", "--base-commit", base, "--diff-path", rawScope["diff_path"].(string), "--verdict", "PASS", "--scope-file", filepath.Join(root, ".shark", "scope.json"))
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	cliReport, err := os.ReadFile(cliReportPath)
	require.NoError(t, err)
	cliHash := sha256.Sum256(cliReport)
	cliRef := fmt.Sprintf(`{"kind":"bug_review_report","pointer":".shark/cli-report.md","summary":"sha256:%x"}`, cliHash)
	require.NoError(t, verifyBugReviewEvidence(gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "fallback"), EnvelopeJSON: envelope(scopeRef + "," + cliRef)}))
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
	require.NoError(t, state.MarkPersistenceComplete())
	require.NoError(t, state.MarkTransitionApplied())
	require.NoError(t, state.Save(dir))
	metaBytes, err := json.Marshal(map[string]string{"gate": "plan_review", "role": "success", "outcome_key": "pass", "run_id": "approved-plan", "operation_digest": digest})
	require.NoError(t, err)
	meta := string(metaBytes)
	notes := bugApprovalNotes{items: []*models.EntityNote{{ID: 1, Metadata: &meta}}}
	from, reason := "plan_review", "gate plan_review outcome pass"
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
	require.NoError(t, ingest("reviewed", "code_review", "qa", envelope(planRef+","+scopeRef+","+reportRef+","+anglesRef)))
	// A transition into QA is not sufficient without its completed review receipt.
	require.Error(t, ingest("missing-review-receipt", "qa", "resolved", envelope(planRef+","+scopeRef)))
	require.Equal(t, "qa", status.status["B001"])
	reviewMeta := written.notes[0].metadata
	notes.items = append(notes.items, &models.EntityNote{ID: 2, Metadata: &reviewMeta})
	reviewFrom, reviewReason := "code_review", "gate code_review outcome pass"
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
