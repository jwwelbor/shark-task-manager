package commands

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/spf13/cobra"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/integration"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/require"
)

// Fan-out cascade tests. These pin the behavior that makes parallel dispatch
// possible at all: `shark next <key>` must stop at a fork and hand back the
// tied candidate tier instead of silently picking children[0]. Without this,
// the rider is never shown a set, so it can never fan out.

func fanoutIntPtr(v int) *int { return &v }

// fanoutChild builds a dispatchable child at the given execution order.
func fanoutChild(key string, entityType models.EntityType, order int) services.PlanHierarchyChild {
	return services.PlanHierarchyChild{
		Key:            key,
		Title:          key + " title",
		Status:         "todo",
		EntityType:     entityType,
		ExecutionOrder: fanoutIntPtr(order),
	}
}

// fanoutCache wires an adapter cache whose statuses come from the supplied map
// and whose action verb is "cascade" for parents and per-status for leaves.
func fanoutCache(
	t *testing.T,
	statuses map[string]string,
	actionForStatus func(status string) *action.PopulatedAction,
) *nextAdapterCache {
	t.Helper()
	transitioner := keyedByEntityTransitioner{statuses: statuses}
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			return actionForStatus(status), nil
		},
	}
	entries := map[string]*nextAdapters{}
	for _, entityType := range []string{"epic", "feature", "task"} {
		entries[entityType] = &nextAdapters{
			transitioner: transitioner,
			generator:    fixedNextPlaceholders{vars: map[string]string{}},
			actionSvc:    actionSvc,
		}
	}
	// Pin the parallel cap. Without this the fork path calls the production
	// closure, which reads the checked-in .sharkconfig.json from disk — adding
	// a max_parallel_items key to the project's own config would then turn
	// these tests red for reasons unrelated to fork shape.
	stubMaxParallelItems(t, 5)
	// This file's fixtures use "epic" as an entity type and "active" as a
	// cascade status, which is exactly what resolveCascade's REQ-F-004 guard
	// (T-E34-F08-008) matches on. Left at its production default, every
	// epic-cascade test here would call the real integration.CaptureBase
	// against this process's actual working directory. Fan-out shape is
	// this file's subject, not integration-run capture, so stub it to a
	// no-op.
	stubNoEpicIntegrationCapture(t)
	return &nextAdapterCache{entries: entries, actionSvcRoot: actionSvc, surfaceForks: true}
}

// stubNoEpicIntegrationCapture points resolveCascade's REQ-F-004 wiring hook
// (nextCaptureEpicIntegrationBase, next.go) at a no-op for the duration of
// one test, so cascade tests unrelated to E34-F08's integration-run capture
// never touch a real git repository or the real .shark/ directory.
func stubNoEpicIntegrationCapture(t *testing.T) {
	t.Helper()
	original := nextCaptureEpicIntegrationBase
	t.Cleanup(func() { nextCaptureEpicIntegrationBase = original })
	nextCaptureEpicIntegrationBase = func(string) (*integration.IntegrationRun, error) { return nil, nil }
}

// stubMaxParallelItems pins the configured fan-out cap for one test.
func stubMaxParallelItems(t *testing.T, limit int) {
	t.Helper()
	original := planGetMaxParallelItems
	t.Cleanup(func() { planGetMaxParallelItems = original })
	planGetMaxParallelItems = func() int { return limit }
}

// stubForkEdges installs a no-op edge loader for tests whose subject is fork
// *shape* rather than edge content. CLI tests must never reach the real
// service, and the fork path fails loudly on an edge-load error, so a test
// that forks without this stub would fail against the live database.
func stubForkEdges(t *testing.T) {
	t.Helper()
	original := fanoutDescribeCandidateEdges
	t.Cleanup(func() { fanoutDescribeCandidateEdges = original })
	fanoutDescribeCandidateEdges = func(_ context.Context, _ string, _ []string) (map[string]services.PlanHierarchyEdges, error) {
		return map[string]services.PlanHierarchyEdges{}, nil
	}
}

// cascadeOrDispatch is the common action mapping: "active" is a cascade step,
// anything else dispatches an agent.
func cascadeOrDispatch(status string) *action.PopulatedAction {
	if status == "active" {
		return &action.PopulatedAction{Action: "cascade", Instruction: "delegate"}
	}
	return &action.PopulatedAction{
		Action: "spawn_agent", AgentType: "backend", Provider: "anthropic",
		Model: "sonnet", Instruction: "implement the task",
	}
}

// TestFanoutStopsAtForkInsteadOfPickingFirstChild is the core contract: two
// tasks tied at the same execution order must come back as a candidate tier,
// not as a dispatch of the first one. If this regresses, the rider silently
// loses every opportunity to parallelize.
func TestFanoutStopsAtForkInsteadOfPickingFirstChild(t *testing.T) {
	original := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = original }()

	planDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
		if entityType == "feature" && key == "E01-F01" {
			return services.PlanHierarchyChildrenState{
				Children: []services.PlanHierarchyChild{
					fanoutChild("T-E01-F01-001", models.EntityTypeTask, 1),
					fanoutChild("T-E01-F01-002", models.EntityTypeTask, 1),
				},
				TotalChildren:       2,
				NonTerminalChildren: 2,
			}, nil
		}
		return services.PlanHierarchyChildrenState{}, nil
	}

	// Edges are decision-support for the rider: without them a fork looks
	// dependency-free on the wire, which reads as "safe to run all of these
	// concurrently". Assert they are actually attached.
	originalEdges := fanoutDescribeCandidateEdges
	defer func() { fanoutDescribeCandidateEdges = originalEdges }()
	var edgeEntityType string
	var edgeKeys []string
	fanoutDescribeCandidateEdges = func(_ context.Context, entityType string, keys []string) (map[string]services.PlanHierarchyEdges, error) {
		edgeEntityType, edgeKeys = entityType, keys
		return map[string]services.PlanHierarchyEdges{
			"T-E01-F01-001": {
				DependsOn: []services.PlanHierarchyEdge{{Key: "T-E01-F01-000", Status: "completed", Type: "depends_on"}},
				Blocks:    []services.PlanHierarchyEdge{{Key: "T-E01-F01-009", Status: "todo", Type: "blocks"}},
				Warnings: []services.PlanHierarchyEdgeWarning{{
					Code:             services.PlanHierarchyWarningDanglingRelationship,
					Direction:        "incoming",
					RelationshipID:   77,
					EndpointType:     models.EntityTypeTask,
					EndpointID:       404,
					RelationshipType: models.EntityRelRelatedTo,
				}},
			},
		}, nil
	}

	cache := fanoutCache(t, map[string]string{"E01-F01": "active"}, cascadeOrDispatch)

	resp, err := resolveNext(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.NotNil(t, resp.selection, "a 2-child tie must return a selection, not a dispatch")
	require.Empty(t, resp.Prompt, "a fork is a selection surface and carries no worker prompt")

	require.Equal(t, "task", edgeEntityType)
	require.Equal(t, []string{"T-E01-F01-001", "T-E01-F01-002"}, edgeKeys,
		"edges must be requested for every candidate in the fork")
	require.Equal(t,
		[]CandidateEdge{{Key: "T-E01-F01-000", Status: "completed", Type: "depends_on"}},
		resp.selection.Entities[0].DependsOn)
	require.Equal(t,
		[]CandidateEdge{{Key: "T-E01-F01-009", Status: "todo", Type: "blocks"}},
		resp.selection.Entities[0].Blocks)
	require.Equal(t, []CandidateEdgeWarning{{
		Code:             services.PlanHierarchyWarningDanglingRelationship,
		Direction:        "incoming",
		RelationshipID:   77,
		EndpointType:     "task",
		EndpointID:       404,
		RelationshipType: "related_to",
	}}, resp.selection.Entities[0].Warnings)
	require.Nil(t, resp.selection.Entities[1].DependsOn,
		"a candidate with no edges stays edge-less rather than being zeroed")

	sel := resp.selection
	require.Equal(t, "hierarchy_selection", sel.Mode)
	require.Equal(t, "parallel_candidates", sel.Action)
	require.Equal(t, "parallel_tie", sel.SelectionReason)
	require.Equal(t, "available", sel.ParallelExecution)
	require.Equal(t, "E01-F01", sel.RootKey)
	require.Equal(t, []string{"E01-F01"}, sel.ResolvedVia)
	require.Len(t, sel.Entities, 2)
	require.Equal(t, "T-E01-F01-001", sel.Entities[0].EntityKey)
	require.Equal(t, "T-E01-F01-002", sel.Entities[1].EntityKey)
}

// TestFanoutDrillsThroughSingleOptionTiersThenForks verifies the "cascade
// until it forks" rule across levels: a single-child epic tier is drilled
// through silently, and resolved_via records the full path to the fork so the
// rider can audit which entities were traversed.
func TestFanoutDrillsThroughSingleOptionTiersThenForks(t *testing.T) {
	stubForkEdges(t)
	original := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = original }()

	planDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
		switch {
		case entityType == "epic" && key == "E01":
			return services.PlanHierarchyChildrenState{
				Children:            []services.PlanHierarchyChild{fanoutChild("E01-F01", models.EntityTypeFeature, 1)},
				TotalChildren:       1,
				NonTerminalChildren: 1,
			}, nil
		case entityType == "feature" && key == "E01-F01":
			return services.PlanHierarchyChildrenState{
				Children: []services.PlanHierarchyChild{
					fanoutChild("T-E01-F01-001", models.EntityTypeTask, 2),
					fanoutChild("T-E01-F01-002", models.EntityTypeTask, 2),
					fanoutChild("T-E01-F01-003", models.EntityTypeTask, 2),
				},
				TotalChildren:       3,
				NonTerminalChildren: 3,
			}, nil
		}
		return services.PlanHierarchyChildrenState{}, nil
	}

	cache := fanoutCache(t, map[string]string{
		"E01":     "active",
		"E01-F01": "active",
	}, cascadeOrDispatch)

	resp, err := resolveNext(context.Background(), cache, "epic", "E01", 0)
	require.NoError(t, err)
	require.NotNil(t, resp.selection)
	require.Equal(t, "E01-F01", resp.selection.RootKey, "the fork is rooted at the forking parent, not the entry key")
	require.Equal(t, []string{"E01", "E01-F01"}, resp.selection.ResolvedVia)
	require.Len(t, resp.selection.Entities, 3)
}

// TestFanoutDoesNotForkOnDistinctExecutionOrders guards the tie-tiering rule:
// sequenced work (orders 1,2,3) is NOT a parallel opportunity, so it must
// still drill to a single dispatch. Forking here would tell the rider to run
// work concurrently that its author explicitly sequenced.
func TestFanoutDoesNotForkOnDistinctExecutionOrders(t *testing.T) {
	original := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = original }()

	planDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
		if entityType == "feature" && key == "E01-F01" {
			return services.PlanHierarchyChildrenState{
				Children: []services.PlanHierarchyChild{
					fanoutChild("T-E01-F01-001", models.EntityTypeTask, 1),
					fanoutChild("T-E01-F01-002", models.EntityTypeTask, 2),
					fanoutChild("T-E01-F01-003", models.EntityTypeTask, 3),
				},
				TotalChildren:       3,
				NonTerminalChildren: 3,
			}, nil
		}
		return services.PlanHierarchyChildrenState{}, nil
	}

	cache := fanoutCache(t, map[string]string{
		"E01-F01":       "active",
		"T-E01-F01-001": "todo",
	}, cascadeOrDispatch)

	resp, err := resolveNext(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.Nil(t, resp.selection, "distinct execution orders are sequenced work, not a fork")
	require.Equal(t, "spawn_agent", resp.Action)
	require.Equal(t, "T-E01-F01-001", resp.EntityKey)
	require.Equal(t, []string{"E01-F01"}, resp.ResolvedVia)
}

// TestFanoutFallsThroughPausedChildAndReTiers encodes the user's stated rule:
// "return the next available work — paused is not available." The order-1 task
// is parked at a status with no dispatch, so it must be skipped, and the
// order-2 pair behind it must then surface as a fork rather than an arbitrary
// pick. A linear scan to the next single child would pass a naive
// "did it fall through?" test but silently lose the fan-out opportunity, so
// this asserts the re-tier specifically.
func TestFanoutFallsThroughPausedChildAndReTiers(t *testing.T) {
	stubForkEdges(t)
	original := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = original }()

	planDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
		if entityType == "feature" && key == "E01-F01" {
			return services.PlanHierarchyChildrenState{
				Children: []services.PlanHierarchyChild{
					fanoutChild("T-E01-F01-001", models.EntityTypeTask, 1),
					fanoutChild("T-E01-F01-002", models.EntityTypeTask, 2),
					fanoutChild("T-E01-F01-003", models.EntityTypeTask, 2),
				},
				TotalChildren:       3,
				NonTerminalChildren: 3,
			}, nil
		}
		return services.PlanHierarchyChildrenState{}, nil
	}

	// The order-1 task sits at "awaiting_human", whose workflow action is a
	// pause — nothing for an agent to pick up.
	cache := fanoutCache(t, map[string]string{
		"E01-F01":       "active",
		"T-E01-F01-001": "awaiting_human",
	}, func(status string) *action.PopulatedAction {
		switch status {
		case "active":
			return &action.PopulatedAction{Action: "cascade", Instruction: "delegate"}
		case "awaiting_human":
			return &action.PopulatedAction{Action: "pause"}
		default:
			return &action.PopulatedAction{
				Action: "spawn_agent", AgentType: "backend", Provider: "anthropic",
				Model: "sonnet", Instruction: "implement the task",
			}
		}
	})

	resp, err := resolveNext(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.NotNil(t, resp.selection, "the paused order-1 child must not pause the whole parent")
	require.Len(t, resp.selection.Entities, 2, "the order-2 pair behind it is the next available work")
	require.Equal(t, "T-E01-F01-002", resp.selection.Entities[0].EntityKey)
	require.Equal(t, "T-E01-F01-003", resp.selection.Entities[1].EntityKey)
	require.Equal(t, "parallel_tie", resp.selection.SelectionReason)
}

// TestFanoutAutoAdvancesParentWhenAllChildrenTerminal proves the fan-out path
// kept keyed-next's auto-advance branch. Losing it would strand features and
// epics at 100% child completion instead of moving them into review/completed.
func TestFanoutAutoAdvancesParentWhenAllChildrenTerminal(t *testing.T) {
	original := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = original }()

	planDescribeDispatchableChildren = func(_ context.Context, _, _ string) (services.PlanHierarchyChildrenState, error) {
		// Children exist but every one is terminal — none dispatchable.
		return services.PlanHierarchyChildrenState{
			Children:            nil,
			TotalChildren:       2,
			NonTerminalChildren: 0,
		}, nil
	}

	statuses := map[string]string{"E01-F01": "active"}
	advanceTransitioner := &fanoutAdvanceTransitioner{
		statuses: statuses,
		next:     map[string]string{"active": "code_review"},
	}
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			if status == "active" {
				return &action.PopulatedAction{Action: "cascade", Instruction: "delegate"}, nil
			}
			return &action.PopulatedAction{
				Action: "spawn_agent", AgentType: "reviewer", Provider: "anthropic",
				Model: "sonnet", Instruction: "review the feature",
			}, nil
		},
	}
	cache := &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"feature": {
				transitioner: advanceTransitioner,
				generator:    fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc:    actionSvc,
			},
		},
		actionSvcRoot: actionSvc,
		surfaceForks:  true,
	}

	resp, err := resolveNext(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.Equal(t, "spawn_agent", resp.Action)
	require.Equal(t, "code_review", resp.Status, "the parent must advance rather than stall at 100% child completion")
	require.True(t, advanceTransitioner.transitioned)
}

// TestSequentialAndFanoutUseSameHierarchyEnumeration proves the mode changes
// only fork emission. Both runs consult the same hierarchy seam and select the
// same first eligible candidate; sequential returns its dispatch while
// fan-out surfaces the complete tied tier.
func TestSequentialAndFanoutUseSameHierarchyEnumeration(t *testing.T) {
	originalPlan := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = originalPlan }()

	calls := 0
	planDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
		calls++
		if entityType == "feature" && key == "E01-F01" {
			order := 1
			return services.PlanHierarchyChildrenState{
				Children: []services.PlanHierarchyChild{
					{
						Key: "T-E01-F01-001", EntityType: models.EntityTypeTask,
						ExecutionOrder: &order,
					},
					{
						Key: "T-E01-F01-002", EntityType: models.EntityTypeTask,
						ExecutionOrder: &order,
					},
				},
				TotalChildren:       2,
				NonTerminalChildren: 2,
			}, nil
		}
		return services.PlanHierarchyChildrenState{}, nil
	}

	statuses := map[string]string{
		"E01-F01":       "active",
		"T-E01-F01-001": "todo",
		"T-E01-F01-002": "todo",
	}
	sequential := fanoutCache(t, statuses, cascadeOrDispatch)
	sequential.surfaceForks = false

	resp, err := resolveNext(context.Background(), sequential, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.Nil(t, resp.selection, "sequential mode never forks")
	require.Equal(t, "spawn_agent", resp.Action)
	require.Equal(t, "T-E01-F01-001", resp.EntityKey)

	stubForkEdges(t)
	fanout := fanoutCache(t, statuses, cascadeOrDispatch)
	fanout.surfaceForks = true
	resp, err = resolveNext(context.Background(), fanout, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.NotNil(t, resp.selection)
	require.Equal(t, []string{
		"T-E01-F01-001", "T-E01-F01-002",
	}, hierarchyPlanCandidateKeys(resp.selection.Entities))
	require.Equal(t, 2, calls, "both modes must consult the same hierarchy seam once")
}

// TestSequentialTraversalStopsAtFirstLiveCandidate protects sequential mode's
// evaluation boundary: once the first candidate can dispatch, later tied
// siblings must not be resolved speculatively.
func TestSequentialTraversalStopsAtFirstLiveCandidate(t *testing.T) {
	originalPlan := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = originalPlan }()

	planDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
		if entityType != "feature" || key != "E01-F01" {
			return services.PlanHierarchyChildrenState{}, nil
		}
		return services.PlanHierarchyChildrenState{
			Children: []services.PlanHierarchyChild{
				fanoutChild("T-E01-F01-001", models.EntityTypeTask, 1),
				fanoutChild("T-E01-F01-002", models.EntityTypeTask, 1),
			},
			TotalChildren:       2,
			NonTerminalChildren: 2,
		}, nil
	}

	statuses := map[string]string{
		"E01-F01":       "active",
		"T-E01-F01-001": "todo",
		"T-E01-F01-002": "must_not_resolve",
	}
	cache := fanoutCache(t, statuses, func(status string) *action.PopulatedAction {
		if status == "must_not_resolve" {
			t.Fatal("sequential mode resolved a sibling after finding live work")
		}
		return cascadeOrDispatch(status)
	})
	cache.surfaceForks = false

	resp, err := resolveNext(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.Equal(t, "spawn_agent", resp.Action)
	require.Equal(t, "T-E01-F01-001", resp.EntityKey)
}

// TestForkFailsLoudlyWhenEdgeLoadFails pins the deliberate asymmetry with the
// claim lookup: a failed edge load must abort, because an edge-less fork is
// indistinguishable on the wire from a genuinely dependency-free one and would
// invite the rider to launch coupled work in parallel.
func TestForkFailsLoudlyWhenEdgeLoadFails(t *testing.T) {
	originalChildren := planDescribeDispatchableChildren
	originalEdges := fanoutDescribeCandidateEdges
	defer func() {
		planDescribeDispatchableChildren = originalChildren
		fanoutDescribeCandidateEdges = originalEdges
	}()

	planDescribeDispatchableChildren = func(_ context.Context, _, _ string) (services.PlanHierarchyChildrenState, error) {
		return services.PlanHierarchyChildrenState{
			Children: []services.PlanHierarchyChild{
				fanoutChild("T-E01-F01-001", models.EntityTypeTask, 1),
				fanoutChild("T-E01-F01-002", models.EntityTypeTask, 1),
			},
			TotalChildren:       2,
			NonTerminalChildren: 2,
		}, nil
	}
	fanoutDescribeCandidateEdges = func(_ context.Context, _ string, _ []string) (map[string]services.PlanHierarchyEdges, error) {
		return nil, errors.New("relationship query failed")
	}

	cache := fanoutCache(t, map[string]string{"E01-F01": "active"}, cascadeOrDispatch)

	_, err := resolveNext(context.Background(), cache, "feature", "E01-F01", 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "relationship query failed")
}

// TestSequentialDispatchPrecedence covers the resolution the user locked:
// default is fan-out, config can force sequential, and an explicitly passed
// --sequential flag overrides config in both directions. Without this,
// inverting the `!` in `adapters.surfaceForks = !sequential` would leave every other
// test passing.
func TestSequentialDispatchPrecedence(t *testing.T) {
	original := nextGetSequentialDispatch
	defer func() { nextGetSequentialDispatch = original }()

	tests := []struct {
		name        string
		config      bool
		flagPassed  bool
		flagValue   bool
		wantSeqMode bool
	}{
		{name: "default is fan-out", config: false, flagPassed: false, wantSeqMode: false},
		{name: "config forces sequential", config: true, flagPassed: false, wantSeqMode: true},
		{name: "flag overrides config off", config: true, flagPassed: true, flagValue: false, wantSeqMode: false},
		{name: "flag overrides config on", config: false, flagPassed: true, flagValue: true, wantSeqMode: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextGetSequentialDispatch = func() bool { return tt.config }

			cmd := newNextCommandForTest()
			if tt.flagPassed {
				require.NoError(t, cmd.Flags().Set("sequential", strconv.FormatBool(tt.flagValue)))
			}

			// Call the production resolver, not a copy of it — otherwise
			// changing runNext's precedence leaves this test green.
			require.Equal(t, tt.wantSeqMode, resolveSequentialDispatch(cmd))
		})
	}
}

// newNextCommandForTest returns a command carrying the real --sequential flag
// registration, taken from the package-level nextCmd so the test cannot drift
// from the flag the binary actually exposes.
func newNextCommandForTest() *cobra.Command {
	cmd := &cobra.Command{Use: "next"}
	flag := nextCmd.Flags().Lookup("sequential")
	if flag == nil {
		panic("nextCmd no longer registers a --sequential flag")
	}
	cmd.Flags().Bool(flag.Name, false, flag.Usage)
	return cmd
}

// fanoutAdvanceTransitioner records a status transition and reflects it back
// on subsequent GetNextStatus calls, so auto-advance recursion sees the new
// status.
type fanoutAdvanceTransitioner struct {
	statuses     map[string]string
	next         map[string]string
	transitioned bool
}

func (t *fanoutAdvanceTransitioner) GetNextStatus(_ context.Context, key string) (*services.NextStatusInfo, error) {
	current := t.statuses[key]
	info := &services.NextStatusInfo{CurrentStatus: current}
	if target, ok := t.next[current]; ok {
		info.AvailableTransitions = []services.TransitionInfoWithAction{
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: target}},
		}
	}
	return info, nil
}

func (t *fanoutAdvanceTransitioner) TransitionStatus(_ context.Context, key, target string, _ services.TransitionOptions) (*services.TransitionResult, error) {
	t.transitioned = true
	t.statuses[key] = target
	return &services.TransitionResult{}, nil
}

// TestUnorderedSiblingsTieIntoAFork covers the tier rule for children that
// carry no execution order at all. Features are created with a NULL
// execution_order unless --order is passed, so without this an epic's features
// would never surface as a parallel opportunity and `shark next <epic>` — the
// entry point riders actually use — would keep silently picking feature[0].
// Nothing sequences these children, which is the same argument that makes an
// equal execution_order a tie.
func TestUnorderedSiblingsTieIntoAFork(t *testing.T) {
	unordered := func(key string, entityType models.EntityType) services.PlanHierarchyChild {
		return services.PlanHierarchyChild{
			Key: key, Title: key + " title", Status: "draft", EntityType: entityType,
		}
	}

	t.Run("unordered features tie", func(t *testing.T) {
		selected, reason := selectPlanChildTier([]services.PlanHierarchyChild{
			unordered("E01-F01", models.EntityTypeFeature),
			unordered("E01-F02", models.EntityTypeFeature),
		})
		require.Equal(t, "parallel_tie", reason)
		require.Len(t, selected, 2)
	})

	t.Run("a lone unordered child is not a tie", func(t *testing.T) {
		selected, reason := selectPlanChildTier([]services.PlanHierarchyChild{
			unordered("E01-F01", models.EntityTypeFeature),
		})
		require.Equal(t, "repository_order", reason)
		require.Len(t, selected, 1)
	})

	t.Run("ordered children still win over unordered ones", func(t *testing.T) {
		// Children arrive with NULL execution_order sorted last, so an
		// ordered child leads and the unordered tail must not join its tier.
		selected, reason := selectPlanChildTier([]services.PlanHierarchyChild{
			fanoutChild("E01-F01", models.EntityTypeFeature, 1),
			unordered("E01-F02", models.EntityTypeFeature),
			unordered("E01-F03", models.EntityTypeFeature),
		})
		require.Equal(t, "execution_order", reason)
		require.Len(t, selected, 1)
		require.Equal(t, "E01-F01", selected[0].Key)
	})

	t.Run("prioritized tasks are not swept in as unordered", func(t *testing.T) {
		prioritized := services.PlanHierarchyChild{
			Key: "T-E01-F01-002", Title: "t2", Status: "todo",
			EntityType: models.EntityTypeTask, Priority: fanoutIntPtr(3),
		}
		selected, reason := selectPlanChildTier([]services.PlanHierarchyChild{
			unordered("T-E01-F01-001", models.EntityTypeTask),
			prioritized,
		})
		require.Equal(t, "repository_order", reason)
		require.Len(t, selected, 1)
	})
}

// TestFanoutForksOnUnorderedFeatures is the end-to-end consequence: an epic
// whose features carry no explicit order must fork rather than drill into the
// first feature.
func TestFanoutForksOnUnorderedFeatures(t *testing.T) {
	stubForkEdges(t)
	original := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = original }()

	planDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
		if entityType == "epic" && key == "E01" {
			return services.PlanHierarchyChildrenState{
				Children: []services.PlanHierarchyChild{
					{Key: "E01-F01", Title: "f1", Status: "draft", EntityType: models.EntityTypeFeature},
					{Key: "E01-F02", Title: "f2", Status: "draft", EntityType: models.EntityTypeFeature},
				},
				TotalChildren:       2,
				NonTerminalChildren: 2,
			}, nil
		}
		return services.PlanHierarchyChildrenState{}, nil
	}

	cache := fanoutCache(t, map[string]string{"E01": "active"}, cascadeOrDispatch)

	resp, err := resolveNext(context.Background(), cache, "epic", "E01", 0)
	require.NoError(t, err)
	require.NotNil(t, resp.selection, "unordered features must fork, not drill into feature[0]")
	require.Len(t, resp.selection.Entities, 2)
	require.Equal(t, "E01", resp.selection.RootKey)
}

// pausedUnlessDispatchable maps a set of "parked" statuses to a pause action
// and everything else to a dispatch, so a test can park specific children.
func pausedUnlessDispatchable(parked map[string]bool) func(string) *action.PopulatedAction {
	return func(status string) *action.PopulatedAction {
		switch {
		case status == "active":
			return &action.PopulatedAction{Action: "cascade", Instruction: "delegate"}
		case parked[status]:
			return &action.PopulatedAction{Action: "pause"}
		default:
			return &action.PopulatedAction{
				Action: "spawn_agent", AgentType: "backend", Provider: "anthropic",
				Model: "sonnet", Instruction: "implement the task",
			}
		}
	}
}

// TestFanoutFallsThroughAnEntirelyParkedTier is the tie-tier counterpart to
// TestFanoutFallsThroughPausedChildAndReTiers, and guards the hole that
// version of the code had: the fork branch used to return the tied tier
// WITHOUT resolving any candidate, so a tier of children that all resolve to
// pause was emitted as parallel_candidates. The rider would dispatch each,
// receive pause for each, and stop — never reaching the dispatchable sibling
// behind them. Claim/dependency filtering cannot prevent this because it has
// no view of the workflow action.
func TestFanoutFallsThroughAnEntirelyParkedTier(t *testing.T) {
	stubForkEdges(t)
	original := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = original }()

	planDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
		if entityType == "feature" && key == "E01-F01" {
			return services.PlanHierarchyChildrenState{
				Children: []services.PlanHierarchyChild{
					fanoutChild("T-E01-F01-001", models.EntityTypeTask, 1),
					fanoutChild("T-E01-F01-002", models.EntityTypeTask, 1),
					fanoutChild("T-E01-F01-003", models.EntityTypeTask, 2),
				},
				TotalChildren:       3,
				NonTerminalChildren: 3,
			}, nil
		}
		return services.PlanHierarchyChildrenState{}, nil
	}

	cache := fanoutCache(t, map[string]string{
		"E01-F01":       "active",
		"T-E01-F01-001": "awaiting_human",
		"T-E01-F01-002": "awaiting_human",
		"T-E01-F01-003": "todo",
	}, pausedUnlessDispatchable(map[string]bool{"awaiting_human": true}))

	resp, err := resolveNext(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.Nil(t, resp.selection,
		"a tier whose every candidate resolves to pause is not available work and must not be offered as a fork")
	require.Equal(t, "spawn_agent", resp.Action)
	require.Equal(t, "T-E01-F01-003", resp.EntityKey,
		"the dispatchable sibling behind the parked tier must still be reached")
	require.Equal(t, []string{"E01-F01"}, resp.ResolvedVia)
}

// TestFanoutForksOnlyOnSurvivingCandidates checks the partial case: a tie where
// some candidates are parked forks on exactly the ones that can actually run.
// Offering a parked candidate would waste a dispatch and, if every sibling the
// rider picked were parked, stall the loop.
func TestFanoutForksOnlyOnSurvivingCandidates(t *testing.T) {
	stubForkEdges(t)
	original := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = original }()

	planDescribeDispatchableChildren = func(_ context.Context, _, _ string) (services.PlanHierarchyChildrenState, error) {
		return services.PlanHierarchyChildrenState{
			Children: []services.PlanHierarchyChild{
				fanoutChild("T-E01-F01-001", models.EntityTypeTask, 1),
				fanoutChild("T-E01-F01-002", models.EntityTypeTask, 1),
				fanoutChild("T-E01-F01-003", models.EntityTypeTask, 1),
			},
			TotalChildren:       3,
			NonTerminalChildren: 3,
		}, nil
	}

	cache := fanoutCache(t, map[string]string{
		"E01-F01":       "active",
		"T-E01-F01-002": "awaiting_human",
	}, pausedUnlessDispatchable(map[string]bool{"awaiting_human": true}))

	resp, err := resolveNext(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.NotNil(t, resp.selection)
	require.Len(t, resp.selection.Entities, 2)
	require.Equal(t, "T-E01-F01-001", resp.selection.Entities[0].EntityKey)
	require.Equal(t, "T-E01-F01-003", resp.selection.Entities[1].EntityKey,
		"the parked candidate must be excluded from the offered set")
}

// TestFanoutWithCapBelowTwoDispatchesInsteadOfEmittingPromptlessSelection pins
// the contract for max_parallel_items = 1, a value the configuration reference
// explicitly recommends for "deterministic singleton selection". Capping the
// tier to one inside buildHierarchyPlanSelection would emit a
// `select_<type>` envelope carrying no prompt — outside the keyed-next wire
// vocabulary — and the harness would stall on it.
func TestFanoutWithCapBelowTwoDispatchesInsteadOfEmittingPromptlessSelection(t *testing.T) {
	stubForkEdges(t)
	original := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = original }()

	planDescribeDispatchableChildren = func(_ context.Context, _, _ string) (services.PlanHierarchyChildrenState, error) {
		return services.PlanHierarchyChildrenState{
			Children: []services.PlanHierarchyChild{
				fanoutChild("T-E01-F01-001", models.EntityTypeTask, 1),
				fanoutChild("T-E01-F01-002", models.EntityTypeTask, 1),
			},
			TotalChildren:       2,
			NonTerminalChildren: 2,
		}, nil
	}

	cache := fanoutCache(t, map[string]string{"E01-F01": "active"}, cascadeOrDispatch)
	stubMaxParallelItems(t, 1)

	resp, err := resolveNext(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.Nil(t, resp.selection, "a cap of 1 means no fan-out, not a one-candidate selection envelope")
	require.Equal(t, "spawn_agent", resp.Action)
	require.NotEmpty(t, resp.Prompt, "the harness needs a dispatchable prompt, not a bare selection")
	require.Equal(t, "T-E01-F01-001", resp.EntityKey)
}
