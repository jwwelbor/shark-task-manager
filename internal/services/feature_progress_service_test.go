package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFeatureProgressWorkflowService(t *testing.T) *workflow.Service {
	t.Helper()

	config.ClearWorkflowCache()

	tmpDir := t.TempDir()
	configData := `{
		"task_workflow": {
			"status_flow_version": "1.0",
			"special_statuses": {
				"_start_": ["todo"],
				"_complete_": ["completed"]
			},
			"status_flow": {
				"todo": ["completed"],
				"completed": []
			},
			"status_metadata": {
				"todo": {
					"phase": "planning",
					"progress_weight": 0.0
				},
				"completed": {
					"phase": "done",
					"progress_weight": 1.0
				}
			}
		},
		"feature_workflow": {
			"status_flow_version": "1.0",
			"special_statuses": {
				"_start_": ["draft"],
				"_complete_": ["completed", "archived"],
				"_aggregation_": ["active"]
			},
			"status_flow": {
				"draft": ["active", "archived"],
				"active": ["completed", "archived"],
				"completed": ["archived"],
				"archived": []
			}
		}
	}`

	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath, []byte(configData), 0o644))

	t.Cleanup(func() {
		config.ClearWorkflowCache()
	})

	return workflow.NewService(tmpDir)
}

func setupFeatureProgressScenario(t *testing.T, featureStatus models.FeatureStatus, override bool, taskStatuses []models.TaskStatus) (*FeatureProgressService, *repository.FeatureRepository, int64) {
	t.Helper()

	ctx := context.Background()
	db := repository.NewDB(test.NewIsolatedTestDB(t))
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	workflowSvc := newFeatureProgressWorkflowService(t)

	epic := &models.Epic{
		BaseEntity: models.BaseEntity{
			Key:   "E19",
			Title: "Feature Progress Test Epic",
		},
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	require.NoError(t, epicRepo.Create(ctx, epic))

	feature := &models.Feature{
		BaseEntity: models.BaseEntity{
			Key:   "E19-F02",
			Title: "Feature Progress Test Feature",
		},
		EpicID:         epic.ID,
		Status:         featureStatus,
		StatusOverride: override,
	}
	require.NoError(t, featureRepo.Create(ctx, feature))

	for i, status := range taskStatuses {
		task := &models.Task{
			BaseEntity: models.BaseEntity{
				Key:   fmt.Sprintf("T-E19-F02-%03d", i+1),
				Title: fmt.Sprintf("Task %d", i+1),
			},
			FeatureID: feature.ID,
			Status:    status,
			AgentType: test.StringPtr("developer"),
			Priority:  5,
		}
		require.NoError(t, taskRepo.Create(ctx, task))
	}

	return NewFeatureProgressService(featureRepo, taskRepo, workflowSvc), featureRepo, feature.ID
}

func TestFeatureProgressService_RecalculateAndSetProgress_StatusDerivation(t *testing.T) {
	tests := []struct {
		name           string
		featureStatus  models.FeatureStatus
		statusOverride bool
		taskStatuses   []models.TaskStatus
		wantStatus     models.FeatureStatus
		wantProgress   float64
	}{
		{
			name:          "downgrades completed when progress drops below 100",
			featureStatus: models.FeatureStatusCompleted,
			taskStatuses:  []models.TaskStatus{"completed", "todo", "todo", "todo"},
			wantStatus:    models.FeatureStatusActive,
			wantProgress:  25.0,
		},
		{
			name:          "preserves completed when all tasks are done",
			featureStatus: models.FeatureStatusCompleted,
			taskStatuses:  []models.TaskStatus{"completed", "completed", "completed", "completed"},
			wantStatus:    models.FeatureStatusCompleted,
			wantProgress:  100.0,
		},
		{
			name:           "respects status override",
			featureStatus:  models.FeatureStatusCompleted,
			statusOverride: true,
			taskStatuses:   []models.TaskStatus{"completed", "todo", "todo", "todo"},
			wantStatus:     models.FeatureStatusCompleted,
			wantProgress:   25.0,
		},
		{
			name:          "preserves manual terminal archived",
			featureStatus: models.FeatureStatusArchived,
			taskStatuses:  []models.TaskStatus{"completed", "todo", "todo", "todo"},
			wantStatus:    models.FeatureStatusArchived,
			wantProgress:  25.0,
		},
		{
			name:          "empty task list does not flip status",
			featureStatus: models.FeatureStatusCompleted,
			taskStatuses:  nil,
			wantStatus:    models.FeatureStatusCompleted,
			wantProgress:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, featureRepo, featureID := setupFeatureProgressScenario(t, tt.featureStatus, tt.statusOverride, tt.taskStatuses)

			require.NoError(t, svc.RecalculateAndSetProgress(context.Background(), featureID))

			updated, err := featureRepo.GetByID(context.Background(), featureID)
			require.NoError(t, err)
			require.NotNil(t, updated)

			assert.Equal(t, tt.wantStatus, updated.Status)
			assert.InDelta(t, tt.wantProgress, updated.ProgressPct, 0.001)
		})
	}
}
