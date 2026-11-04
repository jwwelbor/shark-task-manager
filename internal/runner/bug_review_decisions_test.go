package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

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
