package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
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
		require.NoError(t, validateBugReviewMaterialImpacts(context.Background(), req, scope, header, body))
	})
	t.Run("omitted artifact is rejected", func(t *testing.T) {
		got := dispositions
		got.Decisions = append([]bugReviewMaterialDisposition(nil), dispositions.Decisions...)
		got.Decisions[0].Artifacts = append([]bugReviewArtifactDisposition(nil), dispositions.Decisions[0].Artifacts[:1]...)
		req, scope, header, body := makeReq(expected, got)
		require.ErrorContains(t, validateBugReviewMaterialImpacts(context.Background(), req, scope, header, body), "DEC-1: missing artifact disposition for docs/test-plan.md")
	})
	t.Run("omitted criterion is rejected", func(t *testing.T) {
		got := dispositions
		got.Decisions = append([]bugReviewMaterialDisposition(nil), dispositions.Decisions...)
		got.Decisions[0].Criteria = append([]bugReviewCriterionDisposition(nil), dispositions.Decisions[0].Criteria[:1]...)
		req, scope, header, body := makeReq(expected, got)
		require.ErrorContains(t, validateBugReviewMaterialImpacts(context.Background(), req, scope, header, body), "DEC-1: missing criterion disposition for E01-F01 internal/handler.go AC-2")
	})
	t.Run("missing one sidecar is rejected", func(t *testing.T) {
		req, scope, header, body := makeReq(expected, dispositions)
		env := map[string]any{"kind": "final", "recommended_outcome": "pass", "evidence": []map[string]string{{"kind": "bug_review_material_impacts", "pointer": "docs/expected.json", "summary": "sha256:" + fmt.Sprintf("%x", sha256.Sum256(mustJSON(t, expected)))}}, "gate_result": map[string]any{"schema_version": 1, "summary": "test"}}
		req.EnvelopeJSON = mustJSON(t, env)
		require.ErrorContains(t, validateBugReviewMaterialImpacts(context.Background(), req, scope, header, body), "requires both")
	})
	t.Run("report marker cannot omit both sidecars", func(t *testing.T) {
		req := gatepersist.Request{RunDir: runDir, EnvelopeJSON: []byte(`{"kind":"final","recommended_outcome":"pass","evidence":[],"gate_result":{"schema_version":1,"summary":"test"}}`)}
		require.ErrorContains(t, validateBugReviewMaterialImpacts(context.Background(), req, bugCandidateScope{}, "", "MATERIAL_DECISION: DEC-1"), "requires both")
	})
	t.Run("decision source is bound to the frozen tree", func(t *testing.T) {
		bad := expected
		bad.Decisions = append([]bugReviewMaterialDecision(nil), expected.Decisions...)
		bad.Decisions[0].SourceSHA256 = strings.Repeat("0", 64)
		req, scope, header, body := makeReq(bad, dispositions)
		require.ErrorContains(t, validateBugReviewMaterialImpacts(context.Background(), req, scope, header, body), "source digest does not match")
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

func installMaterialDecisionFixture(t *testing.T, root string) {
	t.Helper()
	for name, content := range map[string]string{
		"spec.md":      "Preserve zero and positive quantities.\n",
		"test-plan.md": "Check that quantities zero and three return zero and three.\n",
		"caller.py":    "import sys\nprint(int(sys.argv[1]))\n",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(content), 0600))
	}
}

func recordMaterialCriterionRuns(t *testing.T, root string, scope bugCandidateScope, decision bugReviewMaterialDecision) {
	t.Helper()
	var observations []map[string]string
	for i, quantity := range []string{"0", "3"} {
		c := decision.AffectedCriteria[i]
		cmd := exec.Command("python3", c.ProductionCallerPath, quantity)
		cmd.Dir = scope.ProjectRoot
		out, err := cmd.Output()
		require.NoError(t, err)
		require.Equal(t, quantity+"\n", string(out))
		observations = append(observations, map[string]string{
			"owner": c.EntityKey, "caller": c.ProductionCallerPath, "criterion": c.AcceptanceCriterionID,
			"candidate": scope.CandidateCommit, "argument": quantity, "observed": string(out), "outcome": "pass",
		})
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, ".shark", "criteria.json"), mustJSON(t, observations), 0600))
}

// Exercise the same hash-bound omission independently at current ingestion and
// on replay of an applied review. Each run starts with its own gate state.
func testMaterialDispositionAdmission(t *testing.T, root string, scope bugCandidateScope, plan []byte, report, scopeRefs, reviewRefs string, planNotes bugApprovalNotes, planHistory bugApprovalHistory) {
	t.Helper()
	identity := bugReviewIdentity{Commit: scope.CandidateCommit, Tree: scope.CandidateTree, Diff: scope.DiffSHA256, Plan: scope.RepairPlanSHA256}
	writer := func(name, kind string, data []byte) string {
		t.Helper()
		path := ".shark/" + name
		require.NoError(t, os.WriteFile(filepath.Join(root, path), data, 0600))
		return fmt.Sprintf(`{"kind":%q,"pointer":%q,"summary":"sha256:%x"}`, kind, path, sha256.Sum256(data))
	}
	envelope := func(refs string) []byte {
		return []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[%s],"gate_result":{"schema_version":1,"summary":"checked review"}}`, refs))
	}
	decision := bugReviewMaterialDecision{
		DecisionID: "DEC-1", SourcePointer: "docs/plan/plan.md", SourceSHA256: fmt.Sprintf("%x", sha256.Sum256(plan)),
		AffectedArtifacts: []bugReviewMaterialArtifact{
			{Path: "spec.md", InvalidatedText: "old rule"},
			{Path: "test-plan.md", InvalidatedText: "old assertion"},
		},
		AffectedCriteria: []bugReviewMaterialCriterion{
			{EntityKey: "E01-F01", ProductionCallerPath: "caller.py", AcceptanceCriterionID: "AC-1", Assertion: "zero returns zero"},
			{EntityKey: "E01-F01", ProductionCallerPath: "caller.py", AcceptanceCriterionID: "AC-2", Assertion: "three returns three"},
		},
	}
	recordMaterialCriterionRuns(t, root, scope, decision)
	expected := mustJSON(t, bugReviewMaterialImpacts{CandidateIdentity: identity, Decisions: []bugReviewMaterialDecision{decision}})
	expectedRef := writer("expected.json", "bug_review_material_impacts", expected)

	for _, trial := range []struct{ name, want string }{
		{"omit-artifact", "DEC-1: missing artifact disposition for test-plan.md"},
		{"omit-criterion", "DEC-1: missing criterion disposition for E01-F01 caller.py AC-2"},
		{"complete", ""},
	} {
		disposition := bugReviewMaterialDisposition{DecisionID: decision.DecisionID}
		for _, a := range decision.AffectedArtifacts {
			disposition.Artifacts = append(disposition.Artifacts, bugReviewArtifactDisposition{Path: a.Path, Disposition: "amended", EvidencePointer: a.Path})
		}
		for _, c := range decision.AffectedCriteria {
			disposition.Criteria = append(disposition.Criteria, bugReviewCriterionDisposition{
				EntityKey: c.EntityKey, ProductionCallerPath: c.ProductionCallerPath, AcceptanceCriterionID: c.AcceptanceCriterionID,
				Disposition: "regression_pass", EvidencePointer: ".shark/criteria.json",
			})
		}
		switch trial.name {
		case "omit-artifact":
			disposition.Artifacts = disposition.Artifacts[:1]
		case "omit-criterion":
			disposition.Criteria = disposition.Criteria[:1]
		}
		got := mustJSON(t, bugReviewMaterialDispositions{CandidateIdentity: identity, Decisions: []bugReviewMaterialDisposition{disposition}})
		dispositionRef := writer(trial.name+".json", "bug_review_material_dispositions", got)
		headers := fmt.Sprintf("\n**Expected material impacts SHA-256:** `%x`\n**Material dispositions SHA-256:** `%x`\n", sha256.Sum256(expected), sha256.Sum256(got))
		bound := strings.Replace(report, "\n---\n", headers+"\n---\n", 1) + "\nMATERIAL_DECISION: DEC-1\n"
		reportRef := writer(trial.name+".md", "bug_review_report", []byte(bound))
		shared := strings.Join([]string{scopeRefs, expectedRef, dispositionRef}, ",")
		prior := envelope(shared + "," + reportRef + "," + reviewRefs)

		for _, gate := range []string{"code_review", "qa"} {
			t.Run(trial.name+"_"+gate, func(t *testing.T) {
				notes, history := planNotes, planHistory
				target, result := "qa", prior
				var seeded string
				var retained map[string][sha256.Size]byte
				if gate == "qa" {
					target, result = "resolved", envelope(shared)
					seeded = "seed-" + trial.name
					seedMaterialReview(t, root, seeded, prior, &notes, &history)
					retained = materialReviewHashes(t, root, seeded, trial.name)
				}
				status := &fakeTransitioner{status: map[string]string{"B001": gate}}
				written := &fakeNoteWriter{}
				coord := gatepersist.NewCoordinator(written, fakeNoteReader{}, fakeHistoryReader{}, fakeStatusValidator{valid: map[string]bool{"qa": true, "resolved": true, "code_review": true}}, status, status, &fakeLeaseReleaser{}, status)
				coord.Admission = NewBugGateAdmission(notes, history)
				request := GateIngestRequest{
					EnvelopeBytes: result, Coordinator: coord, ProjectRoot: root, RunID: trial.name + "-" + gate,
					EntityKey: "B001", EntityType: models.EntityTypeBug, SourceStatus: gate, Gate: gate,
					Session: gatepersist.Session{ID: "parent"}, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
					Outcomes: map[string]string{"pass": target},
				}
				if trial.name == "complete" {
					for _, challenge := range []string{"cancel", "overflow"} {
						t.Run(challenge+"_frozen_source", func(t *testing.T) {
							err := ingestMaterialSourceStream(t, request, scope, decision.SourcePointer, challenge)
							if challenge == "cancel" {
								require.ErrorIs(t, err, context.Canceled)
							} else {
								require.ErrorContains(t, err, fmt.Sprintf("%d bytes expected, read %d", len(plan), len(plan)+1))
							}
							require.Empty(t, written.notes)
							require.Equal(t, gate, status.status["B001"])
						})
					}
				}
				var err error
				if trial.name == "complete" {
					err = ingestMaterialSourceStream(t, request, scope, decision.SourcePointer, "release")
				} else {
					_, err = IngestGateResult(context.Background(), request)
				}
				if trial.want != "" {
					require.ErrorContains(t, err, trial.want)
					require.Empty(t, written.notes)
					require.Equal(t, gate, status.status["B001"])
				} else {
					require.NoError(t, err)
					require.Len(t, written.notes, 1)
					stored, exists, readErr := gaterun.ReadResult(filepath.Join(root, ".shark", "runs", trial.name+"-"+gate))
					require.NoError(t, readErr)
					require.True(t, exists)
					require.Equal(t, result, stored)
					require.Equal(t, target, status.status["B001"])
				}
				if seeded != "" {
					require.Equal(t, retained, materialReviewHashes(t, root, seeded, trial.name))
				}
			})
		}
	}
}

func materialReviewHashes(t *testing.T, root, run, trial string) map[string][sha256.Size]byte {
	t.Helper()
	names := []string{"expected.json", trial + ".json", trial + ".md"}
	for _, file := range []string{"identity.json", "result.json", "operation-state.json"} {
		names = append(names, filepath.Join("runs", run, file))
	}
	hashes := make(map[string][sha256.Size]byte, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(root, ".shark", name))
		require.NoError(t, err)
		hashes[name] = sha256.Sum256(data)
	}
	return hashes
}

func seedMaterialReview(t *testing.T, root, run string, result []byte, notes *bugApprovalNotes, history *bugApprovalHistory) {
	t.Helper()
	gate := "code_review"
	digest, err := gaterun.ComputeOperationDigest("B001", "bug", gate, gate, result)
	require.NoError(t, err)
	dir, err := gaterun.RunDir(root, run)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(dir, 0700))
	_, err = gaterun.CreateIdentity(dir, gaterun.RunIdentity{RunID: run, EntityKey: "B001", EntityType: "bug", SourceStatus: gate, Gate: gate, OperationDigest: digest})
	require.NoError(t, err)
	_, err = gaterun.CreateResult(dir, result)
	require.NoError(t, err)
	state := gaterun.NewOperationState(run, "B001", "bug", gate, gate, digest)
	state.AdmissionVersion = gatepersist.BoundAdmissionVersion
	require.NoError(t, state.MarkPersistenceComplete())
	require.NoError(t, state.MarkTransitionApplied())
	require.NoError(t, state.Save(dir))
	meta := string(mustJSON(t, bugApprovalNote{Gate: gate, Role: "success", Outcome: "pass", RunID: run, Digest: digest}))
	notes.items = append(notes.items, &models.EntityNote{ID: 2, Metadata: &meta})
	reason := gatepersist.BoundTransitionReason(gate, "pass", run, digest)
	history.items = append(history.items, &models.EntityHistory{ID: 2, FromStatus: &gate, ToStatus: "qa", Notes: &reason})
}
