package keygen_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/keygen"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	_ "github.com/mattn/go-sqlite3"
)

// TestEndToEndKeyGeneration tests the complete workflow from file to database
func TestEndToEndKeyGeneration(t *testing.T) {
	// Set up test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer database.Close()

	// Create repositories
	dbWrapper := repository.NewDB(database)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	epicRepo := repository.NewEpicRepository(dbWrapper)

	ctx := context.Background()

	// Create test epic
	testDesc := "Test epic"
	epic := &models.Epic{
		Key:         "E04",
		Title:       "Task Management CLI Core",
		Description: &testDesc,
		Status:      models.EpicStatusActive,
		Priority:    models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	// Create test feature
	testFeatureDesc := "Test feature"
	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E04-F02",
		Title:       "CLI Infrastructure",
		Description: &testFeatureDesc,
		Status:      models.FeatureStatusActive,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	// Create some existing tasks to test sequence generation
	for i := 1; i <= 3; i++ {
		task := &models.Task{
			FeatureID:   feature.ID,
			Key:         fmt.Sprintf("T-E04-F02-%03d", i),
			Title:       fmt.Sprintf("Task %d", i),
			Description: nil,
			Status:      models.TaskStatusTodo,
			AgentType:   nil,
			Priority:    2,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create test task: %v", err)
		}
	}

	// Set up file system structure
	docsRoot := filepath.Join(tmpDir, "docs")
	epicDir := filepath.Join(docsRoot, "plan", "E04-task-mgmt-cli-core")
	featureDir := filepath.Join(epicDir, "E04-F02-cli-infrastructure")
	tasksDir := filepath.Join(featureDir, "tasks")

	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}

	// Create a PRP file without task_key
	prpFile := filepath.Join(tasksDir, "implement-caching.prp.md")
	prpContent := `---
description: Add caching layer for API responses
status: todo
priority: 2
---

# Implement Caching

This task involves implementing a caching layer to improve API response times.

## Requirements

- Use in-memory cache for frequently accessed data
- Implement TTL (time-to-live) for cache entries
- Add cache invalidation on data updates

## Acceptance Criteria

- [ ] Cache reduces API response time by 50%
- [ ] Cache hit rate above 80% for read operations
- [ ] Cache size limited to 100MB
`

	if err := os.WriteFile(prpFile, []byte(prpContent), 0644); err != nil {
		t.Fatalf("Failed to create PRP file: %v", err)
	}

	// Create generator
	generator := keygen.NewTaskKeyGenerator(taskRepo, featureRepo, epicRepo, tmpDir)

	// Test 1: Generate key for file
	t.Run("generate key for PRP file", func(t *testing.T) {
		result, err := generator.GenerateKeyForFile(ctx, prpFile)
		if err != nil {
			t.Fatalf("GenerateKeyForFile() error = %v", err)
		}

		// Verify generated key is T-E04-F02-004 (next after 003)
		expectedKey := "T-E04-F02-004"
		if result.TaskKey != expectedKey {
			t.Errorf("Generated TaskKey = %v, want %v", result.TaskKey, expectedKey)
		}

		if result.EpicKey != "E04" {
			t.Errorf("EpicKey = %v, want E04", result.EpicKey)
		}

		if result.FeatureKey != "E04-F02" {
			t.Errorf("FeatureKey = %v, want E04-F02", result.FeatureKey)
		}

		if result.FeatureID != feature.ID {
			t.Errorf("FeatureID = %v, want %v", result.FeatureID, feature.ID)
		}

		if !result.WrittenToFile {
			t.Errorf("WrittenToFile = false, want true")
		}

		// Verify key was written to file
		writer := keygen.NewFrontmatterWriter()
		hasKey, taskKey, err := writer.HasTaskKey(prpFile)
		if err != nil {
			t.Fatalf("HasTaskKey() error = %v", err)
		}

		if !hasKey {
			t.Errorf("Task key not written to file")
		}

		if taskKey != expectedKey {
			t.Errorf("Written task key = %v, want %v", taskKey, expectedKey)
		}

		// Verify frontmatter was preserved
		fm, err := writer.ReadFrontmatter(prpFile)
		if err != nil {
			t.Fatalf("ReadFrontmatter() error = %v", err)
		}

		if fm["description"] != "Add caching layer for API responses" {
			t.Errorf("Description not preserved in frontmatter")
		}

		if fm["status"] != "todo" {
			t.Errorf("Status not preserved in frontmatter")
		}

		// Read file content to verify markdown body was preserved
		updatedContent, err := os.ReadFile(prpFile)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}

		if !bytes.Contains(updatedContent, []byte("# Implement Caching")) {
			t.Errorf("Markdown heading not preserved")
		}

		if !bytes.Contains(updatedContent, []byte("## Acceptance Criteria")) {
			t.Errorf("Markdown content not preserved")
		}
	})

	// Test 2: Verify idempotency - running again should return existing key
	t.Run("idempotency - second run returns existing key", func(t *testing.T) {
		result, err := generator.GenerateKeyForFile(ctx, prpFile)
		if err != nil {
			t.Fatalf("GenerateKeyForFile() error = %v", err)
		}

		// Should return existing key T-E04-F02-004
		if result.TaskKey != "T-E04-F02-004" {
			t.Errorf("Second run generated different key: %v", result.TaskKey)
		}

		if result.WrittenToFile {
			t.Errorf("Second run claimed to write key (should have found existing)")
		}
	})

	// Test 3: Create another PRP file and verify sequence increments
	t.Run("sequence increments for second file", func(t *testing.T) {
		prpFile2 := filepath.Join(tasksDir, "add-monitoring.prp.md")
		prpContent2 := `---
description: Add monitoring and metrics
---

# Monitoring
`

		if err := os.WriteFile(prpFile2, []byte(prpContent2), 0644); err != nil {
			t.Fatalf("Failed to create second PRP file: %v", err)
		}

		result, err := generator.GenerateKeyForFile(ctx, prpFile2)
		if err != nil {
			t.Fatalf("GenerateKeyForFile() error = %v", err)
		}

		// Should generate T-E04-F02-005
		expectedKey := "T-E04-F02-005"
		if result.TaskKey != expectedKey {
			t.Errorf("Generated TaskKey = %v, want %v", result.TaskKey, expectedKey)
		}

		// Verify it was written
		writer := keygen.NewFrontmatterWriter()
		hasKey, taskKey, err := writer.HasTaskKey(prpFile2)
		if err != nil {
			t.Fatalf("HasTaskKey() error = %v", err)
		}

		if !hasKey || taskKey != expectedKey {
			t.Errorf("Task key not correctly written to second file")
		}
	})

	// Test 4: Test error handling for orphaned file
	t.Run("orphaned file detection", func(t *testing.T) {
		// Create file in non-existent feature folder
		orphanDir := filepath.Join(epicDir, "E04-F99-nonexistent", "tasks")
		if err := os.MkdirAll(orphanDir, 0755); err != nil {
			t.Fatalf("Failed to create orphan directory: %v", err)
		}

		orphanFile := filepath.Join(orphanDir, "orphan.prp.md")
		if err := os.WriteFile(orphanFile, []byte("---\n---\n\n# Orphan"), 0644); err != nil {
			t.Fatalf("Failed to create orphan file: %v", err)
		}

		result, err := generator.GenerateKeyForFile(ctx, orphanFile)
		if err == nil {
			t.Errorf("Expected error for orphaned file, got nil")
		}

		if result != nil {
			t.Errorf("Expected nil result for orphaned file, got %v", result)
		}

		// Verify error message is helpful
		if err != nil {
			errMsg := err.Error()
			if !contains(errMsg, "orphaned") && !contains(errMsg, "not found") {
				t.Errorf("Error message not helpful: %v", errMsg)
			}
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
