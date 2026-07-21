package services

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateFeatureProgressInfo_ExcludesConfiguredStatuses(t *testing.T) {
	workflowSvc := newFeatureProgressWorkflowService(t, `{
		"task_workflow": {
			"special_statuses": {"_start_": ["todo"], "_complete_": ["completed"]},
			"status_flow": {"todo": ["completed"], "completed": [], "cancelled": []},
			"status_metadata": {
				"todo": {"progress_weight": 0.0},
				"completed": {"progress_weight": 1.0},
				"cancelled": {"progress_weight": 1.0, "exclude_from_progress": true}
			}
		},
		"feature_workflow": {
			"special_statuses": {"_start_": ["draft"], "_complete_": ["completed", "archived"], "_aggregation_": ["active"]},
			"status_flow": {"draft": ["active"], "active": ["completed"], "completed": ["archived"], "archived": []},
			"status_metadata": {"active": {"orchestrator_action": {"action": "cascade"}}}
		}
	}`)

	info := calculateFeatureProgressInfo("E01-F01", map[models.TaskStatus]int{
		"todo": 1, "completed": 1, "cancelled": 5,
	}, workflowSvc)

	assert.Equal(t, 2, info.TotalTasks)
	assert.Equal(t, 1, info.CompletedTasks)
	assert.Equal(t, "1/2", info.CompletionRatio)
	assert.Equal(t, 50.0, info.WeightedProgress)
	assert.Equal(t, 50.0, info.CompletionProgress)
}

func TestDeriveFeatureProgressStatus_Boundaries(t *testing.T) {
	workflowSvc := newFeatureProgressWorkflowService(t, defaultFeatureProgressWorkflowConfig())
	tests := []struct {
		name     string
		feature  *models.Feature
		progress *FeatureProgressInfo
		want     models.FeatureStatus
	}{
		{
			name:     "manual override wins",
			feature:  &models.Feature{Status: models.FeatureStatusActive, StatusOverride: true},
			progress: &FeatureProgressInfo{TotalTasks: 1, WeightedProgress: 100},
			want:     models.FeatureStatusActive,
		},
		{
			name:     "cascade-eligible status advances at 100 percent",
			feature:  &models.Feature{Status: models.FeatureStatusActive},
			progress: &FeatureProgressInfo{TotalTasks: 1, WeightedProgress: 100},
			want:     models.FeatureStatusCompleted,
		},
		{
			name:     "non-cascade status does not advance",
			feature:  &models.Feature{Status: models.FeatureStatusDraft},
			progress: &FeatureProgressInfo{TotalTasks: 1, WeightedProgress: 100},
			want:     models.FeatureStatusDraft,
		},
		{
			name:     "completed feature reopens when progress regresses",
			feature:  &models.Feature{Status: models.FeatureStatusCompleted},
			progress: &FeatureProgressInfo{TotalTasks: 1, WeightedProgress: 50},
			want:     models.FeatureStatusActive,
		},
		{
			name:     "other terminal feature is preserved",
			feature:  &models.Feature{Status: models.FeatureStatusArchived},
			progress: &FeatureProgressInfo{TotalTasks: 1, WeightedProgress: 50},
			want:     models.FeatureStatusArchived,
		},
		{
			name:     "empty child set preserves current status",
			feature:  &models.Feature{Status: models.FeatureStatusActive},
			progress: &FeatureProgressInfo{},
			want:     models.FeatureStatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deriveFeatureProgressStatus(tt.feature, tt.progress, workflowSvc)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
