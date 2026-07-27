package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanAndNextUnknownStatusSharePauseAndCommandLabeledWarning(t *testing.T) {
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(
			context.Context,
			string,
			map[string]string,
		) (*action.PopulatedAction, error) {
			return nil, &action.StatusNotFoundError{Status: "legacy_review"}
		},
	}
	newCache := func() *nextAdapterCache {
		return &nextAdapterCache{
			entries: map[string]*nextAdapters{
				"task": {
					transitioner: keyedByEntityTransitioner{statuses: map[string]string{
						"T-E01-F01-001": "legacy_review",
					}},
					generator: fixedNextPlaceholders{vars: map[string]string{}},
					actionSvc: actionSvc,
				},
			},
			actionSvcRoot: actionSvc,
		}
	}

	var nextResp NextResponse
	nextStderr := captureStderrOutput(func() {
		var err error
		nextResp, err = resolveNext(
			context.Background(), newCache(), "task", "T-E01-F01-001", 0,
		)
		require.NoError(t, err)
	})
	var planResp NextResponse
	planStderr := captureStderrOutput(func() {
		var err error
		planResp, err = resolvePlanDispatch(
			context.Background(),
			&planAdapterCache{nextAdapterCache: newCache()},
			"task",
			"T-E01-F01-001",
			0,
		)
		require.NoError(t, err)
	})

	require.Equal(t, "pause", nextResp.Action)
	require.Equal(t, nextResp.Action, planResp.Action)
	require.Equal(t, nextResp.Error, planResp.Error)
	require.Contains(t, nextStderr, "[shark next] warning:")
	require.Contains(t, planStderr, "[shark plan] warning:")
	require.True(t, strings.Contains(nextResp.Error, "legacy status"), nextResp.Error)
}

// TestPlanAndNextLeafResolutionContractsMatch pins that the shared resolver
// returns byte-for-byte-equivalent dispatch data outside the intentionally
// different cascade strategies.
func TestPlanAndNextLeafResolutionContractsMatch(t *testing.T) {
	transitioner := keyedByEntityTransitioner{statuses: map[string]string{
		"T-E01-F01-001": "in_progress",
	}}
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			return &action.PopulatedAction{
				Action: "spawn_agent", AgentType: "backend", Provider: "anthropic", Model: "sonnet",
				Effort: "high", Instruction: "implement the task",
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
	assert.Equal(t, nextResp, planResp)
	assert.Equal(t, "high", planResp.Effort)
}

func TestPlanAndNextShareBasicResolutionOutcomes(t *testing.T) {
	tests := []struct {
		name          string
		info          *services.NextStatusInfo
		populated     *action.PopulatedAction
		wantAction    string
		wantErr       string
		wantRespError string
	}{
		{
			name:       "terminal pauses",
			info:       &services.NextStatusInfo{CurrentStatus: "done", IsTerminal: true},
			wantAction: "pause",
		},
		{
			name:       "archived suffix archives",
			info:       &services.NextStatusInfo{CurrentStatus: "in_qa_archived"},
			wantAction: "archive",
		},
		{
			name:       "missing action pauses",
			info:       &services.NextStatusInfo{CurrentStatus: "waiting"},
			wantAction: "pause",
		},
		{
			name:          "unknown internal action pauses with diagnostic",
			info:          &services.NextStatusInfo{CurrentStatus: "working"},
			populated:     &action.PopulatedAction{Action: "invented"},
			wantAction:    "pause",
			wantRespError: "unknown internal action verb",
		},
		{
			name:      "required instruction is enforced",
			info:      &services.NextStatusInfo{CurrentStatus: "working"},
			populated: &action.PopulatedAction{Action: "spawn_agent", AgentType: "backend"},
			wantErr:   "rendered an empty instruction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actionSvc := &action.MockActionService{
				GetStatusActionPopulatedFunc: func(
					context.Context,
					string,
					map[string]string,
				) (*action.PopulatedAction, error) {
					return tt.populated, nil
				},
			}
			newCache := func() *nextAdapterCache {
				return &nextAdapterCache{
					entries: map[string]*nextAdapters{
						"task": {
							transitioner: fixedNextTransitioner{info: tt.info},
							generator:    fixedNextPlaceholders{vars: map[string]string{}},
							actionSvc:    actionSvc,
						},
					},
					actionSvcRoot: actionSvc,
				}
			}

			nextResp, nextErr := resolveNext(
				context.Background(), newCache(), "task", "T-E01-F01-001", 0,
			)
			planResp, planErr := resolvePlanDispatch(
				context.Background(),
				&planAdapterCache{nextAdapterCache: newCache()},
				"task",
				"T-E01-F01-001",
				0,
			)

			if tt.wantErr != "" {
				require.ErrorContains(t, nextErr, tt.wantErr)
				require.ErrorContains(t, planErr, tt.wantErr)
				require.Equal(t, nextErr.Error(), planErr.Error())
				return
			}
			require.NoError(t, nextErr)
			require.NoError(t, planErr)
			require.Equal(t, nextResp, planResp)
			require.Equal(t, tt.wantAction, planResp.Action)
			if tt.wantRespError != "" {
				require.Contains(t, planResp.Error, tt.wantRespError)
			}
		})
	}
}

// TestTryPlanHierarchyAutoAdvancesCascadeCompleteParentWithNoNewSideEffects
// pins requirement #13: when every direct child is terminal, `shark plan`
// auto-advances the parent exactly one configured step (the same
// cascade-completion behavior keyed next performs) and recurses on
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

func TestResolvePlanDispatchPreservesOneLevelSelectionAfterAgentlessAdvance(t *testing.T) {
	originalPlanDescribe := planDescribeDispatchableChildren
	defer func() {
		planDescribeDispatchableChildren = originalPlanDescribe
	}()

	order := 1
	planDescribeDispatchableChildren = func(
		_ context.Context,
		entityType, key string,
	) (services.PlanHierarchyChildrenState, error) {
		require.Equal(t, "feature", entityType)
		require.Equal(t, "E01-F01", key)
		return services.PlanHierarchyChildrenState{
			Children: []services.PlanHierarchyChild{
				{
					Key: "T-E01-F01-001", Title: "First", Status: "todo",
					EntityType: models.EntityTypeTask, ExecutionOrder: &order,
				},
				{
					Key: "T-E01-F01-002", Title: "Second", Status: "todo",
					EntityType: models.EntityTypeTask, ExecutionOrder: &order,
				},
			},
			TotalChildren:       2,
			NonTerminalChildren: 2,
		}, nil
	}
	transitioner := &keyedPlanTransitioner{
		infos: map[string]*services.NextStatusInfo{
			"E01-F01": {
				CurrentStatus: "routing",
				EntityType:    models.EntityTypeFeature,
				AvailableTransitions: []services.TransitionInfoWithAction{{
					TransitionInfo: workflow.TransitionInfo{TargetStatus: "active"},
				}},
			},
			"T-E01-F01-001": {CurrentStatus: "todo", EntityType: models.EntityTypeTask},
		},
	}
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(
			_ context.Context,
			status string,
			_ map[string]string,
		) (*action.PopulatedAction, error) {
			switch status {
			case "routing":
				return &action.PopulatedAction{Action: "advance_status"}, nil
			case "active":
				return &action.PopulatedAction{Action: "cascade", Instruction: "delegate"}, nil
			case "todo":
				return &action.PopulatedAction{
					Action: "spawn_agent", AgentType: "backend", Instruction: "implement",
				}, nil
			default:
				t.Fatalf("unexpected status action lookup %q", status)
				return nil, nil
			}
		},
	}
	cache := &planAdapterCache{
		nextAdapterCache: &nextAdapterCache{
			entries: map[string]*nextAdapters{
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
		},
		maxParallelItems: 5,
	}

	resp, err := resolvePlanDispatch(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.Equal(t, "active", transitioner.transitions["E01-F01"])
	require.NotNil(t, resp.selection, "plan recursion must retain one-level selection strategy")
	require.Equal(t, "parallel_candidates", resp.selection.Action)
	require.Equal(t, []string{
		"T-E01-F01-001", "T-E01-F01-002",
	}, hierarchyPlanCandidateKeys(resp.selection.Entities))
	require.Empty(t, resp.Prompt, "a hierarchy selection is not a worker dispatch")
}

func TestResolvePlanDispatchPausesAgentlessAdvanceWithoutSafeTarget(t *testing.T) {
	transitioner := &keyedPlanTransitioner{infos: map[string]*services.NextStatusInfo{
		"E01-F01": {
			CurrentStatus: "routing",
			EntityType:    models.EntityTypeFeature,
		},
	}}
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(
			context.Context,
			string,
			map[string]string,
		) (*action.PopulatedAction, error) {
			return &action.PopulatedAction{Action: "advance_status"}, nil
		},
	}
	cache := &planAdapterCache{nextAdapterCache: &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"feature": {
				transitioner: transitioner,
				generator:    fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc:    actionSvc,
			},
		},
		actionSvcRoot: actionSvc,
	}}

	resp, err := resolvePlanDispatch(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.Equal(t, "pause", resp.Action)
	require.Equal(t, "routing", resp.Status)
	require.Empty(t, transitioner.transitions, "no transition is safe")
	require.Nil(t, resp.selection)
}
