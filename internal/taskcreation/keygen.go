package taskcreation

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// KeyGenerator handles automatic task key generation
type KeyGenerator struct {
	taskRepo    *repository.TaskRepository
	featureRepo *repository.FeatureRepository
}

// NewKeyGenerator creates a new KeyGenerator
func NewKeyGenerator(taskRepo *repository.TaskRepository, featureRepo *repository.FeatureRepository) *KeyGenerator {
	return &KeyGenerator{
		taskRepo:    taskRepo,
		featureRepo: featureRepo,
	}
}

// GenerateTaskKey generates the next available task key for a feature
// Format: T-<epic-key>-<feature-key>-<zero-padded-number>
// Example: T-E01-F02-003
func (kg *KeyGenerator) GenerateTaskKey(ctx context.Context, epicKey, featureKey string) (string, error) {
	// Normalize feature key (prepend epic if needed)
	normalizedFeatureKey := normalizeFeatureKey(epicKey, featureKey)

	// Get feature to verify it exists
	feature, err := kg.featureRepo.GetByKey(ctx, normalizedFeatureKey)
	if err != nil {
		return "", fmt.Errorf("feature %s does not exist", normalizedFeatureKey)
	}

	// Get all tasks for this feature to find the highest number
	tasks, err := kg.taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		return "", fmt.Errorf("failed to list tasks for feature: %w", err)
	}

	// Find the highest task number
	maxNumber := 0
	keyPattern := regexp.MustCompile(`^T-E\d{2}-F\d{2}-(\d{3})$`)

	for _, task := range tasks {
		matches := keyPattern.FindStringSubmatch(task.Key)
		if len(matches) == 2 {
			num, err := strconv.Atoi(matches[1])
			if err == nil && num > maxNumber {
				maxNumber = num
			}
		}
	}

	// Calculate next number
	nextNumber := maxNumber + 1

	// Check if we've exceeded the maximum
	if nextNumber > 999 {
		return "", fmt.Errorf("feature %s has reached maximum task count (999)", normalizedFeatureKey)
	}

	// Extract just the feature part (F01, F02, etc.) from the normalized key
	featurePart := extractFeaturePart(normalizedFeatureKey)

	// Generate the key with zero-padded number
	key := fmt.Sprintf("T-%s-%s-%03d", epicKey, featurePart, nextNumber)

	return key, nil
}

// GenerateTaskKeyWithTx generates a task key within a transaction for concurrent safety
func (kg *KeyGenerator) GenerateTaskKeyWithTx(ctx context.Context, tx *sql.Tx, epicKey, featureKey string) (string, error) {
	// Normalize feature key
	normalizedFeatureKey := normalizeFeatureKey(epicKey, featureKey)

	// Get feature to verify it exists
	feature, err := kg.featureRepo.GetByKey(ctx, normalizedFeatureKey)
	if err != nil {
		return "", fmt.Errorf("feature %s does not exist", normalizedFeatureKey)
	}

	// Query max key within transaction with row lock
	query := `
		SELECT key FROM tasks
		WHERE feature_id = ?
		ORDER BY key DESC
		LIMIT 1
	`

	var maxKey sql.NullString
	err = tx.QueryRowContext(ctx, query, feature.ID).Scan(&maxKey)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to query max task key: %w", err)
	}

	// Extract number from max key
	maxNumber := 0
	if maxKey.Valid {
		num := extractNumberFromKey(maxKey.String)
		if num > 0 {
			maxNumber = num
		}
	}

	// Calculate next number
	nextNumber := maxNumber + 1

	// Check limit
	if nextNumber > 999 {
		return "", fmt.Errorf("feature %s has reached maximum task count (999)", normalizedFeatureKey)
	}

	// Extract feature part
	featurePart := extractFeaturePart(normalizedFeatureKey)

	// Generate key
	key := fmt.Sprintf("T-%s-%s-%03d", epicKey, featurePart, nextNumber)

	return key, nil
}

// normalizeFeatureKey prepends epic key to feature key if needed
// Examples: F02 with E01 -> E01-F02, E01-F02 with E01 -> E01-F02
func normalizeFeatureKey(epicKey, featureKey string) string {
	// If feature key already includes epic prefix, return as-is
	if strings.HasPrefix(featureKey, epicKey+"-") {
		return featureKey
	}

	// If feature key is just "F##", prepend epic key
	if strings.HasPrefix(featureKey, "F") {
		return fmt.Sprintf("%s-%s", epicKey, featureKey)
	}

	// Otherwise return as-is (will fail validation later)
	return featureKey
}

// extractFeaturePart extracts the feature part from a full feature key
// Example: E01-F02 -> F02
func extractFeaturePart(fullFeatureKey string) string {
	parts := strings.Split(fullFeatureKey, "-")
	if len(parts) == 2 {
		return parts[1]
	}
	return fullFeatureKey
}

// extractNumberFromKey extracts the numeric part from a task key
// Example: T-E01-F02-042 -> 42
func extractNumberFromKey(key string) int {
	pattern := regexp.MustCompile(`^T-E\d{2}-F\d{2}-(\d{3})$`)
	matches := pattern.FindStringSubmatch(key)
	if len(matches) == 2 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			return num
		}
	}
	return 0
}
