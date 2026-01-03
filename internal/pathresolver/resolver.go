package pathresolver

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EpicRepository defines the interface for epic data access needed by PathResolver
type EpicRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	GetByID(ctx context.Context, id int64) (*models.Epic, error)
}

// FeatureRepository defines the interface for feature data access needed by PathResolver
type FeatureRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	GetByID(ctx context.Context, id int64) (*models.Feature, error)
}

// TaskRepository defines the interface for task data access needed by PathResolver
type TaskRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	GetByID(ctx context.Context, id int64) (*models.Task, error)
}

// PathResolver resolves file paths for epics, features, and tasks by querying the database
// for entity metadata (slug, file_path). This replaces PathBuilder's file-system-first
// approach with a database-first design for improved performance.
type PathResolver struct {
	epicRepo    EpicRepository
	featureRepo FeatureRepository
	taskRepo    TaskRepository
	projectRoot string
}

// NewPathResolver creates a new PathResolver with repository dependencies
func NewPathResolver(
	epicRepo EpicRepository,
	featureRepo FeatureRepository,
	taskRepo TaskRepository,
	projectRoot string,
) *PathResolver {
	return &PathResolver{
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		taskRepo:    taskRepo,
		projectRoot: projectRoot,
	}
}

// ResolveEpicPath resolves the file path for an epic by querying the database.
// Path precedence: explicit file_path > default (docs/plan/{epic-key}/)
func (pr *PathResolver) ResolveEpicPath(ctx context.Context, epicKey string) (string, error) {
	epic, err := pr.epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		return "", fmt.Errorf("failed to get epic %s: %w", epicKey, err)
	}

	// Precedence 1: Explicit file_path
	if epic.FilePath != nil && *epic.FilePath != "" {
		return filepath.Join(pr.projectRoot, *epic.FilePath), nil
	}

	// Precedence 2: Default path (docs/plan/{epic-key}/epic.md)
	slug := ""
	if epic.Slug != nil && *epic.Slug != "" {
		slug = *epic.Slug
	} else {
		slug = epic.Key
	}
	defaultPath := filepath.Join("docs", "plan", epic.Key+"-"+slug, "epic.md")
	return filepath.Join(pr.projectRoot, defaultPath), nil
}

// ResolveFeaturePath resolves the file path for a feature by querying the database.
// Path precedence: explicit file_path > default (docs/plan/{epic-key}/{feature-key}/)
func (pr *PathResolver) ResolveFeaturePath(ctx context.Context, featureKey string) (string, error) {
	feature, err := pr.featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		return "", fmt.Errorf("failed to get feature %s: %w", featureKey, err)
	}

	// Precedence 1: Explicit file_path
	if feature.FilePath != nil && *feature.FilePath != "" {
		return filepath.Join(pr.projectRoot, *feature.FilePath), nil
	}

	// Precedence 2: Default path (docs/plan/{epic-key}/{feature-key}/prd.md)
	// Get parent epic for default path construction
	epic, err := pr.epicRepo.GetByID(ctx, feature.EpicID)
	if err != nil {
		return "", fmt.Errorf("failed to get parent epic for feature %s: %w", featureKey, err)
	}

	epicSlug := ""
	if epic.Slug != nil && *epic.Slug != "" {
		epicSlug = *epic.Slug
	} else {
		epicSlug = epic.Key
	}

	featureSlug := ""
	if feature.Slug != nil && *feature.Slug != "" {
		featureSlug = *feature.Slug
	} else {
		featureSlug = feature.Key
	}

	epicFolder := epic.Key + "-" + epicSlug
	featureFolder := feature.Key + "-" + featureSlug
	defaultPath := filepath.Join("docs", "plan", epicFolder, featureFolder, "prd.md")
	return filepath.Join(pr.projectRoot, defaultPath), nil
}

// ResolveTaskPath resolves the file path for a task by querying the database.
// Path precedence: explicit file_path > default (feature's directory + tasks/{task-key}.md)
func (pr *PathResolver) ResolveTaskPath(ctx context.Context, taskKey string) (string, error) {
	task, err := pr.taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		return "", fmt.Errorf("failed to get task %s: %w", taskKey, err)
	}

	// Precedence 1: Explicit file_path
	if task.FilePath != nil && *task.FilePath != "" {
		return filepath.Join(pr.projectRoot, *task.FilePath), nil
	}

	// Precedence 2: Default path based on feature's location
	// Get parent feature and epic for path construction
	feature, err := pr.featureRepo.GetByID(ctx, task.FeatureID)
	if err != nil {
		return "", fmt.Errorf("failed to get parent feature for task %s: %w", taskKey, err)
	}

	epic, err := pr.epicRepo.GetByID(ctx, feature.EpicID)
	if err != nil {
		return "", fmt.Errorf("failed to get parent epic for task %s: %w", taskKey, err)
	}

	// Determine feature's base directory
	var featureBaseDir string

	if feature.FilePath != nil && *feature.FilePath != "" {
		// Feature has explicit path - use its directory as base
		featureBaseDir = filepath.Dir(*feature.FilePath)
	} else {
		// Default: docs/plan/{epic-key}/{feature-key}
		epicSlug := ""
		if epic.Slug != nil && *epic.Slug != "" {
			epicSlug = *epic.Slug
		} else {
			epicSlug = epic.Key
		}

		featureSlug := ""
		if feature.Slug != nil && *feature.Slug != "" {
			featureSlug = *feature.Slug
		} else {
			featureSlug = feature.Key
		}

		epicFolder := epic.Key + "-" + epicSlug
		featureFolder := feature.Key + "-" + featureSlug
		featureBaseDir = filepath.Join("docs", "plan", epicFolder, featureFolder)
	}

	// Task filename: {task-key}.md
	taskFilename := task.Key + ".md"
	taskPath := filepath.Join(featureBaseDir, "tasks", taskFilename)

	return filepath.Join(pr.projectRoot, taskPath), nil
}
