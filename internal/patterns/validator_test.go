package patterns

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePattern_ValidPattern(t *testing.T) {
	// Test that a valid pattern passes validation
	pattern := `^E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`
	entityType := "epic"

	err := ValidatePattern(pattern, entityType)
	assert.NoError(t, err, "Valid pattern should pass validation")
}

func TestValidatePattern_InvalidRegexSyntax(t *testing.T) {
	// Test that invalid regex syntax produces error
	pattern := `^E(?P<unfinished` // Unclosed named group
	entityType := "epic"

	err := ValidatePattern(pattern, entityType)
	require.Error(t, err, "Invalid regex syntax should produce error")
	assert.Contains(t, err.Error(), "invalid regex syntax", "Error should mention regex syntax")
}

func TestValidatePattern_MissingRequiredCaptureGroups_Epic(t *testing.T) {
	// Epic patterns must include at least one of: epic_id, epic_slug, number
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{
			name:    "has number - valid",
			pattern: `^E(?P<number>\d{2})-.*$`,
			wantErr: false,
		},
		{
			name:    "has epic_id - valid",
			pattern: `^(?P<epic_id>tech-debt|bugs)$`,
			wantErr: false,
		},
		{
			name:    "has epic_slug - valid",
			pattern: `^E\d{2}-(?P<epic_slug>[a-z0-9-]+)$`,
			wantErr: false,
		},
		{
			name:    "has slug (same as epic_slug) - valid",
			pattern: `^E\d{2}-(?P<slug>[a-z0-9-]+)$`,
			wantErr: false,
		},
		{
			name:    "missing all required groups - invalid",
			pattern: `^E\d{2}-[a-z]+$`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePattern(tt.pattern, "epic")
			if tt.wantErr {
				assert.Error(t, err, "Pattern should fail validation")
				assert.Contains(t, err.Error(), "required capture group", "Error should mention required capture group")
			} else {
				assert.NoError(t, err, "Pattern should pass validation")
			}
		})
	}
}

func TestValidatePattern_MissingRequiredCaptureGroups_Feature(t *testing.T) {
	// Feature patterns must include at least one of (feature_id OR feature_slug OR number OR slug)
	// Epic context may be explicit (epic_id OR epic_num) in the pattern OR inferred from parent folder
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{
			name:    "has epic_num and number - valid",
			pattern: `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-.*$`,
			wantErr: false,
		},
		{
			name:    "has epic_id and feature_id - valid",
			pattern: `^(?P<epic_id>tech-debt)/(?P<feature_id>auth)$`,
			wantErr: false,
		},
		{
			name:    "has epic_num and feature_slug - valid",
			pattern: `^E(?P<epic_num>\d{2})-F\d{2}-(?P<feature_slug>[a-z-]+)$`,
			wantErr: false,
		},
		{
			name:    "has epic_num and slug (same as feature_slug) - valid",
			pattern: `^E(?P<epic_num>\d{2})-F\d{2}-(?P<slug>[a-z-]+)$`,
			wantErr: false,
		},
		{
			name:    "has number without epic (nested context) - valid",
			pattern: `^F(?P<number>\d{2})-.*$`,
			wantErr: false,
		},
		{
			name:    "missing feature identifier completely - invalid",
			pattern: `^E(?P<epic_num>\d{2})-F\d{2}-.*$`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePattern(tt.pattern, "feature")
			if tt.wantErr {
				assert.Error(t, err, "Pattern should fail validation")
				assert.Contains(t, err.Error(), "required capture group", "Error should mention required capture group")
			} else {
				assert.NoError(t, err, "Pattern should pass validation")
			}
		})
	}
}

func TestValidatePattern_MissingRequiredCaptureGroups_Task(t *testing.T) {
	// Task patterns must include (epic_id OR epic_num) AND (feature_id OR feature_num) AND (task_id OR number OR task_slug)
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{
			name:    "has all required (epic_num, feature_num, number) - valid",
			pattern: `^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3})\.md$`,
			wantErr: false,
		},
		{
			name:    "has epic_id, feature_id, task_id - valid",
			pattern: `^(?P<epic_id>tech-debt)/(?P<feature_id>auth)/(?P<task_id>T001)\.md$`,
			wantErr: false,
		},
		{
			name:    "has epic_num, feature_num, slug (same as task_slug) - valid",
			pattern: `^E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})/(?P<slug>.+)\.md$`,
			wantErr: false,
		},
		{
			name:    "missing epic identifier - invalid",
			pattern: `^F(?P<feature_num>\d{2})-(?P<number>\d{3})\.md$`,
			wantErr: true,
		},
		{
			name:    "missing feature identifier - invalid",
			pattern: `^E(?P<epic_num>\d{2})-(?P<number>\d{3})\.md$`,
			wantErr: true,
		},
		{
			name:    "missing task identifier - invalid",
			pattern: `^E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-.*\.md$`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePattern(tt.pattern, "task")
			if tt.wantErr {
				assert.Error(t, err, "Pattern should fail validation")
				assert.Contains(t, err.Error(), "required capture group", "Error should mention required capture group")
			} else {
				assert.NoError(t, err, "Pattern should pass validation")
			}
		})
	}
}

func TestValidatePattern_UnrecognizedCaptureGroups(t *testing.T) {
	// Unrecognized capture group names should produce warnings
	pattern := `^E(?P<number>\d{2})-(?P<feature_number>\d{2})$`
	entityType := "epic"

	warnings := GetPatternWarnings(pattern, entityType)
	require.NotEmpty(t, warnings, "Unrecognized capture group should produce warning")
	assert.Contains(t, warnings[0], "feature_number", "Warning should mention the unrecognized group name")
	assert.Contains(t, warnings[0], "Did you mean", "Warning should suggest alternatives")
}

func TestValidatePattern_CatastrophicBacktracking(t *testing.T) {
	// Test that patterns with catastrophic backtracking potential are rejected
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{
			name:    "nested quantifiers - catastrophic",
			pattern: `^(a+)+b$`,
			wantErr: true,
		},
		{
			name:    "multiple nested quantifiers - catastrophic",
			pattern: `^(a*)*b$`,
			wantErr: true,
		},
		{
			name:    "complex nested - catastrophic",
			pattern: `^(a+b*)+c$`,
			wantErr: true,
		},
		{
			name:    "safe pattern - non-catastrophic",
			pattern: `^a+b$`,
			wantErr: false,
		},
		{
			name:    "safe pattern with groups - non-catastrophic",
			pattern: `^(ab)+c$`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For patterns with at least one required group to pass other validations
			testPattern := tt.pattern
			if !hasRequiredCaptureGroup(tt.pattern, "epic") {
				// Add a dummy capture group to test only backtracking detection
				testPattern = `(?P<number>\d+)` + tt.pattern
			}

			err := ValidatePattern(testPattern, "epic")
			if tt.wantErr {
				assert.Error(t, err, "Catastrophic backtracking pattern should be rejected")
				assert.Contains(t, err.Error(), "backtracking", "Error should mention backtracking")
			} else {
				// May have other errors, but should not be backtracking error
				if err != nil {
					assert.NotContains(t, err.Error(), "backtracking")
				}
			}
		})
	}
}

func TestValidatePatternConfig_AllPatterns(t *testing.T) {
	// Test validation of complete pattern configuration
	config := &PatternConfig{
		Epic: EntityPatterns{
			Folder: []string{
				`^E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
			},
		},
		Feature: EntityPatterns{
			Folder: []string{
				`^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
			},
		},
		Task: EntityPatterns{
			File: []string{
				`^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3})\.md$`,
			},
		},
	}

	err := ValidatePatternConfig(config)
	assert.NoError(t, err, "Valid pattern config should pass validation")
}

func TestValidatePatternConfig_WithErrors(t *testing.T) {
	// Test that config with invalid patterns produces errors
	config := &PatternConfig{
		Epic: EntityPatterns{
			Folder: []string{
				`^E(?P<unfinished`, // Invalid regex syntax
			},
		},
		Feature: EntityPatterns{
			Folder: []string{
				`^[a-z]+$`, // Missing feature identifier completely
			},
		},
		Task: EntityPatterns{
			File: []string{
				`^T-(?P<unfinished`, // Invalid regex syntax in task file pattern
			},
		},
	}

	err := ValidatePatternConfig(config)
	require.Error(t, err, "Invalid pattern config should produce error")

	// Should contain multiple error messages
	errMsg := err.Error()
	assert.Contains(t, errMsg, "epic", "Error should mention epic patterns")
	assert.Contains(t, errMsg, "feature", "Error should mention feature patterns")
	assert.Contains(t, errMsg, "task", "Error should mention task patterns")
}

func TestValidationPerformance(t *testing.T) {
	// Test that validation completes in <100ms for typical config (15-20 patterns)
	config := GetDefaultPatterns()

	// Add more patterns to simulate realistic config (with proper capture groups)
	for i := 0; i < 10; i++ {
		config.Epic.Folder = append(config.Epic.Folder, `^E(?P<number>\d{2})-test-(?P<slug>[a-z-]+)$`)
		config.Feature.Folder = append(config.Feature.Folder, `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-test-(?P<slug>[a-z-]+)$`)
		config.Task.File = append(config.Task.File, `^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3})-test\.md$`)
	}

	// Time the validation
	start := time.Now()
	err := ValidatePatternConfig(config)
	elapsed := time.Since(start)

	assert.NoError(t, err, "Validation should succeed")
	assert.Less(t, elapsed.Milliseconds(), int64(100), "Validation should complete in <100ms")
}

// Helper function to check if pattern has required capture group (for tests)
func hasRequiredCaptureGroup(pattern, entityType string) bool {
	groups := extractCaptureGroupNames(pattern)
	switch entityType {
	case "epic":
		for _, g := range groups {
			if g == "epic_id" || g == "epic_slug" || g == "number" || g == "slug" {
				return true
			}
		}
	case "feature":
		hasEpic := false
		hasFeature := false
		for _, g := range groups {
			if g == "epic_id" || g == "epic_num" {
				hasEpic = true
			}
			if g == "feature_id" || g == "feature_slug" || g == "number" || g == "slug" {
				hasFeature = true
			}
		}
		return hasEpic && hasFeature
	case "task":
		hasEpic := false
		hasFeature := false
		hasTask := false
		for _, g := range groups {
			if g == "epic_id" || g == "epic_num" {
				hasEpic = true
			}
			if g == "feature_id" || g == "feature_num" {
				hasFeature = true
			}
			if g == "task_id" || g == "number" || g == "task_slug" || g == "slug" {
				hasTask = true
			}
		}
		return hasEpic && hasFeature && hasTask
	}
	return false
}
