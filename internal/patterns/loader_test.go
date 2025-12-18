package patterns

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("Load config with patterns", func(t *testing.T) {
		// Create temp config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		config := Config{
			ColorEnabled: true,
			JSONOutput:   false,
			Patterns:     GetDefaultPatterns(),
		}

		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		// Load config
		loaded, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if loaded.Patterns == nil {
			t.Error("Loaded config should have patterns")
		}

		if len(loaded.Patterns.Epic.Folder) == 0 {
			t.Error("Loaded patterns should include epic folder patterns")
		}
	})

	t.Run("Load config without patterns - should fallback to defaults", func(t *testing.T) {
		// Create temp config file WITHOUT patterns (backward compatibility)
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		config := Config{
			ColorEnabled: true,
			JSONOutput:   false,
			// No Patterns field - simulates old config format
		}

		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		// Load config
		loaded, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// Should have default patterns even though config didn't specify them
		if loaded.Patterns == nil {
			t.Error("Loaded config should have default patterns when patterns not specified")
		}

		if len(loaded.Patterns.Epic.Folder) == 0 {
			t.Error("Default patterns should include epic folder patterns")
		}

		// Verify it has the standard epic pattern
		foundStandard := false
		for _, pattern := range loaded.Patterns.Epic.Folder {
			if pattern == `^E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$` {
				foundStandard = true
				break
			}
		}
		if !foundStandard {
			t.Error("Default patterns should include standard E##-slug pattern")
		}
	})

	t.Run("Load non-existent config file", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path/.sharkconfig.json")
		if err == nil {
			t.Error("LoadConfig should return error for non-existent file")
		}
	})

	t.Run("Load invalid JSON config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		// Write invalid JSON
		if err := os.WriteFile(configPath, []byte("{invalid json"), 0644); err != nil {
			t.Fatalf("Failed to write invalid config: %v", err)
		}

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Error("LoadConfig should return error for invalid JSON")
		}
	})
}

func TestMergeWithDefaults(t *testing.T) {
	t.Run("Merge nil patterns returns defaults", func(t *testing.T) {
		result := MergeWithDefaults(nil)

		if result == nil {
			t.Fatal("MergeWithDefaults should not return nil")
		}

		if len(result.Epic.Folder) == 0 {
			t.Error("Merged patterns should include default epic folder patterns")
		}
	})

	t.Run("Merge partial user patterns with defaults", func(t *testing.T) {
		// User provides only epic patterns, expects defaults for feature/task
		userPatterns := &PatternConfig{
			Epic: EntityPatterns{
				Folder: []string{`^custom-epic-pattern$`},
				File:   []string{`^custom-epic.md$`},
				Generation: GenerationFormat{
					Format: "custom-{number}",
				},
			},
			// Feature and Task patterns not specified
		}

		result := MergeWithDefaults(userPatterns)

		// Should use user's epic patterns
		if len(result.Epic.Folder) != 1 || result.Epic.Folder[0] != `^custom-epic-pattern$` {
			t.Error("Should use user's epic folder patterns")
		}

		// Should use default feature patterns
		if len(result.Feature.Folder) == 0 {
			t.Error("Should use default feature folder patterns when user doesn't specify")
		}

		// Should use default task patterns
		if len(result.Task.File) == 0 {
			t.Error("Should use default task file patterns when user doesn't specify")
		}
	})

	t.Run("User patterns take precedence over defaults", func(t *testing.T) {
		userPatterns := &PatternConfig{
			Epic: EntityPatterns{
				Folder: []string{`^user-pattern$`},
				File:   []string{},
				Generation: GenerationFormat{
					Format: "user-{number}",
				},
			},
			Feature: EntityPatterns{
				Folder:     []string{},
				File:       []string{},
				Generation: GenerationFormat{},
			},
			Task: EntityPatterns{
				Folder:     []string{},
				File:       []string{},
				Generation: GenerationFormat{},
			},
		}

		result := MergeWithDefaults(userPatterns)

		// User's epic folder pattern should be used
		if len(result.Epic.Folder) != 1 || result.Epic.Folder[0] != `^user-pattern$` {
			t.Error("User patterns should take precedence")
		}

		// User's epic generation format should be used
		if result.Epic.Generation.Format != "user-{number}" {
			t.Error("User generation format should take precedence")
		}

		// Empty user arrays should trigger default fallback
		defaults := GetDefaultPatterns()
		if len(result.Epic.File) == 0 {
			// Empty file patterns should use defaults
			if len(defaults.Epic.File) > 0 {
				t.Error("Empty user file patterns should fallback to defaults")
			}
		}
	})
}
