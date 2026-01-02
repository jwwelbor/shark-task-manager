package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EpicRepoInterface defines methods needed from EpicRepository for file collision detection
type EpicRepoInterface interface {
	GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error)
	Update(ctx context.Context, epic *models.Epic) error
}

// FeatureRepoInterface defines methods needed from FeatureRepository for file collision detection
type FeatureRepoInterface interface {
	GetByFilePath(ctx context.Context, filePath string) (*models.Feature, error)
	Update(ctx context.Context, feature *models.Feature) error
}

// TaskRepoInterface defines methods needed from TaskRepository for file collision detection
type TaskRepoInterface interface {
	GetByFilePath(ctx context.Context, filePath string) (*models.Task, error)
	Update(ctx context.Context, task *models.Task) error
}

// FileCollision represents a file path conflict with an existing entity
type FileCollision struct {
	FilePath string
	Epic     *models.Epic    // Non-nil if epic claims this file
	Feature  *models.Feature // Non-nil if feature claims this file
	Task     *models.Task    // Non-nil if task claims this file
}

// DetectFileCollision checks if a file path is already claimed by an epic, feature, or task
// Returns nil if no collision exists, otherwise returns FileCollision with the claiming entity
func DetectFileCollision(
	ctx context.Context,
	filePath string,
	epicRepo EpicRepoInterface,
	featureRepo FeatureRepoInterface,
	taskRepo TaskRepoInterface,
) (*FileCollision, error) {
	// Check if epic claims this file
	epic, err := epicRepo.GetByFilePath(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to check epic file collision: %w", err)
	}
	if epic != nil {
		return &FileCollision{
			FilePath: filePath,
			Epic:     epic,
		}, nil
	}

	// Check if feature claims this file
	feature, err := featureRepo.GetByFilePath(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to check feature file collision: %w", err)
	}
	if feature != nil {
		return &FileCollision{
			FilePath: filePath,
			Feature:  feature,
		}, nil
	}

	// Check if task claims this file
	task, err := taskRepo.GetByFilePath(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to check task file collision: %w", err)
	}
	if task != nil {
		return &FileCollision{
			FilePath: filePath,
			Task:     task,
		}, nil
	}

	// No collision found
	return nil, nil
}

// HandleFileReassignment handles file reassignment logic with --force flag support
// If collision exists and force=false, returns descriptive error
// If collision exists and force=true, clears the file path from existing entity
func HandleFileReassignment(
	ctx context.Context,
	collision *FileCollision,
	force bool,
	epicRepo EpicRepoInterface,
	featureRepo FeatureRepoInterface,
	taskRepo TaskRepoInterface,
) error {
	// No collision, nothing to do
	if collision == nil {
		return nil
	}

	// Collision exists but force=false, return error
	if !force {
		var entityType, entityKey, entityTitle string

		if collision.Epic != nil {
			entityType = "epic"
			entityKey = collision.Epic.Key
			entityTitle = collision.Epic.Title
		} else if collision.Feature != nil {
			entityType = "feature"
			entityKey = collision.Feature.Key
			entityTitle = collision.Feature.Title
		} else if collision.Task != nil {
			entityType = "task"
			entityKey = collision.Task.Key
			entityTitle = collision.Task.Title
		}

		return fmt.Errorf(
			"file %q already claimed by %s %s (%s). Use --force to reassign",
			collision.FilePath,
			entityType,
			entityKey,
			entityTitle,
		)
	}

	// Collision exists and force=true, clear the file path
	if collision.Epic != nil {
		// Clear epic's file path
		collision.Epic.FilePath = nil
		if err := epicRepo.Update(ctx, collision.Epic); err != nil {
			return fmt.Errorf("failed to clear file path from epic %s: %w", collision.Epic.Key, err)
		}
	} else if collision.Feature != nil {
		// Clear feature's file path
		collision.Feature.FilePath = nil
		if err := featureRepo.Update(ctx, collision.Feature); err != nil {
			return fmt.Errorf("failed to clear file path from feature %s: %w", collision.Feature.Key, err)
		}
	} else if collision.Task != nil {
		// Clear task's file path
		collision.Task.FilePath = nil
		if err := taskRepo.Update(ctx, collision.Task); err != nil {
			return fmt.Errorf("failed to clear file path from task %s: %w", collision.Task.Key, err)
		}
	}

	return nil
}

// CreateBackupIfForce creates a timestamped database backup when force=true
// Returns empty string if force=false
// Returns backup path on success, error on failure
func CreateBackupIfForce(force bool, dbPath string, operation string) (string, error) {
	// No backup needed if not forcing
	if !force {
		return "", nil
	}

	// Generate timestamped backup filename
	timestamp := time.Now().Format("20060102-150405")
	backupFilename := fmt.Sprintf("%s.%s-%s.backup", dbPath, operation, timestamp)

	// Open source file
	srcFile, err := os.Open(dbPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: could not open database: %w", err)
	}
	defer srcFile.Close()

	// Create backup file
	dstFile, err := os.Create(backupFilename)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: could not create backup file: %w", err)
	}
	defer dstFile.Close()

	// Copy database to backup
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		// Cleanup partial backup on error
		os.Remove(backupFilename)
		return "", fmt.Errorf("failed to create backup: copy failed: %w", err)
	}

	// Sync to disk
	if err := dstFile.Sync(); err != nil {
		os.Remove(backupFilename)
		return "", fmt.Errorf("failed to create backup: sync failed: %w", err)
	}

	return backupFilename, nil
}

// GetAbsoluteFilePath converts a relative file path to an absolute path
// Uses current working directory as the project root
func GetAbsoluteFilePath(relativePath string) (string, error) {
	if filepath.IsAbs(relativePath) {
		return relativePath, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	return filepath.Join(wd, relativePath), nil
}
