package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// maxFileSize is the maximum allowed file size (1MB) to prevent DoS
	maxFileSize = 1 * 1024 * 1024
)

// FileScanner recursively scans directories for task markdown files
type FileScanner struct {
	patternRegistry *PatternRegistry
	featurePattern  *regexp.Regexp
	epicPattern     *regexp.Regexp
	keyPattern      *regexp.Regexp
}

// NewFileScanner creates a new FileScanner instance with default patterns (task only)
// This maintains backward compatibility
func NewFileScanner() *FileScanner {
	registry := NewPatternRegistry()
	// Task pattern is enabled by default, PRP is disabled
	return &FileScanner{
		patternRegistry: registry,
		// Match feature directory: E##-F##-* or E##-P##-F##-* (with optional project number)
		featurePattern: regexp.MustCompile(`^(E\d{2})(-P\d{2})?-(F\d{2})`),
		// Match epic directory: E##-*
		epicPattern: regexp.MustCompile(`^(E\d{2})`),
		// Extract keys from filename: T-E##-F##-###.md (strict format, no suffix)
		keyPattern: regexp.MustCompile(`^T-(E\d{2})-(F\d{2})-\d{3}\.md$`),
	}
}

// NewFileScannerWithPatterns creates a FileScanner with specific patterns enabled
func NewFileScannerWithPatterns(patterns []PatternType) *FileScanner {
	scanner := NewFileScanner()
	if err := scanner.patternRegistry.SetActivePatterns(patterns); err != nil {
		// Fall back to default if invalid patterns
		return NewFileScanner()
	}
	return scanner
}

// Scan recursively scans directory for task markdown files
// Returns list of TaskFileInfo with metadata for each discovered file
func (s *FileScanner) Scan(rootPath string) ([]TaskFileInfo, error) {
	var files []TaskFileInfo

	// Verify rootPath exists
	if _, err := os.Stat(rootPath); err != nil {
		return nil, fmt.Errorf("root path does not exist: %w", err)
	}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log warning but continue scanning
			// This handles permission errors gracefully
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if filename matches any enabled pattern
		patternType, matches := s.getMatchingPattern(info.Name())
		if !matches {
			return nil
		}

		// Validate file is regular (not symlink, device, etc.)
		if err := s.validateFileIsRegular(path); err != nil {
			// Log warning but continue (skip this file)
			return nil
		}

		// Validate file size
		if err := s.validateFileSize(path); err != nil {
			// Log warning but continue (skip oversized files)
			return nil
		}

		// Get absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}

		// Infer epic and feature from path
		epicKey, featureKey, err := s.inferEpicFeature(path)
		if err != nil {
			// If inference fails, we still add the file but with empty keys
			// The sync engine can handle this by extracting from filename
			epicKey = ""
			featureKey = ""
		}

		// Add to results
		files = append(files, TaskFileInfo{
			FilePath:    absPath,
			FileName:    info.Name(),
			EpicKey:     epicKey,
			FeatureKey:  featureKey,
			ModifiedAt:  info.ModTime(),
			PatternType: patternType,
		})

		return nil
	})

	return files, err
}

// getMatchingPattern checks if filename matches any enabled pattern
// Returns the pattern type and whether a match was found
func (s *FileScanner) getMatchingPattern(filename string) (PatternType, bool) {
	return s.patternRegistry.MatchesAnyPattern(filename)
}

// isTaskFile checks if filename matches task file pattern (T-E##-F##-###.md)
// Deprecated: Use getMatchingPattern() instead
func (s *FileScanner) isTaskFile(filename string) bool {
	patternType, matches := s.getMatchingPattern(filename)
	return matches && patternType == PatternTypeTask
}

// inferEpicFeature infers epic and feature keys from file path structure
// Returns epic key, feature key, and error if inference fails
// Supports both E##-F##-* and E##-P##-F##-* patterns (with optional project number)
func (s *FileScanner) inferEpicFeature(filePath string) (string, string, error) {
	// Get directory containing the file
	dir := filepath.Dir(filePath)

	// Get parent directory name (should be feature folder)
	parentDir := filepath.Base(dir)

	// Try to extract feature key from parent directory
	// Pattern: E##-F##-* or E##-P##-F##-* (e.g., E04-F07-initialization-sync or E09-P02-F01-character-management)
	// Captures: [1]=E##, [2]=-P## (optional), [3]=F##
	if matches := s.featurePattern.FindStringSubmatch(parentDir); len(matches) >= 4 {
		epicKey := matches[1]   // E##
		projectNum := matches[2] // -P## or empty string
		featureNum := matches[3] // F##

		// Build feature key: E##-F## or E##-P##-F## (includes project if present)
		featureKey := epicKey + projectNum + "-" + featureNum
		return epicKey, featureKey, nil
	}

	// If parent directory doesn't match, try grandparent (might be nested in tasks/ or prps/ folder)
	grandparentDir := filepath.Base(filepath.Dir(dir))
	if matches := s.featurePattern.FindStringSubmatch(grandparentDir); len(matches) >= 4 {
		epicKey := matches[1]
		projectNum := matches[2]
		featureNum := matches[3]
		featureKey := epicKey + projectNum + "-" + featureNum
		return epicKey, featureKey, nil
	}

	// Fallback: Extract from filename (only works for task pattern, not PRP)
	// This handles legacy folder structures (docs/tasks/todo/T-E##-F##-###.md)
	filename := filepath.Base(filePath)
	epicKey, featureKey := s.extractKeyFromFilename(filename)

	if epicKey != "" && featureKey != "" {
		return epicKey, featureKey, nil
	}

	return "", "", fmt.Errorf("could not infer epic/feature from path: %s", filePath)
}

// extractKeyFromFilename extracts epic and feature keys from task filename
// Example: T-E04-F07-001.md -> ("E04", "E04-F07")
func (s *FileScanner) extractKeyFromFilename(filename string) (string, string) {
	matches := s.keyPattern.FindStringSubmatch(filename)
	if len(matches) < 3 {
		return "", ""
	}

	epicKey := matches[1]                   // E##
	featureKey := matches[1] + "-" + matches[2] // E##-F##

	return epicKey, featureKey
}

// validateFilePath ensures file path is within allowed directories
// This prevents path traversal attacks
func (s *FileScanner) validateFilePath(filePath string, rootDir string) error {
	// Convert to absolute paths for comparison
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return fmt.Errorf("invalid root directory: %w", err)
	}

	// Clean paths to resolve any .. or . components
	absPath = filepath.Clean(absPath)
	absRoot = filepath.Clean(absRoot)

	// Check if file is within root directory first
	if !strings.HasPrefix(absPath, absRoot) {
		return fmt.Errorf("file path outside root directory: %s", absPath)
	}

	// Define allowed subdirectories within root
	allowedDirs := []string{
		filepath.Join(absRoot, "docs", "plan"),
		filepath.Join(absRoot, "docs", "tasks"),
		filepath.Join(absRoot, "shark-templates"),
	}

	// Check if file is within any allowed subdirectory
	for _, allowedDir := range allowedDirs {
		if strings.HasPrefix(absPath, allowedDir) {
			return nil
		}
	}

	// If we're scanning a specific subdirectory (not root), check if it's under allowed dirs
	// This handles cases where rootDir is already docs/plan/E04-epic/E04-F07-feature
	for _, allowedDir := range allowedDirs {
		if strings.HasPrefix(absRoot, allowedDir) && strings.HasPrefix(absPath, absRoot) {
			return nil
		}
	}

	return fmt.Errorf("file path outside allowed directories: %s", absPath)
}

// validateFileSize ensures file size is within limits (prevents DoS)
func (s *FileScanner) validateFileSize(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > maxFileSize {
		return fmt.Errorf("file size exceeds limit: %d bytes (max %d)", info.Size(), maxFileSize)
	}

	return nil
}

// validateFileIsRegular ensures file is a regular file (not symlink, device, etc.)
// This is a security measure to prevent symlink attacks
func (s *FileScanner) validateFileIsRegular(filePath string) error {
	// Use Lstat to not follow symlinks
	info, err := os.Lstat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if file is a symlink
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlinks are not allowed: %s", filePath)
	}

	// Check if file is a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", filePath)
	}

	return nil
}
