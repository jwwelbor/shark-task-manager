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
	decisions, err := validateBugReviewDecisions(packet, scope, "PASS-WITH-TRIAGE", false)
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
	decisions, err = validateBugReviewDecisions(wrongDecision, scope, "PASS-WITH-TRIAGE", false)
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
	packet := map[string]any{"candidate_identity": identity, "reviewed_files": []string{"a.go"}, "findings": []any{}}
	raw, err := json.Marshal(packet)
	require.NoError(t, err)
	decisions, err := validateBugReviewDecisions(raw, scope, "PASS", true)
	require.NoError(t, err)
	require.ErrorContains(t, validateBugReviewAngles(rawAngles, scope, decisions, "PASS"), "unresolved blocker")
	packet["findings"] = []any{map[string]any{"file": "a.go", "line": 2, "severity": "blocker", "rule": "BUG", "summary": "broken", "diagnosis": "wrong", "disposition": "open", "reason": "not refuted"}}
	raw, err = json.Marshal(packet)
	require.NoError(t, err)
	_, err = validateBugReviewDecisions(raw, scope, "PASS", true)
	require.ErrorContains(t, err, "unresolved blocker")
	packet["findings"] = []any{map[string]any{"file": "other.go", "line": 2, "severity": "blocker", "rule": "BUG", "summary": "broken", "diagnosis": "wrong", "disposition": "refuted", "reason": "not applicable"}}
	raw, err = json.Marshal(packet)
	require.NoError(t, err)
	decisions, err = validateBugReviewDecisions(raw, scope, "PASS", true)
	require.NoError(t, err)
	require.ErrorContains(t, validateBugReviewAngles(rawAngles, scope, decisions, "PASS"), "unresolved blocker")
	packet["findings"] = []any{map[string]any{"file": "a.go", "line": 2, "severity": "blocker", "rule": "BUG", "summary": "broken", "diagnosis": "wrong", "disposition": "refuted", "reason": "call site proves it cannot occur"}}
	raw, err = json.Marshal(packet)
	require.NoError(t, err)
	decisions, err = validateBugReviewDecisions(raw, scope, "PASS", true)
	require.NoError(t, err)
	require.NoError(t, validateBugReviewAngles(rawAngles, scope, decisions, "PASS"))
}

func TestReviewDecisionsRequireCompleteInspectedFileManifest(t *testing.T) {
	scope := bugCandidateScope{CandidateCommit: "commit", CandidateTree: "tree", DiffSHA256: "diff", ChangedFiles: []string{"a.go", "b.go"}}
	packet := map[string]any{
		"candidate_identity": map[string]string{"commit": "commit", "tree": "tree", "diff_sha256": "diff"},
		"reviewed_files":     []string{"a.go", "context.go"},
		"findings":           []any{},
	}
	raw, err := json.Marshal(packet)
	require.NoError(t, err)
	_, err = validateBugReviewDecisions(raw, scope, "PASS", true)
	require.ErrorContains(t, err, "coverage incomplete")
	packet["reviewed_files"] = []string{"a.go", "b.go", "context.go"}
	raw, err = json.Marshal(packet)
	require.NoError(t, err)
	_, err = validateBugReviewDecisions(raw, scope, "PASS", true)
	require.NoError(t, err)
	for _, path := range []string{"/outside.go", "../outside.go", "x/../../outside.go"} {
		packet["reviewed_files"] = []string{"a.go", "b.go", path}
		raw, err := json.Marshal(packet)
		require.NoError(t, err)
		_, err = validateBugReviewDecisions(raw, scope, "PASS", true)
		require.ErrorContains(t, err, path)
	}
	delete(packet, "reviewed_files")
	raw, err = json.Marshal(packet)
	require.NoError(t, err)
	_, err = validateBugReviewDecisions(raw, scope, "PASS", true)
	require.ErrorContains(t, err, "reviewed_files missing")
}

func TestSixAngleDecisionsUsePerAngleCoverageInsteadOfConsolidatedManifest(t *testing.T) {
	scope := bugCandidateScope{CandidateCommit: "commit", CandidateTree: "tree", DiffSHA256: "diff", ChangedFiles: []string{"a.go", "b.go"}}
	packet := map[string]any{
		"candidate_identity": map[string]string{"commit": "commit", "tree": "tree", "diff_sha256": "diff"},
		"findings":           []any{},
	}
	raw, err := json.Marshal(packet)
	require.NoError(t, err)
	_, err = validateBugReviewDecisions(raw, scope, "PASS", false)
	require.NoError(t, err, "six-angle coverage is validated from each angle manifest")
	_, err = validateBugReviewDecisions(raw, scope, "PASS", true)
	require.ErrorContains(t, err, "reviewed_files missing", "CLI packets still require their consolidated manifest")
}

func TestSixAngleCoverageRejectsFindingFromAnOmittedPath(t *testing.T) {
	scope := bugCandidateScope{CandidateCommit: "commit", CandidateTree: "tree", DiffSHA256: "diff", ChangedFiles: []string{"one.go", "two.go"}}
	identity := map[string]string{"commit": "commit", "tree": "tree", "diff_sha256": "diff"}
	angles := map[string]any{}
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		angles[name] = map[string]any{"candidate_identity": identity, "reviewed_files": []string{"one.go", "two.go"}, "findings": []any{}}
	}
	angles["a"] = map[string]any{"candidate_identity": identity, "reviewed_files": []string{"one.go", "two.go"}, "findings": []any{
		map[string]any{"file": "two.go", "line": 4, "severity": "nit", "rule": "STYLE", "summary": "label", "diagnosis": "unclear label"},
	}}
	decisions := map[bugFindingKey]bugReviewDecision{{File: "two.go", Line: 4, Rule: "STYLE", Summary: "label", Diagnosis: "unclear label"}: {Severity: "nit", Disposition: "report-only", Reason: "owner decision requires no action"}}
	data, err := json.Marshal(angles)
	require.NoError(t, err)
	require.NoError(t, validateBugReviewAngles(data, scope, decisions, "PASS-WITH-TRIAGE"))

	angles["a"].(map[string]any)["reviewed_files"] = []string{"one.go"}
	data, err = json.Marshal(angles)
	require.NoError(t, err)
	require.ErrorContains(t, validateBugReviewAngles(data, scope, decisions, "PASS-WITH-TRIAGE"), "coverage incomplete")
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
				angles[name] = map[string]any{"candidate_identity": fixture.Identity, "reviewed_files": append(append([]string{}, fixture.Files...), "context"), "findings": findings}
			}
			angleBytes, e := json.Marshal(angles)
			require.NoError(t, e)
			verdict := tc.Verdict
			if verdict == "" {
				verdict = "PASS"
			}
			decisions, e := validateBugReviewDecisions(tc.Decisions, scope, verdict, true)
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
	other.FindingID, other.Severity, other.Disposition = "F-2", "nit", "report-only"
	multi := body + "\nHISTORICAL_REVIEW_STATUS: DID_NOT_FINISH\nFINDING_DETAIL: INCOMPLETE PRODUCT BEHAVIOR | F-2 | nit | report-only"
	require.True(t, statusSafeBody("CURRENT_REVIEW_STATUS: COMPLETE\nScope and findings: complete. VERDICT: PASS-WITH-TRIAGE\n"+multi))
	require.NoError(t, requireBugFindingDecisions(multi, map[bugFindingKey]bugReviewDecision{{File: "related"}: related, {File: "other"}: other}))
	require.Error(t, requireBugFindingDecisions(body+"\n"+body, map[bugFindingKey]bugReviewDecision{{File: "related"}: related}))
}

func TestReportFindingDecisionBindingThroughJSON(t *testing.T) {
	scope := bugCandidateScope{CandidateCommit: "commit", CandidateTree: "tree", DiffSHA256: "diff"}
	body := "FINDING_DETAIL: INCOMPLETE VALIDATION | F-1 | non-blocker | triage"
	packet := map[string]any{"candidate_identity": map[string]string{"commit": "commit", "tree": "tree", "diff_sha256": "diff"}, "reviewed_files": []string{}, "findings": []any{map[string]any{"finding_id": "F-2", "file": "other.go", "line": 1, "severity": "nit", "rule": "STYLE", "summary": "typo", "diagnosis": "spelling", "disposition": "report-only", "reason": "owner decision requires no action"}}}
	raw, err := json.Marshal(packet)
	require.NoError(t, err)
	decisions, err := validateBugReviewDecisions(raw, scope, "PASS-WITH-TRIAGE", true)
	require.NoError(t, err)
	require.Error(t, requireBugFindingDecisions(body, decisions))
	packet["findings"].([]any)[0].(map[string]any)["disposition"] = "triage"
	raw, err = json.Marshal(packet)
	require.NoError(t, err)
	_, err = validateBugReviewDecisions(raw, scope, "PASS-WITH-TRIAGE", true)
	require.ErrorContains(t, err, "nit disposition must be report-only")
	packet["findings"] = []any{map[string]any{"finding_id": "F-1", "file": "app.go", "line": 2, "severity": "non-blocker", "rule": "INPUT", "summary": "validation", "diagnosis": "missing check", "disposition": "triage", "reason": "tracked"}}
	raw, err = json.Marshal(packet)
	require.NoError(t, err)
	decisions, err = validateBugReviewDecisions(raw, scope, "PASS-WITH-TRIAGE", true)
	require.NoError(t, err)
	require.NoError(t, requireBugFindingDecisions(body, decisions))
}

func TestReviewFileDeclarationUsesGitPathComponents(t *testing.T) {
	scope := bugCandidateScope{ChangedFiles: []string{" "}}
	for _, tc := range []struct{ member, wantError string }{
		{`" "`, ""}, {`"missing"`, "coverage incomplete"},
		{`" ","a//b"`, "repository-relative"}, {`" ","a/./b"`, "repository-relative"},
		{`" ","a/../b"`, "repository-relative"}, {`" ","end/"`, "repository-relative"},
		{`" ",""`, "repository-relative"}, {`" ","/absolute"`, "repository-relative"},
		{`" ","a\u0000b"`, "repository-relative"},
		{`" ","a\\b","caf\udce9","caf\u00e9","caf\ufffd","caf\\udce9","\ud800\udce9"`, ""},
		{`" ","\ud800"`, "surrogate"}, {`" ","\udc00"`, "surrogate"},
		{`" ","\udc80\udfff"`, "surrogate"}, {"\" \",\"\xe9\"", "UTF-8"},
	} {
		t.Run(tc.member, func(t *testing.T) {
			packet := []byte(`{"candidate_identity":{},"findings":[],"reviewed_files":[` + tc.member + `]}`)
			_, err := validateBugReviewDecisions(packet, scope, "PASS", true)
			if tc.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantError)
			}
		})
	}
}

func TestMissingGitPathDiagnosticPreservesOriginalMember(t *testing.T) {
	scope := bugCandidateScope{ChangedFiles: []string{"caf\xe9.txt"}}
	packet := []byte(`{"candidate_identity":{},"reviewed_files":["caf\ufffd.txt"],"findings":[]}`)
	_, err := validateBugReviewDecisions(packet, scope, "PASS", true)
	require.ErrorContains(t, err, `"caf\udce9.txt"`)
}
