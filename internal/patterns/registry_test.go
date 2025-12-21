package patterns

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewPatternRegistry(t *testing.T) {
	t.Run("Create registry with valid config", func(t *testing.T) {
		config := GetDefaultPatterns()
		registry, err := NewPatternRegistry(config, nil)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if registry == nil {
			t.Fatal("Expected registry to be created, got nil")
		}
		if registry.config == nil {
			t.Fatal("Expected config to be set, got nil")
		}
		if registry.matcher == nil {
			t.Fatal("Expected matcher to be created, got nil")
		}
	})

	t.Run("Create registry with verbose option", func(t *testing.T) {
		config := GetDefaultPatterns()
		opts := &RegistryOptions{Verbose: true}
		registry, err := NewPatternRegistry(config, opts)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if !registry.verbose {
			t.Error("Expected verbose to be true")
		}
	})

	t.Run("Nil config returns error", func(t *testing.T) {
		_, err := NewPatternRegistry(nil, nil)

		if err == nil {
			t.Fatal("Expected error for nil config, got nil")
		}
	})

	t.Run("Invalid pattern config returns validation error", func(t *testing.T) {
		config := &PatternConfig{
			Epic: EntityPatterns{
				Folder: []string{"[invalid(regex"},
			},
		}

		_, err := NewPatternRegistry(config, nil)

		if err == nil {
			t.Fatal("Expected error for invalid regex, got nil")
		}
	})

	t.Run("Config with missing required capture groups returns error", func(t *testing.T) {
		config := &PatternConfig{
			Epic: EntityPatterns{
				Folder: []string{`^E\d{2}$`}, // Missing capture groups
			},
		}

		_, err := NewPatternRegistry(config, nil)

		if err == nil {
			t.Fatal("Expected error for missing capture groups, got nil")
		}
	})
}

func TestNewPatternRegistryFromDefaults(t *testing.T) {
	t.Run("Create registry from defaults", func(t *testing.T) {
		registry, err := NewPatternRegistryFromDefaults(false)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if registry == nil {
			t.Fatal("Expected registry to be created, got nil")
		}

		// Verify default patterns are loaded
		epicPatterns := registry.GetEpicPatterns()
		if len(epicPatterns) == 0 {
			t.Error("Expected default epic patterns to be loaded")
		}

		taskPatterns := registry.GetTaskPatterns()
		if len(taskPatterns) == 0 {
			t.Error("Expected default task patterns to be loaded")
		}
	})

	t.Run("Create registry from defaults with verbose", func(t *testing.T) {
		registry, err := NewPatternRegistryFromDefaults(true)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if !registry.verbose {
			t.Error("Expected verbose to be true")
		}
	})
}

func TestLoadPatternRegistryFromFile(t *testing.T) {
	t.Run("Load registry from valid config file", func(t *testing.T) {
		// Create temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		configContent := `{
			"default_epic": null,
			"default_agent": null,
			"color_enabled": true,
			"json_output": false,
			"patterns": {
				"epic": {
					"folder": ["^E(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$"],
					"file": ["^epic\\.md$"],
					"generation": {"format": "E{number:02d}-{slug}"}
				},
				"feature": {
					"folder": ["^E(?P<epic_num>\\d{2})-F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$"],
					"file": ["^prd\\.md$"],
					"generation": {"format": "E{epic:02d}-F{number:02d}-{slug}"}
				},
				"task": {
					"folder": [],
					"file": ["^T-E(?P<epic_num>\\d{2})-F(?P<feature_num>\\d{2})-(?P<number>\\d{3}).*\\.md$"],
					"generation": {"format": "T-E{epic:02d}-F{feature:02d}-{number:03d}.md"}
				}
			}
		}`

		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		registry, err := LoadPatternRegistryFromFile(configPath, false)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if registry == nil {
			t.Fatal("Expected registry to be created, got nil")
		}

		// Verify patterns are loaded
		epicPatterns := registry.GetEpicPatterns()
		if len(epicPatterns) != 1 {
			t.Errorf("Expected 1 epic pattern, got %d", len(epicPatterns))
		}
	})

	t.Run("Load registry from config without patterns uses defaults", func(t *testing.T) {
		// Create temporary config file without patterns
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		configContent := `{
			"default_epic": null,
			"default_agent": null,
			"color_enabled": true,
			"json_output": false
		}`

		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		registry, err := LoadPatternRegistryFromFile(configPath, false)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Verify default patterns are loaded
		epicPatterns := registry.GetEpicPatterns()
		if len(epicPatterns) == 0 {
			t.Error("Expected default epic patterns to be loaded when config has no patterns")
		}
	})

	t.Run("Load registry from non-existent file returns error", func(t *testing.T) {
		_, err := LoadPatternRegistryFromFile("/non/existent/file.json", false)

		if err == nil {
			t.Fatal("Expected error for non-existent file, got nil")
		}
	})

	t.Run("Load registry from invalid JSON returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		if err := os.WriteFile(configPath, []byte("invalid json{"), 0644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		_, err := LoadPatternRegistryFromFile(configPath, false)

		if err == nil {
			t.Fatal("Expected error for invalid JSON, got nil")
		}
	})
}

func TestRegistryPatternMatching(t *testing.T) {
	registry, err := NewPatternRegistryFromDefaults(false)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	testCases := []struct {
		name           string
		matchFunc      func(string) *MatchResult
		input          string
		shouldMatch    bool
		expectedGroups map[string]string
	}{
		{
			name:        "Match standard task format",
			matchFunc:   registry.MatchTaskFile,
			input:       "T-E04-F05-001.md",
			shouldMatch: true,
			expectedGroups: map[string]string{
				"epic_num":    "04",
				"feature_num": "05",
				"number":      "001",
			},
		},
		{
			name:        "Match numbered task format",
			matchFunc:   registry.MatchTaskFile,
			input:       "042-implement-feature.md",
			shouldMatch: true,
			expectedGroups: map[string]string{
				"number": "042",
				"slug":   "implement-feature",
			},
		},
		{
			name:        "Match PRP task format",
			matchFunc:   registry.MatchTaskFile,
			input:       "authentication-middleware.prp.md",
			shouldMatch: true,
			expectedGroups: map[string]string{
				"slug": "authentication-middleware",
			},
		},
		{
			name:        "Match epic folder with E## format",
			matchFunc:   registry.MatchEpicFolder,
			input:       "E04-task-management",
			shouldMatch: true,
			expectedGroups: map[string]string{
				"number": "04",
				"slug":   "task-management",
			},
		},
		{
			name:        "Match special epic (tech-debt)",
			matchFunc:   registry.MatchEpicFolder,
			input:       "tech-debt",
			shouldMatch: true,
			expectedGroups: map[string]string{
				"epic_id": "tech-debt",
			},
		},
		{
			name:        "Match feature folder",
			matchFunc:   registry.MatchFeatureFolder,
			input:       "E04-F05-user-preferences",
			shouldMatch: true,
			expectedGroups: map[string]string{
				"epic_num": "04",
				"number":   "05",
				"slug":     "user-preferences",
			},
		},
		{
			name:        "No match for invalid format",
			matchFunc:   registry.MatchTaskFile,
			input:       "random-file.txt",
			shouldMatch: false,
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

func TestRegistryFirstMatchWins(t *testing.T) {
	t.Run("First pattern matches - second not evaluated", func(t *testing.T) {
		config := &PatternConfig{
			Task: EntityPatterns{
				File: []string{
					`^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3}).*\.md$`, // Should match
					`^(?P<number>\d{3})-(?P<slug>.+)\.md$`,                                       // Should not be evaluated
				},
			},
		}

		registry, err := NewPatternRegistry(config, nil)
		if err != nil {
			t.Fatalf("Failed to create registry: %v", err)
		}

		result := registry.MatchTaskFile("T-E04-F05-001.md")

		if !result.Matched {
			t.Fatal("Expected match")
		}
		if result.PatternIndex != 0 {
			t.Errorf("Expected pattern index 0 (first pattern), got %d", result.PatternIndex)
		}
	})

	t.Run("First pattern fails - second pattern matches", func(t *testing.T) {
		config := &PatternConfig{
			Task: EntityPatterns{
				File: []string{
					`^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3}).*\.md$`, // Won't match
					`^(?P<slug>.+)\.prp\.md$`,                                                    // Should match
				},
			},
		}

		registry, err := NewPatternRegistry(config, nil)
		if err != nil {
			t.Fatalf("Failed to create registry: %v", err)
		}

		result := registry.MatchTaskFile("implement-auth.prp.md")

		if !result.Matched {
			t.Fatal("Expected match")
		}
		if result.PatternIndex != 1 {
			t.Errorf("Expected pattern index 1 (second pattern), got %d", result.PatternIndex)
		}
		if result.CaptureGroups["slug"] != "implement-auth" {
			t.Errorf("Expected slug=implement-auth, got %s", result.CaptureGroups["slug"])
		}
	})
}

func TestRegistryGetPatterns(t *testing.T) {
	registry, err := NewPatternRegistryFromDefaults(false)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	t.Run("Get task patterns", func(t *testing.T) {
		patterns := registry.GetTaskPatterns()
		if len(patterns) == 0 {
			t.Error("Expected task patterns, got empty array")
		}
	})

	t.Run("Get feature patterns", func(t *testing.T) {
		patterns := registry.GetFeaturePatterns()
		if len(patterns) == 0 {
			t.Error("Expected feature patterns, got empty array")
		}
	})

	t.Run("Get epic patterns", func(t *testing.T) {
		patterns := registry.GetEpicPatterns()
		if len(patterns) == 0 {
			t.Error("Expected epic patterns, got empty array")
		}
	})

	t.Run("Get config", func(t *testing.T) {
		config := registry.GetConfig()
		if config == nil {
			t.Error("Expected config, got nil")
		}
	})
}

func TestRegistrySetVerbose(t *testing.T) {
	registry, err := NewPatternRegistryFromDefaults(false)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	t.Run("Toggle verbose mode", func(t *testing.T) {
		if registry.verbose {
			t.Error("Expected verbose to be false initially")
		}

		registry.SetVerbose(true)
		if !registry.verbose {
			t.Error("Expected verbose to be true after SetVerbose(true)")
		}

		// Test that matching still works with verbose enabled
		result := registry.MatchTaskFile("T-E04-F05-001.md")
		if !result.Matched {
			t.Error("Expected match to work with verbose enabled")
		}

		registry.SetVerbose(false)
		if registry.verbose {
			t.Error("Expected verbose to be false after SetVerbose(false)")
		}
	})
}

func TestRegistryValidatePattern(t *testing.T) {
	registry, err := NewPatternRegistryFromDefaults(false)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	t.Run("Validate valid pattern", func(t *testing.T) {
		pattern := `^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3}).*\.md$`
		err := registry.ValidatePattern(pattern, "task")

		if err != nil {
			t.Errorf("Expected valid pattern to pass validation, got error: %v", err)
		}
	})

	t.Run("Validate invalid regex syntax", func(t *testing.T) {
		pattern := `[invalid(regex`
		err := registry.ValidatePattern(pattern, "task")

		if err == nil {
			t.Error("Expected error for invalid regex syntax, got nil")
		}
	})

	t.Run("Validate pattern with missing capture groups", func(t *testing.T) {
		pattern := `^T-E\d{2}-F\d{2}-\d{3}\.md$` // Missing named capture groups
		err := registry.ValidatePattern(pattern, "task")

		if err == nil {
			t.Error("Expected error for missing capture groups, got nil")
		}
	})

	t.Run("Validate pattern with catastrophic backtracking", func(t *testing.T) {
		pattern := `^(a+)+$` // Catastrophic backtracking
		err := registry.ValidatePattern(pattern, "epic")

		if err == nil {
			t.Error("Expected error for catastrophic backtracking pattern, got nil")
		}
	})
}

func TestRegistryValidatePatternWithTimeout(t *testing.T) {
	registry, err := NewPatternRegistryFromDefaults(false)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	t.Run("Validate pattern with timeout", func(t *testing.T) {
		pattern := `^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3}).*\.md$`
		err := registry.ValidatePatternWithTimeout(pattern, "task", 100*time.Millisecond)

		if err != nil {
			t.Errorf("Expected valid pattern to pass validation, got error: %v", err)
		}
	})
}

func TestRegistryGetPatternWarnings(t *testing.T) {
	registry, err := NewPatternRegistryFromDefaults(false)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	t.Run("Get warnings for pattern with unrecognized capture group", func(t *testing.T) {
		pattern := `^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<unknown_field>\d{3}).*\.md$`
		warnings := registry.GetPatternWarnings(pattern, "task")

		if len(warnings) == 0 {
			t.Error("Expected warnings for unrecognized capture group, got none")
		}
	})

	t.Run("No warnings for valid pattern", func(t *testing.T) {
		pattern := `^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3}).*\.md$`
		warnings := registry.GetPatternWarnings(pattern, "task")

		if len(warnings) > 0 {
			t.Errorf("Expected no warnings for valid pattern, got %d warnings", len(warnings))
		}
	})
}

func TestRegistryGenerateKeys(t *testing.T) {
	registry, err := NewPatternRegistryFromDefaults(false)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	t.Run("Generate task key", func(t *testing.T) {
		params := map[string]interface{}{
			"epic":    4,
			"feature": 5,
			"number":  7,
		}

		key, err := registry.GenerateTaskKey(params)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		expectedKey := "T-E04-F05-007.md"
		if key != expectedKey {
			t.Errorf("Expected key=%s, got %s", expectedKey, key)
		}
	})

	t.Run("Generate feature key", func(t *testing.T) {
		params := map[string]interface{}{
			"epic":   4,
			"number": 5,
			"slug":   "user-prefs",
		}

		key, err := registry.GenerateFeatureKey(params)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		expectedKey := "E04-F05-user-prefs"
		if key != expectedKey {
			t.Errorf("Expected key=%s, got %s", expectedKey, key)
		}
	})

	t.Run("Generate epic key", func(t *testing.T) {
		params := map[string]interface{}{
			"number": 6,
			"slug":   "intelligent-scanning",
		}

		key, err := registry.GenerateEpicKey(params)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		expectedKey := "E06-intelligent-scanning"
		if key != expectedKey {
			t.Errorf("Expected key=%s, got %s", expectedKey, key)
		}
	})
}

func TestRegistryPerformance(t *testing.T) {
	t.Run("Pattern compilation is cached", func(t *testing.T) {
		config := GetDefaultPatterns()

		start := time.Now()
		registry, err := NewPatternRegistry(config, nil)
		initDuration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to create registry: %v", err)
		}

		// Perform multiple matches - should be fast since patterns are cached
		start = time.Now()
		for i := 0; i < 1000; i++ {
			_ = registry.MatchTaskFile("T-E04-F05-001.md")
		}
		matchDuration := time.Since(start)

		avgPerMatch := matchDuration / 1000
		if avgPerMatch > time.Millisecond {
			t.Errorf("Average match time %v exceeds 1ms target (init took %v)", avgPerMatch, initDuration)
		}

		t.Logf("Initialization: %v, 1000 matches: %v, avg per match: %v", initDuration, matchDuration, avgPerMatch)
	})
}

// Integration test: Complete workflow from config file to pattern matching
func TestIntegration_RegistryWorkflow(t *testing.T) {
	t.Run("Load config, validate, and match patterns", func(t *testing.T) {
		// Create temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		configContent := `{
			"default_epic": null,
			"default_agent": null,
			"color_enabled": true,
			"json_output": false,
			"patterns": {
				"epic": {
					"folder": [
						"^E(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$",
						"^(?P<epic_id>tech-debt|bugs|change-cards)$"
					],
					"file": ["^epic\\.md$"],
					"generation": {"format": "E{number:02d}-{slug}"}
				},
				"feature": {
					"folder": ["^E(?P<epic_num>\\d{2})-F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$"],
					"file": ["^prd\\.md$"],
					"generation": {"format": "E{epic:02d}-F{number:02d}-{slug}"}
				},
				"task": {
					"folder": [],
					"file": [
						"^T-E(?P<epic_num>\\d{2})-F(?P<feature_num>\\d{2})-(?P<number>\\d{3}).*\\.md$",
						"^(?P<number>\\d{3})-(?P<slug>.+)\\.md$",
						"^(?P<slug>.+)\\.prp\\.md$"
					],
					"generation": {"format": "T-E{epic:02d}-F{feature:02d}-{number:03d}.md"}
				}
			}
		}`

		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		// Load registry from file
		registry, err := LoadPatternRegistryFromFile(configPath, false)
		if err != nil {
			t.Fatalf("Failed to load registry: %v", err)
		}

		// Test pattern matching for various file types
		testCases := []struct {
			input       string
			matchFunc   func(string) *MatchResult
			shouldMatch bool
		}{
			{"T-E04-F05-001.md", registry.MatchTaskFile, true},
			{"042-implement-feature.md", registry.MatchTaskFile, true},
			{"auth-middleware.prp.md", registry.MatchTaskFile, true},
			{"E04-task-management", registry.MatchEpicFolder, true},
			{"tech-debt", registry.MatchEpicFolder, true},
			{"E04-F05-user-prefs", registry.MatchFeatureFolder, true},
			{"random-file.txt", registry.MatchTaskFile, false},
		}

		for _, tc := range testCases {
			result := tc.matchFunc(tc.input)
			if result.Matched != tc.shouldMatch {
				t.Errorf("File '%s': expected match=%v, got %v", tc.input, tc.shouldMatch, result.Matched)
			}
		}

		// Test key generation
		taskKey, err := registry.GenerateTaskKey(map[string]interface{}{
			"epic":    4,
			"feature": 5,
			"number":  7,
		})
		if err != nil {
			t.Errorf("Failed to generate task key: %v", err)
		}
		if taskKey != "T-E04-F05-007.md" {
			t.Errorf("Expected task key T-E04-F05-007.md, got %s", taskKey)
		}

		// Verify generated key matches pattern
		result := registry.MatchTaskFile(taskKey)
		if !result.Matched {
			t.Error("Generated task key does not match pattern")
		}
	})
}

// Performance benchmark
func BenchmarkRegistryPatternMatching(b *testing.B) {
	registry, err := NewPatternRegistryFromDefaults(false)
	if err != nil {
		b.Fatalf("Failed to create registry: %v", err)
	}

	testInputs := []string{
		"T-E04-F05-001.md",
		"042-implement-feature.md",
		"auth-middleware.prp.md",
		"E04-task-management",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range testInputs {
			_ = registry.MatchTaskFile(input)
		}
	}
}
