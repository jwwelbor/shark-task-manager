package status

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// CalculationService handles cascading status calculations
// for features and epics based on their child entities
type CalculationService struct {
	featureRepo *repository.FeatureRepository
	epicRepo    *repository.EpicRepository
	taskRepo    *repository.TaskRepository
}

// NewCalculationService creates a new StatusCalculationService
func NewCalculationService(db *repository.DB) *CalculationService {
	return &CalculationService{
		featureRepo: repository.NewFeatureRepository(db),
		epicRepo:    repository.NewEpicRepository(db),
		taskRepo:    repository.NewTaskRepository(db),
	}
}

// RecalculateFeatureStatus calculates and updates feature status from task statuses
// Returns the result of the calculation including whether status was changed
func (s *CalculationService) RecalculateFeatureStatus(ctx context.Context, featureID int64) (*StatusChangeResult, error) {
	// Get the feature
	feature, err := s.featureRepo.GetByID(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	// Get task status breakdown
	taskCounts, err := s.featureRepo.GetTaskStatusBreakdown(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown: %w", err)
	}

	// Calculate derived status
	derivedStatus := DeriveFeatureStatus(taskCounts)
	previousStatus := string(feature.Status)
	now := time.Now()

	result := &StatusChangeResult{
		EntityType:     "feature",
		EntityKey:      feature.Key,
		EntityID:       feature.ID,
		PreviousStatus: previousStatus,
		NewStatus:      string(derivedStatus),
		CalculatedAt:   now,
	}

	// Check if feature has override enabled
	if feature.StatusOverride {
		result.WasSkipped = true
		result.SkipReason = "status_override enabled"
		return result, nil
	}

	// Skip if status unchanged
	if feature.Status == derivedStatus {
		result.WasChanged = false
		return result, nil
	}

	// Update the status
	updated, err := s.featureRepo.UpdateStatusIfNotOverridden(ctx, featureID, derivedStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to update feature status: %w", err)
	}

	result.WasChanged = updated
	if !updated {
		result.WasSkipped = true
		result.SkipReason = "status_override enabled (race condition)"
	}

	return result, nil
}

// RecalculateFeatureStatusByKey calculates and updates feature status by key
func (s *CalculationService) RecalculateFeatureStatusByKey(ctx context.Context, featureKey string) (*StatusChangeResult, error) {
	feature, err := s.featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature by key: %w", err)
	}
	return s.RecalculateFeatureStatus(ctx, feature.ID)
}

// RecalculateEpicStatus calculates and updates epic status from feature statuses
func (s *CalculationService) RecalculateEpicStatus(ctx context.Context, epicID int64) (*StatusChangeResult, error) {
	// Get the epic
	epic, err := s.epicRepo.GetByID(ctx, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic: %w", err)
	}

	// Get feature status breakdown
	featureCounts, err := s.epicRepo.GetFeatureStatusBreakdown(ctx, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature status breakdown: %w", err)
	}

	// Calculate derived status
	derivedStatus := DeriveEpicStatus(featureCounts)
	previousStatus := string(epic.Status)
	now := time.Now()

	result := &StatusChangeResult{
		EntityType:     "epic",
		EntityKey:      epic.Key,
		EntityID:       epic.ID,
		PreviousStatus: previousStatus,
		NewStatus:      string(derivedStatus),
		CalculatedAt:   now,
	}

	// Skip if status unchanged
	if epic.Status == derivedStatus {
		result.WasChanged = false
		return result, nil
	}

	// Update the status (epics don't have override yet, so always update)
	err = s.epicRepo.UpdateStatus(ctx, epicID, derivedStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to update epic status: %w", err)
	}

	result.WasChanged = true
	return result, nil
}

// RecalculateEpicStatusByKey calculates and updates epic status by key
func (s *CalculationService) RecalculateEpicStatusByKey(ctx context.Context, epicKey string) (*StatusChangeResult, error) {
	epic, err := s.epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic by key: %w", err)
	}
	return s.RecalculateEpicStatus(ctx, epic.ID)
}

// CascadeFromTask triggers status recalculation up the hierarchy from a task
// Returns results for feature and epic updates
func (s *CalculationService) CascadeFromTask(ctx context.Context, taskKey string) ([]StatusChangeResult, error) {
	// Get the task to find its feature
	task, err := s.taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return s.CascadeFromFeatureID(ctx, task.FeatureID)
}

// CascadeFromFeatureID triggers status recalculation for a feature and its parent epic
func (s *CalculationService) CascadeFromFeatureID(ctx context.Context, featureID int64) ([]StatusChangeResult, error) {
	var results []StatusChangeResult

	// Recalculate feature status
	featureResult, err := s.RecalculateFeatureStatus(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to recalculate feature status: %w", err)
	}
	results = append(results, *featureResult)

	// Get feature to find epic ID
	feature, err := s.featureRepo.GetByID(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	// Recalculate epic status
	epicResult, err := s.RecalculateEpicStatus(ctx, feature.EpicID)
	if err != nil {
		return nil, fmt.Errorf("failed to recalculate epic status: %w", err)
	}
	results = append(results, *epicResult)

	return results, nil
}

// RecalculateAll recalculates status for all features and epics
// Returns a summary of all changes made
func (s *CalculationService) RecalculateAll(ctx context.Context) (*RecalculationSummary, error) {
	startedAt := time.Now()
	summary := &RecalculationSummary{
		StartedAt: startedAt,
	}

	// Get all features
	features, err := s.featureRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}

	// Recalculate each feature
	for _, feature := range features {
		result, err := s.RecalculateFeatureStatus(ctx, feature.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to recalculate feature %s: %w", feature.Key, err)
		}

		if result.WasChanged {
			summary.FeaturesUpdated++
		}
		if result.WasSkipped {
			summary.FeaturesSkipped++
		}
		summary.Changes = append(summary.Changes, *result)
	}

	// Get all epics (nil filter = all epics)
	epics, err := s.epicRepo.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list epics: %w", err)
	}

	// Recalculate each epic
	for _, epic := range epics {
		result, err := s.RecalculateEpicStatus(ctx, epic.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to recalculate epic %s: %w", epic.Key, err)
		}

		if result.WasChanged {
			summary.EpicsUpdated++
		}
		summary.Changes = append(summary.Changes, *result)
	}

	summary.CompletedAt = time.Now()
	summary.DurationMs = summary.CompletedAt.Sub(startedAt).Milliseconds()

	return summary, nil
}
