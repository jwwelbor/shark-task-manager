package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSyncWithStandardTaskFiles tests syncing standard T-E##-F##-###.md files
func TestSyncWithStandardTaskFiles(t *testing.T) {
	// Create temp directory with test fixtures
	tempDir := t.TempDir()

	// Create epic and feature folder structure
	featureDir := filepath.Join(tempDir, "E04-epic", "E04-F07-feature", "tasks")
	require.NoError(t, os.MkdirAll(featureDir, 0755))

	// Create 10 standard task files
	for i := 1; i <= 10; i++ {
		taskFile := filepath.Join(featureDir, fmt.Sprintf("T-E04-F07-%03d-task-%d.md", i, i))
		content := fmt.Sprintf(`---
task_key: T-E04-F07-%03d
title: Task %d
status: todo
---

# Task: Task %d

This is task number %d.
`, i, i, i, i)
		require.NoError(t, os.WriteFile(taskFile, []byte(content), 0644))
	}

	// TODO: Initialize database schema and seed with epic/feature
	// This requires database setup which is beyond scope of this test
	// For now, we'll test the pattern matching logic separately

	t.Skip("Database setup required for full integration test")
}

// TestSyncWithNumberedTaskFiles tests syncing numbered ##-name.md files
func TestSyncWithNumberedTaskFiles(t *testing.T) {
	// Create temp directory with test fixtures
	tempDir := t.TempDir()

	// Create epic and feature folder structure
	featureDir := filepath.Join(tempDir, "E04-epic", "E04-F08-feature", "tasks")
	require.NoError(t, os.MkdirAll(featureDir, 0755))

	// Create 5 numbered task files (no embedded task_key)
	taskNames := []string{
		"setup-environment",
		"install-dependencies",
		"configure-database",
		"run-migrations",
		"seed-test-data",
	}

	for i, name := range taskNames {
		taskFile := filepath.Join(featureDir, fmt.Sprintf("%02d-%s.md", i+1, name))
		content := fmt.Sprintf(`---
title: %s
status: todo
---

# %s

Description for task %s.
`, name, name, name)
		require.NoError(t, os.WriteFile(taskFile, []byte(content), 0644))
	}

	t.Skip("Database setup required for full integration test")
}

// TestSyncWithPRPFiles tests syncing name.prp.md files
func TestSyncWithPRPFiles(t *testing.T) {
	// Create temp directory with test fixtures
	tempDir := t.TempDir()

	// Create epic and feature folder structure
	featureDir := filepath.Join(tempDir, "E05-epic", "E05-F01-feature", "prps")
	require.NoError(t, os.MkdirAll(featureDir, 0755))

	// Create 3 PRP files
	prpNames := []string{
		"user-authentication",
		"password-reset-flow",
		"session-management",
	}

	for _, name := range prpNames {
		prpFile := filepath.Join(featureDir, name+".prp.md")
		content := fmt.Sprintf(`---
status: todo
---

# PRP: %s

This is the product requirement prompt for %s.

## Requirements

- Requirement 1
- Requirement 2
`, name, name)
		require.NoError(t, os.WriteFile(prpFile, []byte(content), 0644))
	}

	t.Skip("Database setup required for full integration test")
}

// TestSyncWithMixedFormats tests syncing a mix of standard, numbered, and PRP files
func TestSyncWithMixedFormats(t *testing.T) {
	// Create temp directory with test fixtures
	tempDir := t.TempDir()

	// Create epic and feature folder structure
	featureDir := filepath.Join(tempDir, "E06-epic", "E06-F01-feature", "tasks")
	require.NoError(t, os.MkdirAll(featureDir, 0755))

	// Create 2 standard task files
	standardFiles := []string{
		"T-E06-F01-001-standard-task-one.md",
		"T-E06-F01-002-standard-task-two.md",
	}
	for _, filename := range standardFiles {
		filepath := filepath.Join(featureDir, filename)
		content := `---
task_key: ` + filename[:14] + `
title: Standard Task
status: todo
---

# Standard Task
`
		require.NoError(t, os.WriteFile(filepath, []byte(content), 0644))
	}

	// Create 2 numbered files
	numberedFiles := []string{
		"01-numbered-task-one.md",
		"02-numbered-task-two.md",
	}
	for _, filename := range numberedFiles {
		filepath := filepath.Join(featureDir, filename)
		content := `---
title: Numbered Task
status: todo
---

# Numbered Task
`
		require.NoError(t, os.WriteFile(filepath, []byte(content), 0644))
	}

	// Create 1 PRP file
	prpFile := filepath.Join(featureDir, "authentication.prp.md")
	content := `---
status: todo
---

# PRP: Authentication

Authentication requirements.
`
	require.NoError(t, os.WriteFile(prpFile, []byte(content), 0644))

	t.Skip("Database setup required for full integration test")
}

// TestPatternMatchingLogic tests the pattern matching without database
func TestPatternMatchingLogic(t *testing.T) {
	// Create temp config directory
	tempDir := t.TempDir()

	// Create .sharkconfig.json with test patterns
	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	configContent := `{
  "patterns": {
    "task": {
      "file": [
        "^T-E(?P<epic_num>\\d{2})-F(?P<feature_num>\\d{2})-(?P<number>\\d{3}).*\\.md$",
        "^(?P<number>\\d{2,3})-(?P<slug>.+)\\.md$",
        "^(?P<slug>.+)\\.prp\\.md$"
      ]
    }
  }
}`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Change to temp directory
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldDir) }()
	require.NoError(t, os.Chdir(tempDir))

	// Load pattern registry
	registry, err := loadPatternRegistry(configPath)
	require.NoError(t, err)
	require.NotNil(t, registry)

	t.Run("standard task pattern", func(t *testing.T) {
		result := registry.MatchTaskFile("T-E04-F07-001-my-task.md")
		assert.True(t, result.Matched, "Should match standard task pattern")
		assert.Equal(t, "04", result.CaptureGroups["epic_num"])
		assert.Equal(t, "07", result.CaptureGroups["feature_num"])
		assert.Equal(t, "001", result.CaptureGroups["number"])
	})

	t.Run("numbered task pattern", func(t *testing.T) {
		result := registry.MatchTaskFile("01-setup-environment.md")
		assert.True(t, result.Matched, "Should match numbered task pattern")
		assert.Equal(t, "01", result.CaptureGroups["number"])
		assert.Equal(t, "setup-environment", result.CaptureGroups["slug"])
	})

	t.Run("PRP pattern", func(t *testing.T) {
		result := registry.MatchTaskFile("user-authentication.prp.md")
		assert.True(t, result.Matched, "Should match PRP pattern")
		assert.Equal(t, "user-authentication", result.CaptureGroups["slug"])
	})

	t.Run("no match", func(t *testing.T) {
		result := registry.MatchTaskFile("README.md")
		assert.False(t, result.Matched, "Should not match README.md")
	})
}

// TestMetadataExtraction tests the metadata extractor with different patterns
func TestMetadataExtraction(t *testing.T) {
	// This test validates the parser.ExtractMetadata integration
	t.Run("standard task with frontmatter title", func(t *testing.T) {
		content := `---
task_key: T-E04-F07-001
title: My Task Title
status: todo
---

# Task: Different Title

Task description here.
`
		// Test that frontmatter title takes priority
		// TODO: Implement actual test once pattern registry is available
		_ = content
		t.Skip("Pattern registry setup required")
	})

	t.Run("numbered task without frontmatter", func(t *testing.T) {
		content := `# Setup Environment

This task sets up the development environment.
`
		// Test that title is extracted from filename or H1
		_ = content
		t.Skip("Pattern registry setup required")
	})

	t.Run("PRP file with description", func(t *testing.T) {
		content := `---
status: todo
---

# PRP: User Authentication

Implement user authentication with email and password.

This is the first paragraph of description.
`
		// Test description extraction
		_ = content
		t.Skip("Pattern registry setup required")
	})
}

// TestKeyGeneration tests the key generator integration
func TestKeyGeneration(t *testing.T) {
	t.Run("generate key for PRP file", func(t *testing.T) {
		// Test that key is generated and written to frontmatter
		t.Skip("Database setup required")
	})

	t.Run("generate key for numbered file", func(t *testing.T) {
		// Test that key is generated for numbered files
		t.Skip("Database setup required")
	})

	t.Run("skip generation for standard tasks", func(t *testing.T) {
		// Test that standard tasks with embedded keys don't trigger generation
		t.Skip("Database setup required")
	})
}

// TestSyncReportStatistics tests that the sync report includes pattern statistics
func TestSyncReportStatistics(t *testing.T) {
	t.Run("pattern match counts", func(t *testing.T) {
		// Verify that report.PatternMatches contains accurate counts
		t.Skip("Full sync test required")
	})

	t.Run("keys generated count", func(t *testing.T) {
		// Verify that report.KeysGenerated is incremented
		t.Skip("Full sync test required")
	})
}

// TestErrorRecovery tests file-level error recovery
func TestErrorRecovery(t *testing.T) {
	t.Run("invalid frontmatter doesn't abort sync", func(t *testing.T) {
		// Create files with invalid YAML
		// Verify other files are still imported
		t.Skip("Full sync test required")
	})

	t.Run("orphaned PRP file doesn't abort sync", func(t *testing.T) {
		// Create PRP file in non-existent feature folder
		// Verify error is logged but other files proceed
		t.Skip("Full sync test required")
	})

	t.Run("pattern mismatch is logged as warning", func(t *testing.T) {
		// Create file that doesn't match any pattern
		// Verify warning is logged but sync continues
		t.Skip("Full sync test required")
	})
}

// TestBackwardCompatibility tests that existing E04 task files still work
func TestBackwardCompatibility(t *testing.T) {
	t.Run("existing T-E##-F##-### files import correctly", func(t *testing.T) {
		// Use actual E04 task files from the project
		// Verify they import without errors
		t.Skip("Requires access to actual project files")
	})
}

// BenchmarkSyncPerformance benchmarks sync performance with 1000 task files
func BenchmarkSyncPerformance(b *testing.B) {
	// Create temp directory with 1000 task files
	tempDir := b.TempDir()

	// Create feature directory
	featureDir := filepath.Join(tempDir, "E99-epic", "E99-F99-feature", "tasks")
	require.NoError(b, os.MkdirAll(featureDir, 0755))

	// Create 1000 task files
	for i := 1; i <= 1000; i++ {
		taskFile := filepath.Join(featureDir, fmt.Sprintf("T-E99-F99-%03d.md", i))
		content := fmt.Sprintf(`---
task_key: T-E99-F99-%03d
title: Task %d
status: todo
---

# Task %d
`, i, i, i)
		require.NoError(b, os.WriteFile(taskFile, []byte(content), 0644))
	}

	b.Skip("Database setup required for benchmark")

	// TODO: Initialize database and run sync
	// Verify completion time is < 5 seconds
}
