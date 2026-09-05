package commands

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/integration"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/stretchr/testify/require"
)

type keyedByEntityTransitioner struct {
	statuses map[string]string
}

func (t keyedByEntityTransitioner) GetNextStatus(_ context.Context, key string) (*services.NextStatusInfo, error) {
	return &services.NextStatusInfo{CurrentStatus: t.statuses[key]}, nil
}

func (t keyedByEntityTransitioner) TransitionStatus(context.Context, string, string, services.TransitionOptions) (*services.TransitionResult, error) {
	return &services.TransitionResult{}, nil
}

// TestResolveNextTraversesMultiLevelCascadeAndRecordsResolvedVia pins the
// 0e3f0103 keyed `shark next <epic>` contract this split restores: an epic at
// a cascade step traverses through a feature (also at a cascade step) down to
// the first dispatchable task, returning that task's concrete dispatch
// response with both parents recorded in resolved_via, in traversal order.
func TestResolveNextTraversesMultiLevelCascadeAndRecordsResolvedVia(t *testing.T) {
	stubNoEpicIntegrationCapture(t)
	originalDescribe := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = originalDescribe }()

	planDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
		switch {
		case entityType == "epic" && key == "E01":
			return services.PlanHierarchyChildrenState{
				Children:            []services.PlanHierarchyChild{{Key: "E01-F01", EntityType: models.EntityTypeFeature}},
				TotalChildren:       1,
				NonTerminalChildren: 1,
			}, nil
		case entityType == "feature" && key == "E01-F01":
			return services.PlanHierarchyChildrenState{
				Children:            []services.PlanHierarchyChild{{Key: "T-E01-F01-001", EntityType: models.EntityTypeTask}},
				TotalChildren:       1,
				NonTerminalChildren: 1,
			}, nil
		default:
			return services.PlanHierarchyChildrenState{}, nil
		}
	}

	transitioner := keyedByEntityTransitioner{statuses: map[string]string{
		"E01":           "active",
		"E01-F01":       "active",
		"T-E01-F01-001": "in_progress",
	}}
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			if status == "active" {
				return &action.PopulatedAction{Action: "cascade", Instruction: "delegate"}, nil
			}
			return &action.PopulatedAction{
				Action: "spawn_agent", AgentType: "backend", Provider: "anthropic", Model: "sonnet",
				Instruction: "implement the task",
			}, nil
		},
	}
	cache := &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"epic": {
				transitioner: transitioner,
				generator:    fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc:    actionSvc,
			},
			"feature": {
				transitioner: transitioner,
				generator:    fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc:    actionSvc,
			},
			"task": {
				transitioner: transitioner,
				generator:    fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc:    actionSvc,
			},
		},
		actionSvcRoot: actionSvc,
	}

	resp, err := resolveNext(context.Background(), cache, "epic", "E01", 0)
	require.NoError(t, err)
	require.Equal(t, "T-E01-F01-001", resp.EntityKey)
	require.Equal(t, "spawn_agent", resp.Action)
	require.Equal(t, []string{"E01", "E01-F01"}, resp.ResolvedVia)
}

// TC-103: cascade resolution must treat a ready Question as parked before
// responder rendering, then leave its parent paused rather than surfacing the
// child as a broken worker dispatch.
func TestResolveNextCascadeSkipsReadyQuestionBeforeResponderRendering_TC103(t *testing.T) {
	originalDescribe := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = originalDescribe }()
	planDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
		if entityType == "feature" && key == "E01-F01" {
			return services.PlanHierarchyChildrenState{
				Children: []services.PlanHierarchyChild{{Key: "Q001", EntityType: models.EntityTypeQuestion}}, TotalChildren: 1, NonTerminalChildren: 1,
			}, nil
		}
		return services.PlanHierarchyChildrenState{}, nil
	}
	questionPlaceholderCalls := 0
	questionActionCalls := 0
	cache := &nextAdapterCache{entries: map[string]*nextAdapters{
		"feature": {
			transitioner: keyedByEntityTransitioner{statuses: map[string]string{"E01-F01": "active"}},
			generator:    fixedNextPlaceholders{vars: map[string]string{}},
			actionSvc: &action.MockActionService{GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
				if status != "active" {
					t.Fatalf("parent action status = %q", status)
				}
				return &action.PopulatedAction{Action: "cascade", Instruction: "delegate"}, nil
			}},
		},
		"question": {
			transitioner: nextStatusOnlyTransitioner{next: &services.NextStatusInfo{EntityType: models.EntityTypeQuestion, EntityKey: "Q001", CurrentStatus: "ready_for_resolution", IsTerminal: true}},
			generator: runnerPlaceholderFunc(func(context.Context, string) (map[string]string, error) {
				questionPlaceholderCalls++
				return nil, context.Canceled
			}),
			actionSvc: &action.MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
				questionActionCalls++
				return nil, context.Canceled
			}},
		},
	}}

	resp, err := resolveNext(context.Background(), cache, "feature", "E01-F01", 0)
	if err != nil {
		t.Fatalf("resolveNext(cascade ready Question) error = %v", err)
	}
	if resp.EntityKey != "E01-F01" || resp.Action != action.ActionPause || resp.Prompt != "" {
		t.Fatalf("resolveNext(cascade ready Question) = %#v, want paused parent", resp)
	}
	if questionPlaceholderCalls != 0 || questionActionCalls != 0 {
		t.Fatalf("cascade ready Question rendered placeholders=%d actions=%d, want neither", questionPlaceholderCalls, questionActionCalls)
	}
}

// initCascadeIntegrationGitRepo initializes a minimal real git repository at
// dir with one commit and returns that commit's full hash. This mirrors
// internal/integration's own run_test.go helper (initTestGitRepo /
// chdirProjectRoot) — duplicated here rather than imported because it is a
// small, unexported test helper in a different package: CaptureBase (via
// this test's real resolveCascade drive) resolves its project root by
// walking up from the process's working directory, so a real git repo is
// required for TC-011's "no mock CaptureBase" Caller-Path Contract.
func initCascadeIntegrationGitRepo(t *testing.T, dir string) string {
	t.Helper()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("seed"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	run("add", "seed.txt")
	run("commit", "-q", "-m", "seed")

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// TestResolveCascadeCapturesEpicIntegrationBaseOnFirstFeatureDispatchOnly_TC011
// implements test-plan.md TC-011: drives the real cascade dispatch path
// (resolveNext -> resolveEntity -> entityResolutionStrategy.resolveCascade)
// against a temp, real git-backed `.shark/` project root and a fixture
// epic/feature pair — the epic's `active` step's cascade action must call
// integration.CaptureBase itself (this test never calls CaptureBase
// directly), and a second feature's dispatch must be a no-op that leaves
// the persisted run record unchanged, per REQ-F-004 / task T-E34-F08-008
// AC-T1.
//
// nextCaptureEpicIntegrationBase is deliberately left at its production
// default (integration.CaptureBase) here — overriding it, as the other
// cascade tests in this package do, would defeat the entire point of this
// test.
func TestResolveCascadeCapturesEpicIntegrationBaseOnFirstFeatureDispatchOnly_TC011(t *testing.T) {
	dir := t.TempDir()
	headCommit := initCascadeIntegrationGitRepo(t, dir)
	t.Chdir(dir)

	originalDescribe := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = originalDescribe }()

	// The first `shark next E99` call sees only the epic's first feature as
	// dispatchable; the second call (simulating the harness dispatching
	// again once F01 has been claimed elsewhere) sees only the second.
	dispatchCall := 0
	planDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
		if entityType != "epic" || key != "E99" {
			return services.PlanHierarchyChildrenState{}, nil
		}
		dispatchCall++
		if dispatchCall == 1 {
			return services.PlanHierarchyChildrenState{
				Children:            []services.PlanHierarchyChild{{Key: "E99-F01", EntityType: models.EntityTypeFeature}},
				TotalChildren:       2,
				NonTerminalChildren: 2,
			}, nil
		}
		return services.PlanHierarchyChildrenState{
			Children:            []services.PlanHierarchyChild{{Key: "E99-F02", EntityType: models.EntityTypeFeature}},
			TotalChildren:       2,
			NonTerminalChildren: 2,
		}, nil
	}

	transitioner := keyedByEntityTransitioner{statuses: map[string]string{
		"E99":     "active",
		"E99-F01": "in_progress",
		"E99-F02": "in_progress",
	}}
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			if status == "active" {
				return &action.PopulatedAction{Action: "cascade", Instruction: "delegate"}, nil
			}
			return &action.PopulatedAction{
				Action: "spawn_agent", AgentType: "backend", Provider: "anthropic", Model: "sonnet",
				Instruction: "implement the feature",
			}, nil
		},
	}
	cache := &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"epic": {
				transitioner: transitioner,
				generator:    fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc:    actionSvc,
			},
			"feature": {
				transitioner: transitioner,
				generator:    fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc:    actionSvc,
			},
		},
		actionSvcRoot: actionSvc,
	}

	// Before any dispatch: no run captured yet.
	preRun, err := integration.GetRun("E99")
	require.NoError(t, err)
	require.Nil(t, preRun, "no IntegrationRun should exist before the epic's cascade ever dispatches a feature")

	resp1, err := resolveNext(context.Background(), cache, "epic", "E99", 0)
	require.NoError(t, err)
	require.Equal(t, "E99-F01", resp1.EntityKey)
	require.Equal(t, "spawn_agent", resp1.Action)

	run1, err := integration.GetRun("E99")
	require.NoError(t, err)
	require.NotNil(t, run1, "the first feature dispatch must have captured the epic's IntegrationRun")
	require.Equal(t, headCommit, run1.BaseCommit)
	require.NotEmpty(t, run1.EpicRunID)

	resp2, err := resolveNext(context.Background(), cache, "epic", "E99", 0)
	require.NoError(t, err)
	require.Equal(t, "E99-F02", resp2.EntityKey)
	require.Equal(t, "spawn_agent", resp2.Action)

	run2, err := integration.GetRun("E99")
	require.NoError(t, err)
	require.NotNil(t, run2)
	require.Equal(t, run1.EpicRunID, run2.EpicRunID, "a second feature's dispatch must not create a second run")
	require.Equal(t, run1.BaseCommit, run2.BaseCommit, "BaseCommit must never be recomputed after the first capture")
}
