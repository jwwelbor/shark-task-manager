package dispatch

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stepTransitioner struct {
	info *services.NextStatusInfo
}

func (s stepTransitioner) TransitionStatus(context.Context, string, string, services.TransitionOptions) (*services.TransitionResult, error) {
	return nil, errors.New("transition not expected during resolution")
}

func (s stepTransitioner) GetNextStatus(context.Context, string) (*services.NextStatusInfo, error) {
	return s.info, nil
}

type stepPlaceholders struct {
	vars map[string]string
}

func (s stepPlaceholders) GeneratePlaceholders(context.Context, string) (map[string]string, error) {
	return s.vars, nil
}

type stepPromptAssembler struct {
	prompt string
}

func (s stepPromptAssembler) AssemblePrompt(context.Context, PromptAssemblyInput) (string, error) {
	return s.prompt, nil
}

func TestStepResolver_Resolve_TC010(t *testing.T) {
	tests := []struct {
		name           string
		entityType     models.EntityType
		status         string
		action         *action.PopulatedAction
		terminal       bool
		wantAction     string
		wantGate       GateClassification
		wantPrompt     string
		wantUnresolved []string
		wantAgentType  string
		wantProvider   string
		wantModel      string
		wantEffort     string
	}{
		{
			name:       "task worker step",
			entityType: models.EntityTypeTask,
			status:     "in_development",
			action: &action.PopulatedAction{
				Action:      action.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Model:       "claude-sonnet",
				Effort:      "medium",
				Instruction: "implement {task_id}",
			},
			wantAction:    action.ActionSpawnAgent,
			wantGate:      GateNone,
			wantPrompt:    "worker prompt",
			wantAgentType: "developer",
			wantProvider:  "anthropic",
			wantModel:     "claude-sonnet",
			wantEffort:    "medium",
		},
		{
			name:       "feature approval gate",
			entityType: models.EntityTypeFeature,
			status:     "awaiting_approval",
			action: &action.PopulatedAction{
				Action:      action.ActionCheckOrResume,
				Instruction: "review the feature",
			},
			wantAction: action.ActionCheckOrResume,
			wantGate:   GateHuman,
		},
		{
			name:       "pause step",
			entityType: models.EntityTypeTask,
			status:     "blocked",
			action: &action.PopulatedAction{
				Action: action.ActionPause,
			},
			wantAction: action.ActionPause,
			wantGate:   GatePause,
		},
		{
			name:       "terminal step",
			entityType: models.EntityTypeTask,
			status:     "completed",
			terminal:   true,
			wantAction: action.ActionArchive,
			wantGate:   GateTerminal,
		},
		{
			name:       "unresolved placeholder diagnostic",
			entityType: models.EntityTypeTask,
			status:     "in_development",
			action: &action.PopulatedAction{
				Action:      action.ActionSpawnAgent,
				AgentType:   "developer",
				Instruction: "implement <missing-token>",
			},
			wantAction:     action.ActionSpawnAgent,
			wantGate:       GateNone,
			wantPrompt:     "prompt with <missing-token>",
			wantUnresolved: []string{"<missing-token>"},
			wantAgentType:  "developer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var populated *action.PopulatedAction
			if tt.action != nil {
				populated = tt.action
			}
			actions := &action.MockActionService{
				GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
					return populated, nil
				},
			}
			resolver, err := NewStepResolver(StepResolverDeps{
				Transitioner: stepTransitioner{info: &services.NextStatusInfo{
					CurrentStatus: tt.status,
					IsTerminal:    tt.terminal,
				}},
				Placeholders:  stepPlaceholders{vars: map[string]string{"task_id": "T-E38-F01-001"}},
				ActionService: actions,
				PromptAssembler: stepPromptAssembler{
					prompt: tt.wantPrompt,
				},
				IsArchivedStatus: func(models.EntityType, string) bool { return false },
			})
			require.NoError(t, err)

			got, err := resolver.Resolve(context.Background(), tt.entityType, "T-E38-F01-001")
			require.NoError(t, err)
			assert.Equal(t, tt.wantAction, got.Action)
			assert.Equal(t, tt.entityType, got.EntityType)
			assert.Equal(t, "T-E38-F01-001", got.EntityKey)
			assert.Equal(t, tt.status, got.Status)
			assert.Equal(t, tt.wantGate, got.GateClassification)
			assert.Equal(t, tt.wantPrompt, got.Prompt)
			assert.Equal(t, tt.wantUnresolved, got.UnresolvedPlaceholders)
			assert.Equal(t, tt.wantAgentType, got.AgentType)
			assert.Equal(t, tt.wantProvider, got.Provider)
			assert.Equal(t, tt.wantModel, got.Model)
			assert.Equal(t, tt.wantEffort, got.Effort)
		})
	}
}

func TestStepResolver_PropagatesActionProviderFailure_TC010(t *testing.T) {
	wantErr := errors.New("provider configuration unavailable")
	resolver, err := NewStepResolver(StepResolverDeps{
		Transitioner: stepTransitioner{info: &services.NextStatusInfo{CurrentStatus: "in_development"}},
		ActionService: &action.MockActionService{
			GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
				return nil, wantErr
			},
		},
	})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background(), models.EntityTypeTask, "T-E38-F01-001")
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
	assert.True(t, strings.Contains(err.Error(), "populate action"), "error should identify the resolution phase: %v", err)
}

func TestStepResolver_PropagatesPromptAssemblyFailure_TC010(t *testing.T) {
	wantErr := errors.New("prompt renderer unavailable")
	resolver, err := NewStepResolver(StepResolverDeps{
		Transitioner: stepTransitioner{info: &services.NextStatusInfo{CurrentStatus: "in_development"}},
		ActionService: &action.MockActionService{
			GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
				return &action.PopulatedAction{Action: action.ActionSpawnAgent, AgentType: "developer", Instruction: "implement"}, nil
			},
		},
		PromptAssembler: PromptAssemblerFunc(func(context.Context, PromptAssemblyInput) (string, error) {
			return "", wantErr
		}),
	})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background(), models.EntityTypeTask, "T-E38-F01-001")
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
	assert.Contains(t, err.Error(), "assemble dispatch prompt")
}
