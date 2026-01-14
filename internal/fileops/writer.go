// Package fileops provides unified file operations for entity (epic, feature, task) file management.
package fileops

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteOptions configures behavior for file writing operations
type WriteOptions struct {
	// Content to write to the file
	Content []byte

	// ProjectRoot is the absolute path to the project root directory
	ProjectRoot string

	// FilePath is the file path (absolute or relative to ProjectRoot)
	FilePath string

	// Force determines whether to overwrite existing files
	// If false and file exists, operation is a no-op (links to existing file)
	Force bool

	// CreateIfMissing determines whether to create files that don't exist
	// Only used for tasks (--create flag). If false and file doesn't exist, returns error
	CreateIfMissing bool

	// Verbose enables detailed logging
	Verbose bool

	// EntityType is used for logging (e.g., "epic", "feature", "task")
	EntityType string

	// UseAtomicWrite enables O_EXCL flag for exclusive creation
	// Prevents race conditions when multiple processes try to create same file
	UseAtomicWrite bool

	// Logger is an optional function for verbose output
	// If nil, verbose logging is disabled
	Logger func(message string)
}

// WriteResult contains information about the file write operation
type WriteResult struct {
	// Written indicates whether the file was actually written
	Written bool

	// Linked indicates whether we linked to an existing file instead of writing
	Linked bool

	// AbsolutePath is the absolute path to the file
	AbsolutePath string

	// RelativePath is the path relative to ProjectRoot
	RelativePath string
}

// EntityFileWriter handles file writing operations for epics, features, and tasks
type EntityFileWriter struct {
	// Optional configuration can be stored here if needed
}

// NewEntityFileWriter creates a new file writer instance
func NewEntityFileWriter() *EntityFileWriter {
	return &EntityFileWriter{}
}

// WriteEntityFile writes an entity file according to the provided options
// Returns WriteResult with operation details and any error encountered
func (w *EntityFileWriter) WriteEntityFile(opts WriteOptions) (*WriteResult, error) {
	// 1. Validate file path
	if opts.FilePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// 2. Resolve absolute path
	absPath, relPath, err := w.resolvePaths(opts.FilePath, opts.ProjectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path: %w", err)
	}

	result := &WriteResult{
		AbsolutePath: absPath,
		RelativePath: relPath,
	}

	// 3. Check if file exists
	fileExists, statErr := w.checkFileExists(absPath)
	if statErr != nil {
		// Permission denied or other stat error
		return nil, fmt.Errorf("failed to check file status for %s: %w", relPath, statErr)
	}

	// 4. Handle existing file case
	if fileExists {
		// If Force is set, we should overwrite
		if opts.Force {
			// Delete existing file and continue to write
			if err := os.Remove(absPath); err != nil {
				return nil, fmt.Errorf("failed to remove existing file: %w", err)
			}
			fileExists = false
		} else {
			// Link to existing file
			if opts.Verbose && opts.Logger != nil {
				opts.Logger(fmt.Sprintf("File already exists, linking to existing %s file: %s",
					opts.EntityType, absPath))
			}
			result.Linked = true
			result.Written = false
			return result, nil
		}
	}

	// 5. Handle missing file case
	if !fileExists && !opts.CreateIfMissing && opts.EntityType == "task" {
		// Task-specific behavior: require --create flag for custom files
		return nil, fmt.Errorf("file %q does not exist. Use --create flag to create it", relPath)
	}

	// 6. Create parent directories
	if err := w.ensureParentDir(absPath); err != nil {
		return nil, fmt.Errorf("failed to create parent directories for %s: %w", relPath, err)
	}

	// 7. Write file
	if opts.UseAtomicWrite {
		// Atomic write with O_EXCL flag (race-condition safe)
		if err := w.writeFileExclusive(absPath, opts.Content); err != nil {
			return nil, fmt.Errorf("failed to write %s file: %w", opts.EntityType, err)
		}
	} else {
		// Simple write (backward compatible with epic/feature behavior)
		if err := os.WriteFile(absPath, opts.Content, 0644); err != nil {
			return nil, fmt.Errorf("failed to write %s file: %w", opts.EntityType, err)
		}
	}

	if opts.Verbose && opts.Logger != nil {
		opts.Logger(fmt.Sprintf("Created %s file: %s", opts.EntityType, absPath))
	}

	result.Written = true
	result.Linked = false
	return result, nil
}

// resolvePaths converts a file path to both absolute and relative forms
func (w *EntityFileWriter) resolvePaths(filePath, projectRoot string) (absPath, relPath string, err error) {
	if filepath.IsAbs(filePath) {
		// Already absolute
		absPath = filePath
		relPath, err = filepath.Rel(projectRoot, filePath)
		if err != nil {
			// File is outside project root, use absolute path for both
			relPath = filePath
		}
	} else {
		// Relative path
		relPath = filePath
		absPath = filepath.Join(projectRoot, filePath)
	}

	return absPath, relPath, nil
}

// checkFileExists checks if a file exists at the given path
// Returns (true, nil) if file exists
// Returns (false, nil) if file doesn't exist
// Returns (false, error) for permission errors or other issues
func (w *EntityFileWriter) checkFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	// Permission denied or other error
	return false, err
}

// ensureParentDir creates all parent directories for the given file path
func (w *EntityFileWriter) ensureParentDir(filePath string) error {
	parentDir := filepath.Dir(filePath)
	return os.MkdirAll(parentDir, 0755)
}

// writeFileExclusive writes a file atomically with O_EXCL flag
// Fails if the file already exists (prevents race conditions)
func (w *EntityFileWriter) writeFileExclusive(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("file already exists: %s", path)
		}
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	// Sync to disk for durability
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}
