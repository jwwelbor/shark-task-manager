package services

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
)

func TestCalculateEpicProgress(t *testing.T) {
	tests := []struct {
		name string
		data []repository.FeatureProgressData
		want float64
	}{
		{
			name: "nil rows",
			data: nil,
			want: 0,
		},
		{
			name: "empty rows",
			data: []repository.FeatureProgressData{},
			want: 0,
		},
		{
			name: "cancelled-only rows are excluded",
			data: []repository.FeatureProgressData{
				{Status: "cancelled", ProgressPct: 0},
				{Status: "cancelled", ProgressPct: 100},
			},
			want: 0,
		},
		{
			name: "completed overrides stored progress",
			data: []repository.FeatureProgressData{
				{Status: "completed", ProgressPct: 0},
			},
			want: 100,
		},
		{
			name: "archived overrides stored progress",
			data: []repository.FeatureProgressData{
				{Status: "archived", ProgressPct: 25},
			},
			want: 100,
		},
		{
			name: "mixed rows use stored and forced-complete contributions",
			data: []repository.FeatureProgressData{
				{Status: "active", ProgressPct: 50},
				{Status: "completed", ProgressPct: 75},
				{Status: "draft", ProgressPct: 0},
				{Status: "archived", ProgressPct: 10},
				{Status: "cancelled", ProgressPct: 100},
			},
			want: 62.5,
		},
		{
			name: "stored progress boundaries preserve the range invariant",
			data: []repository.FeatureProgressData{
				{Status: "active", ProgressPct: 0},
				{Status: "active", ProgressPct: 100},
			},
			want: 50,
		},
		{
			name: "repeating average remains unrounded",
			data: []repository.FeatureProgressData{
				{Status: "draft", ProgressPct: 0},
				{Status: "active", ProgressPct: 0},
				{Status: "active", ProgressPct: 100},
			},
			want: 100.0 / 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.want, calculateEpicProgress(tt.data), 1e-12)
		})
	}
}

func TestCalculateEpicProgressWithWorkflow_UsesConfiguredTerminalAndExclusion(t *testing.T) {
	featureWorkflow := &config.WorkflowConfig{
		Steps: map[string]*config.Step{
			"delivered": {Terminal: true, Phase: "done"},
			"discarded": {ExcludeFromProgress: true},
		},
		StatusMetadata: map[string]config.StatusMetadata{
			"delivered": {Phase: "done"},
			"discarded": {ExcludeFromProgress: true},
			"active":    {Phase: "development"},
		},
	}
	featureWorkflow.DeriveLegacy()
	svc := workflow.NewServiceFromMultiLevel(&config.MultiLevelWorkflow{Feature: featureWorkflow})
	data := []repository.FeatureProgressData{{Status: "delivered", ProgressPct: 0}, {Status: "discarded", ProgressPct: 100}, {Status: "active", ProgressPct: 40}}
	assert.Equal(t, 70.0, calculateEpicProgressWithWorkflow(data, svc.ForLevel(workflow.LevelFeature)))
}
