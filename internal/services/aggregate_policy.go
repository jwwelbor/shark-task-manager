package services

import (
	"errors"
	"fmt"
	"math"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/progress"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// calculateFeatureProgressInfo is the single pure policy for deriving feature
// progress metrics from a task-status breakdown.
func calculateFeatureProgressInfo(key string, breakdown map[models.TaskStatus]int, workflowSvc *workflow.Service) *FeatureProgressInfo {
	statusCounts := make(map[string]int, len(breakdown))
	taskWorkflow := workflowSvc.ForLevel(workflow.LevelTask)
	totalTasks := 0
	completedTasks := 0
	for status, count := range breakdown {
		statusCounts[string(status)] = count
		if taskWorkflow.GetStatusMetadata(string(status)).ExcludeFromProgress {
			continue
		}
		totalTasks += count
		if taskWorkflow.IsTerminalStatus(string(status)) {
			completedTasks += count
		}
	}

	progressInfo := progress.CalculateProgress(statusCounts, taskWorkflow.GetWorkflow())
	completionPct := 0.0
	if totalTasks > 0 {
		completionPct = (float64(completedTasks) / float64(totalTasks)) * 100.0
	}

	return &FeatureProgressInfo{
		FeatureKey:         key,
		WeightedProgress:   math.Round(progressInfo.WeightedPct*100) / 100,
		CompletionProgress: math.Round(completionPct*100) / 100,
		TotalTasks:         totalTasks,
		CompletedTasks:     completedTasks,
		WeightedRatio:      progressInfo.WeightedRatio,
		CompletionRatio:    fmt.Sprintf("%d/%d", completedTasks, totalTasks),
	}
}

// deriveFeatureProgressStatus is the single pure policy for status changes
// caused by a feature-progress refresh.
func deriveFeatureProgressStatus(feature *models.Feature, progressInfo *FeatureProgressInfo, workflowSvc *workflow.Service) (models.FeatureStatus, error) {
	if feature.StatusOverride || progressInfo == nil || progressInfo.TotalTasks == 0 {
		return feature.Status, nil
	}

	featureWorkflow := workflowSvc.ForLevel(workflow.LevelFeature)
	if progressInfo.WeightedProgress >= 100.0 {
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
