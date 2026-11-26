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
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/stretchr/testify/require"
)

// B13: the controller, not the worker or a host adapter, measures the triage
// actions behind a report-only withdrawal. Each trial feeds the resulting
// envelope to the same reader that admits the review.
type triageFixture struct {
	root      string
	git       func(...string) string
	scope     bugCandidateScope
	decisions map[bugFindingKey]bugReviewDecision
	envelope  []byte
}

func newTriageFixture(t *testing.T) triageFixture {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		require.NoError(t, err, string(out))
		return strings.TrimSpace(string(out))
	}
	git("init", "-b", "main")
	git("config", "user.name", "T")
	git("config", "user.email", "t@example.com")
	git("config", "commit.gpgsign", "false")
	require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("v1\n"), 0600))
	git("add", ".")
	git("commit", "-m", "candidate")
	identity := bugReviewIdentity{git("rev-parse", "HEAD"), git("rev-parse", "HEAD^{tree}"), "d", "p"}
	scope := bugCandidateScope{RuntimeProject: ".", CandidateCommit: identity.Commit, CandidateTree: identity.Tree, DiffSHA256: "d", RepairPlanSHA256: "p", ChangedFiles: reviewGitPaths{"file.txt"}}
	decision := bugReviewDecision{FindingID: "F-1", File: "file.txt", Line: 1, Rule: "STYLE", Summary: "s", Diagnosis: "d", Severity: "nit", Disposition: "report-only", Reason: "r"}
	save := func(name, kind string, v any) string {
		data, err := json.Marshal(v)
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".shark", "runs", "evidence"), 0700))
		require.NoError(t, os.WriteFile(filepath.Join(root, ".shark", "runs", "evidence", name), data, 0600))
		return fmt.Sprintf(`{"kind":%q,"pointer":".shark/runs/evidence/%s","summary":"sha256:%x"}`, kind, name, sha256.Sum256(data))
	}
	refs := save("scope.json", "bug_review_scope", scope) + "," + save("decisions.json", "bug_review_decisions", bugReviewDecisions{CandidateIdentity: identity, ReviewedFiles: reviewGitPaths{"file.txt"}, Findings: []bugReviewDecision{decision}})
	envelope := []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[%s],"gate_result":{"schema_version":1,"summary":"checked"}}`, refs))
	decisions := map[bugFindingKey]bugReviewDecision{{File: decision.File, Line: decision.Line, Rule: decision.Rule, Summary: decision.Summary, Diagnosis: decision.Diagnosis}: decision}

	return triageFixture{root, git, scope, decisions, envelope}
}

func TestObserveBugReportOnlyTriageIsParentMeasured(t *testing.T) {
	f := newTriageFixture(t)
	root, git, scope, decisions, envelope := f.root, f.git, f.scope, f.decisions, f.envelope
	run := func(name string, writes int, plant bool, dirty bool) error {
		runID := "obs-" + name
		if plant {
			dir := filepath.Join(root, ".shark", "runs", runID)
			require.NoError(t, os.MkdirAll(dir, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "parent-triage-observation.json"), []byte(`{}`), 0600))
		}
		if dirty {
			require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("edited\n"), 0600))
			t.Cleanup(func() { git("checkout", "--", "file.txt") })
		}
		attached, injected, err := ObserveBugReportOnlyTriage(context.Background(), BugTriageWindow{ProjectRoot: root, WorkingDir: root, RunID: runID, TrackerWrites: writes}, envelope)
		if err != nil {
			return err
		}
		require.True(t, injected)
		req := gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", runID), EnvelopeJSON: attached, ParentObservationInjected: true}
		return validateBugParentTriageObservation(req, scope, decisions)
	}

	require.NoError(t, run("legit", 0, false, false))
	require.ErrorContains(t, run("writes", 1, false, false), "tracker_writes of zero")
	require.ErrorContains(t, run("edits", 0, false, true), "edits of zero")
	require.ErrorContains(t, run("planted", 0, true, false), "already exists")

	// Counterfactual: without the parent's injection the same legitimate envelope is rejected.
	req := gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "none"), EnvelopeJSON: envelope}
	require.ErrorContains(t, validateBugParentTriageObservation(req, scope, decisions), "receipt is required")
}

// Drive the controller entry point itself. The admission callback applies the
// production reader to what the controller handed the coordinator.
func TestControllerInjectsMeasuredBugTriageObservation(t *testing.T) {
	f := newTriageFixture(t)
	for _, trial := range []struct {
		name    string
		writes  int
		plant   bool
		want    string
		disable bool
	}{
		{name: "legit"},
		{name: "removed-injection", disable: true, want: "receipt is required"},
		{name: "tracker-write", writes: 1, want: "tracker_writes of zero"},
		{name: "planted", plant: true, want: "already exists"},
	} {
		t.Run(trial.name, func(t *testing.T) {
			transitioner := &fakeTransitioner{status: map[string]string{"B001": "code_review"}}
			reader := &growingNoteReader{step: trial.writes}
			coordinator := gatepersist.NewCoordinator(&fakeNoteWriter{}, reader, fakeHistoryReader{},
				fakeStatusValidator{valid: map[string]bool{"code_review": true, "qa": true}}, transitioner, transitioner, &fakeLeaseReleaser{}, transitioner)
			coordinator.Admission = func(_ context.Context, req gatepersist.Request) error {
				return validateBugParentTriageObservation(req, f.scope, f.decisions)
			}
			controller, err := NewRunController(RunControllerDeps{
				Transitioner: &oneShotStatusTransitioner{}, ActionSvc: &MockActionService{}, WorkflowSvc: defaultWorkflowSvc(),
				Dispatchers: map[string]AgentDispatcher{"anthropic": &MockDispatcher{}},
				GateIngest:  &GateIngestDeps{Coordinator: coordinator, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}},
			})
			require.NoError(t, err)
			opts := RunOptions{ProjectRoot: f.root, WorkingDir: f.root, RunID: "ctl-" + trial.name, EntityType: "bug", SessionID: "s"}
			runID := gateStageRunID(opts.RunID, "B001", 1)
			if trial.plant {
				dir := filepath.Join(f.root, ".shark", "runs", runID)
				require.NoError(t, os.MkdirAll(dir, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "parent-triage-observation.json"), []byte(`{}`), 0600))
			}
			before := controller.countTriageNotes(context.Background(), opts, "B001", "code_review")
			require.Equal(t, 0, before)
			if trial.disable {
				before = -1
			}
			nextInfo := &services.NextStatusInfo{Outcomes: map[string]string{"pass": "qa"}}
			_, _, err = controller.ingestGateResultForDispatch(context.Background(), "B001", "code_review", nextInfo,
				&config.PopulatedAction{Provider: "anthropic"}, opts, &DispatchResult{Stdout: string(f.envelope), Duration: time.Millisecond}, new(bool), 1, before)
			if trial.want != "" {
				require.ErrorContains(t, err, trial.want)
				return
			}
			require.NoError(t, err)
		})
	}
}

// growingNoteReader reports step more notes on every call after the first.
type growingNoteReader struct{ step, calls int }

func (r *growingNoteReader) ListNotes(context.Context, models.EntityType, string, []string) ([]*models.EntityNote, error) {
	r.calls++
	if r.calls == 1 {
		return nil, nil
	}
	return make([]*models.EntityNote, r.step), nil
}

// B09: expected host actions come from the raw angle findings. A raw blocker
// relabeled as a triaged non-blocker (not refuted, not report-only) cannot
// suppress its host action, and the parent-measured observation covers it.
func TestHostActionsAreDerivedFromRawFindings(t *testing.T) {
	root := t.TempDir()
	scope := bugCandidateScope{CandidateCommit: "c", CandidateTree: "t", DiffSHA256: "d", RepairPlanSHA256: "p"}
	key := bugFindingKey{File: "a.go", Line: 4, Rule: "r", Summary: "s", Diagnosis: "d"}
	relabeled := map[bugFindingKey]bugReviewDecision{key: {FindingID: "F-1", Severity: "non-blocker", Disposition: "accepted"}}
	raw := map[bugFindingKey]string{key: "blocker"}
	runDir := filepath.Join(root, ".shark", "runs", "review")
	require.NoError(t, os.MkdirAll(runDir, 0o700))
	write := func(actions []map[string]any) gatepersist.Request {
		data := mustJSON(t, map[string]any{"candidate_identity": bugReviewIdentity{"c", "t", "d", "p"}, "observed_by": "parent", "actions": actions})
		require.NoError(t, os.WriteFile(filepath.Join(runDir, "parent-triage-observation.json"), data, 0o600))
		envelope := fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_parent_triage_observation","pointer":".shark/runs/review/parent-triage-observation.json","summary":"sha256:%x"}],"gate_result":{"schema_version":1,"summary":"x"}}`, sha256.Sum256(data))
		return gatepersist.Request{RunDir: runDir, EnvelopeJSON: []byte(envelope), ParentObservationInjected: true}
	}
	triage := func(edits any) []map[string]any {
		return []map[string]any{{"finding_id": "F-1", "action": "triage", "edits": edits, "tracker_writes": 1}}
	}
	// Without the raw findings the relabeled blocker needs no observation (the defect).
	require.NoError(t, validateBugParentTriageObservationRaw(gatepersist.Request{RunDir: runDir}, scope, relabeled, nil))
	// With them, the observation is required, per finding, with non-null counters.
	require.ErrorContains(t, validateBugParentTriageObservationRaw(gatepersist.Request{RunDir: runDir}, scope, relabeled, raw), "receipt is required")
	require.ErrorContains(t, validateBugParentTriageObservationRaw(write(nil), scope, relabeled, raw), "missing triage observation for F-1")
	require.ErrorContains(t, validateBugParentTriageObservationRaw(write(triage(nil)), scope, relabeled, raw), "triage requires observed edits")
	require.ErrorContains(t, validateBugParentTriageObservationRaw(write([]map[string]any{{"finding_id": "F-1", "action": "report_only", "edits": 0, "tracker_writes": 0}}), scope, relabeled, raw), "expected triage host action")
	require.NoError(t, validateBugParentTriageObservationRaw(write(triage(2)), scope, relabeled, raw))
	// A refuted raw blocker needs no host action.
	refuted := map[bugFindingKey]bugReviewDecision{key: {FindingID: "F-1", Severity: "blocker", Disposition: "refuted"}}
	require.NoError(t, validateBugParentTriageObservationRaw(gatepersist.Request{RunDir: runDir}, scope, refuted, raw))
}

// B088 R4: an edit that is later restored leaves no net change, so only an
// attempt measured in this run's dispatch window can disqualify a withdrawal.
// Each trial feeds the parent's observation to the production admission reader.
func TestObservedEditAttemptsBindHostActionToRunAndFinding(t *testing.T) {
	f := newTriageFixture(t)
	root, scope, decisions, envelope := f.root, f.scope, f.decisions, f.envelope
	observe := func(runID string, started time.Time, touch func()) (gatepersist.Request, error) {
		touch()
		attached, injected, err := ObserveBugReportOnlyTriage(context.Background(),
			BugTriageWindow{ProjectRoot: root, WorkingDir: root, RunID: runID, Started: started}, envelope)
		if err != nil {
			return gatepersist.Request{}, err
		}
		require.True(t, injected)
		return gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", runID), EnvelopeJSON: attached, ParentObservationInjected: true}, nil
	}
	started := time.Now().Add(-time.Second)
	file := filepath.Join(root, "file.txt")

	req, err := observe("untouched", started, func() {
		// Predate the window so the unmodified checkout has zero attempts.
		require.NoError(t, os.Chtimes(file, started.Add(-time.Hour), started.Add(-time.Hour)))
	})
	require.NoError(t, err)
	require.NoError(t, validateBugParentTriageObservation(req, scope, decisions))

	req, err = observe("restored", started, func() {
		require.NoError(t, os.WriteFile(file, []byte("edited\n"), 0600))
		require.NoError(t, os.WriteFile(file, []byte("v1\n"), 0600))
	})
	require.NoError(t, err)
	// The byte-identical checkout has no net edit, so the counters alone accept it.
	counters, err := observe("restored-counters", time.Time{}, func() {})
	require.NoError(t, err)
	require.NoError(t, validateBugParentTriageObservation(counters, scope, decisions))
	require.ErrorContains(t, validateBugParentTriageObservation(req, scope, decisions), "F-1: report-only requires observed edit_attempts of zero")

	// A genuine receipt copied under another run's path belongs to its own run.
	genuine, err := observe("genuine", started, func() {
		require.NoError(t, os.Chtimes(file, started.Add(-time.Hour), started.Add(-time.Hour)))
	})
	require.NoError(t, err)
	// Moving the file and rewriting the pointer keeps the bytes and digest valid, so
	// only the run named inside the observation can detect it.
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".shark", "runs", "other"), 0700))
	copied, err := os.ReadFile(filepath.Join(genuine.RunDir, "parent-triage-observation.json"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".shark", "runs", "other", "parent-triage-observation.json"), copied, 0600))
	other := genuine
	other.RunDir = filepath.Join(root, ".shark", "runs", "other")
	other.EnvelopeJSON = []byte(strings.ReplaceAll(string(genuine.EnvelopeJSON), ".shark/runs/genuine/", ".shark/runs/other/"))
	require.ErrorContains(t, validateBugParentTriageObservation(other, scope, decisions), `belongs to review run "genuine", not "other"`)
}
