package patterns

import (
	"testing"
)

func TestGetDefaultPatterns(t *testing.T) {
	patterns := GetDefaultPatterns()

	if patterns == nil {
		t.Fatal("GetDefaultPatterns() returned nil")
	}

	// Test Epic patterns
	t.Run("Epic folder patterns", func(t *testing.T) {
		if len(patterns.Epic.Folder) == 0 {
			t.Error("Epic folder patterns should not be empty")
		}

		// Should include standard E##-slug pattern
		foundStandard := false
		for _, pattern := range patterns.Epic.Folder {
			if pattern == `^E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$` {
				foundStandard = true
				break
			}
		}
		if !foundStandard {
			t.Error("Epic folder patterns missing standard E##-slug pattern")
		}

		// Should include special epic types pattern
		foundSpecial := false
		for _, pattern := range patterns.Epic.Folder {
			if pattern == `^(?P<epic_id>tech-debt|bugs|change-cards)$` {
				foundSpecial = true
				break
			}
		}
		if !foundSpecial {
			t.Error("Epic folder patterns missing special epic types pattern")
		}
	})

	t.Run("Epic generation format", func(t *testing.T) {
		if patterns.Epic.Generation.Format == "" {
			t.Error("Epic generation format should not be empty")
		}
		if patterns.Epic.Generation.Format != "E{number:02d}-{slug}" {
			t.Errorf("Epic generation format incorrect: got %s", patterns.Epic.Generation.Format)
		}
	})

	// Test Feature patterns
	t.Run("Feature folder patterns", func(t *testing.T) {
		if len(patterns.Feature.Folder) == 0 {
			t.Error("Feature folder patterns should not be empty")
		}

		// Should include standard E##-F##-slug pattern
		foundStandard := false
		for _, pattern := range patterns.Feature.Folder {
			if pattern == `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$` {
				foundStandard = true
				break
			}
		}
		if !foundStandard {
			t.Error("Feature folder patterns missing standard E##-F##-slug pattern")
		}
	})

	t.Run("Feature file patterns", func(t *testing.T) {
		if len(patterns.Feature.File) == 0 {
			t.Error("Feature file patterns should not be empty")
		}

		// Should include prd.md as first pattern (priority)
		if patterns.Feature.File[0] != `^prd\.md$` {
			t.Errorf("First feature file pattern should be prd.md, got %s", patterns.Feature.File[0])
		}
	})

	t.Run("Feature generation format", func(t *testing.T) {
		if patterns.Feature.Generation.Format == "" {
			t.Error("Feature generation format should not be empty")
		}
		if patterns.Feature.Generation.Format != "E{epic:02d}-F{number:02d}-{slug}" {
			t.Errorf("Feature generation format incorrect: got %s", patterns.Feature.Generation.Format)
		}
	})

	// Test Task patterns
	t.Run("Task file patterns", func(t *testing.T) {
		if len(patterns.Task.File) == 0 {
			t.Error("Task file patterns should not be empty")
		}

		// Should include full task key pattern
		foundFullKey := false
		for _, pattern := range patterns.Task.File {
			if pattern == `^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3}).*\.md$` {
				foundFullKey = true
				break
			}
		}
		if !foundFullKey {
			t.Error("Task file patterns missing full key pattern T-E##-F##-###.md")
		}

		// Should include legacy PRP pattern
		foundPRP := false
		for _, pattern := range patterns.Task.File {
			if pattern == `^(?P<slug>.+)\.prp\.md$` {
				foundPRP = true
				break
			}
		}
		if !foundPRP {
			t.Error("Task file patterns missing legacy PRP pattern")
		}
	})

	t.Run("Task generation format", func(t *testing.T) {
		if patterns.Task.Generation.Format == "" {
			t.Error("Task generation format should not be empty")
		}
		if patterns.Task.Generation.Format != "T-E{epic:02d}-F{feature:02d}-{number:03d}.md" {
			t.Errorf("Task generation format incorrect: got %s", patterns.Task.Generation.Format)
		}
	})
}

func TestDefaultPatternsContainNamedCaptureGroups(t *testing.T) {
	patterns := GetDefaultPatterns()

	t.Run("Epic patterns contain required capture groups", func(t *testing.T) {
		// Epic patterns should include: epic_id, epic_slug, or number
		for i, pattern := range patterns.Epic.Folder {
			hasRequiredGroup := false
			requiredGroups := []string{"epic_id", "epic_slug", "number"}
			for _, group := range requiredGroups {
				if containsNamedGroup(pattern, group) {
					hasRequiredGroup = true
					break
				}
			}
			if !hasRequiredGroup {
				t.Errorf("Epic folder pattern[%d] missing required capture group (epic_id, epic_slug, or number): %s", i, pattern)
			}
		}
	})

	t.Run("Feature patterns contain required capture groups", func(t *testing.T) {
		// Feature patterns should include: (epic_id OR epic_num) AND (feature_id, feature_slug, OR number)
		for i, pattern := range patterns.Feature.Folder {
			hasEpicGroup := containsNamedGroup(pattern, "epic_id") || containsNamedGroup(pattern, "epic_num")
			hasFeatureGroup := containsNamedGroup(pattern, "feature_id") || containsNamedGroup(pattern, "feature_slug") || containsNamedGroup(pattern, "number")

			if !hasEpicGroup {
				t.Errorf("Feature folder pattern[%d] missing epic identifier (epic_id or epic_num): %s", i, pattern)
			}
			if !hasFeatureGroup {
				t.Errorf("Feature folder pattern[%d] missing feature identifier (feature_id, feature_slug, or number): %s", i, pattern)
			}
		}
	})

	t.Run("Task patterns contain required capture groups", func(t *testing.T) {
		// Task patterns should include: (epic_id OR epic_num) AND (feature_id OR feature_num) AND (task_id, number, OR task_slug)
		for i, pattern := range patterns.Task.File {
			hasEpicGroup := containsNamedGroup(pattern, "epic_id") || containsNamedGroup(pattern, "epic_num")
			hasFeatureGroup := containsNamedGroup(pattern, "feature_id") || containsNamedGroup(pattern, "feature_num")
			hasTaskGroup := containsNamedGroup(pattern, "task_id") || containsNamedGroup(pattern, "number") || containsNamedGroup(pattern, "task_slug") || containsNamedGroup(pattern, "slug")

			// Full key pattern should have all groups
			if pattern == `^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3}).*\.md$` {
				if !hasEpicGroup || !hasFeatureGroup || !hasTaskGroup {
					t.Errorf("Full key task pattern[%d] missing required capture groups: %s", i, pattern)
				}
			}
		}
	})
}

// Helper function to check if pattern contains a named capture group
func containsNamedGroup(pattern, groupName string) bool {
	searchStr := "?P<" + groupName + ">"
	for i := 0; i < len(pattern)-len(searchStr); i++ {
		if pattern[i:i+len(searchStr)] == searchStr {
			return true
		}
	}
	return false
}
