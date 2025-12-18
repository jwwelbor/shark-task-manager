package patterns

import (
	"encoding/json"
	"testing"
)

func TestGetPresetNames(t *testing.T) {
	names := GetPresetNames()

	// Should have exactly 4 presets
	if len(names) != 4 {
		t.Errorf("Expected 4 presets, got %d", len(names))
	}

	// Check for required presets
	required := []string{"standard", "special-epics", "numeric-only", "legacy-prp"}
	for _, name := range required {
		found := false
		for _, preset := range names {
			if preset == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required preset '%s' not found", name)
		}
	}
}

func TestGetPresetInfo(t *testing.T) {
	tests := []struct {
		name        string
		presetName  string
		shouldExist bool
	}{
		{"standard preset exists", "standard", true},
		{"special-epics preset exists", "special-epics", true},
		{"numeric-only preset exists", "numeric-only", true},
		{"legacy-prp preset exists", "legacy-prp", true},
		{"unknown preset does not exist", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := GetPresetInfo(tt.presetName)

			if tt.shouldExist {
				if err != nil {
					t.Errorf("Expected preset to exist, got error: %v", err)
				}
				if info == nil {
					t.Errorf("Expected preset info, got nil")
				}
				if info.Name != tt.presetName {
					t.Errorf("Expected name '%s', got '%s'", tt.presetName, info.Name)
				}
				if info.Description == "" {
					t.Errorf("Expected non-empty description")
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for unknown preset, got nil")
				}
			}
		})
	}
}

func TestGetPreset(t *testing.T) {
	tests := []struct {
		name        string
		presetName  string
		shouldExist bool
	}{
		{"standard preset", "standard", true},
		{"special-epics preset", "special-epics", true},
		{"numeric-only preset", "numeric-only", true},
		{"legacy-prp preset", "legacy-prp", true},
		{"unknown preset", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preset, err := GetPreset(tt.presetName)

			if tt.shouldExist {
				if err != nil {
					t.Errorf("Expected preset to exist, got error: %v", err)
				}
				if preset == nil {
					t.Errorf("Expected preset config, got nil")
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for unknown preset, got nil")
				}
			}
		})
	}
}

func TestStandardPreset(t *testing.T) {
	preset, err := GetPreset("standard")
	if err != nil {
		t.Fatalf("Failed to get standard preset: %v", err)
	}

	// Standard preset should have epic patterns
	if len(preset.Epic.Folder) == 0 {
		t.Error("Standard preset should have epic folder patterns")
	}

	// Standard preset should have feature patterns
	if len(preset.Feature.Folder) == 0 {
		t.Error("Standard preset should have feature folder patterns")
	}

	// Standard preset should have task patterns
	if len(preset.Task.File) == 0 {
		t.Error("Standard preset should have task file patterns")
	}

	// Validate that all patterns are valid
	if err := ValidatePatternConfig(preset); err != nil {
		t.Errorf("Standard preset patterns should be valid: %v", err)
	}
}

func TestSpecialEpicsPreset(t *testing.T) {
	preset, err := GetPreset("special-epics")
	if err != nil {
		t.Fatalf("Failed to get special-epics preset: %v", err)
	}

	// Should have at least one epic folder pattern
	if len(preset.Epic.Folder) == 0 {
		t.Error("Special-epics preset should have epic folder patterns")
	}

	// Should match tech-debt, bugs, change-cards
	hasSpecialPattern := false
	for _, pattern := range preset.Epic.Folder {
		if contains(pattern, "tech-debt") || contains(pattern, "bugs") || contains(pattern, "change-cards") {
			hasSpecialPattern = true
			break
		}
	}

	if !hasSpecialPattern {
		t.Error("Special-epics preset should have patterns for tech-debt, bugs, or change-cards")
	}

	// Validate patterns
	if err := ValidatePatternConfig(preset); err != nil {
		t.Errorf("Special-epics preset patterns should be valid: %v", err)
	}
}

func TestNumericOnlyPreset(t *testing.T) {
	preset, err := GetPreset("numeric-only")
	if err != nil {
		t.Fatalf("Failed to get numeric-only preset: %v", err)
	}

	// Should have patterns for epic, feature, task
	if len(preset.Epic.Folder) == 0 {
		t.Error("Numeric-only preset should have epic folder patterns")
	}
	if len(preset.Feature.Folder) == 0 {
		t.Error("Numeric-only preset should have feature folder patterns")
	}
	if len(preset.Task.File) == 0 {
		t.Error("Numeric-only preset should have task file patterns")
	}

	// Validate patterns
	if err := ValidatePatternConfig(preset); err != nil {
		t.Errorf("Numeric-only preset patterns should be valid: %v", err)
	}
}

func TestLegacyPRPPreset(t *testing.T) {
	preset, err := GetPreset("legacy-prp")
	if err != nil {
		t.Fatalf("Failed to get legacy-prp preset: %v", err)
	}

	// Should have task file patterns for .prp.md
	if len(preset.Task.File) == 0 {
		t.Error("Legacy-prp preset should have task file patterns")
	}

	// Should have pattern matching .prp.md
	hasPRPPattern := false
	for _, pattern := range preset.Task.File {
		if contains(pattern, ".prp") {
			hasPRPPattern = true
			break
		}
	}

	if !hasPRPPattern {
		t.Error("Legacy-prp preset should have .prp.md pattern")
	}

	// Note: Legacy PRP patterns may not have all required capture groups
	// since they're meant to match legacy files in parent context
}

func TestPresetJSON(t *testing.T) {
	// Test that presets can be marshaled to JSON
	preset, err := GetPreset("standard")
	if err != nil {
		t.Fatalf("Failed to get standard preset: %v", err)
	}

	data, err := json.MarshalIndent(preset, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal preset to JSON: %v", err)
	}

	// Should be valid JSON
	var unmarshaled PatternConfig
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Errorf("Failed to unmarshal preset JSON: %v", err)
	}

	// Should have same number of patterns
	if len(unmarshaled.Epic.Folder) != len(preset.Epic.Folder) {
		t.Error("JSON round-trip changed epic folder pattern count")
	}
}

func TestListPresets(t *testing.T) {
	presets := ListPresets()

	// Should have exactly 4 presets
	if len(presets) != 4 {
		t.Errorf("Expected 4 presets, got %d", len(presets))
	}

	// Each preset should have name and description
	for _, preset := range presets {
		if preset.Name == "" {
			t.Error("Preset should have non-empty name")
		}
		if preset.Description == "" {
			t.Error("Preset should have non-empty description")
		}

		// Verify we can get the actual preset
		_, err := GetPreset(preset.Name)
		if err != nil {
			t.Errorf("Listed preset '%s' should be retrievable: %v", preset.Name, err)
		}
	}
}

func TestMergePatterns(t *testing.T) {
	// Create base config
	base := &PatternConfig{
		Epic: EntityPatterns{
			Folder: []string{"^E(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$"},
		},
	}

	// Create preset to merge
	preset := &PatternConfig{
		Epic: EntityPatterns{
			Folder: []string{"^(?P<epic_id>tech-debt)$"},
		},
	}

	// Merge patterns
	result := MergePatterns(base, preset)

	// Should have both patterns
	if len(result.Epic.Folder) != 2 {
		t.Errorf("Expected 2 epic folder patterns after merge, got %d", len(result.Epic.Folder))
	}

	// Original pattern should be preserved
	if result.Epic.Folder[0] != base.Epic.Folder[0] {
		t.Error("Original pattern should be preserved in merge")
	}

	// Preset pattern should be appended
	if result.Epic.Folder[1] != preset.Epic.Folder[0] {
		t.Error("Preset pattern should be appended in merge")
	}
}

func TestMergePatternsSkipsDuplicates(t *testing.T) {
	// Create base config
	base := &PatternConfig{
		Epic: EntityPatterns{
			Folder: []string{
				"^E(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$",
				"^(?P<epic_id>tech-debt)$",
			},
		},
	}

	// Create preset with duplicate pattern
	preset := &PatternConfig{
		Epic: EntityPatterns{
			Folder: []string{
				"^(?P<epic_id>tech-debt)$", // duplicate
				"^(?P<epic_id>bugs)$",      // new
			},
		},
	}

	// Merge patterns
	result := MergePatterns(base, preset)

	// Should have 3 patterns (not 4) - duplicate skipped
	if len(result.Epic.Folder) != 3 {
		t.Errorf("Expected 3 epic folder patterns after merge (duplicate skipped), got %d", len(result.Epic.Folder))
	}

	// Check that we have the right patterns
	hasOriginal1 := false
	hasTechDebt := false
	hasBugs := false

	for _, pattern := range result.Epic.Folder {
		if pattern == "^E(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$" {
			hasOriginal1 = true
		}
		if pattern == "^(?P<epic_id>tech-debt)$" {
			hasTechDebt = true
		}
		if pattern == "^(?P<epic_id>bugs)$" {
			hasBugs = true
		}
	}

	if !hasOriginal1 || !hasTechDebt || !hasBugs {
		t.Error("Merged patterns should contain original patterns and new patterns, skipping duplicates")
	}
}

func TestMergePatternsPreservesGeneration(t *testing.T) {
	// Create base config with generation format
	base := &PatternConfig{
		Epic: EntityPatterns{
			Generation: GenerationFormat{
				Format: "E{number:02d}-{slug}",
			},
		},
	}

	// Create preset without generation format
	preset := &PatternConfig{
		Epic: EntityPatterns{
			Folder: []string{"^(?P<epic_id>tech-debt)$"},
		},
	}

	// Merge patterns
	result := MergePatterns(base, preset)

	// Generation format should be preserved from base
	if result.Epic.Generation.Format != "E{number:02d}-{slug}" {
		t.Error("Generation format should be preserved from base config")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || (len(s) >= len(substr) && findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
