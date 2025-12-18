package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/patterns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFolderScanner_Scan(t *testing.T) {
	tests := []struct {
		name              string
		setupFunc         func(t *testing.T) string // Returns temp dir path
		patternOverrides  *patterns.PatternConfig
		expectedEpics     int
		expectedFeatures  int
		expectedStats     ScanStats
		validateEpics     func(t *testing.T, epics []FolderEpic)
		validateFeatures  func(t *testing.T, features []FolderFeature)
		shouldError       bool
	}{
		{
			name: "scan standard E##-epic-slug folders",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "E04-task-mgmt-cli-core"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "E05-advanced-querying"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "E06-intelligent-scanning"), 0755))
				return tmpDir
			},
			expectedEpics:    3,
			expectedFeatures: 0,
			expectedStats: ScanStats{
				FoldersScanned: 3,
				FilesAnalyzed:  0,
			},
			validateEpics: func(t *testing.T, epics []FolderEpic) {
				keys := make(map[string]bool)
				for _, epic := range epics {
					keys[epic.Key] = true
				}
				assert.True(t, keys["E04"], "Should find E04")
				assert.True(t, keys["E05"], "Should find E05")
				assert.True(t, keys["E06"], "Should find E06")

				// Verify slugs are extracted
				for _, epic := range epics {
					assert.NotEmpty(t, epic.Slug, "Epic slug should be extracted")
				}
			},
		},
		{
			name: "scan special epic types (tech-debt, bugs)",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "tech-debt"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "bugs"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "change-cards"), 0755))
				return tmpDir
			},
			expectedEpics:    3,
			expectedFeatures: 0,
			expectedStats: ScanStats{
				FoldersScanned: 3,
				FilesAnalyzed:  0,
			},
			validateEpics: func(t *testing.T, epics []FolderEpic) {
				keys := make(map[string]bool)
				for _, epic := range epics {
					keys[epic.Key] = true
				}
				assert.True(t, keys["tech-debt"], "Should find tech-debt")
				assert.True(t, keys["bugs"], "Should find bugs")
				assert.True(t, keys["change-cards"], "Should find change-cards")
			},
		},
		{
			name: "scan epic with epic.md file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				epicDir := filepath.Join(tmpDir, "E04-task-mgmt-cli-core")
				require.NoError(t, os.MkdirAll(epicDir, 0755))
				epicMdPath := filepath.Join(epicDir, "epic.md")
				require.NoError(t, os.WriteFile(epicMdPath, []byte("# Epic Title"), 0644))
				return tmpDir
			},
			expectedEpics:    1,
			expectedFeatures: 0,
			expectedStats: ScanStats{
				FoldersScanned: 1,
				FilesAnalyzed:  1,
			},
			validateEpics: func(t *testing.T, epics []FolderEpic) {
				require.Len(t, epics, 1)
				assert.Equal(t, "E04", epics[0].Key)
				assert.NotNil(t, epics[0].EpicMdPath)
				assert.Contains(t, *epics[0].EpicMdPath, "epic.md")
			},
		},
		{
			name: "scan epic with features (E##-F##-feature-slug)",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				epicDir := filepath.Join(tmpDir, "E04-task-mgmt-cli-core")
				require.NoError(t, os.MkdirAll(epicDir, 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(epicDir, "E04-F01-database-schema"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(epicDir, "E04-F07-initialization-sync"), 0755))
				return tmpDir
			},
			expectedEpics:    1,
			expectedFeatures: 2,
			expectedStats: ScanStats{
				FoldersScanned: 3,
				FilesAnalyzed:  0,
			},
			validateFeatures: func(t *testing.T, features []FolderFeature) {
				keys := make(map[string]bool)
				for _, feature := range features {
					keys[feature.Key] = true
					assert.Equal(t, "E04", feature.EpicKey, "Feature should belong to E04")
				}
				assert.True(t, keys["E04-F01"], "Should find E04-F01")
				assert.True(t, keys["E04-F07"], "Should find E04-F07")
			},
		},
		{
			name: "scan feature with prd.md file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				epicDir := filepath.Join(tmpDir, "E04-task-mgmt-cli-core")
				featureDir := filepath.Join(epicDir, "E04-F07-initialization-sync")
				require.NoError(t, os.MkdirAll(featureDir, 0755))
				prdPath := filepath.Join(featureDir, "prd.md")
				require.NoError(t, os.WriteFile(prdPath, []byte("# Feature PRD"), 0644))
				return tmpDir
			},
			expectedEpics:    1,
			expectedFeatures: 1,
			expectedStats: ScanStats{
				FoldersScanned: 2,
				FilesAnalyzed:  1,
			},
			validateFeatures: func(t *testing.T, features []FolderFeature) {
				require.Len(t, features, 1)
				assert.Equal(t, "E04-F07", features[0].Key)
				assert.NotNil(t, features[0].PrdPath)
				assert.Contains(t, *features[0].PrdPath, "prd.md")
			},
		},
		{
			name: "scan feature with related documents",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				epicDir := filepath.Join(tmpDir, "E04-task-mgmt-cli-core")
				featureDir := filepath.Join(epicDir, "E04-F07-initialization-sync")
				require.NoError(t, os.MkdirAll(featureDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "prd.md"), []byte("# PRD"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "02-architecture.md"), []byte("# Arch"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "04-backend-design.md"), []byte("# Backend"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "09-test-criteria.md"), []byte("# Tests"), 0644))
				return tmpDir
			},
			expectedEpics:    1,
			expectedFeatures: 1,
			expectedStats: ScanStats{
				FoldersScanned: 2,
				FilesAnalyzed:  4, // prd.md + 3 related docs
			},
			validateFeatures: func(t *testing.T, features []FolderFeature) {
				require.Len(t, features, 1)
				assert.Len(t, features[0].RelatedDocs, 3)

				// Verify related docs don't include prd.md
				for _, doc := range features[0].RelatedDocs {
					assert.NotContains(t, doc, "prd.md")
				}
			},
		},
		{
			name: "skip hidden directories",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "E04-task-mgmt-cli-core"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".hidden-folder"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755))
				return tmpDir
			},
			expectedEpics:    1,
			expectedFeatures: 0,
			expectedStats: ScanStats{
				FoldersScanned: 1, // Only E04, hidden folders skipped
				FilesAnalyzed:  0,
			},
			validateEpics: func(t *testing.T, epics []FolderEpic) {
				require.Len(t, epics, 1)
				assert.Equal(t, "E04", epics[0].Key)
			},
		},
		{
			name: "skip tasks subfolder from related docs",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				epicDir := filepath.Join(tmpDir, "E04-task-mgmt-cli-core")
				featureDir := filepath.Join(epicDir, "E04-F07-initialization-sync")
				tasksDir := filepath.Join(featureDir, "tasks")
				require.NoError(t, os.MkdirAll(tasksDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "prd.md"), []byte("# PRD"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "02-architecture.md"), []byte("# Arch"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(tasksDir, "T-E04-F07-001.md"), []byte("# Task"), 0644))
				return tmpDir
			},
			expectedEpics:    1,
			expectedFeatures: 1,
			expectedStats: ScanStats{
				FoldersScanned: 3, // epic, feature, tasks
				FilesAnalyzed:  2, // prd.md + architecture.md (tasks folder excluded)
			},
			validateFeatures: func(t *testing.T, features []FolderFeature) {
				require.Len(t, features, 1)
				assert.Len(t, features[0].RelatedDocs, 1)
				assert.Contains(t, features[0].RelatedDocs[0], "02-architecture.md")

				// Verify task file is not in related docs
				for _, doc := range features[0].RelatedDocs {
					assert.NotContains(t, doc, "T-E04-F07-001.md")
				}
			},
		},
		{
			name: "handle empty docs root",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return tmpDir
			},
			expectedEpics:    0,
			expectedFeatures: 0,
			expectedStats: ScanStats{
				FoldersScanned: 0,
				FilesAnalyzed:  0,
			},
		},
		{
			name: "handle non-existent docs root",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "non-existent")
			},
			shouldError: true,
		},
		{
			name: "mixed epic types in same project",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "E04-task-mgmt-cli-core"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "E05-advanced-querying"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "tech-debt"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "bugs"), 0755))
				return tmpDir
			},
			expectedEpics:    4,
			expectedFeatures: 0,
			expectedStats: ScanStats{
				FoldersScanned: 4,
				FilesAnalyzed:  0,
			},
			validateEpics: func(t *testing.T, epics []FolderEpic) {
				keys := make(map[string]bool)
				for _, epic := range epics {
					keys[epic.Key] = true
				}
				assert.True(t, keys["E04"])
				assert.True(t, keys["E05"])
				assert.True(t, keys["tech-debt"])
				assert.True(t, keys["bugs"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docsRoot := tt.setupFunc(t)
			scanner := NewFolderScanner()

			epics, features, stats, err := scanner.Scan(docsRoot, tt.patternOverrides)

			if tt.shouldError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedEpics, len(epics), "Epic count mismatch")
			assert.Equal(t, tt.expectedFeatures, len(features), "Feature count mismatch")
			assert.Equal(t, tt.expectedStats.FoldersScanned, stats.FoldersScanned, "Folders scanned mismatch")
			assert.Equal(t, tt.expectedStats.FilesAnalyzed, stats.FilesAnalyzed, "Files analyzed mismatch")

			if tt.validateEpics != nil {
				tt.validateEpics(t, epics)
			}

			if tt.validateFeatures != nil {
				tt.validateFeatures(t, features)
			}
		})
	}
}

func TestFolderScanner_MatchEpicFolder(t *testing.T) {
	scanner := NewFolderScanner()

	tests := []struct {
		name       string
		folderName string
		shouldMatch bool
		expectedKey string
		expectedSlug string
	}{
		{
			name:         "match standard E##-slug format",
			folderName:   "E04-task-mgmt-cli-core",
			shouldMatch:  true,
			expectedKey:  "E04",
			expectedSlug: "task-mgmt-cli-core",
		},
		{
			name:         "match tech-debt special type",
			folderName:   "tech-debt",
			shouldMatch:  true,
			expectedKey:  "tech-debt",
			expectedSlug: "",
		},
		{
			name:         "match bugs special type",
			folderName:   "bugs",
			shouldMatch:  true,
			expectedKey:  "bugs",
			expectedSlug: "",
		},
		{
			name:        "no match for random folder",
			folderName:  "random-folder",
			shouldMatch: false,
		},
		{
			name:        "no match for feature folder",
			folderName:  "E04-F07-initialization-sync",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epic, matched := scanner.matchEpicFolder("", tt.folderName)

			assert.Equal(t, tt.shouldMatch, matched, "Match result mismatch")

			if tt.shouldMatch {
				assert.Equal(t, tt.expectedKey, epic.Key, "Epic key mismatch")
				if tt.expectedSlug != "" {
					assert.Equal(t, tt.expectedSlug, epic.Slug, "Epic slug mismatch")
				}
			}
		})
	}
}

func TestFolderScanner_MatchFeatureFolder(t *testing.T) {
	scanner := NewFolderScanner()

	tests := []struct {
		name           string
		folderName     string
		epicFolderName string
		shouldMatch    bool
		expectedKey    string
		expectedEpicKey string
		expectedSlug   string
	}{
		{
			name:            "match standard E##-F##-slug format",
			folderName:      "E04-F07-initialization-sync",
			epicFolderName:  "E04-task-mgmt-cli-core",
			shouldMatch:     true,
			expectedKey:     "E04-F07",
			expectedEpicKey: "E04",
			expectedSlug:    "initialization-sync",
		},
		// Note: Short F##-slug format is not in default patterns (would need to be added via config)
		// {
		// 	name:            "match short F##-slug format (infer epic from parent)",
		// 	folderName:      "F07-initialization-sync",
		// 	epicFolderName:  "E04-task-mgmt-cli-core",
		// 	shouldMatch:     true,
		// 	expectedKey:     "E04-F07",
		// 	expectedEpicKey: "E04",
		// 	expectedSlug:    "initialization-sync",
		// },
		{
			name:           "no match for random folder",
			folderName:     "random-folder",
			epicFolderName: "E04-task-mgmt-cli-core",
			shouldMatch:    false,
		},
		{
			name:           "no match for epic folder",
			folderName:     "E05-advanced-querying",
			epicFolderName: "E04-task-mgmt-cli-core",
			shouldMatch:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature, matched := scanner.matchFeatureFolder("", tt.folderName, tt.epicFolderName)

			assert.Equal(t, tt.shouldMatch, matched, "Match result mismatch")

			if tt.shouldMatch {
				assert.Equal(t, tt.expectedKey, feature.Key, "Feature key mismatch")
				assert.Equal(t, tt.expectedEpicKey, feature.EpicKey, "Epic key mismatch")
				assert.Equal(t, tt.expectedSlug, feature.Slug, "Feature slug mismatch")
			}
		})
	}
}

func TestFolderScanner_Performance(t *testing.T) {
	// Create test structure with 100+ folders
	tmpDir := t.TempDir()

	// Create 20 epics with 5 features each = 100 feature folders
	for epicNum := 1; epicNum <= 20; epicNum++ {
		epicDir := filepath.Join(tmpDir, fmt.Sprintf("E%02d-test-epic", epicNum))
		require.NoError(t, os.MkdirAll(epicDir, 0755))

		for featureNum := 1; featureNum <= 5; featureNum++ {
			featureDir := filepath.Join(epicDir, fmt.Sprintf("E%02d-F%02d-test-feature", epicNum, featureNum))
			require.NoError(t, os.MkdirAll(featureDir, 0755))

			// Add prd.md
			require.NoError(t, os.WriteFile(filepath.Join(featureDir, "prd.md"), []byte("# PRD"), 0644))

			// Add 2 related docs
			require.NoError(t, os.WriteFile(filepath.Join(featureDir, "02-architecture.md"), []byte("# Arch"), 0644))
			require.NoError(t, os.WriteFile(filepath.Join(featureDir, "04-backend-design.md"), []byte("# Backend"), 0644))
		}
	}

	scanner := NewFolderScanner()

	// Measure performance
	epics, features, stats, err := scanner.Scan(tmpDir, nil)

	require.NoError(t, err)
	assert.Equal(t, 20, len(epics), "Should find 20 epics")
	assert.Equal(t, 100, len(features), "Should find 100 features")
	assert.GreaterOrEqual(t, stats.FoldersScanned, 120, "Should scan at least 120 folders")

	// Performance assertion: scan should complete quickly
	// This is implicit in the test completing without timeout
	// For explicit timing, we could add timing code here
}
