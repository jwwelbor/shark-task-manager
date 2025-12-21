package validation

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Repository defines the interface for database operations needed by the validator
type Repository interface {
	GetAllEpics(ctx context.Context) ([]*models.Epic, error)
	GetAllFeatures(ctx context.Context) ([]*models.Feature, error)
	GetAllTasks(ctx context.Context) ([]*models.Task, error)
	GetEpicByID(ctx context.Context, id int64) (*models.Epic, error)
	GetFeatureByID(ctx context.Context, id int64) (*models.Feature, error)
}

// Validator performs database integrity validation
type Validator struct {
	repo Repository
}

// NewValidator creates a new Validator instance
func NewValidator(repo Repository) *Validator {
	return &Validator{
		repo: repo,
	}
}

// Validate performs all validation checks and returns a comprehensive result
func (v *Validator) Validate(ctx context.Context) (*ValidationResult, error) {
	startTime := time.Now()

	result := &ValidationResult{
		BrokenFilePaths: []ValidationFailure{},
		OrphanedRecords: []ValidationFailure{},
		Summary: ValidationSummary{
			TotalChecked:    0,
			TotalIssues:     0,
			BrokenFilePaths: 0,
			OrphanedRecords: 0,
		},
	}

	// Get all entities
	epics, err := v.repo.GetAllEpics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get epics: %w", err)
	}

	features, err := v.repo.GetAllFeatures(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get features: %w", err)
	}

	tasks, err := v.repo.GetAllTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	result.Summary.TotalChecked = len(epics) + len(features) + len(tasks)

	// Validate file paths (only tasks have file_path currently)
	v.validateTaskFilePaths(tasks, result)

	// Validate relationships
	v.validateFeatureRelationships(ctx, features, result)
	v.validateTaskRelationships(ctx, tasks, result)

	// Calculate summary
	result.Summary.BrokenFilePaths = len(result.BrokenFilePaths)
	result.Summary.OrphanedRecords = len(result.OrphanedRecords)
	result.Summary.TotalIssues = result.Summary.BrokenFilePaths + result.Summary.OrphanedRecords

	// Calculate duration
	result.DurationMs = time.Since(startTime).Milliseconds()

	return result, nil
}

// validateTaskFilePaths checks if all task file paths exist on the filesystem
func (v *Validator) validateTaskFilePaths(tasks []*models.Task, result *ValidationResult) {
	for _, task := range tasks {
		// Skip tasks without file paths or empty file paths
		if task.FilePath == nil || *task.FilePath == "" {
			continue
		}

		filePath := *task.FilePath

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			result.BrokenFilePaths = append(result.BrokenFilePaths, ValidationFailure{
				EntityType:   "task",
				EntityKey:    task.Key,
				FilePath:     filePath,
				Issue:        "File does not exist (may have been moved or deleted)",
				SuggestedFix: "Re-scan to update file paths: 'shark sync --incremental' or update path manually in database",
			})
		}
	}
}

// validateFeatureRelationships checks if all features have valid parent epics
func (v *Validator) validateFeatureRelationships(ctx context.Context, features []*models.Feature, result *ValidationResult) {
	for _, feature := range features {
		// Check if parent epic exists
		_, err := v.repo.GetEpicByID(ctx, feature.EpicID)
		if err == sql.ErrNoRows {
			result.OrphanedRecords = append(result.OrphanedRecords, ValidationFailure{
				EntityType:        "feature",
				EntityKey:         feature.Key,
				MissingParentType: "epic",
				MissingParentID:   feature.EpicID,
				Issue:             fmt.Sprintf("Orphaned feature: parent epic with ID %d does not exist", feature.EpicID),
				SuggestedFix:      fmt.Sprintf("Create missing parent epic or delete orphaned feature: 'shark feature delete %s'", feature.Key),
			})
		} else if err != nil {
			// Log error but continue validation
			continue
		}
	}
}

// validateTaskRelationships checks if all tasks have valid parent features
func (v *Validator) validateTaskRelationships(ctx context.Context, tasks []*models.Task, result *ValidationResult) {
	for _, task := range tasks {
		// Check if parent feature exists
		_, err := v.repo.GetFeatureByID(ctx, task.FeatureID)
		if err == sql.ErrNoRows {
			result.OrphanedRecords = append(result.OrphanedRecords, ValidationFailure{
				EntityType:        "task",
				EntityKey:         task.Key,
				MissingParentType: "feature",
				MissingParentID:   task.FeatureID,
				Issue:             fmt.Sprintf("Orphaned task: parent feature with ID %d does not exist", task.FeatureID),
				SuggestedFix:      fmt.Sprintf("Create missing parent feature or delete orphaned task: 'shark task delete %s'", task.Key),
			})
		} else if err != nil {
			// Log error but continue validation
			continue
		}
	}
}
