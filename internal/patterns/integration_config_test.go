package patterns

import (
	"os"
	"testing"
)

// TestIntegration_LoadActualSharkConfig tests loading the actual .sharkconfig.json from the project root
func TestIntegration_LoadActualSharkConfig(t *testing.T) {
	configPath := ".sharkconfig.json"

	// Skip test if config file doesn't exist (common in CI without project root)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skipf("Config file %s not found, skipping integration test", configPath)
	}

	// Load registry from actual config
	registry, err := LoadPatternRegistryFromFile(configPath, false)
	if err != nil {
		t.Fatalf("Failed to load registry from .sharkconfig.json: %v", err)
	}

	t.Run("Epic folder patterns work", func(t *testing.T) {
		result := registry.MatchEpicFolder("E04-task-management")
		if !result.Matched {
			t.Error("Failed to match epic folder E04-task-management")
		}
		if result.CaptureGroups["number"] != "04" {
			t.Errorf("Expected number=04, got %s", result.CaptureGroups["number"])
		}
		if result.CaptureGroups["slug"] != "task-management" {
			t.Errorf("Expected slug=task-management, got %s", result.CaptureGroups["slug"])
		}
	})

	t.Run("Task file standard format works", func(t *testing.T) {
		result := registry.MatchTaskFile("T-E04-F05-001.md")
		if !result.Matched {
			t.Error("Failed to match standard task file")
		}
		if result.CaptureGroups["epic_num"] != "04" {
			t.Errorf("Expected epic_num=04, got %s", result.CaptureGroups["epic_num"])
		}
		if result.CaptureGroups["feature_num"] != "05" {
			t.Errorf("Expected feature_num=05, got %s", result.CaptureGroups["feature_num"])
		}
		if result.CaptureGroups["number"] != "001" {
			t.Errorf("Expected number=001, got %s", result.CaptureGroups["number"])
		}
	})

	t.Run("Task file numbered format works", func(t *testing.T) {
		result := registry.MatchTaskFile("042-implement-feature.md")
		if !result.Matched {
			t.Error("Failed to match numbered task file")
		}
		if result.CaptureGroups["number"] != "042" {
			t.Errorf("Expected number=042, got %s", result.CaptureGroups["number"])
		}
		if result.CaptureGroups["slug"] != "implement-feature" {
			t.Errorf("Expected slug=implement-feature, got %s", result.CaptureGroups["slug"])
		}
	})

	t.Run("Task file PRP format works", func(t *testing.T) {
		result := registry.MatchTaskFile("auth-middleware.prp.md")
		if !result.Matched {
			t.Error("Failed to match PRP task file")
		}
		if result.CaptureGroups["slug"] != "auth-middleware" {
			t.Errorf("Expected slug=auth-middleware, got %s", result.CaptureGroups["slug"])
		}
	})

	t.Run("Feature folder patterns work", func(t *testing.T) {
		result := registry.MatchFeatureFolder("E06-F03-task-recognition-import")
		if !result.Matched {
			t.Error("Failed to match feature folder")
		}
		if result.CaptureGroups["epic_num"] != "06" {
			t.Errorf("Expected epic_num=06, got %s", result.CaptureGroups["epic_num"])
		}
		if result.CaptureGroups["number"] != "03" {
			t.Errorf("Expected number=03, got %s", result.CaptureGroups["number"])
		}
		if result.CaptureGroups["slug"] != "task-recognition-import" {
			t.Errorf("Expected slug=task-recognition-import, got %s", result.CaptureGroups["slug"])
		}
	})

	t.Run("Special epic patterns work", func(t *testing.T) {
		result := registry.MatchEpicFolder("tech-debt")
		if !result.Matched {
			t.Error("Failed to match tech-debt epic")
		}
		if result.CaptureGroups["epic_id"] != "tech-debt" {
			t.Errorf("Expected epic_id=tech-debt, got %s", result.CaptureGroups["epic_id"])
		}
	})
}
