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
	decisions := map[bugFindingKey]bugReviewDecision{{decision.File, decision.Line, decision.Rule, decision.Summary, decision.Diagnosis}: decision}

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
