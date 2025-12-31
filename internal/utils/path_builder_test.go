package utils

import (
	"path/filepath"
	"testing"
)

func TestPathBuilder_ResolveEpicPath(t *testing.T) {
	projectRoot := "/home/user/project"
	pb := NewPathBuilder(projectRoot)

	tests := []struct {
		name             string
		epicKey          string
		filename         *string
		customFolderPath *string
		wantPath         string
		wantErr          bool
	}{
		{
			name:             "default path",
			epicKey:          "E01",
			filename:         nil,
			customFolderPath: nil,
			wantPath:         filepath.Join(projectRoot, "docs", "plan", "E01", "epic.md"),
			wantErr:          false,
		},
		{
			name:             "with custom folder path",
			epicKey:          "E01",
			filename:         nil,
			customFolderPath: strPtr("docs/custom"),
			wantPath:         filepath.Join(projectRoot, "docs", "custom", "E01", "epic.md"),
			wantErr:          false,
		},
		{
			name:             "with custom folder path multiple levels",
			epicKey:          "E01",
			filename:         nil,
			customFolderPath: strPtr("docs/roadmap/2025-q1"),
			wantPath:         filepath.Join(projectRoot, "docs", "roadmap", "2025-q1", "E01", "epic.md"),
			wantErr:          false,
		},
		{
			name:             "filename overrides custom path",
			epicKey:          "E01",
			filename:         strPtr("docs/special/epic.md"),
			customFolderPath: strPtr("docs/custom"),
			wantPath:         "docs/special/epic.md",
			wantErr:          false,
		},
		{
			name:             "filename only",
			epicKey:          "E01",
			filename:         strPtr("docs/override/epic.md"),
			customFolderPath: nil,
			wantPath:         "docs/override/epic.md",
			wantErr:          false,
		},
		{
			name:             "empty custom folder path treated as nil",
			epicKey:          "E01",
			filename:         nil,
			customFolderPath: strPtr(""),
			wantPath:         filepath.Join(projectRoot, "docs", "plan", "E01", "epic.md"),
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pb.ResolveEpicPath(tt.epicKey, tt.filename, tt.customFolderPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveEpicPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.wantPath {
				t.Errorf("ResolveEpicPath() = %v, want %v", got, tt.wantPath)
			}
		})
	}
}

func TestPathBuilder_ResolveFeaturePath(t *testing.T) {
	projectRoot := "/home/user/project"
	pb := NewPathBuilder(projectRoot)

	tests := []struct {
		name              string
		epicKey           string
		featureKey        string
		filename          *string
		featureCustomPath *string
		epicCustomPath    *string
		wantPath          string
		wantErr           bool
	}{
		{
			name:              "default path",
			epicKey:           "E01",
			featureKey:        "F01",
			filename:          nil,
			featureCustomPath: nil,
			epicCustomPath:    nil,
			wantPath:          filepath.Join(projectRoot, "docs", "plan", "E01", "F01", "feature.md"),
			wantErr:           false,
		},
		{
			name:              "inherit epic custom path",
			epicKey:           "E01",
			featureKey:        "F01",
			filename:          nil,
			featureCustomPath: nil,
			epicCustomPath:    strPtr("docs/custom"),
			wantPath:          filepath.Join(projectRoot, "docs", "custom", "E01", "F01", "feature.md"),
			wantErr:           false,
		},
		{
			name:              "feature custom path overrides epic",
			epicKey:           "E01",
			featureKey:        "F01",
			filename:          nil,
			featureCustomPath: strPtr("docs/feature-custom"),
			epicCustomPath:    strPtr("docs/epic-custom"),
			wantPath:          filepath.Join(projectRoot, "docs", "feature-custom", "F01", "feature.md"),
			wantErr:           false,
		},
		{
			name:              "filename overrides all paths",
			epicKey:           "E01",
			featureKey:        "F01",
			filename:          strPtr("docs/override.md"),
			featureCustomPath: strPtr("docs/feature-custom"),
			epicCustomPath:    strPtr("docs/epic-custom"),
			wantPath:          "docs/override.md",
			wantErr:           false,
		},
		{
			name:              "feature custom path with multiple levels",
			epicKey:           "E01",
			featureKey:        "F01",
			filename:          nil,
			featureCustomPath: strPtr("docs/roadmap/2025-q1/modules/auth"),
			epicCustomPath:    nil,
			wantPath:          filepath.Join(projectRoot, "docs", "roadmap", "2025-q1", "modules", "auth", "F01", "feature.md"),
			wantErr:           false,
		},
		{
			name:              "empty feature custom path uses epic path",
			epicKey:           "E01",
			featureKey:        "F01",
			filename:          nil,
			featureCustomPath: strPtr(""),
			epicCustomPath:    strPtr("docs/epic-custom"),
			wantPath:          filepath.Join(projectRoot, "docs", "epic-custom", "E01", "F01", "feature.md"),
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pb.ResolveFeaturePath(tt.epicKey, tt.featureKey, tt.filename, tt.featureCustomPath, tt.epicCustomPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveFeaturePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.wantPath {
				t.Errorf("ResolveFeaturePath() = %v, want %v", got, tt.wantPath)
			}
		})
	}
}

func TestPathBuilder_ResolveTaskPath(t *testing.T) {
	projectRoot := "/home/user/project"
	pb := NewPathBuilder(projectRoot)

	tests := []struct {
		name              string
		epicKey           string
		featureKey        string
		taskKey           string
		taskTitle         string
		filename          *string
		featureCustomPath *string
		epicCustomPath    *string
		wantPath          string
		wantErr           bool
	}{
		{
			name:              "default path without title (backward compat)",
			epicKey:           "E01",
			featureKey:        "F01",
			taskKey:           "T-E01-F01-001",
			taskTitle:         "",
			filename:          nil,
			featureCustomPath: nil,
			epicCustomPath:    nil,
			wantPath:          filepath.Join(projectRoot, "docs", "plan", "E01", "F01", "tasks", "T-E01-F01-001.md"),
			wantErr:           false,
		},
		{
			name:              "default path with title and slug",
			epicKey:           "E01",
			featureKey:        "F01",
			taskKey:           "T-E01-F01-001",
			taskTitle:         "Some Task Description",
			filename:          nil,
			featureCustomPath: nil,
			epicCustomPath:    nil,
			wantPath:          filepath.Join(projectRoot, "docs", "plan", "E01", "F01", "tasks", "T-E01-F01-001-some-task-description.md"),
			wantErr:           false,
		},
		{
			name:              "inherit epic custom path",
			epicKey:           "E01",
			featureKey:        "F01",
			taskKey:           "T-E01-F01-001",
			taskTitle:         "",
			filename:          nil,
			featureCustomPath: nil,
			epicCustomPath:    strPtr("docs/custom"),
			wantPath:          filepath.Join(projectRoot, "docs", "custom", "E01", "F01", "tasks", "T-E01-F01-001.md"),
			wantErr:           false,
		},
		{
			name:              "inherit feature custom path",
			epicKey:           "E01",
			featureKey:        "F01",
			taskKey:           "T-E01-F01-001",
			taskTitle:         "",
			filename:          nil,
			featureCustomPath: strPtr("docs/feature-custom"),
			epicCustomPath:    strPtr("docs/epic-custom"),
			wantPath:          filepath.Join(projectRoot, "docs", "feature-custom", "F01", "tasks", "T-E01-F01-001.md"),
			wantErr:           false,
		},
		{
			name:              "filename overrides all paths",
			epicKey:           "E01",
			featureKey:        "F01",
			taskKey:           "T-E01-F01-001",
			taskTitle:         "",
			filename:          strPtr("docs/investigation.md"),
			featureCustomPath: strPtr("docs/feature-custom"),
			epicCustomPath:    strPtr("docs/epic-custom"),
			wantPath:          "docs/investigation.md",
			wantErr:           false,
		},
		{
			name:              "feature path overrides epic path",
			epicKey:           "E01",
			featureKey:        "F01",
			taskKey:           "T-E01-F01-001",
			taskTitle:         "",
			filename:          nil,
			featureCustomPath: strPtr("docs/feature-custom"),
			epicCustomPath:    strPtr("docs/epic-custom"),
			wantPath:          filepath.Join(projectRoot, "docs", "feature-custom", "F01", "tasks", "T-E01-F01-001.md"),
			wantErr:           false,
		},
		{
			name:              "empty feature custom path uses epic path",
			epicKey:           "E01",
			featureKey:        "F01",
			taskKey:           "T-E01-F01-001",
			taskTitle:         "",
			filename:          nil,
			featureCustomPath: strPtr(""),
			epicCustomPath:    strPtr("docs/epic-custom"),
			wantPath:          filepath.Join(projectRoot, "docs", "epic-custom", "E01", "F01", "tasks", "T-E01-F01-001.md"),
			wantErr:           false,
		},
		{
			name:              "title with slug and custom path",
			epicKey:           "E01",
			featureKey:        "F01",
			taskKey:           "T-E01-F01-002",
			taskTitle:         "Fix API Bug",
			filename:          nil,
			featureCustomPath: strPtr("docs/features"),
			epicCustomPath:    nil,
			wantPath:          filepath.Join(projectRoot, "docs", "features", "F01", "tasks", "T-E01-F01-002-fix-api-bug.md"),
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pb.ResolveTaskPath(tt.epicKey, tt.featureKey, tt.taskKey, tt.taskTitle, tt.filename, tt.featureCustomPath, tt.epicCustomPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveTaskPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.wantPath {
				t.Errorf("ResolveTaskPath() = %v, want %v", got, tt.wantPath)
			}
		})
	}
}

func TestPathBuilder_Precedence(t *testing.T) {
	// Test that precedence rules are correctly implemented
	projectRoot := "/home/user/project"
	pb := NewPathBuilder(projectRoot)

	// Test: filename > custom path > default
	customPath := strPtr("docs/custom")
	filename := strPtr("docs/override.md")

	path, err := pb.ResolveEpicPath("E01", filename, customPath)
	if err != nil {
		t.Errorf("ResolveEpicPath() error = %v", err)
	}
	if path != "docs/override.md" {
		t.Errorf("Precedence test: filename should win, got %v", path)
	}

	// Test: custom path > default
	path, err = pb.ResolveEpicPath("E01", nil, customPath)
	if err != nil {
		t.Errorf("ResolveEpicPath() error = %v", err)
	}
	expected := filepath.Join(projectRoot, "docs", "custom", "E01", "epic.md")
	if path != expected {
		t.Errorf("Precedence test: custom path should win, got %v, want %v", path, expected)
	}
}

// TestResolveTaskPathFromFeatureFile tests task path resolution when feature has a slug-based directory
// This test reproduces the bug where epic directory is "E10-advanced-task..." but epic key is just "E10"
func TestResolveTaskPathFromFeatureFile(t *testing.T) {
	projectRoot := "/home/user/project"

	// Real-world scenario from the bug:
	// - Epic key in DB: "E10"
	// - Epic directory: "docs/plan/E10-advanced-task-intelligence-context-management/"
	// - Feature file_path in DB: "docs/plan/E10-advanced-task-intelligence-context-management/E10-F01-task-activity-notes-system/feature.md"
	// - Feature custom_folder_path in DB: NULL
	// - Epic custom_folder_path in DB: NULL

	epicKey := "E10"
	featureKey := "E10-F01"
	taskKey := "T-E10-F01-001"
	featureFilePath := "docs/plan/E10-advanced-task-intelligence-context-management/E10-F01-task-activity-notes-system/feature.md"

	// When custom_folder_path is NULL, PathBuilder uses epicKey directly
	pb := NewPathBuilder(projectRoot)

	// BUG: This will produce docs/plan/E10/E10-F01/tasks/T-E10-F01-001.md
	// because it uses epic key "E10" instead of finding actual epic directory
	buggyPath, err := pb.ResolveTaskPath(epicKey, featureKey, taskKey, "", nil, nil, nil)
	if err != nil {
		t.Fatalf("ResolveTaskPath failed: %v", err)
	}

	// This is what it currently produces (WRONG):
	wrongPath := filepath.Join(projectRoot, "docs", "plan", "E10", "E10-F01", "tasks", "T-E10-F01-001.md")
	if buggyPath != wrongPath {
		t.Errorf("Expected buggy behavior to produce %s, got %s", wrongPath, buggyPath)
	}

	// This is what it SHOULD produce (derived from feature's actual location):
	// Extract feature directory from feature file path
	featureDir := filepath.Dir(featureFilePath) // "docs/plan/E10-advanced-task-intelligence-context-management/E10-F01-task-activity-notes-system"
	correctPath := filepath.Join(featureDir, "tasks", taskKey+".md")
	expectedCorrectPath := "docs/plan/E10-advanced-task-intelligence-context-management/E10-F01-task-activity-notes-system/tasks/T-E10-F01-001.md"

	if correctPath != expectedCorrectPath {
		t.Errorf("Correct path derivation failed. Got: %s, Expected: %s", correctPath, expectedCorrectPath)
	}

	// The bug: buggyPath != correctPath
	if buggyPath == filepath.Join(projectRoot, correctPath) {
		t.Log("PATH RESOLUTION WORKS CORRECTLY")
	} else {
		t.Logf("BUG REPRODUCED:")
		t.Logf("  Current (wrong):  %s", buggyPath)
		t.Logf("  Expected (right): %s", filepath.Join(projectRoot, correctPath))
		t.Log("SOLUTION: Derive task path from feature's file_path, not from reconstructing via epic/feature keys")
	}
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
