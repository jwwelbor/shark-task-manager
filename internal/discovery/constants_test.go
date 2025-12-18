package discovery

import (
	"regexp"
	"testing"
)

// TestCaptureGroupConstants tests that capture group constants are defined
func TestCaptureGroupConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"epic_id", CaptureGroupEpicID, "epic_id"},
		{"epic_slug", CaptureGroupEpicSlug, "epic_slug"},
		{"epic_num", CaptureGroupEpicNum, "epic_num"},
		{"feature_id", CaptureGroupFeatureID, "feature_id"},
		{"feature_slug", CaptureGroupFeatureSlug, "feature_slug"},
		{"feature_num", CaptureGroupFeatureNum, "feature_num"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("Constant = %v, want %v", tt.value, tt.want)
			}
		})
	}
}

// TestDefaultPatternValidity tests that default patterns compile successfully
func TestDefaultPatternValidity(t *testing.T) {
	patterns := []struct {
		name    string
		pattern string
	}{
		{"DefaultEpicFolderPattern", DefaultEpicFolderPattern},
		{"DefaultEpicSpecialPattern", DefaultEpicSpecialPattern},
		{"DefaultFeatureFolderPattern", DefaultFeatureFolderPattern},
		{"DefaultFeatureFolderShortPattern", DefaultFeatureFolderShortPattern},
		{"DefaultFeatureFilePattern", DefaultFeatureFilePattern},
		{"DefaultFeatureFileLongPattern", DefaultFeatureFileLongPattern},
	}

	for _, tt := range patterns {
		t.Run(tt.name, func(t *testing.T) {
			_, err := regexp.Compile(tt.pattern)
			if err != nil {
				t.Errorf("Pattern %s failed to compile: %v", tt.name, err)
			}
		})
	}
}

// TestDefaultEpicFolderPattern tests epic folder pattern matching
func TestDefaultEpicFolderPattern(t *testing.T) {
	re := regexp.MustCompile(DefaultEpicFolderPattern)

	tests := []struct {
		name      string
		input     string
		wantMatch bool
		wantEpicID string
		wantSlug  string
	}{
		{
			name:      "valid E04",
			input:     "E04-task-mgmt-cli-core",
			wantMatch: true,
			wantEpicID: "E04",
			wantSlug:  "task-mgmt-cli-core",
		},
		{
			name:      "valid E06",
			input:     "E06-intelligent-scanning",
			wantMatch: true,
			wantEpicID: "E06",
			wantSlug:  "intelligent-scanning",
		},
		{
			name:      "invalid single digit",
			input:     "E4-epic",
			wantMatch: false,
		},
		{
			name:      "invalid no slug",
			input:     "E04",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tt.input)
			if tt.wantMatch && match == nil {
				t.Errorf("Expected match but got none for input: %s", tt.input)
				return
			}
			if !tt.wantMatch && match != nil {
				t.Errorf("Expected no match but got match for input: %s", tt.input)
				return
			}
			if tt.wantMatch && match != nil {
				epicID := match[re.SubexpIndex(CaptureGroupEpicID)]
				slug := match[re.SubexpIndex(CaptureGroupEpicSlug)]
				if epicID != tt.wantEpicID {
					t.Errorf("epic_id = %v, want %v", epicID, tt.wantEpicID)
				}
				if slug != tt.wantSlug {
					t.Errorf("epic_slug = %v, want %v", slug, tt.wantSlug)
				}
			}
		})
	}
}

// TestDefaultEpicSpecialPattern tests special epic type pattern matching
func TestDefaultEpicSpecialPattern(t *testing.T) {
	re := regexp.MustCompile(DefaultEpicSpecialPattern)

	tests := []struct {
		name      string
		input     string
		wantMatch bool
		wantEpicID string
	}{
		{
			name:      "tech-debt",
			input:     "tech-debt",
			wantMatch: true,
			wantEpicID: "tech-debt",
		},
		{
			name:      "bugs",
			input:     "bugs",
			wantMatch: true,
			wantEpicID: "bugs",
		},
		{
			name:      "change-cards",
			input:     "change-cards",
			wantMatch: true,
			wantEpicID: "change-cards",
		},
		{
			name:      "invalid other",
			input:     "other-special",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tt.input)
			if tt.wantMatch && match == nil {
				t.Errorf("Expected match but got none for input: %s", tt.input)
				return
			}
			if !tt.wantMatch && match != nil {
				t.Errorf("Expected no match but got match for input: %s", tt.input)
				return
			}
			if tt.wantMatch && match != nil {
				epicID := match[re.SubexpIndex(CaptureGroupEpicID)]
				if epicID != tt.wantEpicID {
					t.Errorf("epic_id = %v, want %v", epicID, tt.wantEpicID)
				}
			}
		})
	}
}

// TestDefaultFeatureFolderPattern tests feature folder pattern matching
func TestDefaultFeatureFolderPattern(t *testing.T) {
	re := regexp.MustCompile(DefaultFeatureFolderPattern)

	tests := []struct {
		name           string
		input          string
		wantMatch      bool
		wantEpicID     string
		wantFeatureID  string
		wantSlug       string
	}{
		{
			name:          "valid E04-F07",
			input:         "E04-F07-initialization-sync",
			wantMatch:     true,
			wantEpicID:    "E04",
			wantFeatureID: "F07",
			wantSlug:      "initialization-sync",
		},
		{
			name:          "valid E06-F02",
			input:         "E06-F02-epic-feature-discovery",
			wantMatch:     true,
			wantEpicID:    "E06",
			wantFeatureID: "F02",
			wantSlug:      "epic-feature-discovery",
		},
		{
			name:      "invalid single digit epic",
			input:     "E4-F07-feature",
			wantMatch: false,
		},
		{
			name:      "invalid no slug",
			input:     "E04-F07",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tt.input)
			if tt.wantMatch && match == nil {
				t.Errorf("Expected match but got none for input: %s", tt.input)
				return
			}
			if !tt.wantMatch && match != nil {
				t.Errorf("Expected no match but got match for input: %s", tt.input)
				return
			}
			if tt.wantMatch && match != nil {
				epicID := match[re.SubexpIndex(CaptureGroupEpicID)]
				featureID := match[re.SubexpIndex(CaptureGroupFeatureID)]
				slug := match[re.SubexpIndex(CaptureGroupFeatureSlug)]
				if epicID != tt.wantEpicID {
					t.Errorf("epic_id = %v, want %v", epicID, tt.wantEpicID)
				}
				if featureID != tt.wantFeatureID {
					t.Errorf("feature_id = %v, want %v", featureID, tt.wantFeatureID)
				}
				if slug != tt.wantSlug {
					t.Errorf("feature_slug = %v, want %v", slug, tt.wantSlug)
				}
			}
		})
	}
}

// TestDefaultFeatureFilePattern tests PRD file pattern matching
func TestDefaultFeatureFilePattern(t *testing.T) {
	re := regexp.MustCompile(DefaultFeatureFilePattern)

	tests := []struct {
		name      string
		input     string
		wantMatch bool
	}{
		{"valid prd.md", "prd.md", true},
		{"invalid PRD.md", "PRD.md", false},
		{"invalid prd-feature.md", "prd-feature.md", false},
		{"invalid epic.md", "epic.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := re.MatchString(tt.input)
			if match != tt.wantMatch {
				t.Errorf("Match = %v, want %v for input: %s", match, tt.wantMatch, tt.input)
			}
		})
	}
}

// TestDefaultConstants tests default constant values
func TestDefaultConstants(t *testing.T) {
	if DefaultDocsRoot != "docs/plan" {
		t.Errorf("DefaultDocsRoot = %v, want docs/plan", DefaultDocsRoot)
	}
	if DefaultIndexFileName != "epic-index.md" {
		t.Errorf("DefaultIndexFileName = %v, want epic-index.md", DefaultIndexFileName)
	}
}

// TestRelatedDocPatterns tests that related doc patterns are defined
func TestRelatedDocPatterns(t *testing.T) {
	if len(RelatedDocPatterns) == 0 {
		t.Error("RelatedDocPatterns should not be empty")
	}

	// Verify some expected patterns exist
	expectedPatterns := []string{
		"02-*.md",
		"architecture.md",
		"test-criteria.md",
	}

	for _, expected := range expectedPatterns {
		found := false
		for _, pattern := range RelatedDocPatterns {
			if pattern == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected pattern %s not found in RelatedDocPatterns", expected)
		}
	}
}

// TestExcludedSubfolders tests that excluded subfolders are defined
func TestExcludedSubfolders(t *testing.T) {
	if len(ExcludedSubfolders) == 0 {
		t.Error("ExcludedSubfolders should not be empty")
	}

	// Verify expected folders are excluded
	expectedFolders := []string{"tasks", "prps"}
	for _, expected := range expectedFolders {
		found := false
		for _, folder := range ExcludedSubfolders {
			if folder == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected folder %s not found in ExcludedSubfolders", expected)
		}
	}
}
