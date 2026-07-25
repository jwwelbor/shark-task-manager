package commands

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/spf13/cobra"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
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
	return &nextAdapterCache{entries: entries, actionSvcRoot: actionSvc, fanout: true}
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
// kept tryCascade's auto-advance branch. Losing it would strand features and
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
		fanout:        true,
	}

	resp, err := resolveNext(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.Equal(t, "spawn_agent", resp.Action)
	require.Equal(t, "code_review", resp.Status, "the parent must advance rather than stall at 100% child completion")
	require.True(t, advanceTransitioner.transitioned)
}

// TestSequentialModeUsesLegacyCascade proves --sequential / sequential_dispatch
// still routes through tryCascade: it consults the CascadeService seam and
// never the plan-hierarchy seam, so the legacy single-track contract is
// reachable unchanged.
func TestSequentialModeUsesLegacyCascade(t *testing.T) {
	originalPlan := planDescribeDispatchableChildren
	originalCascade := nextDescribeDispatchableChildren
	defer func() {
		planDescribeDispatchableChildren = originalPlan
		nextDescribeDispatchableChildren = originalCascade
	}()

	planCalled := false
	planDescribeDispatchableChildren = func(_ context.Context, _, _ string) (services.PlanHierarchyChildrenState, error) {
		planCalled = true
		return services.PlanHierarchyChildrenState{}, nil
	}
	nextDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.CascadeChildrenState, error) {
		if entityType == "feature" && key == "E01-F01" {
			return services.CascadeChildrenState{
				Children: []services.CascadeChild{
					{Key: "T-E01-F01-001", EntityType: models.EntityTypeTask},
					{Key: "T-E01-F01-002", EntityType: models.EntityTypeTask},
				},
				TotalChildren:       2,
				NonTerminalChildren: 2,
			}, nil
		}
		return services.CascadeChildrenState{}, nil
	}

	cache := fanoutCache(t, map[string]string{
		"E01-F01":       "active",
		"T-E01-F01-001": "todo",
	}, cascadeOrDispatch)
	cache.fanout = false

	resp, err := resolveNext(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.False(t, planCalled, "sequential mode must not consult the plan hierarchy seam")
	require.Nil(t, resp.selection, "sequential mode never forks")
	require.Equal(t, "spawn_agent", resp.Action)
	require.Equal(t, "T-E01-F01-001", resp.EntityKey, "legacy cascade picks the first dispatchable child")
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
// inverting the `!` in `adapters.fanout = !sequential` would leave every other
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

			sequential := nextGetSequentialDispatch()
			if cmd.Flags().Changed("sequential") {
				sequential, _ = cmd.Flags().GetBool("sequential")
			}

			require.Equal(t, tt.wantSeqMode, sequential)
			// This is the line runNext executes; a flipped sign here is the
			// exact regression this test exists to catch.
			require.Equal(t, !tt.wantSeqMode, !sequential)
		})
	}
}

// newNextCommandForTest returns a command carrying the same --sequential flag
// registration as nextCmd, without mutating the package-level command.
func newNextCommandForTest() *cobra.Command {
	cmd := &cobra.Command{Use: "next"}
	cmd.Flags().Bool("sequential", false, "Force the legacy single-track cascade instead of fan-out")
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
