package commands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/integration"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/require"
)

// This file drives the real, production `shark run` cascade path
// (runner.RunController.handleCascade, wired via cascadeIntegrationGuard in
// run_cascade_integration_guard.go and run.go) through an epic
// ActionCascade — the `shark run` counterpart to
// TestResolveCascadeCapturesEpicIntegrationBaseOnFirstFeatureDispatchOnly_TC011
// and TestResolveCascadeBlocksDispatchOnCaptureFailure_Finding1 in
// next_cascade_traversal_test.go.
//
// UAT round-3 rejection Finding 1 (docs/review/.../uat-20260905-203832-E34-F08.md)
// found that `shark run`'s cascade path had NO CaptureBase precondition at
// all — a repo-wide grep found no call to CaptureBase, resolveCascade, or
// resolveNext anywhere in run.go/runner/controller.go. These tests exist
// specifically to prove the production `shark run` wiring, not a mocked
// guard: nextCaptureEpicIntegrationBase is left at its production default
// (integration.CaptureBase) in both tests below.

// fakeCascadeChildrenService is a runner.CascadeChildrenService test double.
type fakeCascadeChildrenService struct {
	fn func(ctx context.Context, entityType, key string) (services.CascadeChildrenState, error)
}

func (f fakeCascadeChildrenService) DescribeDispatchableChildren(ctx context.Context, entityType, key string) (services.CascadeChildrenState, error) {
	return f.fn(ctx, entityType, key)
}

// stubbingAgentDispatcher fails the test if ever invoked: the epic-level
// cascade action never spawns an agent directly, so a production-path test
// exercising only the cascade stage must never reach a dispatcher.
type stubbingAgentDispatcher struct {
	t *testing.T
}

func (s stubbingAgentDispatcher) Dispatch(context.Context, runner.DispatchInput) (*runner.DispatchResult, error) {
	s.t.Fatal("stubbingAgentDispatcher.Dispatch should never be called by an epic-level cascade stage")
	return nil, nil
}
func (s stubbingAgentDispatcher) Name() string { return "stub" }
func (s stubbingAgentDispatcher) BuildCommand(runner.DispatchInput) (string, error) {
	return "", nil
}

// TestRunControllerCascadeCapturesEpicIntegrationBaseBeforeChildDispatch
// drives a real runner.RunController (the exact production ActionCascade
// path internal/cli/commands/run.go wires, IntegrationGuard included)
// through an epic cascade in a real temp git project, and asserts
// CaptureBase's `.shark/integration/<epic>/run.json` exists — with
// BaseCommit equal to the repo's real HEAD — before the (fake)
// DescribeDispatchableChildren call that would enumerate/dispatch children
// ever fires. Before the fix, RunController had no IntegrationGuard field at
// all: this test would fail to compile against the pre-fix controller.go,
// and even once the field exists but is left unwired (the exact drift TD-208
// tracked), runBeforeChildDispatch would be nil.
func TestRunControllerCascadeCapturesEpicIntegrationBaseBeforeChildDispatch(t *testing.T) {
	dir := t.TempDir()
	headCommit := initCascadeIntegrationGitRepo(t, dir)
	t.Chdir(dir)

	transitioner := keyedByEntityTransitioner{statuses: map[string]string{"E99": "active"}}
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			return &action.PopulatedAction{Action: "cascade", Instruction: "delegate"}, nil
		},
	}

	childrenLookupCalls := 0
	var runAtLookupTime *integration.IntegrationRun
	var runFileExistsAtLookupTime bool
	childrenSvc := fakeCascadeChildrenService{
		fn: func(_ context.Context, entityType, key string) (services.CascadeChildrenState, error) {
			childrenLookupCalls++
			run, err := integration.GetRun("E99")
			require.NoError(t, err)
			runAtLookupTime = run
			if _, statErr := os.Stat(filepath.Join(dir, ".shark", "integration", "E99", "run.json")); statErr == nil {
				runFileExistsAtLookupTime = true
			}
			// No dispatchable children — asserting the capture ordering does
			// not require an actual child dispatch.
			return services.CascadeChildrenState{}, nil
		},
	}

	runChildCalled := false
	runChild := func(ctx context.Context, childType, key string, opts runner.RunOptions) (*runner.RunResult, error) {
		runChildCalled = true
		return &runner.RunResult{EntityKey: key, Outcome: "completed"}, nil
	}

	ctrl, err := runner.NewRunController(runner.RunControllerDeps{
		Transitioner: transitioner,
		ActionSvc:    actionSvc,
		WorkflowSvc:  workflow.NewService(""),
		Dispatchers: map[string]runner.AgentDispatcher{
			"": stubbingAgentDispatcher{t: t},
		},
		ChildrenSvc:      childrenSvc,
		RunChild:         runChild,
		IntegrationGuard: cascadeIntegrationGuard{commandLabel: "run"},
	})
	require.NoError(t, err)

	// Before any dispatch: no run captured yet.
	preRun, err := integration.GetRun("E99")
	require.NoError(t, err)
	require.Nil(t, preRun, "no IntegrationRun should exist before the epic cascade ever runs")

	result, err := ctrl.Run(context.Background(), "E99", runner.RunOptions{EntityType: "epic"})
	require.NoError(t, err)

	require.Equal(t, 1, childrenLookupCalls, "DescribeDispatchableChildren must have been consulted exactly once")
	require.False(t, runChildCalled, "no children were dispatchable in this fixture")
	require.NotNil(t, runAtLookupTime, "the epic's IntegrationRun must already exist by the time cascade children are looked up")
	require.Equal(t, headCommit, runAtLookupTime.BaseCommit)
	require.True(t, runFileExistsAtLookupTime, ".shark/integration/E99/run.json must exist on disk before cascade child dispatch")
	require.NotEqual(t, "failed", result.Outcome, "a successful capture must not fail the cascade stage")

	postRun, err := integration.GetRun("E99")
	require.NoError(t, err)
	require.NotNil(t, postRun)
	require.Equal(t, headCommit, postRun.BaseCommit)
}

// TestRunControllerCascadeBlocksChildDispatchOnCaptureFailure is the `shark
// run` counterpart to
// TestResolveCascadeBlocksDispatchOnCaptureFailure_Finding1: when CaptureBase
// genuinely fails (a real "not a git repository" error, not a mocked one),
// the epic cascade must block entirely — DescribeDispatchableChildren and
// RunChild must never be called — and a durable, deduped epic-level
// `review-finding` note must record the failure, exactly as `shark next`
// already does.
func TestRunControllerCascadeBlocksChildDispatchOnCaptureFailure(t *testing.T) {
	dir := t.TempDir()
	// Deliberately no `git init`: CaptureBase's currentHeadCommit genuinely
	// fails against a non-git directory.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sharkconfig.json"), []byte("{}"), 0o644))
	t.Chdir(dir)

	recorder := &fakeCaptureFailureNoteRecorder{}
	originalRecorderFn := nextIntegrationCaptureFailureRecorder
	t.Cleanup(func() { nextIntegrationCaptureFailureRecorder = originalRecorderFn })
	nextIntegrationCaptureFailureRecorder = func(context.Context) (integration.NoteRecorder, error) {
		return recorder, nil
	}

	transitioner := keyedByEntityTransitioner{statuses: map[string]string{"E99": "active"}}
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			return &action.PopulatedAction{Action: "cascade", Instruction: "delegate"}, nil
		},
	}
	childrenLookupCalls := 0
	childrenSvc := fakeCascadeChildrenService{
		fn: func(_ context.Context, entityType, key string) (services.CascadeChildrenState, error) {
			childrenLookupCalls++
			return services.CascadeChildrenState{
				Children:            []services.CascadeChild{{Key: "E99-F01", EntityType: models.EntityTypeFeature}},
				TotalChildren:       1,
				NonTerminalChildren: 1,
			}, nil
		},
	}
	runChildCalled := false
	runChild := func(ctx context.Context, childType, key string, opts runner.RunOptions) (*runner.RunResult, error) {
		runChildCalled = true
		return &runner.RunResult{EntityKey: key, Outcome: "completed"}, nil
	}

	ctrl, err := runner.NewRunController(runner.RunControllerDeps{
		Transitioner: transitioner,
		ActionSvc:    actionSvc,
		WorkflowSvc:  workflow.NewService(""),
		Dispatchers: map[string]runner.AgentDispatcher{
			"": stubbingAgentDispatcher{t: t},
		},
		ChildrenSvc:      childrenSvc,
		RunChild:         runChild,
		IntegrationGuard: cascadeIntegrationGuard{commandLabel: "run"},
	})
	require.NoError(t, err)

	result, err := ctrl.Run(context.Background(), "E99", runner.RunOptions{EntityType: "epic"})
	require.NoError(t, err)

	require.Equal(t, 0, childrenLookupCalls, "DescribeDispatchableChildren must never be called after a capture failure")
	require.False(t, runChildCalled, "no cascade child may be dispatched after a capture failure")
	require.Equal(t, "failed", result.Outcome)
	require.NotEmpty(t, result.Error)
	require.Len(t, recorder.notes, 1, "exactly one durable capture-failure note must be recorded")

	var meta map[string]string
	require.NoError(t, json.Unmarshal([]byte(recorder.notes[0].metadata), &meta))
	require.Equal(t, "integration_capture", meta["gate"])
	require.Equal(t, "capture_base", meta["stage"])
	require.Equal(t, "open", meta["disposition"])

	// A second failing run (e.g. a harness retrying `shark run` before an
	// operator fixes the underlying condition) must still block dispatch and
	// must NOT accumulate a second note — this proves `shark run` shares
	// `shark next`'s dedupe, not a second independent implementation.
	result2, err := ctrl.Run(context.Background(), "E99", runner.RunOptions{EntityType: "epic"})
	require.NoError(t, err)
	require.Equal(t, "failed", result2.Outcome)
	require.Equal(t, 0, childrenLookupCalls)
	require.Len(t, recorder.notes, 1, "a persistently failing epic must accumulate exactly one open note, not one per shark run attempt")
}
