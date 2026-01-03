package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/pathresolver"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// TestTaskGetPathDisplay tests path and filename display for tasks with all combinations
func TestTaskGetPathDisplay(t *testing.T) {
	// Save and restore working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	// Setup test database
	tmpDB := t.TempDir() + "/test-task-path.db"
	database, err := db.InitDB(tmpDB)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Create test project structure
	projectRoot := t.TempDir()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}

	// Create base epic for all tests
	epic := &models.Epic{
		Key:      "E99",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err = epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	// Test cases
	// Note: Slugs are auto-generated from titles using GenerateSlug:
	// - "Test Feature default path + default filename" -> "test-feature-default-path-default-filename"
	// - Task filenames use just {task.Key}.md (no slug in filename)
	tests := []struct {
		name               string
		customTaskFilename *string
		expectedPath       string
		expectedFilename   string
	}{
		{
			name:               "default path + default filename",
			customTaskFilename: nil,
			// Path includes slugged epic and feature folders
			// Epic slug: "test-epic", Feature slug: "test-feature-default-path-default-filename"
			expectedPath:     "docs/plan/E99-test-epic/E99-F01-test-feature-default-path-default-filename/tasks/",
			expectedFilename: "T-E99-F01-001.md", // Task filename is just key, no slug
		},
		{
			name:               "default path + custom filename",
			customTaskFilename: stringPtr("custom-task.md"), // Relative path, not absolute
			expectedPath:       "./",
			expectedFilename:   "custom-task.md",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create feature
			featureKey := "E99-F" + getFeatureNum(i+1)
			feature := &models.Feature{
				EpicID: epic.ID,
				Key:    featureKey,
				Title:  "Test Feature " + tt.name,
				Status: models.FeatureStatusActive,
			}
			err = featureRepo.Create(ctx, feature)
			if err != nil {
				t.Fatalf("Failed to create feature: %v", err)
			}

			// Create task with custom filename if specified
			taskKey := "T-" + featureKey + "-001"
			task := &models.Task{
				FeatureID: feature.ID,
				Key:       taskKey,
				Title:     "Test Task " + tt.name,
				Status:    models.TaskStatusTodo,
				Priority:  5,
			}
			if tt.customTaskFilename != nil {
				task.FilePath = tt.customTaskFilename
			}
			err = taskRepo.Create(ctx, task)
			if err != nil {
				t.Fatalf("Failed to create task: %v", err)
			}

			// Resolve the path using PathResolver
			pathResolver := pathresolver.NewPathResolver(epicRepo, featureRepo, taskRepo, projectRoot)
			resolvedPath, err := pathResolver.ResolveTaskPath(ctx, task.Key)
			if err != nil {
				t.Fatalf("Failed to resolve task path: %v", err)
			}

			// Get relative path
			relPath, err := filepath.Rel(projectRoot, resolvedPath)
			if err != nil {
				t.Fatalf("Failed to get relative path: %v", err)
			}

			// Extract directory and filename
			dirPath := filepath.Dir(relPath) + "/"
			filename := filepath.Base(relPath)

			// Verify path
			if dirPath != tt.expectedPath {
				t.Errorf("Path mismatch:\n  got:  %q\n  want: %q", dirPath, tt.expectedPath)
			}

			// Verify filename
			if filename != tt.expectedFilename {
				t.Errorf("Filename mismatch:\n  got:  %q\n  want: %q", filename, tt.expectedFilename)
			}
		})
	}
}

// TestEpicGetPathDisplay tests path and filename display for epics with all combinations
func TestEpicGetPathDisplay(t *testing.T) {
	// Save and restore working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	// Setup test database
	tmpDB := t.TempDir() + "/test-epic-path.db"
	database, err := db.InitDB(tmpDB)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Create test project structure
	projectRoot := t.TempDir()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}

	// Test cases
	// Note: Slugs are auto-generated from titles using GenerateSlug:
	// - "Test Epic default path + default filename" -> "test-epic-default-path-default-filename"
	// - When FilePath is set, it takes precedence and is used directly
	tests := []struct {
		name             string
		customFilename   *string
		expectedPath     string
		expectedFilename string
	}{
		{
			name:           "default path + default filename",
			customFilename: nil,
			// Default path includes slugged epic folder: {key}-{slug}
			expectedPath:     "docs/plan/E98-test-epic-default-path-default-filename/",
			expectedFilename: "epic.md",
		},
		{
			name:           "default path + custom filename",
			customFilename: stringPtr("docs/plan/E97/custom-epic.md"), // Relative path, not absolute
			// When FilePath is set, it takes precedence
			expectedPath:     "docs/plan/E97/",
			expectedFilename: "custom-epic.md",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create epic
			epicKey := "E" + getEpicNum(98-i)
			epic := &models.Epic{
				Key:      epicKey,
				Title:    "Test Epic " + tt.name,
				Status:   models.EpicStatusActive,
				Priority: models.PriorityMedium,
			}
			if tt.customFilename != nil {
				epic.FilePath = tt.customFilename
			}
			err = epicRepo.Create(ctx, epic)
			if err != nil {
				t.Fatalf("Failed to create epic: %v", err)
			}

			// Resolve the path using PathResolver
			pathResolver := pathresolver.NewPathResolver(epicRepo, featureRepo, taskRepo, projectRoot)
			resolvedPath, err := pathResolver.ResolveEpicPath(ctx, epic.Key)
			if err != nil {
				t.Fatalf("Failed to resolve epic path: %v", err)
			}

			// Get relative path
			relPath, err := filepath.Rel(projectRoot, resolvedPath)
			if err != nil {
				t.Fatalf("Failed to get relative path: %v", err)
			}

			// Extract directory and filename
			dirPath := filepath.Dir(relPath) + "/"
			filename := filepath.Base(relPath)

			// Verify path
			if dirPath != tt.expectedPath {
				t.Errorf("Path mismatch:\n  got:  %q\n  want: %q", dirPath, tt.expectedPath)
			}

			// Verify filename
			if filename != tt.expectedFilename {
				t.Errorf("Filename mismatch:\n  got:  %q\n  want: %q", filename, tt.expectedFilename)
			}
		})
	}
}

// TestFeatureGetPathDisplay tests path and filename display for features with all combinations
func TestFeatureGetPathDisplay(t *testing.T) {
	// Save and restore working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	// Setup test database
	tmpDB := t.TempDir() + "/test-feature-path.db"
	database, err := db.InitDB(tmpDB)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Create test project structure
	projectRoot := t.TempDir()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}

	// Create base epic for all tests
	epic := &models.Epic{
		Key:      "E96",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err = epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	// Test cases
	// Note: Slugs are auto-generated from titles using GenerateSlug:
	// - "Test Feature default path + default filename" -> "test-feature-default-path-default-filename"
	// - Default feature filename is prd.md (not feature.md)
	// - When FilePath is set, it takes precedence
	tests := []struct {
		name             string
		customFilename   *string
		expectedPath     string
		expectedFilename string
	}{
		{
			name:           "default path + default filename",
			customFilename: nil,
			// Path includes slugged epic and feature folders
			// Epic slug: "test-epic", Feature slug: "test-feature-default-path-default-filename"
			expectedPath:     "docs/plan/E96-test-epic/E96-F01-test-feature-default-path-default-filename/",
			expectedFilename: "prd.md", // Default feature filename is prd.md
		},
		{
			name:           "default path + custom filename",
			customFilename: stringPtr("docs/plan/E96/E96-F02/spec.md"), // Relative path, not absolute
			// When FilePath is set, it takes precedence
			expectedPath:     "docs/plan/E96/E96-F02/",
			expectedFilename: "spec.md",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create feature
			featureKey := "E96-F" + getFeatureNum(i+1)
			feature := &models.Feature{
				EpicID: epic.ID,
				Key:    featureKey,
				Title:  "Test Feature " + tt.name,
				Status: models.FeatureStatusActive,
			}
			if tt.customFilename != nil {
				feature.FilePath = tt.customFilename
			}
			err = featureRepo.Create(ctx, feature)
			if err != nil {
				t.Fatalf("Failed to create feature: %v", err)
			}

			// Resolve the path using PathResolver
			pathResolver := pathresolver.NewPathResolver(epicRepo, featureRepo, taskRepo, projectRoot)
			resolvedPath, err := pathResolver.ResolveFeaturePath(ctx, feature.Key)
			if err != nil {
				t.Fatalf("Failed to resolve feature path: %v", err)
			}

			// Get relative path
			relPath, err := filepath.Rel(projectRoot, resolvedPath)
			if err != nil {
				t.Fatalf("Failed to get relative path: %v", err)
			}

			// Extract directory and filename
			dirPath := filepath.Dir(relPath) + "/"
			filename := filepath.Base(relPath)

			// Verify path
			if dirPath != tt.expectedPath {
				t.Errorf("Path mismatch:\n  got:  %q\n  want: %q", dirPath, tt.expectedPath)
			}

			// Verify filename
			if filename != tt.expectedFilename {
				t.Errorf("Filename mismatch:\n  got:  %q\n  want: %q", filename, tt.expectedFilename)
			}
		})
	}
}

// Helper functions

func stringPtr(s string) *string {
	return &s
}

func getFeatureNum(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func getEpicNum(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
