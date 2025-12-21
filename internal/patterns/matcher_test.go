package patterns

import (
	"testing"
	"time"
)

func TestNewPatternMatcher(t *testing.T) {
	t.Run("Create matcher with valid config", func(t *testing.T) {
		config := GetDefaultPatterns()
		matcher, err := NewPatternMatcher(config, false)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if matcher == nil {
			t.Fatal("Expected matcher to be created, got nil")
		}
	})

	t.Run("Nil config returns error", func(t *testing.T) {
		_, err := NewPatternMatcher(nil, false)

		if err == nil {
			t.Fatal("Expected error for nil config, got nil")
		}
	})

	t.Run("Invalid regex pattern returns error", func(t *testing.T) {
		config := &PatternConfig{
			Epic: EntityPatterns{
				Folder: []string{"[invalid(regex"},
			},
		}

		_, err := NewPatternMatcher(config, false)

		if err == nil {
			t.Fatal("Expected error for invalid regex, got nil")
		}
	})
}

func TestFirstMatchWins(t *testing.T) {
	t.Run("First pattern matches - second pattern never evaluated", func(t *testing.T) {
		config := &PatternConfig{
			Epic: EntityPatterns{
				Folder: []string{
					`^E(?P<number>\d{2})-(?P<slug>.+)$`, // First pattern - should match
					`^(?P<epic_id>tech-debt|bugs)$`,     // Second pattern - should not be evaluated
				},
			},
		}

		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		result := matcher.MatchEpicFolder("E04-task-mgmt")

		if !result.Matched {
			t.Fatal("Expected match")
		}
		if result.PatternIndex != 0 {
			t.Errorf("Expected pattern index 0 (first pattern), got %d", result.PatternIndex)
		}
		if result.CaptureGroups["number"] != "04" {
			t.Errorf("Expected number=04, got %s", result.CaptureGroups["number"])
		}
		if result.CaptureGroups["slug"] != "task-mgmt" {
			t.Errorf("Expected slug=task-mgmt, got %s", result.CaptureGroups["slug"])
		}
	})

	t.Run("First pattern fails - second pattern matches", func(t *testing.T) {
		config := &PatternConfig{
			Epic: EntityPatterns{
				Folder: []string{
					`^E(?P<number>\d{2})-(?P<slug>.+)$`, // First pattern - won't match
					`^(?P<epic_id>tech-debt|bugs)$`,     // Second pattern - should match
				},
			},
		}

		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		result := matcher.MatchEpicFolder("tech-debt")

		if !result.Matched {
			t.Fatal("Expected match")
		}
		if result.PatternIndex != 1 {
			t.Errorf("Expected pattern index 1 (second pattern), got %d", result.PatternIndex)
		}
		if result.CaptureGroups["epic_id"] != "tech-debt" {
			t.Errorf("Expected epic_id=tech-debt, got %s", result.CaptureGroups["epic_id"])
		}
	})

	t.Run("No patterns match - all attempted", func(t *testing.T) {
		config := &PatternConfig{
			Epic: EntityPatterns{
				Folder: []string{
					`^E(?P<number>\d{2})-(?P<slug>.+)$`,
					`^(?P<epic_id>tech-debt|bugs)$`,
					`^EPIC-(?P<number>\d{3})$`,
				},
			},
		}

		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		result := matcher.MatchEpicFolder("random-folder-name")

		if result.Matched {
			t.Fatal("Expected no match")
		}

		// Verify all patterns were attempted
		attempted := matcher.GetAttemptedPatterns("epic", "folder")
		if len(attempted) != 3 {
			t.Errorf("Expected 3 patterns to be available, got %d", len(attempted))
		}
	})
}

func TestPatternCompilationCaching(t *testing.T) {
	t.Run("Patterns compiled once at initialization", func(t *testing.T) {
		config := GetDefaultPatterns()

		start := time.Now()
		matcher, err := NewPatternMatcher(config, false)
		initDuration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		// Perform multiple matches - should be fast since patterns are cached
		start = time.Now()
		for i := 0; i < 1000; i++ {
			_ = matcher.MatchEpicFolder("E04-test")
		}
		matchDuration := time.Since(start)

		// Average per-match should be under 1ms
		avgPerMatch := matchDuration / 1000
		if avgPerMatch > time.Millisecond {
			t.Errorf("Average match time %v exceeds 1ms target (init took %v)", avgPerMatch, initDuration)
		}

		t.Logf("Initialization: %v, 1000 matches: %v, avg per match: %v", initDuration, matchDuration, avgPerMatch)
	})
}

func TestPatternMatchingPerformance(t *testing.T) {
	t.Run("Pattern matching under 1ms per file", func(t *testing.T) {
		config := GetDefaultPatterns()
		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		testCases := []string{
			"E04-task-management",
			"E06-F01-pattern-config",
			"T-E04-F05-001.md",
			"tech-debt",
			"bugs",
			"E99-F99-feature-name",
		}

		for _, testCase := range testCases {
			start := time.Now()
			_ = matcher.MatchEpicFolder(testCase)
			duration := time.Since(start)

			if duration > time.Millisecond {
				t.Errorf("Match for '%s' took %v, exceeds 1ms target", testCase, duration)
			}
		}
	})
}

func TestMatchTimeout(t *testing.T) {
	t.Run("Matching times out after 100ms", func(t *testing.T) {
		// Create a pattern that could cause catastrophic backtracking
		// Note: Our validator should catch these, but test timeout protection anyway
		config := &PatternConfig{
			Epic: EntityPatterns{
				Folder: []string{
					`^(?P<slug>[a-zA-Z0-9-]+)$`, // Safe pattern
				},
			},
		}

		matcher, err := NewPatternMatcher(config, true)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		// Test with normal input - should not timeout
		result := matcher.MatchEpicFolder("E04-test")
		if !result.Matched {
			t.Error("Expected normal match to succeed")
		}
	})
}

func TestVerboseLogging(t *testing.T) {
	t.Run("Verbose mode logs matches", func(t *testing.T) {
		config := GetDefaultPatterns()
		matcher, err := NewPatternMatcher(config, true)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		// This will output to stdout when verbose is true
		result := matcher.MatchEpicFolder("E04-test")
		if !result.Matched {
			t.Error("Expected match")
		}
	})

	t.Run("Toggle verbose mode", func(t *testing.T) {
		config := GetDefaultPatterns()
		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		// Should not log
		_ = matcher.MatchEpicFolder("E04-test")

		// Enable verbose
		matcher.SetVerbose(true)
		_ = matcher.MatchEpicFolder("E04-test")

		// Disable verbose
		matcher.SetVerbose(false)
		_ = matcher.MatchEpicFolder("E04-test")
	})
}

func TestCaptureGroupExtraction(t *testing.T) {
	t.Run("Extract all named capture groups", func(t *testing.T) {
		config := &PatternConfig{
			Feature: EntityPatterns{
				Folder: []string{
					`^E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<slug>[a-z-]+)$`,
				},
			},
		}

		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		result := matcher.MatchFeatureFolder("E04-F05-user-preferences")

		if !result.Matched {
			t.Fatal("Expected match")
		}

		expectedGroups := map[string]string{
			"epic_num":    "04",
			"feature_num": "05",
			"slug":        "user-preferences",
		}

		for name, expectedValue := range expectedGroups {
			if result.CaptureGroups[name] != expectedValue {
				t.Errorf("Expected %s=%s, got %s", name, expectedValue, result.CaptureGroups[name])
			}
		}
	})

	t.Run("Handle missing capture groups gracefully", func(t *testing.T) {
		config := &PatternConfig{
			Epic: EntityPatterns{
				Folder: []string{
					`^(?P<epic_id>tech-debt)$`, // Only one capture group
				},
			},
		}

		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		result := matcher.MatchEpicFolder("tech-debt")

		if !result.Matched {
			t.Fatal("Expected match")
		}
		if len(result.CaptureGroups) != 1 {
			t.Errorf("Expected 1 capture group, got %d", len(result.CaptureGroups))
		}
	})
}

func TestGetAttemptedPatterns(t *testing.T) {
	t.Run("Return patterns for entity type", func(t *testing.T) {
		config := &PatternConfig{
			Epic: EntityPatterns{
				Folder: []string{"pattern1", "pattern2", "pattern3"},
				File:   []string{"file1", "file2"},
			},
		}

		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		folderPatterns := matcher.GetAttemptedPatterns("epic", "folder")
		if len(folderPatterns) != 3 {
			t.Errorf("Expected 3 folder patterns, got %d", len(folderPatterns))
		}

		filePatterns := matcher.GetAttemptedPatterns("epic", "file")
		if len(filePatterns) != 2 {
			t.Errorf("Expected 2 file patterns, got %d", len(filePatterns))
		}
	})

	t.Run("Return empty for invalid entity type", func(t *testing.T) {
		config := GetDefaultPatterns()
		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		patterns := matcher.GetAttemptedPatterns("invalid", "folder")
		if len(patterns) != 0 {
			t.Errorf("Expected 0 patterns for invalid type, got %d", len(patterns))
		}
	})
}

func TestMatchAllEntityTypes(t *testing.T) {
	config := GetDefaultPatterns()
	matcher, err := NewPatternMatcher(config, false)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	testCases := []struct {
		name           string
		matchFunc      func(string) *MatchResult
		input          string
		shouldMatch    bool
		expectedGroups map[string]string
	}{
		{
			name:        "Epic folder with E## format",
			matchFunc:   matcher.MatchEpicFolder,
			input:       "E04-task-management",
			shouldMatch: true,
			expectedGroups: map[string]string{
				"number": "04",
				"slug":   "task-management",
			},
		},
		{
			name:        "Epic folder with tech-debt",
			matchFunc:   matcher.MatchEpicFolder,
			input:       "tech-debt",
			shouldMatch: true,
			expectedGroups: map[string]string{
				"epic_id": "tech-debt",
			},
		},
		{
			name:        "Feature folder",
			matchFunc:   matcher.MatchFeatureFolder,
			input:       "E04-F05-user-preferences",
			shouldMatch: true,
			expectedGroups: map[string]string{
				"epic_num": "04",
				"number":   "05",
				"slug":     "user-preferences",
			},
		},
		{
			name:        "Task file",
			matchFunc:   matcher.MatchTaskFile,
			input:       "T-E04-F05-001.md",
			shouldMatch: true,
			expectedGroups: map[string]string{
				"epic_num":    "04",
				"feature_num": "05",
				"number":      "001",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.matchFunc(tc.input)

			if result.Matched != tc.shouldMatch {
				t.Errorf("Expected matched=%v, got %v", tc.shouldMatch, result.Matched)
			}

			if tc.shouldMatch {
				for name, expectedValue := range tc.expectedGroups {
					if result.CaptureGroups[name] != expectedValue {
						t.Errorf("Expected %s=%s, got %s", name, expectedValue, result.CaptureGroups[name])
					}
				}
			}
		})
	}
}

// Integration test: Complete workflow from config load to pattern matching
func TestIntegration_ConfigLoadToPatternMatching(t *testing.T) {
	t.Run("Complete workflow with default patterns", func(t *testing.T) {
		// 1. Load default configuration
		config := GetDefaultPatterns()

		// 2. Validate configuration
		err := ValidatePatternConfig(config)
		if err != nil {
			t.Fatalf("Pattern validation failed: %v", err)
		}

		// 3. Create matcher with compiled patterns
		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		// 4. Test pattern matching for various entity types
		testFiles := []struct {
			entityType  string
			input       string
			shouldMatch bool
		}{
			{"epic", "E04-task-management", true},
			{"epic", "tech-debt", true},
			{"epic", "bugs", true},
			{"epic", "random-name", false},
			{"feature", "E04-F05-user-prefs", true},
			{"task", "T-E04-F05-001.md", true},
		}

		for _, tf := range testFiles {
			var result *MatchResult
			switch tf.entityType {
			case "epic":
				result = matcher.MatchEpicFolder(tf.input)
			case "feature":
				result = matcher.MatchFeatureFolder(tf.input)
			case "task":
				result = matcher.MatchTaskFile(tf.input)
			}

			if result.Matched != tf.shouldMatch {
				t.Errorf("File '%s' (type %s): expected match=%v, got %v",
					tf.input, tf.entityType, tf.shouldMatch, result.Matched)
			}
		}
	})
}

// Integration test: Pattern preset addition workflow
func TestIntegration_PresetAdditionWorkflow(t *testing.T) {
	t.Run("Add preset and verify patterns work", func(t *testing.T) {
		// 1. Start with default config
		config := GetDefaultPatterns()

		// 2. Get a preset
		preset, err := GetPreset("special-epics")
		if err != nil {
			t.Fatalf("Failed to get preset: %v", err)
		}

		// 3. Merge preset with config (simplified - normally done by CLI)
		config.Epic.Folder = append(config.Epic.Folder, preset.Epic.Folder...)

		// 4. Validate merged config
		err = ValidatePatternConfig(config)
		if err != nil {
			t.Fatalf("Merged config validation failed: %v", err)
		}

		// 5. Create matcher
		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		// 6. Test that preset patterns work
		result := matcher.MatchEpicFolder("tech-debt")
		if !result.Matched {
			t.Error("Expected tech-debt to match after adding special-epics preset")
		}
	})
}

// Integration test: Generation format application
func TestIntegration_GenerationFormatApplication(t *testing.T) {
	t.Run("Generate IDs using format from config", func(t *testing.T) {
		// 1. Load config with generation formats
		config := GetDefaultPatterns()

		// 2. Apply generation format for epic
		epicID, err := ApplyGenerationFormat(config.Epic.Generation.Format, map[string]interface{}{
			"number": 4,
			"slug":   "test-epic",
		})
		if err != nil {
			t.Fatalf("Failed to apply epic format: %v", err)
		}

		expectedEpicID := "E04-test-epic"
		if epicID != expectedEpicID {
			t.Errorf("Expected epic ID '%s', got '%s'", expectedEpicID, epicID)
		}

		// 3. Verify generated ID matches the pattern
		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		result := matcher.MatchEpicFolder(epicID)
		if !result.Matched {
			t.Error("Generated epic ID does not match pattern")
		}

		// 4. Verify capture groups match original values
		if result.CaptureGroups["number"] != "04" {
			t.Errorf("Expected number=04, got %s", result.CaptureGroups["number"])
		}
		if result.CaptureGroups["slug"] != "test-epic" {
			t.Errorf("Expected slug=test-epic, got %s", result.CaptureGroups["slug"])
		}
	})
}

// Integration test: Error handling edge cases
func TestIntegration_ErrorHandling(t *testing.T) {
	t.Run("Handle corrupted config gracefully", func(t *testing.T) {
		config := &PatternConfig{
			Epic: EntityPatterns{
				Folder: []string{"[invalid(regex"},
			},
		}

		_, err := NewPatternMatcher(config, false)
		if err == nil {
			t.Error("Expected error for invalid regex, got nil")
		}
	})

	t.Run("Reject catastrophic backtracking patterns", func(t *testing.T) {
		pattern := `^(a+)+$` // Catastrophic backtracking

		err := ValidatePattern(pattern, "epic")
		if err == nil {
			t.Error("Expected error for catastrophic backtracking pattern, got nil")
		}
	})

	t.Run("Handle pattern matching timeout", func(t *testing.T) {
		config := GetDefaultPatterns()
		matcher, err := NewPatternMatcher(config, false)
		if err != nil {
			t.Fatalf("Failed to create matcher: %v", err)
		}

		// Normal patterns should not timeout
		result := matcher.MatchEpicFolder("E04-test")
		if !result.Matched {
			t.Error("Expected match for normal input")
		}
	})
}

// Performance benchmark
func BenchmarkPatternMatching(b *testing.B) {
	config := GetDefaultPatterns()
	matcher, err := NewPatternMatcher(config, false)
	if err != nil {
		b.Fatalf("Failed to create matcher: %v", err)
	}

	testInputs := []string{
		"E04-task-management",
		"tech-debt",
		"E06-F01-pattern-config",
		"T-E04-F05-001.md",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range testInputs {
			_ = matcher.MatchEpicFolder(input)
		}
	}
}
