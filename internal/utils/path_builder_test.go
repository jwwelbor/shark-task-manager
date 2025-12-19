package utils

import (
	"path/filepath"
	"testing"
)

func TestPathBuilder_ResolveEpicPath(t *testing.T) {
	projectRoot := "/home/user/project"
	pb := NewPathBuilder(projectRoot)

	tests := []struct {
		name               string
		epicKey            string
		filename           *string
		customFolderPath   *string
		wantPath           string
		wantErr            bool
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
		name                string
		epicKey             string
		featureKey          string
		filename            *string
		featureCustomPath   *string
		epicCustomPath      *string
		wantPath            string
		wantErr             bool
	}{
		{
			name:               "default path",
			epicKey:            "E01",
			featureKey:         "F01",
			filename:           nil,
			featureCustomPath:  nil,
			epicCustomPath:     nil,
			wantPath:           filepath.Join(projectRoot, "docs", "plan", "E01", "F01", "feature.md"),
			wantErr:            false,
		},
		{
			name:               "inherit epic custom path",
			epicKey:            "E01",
			featureKey:         "F01",
			filename:           nil,
			featureCustomPath:  nil,
			epicCustomPath:     strPtr("docs/custom"),
			wantPath:           filepath.Join(projectRoot, "docs", "custom", "E01", "F01", "feature.md"),
			wantErr:            false,
		},
		{
			name:               "feature custom path overrides epic",
			epicKey:            "E01",
			featureKey:         "F01",
			filename:           nil,
			featureCustomPath:  strPtr("docs/feature-custom"),
			epicCustomPath:     strPtr("docs/epic-custom"),
			wantPath:           filepath.Join(projectRoot, "docs", "feature-custom", "F01", "feature.md"),
			wantErr:            false,
		},
		{
			name:               "filename overrides all paths",
			epicKey:            "E01",
			featureKey:         "F01",
			filename:           strPtr("docs/override.md"),
			featureCustomPath:  strPtr("docs/feature-custom"),
			epicCustomPath:     strPtr("docs/epic-custom"),
			wantPath:           "docs/override.md",
			wantErr:            false,
		},
		{
			name:               "feature custom path with multiple levels",
			epicKey:            "E01",
			featureKey:         "F01",
			filename:           nil,
			featureCustomPath:  strPtr("docs/roadmap/2025-q1/modules/auth"),
			epicCustomPath:     nil,
			wantPath:           filepath.Join(projectRoot, "docs", "roadmap", "2025-q1", "modules", "auth", "F01", "feature.md"),
			wantErr:            false,
		},
		{
			name:               "empty feature custom path uses epic path",
			epicKey:            "E01",
			featureKey:         "F01",
			filename:           nil,
			featureCustomPath:  strPtr(""),
			epicCustomPath:     strPtr("docs/epic-custom"),
			wantPath:           filepath.Join(projectRoot, "docs", "epic-custom", "E01", "F01", "feature.md"),
			wantErr:            false,
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
		name               string
		epicKey            string
		featureKey         string
		taskKey            string
		filename           *string
		featureCustomPath  *string
		epicCustomPath     *string
		wantPath           string
		wantErr            bool
	}{
		{
			name:               "default path",
			epicKey:            "E01",
			featureKey:         "F01",
			taskKey:            "T-E01-F01-001",
			filename:           nil,
			featureCustomPath:  nil,
			epicCustomPath:     nil,
			wantPath:           filepath.Join(projectRoot, "docs", "plan", "E01", "F01", "tasks", "T-E01-F01-001.md"),
			wantErr:            false,
		},
		{
			name:               "inherit epic custom path",
			epicKey:            "E01",
			featureKey:         "F01",
			taskKey:            "T-E01-F01-001",
			filename:           nil,
			featureCustomPath:  nil,
			epicCustomPath:     strPtr("docs/custom"),
			wantPath:           filepath.Join(projectRoot, "docs", "custom", "E01", "F01", "tasks", "T-E01-F01-001.md"),
			wantErr:            false,
		},
		{
			name:               "inherit feature custom path",
			epicKey:            "E01",
			featureKey:         "F01",
			taskKey:            "T-E01-F01-001",
			filename:           nil,
			featureCustomPath:  strPtr("docs/feature-custom"),
			epicCustomPath:     strPtr("docs/epic-custom"),
			wantPath:           filepath.Join(projectRoot, "docs", "feature-custom", "F01", "tasks", "T-E01-F01-001.md"),
			wantErr:            false,
		},
		{
			name:               "filename overrides all paths",
			epicKey:            "E01",
			featureKey:         "F01",
			taskKey:            "T-E01-F01-001",
			filename:           strPtr("docs/investigation.md"),
			featureCustomPath:  strPtr("docs/feature-custom"),
			epicCustomPath:     strPtr("docs/epic-custom"),
			wantPath:           "docs/investigation.md",
			wantErr:            false,
		},
		{
			name:               "feature path overrides epic path",
			epicKey:            "E01",
			featureKey:         "F01",
			taskKey:            "T-E01-F01-001",
			filename:           nil,
			featureCustomPath:  strPtr("docs/feature-custom"),
			epicCustomPath:     strPtr("docs/epic-custom"),
			wantPath:           filepath.Join(projectRoot, "docs", "feature-custom", "F01", "tasks", "T-E01-F01-001.md"),
			wantErr:            false,
		},
		{
			name:               "empty feature custom path uses epic path",
			epicKey:            "E01",
			featureKey:         "F01",
			taskKey:            "T-E01-F01-001",
			filename:           nil,
			featureCustomPath:  strPtr(""),
			epicCustomPath:     strPtr("docs/epic-custom"),
			wantPath:           filepath.Join(projectRoot, "docs", "epic-custom", "E01", "F01", "tasks", "T-E01-F01-001.md"),
			wantErr:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pb.ResolveTaskPath(tt.epicKey, tt.featureKey, tt.taskKey, tt.filename, tt.featureCustomPath, tt.epicCustomPath)

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

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
