package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/patterns"
)

// FolderScanner discovers epics and features by scanning directory structure
type FolderScanner struct {
	patternMatcher *PatternMatcher
}

// NewFolderScanner creates a new folder scanner with default patterns
func NewFolderScanner() *FolderScanner {
	return &FolderScanner{
		patternMatcher: NewPatternMatcher(patterns.GetDefaultPatterns()),
	}
}

// ScanStats contains statistics from folder scan
type ScanStats struct {
	FoldersScanned int
	FilesAnalyzed  int
}

// Scan walks directory tree and discovers epics/features
func (s *FolderScanner) Scan(docsRoot string, patternOverrides *patterns.PatternConfig) (
	[]FolderEpic, []FolderFeature, ScanStats, error) {

	// Override patterns if provided
	if patternOverrides != nil {
		s.patternMatcher = NewPatternMatcher(patternOverrides)
	}

	// Validate docs root exists
	if _, err := os.Stat(docsRoot); os.IsNotExist(err) {
		return nil, nil, ScanStats{}, fmt.Errorf("docs root does not exist: %s", docsRoot)
	}

	stats := ScanStats{}
	epics := []FolderEpic{}
	features := []FolderFeature{}

	// Track epic folders to associate features
	epicFolderNames := make(map[string]string) // path -> epic folder name

	// Walk docs root directory
	err := filepath.Walk(docsRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Only process directories
		if !info.IsDir() {
			return nil
		}

		// Skip the root directory itself
		if path == docsRoot {
			return nil
		}

		stats.FoldersScanned++

		// Get parent directory info
		parentPath := filepath.Dir(path)

		// Try to match as epic folder first
		if epic, matched := s.matchEpicFolder(path, info.Name()); matched {
			// Check if epic.md exists
			epicMdPath := filepath.Join(path, "epic.md")
			if _, err := os.Stat(epicMdPath); err == nil {
				epic.EpicMdPath = &epicMdPath
				stats.FilesAnalyzed++

				// Parse frontmatter to extract custom_folder_path
				if frontmatter, _, err := ParseFrontmatter(epicMdPath); err == nil {
					epic.CustomFolderPath = frontmatter.CustomFolderPath
				}
			}
			epics = append(epics, epic)
			epicFolderNames[path] = info.Name()
			return nil // Continue walking into epic folder
		}

		// Try to match as feature folder (within epic)
		// Check if parent is an epic folder
		if epicFolderName, isEpicParent := epicFolderNames[parentPath]; isEpicParent {
			if feature, matched := s.matchFeatureFolder(path, info.Name(), epicFolderName); matched {
				// Find PRD file
				prdPath, prdFilename := s.findPrdFile(path)
				if prdPath != nil {
					feature.PrdPath = prdPath
					stats.FilesAnalyzed++

					// Parse frontmatter to extract custom_folder_path
					if frontmatter, _, err := ParseFrontmatter(*prdPath); err == nil {
						feature.CustomFolderPath = frontmatter.CustomFolderPath
					}
				}

				// Catalog related documents (exclude PRD file)
				relatedDocs := s.catalogRelatedDocs(path, prdFilename)
				feature.RelatedDocs = relatedDocs
				stats.FilesAnalyzed += len(relatedDocs)

				features = append(features, feature)
			}
		}

		return nil
	})

	if err != nil {
		return nil, nil, stats, fmt.Errorf("walk failed: %w", err)
	}

	return epics, features, stats, nil
}

// matchEpicFolder tries to match folder name against epic patterns
func (s *FolderScanner) matchEpicFolder(fullPath, folderName string) (FolderEpic, bool) {
	result, matched := s.patternMatcher.MatchEpicPattern(folderName)
	if !matched {
		return FolderEpic{}, false
	}

	return FolderEpic{
		Key:  result.EpicID,
		Slug: result.EpicSlug,
		Path: fullPath,
	}, true
}

// matchFeatureFolder tries to match folder name against feature patterns
func (s *FolderScanner) matchFeatureFolder(fullPath, folderName, epicFolderName string) (FolderFeature, bool) {
	// Extract epic key from parent folder name first
	epicResult, epicMatched := s.patternMatcher.MatchEpicPattern(epicFolderName)
	if !epicMatched {
		return FolderFeature{}, false
	}

	parentEpicKey := epicResult.EpicID

	// Match feature pattern
	result, matched := s.patternMatcher.MatchFeaturePattern(folderName, parentEpicKey)
	if !matched {
		return FolderFeature{}, false
	}

	// Build full feature key: E04-F07
	featureKey := result.EpicID + "-" + result.FeatureID

	return FolderFeature{
		Key:     featureKey,
		EpicKey: result.EpicID,
		Slug:    result.FeatureSlug,
		Path:    fullPath,
	}, true
}

// findPrdFile searches for PRD file using feature file patterns
// Returns the full path and filename (so caller can exclude it from related docs)
// Prioritizes exact "prd.md" match first, then tries other patterns
func (s *FolderScanner) findPrdFile(featurePath string) (*string, string) {
	entries, err := os.ReadDir(featurePath)
	if err != nil {
		return nil, ""
	}

	// First pass: look for exact "prd.md" match (highest priority)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if entry.Name() == "prd.md" {
			prdPath := filepath.Join(featurePath, entry.Name())
			return &prdPath, entry.Name()
		}
	}

	// Second pass: try pattern matching for alternative PRD file names
	// (like PRD_F07-name.md)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Skip the fallback pattern that matches everything
		// We only want to match explicit PRD patterns here
		if entry.Name() != "prd.md" && s.patternMatcher.MatchFeatureFilePattern(entry.Name()) {
			// Only match if it starts with "PRD_" (to avoid matching related docs)
			if strings.HasPrefix(entry.Name(), "PRD_") {
				prdPath := filepath.Join(featurePath, entry.Name())
				return &prdPath, entry.Name()
			}
		}
	}

	return nil, ""
}

// catalogRelatedDocs finds all related documents in feature folder
// Excludes the PRD file (prdFilename parameter) from related docs
func (s *FolderScanner) catalogRelatedDocs(featurePath, prdFilename string) []string {
	relatedDocs := []string{}

	entries, err := os.ReadDir(featurePath)
	if err != nil {
		return relatedDocs
	}

	for _, entry := range entries {
		// Skip directories (including tasks/, prps/)
		if entry.IsDir() {
			continue
		}

		// Only include .md files
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Exclude PRD file (by exact filename match)
		if prdFilename != "" && entry.Name() == prdFilename {
			continue
		}

		// Also explicitly exclude common PRD file names as a safety measure
		if entry.Name() == "prd.md" {
			continue
		}

		// Include all other markdown files
		relatedDocs = append(relatedDocs, filepath.Join(featurePath, entry.Name()))
	}

	return relatedDocs
}
