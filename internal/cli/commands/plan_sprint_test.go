package commands

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/stretchr/testify/require"
)

type sprintSelectorStub struct {
	selectFn func(context.Context, services.SprintSelectionInput) (*services.SprintSelection, error)
	activeFn func(context.Context, services.SprintSelectionInput) (*services.SprintSelection, error)
}

func (s sprintSelectorStub) SelectSprint(ctx context.Context, input services.SprintSelectionInput) (*services.SprintSelection, error) {
	return s.selectFn(ctx, input)
}

func (s sprintSelectorStub) SelectActiveSprint(ctx context.Context, input services.SprintSelectionInput) (*services.SprintSelection, error) {
	return s.activeFn(ctx, input)
}

func TestRunPlanSprintSelectionReturnsPreviewCandidates(t *testing.T) {
	original := planGetSprintSelector
	originalLimit := planGetMaxParallelItems
	defer func() {
		planGetSprintSelector = original
		planGetMaxParallelItems = originalLimit
	}()
	planGetSprintSelector = func() sprintSelector {
		return sprintSelectorStub{
			selectFn: func(_ context.Context, input services.SprintSelectionInput) (*services.SprintSelection, error) {
				require.Equal(t, "S001", input.SprintKey)
				require.Equal(t, "backend", input.AgentType)
				require.Equal(t, 2, input.Limit)
				return &services.SprintSelection{SprintKey: "S001", Preview: true, Items: []*services.BacklogItemView{
					{Key: "T-E19-F09-001", EntityType: "task"},
					{Key: "T-E19-F09-002", EntityType: "task"},
				}}, nil
			},
			activeFn: func(context.Context, services.SprintSelectionInput) (*services.SprintSelection, error) {
				t.Fatal("explicit sprint key must not select an active sprint")
				return nil, nil
			},
		}
	}
	planGetMaxParallelItems = func() int { return 2 }

	cmd := newPlanCommand()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.Flags().Set("agent", "backend"))
	var runErr error
	output := capturingOutput(func() { runErr = runPlan(cmd, []string{"S001"}) })
	require.NoError(t, runErr)

	var response SprintPlanSelectionResponse
	require.NoError(t, json.Unmarshal([]byte(output), &response), output)
	require.Equal(t, "sprint_selection", response.Mode)
	require.Equal(t, "parallel_candidates", response.Action)
	require.Equal(t, "S001", response.SprintKey)
	require.True(t, response.Preview)
	require.Len(t, response.Entities, 2)
}

// TC-007: the F09 selection envelope retains its original shape while adding
// the read-only roadmap gate audit context.
func TestRunPlanSprintSelectionPublishesAdmissionAuditContext(t *testing.T) {
	original := planGetSprintSelector
	defer func() { planGetSprintSelector = original }()
	planGetSprintSelector = func() sprintSelector {
		return sprintSelectorStub{
			selectFn: func(context.Context, services.SprintSelectionInput) (*services.SprintSelection, error) {
				return &services.SprintSelection{SprintKey: "S001", PortfolioEpicKey: "E19", ExcludedByReason: map[string]int{"ancestor_dependency_unmet": 1}}, nil
			},
			activeFn: func(context.Context, services.SprintSelectionInput) (*services.SprintSelection, error) {
				return nil, nil
			},
		}
	}
	cmd := newPlanCommand()
	cmd.SetContext(context.Background())
	var runErr error
	output := capturingOutput(func() { runErr = runPlan(cmd, []string{"S001"}) })
	require.NoError(t, runErr)
	var response SprintPlanSelectionResponse
	require.NoError(t, json.Unmarshal([]byte(output), &response), output)
	require.Equal(t, "E19", response.PortfolioEpicKey)
	require.Equal(t, 1, response.ExcludedByReason["ancestor_dependency_unmet"])
}

func TestRunPlanSprintUsesActiveSprintSelector(t *testing.T) {
	original := planGetSprintSelector
	defer func() { planGetSprintSelector = original }()
	planGetSprintSelector = func() sprintSelector {
		return sprintSelectorStub{
			selectFn: func(context.Context, services.SprintSelectionInput) (*services.SprintSelection, error) {
				t.Fatal("implicit sprint selection must use active sprint selector")
				return nil, nil
			},
			activeFn: func(_ context.Context, input services.SprintSelectionInput) (*services.SprintSelection, error) {
				require.Empty(t, input.SprintKey)
				return &services.SprintSelection{SprintKey: "S002", Items: []*services.BacklogItemView{{Key: "T-E19-F09-001", EntityType: "task"}}}, nil
			},
		}
	}

	cmd := newPlanCommand()
	cmd.SetContext(context.Background())
	var runErr error
	output := capturingOutput(func() { runErr = runPlan(cmd, []string{"sprint"}) })
	require.NoError(t, runErr)
	require.True(t, strings.Contains(output, "select_item"), output)
}

func TestRunPlanSprintSelectionPreservesExpansionMarker(t *testing.T) {
	original := planGetSprintSelector
	defer func() { planGetSprintSelector = original }()
	planGetSprintSelector = func() sprintSelector {
		return sprintSelectorStub{
			selectFn: func(context.Context, services.SprintSelectionInput) (*services.SprintSelection, error) {
				return nil, nil
			},
			activeFn: func(context.Context, services.SprintSelectionInput) (*services.SprintSelection, error) {
				return &services.SprintSelection{SprintKey: "S002", Items: []*services.BacklogItemView{
					{Key: "E19-F09", EntityType: "feature", RequiresExpansion: true},
				}}, nil
			},
		}
	}

	cmd := newPlanCommand()
	cmd.SetContext(context.Background())
	var runErr error
	output := capturingOutput(func() { runErr = runPlan(cmd, []string{"sprint"}) })
	require.NoError(t, runErr)
	var response SprintPlanSelectionResponse
	require.NoError(t, json.Unmarshal([]byte(output), &response), output)
	require.True(t, response.Entity.RequiresExpansion)
}
