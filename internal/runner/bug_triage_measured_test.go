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
	"github.com/jwwelbor/shark-task-manager/internal/trackerattempts"
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
			_, err = trackerAttemptEnv(f.root, runID)
			require.NoError(t, err)
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

// B088 R4: a tracker write rejected before it mutates leaves no note, so only
// the run's attempt ledger can disqualify a report-only withdrawal. Each trial
// feeds the parent's observation to the production admission reader.
func TestTrackerAttemptLedgerBindsWithdrawalToRunAndFinding(t *testing.T) {
	f := newTriageFixture(t)
	root, scope, decisions, envelope := f.root, f.scope, f.decisions, f.envelope
	observe := func(runID string, attempts ...trackerattempts.Entry) gatepersist.Request {
		env, err := trackerAttemptEnv(root, runID)
		require.NoError(t, err)
		path := strings.TrimPrefix(env[0], trackerattempts.EnvLog+"=")
		for _, entry := range attempts {
			require.NoError(t, trackerattempts.Append(path, runID, entry.Command, entry.Argv, entry.Write))
		}
		attached, injected, err := ObserveBugReportOnlyTriage(context.Background(),
			BugTriageWindow{ProjectRoot: root, WorkingDir: root, RunID: runID, RequireLedger: true}, envelope)
		require.NoError(t, err)
		require.True(t, injected)
		return gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", runID), EnvelopeJSON: attached, ParentObservationInjected: true}
	}
	rejected := trackerattempts.Entry{Command: "create note", Argv: []string{"create", "note", "B001", "--bogus"}, Write: true}

	require.NoError(t, validateBugParentTriageObservation(observe("clean", trackerattempts.Entry{Command: "get", Argv: []string{"get", "B001"}}), scope, decisions))

	// The attempt created no note and no edit, so every net counter is zero.
	req := observe("rejected", rejected)
	require.ErrorContains(t, validateBugParentTriageObservation(req, scope, decisions), "report-only requires zero attempted tracker writes, ledger has 1")

	require.ErrorContains(t, validateBugParentTriageObservation(observe("own", trackerattempts.Entry{Command: "create note", Argv: []string{"create", "note", "B001", "F-1 withdrawn"}, Write: true}), scope, decisions), "ledger has 1")

	// Another finding's own filing never disqualifies F-1's withdrawal, but an
	// unnamed write does when no surviving finding could have made it.
	ledger := trackerattempts.Ledger{Entries: []trackerattempts.Entry{{Argv: []string{"create", "note", "F-2 needs filing"}, Write: true}, {Argv: []string{"create", "note", "unnamed"}, Write: true}}}
	withdrawn := bugReviewDecision{FindingID: "F-1", Disposition: "report-only"}
	survivor := bugReviewDecision{FindingID: "F-2", Disposition: "triaged"}
	require.Equal(t, 0, findingWriteAttempts(ledger, "F-1", []bugReviewDecision{withdrawn, survivor}))
	require.Equal(t, 1, findingWriteAttempts(ledger, "F-1", []bugReviewDecision{withdrawn, {FindingID: "F-2", Disposition: "refuted"}}))

	// Falsifying the count to zero while the ledger holds the attempt is detected,
	// even with the observation's own digest rebound in the envelope.
	falsified := observe("falsified", rejected)
	path := filepath.Join(falsified.RunDir, "parent-triage-observation.json")
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(before, &doc))
	doc["actions"].([]any)[0].(map[string]any)["tracker_attempts"] = 0
	after, err := json.Marshal(doc)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, after, 0o600))
	falsified.EnvelopeJSON = []byte(strings.ReplaceAll(string(falsified.EnvelopeJSON), fmt.Sprintf("%x", sha256.Sum256(before)), fmt.Sprintf("%x", sha256.Sum256(after))))
	require.ErrorContains(t, validateBugParentTriageObservation(falsified, scope, decisions), "tracker_attempts 0 does not match 1 attempts in the run ledger")

	// A ledger altered after observation no longer matches its recorded digest.
	tampered := observe("tampered")
	require.NoError(t, validateBugParentTriageObservation(tampered, scope, decisions))
	require.NoError(t, trackerattempts.Append(filepath.Join(tampered.RunDir, trackerattempts.LedgerName), "tampered", rejected.Command, rejected.Argv, true))
	require.ErrorContains(t, validateBugParentTriageObservation(tampered, scope, decisions), "ledger changed after the parent observed it")

	// An entry bound to another run is rejected when the parent reads the ledger.
	env, err := trackerAttemptEnv(root, "foreign")
	require.NoError(t, err)
	require.NoError(t, trackerattempts.Append(strings.TrimPrefix(env[0], trackerattempts.EnvLog+"="), "other-run", "create note", rejected.Argv, true))
	_, _, err = ObserveBugReportOnlyTriage(context.Background(), BugTriageWindow{ProjectRoot: root, WorkingDir: root, RunID: "foreign", RequireLedger: true}, envelope)
	require.ErrorContains(t, err, `belongs to run "other-run", not "foreign"`)

	// A missing ledger is an error, never "no attempts".
	_, _, err = ObserveBugReportOnlyTriage(context.Background(), BugTriageWindow{ProjectRoot: root, WorkingDir: root, RunID: "never-created", RequireLedger: true}, envelope)
	require.ErrorContains(t, err, "ledger unavailable")
}

// B088 R4 live host evidence: replays a ledger written by a real host's shark
// invocations through the parent observation and the production admission
// reader. It runs only when B088_LIVE_LEDGER names that ledger and
// B088_LIVE_EXPECT is "accept" or "reject"; the run id is the ledger's own.
func TestLiveHostLedgerAdmission(t *testing.T) {
	path, expect := os.Getenv("B088_LIVE_LEDGER"), os.Getenv("B088_LIVE_EXPECT")
	if path == "" {
		t.Skip("B088_LIVE_LEDGER not set")
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var first trackerattempts.Entry
	require.NoError(t, json.Unmarshal([]byte(strings.SplitN(string(data), "\n", 2)[0]), &first))
	runID := first.RunID

	f := newTriageFixture(t)
	env, err := trackerAttemptEnv(f.root, runID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(strings.TrimPrefix(env[0], trackerattempts.EnvLog+"="), data, 0o600))
	attached, injected, err := ObserveBugReportOnlyTriage(context.Background(),
		BugTriageWindow{ProjectRoot: f.root, WorkingDir: f.root, RunID: runID, RequireLedger: true}, f.envelope)
	require.NoError(t, err)
	require.True(t, injected)
	req := gatepersist.Request{RunDir: filepath.Join(f.root, ".shark", "runs", runID), EnvelopeJSON: attached, ParentObservationInjected: true}
	admission := validateBugParentTriageObservation(req, f.scope, f.decisions)
	observation, err := os.ReadFile(filepath.Join(req.RunDir, "parent-triage-observation.json"))
	require.NoError(t, err)
	t.Logf("observation: %s", observation)
	t.Logf("admission: %v", admission)
	if expect == "reject" {
		require.ErrorContains(t, admission, "report-only requires zero attempted tracker writes")
		return
	}
	require.NoError(t, admission)
}
