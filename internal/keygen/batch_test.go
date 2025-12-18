package keygen_test

import (
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

// TestBatchProcessing_NoDuplicateKeys tests the critical bug fix:
// Multiple PRP files in the same feature should receive unique sequential keys
func TestBatchProcessing_NoDuplicateKeys(t *testing.T) {
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

	// Create existing tasks 001-003 in database
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

	// Create multiple PRP files without task_key (simulating batch import)
	prpFiles := []string{
		filepath.Join(tasksDir, "add-caching.prp.md"),
		filepath.Join(tasksDir, "add-monitoring.prp.md"),
		filepath.Join(tasksDir, "add-logging.prp.md"),
		filepath.Join(tasksDir, "add-metrics.prp.md"),
	}

	for i, prpFile := range prpFiles {
		content := fmt.Sprintf(`---
description: Task %d description
status: todo
---

# Task %d
`, i+1, i+1)
		if err := os.WriteFile(prpFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create PRP file: %v", err)
		}
	}

	// Create generator
	generator := keygen.NewTaskKeyGenerator(taskRepo, featureRepo, epicRepo, tmpDir)

	// CRITICAL TEST: Process all files in sequence without database insertion
	// This simulates the bug scenario where keys are written to files but NOT to database
	expectedKeys := []string{
		"T-E04-F02-004",
		"T-E04-F02-005",
		"T-E04-F02-006",
		"T-E04-F02-007",
	}

	generatedKeys := make([]string, len(prpFiles))

	for i, prpFile := range prpFiles {
		result, err := generator.GenerateKeyForFile(ctx, prpFile)
		if err != nil {
			t.Fatalf("GenerateKeyForFile() failed for file %d: %v", i, err)
		}
		generatedKeys[i] = result.TaskKey

		t.Logf("File %d: %s -> %s", i+1, filepath.Base(prpFile), result.TaskKey)
	}

	// Verify all keys are unique and sequential
	for i, key := range generatedKeys {
		if key != expectedKeys[i] {
			t.Errorf("File %d: got key %s, want %s", i+1, key, expectedKeys[i])
		}
	}

	// Verify no duplicates
	keySet := make(map[string]bool)
	for i, key := range generatedKeys {
		if keySet[key] {
			t.Errorf("DUPLICATE KEY DETECTED: File %d has duplicate key %s", i+1, key)
		}
		keySet[key] = true
	}

	// Verify keys were written to files
	writer := keygen.NewFrontmatterWriter()
	for i, prpFile := range prpFiles {
		hasKey, taskKey, err := writer.HasTaskKey(prpFile)
		if err != nil {
			t.Fatalf("HasTaskKey() error for file %d: %v", i, err)
		}
		if !hasKey {
			t.Errorf("File %d: task_key not written", i+1)
		}
		if taskKey != expectedKeys[i] {
			t.Errorf("File %d: written key %s, want %s", i+1, taskKey, expectedKeys[i])
		}
	}

	t.Log("SUCCESS: All files received unique sequential task keys")
}

// TestBatchProcessing_WithExistingKeys tests that files with existing keys
// don't interfere with sequence generation for new files
func TestBatchProcessing_WithExistingKeys(t *testing.T) {
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

	// Create test epic and feature
	testDesc := "Test epic"
	epic := &models.Epic{
		Key:         "E04",
		Title:       "Test Epic",
		Description: &testDesc,
		Status:      models.EpicStatusActive,
		Priority:    models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	testFeatureDesc := "Test feature"
	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E04-F02",
		Title:       "Test Feature",
		Description: &testFeatureDesc,
		Status:      models.FeatureStatusActive,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	// Create existing task 001 in database
	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E04-F02-001",
		Title:     "Existing Task",
		Status:    models.TaskStatusTodo,
		Priority:  2,
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// Set up file system
	docsRoot := filepath.Join(tmpDir, "docs")
	tasksDir := filepath.Join(docsRoot, "plan", "E04-epic", "E04-F02-feature", "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}

	// Create files: one with existing key, two without
	files := []struct {
		name        string
		hasKey      bool
		existingKey string
		expectedKey string
	}{
		{"task-001.prp.md", true, "T-E04-F02-001", "T-E04-F02-001"}, // existing
		{"task-002.prp.md", false, "", "T-E04-F02-002"},             // new
		{"task-003.prp.md", false, "", "T-E04-F02-003"},             // new
	}

	for _, f := range files {
		filePath := filepath.Join(tasksDir, f.name)
		var content string
		if f.hasKey {
			content = fmt.Sprintf("---\ntask_key: %s\n---\n\n# Task", f.existingKey)
		} else {
			content = "---\n---\n\n# Task"
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", f.name, err)
		}
	}

	// Process all files
	generator := keygen.NewTaskKeyGenerator(taskRepo, featureRepo, epicRepo, tmpDir)

	for i, f := range files {
		filePath := filepath.Join(tasksDir, f.name)
		result, err := generator.GenerateKeyForFile(ctx, filePath)
		if err != nil {
			t.Fatalf("GenerateKeyForFile() failed for %s: %v", f.name, err)
		}

		if result.TaskKey != f.expectedKey {
			t.Errorf("File %d (%s): got key %s, want %s",
				i+1, f.name, result.TaskKey, f.expectedKey)
		}

		t.Logf("File %s: %s (existing: %v)", f.name, result.TaskKey, f.hasKey)
	}
}
