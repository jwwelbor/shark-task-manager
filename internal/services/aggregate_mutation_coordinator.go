package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// AggregateMutationRepository is the service-owned persistence contract for
// transaction-aware hierarchy maintenance.
type AggregateMutationRepository interface {
	GetFeatureForProgressTx(ctx context.Context, tx *sql.Tx, id int64) (*models.Feature, error)
	GetTaskStatusBreakdownTx(ctx context.Context, tx *sql.Tx, featureID int64) (map[models.TaskStatus]int, error)
	UpdateFeatureProgressAndStatusTx(ctx context.Context, tx *sql.Tx, featureID int64, progress float64, status models.FeatureStatus) error
	GetEpicForStatusTx(ctx context.Context, tx *sql.Tx, id int64) (*models.Epic, error)
	GetFeatureStatusBreakdownTx(ctx context.Context, tx *sql.Tx, epicID int64) (map[models.FeatureStatus]int, error)
	UpdateEpicStatusTx(ctx context.Context, tx *sql.Tx, epicID int64, status models.EpicStatus) error
	GetLastNonTerminalStatusTx(ctx context.Context, tx *sql.Tx, entityType models.EntityType, entityID int64, terminalStatuses []string) (string, bool, error)
	CreateEntityHistoryTx(ctx context.Context, tx *sql.Tx, history *models.EntityHistory) error
}

// AggregateMutationCoordinator owns the post-mutation hierarchy invariant inside
// a caller-owned transaction: task mutation -> feature cache/status -> epic status.
type AggregateMutationCoordinator struct {
	repo        AggregateMutationRepository
	workflowSvc *workflow.Service
}

// NewAggregateMutationCoordinator creates a coordinator for progress-affecting writes.
func NewAggregateMutationCoordinator(repo AggregateMutationRepository, workflowSvc *workflow.Service) *AggregateMutationCoordinator {
	requireNonNil(repo, "AggregateMutationCoordinator requires a non-nil repository")
	requireNonNil(workflowSvc, "AggregateMutationCoordinator requires a non-nil workflow.Service")
	return &AggregateMutationCoordinator{repo: repo, workflowSvc: workflowSvc}
}

// RefreshFeatureAndEpic recalculates and persists a feature cache/status and
// then derives its parent epic status in the same transaction. The trigger is
// used to preserve history-based reopen semantics for creation and regression.
func (c *AggregateMutationCoordinator) RefreshFeatureAndEpic(ctx context.Context, tx *sql.Tx, featureID int64, trigger cascadeTrigger) error {
	return c.refreshFeature(ctx, tx, featureID, trigger, true)
}

// RefreshFeature recalculates and persists only the feature cache/status. It is
// used when the caller owns the final epic status write, such as forced epic
// completion, so intermediate child refreshes cannot move the epic repeatedly.
func (c *AggregateMutationCoordinator) RefreshFeature(ctx context.Context, tx *sql.Tx, featureID int64, trigger cascadeTrigger) error {
	return c.refreshFeature(ctx, tx, featureID, trigger, false)
}

func (c *AggregateMutationCoordinator) refreshFeature(ctx context.Context, tx *sql.Tx, featureID int64, trigger cascadeTrigger, refreshEpic bool) error {
	if tx == nil {
		return fmt.Errorf("refresh feature and epic: transaction is required")
	}
	if featureID == 0 {
		return nil
	}

	feature, err := c.repo.GetFeatureForProgressTx(ctx, tx, featureID)
	if err != nil {
		return fmt.Errorf("refresh feature %d: load feature: %w", featureID, err)
	}
	breakdown, err := c.repo.GetTaskStatusBreakdownTx(ctx, tx, featureID)
	if err != nil {
		return fmt.Errorf("refresh feature %s: load task breakdown: %w", feature.Key, err)
	}

	progressInfo := calculateFeatureProgressInfo(feature.Key, breakdown, c.workflowSvc)
	status, err := deriveFeatureProgressStatus(feature, progressInfo, c.workflowSvc)
	if err != nil {
		return fmt.Errorf("refresh feature %s: derive status: %w", feature.Key, err)
	}

	featureWorkflow := c.workflowSvc.ForLevel(workflow.LevelFeature)
	isReopen := featureWorkflow.IsTerminalStatus(string(feature.Status)) &&
		(triggerReopensAncestors(trigger) || !featureWorkflow.IsTerminalStatus(string(status)))
	fallbackKind := ""
	if isReopen {
		var reopenStatus string
		reopenStatus, fallbackKind, err = c.resolveReopenTargetTx(ctx, tx, models.EntityTypeFeature, feature.ID, featureWorkflow)
		if err != nil {
			return fmt.Errorf("refresh feature %s: resolve reopen target: %w", feature.Key, err)
		}
		status = models.FeatureStatus(reopenStatus)
	}

	if err := c.repo.UpdateFeatureProgressAndStatusTx(ctx, tx, feature.ID, progressInfo.WeightedProgress, status); err != nil {
		return fmt.Errorf("refresh feature %s: persist progress and status: %w", feature.Key, err)
	}
	if status != feature.Status {
		notes := aggregateStatusNotes(trigger, isReopen, fallbackKind)
		if err := c.recordStatusChangeTx(ctx, tx, models.EntityTypeFeature, feature.ID, string(feature.Status), string(status), notes); err != nil {
			return fmt.Errorf("refresh feature %s: record status history: %w", feature.Key, err)
		}
		if isReopen {
			trigger = cascadeTrigger{
				triggerKey:  feature.Key,
				triggerKind: "regression",
				triggerType: models.EntityTypeFeature,
				startLeg:    cascadeLegEpic,
				epicID:      feature.EpicID,
			}
		}
	}
	if refreshEpic {
		if err := c.RefreshEpicStatus(ctx, tx, feature.EpicID, trigger); err != nil {
			return fmt.Errorf("refresh feature %s parent epic: %w", feature.Key, err)
		}
	}
	return nil
}

// RefreshEpicStatus derives and persists an epic's status from its feature rows.
func (c *AggregateMutationCoordinator) RefreshEpicStatus(ctx context.Context, tx *sql.Tx, epicID int64, trigger cascadeTrigger) error {
	if tx == nil {
		return fmt.Errorf("refresh epic: transaction is required")
	}
	if epicID == 0 {
		return nil
	}

	epic, err := c.repo.GetEpicForStatusTx(ctx, tx, epicID)
	if err != nil {
		return fmt.Errorf("refresh epic %d: load epic: %w", epicID, err)
	}
	breakdown, err := c.repo.GetFeatureStatusBreakdownTx(ctx, tx, epicID)
	if err != nil {
		return fmt.Errorf("refresh epic %s: load feature breakdown: %w", epic.Key, err)
	}

	status := deriveEpicStatusFromFeatures(breakdown, epic.Status, c.workflowSvc)
	epicWorkflow := c.workflowSvc.ForLevel(workflow.LevelEpic)
	// A creation or regression only reopens a terminal epic when the current
	// feature breakdown contains a non-terminal child. deriveEpicStatusFromFeatures
	// intentionally preserves a terminal epic status, so its result cannot be
	// used to distinguish a completed child import from a real regression.
	isReopen := epicWorkflow.IsTerminalStatus(string(epic.Status)) &&
		hasNonTerminalFeature(breakdown, c.workflowSvc) &&
		triggerReopensAncestors(trigger)
	fallbackKind := ""
	if isReopen {
		status, fallbackKind, err = c.resolveEpicReopenTargetTx(ctx, tx, epic, epicWorkflow)
		if err != nil {
			return fmt.Errorf("refresh epic %s: resolve reopen target: %w", epic.Key, err)
		}
	}
	if status == epic.Status {
		return nil
	}
	if err := c.repo.UpdateEpicStatusTx(ctx, tx, epic.ID, status); err != nil {
		return fmt.Errorf("refresh epic %s: persist status: %w", epic.Key, err)
	}
	notes := aggregateStatusNotes(trigger, isReopen, fallbackKind)
	if err := c.recordStatusChangeTx(ctx, tx, models.EntityTypeEpic, epic.ID, string(epic.Status), string(status), notes); err != nil {
		return fmt.Errorf("refresh epic %s: record status history: %w", epic.Key, err)
	}
	return nil
}

func hasNonTerminalFeature(breakdown map[models.FeatureStatus]int, workflowSvc *workflow.Service) bool {
	featureWorkflow := workflowSvc.ForLevel(workflow.LevelFeature)
	for status, count := range breakdown {
		if count > 0 && !featureWorkflow.IsTerminalStatus(string(status)) {
			return true
		}
	}
	return false
}

func (c *AggregateMutationCoordinator) resolveEpicReopenTargetTx(ctx context.Context, tx *sql.Tx, epic *models.Epic, epicWorkflow *workflow.Service) (models.EpicStatus, string, error) {
	status, fallbackKind, err := c.resolveReopenTargetTx(ctx, tx, models.EntityTypeEpic, epic.ID, epicWorkflow)
	return models.EpicStatus(status), fallbackKind, err
}

func (c *AggregateMutationCoordinator) resolveReopenTargetTx(ctx context.Context, tx *sql.Tx, entityType models.EntityType, entityID int64, levelWf levelWorkflow) (string, string, error) {
	status, found, err := c.repo.GetLastNonTerminalStatusTx(ctx, tx, entityType, entityID, levelWf.GetTerminalStatuses())
	if err != nil {
		return "", "", fmt.Errorf("query history: %w", err)
	}
	if found {
		return status, "", nil
	}
	status, err = levelWf.PrimaryAggregationStatus()
	if err == nil {
		return status, "aggregation", nil
	}
	if isNoCandidateError(err) {
		return levelWf.GetInitialStatusString(), "initial", nil
	}
	return "", "", err
}

func (c *AggregateMutationCoordinator) recordStatusChangeTx(ctx context.Context, tx *sql.Tx, entityType models.EntityType, entityID int64, fromStatus, toStatus, notes string) error {
	changedBy := "system"
	return c.repo.CreateEntityHistoryTx(ctx, tx, &models.EntityHistory{
		EntityType: entityType,
		EntityID:   entityID,
		FromStatus: &fromStatus,
		ToStatus:   toStatus,
		ChangedBy:  &changedBy,
		Notes:      &notes,
		ChangedAt:  time.Now(),
	})
}

func triggerReopensAncestors(trigger cascadeTrigger) bool {
	return trigger.triggerKind == "creation" || trigger.triggerKind == "regression"
}

func aggregateStatusNotes(trigger cascadeTrigger, reopen bool, fallbackKind string) string {
	if reopen {
		return buildAutoReopenNotes(trigger, fallbackKind)
	}
	if trigger.triggerKey == "" {
		return "aggregate_refresh: child state changed"
	}
	return fmt.Sprintf("aggregate_refresh: %s (%s)", trigger.triggerKey, trigger.triggerType)
}

func isNoCandidateError(err error) bool {
	var noCandidate *config.NoCandidateError
	return errors.As(err, &noCandidate)
}

// rollbackAfterAggregateMutation is safe both before and after commit. A
// post-commit Rollback returns sql.ErrTxDone, which is intentionally ignored.
func rollbackAfterAggregateMutation(tx *sql.Tx) {
	_ = tx.Rollback()
}
