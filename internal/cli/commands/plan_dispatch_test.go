package commands

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolvePlanEntityLeafMatchesKeyedNextRenderedResponse pins requirement
// #11: `shark plan <leaf-entity>` (or a parent already at its own agent
// step) must return the exact same rendered dispatch response `shark next`
// would — same action, agent metadata, and fully-assembled prompt — with no
// selection or parallel envelope attached.
func TestResolvePlanEntityLeafMatchesKeyedNextRenderedResponse(t *testing.T) {
	transitioner := keyedByEntityTransitioner{statuses: map[string]string{
		"T-E01-F01-001": "in_progress",
	}}
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			return &action.PopulatedAction{
				Action: "spawn_agent", AgentType: "backend", Provider: "anthropic", Model: "sonnet",
				Instruction: "implement the task",
			}, nil
		},
	}
	baseCache := &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"task": {transitioner: transitioner, generator: fixedNextPlaceholders{vars: map[string]string{}}, actionSvc: actionSvc},
		},
		actionSvcRoot: actionSvc,
	}

	nextResp, err := resolveNext(context.Background(), baseCache, "task", "T-E01-F01-001", 0)
	require.NoError(t, err)

	planCache := &planAdapterCache{nextAdapterCache: &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"task": {transitioner: transitioner, generator: fixedNextPlaceholders{vars: map[string]string{}}, actionSvc: actionSvc},
		},
		actionSvcRoot: actionSvc,
	}, maxParallelItems: 5}

	planResp, err := resolvePlanDispatch(context.Background(), planCache, "task", "T-E01-F01-001", 0)
	require.NoError(t, err)

	assert.Nil(t, planResp.selection, "leaf plan resolution must not attach a selection envelope")
	assert.Equal(t, nextResp.EntityKey, planResp.EntityKey)
	assert.Equal(t, nextResp.Action, planResp.Action)
	assert.Equal(t, nextResp.AgentType, planResp.AgentType)
	assert.Equal(t, nextResp.Provider, planResp.Provider)
	assert.Equal(t, nextResp.Model, planResp.Model)
	assert.Equal(t, nextResp.Prompt, planResp.Prompt)
}

// TestTryPlanHierarchyAutoAdvancesCascadeCompleteParentWithNoNewSideEffects
// pins requirement #13: when every direct child is terminal, `shark plan`
// auto-advances the parent exactly one configured step (the same
// cascade-completion behavior next.go's tryCascade performs) and recurses on
// the same entity, performing exactly one transition — no claim, heartbeat,
// release, or extra write.
func TestTryPlanHierarchyAutoAdvancesCascadeCompleteParentWithNoNewSideEffects(t *testing.T) {
	originalDescribe := planDescribeDispatchableChildren
	originalTransitionerBuilder := nextBuildTransitioner
	originalPlaceholderBuilder := nextBuildPlaceholderGenerator
	defer func() {
		planDescribeDispatchableChildren = originalDescribe
		nextBuildTransitioner = originalTransitionerBuilder
		nextBuildPlaceholderGenerator = originalPlaceholderBuilder
	}()

	planDescribeDispatchableChildren = func(context.Context, string, string) (services.PlanHierarchyChildrenState, error) {
		return services.PlanHierarchyChildrenState{
			TotalChildren:       3,
			NonTerminalChildren: 0,
		}, nil
	}

	transitioner := &cascadeAutoAdvanceTransitioner{currentStatus: "active"}
	nextBuildTransitioner = func(_ context.Context, entityType string) (runner.EntityTransitioner, error) {
		return transitioner, nil
	}
	nextBuildPlaceholderGenerator = func(_ context.Context, entityType string) runner.PlaceholderGenerator {
		return fixedNextPlaceholders{vars: map[string]string{}}
	}
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			switch status {
			case "active":
				return &action.PopulatedAction{Action: "cascade", Instruction: "delegate child work"}, nil
			case "code_review":
				return &action.PopulatedAction{
					Action: "spawn_agent", AgentType: "tech-lead", Provider: "anthropic", Model: "sonnet",
					Instruction: "review completed feature E03-F02",
				}, nil
			default:
				return nil, nil
			}
		},
	}
	cache := &planAdapterCache{nextAdapterCache: &nextAdapterCache{
		entries:       map[string]*nextAdapters{},
		actionSvcRoot: actionSvc,
	}}

	resp, err := resolvePlanDispatch(context.Background(), cache, "feature", "E03-F02", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"code_review"}, transitioner.transitionedTo, "exactly one auto-advance transition, same as next.go's cascade completion")
	assert.Equal(t, "E03-F02", resp.EntityKey)
	assert.Equal(t, "spawn_agent", resp.Action)
	assert.Nil(t, resp.selection)
}
