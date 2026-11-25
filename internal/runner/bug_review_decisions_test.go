package runner

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReviewPathsDistinguishAByteFromTheReplacementCharacter(t *testing.T) {
	const identity = `{"commit":"c","tree":"t","diff_sha256":"d"}`
	const site = `{"file":"caf\udce9","line":1,"severity":"non-blocker","rule":"BUG","summary":"result","diagnosis":"caller","disposition":"triage","reason":"observed"}`
	scope := bugCandidateScope{CandidateCommit: "c", CandidateTree: "t", DiffSHA256: "d", ChangedFiles: []string{"caf\xe9", "caf\uFFFD"}}
	packet := []byte(`{"candidate_identity":` + identity + `,"findings":[` + site + `]}`)
	decisions, err := validateBugReviewDecisions(packet, scope, "PASS-WITH-TRIAGE")
	require.NoError(t, err)
	angles := map[string]json.RawMessage{}
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		angles[name] = json.RawMessage(`{"candidate_identity":` + identity + `,"reviewed_files":["caf\udce9","caf\ufffd"],"findings":[]}`)
	}
	angles["a"] = json.RawMessage(`{"candidate_identity":` + identity + `,"reviewed_files":["caf\udce9","caf\ufffd"],"findings":[` + site + `]}`)
	raw, err := json.Marshal(angles)
	require.NoError(t, err)
	require.NoError(t, validateBugReviewAngles(raw, scope, decisions, "PASS-WITH-TRIAGE"))
	for angle, member := range angles {
		angles[angle] = bytes.Replace(member, []byte(`["caf\udce9"`), []byte(`["caf\ufffd"`), 1)
		raw, err := json.Marshal(angles)
		require.NoError(t, err)
		require.ErrorContains(t, validateBugReviewAngles(raw, scope, decisions, "PASS-WITH-TRIAGE"), "angle "+angle)
		angles[angle] = member
	}
	wrongDecision := bytes.Replace(packet, []byte(`"file":"caf\udce9"`), []byte(`"file":"caf\ufffd"`), 1)
	decisions, err = validateBugReviewDecisions(wrongDecision, scope, "PASS-WITH-TRIAGE")
	require.NoError(t, err)
	require.ErrorContains(t, validateBugReviewAngles(raw, scope, decisions, "PASS-WITH-TRIAGE"), "no explicit disposition")
}

func TestReviewDecisionsRequireExactBlockerRefutation(t *testing.T) {
	scope := bugCandidateScope{CandidateCommit: "commit", CandidateTree: "tree", DiffSHA256: "diff", RepairPlanSHA256: "plan", ChangedFiles: []string{"a.go"}}
	identity := map[string]string{"commit": "commit", "tree": "tree", "diff_sha256": "diff", "repair_plan_sha256": "plan"}
	blocker := map[string]any{"file": "a.go", "line": 2, "severity": "blocker", "rule": "BUG", "summary": "broken", "diagnosis": "wrong"}
	angles := map[string]any{}
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		angles[name] = map[string]any{"candidate_identity": identity, "reviewed_files": []string{"a.go", "context.go"}, "findings": []any{}}
	}
	angles["a"] = map[string]any{"candidate_identity": identity, "reviewed_files": []string{"a.go"}, "findings": []any{blocker}}
	rawAngles, err := json.Marshal(angles)
	require.NoError(t, err)
	packet := map[string]any{"candidate_identity": identity, "findings": []any{}}
	raw, err := json.Marshal(packet)
	require.NoError(t, err)
	decisions, err := validateBugReviewDecisions(raw, scope, "PASS")
	require.NoError(t, err)
	require.ErrorContains(t, validateBugReviewAngles(rawAngles, scope, decisions, "PASS"), "unresolved blocker")
	packet["findings"] = []any{map[string]any{"file": "a.go", "line": 2, "severity": "blocker", "rule": "BUG", "summary": "broken", "diagnosis": "wrong", "disposition": "open", "reason": "not refuted"}}
	raw, err = json.Marshal(packet)
	require.NoError(t, err)
	_, err = validateBugReviewDecisions(raw, scope, "PASS")
	require.ErrorContains(t, err, "unresolved blocker")
	packet["findings"] = []any{map[string]any{"file": "other.go", "line": 2, "severity": "blocker", "rule": "BUG", "summary": "broken", "diagnosis": "wrong", "disposition": "refuted", "reason": "not applicable"}}
	raw, err = json.Marshal(packet)
	require.NoError(t, err)
	decisions, err = validateBugReviewDecisions(raw, scope, "PASS")
	require.NoError(t, err)
	require.ErrorContains(t, validateBugReviewAngles(rawAngles, scope, decisions, "PASS"), "unresolved blocker")
	packet["findings"] = []any{map[string]any{"file": "a.go", "line": 2, "severity": "blocker", "rule": "BUG", "summary": "broken", "diagnosis": "wrong", "disposition": "refuted", "reason": "call site proves it cannot occur"}}
	raw, err = json.Marshal(packet)
	require.NoError(t, err)
	decisions, err = validateBugReviewDecisions(raw, scope, "PASS")
	require.NoError(t, err)
	require.NoError(t, validateBugReviewAngles(rawAngles, scope, decisions, "PASS"))
}

func TestVersionTwoReviewContractRequiresImpact(t *testing.T) {
	scope := bugCandidateScope{CandidateCommit: "commit", CandidateTree: "tree", DiffSHA256: "diff", RepairPlanSHA256: "plan"}
	packet := map[string]any{
		"review_contract_version": 2,
		"candidate_identity":      map[string]string{"commit": "commit", "tree": "tree", "diff_sha256": "diff", "repair_plan_sha256": "plan"},
		"findings":                []any{map[string]any{"file": "a.go", "line": 2, "severity": "non-blocker", "rule": "CONTRACT", "summary": "missing event", "diagnosis": "event is absent", "disposition": "triage", "reason": "route it"}},
	}
	raw, err := json.Marshal(packet)
	require.NoError(t, err)
	_, err = validateBugReviewDecisions(raw, scope, "PASS-WITH-TRIAGE")
	require.ErrorContains(t, err, "missing impact")
	packet["findings"] = []any{map[string]any{"file": "a.go", "line": 2, "severity": "nit", "rule": "STYLE", "summary": "name preference", "diagnosis": "alternate name", "impact": "no impact", "disposition": "triage", "reason": "style"}}
	raw, err = json.Marshal(packet)
	require.NoError(t, err)
	_, err = validateBugReviewDecisions(raw, scope, "PASS-WITH-TRIAGE")
	require.ErrorContains(t, err, "legacy nit severity")
}

func TestReviewDecisionConformanceFixtures(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	path := filepath.Join(filepath.Dir(file), "..", "..", "skills", "shark-rider", "skills", "deep-review", "scripts", "testdata", "decisions.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var fixture struct {
		Identity map[string]string `json:"identity"`
		Files    []string          `json:"files"`
		Cases    []struct {
			Name        string           `json:"name"`
			RawFindings []map[string]any `json:"raw_findings"`
			Decisions   json.RawMessage  `json:"decisions"`
			Valid       bool             `json:"valid"`
			Verdict     string           `json:"verdict"`
		} `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(raw, &fixture))
	scope := bugCandidateScope{CandidateCommit: fixture.Identity["commit"], CandidateTree: fixture.Identity["tree"], DiffSHA256: fixture.Identity["diff_sha256"], RepairPlanSHA256: fixture.Identity["repair_plan_sha256"], ChangedFiles: fixture.Files}
	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			angles := map[string]any{}
			for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
				findings := []map[string]any{}
				if name == "a" {
					findings = tc.RawFindings
				}
				angles[name] = map[string]any{"review_contract_version": 2, "candidate_identity": fixture.Identity, "reviewed_files": append(append([]string{}, fixture.Files...), "context"), "findings": findings}
			}
			angleBytes, e := json.Marshal(angles)
			require.NoError(t, e)
			verdict := tc.Verdict
			if verdict == "" {
				verdict = "PASS"
			}
			decisions, e := validateBugReviewDecisions(tc.Decisions, scope, verdict)
			if e == nil {
				e = validateBugReviewAngles(angleBytes, scope, decisions, verdict)
			}
			require.Equal(t, tc.Valid, e == nil, "error: %v", e)
		})
	}
}

func TestReviewBodyDoesNotTreatSubstringAsIncomplete(t *testing.T) {
	body := "Scope: all changed files and tests reviewed. Findings: zero defects; an incompletely documented prior attempt was resolved. VERDICT: PASS"
	require.NoError(t, validateBugReviewBody("**Verdict:** PASS", body))
	require.Error(t, validateBugReviewBody("**Verdict:** PASS", body+" VERDICT: FAIL"))
	require.Error(t, validateBugReviewBody("**Verdict:** PASS", body+string([]byte{0xff})))
}

func TestReviewBodyConformanceFixtures(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	path := filepath.Join(filepath.Dir(file), "..", "..", "skills", "shark-rider", "skills", "deep-review", "scripts", "testdata", "report_bodies.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var cases []struct {
		Name  string `json:"name"`
		Body  string `json:"body"`
		Valid bool   `json:"valid"`
	}
	require.NoError(t, json.Unmarshal(raw, &cases))
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			err := validateBugReviewBody("**Verdict:** PASS", tc.Body)
			require.Equal(t, tc.Valid, err == nil, "error: %v", err)
		})
	}
}

func TestFailedReviewMayDescribeIncompleteProductValidation(t *testing.T) {
	body := "CURRENT_REVIEW_STATUS: COMPLETE\nFINDING_CONTEXT: application external input validation is deficient.\nScope: all candidate files reviewed. Findings: validation bug. VERDICT: FAIL\nFINDING_DETAIL: INCOMPLETE VALIDATION | F-1 | non-blocker | triage"
	require.True(t, statusSafeBody(body))
	require.False(t, statusSafeBody(body+"\nThe current review is incomplete."))
	require.Error(t, validateBugReviewBody("**Verdict:** FAIL", body))
}

func TestReportFindingBindsToItsDecision(t *testing.T) {
	body := "FINDING_DETAIL: INCOMPLETE VALIDATION | F-1 | non-blocker | triage"
	other := bugReviewDecision{FindingID: "F-2", Severity: "non-blocker", Disposition: "triage"}
	related := bugReviewDecision{FindingID: "F-1", Severity: "non-blocker", Disposition: "triage"}
	require.Error(t, requireBugFindingDecisions(body, map[bugFindingKey]bugReviewDecision{{File: "other"}: other}))
	require.NoError(t, requireBugFindingDecisions(body, map[bugFindingKey]bugReviewDecision{{File: "related"}: related}))
	related.Disposition = "refuted"
	require.Error(t, requireBugFindingDecisions(body, map[bugFindingKey]bugReviewDecision{{File: "related"}: related}))
	related.Disposition = "Triage"
	require.Error(t, requireBugFindingDecisions(body, map[bugFindingKey]bugReviewDecision{{File: "related"}: related}))
	related.Disposition = "triage"
	other.FindingID, other.Severity = "F-2", "nit"
	multi := body + "\nHISTORICAL_REVIEW_STATUS: DID_NOT_FINISH\nFINDING_DETAIL: INCOMPLETE PRODUCT BEHAVIOR | F-2 | nit | triage"
	require.True(t, statusSafeBody("CURRENT_REVIEW_STATUS: COMPLETE\nScope and findings: complete. VERDICT: PASS-WITH-TRIAGE\n"+multi))
	require.NoError(t, requireBugFindingDecisions(multi, map[bugFindingKey]bugReviewDecision{{File: "related"}: related, {File: "other"}: other}))
	require.Error(t, requireBugFindingDecisions(body+"\n"+body, map[bugFindingKey]bugReviewDecision{{File: "related"}: related}))
}

func TestReportFindingDecisionBindingThroughJSON(t *testing.T) {
	scope := bugCandidateScope{CandidateCommit: "commit", CandidateTree: "tree", DiffSHA256: "diff"}
	body := "FINDING_DETAIL: INCOMPLETE VALIDATION | F-1 | non-blocker | triage"
	packet := map[string]any{"candidate_identity": map[string]string{"commit": "commit", "tree": "tree", "diff_sha256": "diff"}, "findings": []any{map[string]any{"finding_id": "F-2", "file": "other.go", "line": 1, "severity": "nit", "rule": "STYLE", "summary": "typo", "diagnosis": "spelling", "disposition": "triage", "reason": "tracked"}}}
	raw, err := json.Marshal(packet)
	require.NoError(t, err)
	decisions, err := validateBugReviewDecisions(raw, scope, "PASS-WITH-TRIAGE")
	require.NoError(t, err)
	require.Error(t, requireBugFindingDecisions(body, decisions.byFinding))
	packet["findings"] = []any{map[string]any{"finding_id": "F-1", "file": "app.go", "line": 2, "severity": "non-blocker", "rule": "INPUT", "summary": "validation", "diagnosis": "missing check", "disposition": "triage", "reason": "tracked"}}
	raw, err = json.Marshal(packet)
	require.NoError(t, err)
	decisions, err = validateBugReviewDecisions(raw, scope, "PASS-WITH-TRIAGE")
	require.NoError(t, err)
	require.NoError(t, requireBugFindingDecisions(body, decisions.byFinding))
}
