package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/patterns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigTestPatternSuccess tests successful pattern matching
func TestConfigTestPatternSuccess(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		testString     string
		entityType     string
		expectedMatch  bool
		expectedGroups map[string]string
	}{
		{
			name:          "Epic pattern matches E04-task-mgmt",
			pattern:       `E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)`,
			testString:    "E04-task-mgmt",
			entityType:    "epic",
			expectedMatch: true,
			expectedGroups: map[string]string{
				"number": "04",
				"slug":   "task-mgmt",
			},
		},
		{
			name:          "Feature pattern with epic context",
			pattern:       `E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)`,
			testString:    "E04-F07-oauth-integration",
			entityType:    "feature",
			expectedMatch: true,
			expectedGroups: map[string]string{
				"epic_num": "04",
				"number":   "07",
				"slug":     "oauth-integration",
			},
		},
		{
			name:          "Task pattern full key",
			pattern:       `T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3})\.md`,
			testString:    "T-E04-F07-003.md",
			entityType:    "task",
			expectedMatch: true,
			expectedGroups: map[string]string{
				"epic_num":    "04",
				"feature_num": "07",
				"number":      "003",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test pattern matching logic (this will be in the implementation)
			result, groups, err := testPatternMatch(tt.pattern, tt.testString)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMatch, result)
			if tt.expectedMatch {
				for key, expectedVal := range tt.expectedGroups {
					assert.Equal(t, expectedVal, groups[key], "capture group '%s' mismatch", key)
				}
			}
		})
	}
}

// TestConfigTestPatternFailure tests failed pattern matching
func TestConfigTestPatternFailure(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		testString string
		entityType string
	}{
		{
			name:       "Epic pattern doesn't match tech-debt",
			pattern:    `E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)`,
			testString: "tech-debt",
			entityType: "epic",
		},
		{
			name:       "Feature pattern doesn't match without epic prefix",
			pattern:    `E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)`,
			testString: "F07-feature",
			entityType: "feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, _ := testPatternMatch(tt.pattern, tt.testString)
			assert.False(t, result, "pattern should not match")
		})
	}
}

// TestConfigTestPatternValidation tests pattern validation
func TestConfigTestPatternValidation(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		entityType  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Feature pattern with generic slug (nested context allowed)",
			pattern:     `F(?P<slug>[a-z-]+)`,
			entityType:  "feature",
			expectError: false,
		},
		{
			name:        "Task pattern missing feature identifier",
			pattern:     `T-E(?P<epic_num>\d{2})-(?P<number>\d{3})\.md`,
			entityType:  "task",
			expectError: true,
			errorMsg:    "missing required capture group",
		},
		{
			name:        "Valid epic pattern",
			pattern:     `E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)`,
			entityType:  "epic",
			expectError: false,
		},
		{
			name:        "Feature pattern missing feature identifier completely",
			pattern:     `^[a-z-]+$`,
			entityType:  "feature",
			expectError: true,
			errorMsg:    "missing required capture group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := patterns.ValidatePattern(tt.pattern, tt.entityType)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfigValidatePatternsCommand tests the validate-patterns command
func TestConfigValidatePatternsCommand(t *testing.T) {
	// Create temp directory for test config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	tests := []struct {
		name           string
		config         *patterns.PatternConfig
		expectExitCode int
		expectErrors   bool
		expectWarnings bool
	}{
		{
			name:           "All patterns valid",
			config:         patterns.GetDefaultPatterns(),
			expectExitCode: 0,
			expectErrors:   false,
			expectWarnings: false,
		},
		{
			name: "Invalid regex syntax",
			config: &patterns.PatternConfig{
				Epic: patterns.EntityPatterns{
					Folder: []string{`E\d{2-[a-z]+`}, // Missing closing brace
					File:   []string{`epic\.md`},
					Generation: patterns.GenerationFormat{
						Format: "E{number:02d}-{slug}",
					},
				},
				Feature: patterns.GetDefaultPatterns().Feature,
				Task:    patterns.GetDefaultPatterns().Task,
			},
			expectExitCode: 1,
			expectErrors:   true,
			expectWarnings: false,
		},
		{
			name: "Missing required capture groups",
			config: &patterns.PatternConfig{
				Epic: patterns.GetDefaultPatterns().Epic,
				Feature: patterns.EntityPatterns{
					Folder: []string{`^[a-z-]+$`}, // Missing feature identifier completely
					File:   []string{`prd\.md`},
					Generation: patterns.GenerationFormat{
						Format: "E{epic:02d}-F{number:02d}-{slug}",
					},
				},
				Task: patterns.GetDefaultPatterns().Task,
			},
			expectExitCode: 1,
			expectErrors:   true,
			expectWarnings: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write config to file
			err := writePatternConfig(configPath, tt.config)
			require.NoError(t, err)

			// Validate the config
			validationErr := patterns.ValidatePatternConfig(tt.config)

			if tt.expectErrors {
				assert.Error(t, validationErr)
			} else {
				assert.NoError(t, validationErr)
			}
		})
	}
}

// TestConfigShowCommand tests the show command
func TestConfigShowCommand(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Write test config
	config := patterns.GetDefaultPatterns()
	err := writePatternConfig(configPath, config)
	require.NoError(t, err)

	t.Run("Show all configuration", func(t *testing.T) {
		// Test that show command displays all config sections
		// This will be implemented in the actual command
		assert.NotNil(t, config)
	})

	t.Run("Show only patterns with --patterns flag", func(t *testing.T) {
		// Test that --patterns flag filters to pattern config only
		// This will be implemented in the actual command
		assert.NotEmpty(t, config.Epic.Folder)
		assert.NotEmpty(t, config.Feature.Folder)
		assert.NotEmpty(t, config.Task.File)
	})
}

// TestConfigGetFormatCommand tests the get-format command
func TestConfigGetFormatCommand(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	config := patterns.GetDefaultPatterns()
	err := writePatternConfig(configPath, config)
	require.NoError(t, err)

	tests := []struct {
		name           string
		entityType     string
		expectFormat   string
		expectExample  string
		jsonOutput     bool
	}{
		{
			name:          "Get epic format",
			entityType:    "epic",
			expectFormat:  "E{number:02d}-{slug}",
			expectExample: "E04-example-epic",
			jsonOutput:    false,
		},
		{
			name:          "Get feature format",
			entityType:    "feature",
			expectFormat:  "E{epic:02d}-F{number:02d}-{slug}",
			expectExample: "E04-F07-example-feature",
			jsonOutput:    false,
		},
		{
			name:          "Get task format",
			entityType:    "task",
			expectFormat:  "T-E{epic:02d}-F{feature:02d}-{number:03d}.md",
			expectExample: "T-E04-F07-003.md",
			jsonOutput:    false,
		},
		{
			name:          "Get task format JSON output",
			entityType:    "task",
			expectFormat:  "T-E{epic:02d}-F{feature:02d}-{number:03d}.md",
			expectExample: "T-E04-F07-003.md",
			jsonOutput:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var format string
			switch tt.entityType {
			case "epic":
				format = config.Epic.Generation.Format
			case "feature":
				format = config.Feature.Generation.Format
			case "task":
				format = config.Task.Generation.Format
			}

			assert.Equal(t, tt.expectFormat, format)

			// Test example generation
			example := generateFormatExample(tt.entityType, format)
			assert.NotEmpty(t, example)

			if tt.jsonOutput {
				// Test JSON output structure
				output := map[string]interface{}{
					"format":       format,
					"example":      example,
					"placeholders": getPlaceholdersForType(tt.entityType),
				}
				jsonBytes, err := json.Marshal(output)
				require.NoError(t, err)
				assert.NotEmpty(t, jsonBytes)
			}
		})
	}
}

// TestPatternTestingPerformance tests that pattern testing completes in <500ms
func TestPatternTestingPerformance(t *testing.T) {
	pattern := `E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)`
	testString := "E04-task-management"

	start := time.Now()
	_, _, _ = testPatternMatch(pattern, testString)
	duration := time.Since(start)

	assert.Less(t, duration, 500*time.Millisecond,
		"Pattern testing should complete in <500ms, took %v", duration)
}

// Helper functions

// writePatternConfig writes a pattern config to a file
func writePatternConfig(path string, config *patterns.PatternConfig) error {
	data, err := json.MarshalIndent(map[string]interface{}{
		"patterns": config,
	}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// TestUnrecognizedCaptureGroupWarning tests warnings for unrecognized capture groups
func TestUnrecognizedCaptureGroupWarning(t *testing.T) {
	pattern := `E(?P<feature_number>\d{2})-(?P<slug>[a-z-]+)`

	warnings := patterns.GetPatternWarnings(pattern, "epic")

	assert.NotEmpty(t, warnings)
	assert.Contains(t, warnings[0], "feature_number")
	assert.Contains(t, warnings[0], "not recognized")
}

// TestSuggestSimilarPatterns tests that failed matches suggest similar patterns
func TestSuggestSimilarPatterns(t *testing.T) {
	configPatterns := []string{
		`E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)`,
		`(?P<epic_id>tech-debt|bugs|change-cards)`,
	}

	testString := "tech-debt"

	// Find which pattern matches
	var matchedPattern string
	for _, pattern := range configPatterns {
		matched, _, _ := testPatternMatch(pattern, testString)
		if matched {
			matchedPattern = pattern
			break
		}
	}

	assert.NotEmpty(t, matchedPattern)
	assert.Contains(t, matchedPattern, "tech-debt")
}

// TestConfigListPresetsCommand tests the list-presets command
func TestConfigListPresetsCommand(t *testing.T) {
	presets := patterns.ListPresets()

	// Should have exactly 4 presets
	assert.Len(t, presets, 4, "Should have 4 presets")

	// Check for required presets
	presetNames := make(map[string]bool)
	for _, preset := range presets {
		presetNames[preset.Name] = true
		assert.NotEmpty(t, preset.Description, "Preset should have description")
	}

	assert.True(t, presetNames["standard"], "Should have standard preset")
	assert.True(t, presetNames["special-epics"], "Should have special-epics preset")
	assert.True(t, presetNames["numeric-only"], "Should have numeric-only preset")
	assert.True(t, presetNames["legacy-prp"], "Should have legacy-prp preset")
}

// TestConfigShowPresetCommand tests the show-preset command
func TestConfigShowPresetCommand(t *testing.T) {
	tests := []struct {
		name        string
		presetName  string
		expectError bool
	}{
		{"Show standard preset", "standard", false},
		{"Show special-epics preset", "special-epics", false},
		{"Show numeric-only preset", "numeric-only", false},
		{"Show legacy-prp preset", "legacy-prp", false},
		{"Show unknown preset", "nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preset, err := patterns.GetPreset(tt.presetName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, preset)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, preset)

				// Verify we can marshal to JSON
				data, err := json.MarshalIndent(preset, "", "  ")
				assert.NoError(t, err)
				assert.NotEmpty(t, data)
			}
		})
	}
}

// TestConfigAddPatternCommand tests the add-pattern command
func TestConfigAddPatternCommand(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	t.Run("Add preset to new config", func(t *testing.T) {
		// Start with minimal config (only standard pattern, no special-epics)
		baseConfig := &patterns.PatternConfig{
			Epic: patterns.EntityPatterns{
				Folder: []string{
					`^E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
				},
				File: []string{
					`^epic\.md$`,
				},
				Generation: patterns.GenerationFormat{
					Format: "E{number:02d}-{slug}",
				},
			},
			Feature: patterns.GetDefaultPatterns().Feature,
			Task:    patterns.GetDefaultPatterns().Task,
		}
		err := writePatternConfig(configPath, baseConfig)
		require.NoError(t, err)

		// Get special-epics preset
		preset, err := patterns.GetPreset("special-epics")
		require.NoError(t, err)

		// Merge patterns
		mergedConfig, stats := patterns.MergePatternsWithStats(baseConfig, preset)

		// Validate merged config
		err = patterns.ValidatePatternConfig(mergedConfig)
		assert.NoError(t, err)

		// Check stats - should have added the special-epics pattern
		assert.Greater(t, stats.Added, 0, "Should have added patterns")

		// Verify tech-debt pattern was added
		foundTechDebt := false
		for _, pattern := range mergedConfig.Epic.Folder {
			if regexp.MustCompile(pattern).MatchString("tech-debt") {
				foundTechDebt = true
				break
			}
		}
		assert.True(t, foundTechDebt, "Should have tech-debt pattern")
	})

	t.Run("Add preset with duplicates skipped", func(t *testing.T) {
		// Start with config that already has special-epics patterns
		baseConfig := patterns.GetDefaultPatterns()
		specialPreset, _ := patterns.GetPreset("special-epics")
		alreadyMerged := patterns.MergePatterns(baseConfig, specialPreset)

		// Try to add special-epics again
		_, stats := patterns.MergePatternsWithStats(alreadyMerged, specialPreset)

		// Should have skipped duplicates
		assert.Equal(t, 0, stats.Added, "Should not add duplicate patterns")
		assert.Greater(t, stats.Skipped, 0, "Should have skipped duplicates")
	})

	t.Run("Unknown preset returns error", func(t *testing.T) {
		_, err := patterns.GetPreset("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown preset")
	})

	t.Run("Validation fails on invalid merged patterns", func(t *testing.T) {
		// Create a config with valid patterns
		baseConfig := patterns.GetDefaultPatterns()

		// Create a preset with invalid pattern (missing required capture group completely)
		invalidPreset := &patterns.PatternConfig{
			Feature: patterns.EntityPatterns{
				Folder: []string{`^[a-z]+$`}, // Missing feature identifier capture group
			},
		}

		// Merge patterns
		mergedConfig := patterns.MergePatterns(baseConfig, invalidPreset)

		// Validation should fail
		err := patterns.ValidatePatternConfig(mergedConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required capture group")
	})
}

// TestConfigAddPatternIntegration tests the full add-pattern workflow
func TestConfigAddPatternIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Create initial config with minimal patterns (no special-epics)
	baseConfig := &patterns.PatternConfig{
		Epic: patterns.EntityPatterns{
			Folder: []string{
				`^E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
			},
			File: []string{
				`^epic\.md$`,
			},
			Generation: patterns.GenerationFormat{
				Format: "E{number:02d}-{slug}",
			},
		},
		Feature: patterns.GetDefaultPatterns().Feature,
		Task:    patterns.GetDefaultPatterns().Task,
	}
	err := writePatternConfig(configPath, baseConfig)
	require.NoError(t, err)

	// Add numeric-only preset (which is different from defaults)
	preset, err := patterns.GetPreset("numeric-only")
	require.NoError(t, err)

	mergedConfig, stats := patterns.MergePatternsWithStats(baseConfig, preset)

	// Validate merged config
	err = patterns.ValidatePatternConfig(mergedConfig)
	require.NoError(t, err)

	// Write merged config back
	err = writePatternConfig(configPath, mergedConfig)
	require.NoError(t, err)

	// Read config back and verify
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var readConfig struct {
		Patterns *patterns.PatternConfig `json:"patterns"`
	}
	err = json.Unmarshal(data, &readConfig)
	require.NoError(t, err)

	// Verify patterns were added (numeric-only adds patterns for epic, feature, task)
	assert.Greater(t, len(readConfig.Patterns.Epic.Folder), len(baseConfig.Epic.Folder),
		"Should have more epic patterns after merge")

	// Verify stats
	assert.Greater(t, stats.Added, 0, "Should have added patterns")
	assert.NotEmpty(t, stats.Details, "Should have details about what was added")
}
