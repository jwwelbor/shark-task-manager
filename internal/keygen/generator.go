package keygen

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// TaskKeyGenerator generates task keys for PRP files without explicit task_key
type TaskKeyGenerator struct {
	taskRepo      *repository.TaskRepository
	featureRepo   *repository.FeatureRepository
	epicRepo      *repository.EpicRepository
	pathParser    *PathParser
	fmWriter      *FrontmatterWriter
	generatedKeys map[string]int // track keys generated in current batch
	mutex         sync.RWMutex   // protect concurrent access
}

// NewTaskKeyGenerator creates a new TaskKeyGenerator
func NewTaskKeyGenerator(
	taskRepo *repository.TaskRepository,
	featureRepo *repository.FeatureRepository,
	epicRepo *repository.EpicRepository,
	docsRoot string,
) *TaskKeyGenerator {
	return &TaskKeyGenerator{
		taskRepo:      taskRepo,
		featureRepo:   featureRepo,
		epicRepo:      epicRepo,
		pathParser:    NewPathParser(docsRoot),
		fmWriter:      NewFrontmatterWriter(),
		generatedKeys: make(map[string]int),
	}
}

// GenerateKeyForFile generates a task key for a PRP file that lacks one
// Returns the generated key and whether it was written to the file
type GenerateResult struct {
	TaskKey       string
	FeatureID     int64
	EpicKey       string
	FeatureKey    string
	WrittenToFile bool
	Error         error
}

// GenerateKeyForFile generates a task key for the given file
func (g *TaskKeyGenerator) GenerateKeyForFile(ctx context.Context, filePath string) (*GenerateResult, error) {
	result := &GenerateResult{}

	// Check if file already has a task_key
	hasKey, existingKey, err := g.fmWriter.HasTaskKey(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing task key: %w", err)
	}

	if hasKey {
		// File already has a key, extract epic/feature for completeness
		components, err := g.pathParser.ParsePath(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse path: %w", err)
		}

		// Get feature ID
		feature, err := g.featureRepo.GetByKey(ctx, components.FeatureKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get feature: %w", err)
		}

		// Extract sequence number from existing key and track it
		// This prevents duplicates when processing files that already have keys
		var sequence int
		if _, err := fmt.Sscanf(existingKey, "T-"+components.FeatureKey+"-%d", &sequence); err == nil {
			g.mutex.Lock()
			if current, ok := g.generatedKeys[components.FeatureKey]; !ok || sequence > current {
				g.generatedKeys[components.FeatureKey] = sequence
			}
			g.mutex.Unlock()
		}

		result.TaskKey = existingKey
		result.FeatureID = feature.ID
		result.EpicKey = components.EpicKey
		result.FeatureKey = components.FeatureKey
		result.WrittenToFile = false
		return result, nil
	}

	// Parse file path to extract epic and feature
	components, err := g.pathParser.ParsePath(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path: %w", err)
	}

	result.EpicKey = components.EpicKey
	result.FeatureKey = components.FeatureKey

	// Validate epic exists in database
	_, err = g.epicRepo.GetByKey(ctx, components.EpicKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("orphaned file: epic '%s' not found in database. Suggestion: Create epic '%s' first or move file to existing epic folder", components.EpicKey, components.EpicKey)
		}
		return nil, fmt.Errorf("failed to validate epic: %w", err)
	}

	// Validate feature exists in database
	feature, err := g.featureRepo.GetByKey(ctx, components.FeatureKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("orphaned file: feature '%s' not found in database. Suggestion: Create feature '%s' first or move file to existing feature folder", components.FeatureKey, components.FeatureKey)
		}
		return nil, fmt.Errorf("failed to validate feature: %w", err)
	}

	result.FeatureID = feature.ID

	// Get next sequence number for this feature
	maxSequence, err := g.taskRepo.GetMaxSequenceForFeature(ctx, components.FeatureKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get max sequence: %w", err)
	}

	// Check in-memory generated keys to prevent duplicates in batch processing
	g.mutex.RLock()
	if generated, ok := g.generatedKeys[components.FeatureKey]; ok && generated > maxSequence {
		maxSequence = generated
	}
	g.mutex.RUnlock()

	nextSequence := maxSequence + 1

	// Track the generated key before writing to file
	g.mutex.Lock()
	g.generatedKeys[components.FeatureKey] = nextSequence
	g.mutex.Unlock()

	// Generate task key in format: T-E##-F##-###
	taskKey := fmt.Sprintf("T-%s-%03d", components.FeatureKey, nextSequence)
	result.TaskKey = taskKey

	// Write task key to file frontmatter
	if err := g.fmWriter.WriteTaskKey(filePath, taskKey); err != nil {
		// Log warning but don't fail - return the generated key for in-memory use
		result.WrittenToFile = false
		result.Error = fmt.Errorf("warning: failed to write task key to file: %w", err)
		return result, nil
	}

	result.WrittenToFile = true
	return result, nil
}

// GenerateKeysForFiles generates task keys for multiple files in batch
// This is more efficient when processing multiple files from the same feature
func (g *TaskKeyGenerator) GenerateKeysForFiles(ctx context.Context, filePaths []string) ([]*GenerateResult, error) {
	results := make([]*GenerateResult, len(filePaths))

	for i, filePath := range filePaths {
		result, err := g.GenerateKeyForFile(ctx, filePath)
		if err != nil {
			// Store error in result but continue processing other files
			results[i] = &GenerateResult{
				Error: err,
			}
			continue
		}
		results[i] = result
	}

	return results, nil
}

// ValidateFile checks if a file is ready for key generation
// Returns error if file is not valid (missing epic/feature, not writable, etc.)
func (g *TaskKeyGenerator) ValidateFile(ctx context.Context, filePath string) error {
	// Check if file is writable
	if err := g.fmWriter.ValidateFileWritable(filePath); err != nil {
		return fmt.Errorf("file validation failed: %w", err)
	}

	// Parse path
	components, err := g.pathParser.ParsePath(filePath)
	if err != nil {
		return fmt.Errorf("path validation failed: %w", err)
	}

	// Validate epic exists
	_, err = g.epicRepo.GetByKey(ctx, components.EpicKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("epic '%s' not found in database", components.EpicKey)
		}
		return fmt.Errorf("failed to validate epic: %w", err)
	}

	// Validate feature exists
	_, err = g.featureRepo.GetByKey(ctx, components.FeatureKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("feature '%s' not found in database", components.FeatureKey)
		}
		return fmt.Errorf("failed to validate feature: %w", err)
	}

	return nil
}

// GetFileFeature returns the feature for a given file path
// Useful for grouping files by feature before batch processing
func (g *TaskKeyGenerator) GetFileFeature(ctx context.Context, filePath string) (string, error) {
	components, err := g.pathParser.ParsePath(filePath)
	if err != nil {
		return "", err
	}
	return components.FeatureKey, nil
}
