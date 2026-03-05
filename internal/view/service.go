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

// ChangeCardRepository interface defines methods needed from change-card repository
type ChangeCardRepository interface {
	GetByKey(ctx context.Context, key string) (*models.ChangeCard, error)
}

// BugRepository interface defines methods needed from bug repository
type BugRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Bug, error)
}

// Service handles viewing specification files
// It follows the Single Responsibility Principle by focusing only on viewing operations
type Service struct {
	epicRepo       EpicRepository
	featureRepo    FeatureRepository
	taskRepo       TaskRepository
	changeCardRepo ChangeCardRepository
	bugRepo        BugRepository
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

// SetChangeCardRepo sets the change-card repository for resolving CC-### keys.
// This allows the view service to resolve change-card file paths when viewing CC-### keys.
func (s *Service) SetChangeCardRepo(repo ChangeCardRepository) {
	s.changeCardRepo = repo
}

// SetBugRepo sets the bug repository for resolving B### keys.
// This allows the view service to resolve bug file paths when viewing B### keys.
func (s *Service) SetBugRepo(repo BugRepository) {
	s.bugRepo = repo
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

	case scope.ScopeChangeCard:
		if s.changeCardRepo == nil {
			return "", fmt.Errorf("change-card repository not configured in view service")
		}
		card, err := s.changeCardRepo.GetByKey(ctx, parsedScope.Key)
		if err != nil {
			return "", fmt.Errorf("change-card not found: %w", err)
		}
		if card.FilePath == "" {
			return "", fmt.Errorf("change-card %s has no file path set", parsedScope.Key)
		}
		return card.FilePath, nil

	case scope.ScopeBug:
		if s.bugRepo == nil {
			return "", fmt.Errorf("bug repository not configured in view service")
		}
		bug, err := s.bugRepo.GetByKey(ctx, parsedScope.Key)
		if err != nil {
			return "", fmt.Errorf("bug not found: %w", err)
		}
		if bug.FilePath == nil || *bug.FilePath == "" {
			return "", fmt.Errorf("bug %s has no file path set", parsedScope.Key)
		}
		return *bug.FilePath, nil

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
