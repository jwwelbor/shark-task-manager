package services

// Regression tests for the positional-selection defect class: call sites that
// used to pick element [0] of an incidentally-ordered slice now go through the
// named selectors (config/workflow/selectors.go). Each test workflow names its
// candidates so alphabetical order contradicts semantic order — an
// "aaa_wrong_*" twin always sorts first — so a regression back to a positional
// pick chooses the wrong status and fails loudly.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// twoAggregationFeatureConfig returns a .sharkconfig.json feature workflow
// with TWO aggregation steps. With primaryTagged, "reopen" (alphabetically
// second) carries primary: true; without, the selection is ambiguous.
func twoAggregationFeatureConfig(primaryTagged bool) string {
	primary := ""
	if primaryTagged {
		primary = `"primary": true,`
	}
	return fmt.Sprintf(`{
		"feature_workflow": {
			"start": "draft",
			"steps": {
				"draft": {"phase": "planning", "outcomes": {"pass": "reopen", "fail": "draft", "blocked": "draft"}},
				"aaa_wrong_reopen": {"phase": "development", "aggregates_from": "tasks", "outcomes": {"pass": "completed", "fail": "aaa_wrong_reopen", "blocked": "aaa_wrong_reopen"}},
				"reopen": {"phase": "development", "aggregates_from": "tasks", %s "outcomes": {"pass": "completed", "fail": "reopen", "blocked": "reopen"}},
				"completed": {"phase": "done", "terminal": true}
			}
		}
	}`, primary)
}

// --- Site #1: TaskService.legacyMaybeReopenParentFeature -------------------

// A workflow with no aggregation statuses must skip the auto-reopen (warn and
// return) instead of indexing into an empty slice or guessing a target.
func TestLegacyMaybeReopenParentFeature_NoAggregationStatuses_SkipsReopen(t *testing.T) {
	ctx := context.Background()

	// Legacy feature workflow with a terminal "completed" and no _aggregation_.
	wfSvc := newFeatureProgressWorkflowService(t, `{
		"feature_workflow": {
			"status_flow_version": "1.0",
			"status_flow": {
				"draft": ["active"],
				"active": ["completed"],
				"completed": []
			},
			"status_metadata": {
				"draft": {"phase": "planning"},
				"active": {"phase": "development"},
				"completed": {"phase": "done"}
			},
			"special_statuses": {
				"_start_": ["draft"],
				"_complete_": ["completed"]
			}
		}
	}`)

	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				BaseEntity: models.BaseEntity{ID: 7, Key: "E01-F01"},
				Status:     models.FeatureStatusCompleted,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			t.Errorf("UpdateFeature must not be called when the workflow has no aggregation statuses (attempted status %q)", feature.Status)
			return nil
		},
	}
	entitySvc := NewEntityService(wfSvc)
	featureSvc := NewFeatureService(repo, entitySvc, featureRepoAsEntityRepo(repo), nil, nil)

	taskSvc := NewTaskService(&MockTaskRepository{}, entitySvc, nil)
	taskSvc.SetFeatureService(featureSvc)

	// Must not panic and must not reopen.
	taskSvc.legacyMaybeReopenParentFeature(ctx, "E01-F01", "T-E01-F01-001")
}

// --- Sites #2/#3: reopen target picks the designated aggregation status ----

func TestResolveReopenTarget_PicksPrimaryTaggedAggregation(t *testing.T) {
	ctx := context.Background()

	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
			return "", false, nil // no history: force the aggregation fallback
		},
	}

	wfSvc := newFeatureProgressWorkflowService(t, twoAggregationFeatureConfig(true))
	levelWf := (&workflowProviderAdapter{svc: wfSvc}).ForLevel("feature")

	status, fallbackKind, err := resolveReopenTarget(ctx, histQuerier, models.EntityTypeFeature, 42, levelWf)
	require.NoError(t, err)
	assert.Equal(t, "aggregation", fallbackKind)
	assert.Equal(t, "reopen", status, "must pick the primary-tagged step, not the alphabetical first")
}

func TestResolveReopenTarget_AmbiguousAggregation_Errors(t *testing.T) {
	ctx := context.Background()

	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
			return "", false, nil
		},
	}

	wfSvc := newFeatureProgressWorkflowService(t, twoAggregationFeatureConfig(false))
	levelWf := (&workflowProviderAdapter{svc: wfSvc}).ForLevel("feature")

	_, _, err := resolveReopenTarget(ctx, histQuerier, models.EntityTypeFeature, 42, levelWf)
	require.Error(t, err, "ambiguous aggregation designation must surface, never resolve to an arbitrary pick")
	assert.Contains(t, err.Error(), "primary: true")
}

// --- Site #2: FeatureProgressService reopen-on-regression ------------------

func TestFeatureProgress_ReopensToPrimaryTaggedAggregation(t *testing.T) {
	ctx := context.Background()

	svc, featureRepo, featureID := setupFeatureProgressScenario(t,
		twoAggregationFeatureConfig(true),
		models.FeatureStatusCompleted, false,
		[]models.TaskStatus{"todo"}, // <100% progress: completed feature must reopen
	)

	require.NoError(t, svc.RecalculateAndSetProgress(ctx, featureID))

	feature, err := featureRepo.GetByID(ctx, featureID)
	require.NoError(t, err)
	assert.Equal(t, models.FeatureStatus("reopen"), feature.Status,
		"must reopen to the primary-tagged step, not the alphabetical first")
}

func TestFeatureProgress_AmbiguousAggregation_ErrorsWithoutStatusChange(t *testing.T) {
	ctx := context.Background()

	svc, featureRepo, featureID := setupFeatureProgressScenario(t,
		twoAggregationFeatureConfig(false),
		models.FeatureStatusCompleted, false,
		[]models.TaskStatus{"todo"},
	)

	err := svc.RecalculateAndSetProgress(ctx, featureID)
	require.Error(t, err, "ambiguous designation must hard-fail")
	assert.Contains(t, err.Error(), "primary: true")

	feature, getErr := featureRepo.GetByID(ctx, featureID)
	require.NoError(t, getErr)
	assert.Equal(t, models.FeatureStatusCompleted, feature.Status,
		"an ambiguous config must never change the entity's status")
}

// --- Sites #4/#5: sprint lifecycle statuses --------------------------------

// twoExecutionSprintWorkflow writes a sprint workflow with TWO execution-phase
// steps into a temp project root and returns a workflow.Service for it. With
// primaryTagged, "burn" (alphabetically second) carries primary: true.
func twoExecutionSprintWorkflow(t *testing.T, primaryTagged bool) *workflow.Service {
	t.Helper()

	primary := ""
	if primaryTagged {
		primary = "\n    primary: true"
	}
	sprintYAML := fmt.Sprintf(`version: "1.0"
start: planning
steps:
  planning:
    phase: planning
    action: advance_status
    outcomes:
      pass: burn
      fail: planning
      blocked: planning
  aaa_wrong_active:
    phase: execution
    action: advance_status
    outcomes:
      pass: closing
      fail: aaa_wrong_active
      blocked: aaa_wrong_active
  burn:
    phase: execution%s
    action: advance_status
    outcomes:
      pass: closing
      fail: burn
      blocked: burn
  closing:
    phase: review
    action: advance_status
    outcomes:
      pass: archived
      fail: closing
      blocked: closing
  archived:
    phase: done
    action: archive
    terminal: true
`, primary)

	projectRoot := t.TempDir()
	workflowDir := filepath.Join(projectRoot, "shark-data", "workflow")
	require.NoError(t, os.MkdirAll(workflowDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workflowDir, "sprint.yaml"), []byte(sprintYAML), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, ".sharkconfig.json"),
		[]byte(`{"workflow_config": "shark-data/workflow"}`), 0o644))

	config.ClearWorkflowCache()
	t.Cleanup(config.ClearWorkflowCache)

	return workflow.NewService(projectRoot)
}

func TestSprintService_StartSprint_PicksPrimaryExecutionStatus(t *testing.T) {
	ctx := context.Background()
	workflowSvc := twoExecutionSprintWorkflow(t, true)

	var updatedTo models.SprintStatus
	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return &models.Sprint{ID: 1, Key: "S001", Name: "Sprint 1", Status: "planning"}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id int64, status models.SprintStatus) error {
			updatedTo = status
			return nil
		},
	}
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	_, err := svc.StartSprint(ctx, "S001")
	require.NoError(t, err)
	assert.Equal(t, models.SprintStatus("burn"), updatedTo,
		"must start into the primary-tagged execution status, not the alphabetical first")
}

func TestSprintService_StartSprint_AmbiguousExecutionPhase_ErrorsWithoutStatusChange(t *testing.T) {
	ctx := context.Background()
	workflowSvc := twoExecutionSprintWorkflow(t, false)

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return &models.Sprint{ID: 1, Key: "S001", Name: "Sprint 1", Status: "planning"}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id int64, status models.SprintStatus) error {
			t.Errorf("UpdateStatus must not be called on an ambiguous workflow (attempted %q)", status)
			return nil
		},
	}
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	_, err := svc.StartSprint(ctx, "S001")
	require.Error(t, err, "a mid-flight sprint on an ambiguous workflow must get the error")
	// The error names the candidates and the fix.
	for _, want := range []string{"aaa_wrong_active", "burn", "primary: true"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error message missing %q: %s", want, err.Error())
		}
	}
}
