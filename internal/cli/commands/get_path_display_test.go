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
	tests := []struct {
		name               string
		customFeaturePath  *string
		customTaskFilename *string
		expectedPath       string
		expectedFilename   string
	}{
		{
			name:               "default path + default filename",
			customFeaturePath:  nil,
			customTaskFilename: nil,
			expectedPath:       "docs/plan/E99/E99-F01/tasks/",
			expectedFilename:   "T-E99-F01-001-test-task-default-path-default-filename.md",
		},
		{
			name:               "default path + custom filename",
			customFeaturePath:  nil,
			customTaskFilename: stringPtr(projectRoot + "/custom-task.md"),
			expectedPath:       "./",
			expectedFilename:   "custom-task.md",
		},
		{
			name:               "custom path + default filename",
			customFeaturePath:  stringPtr("custom/feature/path"),
			customTaskFilename: nil,
			expectedPath:       "custom/feature/path/E99-F03/tasks/",
			expectedFilename:   "T-E99-F03-001-test-task-custom-path-default-filename.md",
		},
		{
			name:               "custom path + custom filename",
			customFeaturePath:  stringPtr("custom/feature/path"),
			customTaskFilename: stringPtr(projectRoot + "/custom/prp/task-spec.md"),
			expectedPath:       "custom/prp/",
			expectedFilename:   "task-spec.md",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create feature with custom path if specified
			featureKey := "E99-F" + getFeatureNum(i+1)
			feature := &models.Feature{
				EpicID: epic.ID,
				Key:    featureKey,
				Title:  "Test Feature " + tt.name,
				Status: models.FeatureStatusActive,
			}
			if tt.customFeaturePath != nil {
				feature.CustomFolderPath = tt.customFeaturePath
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
	tests := []struct {
		name             string
		customEpicPath   *string
		customFilename   *string
		expectedPath     string
		expectedFilename string
	}{
		{
			name:             "default path + default filename",
			customEpicPath:   nil,
			customFilename:   nil,
			expectedPath:     "docs/plan/E98/",
			expectedFilename: "epic.md",
		},
		{
			name:             "default path + custom filename",
			customEpicPath:   nil,
			customFilename:   stringPtr(projectRoot + "/docs/plan/E97/custom-epic.md"),
			expectedPath:     "docs/plan/E97/",
			expectedFilename: "custom-epic.md",
		},
		{
			name:             "custom path + default filename",
			customEpicPath:   stringPtr("roadmap/2025-q1"),
			customFilename:   nil,
			expectedPath:     "roadmap/2025-q1/E96/",
			expectedFilename: "epic.md",
		},
		{
			name:             "custom path + custom filename",
			customEpicPath:   stringPtr("roadmap/2025-q2"),
			customFilename:   stringPtr(projectRoot + "/roadmap/2025-q2/overview.md"),
			expectedPath:     "roadmap/2025-q2/",
			expectedFilename: "overview.md",
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
			if tt.customEpicPath != nil {
				epic.CustomFolderPath = tt.customEpicPath
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
	tests := []struct {
		name             string
		customPath       *string
		customFilename   *string
		expectedPath     string
		expectedFilename string
	}{
		{
			name:             "default path + default filename",
			customPath:       nil,
			customFilename:   nil,
			expectedPath:     "docs/plan/E96/E96-F01/",
			expectedFilename: "feature.md",
		},
		{
			name:             "default path + custom filename",
			customPath:       nil,
			customFilename:   stringPtr(projectRoot + "/docs/plan/E96/E96-F02/spec.md"),
			expectedPath:     "docs/plan/E96/E96-F02/",
			expectedFilename: "spec.md",
		},
		{
			name:             "custom path + default filename",
			customPath:       stringPtr("features/auth"),
			customFilename:   nil,
			expectedPath:     "features/auth/E96-F03/",
			expectedFilename: "feature.md",
		},
		{
			name:             "custom path + custom filename",
			customPath:       stringPtr("features/payments"),
			customFilename:   stringPtr(projectRoot + "/features/payments/requirements.md"),
			expectedPath:     "features/payments/",
			expectedFilename: "requirements.md",
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
			if tt.customPath != nil {
				feature.CustomFolderPath = tt.customPath
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
