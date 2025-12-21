package keygen

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// PathParser extracts epic and feature keys from file paths
type PathParser struct {
	docsRoot       string
	epicPattern    *regexp.Regexp
	featurePattern *regexp.Regexp
}

// NewPathParser creates a new PathParser with the given docs root
func NewPathParser(docsRoot string) *PathParser {
	return &PathParser{
		docsRoot: docsRoot,
		// Match epic directory: E##-* (e.g., E04-task-mgmt-cli-core)
		epicPattern: regexp.MustCompile(`^(E\d{2})`),
		// Match feature directory: E##-F##-* or E##-P##-F##-* (with optional project number)
		featurePattern: regexp.MustCompile(`^(E\d{2})(-P\d{2})?-(F\d{2})`),
	}
}

// PathComponents represents the parsed components from a file path
type PathComponents struct {
	EpicKey    string
	FeatureKey string
	FilePath   string
}

// ParsePath extracts epic and feature keys from the file path structure
// Expected structure: {docs_root}/{epic_folder}/{feature_folder}/tasks/{file}
// or: {docs_root}/{epic_folder}/{feature_folder}/prps/{file}
func (p *PathParser) ParsePath(filePath string) (*PathComponents, error) {
	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get the directory containing the file
	dir := filepath.Dir(absPath)

	// Get parent directory name (should be 'tasks' or 'prps' folder)
	parentDir := filepath.Base(dir)

	// Check if parent is tasks or prps folder
	var featureDirName string
	if parentDir == "tasks" || parentDir == "prps" {
		// Get grandparent directory (should be feature folder)
		featureDir := filepath.Dir(dir)
		featureDirName = filepath.Base(featureDir)
	} else {
		// Parent directory is the feature folder itself
		featureDirName = parentDir
	}

	// Extract feature key from directory name
	featureMatches := p.featurePattern.FindStringSubmatch(featureDirName)
	if len(featureMatches) < 4 {
		return nil, fmt.Errorf("cannot infer epic/feature from path '%s': expected directory structure like docs/plan/{E##-epic-slug}/{E##-F##-feature-slug}/tasks/{file}", absPath)
	}

	epicKey := featureMatches[1]    // E##
	projectNum := featureMatches[2] // -P## or empty string
	featureNum := featureMatches[3] // F##

	// Build feature key: E##-F## or E##-P##-F## (includes project if present)
	featureKey := epicKey + projectNum + "-" + featureNum

	// Validate epic directory is in the path hierarchy
	// Walk up the directory tree to find epic folder (must match full epic pattern, not just feature folder)
	currentDir := dir
	foundEpic := false
	for i := 0; i < 5; i++ { // Limit search depth to prevent infinite loops
		parentName := filepath.Base(currentDir)
		// Must match epic pattern exactly: E##* where * is not F##
		// This prevents matching feature folders like E04-F02-feature
		epicMatches := p.epicPattern.FindStringSubmatch(parentName)
		if len(epicMatches) > 0 && epicMatches[1] == epicKey {
			// Make sure this isn't a feature folder
			if !strings.Contains(parentName, "-F") {
				foundEpic = true
				break
			}
		}
		currentDir = filepath.Dir(currentDir)
		// Stop if we've reached the root or haven't moved up
		if currentDir == filepath.Dir(currentDir) {
			break
		}
	}

	if !foundEpic {
		return nil, fmt.Errorf("epic folder '%s' not found in path hierarchy for file '%s'", epicKey, absPath)
	}

	return &PathComponents{
		EpicKey:    epicKey,
		FeatureKey: featureKey,
		FilePath:   absPath,
	}, nil
}

// ValidatePath checks if the path structure is valid without fully parsing it
func (p *PathParser) ValidatePath(filePath string) error {
	_, err := p.ParsePath(filePath)
	return err
}
