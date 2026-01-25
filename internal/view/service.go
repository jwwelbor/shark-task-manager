package view

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/jwwelbor/shark-task-manager/internal/cli/scope"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EpicRepository interface defines methods needed from epic repository
type EpicRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
}

// FeatureRepository interface defines methods needed from feature repository
type FeatureRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
}

// TaskRepository interface defines methods needed from task repository
type TaskRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
}

// Service handles viewing specification files
// It follows the Single Responsibility Principle by focusing only on viewing operations
type Service struct {
	epicRepo    EpicRepository
	featureRepo FeatureRepository
	taskRepo    TaskRepository
}

// NewService creates a new ViewService with injected dependencies
func NewService(
	epicRepo EpicRepository,
	featureRepo FeatureRepository,
	taskRepo TaskRepository,
) *Service {
	return &Service{
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		taskRepo:    taskRepo,
	}
}

// GetFilePath retrieves the file path for a given scope
// Returns the file path string or an error if the entity is not found or has no file path
func (s *Service) GetFilePath(ctx context.Context, parsedScope *scope.Scope) (string, error) {
	switch parsedScope.Type {
	case scope.ScopeEpic:
		epic, err := s.epicRepo.GetByKey(ctx, parsedScope.Key)
		if err != nil {
			return "", fmt.Errorf("epic not found: %w", err)
		}
		if epic.FilePath == nil || *epic.FilePath == "" {
			return "", fmt.Errorf("epic %s has no file path set", parsedScope.Key)
		}
		return *epic.FilePath, nil

	case scope.ScopeFeature:
		feature, err := s.featureRepo.GetByKey(ctx, parsedScope.Key)
		if err != nil {
			return "", fmt.Errorf("feature not found: %w", err)
		}
		if feature.FilePath == nil || *feature.FilePath == "" {
			return "", fmt.Errorf("feature %s has no file path set", parsedScope.Key)
		}
		return *feature.FilePath, nil

	case scope.ScopeTask:
		task, err := s.taskRepo.GetByKey(ctx, parsedScope.Key)
		if err != nil {
			return "", fmt.Errorf("task not found: %w", err)
		}
		if task.FilePath == nil || *task.FilePath == "" {
			return "", fmt.Errorf("task %s has no file path set", parsedScope.Key)
		}
		return *task.FilePath, nil

	default:
		return "", fmt.Errorf("unknown scope type: %s", parsedScope.Type)
	}
}

// LaunchViewer opens the file in the specified viewer
// Returns an error if the file doesn't exist or the viewer command fails
func (s *Service) LaunchViewer(ctx context.Context, filePath string, viewerCmd string) error {
	// Validate file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}

	// Construct command
	cmd := exec.CommandContext(ctx, viewerCmd, filePath)

	// Connect to stdin, stdout, stderr for interactive viewers
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Execute viewer
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to launch viewer %q: %w", viewerCmd, err)
	}

	return nil
}
