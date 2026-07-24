package commands

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
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
	originalDescribe := nextDescribeDispatchableChildren
	defer func() { nextDescribeDispatchableChildren = originalDescribe }()

	nextDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.CascadeChildrenState, error) {
		switch {
		case entityType == "epic" && key == "E01":
			return services.CascadeChildrenState{
				Children:            []services.CascadeChild{{Key: "E01-F01", EntityType: models.EntityTypeFeature}},
				TotalChildren:       1,
				NonTerminalChildren: 1,
			}, nil
		case entityType == "feature" && key == "E01-F01":
			return services.CascadeChildrenState{
				Children:            []services.CascadeChild{{Key: "T-E01-F01-001", EntityType: models.EntityTypeTask}},
				TotalChildren:       1,
				NonTerminalChildren: 1,
			}, nil
		default:
			return services.CascadeChildrenState{}, nil
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
