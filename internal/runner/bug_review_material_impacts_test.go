package runner

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/stretchr/testify/require"
)

func TestValidateBugReviewMaterialImpactsRequiresExactDispositionCoverage(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, ".shark", "runs", "review")
	require.NoError(t, os.MkdirAll(runDir, 0o700))
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
	git("commit", "--allow-empty", "-m", "baseline")
	baseline := git("rev-parse", "HEAD")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "docs"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs", "decision.md"), []byte("decision"), 0o600))
	git("add", "docs/decision.md")
	git("commit", "-m", "decision")
	candidate := git("rev-parse", "HEAD")
	tree := git("rev-parse", "HEAD^{tree}")
	diff := git("diff", "--no-ext-diff", "--binary", baseline, candidate)
	diffHash := sha256.Sum256([]byte(diff))
	identity := bugReviewIdentity{Commit: candidate, Tree: tree, Diff: fmt.Sprintf("%x", diffHash), Plan: "plan"}
	expected := bugReviewMaterialImpacts{
		CandidateIdentity: identity,
		Decisions: []bugReviewMaterialDecision{{
			DecisionID: "DEC-1", SourcePointer: "docs/decision.md", SourceSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("decision"))),
			AffectedArtifacts: []bugReviewMaterialArtifact{{Path: "docs/spec.md", InvalidatedText: "old rule"}, {Path: "docs/test-plan.md", InvalidatedText: "old assertion"}},
			AffectedCriteria:  []bugReviewMaterialCriterion{{EntityKey: "E01-F01", ProductionCallerPath: "internal/handler.go", AcceptanceCriterionID: "AC-1", Assertion: "replacement works"}, {EntityKey: "E01-F01", ProductionCallerPath: "internal/handler.go", AcceptanceCriterionID: "AC-2", Assertion: "old behavior is rejected"}},
		}},
	}
	dispositions := bugReviewMaterialDispositions{
		CandidateIdentity: expected.CandidateIdentity,
		Decisions: []bugReviewMaterialDisposition{{
			DecisionID: "DEC-1",
			Artifacts:  []bugReviewArtifactDisposition{{Path: "docs/spec.md", Disposition: "amended", EvidencePointer: "docs/spec.md"}, {Path: "docs/test-plan.md", Disposition: "tracked_follow_up", EvidencePointer: "TD-001"}},
			Criteria:   []bugReviewCriterionDisposition{{EntityKey: "E01-F01", ProductionCallerPath: "internal/handler.go", AcceptanceCriterionID: "AC-1", Disposition: "regression_pass", EvidencePointer: "artifacts/ac-1.json"}, {EntityKey: "E01-F01", ProductionCallerPath: "internal/handler.go", AcceptanceCriterionID: "AC-2", Disposition: "regression_pass", EvidencePointer: "artifacts/ac-2.json"}},
		}},
	}
	encode := func(value any) []byte { data, err := json.Marshal(value); require.NoError(t, err); return data }
	write := func(name string, value any) string {
		data := encode(value)
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, data, 0o600))
		digest := sha256.Sum256(data)
		return fmt.Sprintf("sha256:%x", digest)
	}
	makeReq := func(exp bugReviewMaterialImpacts, got bugReviewMaterialDispositions) (gatepersist.Request, bugCandidateScope, string, string) {
		expHash := write("docs/expected.json", exp)
		gotHash := write("docs/dispositions.json", got)
		header := fmt.Sprintf("**Expected material impacts SHA-256:** `%s`\n**Material dispositions SHA-256:** `%s`", expHash[len("sha256:"):], gotHash[len("sha256:"):])
		req := gatepersist.Request{RunDir: runDir, EnvelopeJSON: []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_review_material_impacts","pointer":"docs/expected.json","summary":"%s"},{"kind":"bug_review_material_dispositions","pointer":"docs/dispositions.json","summary":"%s"}],"gate_result":{"schema_version":1,"summary":"test"}}`, expHash, gotHash))}
		return req, bugCandidateScope{ProjectRoot: root, CandidateCommit: candidate, CandidateTree: tree, DiffSHA256: fmt.Sprintf("%x", diffHash), RepairPlanSHA256: "plan"}, header, "MATERIAL_DECISION: DEC-1"
	}
	t.Run("complete member set passes", func(t *testing.T) {
		req, scope, header, body := makeReq(expected, dispositions)
		require.NoError(t, validateBugReviewMaterialImpacts(req, scope, header, body))
	})
	t.Run("omitted artifact is rejected", func(t *testing.T) {
		got := dispositions
		got.Decisions = append([]bugReviewMaterialDisposition(nil), dispositions.Decisions...)
		got.Decisions[0].Artifacts = append([]bugReviewArtifactDisposition(nil), dispositions.Decisions[0].Artifacts[:1]...)
		req, scope, header, body := makeReq(expected, got)
		require.ErrorContains(t, validateBugReviewMaterialImpacts(req, scope, header, body), "do not cover every expected member")
	})
	t.Run("omitted criterion is rejected", func(t *testing.T) {
		got := dispositions
		got.Decisions = append([]bugReviewMaterialDisposition(nil), dispositions.Decisions...)
		got.Decisions[0].Criteria = append([]bugReviewCriterionDisposition(nil), dispositions.Decisions[0].Criteria[:1]...)
		req, scope, header, body := makeReq(expected, got)
		require.ErrorContains(t, validateBugReviewMaterialImpacts(req, scope, header, body), "do not cover every expected member")
	})
	t.Run("missing one sidecar is rejected", func(t *testing.T) {
		req, scope, header, body := makeReq(expected, dispositions)
		env := map[string]any{"kind": "final", "recommended_outcome": "pass", "evidence": []map[string]string{{"kind": "bug_review_material_impacts", "pointer": "docs/expected.json", "summary": "sha256:" + fmt.Sprintf("%x", sha256.Sum256(mustJSON(t, expected)))}}, "gate_result": map[string]any{"schema_version": 1, "summary": "test"}}
		req.EnvelopeJSON = mustJSON(t, env)
		require.ErrorContains(t, validateBugReviewMaterialImpacts(req, scope, header, body), "requires both")
	})
	t.Run("report marker cannot omit both sidecars", func(t *testing.T) {
		req := gatepersist.Request{RunDir: runDir, EnvelopeJSON: []byte(`{"kind":"final","recommended_outcome":"pass","evidence":[],"gate_result":{"schema_version":1,"summary":"test"}}`)}
		require.ErrorContains(t, validateBugReviewMaterialImpacts(req, bugCandidateScope{}, "", "MATERIAL_DECISION: DEC-1"), "requires both")
	})
	t.Run("decision source is bound to the frozen tree", func(t *testing.T) {
		bad := expected
		bad.Decisions = append([]bugReviewMaterialDecision(nil), expected.Decisions...)
		bad.Decisions[0].SourceSHA256 = strings.Repeat("0", 64)
		req, scope, header, body := makeReq(bad, dispositions)
		require.ErrorContains(t, validateBugReviewMaterialImpacts(req, scope, header, body), "source digest does not match")
	})
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return data
}

func TestValidateBugParentTriageObservationRequiresParentReportOnlyReceipt(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, ".shark", "runs", "review")
	require.NoError(t, os.MkdirAll(runDir, 0o700))
	scope := bugCandidateScope{}
	scope.CandidateCommit, scope.CandidateTree, scope.DiffSHA256, scope.RepairPlanSHA256 = "c", "t", "d", "p"
	key := bugFindingKey{File: "a.go", Line: 4, Rule: "style", Summary: "style", Diagnosis: "style"}
	decisions := map[bugFindingKey]bugReviewDecision{key: {FindingID: "F-1", Severity: "nit", Disposition: "report_only"}}
	writeObservation := func(value any) gatepersist.Request {
		data := mustJSON(t, value)
		path := filepath.Join(root, "triage.json")
		require.NoError(t, os.WriteFile(path, data, 0o600))
		digest := sha256.Sum256(data)
		envelope := []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_parent_triage_observation","pointer":"triage.json","summary":"sha256:%x"}],"gate_result":{"schema_version":1,"summary":"test"}}`, digest))
		return gatepersist.Request{RunDir: runDir, EnvelopeJSON: envelope, ParentObservationInjected: true}
	}
	require.ErrorContains(t, validateBugParentTriageObservation(gatepersist.Request{RunDir: runDir, EnvelopeJSON: []byte(`{"kind":"final","evidence":[]}`)}, scope, decisions), "receipt is required")
	valid := bugParentTriageObservation{CandidateIdentity: bugReviewIdentity{Commit: "c", Tree: "t", Diff: "d", Plan: "p"}, ObservedBy: "parent", Actions: []bugParentTriageAction{{FindingID: "F-1", Action: "report_only"}}}
	require.NoError(t, validateBugParentTriageObservation(writeObservation(valid), scope, decisions))
	untrusted := writeObservation(valid)
	untrusted.ParentObservationInjected = false
	require.ErrorContains(t, validateBugParentTriageObservation(untrusted, scope, decisions), "parent-observed")
	valid.Actions[0].Edits = 1
	require.ErrorContains(t, validateBugParentTriageObservation(writeObservation(valid), scope, decisions), "does not prove report-only")
}
