package validation

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// MockRepository implements the repository interface for testing
type MockRepository struct {
	epics    []*models.Epic
	features []*models.Feature
	tasks    []*models.Task
}

func (m *MockRepository) GetAllEpics(ctx context.Context) ([]*models.Epic, error) {
	return m.epics, nil
}

func (m *MockRepository) GetAllFeatures(ctx context.Context) ([]*models.Feature, error) {
	return m.features, nil
}

func (m *MockRepository) GetAllTasks(ctx context.Context) ([]*models.Task, error) {
	return m.tasks, nil
}

func (m *MockRepository) GetEpicByID(ctx context.Context, id int64) (*models.Epic, error) {
	for _, epic := range m.epics {
		if epic.ID == id {
			return epic, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *MockRepository) GetFeatureByID(ctx context.Context, id int64) (*models.Feature, error) {
	for _, feature := range m.features {
		if feature.ID == id {
			return feature, nil
		}
	}
	return nil, sql.ErrNoRows
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}

func TestValidator_ValidateFilePaths(t *testing.T) {
	// Setup temporary directory for test files
	tempDir := t.TempDir()

	// Create test files
	taskFile := filepath.Join(tempDir, "task.md")

	if err := os.WriteFile(taskFile, []byte("task content"), 0644); err != nil {
		t.Fatalf("Failed to create test task file: %v", err)
	}

	tests := []struct {
		name            string
		tasks           []*models.Task
		wantBrokenPaths int
	}{
		{
			name: "all file paths exist",
			tasks: []*models.Task{
				{Key: "T-E01-F01-001", FilePath: strPtr(taskFile)},
			},
			wantBrokenPaths: 0,
		},
		{
			name: "missing task file",
			tasks: []*models.Task{
				{Key: "T-E01-F01-001", FilePath: strPtr(filepath.Join(tempDir, "missing-task.md"))},
			},
			wantBrokenPaths: 1,
		},
		{
			name: "multiple missing files",
			tasks: []*models.Task{
				{Key: "T-E01-F01-001", FilePath: strPtr(filepath.Join(tempDir, "missing1.md"))},
				{Key: "T-E01-F01-002", FilePath: strPtr(filepath.Join(tempDir, "missing2.md"))},
				{Key: "T-E01-F01-003", FilePath: strPtr(taskFile)}, // This one exists
			},
			wantBrokenPaths: 2,
		},
		{
			name: "nil file paths are skipped",
			tasks: []*models.Task{
				{Key: "T-E01-F01-001", FilePath: nil},
			},
			wantBrokenPaths: 0,
		},
		{
			name: "empty file paths are skipped",
			tasks: []*models.Task{
				{Key: "T-E01-F01-001", FilePath: strPtr("")},
			},
			wantBrokenPaths: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockRepository{
				epics:    []*models.Epic{},
				features: []*models.Feature{},
				tasks:    tt.tasks,
			}

			validator := NewValidator(repo)
			result, err := validator.Validate(context.Background())

			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			if len(result.BrokenFilePaths) != tt.wantBrokenPaths {
				t.Errorf("Validate() got %d broken file paths, want %d", len(result.BrokenFilePaths), tt.wantBrokenPaths)
			}

			// Verify suggestions are present for broken paths
			if len(result.BrokenFilePaths) > 0 {
				for _, failure := range result.BrokenFilePaths {
					if failure.SuggestedFix == "" {
						t.Errorf("Broken file path for %s has empty suggestion", failure.EntityKey)
					}
				}
			}
		})
	}
}

func TestValidator_ValidateRelationships(t *testing.T) {
	tests := []struct {
		name            string
		epics           []*models.Epic
		features        []*models.Feature
		tasks           []*models.Task
		wantOrphans     int
		wantOrphanFeats int
		wantOrphanTasks int
	}{
		{
			name: "all relationships valid",
			epics: []*models.Epic{
				{ID: 1, Key: "E01-test"},
			},
			features: []*models.Feature{
				{ID: 1, EpicID: 1, Key: "E01-F01-test"},
			},
			tasks: []*models.Task{
				{ID: 1, FeatureID: 1, Key: "T-E01-F01-001"},
			},
			wantOrphans: 0,
		},
		{
			name:  "orphaned feature - missing parent epic",
			epics: []*models.Epic{},
			features: []*models.Feature{
				{ID: 1, EpicID: 999, Key: "E99-F01-test"},
			},
			tasks:           []*models.Task{},
			wantOrphans:     1,
			wantOrphanFeats: 1,
		},
		{
			name: "orphaned task - missing parent feature",
			epics: []*models.Epic{
				{ID: 1, Key: "E01-test"},
			},
			features: []*models.Feature{},
			tasks: []*models.Task{
				{ID: 1, FeatureID: 999, Key: "T-E01-F99-001"},
			},
			wantOrphans:     1,
			wantOrphanTasks: 1,
		},
		{
			name: "multiple orphaned records",
			epics: []*models.Epic{
				{ID: 1, Key: "E01-test"},
			},
			features: []*models.Feature{
				{ID: 1, EpicID: 999, Key: "E99-F01-test"},
				{ID: 2, EpicID: 1, Key: "E01-F01-test"},
			},
			tasks: []*models.Task{
				{ID: 1, FeatureID: 999, Key: "T-E01-F99-001"},
				{ID: 2, FeatureID: 998, Key: "T-E01-F98-001"},
			},
			wantOrphans:     3,
			wantOrphanFeats: 1,
			wantOrphanTasks: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockRepository{
				epics:    tt.epics,
				features: tt.features,
				tasks:    tt.tasks,
			}

			validator := NewValidator(repo)
			result, err := validator.Validate(context.Background())

			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			if len(result.OrphanedRecords) != tt.wantOrphans {
				t.Errorf("Validate() got %d orphaned records, want %d", len(result.OrphanedRecords), tt.wantOrphans)
			}

			// Count orphans by entity type
			gotOrphanFeatures := 0
			gotOrphanTasks := 0
			for _, failure := range result.OrphanedRecords {
				switch failure.EntityType {
				case "feature":
					gotOrphanFeatures++
				case "task":
					gotOrphanTasks++
				}
			}

			if gotOrphanFeatures != tt.wantOrphanFeats {
				t.Errorf("Validate() got %d orphaned features, want %d", gotOrphanFeatures, tt.wantOrphanFeats)
			}
			if gotOrphanTasks != tt.wantOrphanTasks {
				t.Errorf("Validate() got %d orphaned tasks, want %d", gotOrphanTasks, tt.wantOrphanTasks)
			}

			// Verify suggestions are present for orphans
			if len(result.OrphanedRecords) > 0 {
				for _, failure := range result.OrphanedRecords {
					if failure.SuggestedFix == "" {
						t.Errorf("Orphaned record %s has empty suggestion", failure.EntityKey)
					}
				}
			}
		})
	}
}

func TestValidator_ValidationSummary(t *testing.T) {
	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "valid.md")
	if err := os.WriteFile(validFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name             string
		epics            []*models.Epic
		features         []*models.Feature
		tasks            []*models.Task
		wantTotalChecked int
		wantTotalIssues  int
		wantBrokenPaths  int
		wantOrphans      int
		wantSuccess      bool
	}{
		{
			name: "all validations pass",
			epics: []*models.Epic{
				{ID: 1, Key: "E01-test"},
			},
			features: []*models.Feature{
				{ID: 1, EpicID: 1, Key: "E01-F01-test"},
			},
			tasks: []*models.Task{
				{ID: 1, FeatureID: 1, Key: "T-E01-F01-001", FilePath: strPtr(validFile)},
			},
			wantTotalChecked: 3,
			wantTotalIssues:  0,
			wantBrokenPaths:  0,
			wantOrphans:      0,
			wantSuccess:      true,
		},
		{
			name: "mixed validation failures",
			epics: []*models.Epic{
				{ID: 1, Key: "E01-test"},
			},
			features: []*models.Feature{
				{ID: 1, EpicID: 999, Key: "E99-F01-test"},
			},
			tasks: []*models.Task{
				{ID: 1, FeatureID: 1, Key: "T-E01-F01-001", FilePath: strPtr(filepath.Join(tempDir, "missing.md"))},
			},
			wantTotalChecked: 3,
			wantTotalIssues:  2, // 1 broken path + 1 orphaned feature
			wantBrokenPaths:  1,
			wantOrphans:      1,
			wantSuccess:      false,
		},
		{
			name: "no entities to validate",
			epics:            []*models.Epic{},
			features:         []*models.Feature{},
			tasks:            []*models.Task{},
			wantTotalChecked: 0,
			wantTotalIssues:  0,
			wantBrokenPaths:  0,
			wantOrphans:      0,
			wantSuccess:      true, // Empty database is considered valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockRepository{
				epics:    tt.epics,
				features: tt.features,
				tasks:    tt.tasks,
			}

			validator := NewValidator(repo)
			result, err := validator.Validate(context.Background())

			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			if result.Summary.TotalChecked != tt.wantTotalChecked {
				t.Errorf("Summary.TotalChecked = %d, want %d", result.Summary.TotalChecked, tt.wantTotalChecked)
			}

			if result.Summary.TotalIssues != tt.wantTotalIssues {
				t.Errorf("Summary.TotalIssues = %d, want %d", result.Summary.TotalIssues, tt.wantTotalIssues)
			}

			if result.Summary.BrokenFilePaths != tt.wantBrokenPaths {
				t.Errorf("Summary.BrokenFilePaths = %d, want %d", result.Summary.BrokenFilePaths, tt.wantBrokenPaths)
			}

			if result.Summary.OrphanedRecords != tt.wantOrphans {
				t.Errorf("Summary.OrphanedRecords = %d, want %d", result.Summary.OrphanedRecords, tt.wantOrphans)
			}

			if result.IsSuccess() != tt.wantSuccess {
				t.Errorf("IsSuccess() = %v, want %v", result.IsSuccess(), tt.wantSuccess)
			}
		})
	}
}

func TestValidator_CorrectiveActionSuggestions(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name            string
		epics           []*models.Epic
		features        []*models.Feature
		tasks           []*models.Task
		wantSuggestions []string
	}{
		{
			name:  "missing file suggestion",
			epics: []*models.Epic{{ID: 1, Key: "E01-test"}},
			features: []*models.Feature{
				{ID: 1, EpicID: 1, Key: "E01-F01-test"},
			},
			tasks: []*models.Task{
				{FeatureID: 1, Key: "T-E01-F01-001", FilePath: strPtr(filepath.Join(tempDir, "missing.md"))},
			},
			wantSuggestions: []string{
				"re-scan",
				"update",
			},
		},
		{
			name:  "orphaned feature suggestion",
			epics: []*models.Epic{},
			features: []*models.Feature{
				{EpicID: 999, Key: "E99-F01-test"},
			},
			tasks: []*models.Task{},
			wantSuggestions: []string{
				"parent epic",
				"delete",
			},
		},
		{
			name: "orphaned task suggestion",
			epics: []*models.Epic{
				{ID: 1, Key: "E01-test"},
			},
			features: []*models.Feature{},
			tasks: []*models.Task{
				{FeatureID: 999, Key: "T-E01-F99-001"},
			},
			wantSuggestions: []string{
				"parent feature",
				"delete",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockRepository{
				epics:    tt.epics,
				features: tt.features,
				tasks:    tt.tasks,
			}

			validator := NewValidator(repo)
			result, err := validator.Validate(context.Background())

			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			// Collect all suggestions
			allSuggestions := make([]string, 0)
			for _, failure := range result.BrokenFilePaths {
				allSuggestions = append(allSuggestions, failure.SuggestedFix)
			}
			for _, failure := range result.OrphanedRecords {
				allSuggestions = append(allSuggestions, failure.SuggestedFix)
			}

			// Check if expected suggestion keywords are present (case-insensitive)
			for _, want := range tt.wantSuggestions {
				found := false
				for _, got := range allSuggestions {
					if containsStringCaseInsensitive(got, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected suggestion containing '%s', but not found in: %v", want, allSuggestions)
				}
			}
		})
	}
}

func TestValidator_Performance(t *testing.T) {
	// Generate 1000 entities for performance test
	epics := make([]*models.Epic, 100)
	features := make([]*models.Feature, 300)
	tasks := make([]*models.Task, 600)

	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "valid.md")
	if err := os.WriteFile(validFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	for i := 0; i < 100; i++ {
		epics[i] = &models.Epic{
			ID:  int64(i + 1),
			Key: "E01-test",
		}
	}

	for i := 0; i < 300; i++ {
		features[i] = &models.Feature{
			ID:     int64(i + 1),
			EpicID: 1,
			Key:    "E01-F01-test",
		}
	}

	for i := 0; i < 600; i++ {
		tasks[i] = &models.Task{
			ID:        int64(i + 1),
			FeatureID: 1,
			Key:       "T-E01-F01-001",
			FilePath:  strPtr(validFile),
		}
	}

	repo := &MockRepository{
		epics:    epics,
		features: features,
		tasks:    tasks,
	}

	validator := NewValidator(repo)

	// Performance requirement: validate 1000 entities in <1 second
	result, err := validator.Validate(context.Background())

	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if result.DurationMs > 1000 {
		t.Errorf("Performance requirement failed: validation took %dms, should be <1000ms", result.DurationMs)
	}

	if result.Summary.TotalChecked != 1000 {
		t.Errorf("Expected to check 1000 entities, but checked %d", result.Summary.TotalChecked)
	}
}

// Helper function
func containsStringCaseInsensitive(s, substr string) bool {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}
