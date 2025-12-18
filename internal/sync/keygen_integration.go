package sync

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/keygen"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// KeyGenerator wraps the keygen package for use in sync engine
type KeyGenerator struct {
	generator *keygen.TaskKeyGenerator
}

// NewKeyGenerator creates a new KeyGenerator for sync integration
func NewKeyGenerator(
	taskRepo *repository.TaskRepository,
	featureRepo *repository.FeatureRepository,
	epicRepo *repository.EpicRepository,
	docsRoot string,
) *KeyGenerator {
	return &KeyGenerator{
		generator: keygen.NewTaskKeyGenerator(taskRepo, featureRepo, epicRepo, docsRoot),
	}
}

// GenerateTaskKey generates a task key for the given epic and feature
// This is a compatibility wrapper for the existing sync engine code
func (kg *KeyGenerator) GenerateTaskKey(ctx context.Context, epicKey, featureKey string) (string, error) {
	// This method is called when epic/feature are known but we need to generate next sequence
	// We can't use the file-based generation here, so we need to directly query max sequence

	// Build a fake file path that matches the pattern
	// This is a workaround for the current architecture
	// In the future, we should refactor to separate path parsing from key generation
	fakePath := fmt.Sprintf("/dummy/docs/plan/%s-dummy/%s-dummy/tasks/dummy.prp.md", epicKey, featureKey)

	result, err := kg.generator.GenerateKeyForFile(ctx, fakePath)
	if err != nil {
		return "", err
	}

	return result.TaskKey, nil
}

// GenerateKeyForFile generates a task key for a specific file
// Returns the generated key and writes it to the file's frontmatter
func (kg *KeyGenerator) GenerateKeyForFile(ctx context.Context, filePath string) (string, error) {
	result, err := kg.generator.GenerateKeyForFile(ctx, filePath)
	if err != nil {
		return "", err
	}

	// Log warning if key wasn't written to file
	if result.Error != nil {
		// Warning logged by caller
	}

	return result.TaskKey, nil
}

// ValidateFile checks if a file is ready for key generation
func (kg *KeyGenerator) ValidateFile(ctx context.Context, filePath string) error {
	return kg.generator.ValidateFile(ctx, filePath)
}
