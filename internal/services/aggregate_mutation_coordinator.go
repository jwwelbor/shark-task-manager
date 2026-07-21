package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/progress"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// AggregateMutationCoordinator owns the post-mutation hierarchy invariant inside
// a caller-owned transaction: task mutation -> feature cache/status -> epic status.
// It is intentionally narrow: callers remain responsible for their own entity
// write and transaction boundary.
type AggregateMutationCoordinator struct {
	repo        *repository.ProgressMutationRepository
	workflowSvc *workflow.Service
}

// NewAggregateMutationCoordinator creates a coordinator for progress-affecting writes.
func NewAggregateMutationCoordinator(repo *repository.ProgressMutationRepository, workflowSvc *workflow.Service) *AggregateMutationCoordinator {
	requireNonNil(repo, "AggregateMutationCoordinator requires a non-nil repository")
	requireNonNil(workflowSvc, "AggregateMutationCoordinator requires a non-nil workflow.Service")
	return &AggregateMutationCoordinator{repo: repo, workflowSvc: workflowSvc}
}

// RefreshFeatureAndEpic recalculates and persists a feature cache/status and
// then derives its parent epic status in the same transaction.
func (c *AggregateMutationCoordinator) RefreshFeatureAndEpic(ctx context.Context, tx *sql.Tx, featureID int64) error {
	if tx == nil {
		return fmt.Errorf("aggregate mutation requires a transaction")
	}
	if featureID == 0 {
		return nil
	}

	feature, err := c.repo.GetFeatureForProgressTx(ctx, tx, featureID)
	if err != nil {
		return err
	}
	breakdown, err := c.repo.GetTaskStatusBreakdownTx(ctx, tx, featureID)
	if err != nil {
		return err
	}

	weightedProgress, totalTasks, err := c.weightedProgress(breakdown)
	if err != nil {
		return err
	}
	status, err := c.derivedFeatureStatus(feature, weightedProgress, totalTasks)
	if err != nil {
		return err
	}
	if err := c.repo.UpdateFeatureProgressAndStatusTx(ctx, tx, featureID, weightedProgress, status); err != nil {
		return err
	}
	return c.RefreshEpicStatus(ctx, tx, feature.EpicID)
}

// RefreshEpicStatus derives and persists an epic's status from its feature rows.
func (c *AggregateMutationCoordinator) RefreshEpicStatus(ctx context.Context, tx *sql.Tx, epicID int64) error {
	if tx == nil {
		return fmt.Errorf("aggregate mutation requires a transaction")
	}
	if epicID == 0 {
		return nil
	}
	epic, err := c.repo.GetEpicForStatusTx(ctx, tx, epicID)
	if err != nil {
		return err
	}
	breakdown, err := c.repo.GetFeatureStatusBreakdownTx(ctx, tx, epicID)
	if err != nil {
		return err
	}
	status := deriveEpicStatusFromFeatures(breakdown, epic.Status, c.workflowSvc)
	if status == epic.Status {
		return nil
	}
	return c.repo.UpdateEpicStatusTx(ctx, tx, epic.ID, status)
}

func (c *AggregateMutationCoordinator) weightedProgress(breakdown map[models.TaskStatus]int) (float64, int, error) {
	counts := make(map[string]int, len(breakdown))
	total := 0
	taskWorkflow := c.workflowSvc.ForLevel(workflow.LevelTask)
	for status, count := range breakdown {
		counts[string(status)] = count
		if !taskWorkflow.GetStatusMetadata(string(status)).ExcludeFromProgress {
			total += count
		}
	}
	value := progress.CalculateProgress(counts, taskWorkflow.GetWorkflow()).WeightedPct
	return math.Round(value*100) / 100, total, nil
}

func (c *AggregateMutationCoordinator) derivedFeatureStatus(feature *models.Feature, weightedProgress float64, totalTasks int) (models.FeatureStatus, error) {
	if feature.StatusOverride || totalTasks == 0 {
		return feature.Status, nil
	}
	featureWorkflow := c.workflowSvc.ForLevel(workflow.LevelFeature)
	if weightedProgress >= 100.0 {
		if featureWorkflow.IsTerminalStatus(string(feature.Status)) && feature.Status != models.FeatureStatusCompleted {
			return feature.Status, nil
		}
		if !featureWorkflow.HasOrchestratorAction(string(feature.Status), config.ActionCascade) {
			return feature.Status, nil
		}
		if nextStatus, ok := featureWorkflow.GetSingleNextStatus(string(feature.Status)); ok {
			return models.FeatureStatus(nextStatus), nil
		}
		return feature.Status, nil
	}
	if feature.Status != models.FeatureStatusCompleted {
		return feature.Status, nil
	}
	reopenStatus, err := featureWorkflow.PrimaryAggregationStatus()
	if err == nil {
		return models.FeatureStatus(reopenStatus), nil
	}
	var noCandidate *config.NoCandidateError
	if errors.As(err, &noCandidate) {
		return models.FeatureStatus(featureWorkflow.GetInitialStatusString()), nil
	}
	return "", err
}
