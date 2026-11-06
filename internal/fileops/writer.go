// Package fileops provides unified file operations for entity (epic, feature, task) file management.
package fileops

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
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

	// UseAtomicWrite stages and syncs a complete file before publishing it
	// without replacing a file created concurrently.
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

	// CreatedFileInfo identifies the file created by this write, when available
	CreatedFileInfo os.FileInfo
}

// EntityFileWriter handles file writing operations for epics, features, and tasks
type EntityFileWriter struct {
	ioHooks *fileWriterIOHooks
}

type fileWriterIOHooks struct {
	rename func(string, string) error
	write  func(*os.File, []byte) (int, error)
	sync   func(*os.File) error
	link   func(string, string) error
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
	fileExists, statErr := w.checkFileExists(absPath, opts.UseAtomicWrite)
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
		// Stage the complete file, then publish it without replacing a raced-in path.
		fileInfo, linked, err := w.writeFileExclusive(absPath, opts.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to write %s file: %w", opts.EntityType, err)
		}
		if linked {
			if opts.Verbose && opts.Logger != nil {
				opts.Logger(fmt.Sprintf("File already exists, linking to existing %s file: %s",
					opts.EntityType, absPath))
			}
			result.Linked = true
			return result, nil
		}
		result.CreatedFileInfo = fileInfo
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
func (w *EntityFileWriter) checkFileExists(path string, atomic bool) (bool, error) {
	stat := os.Stat
	if atomic {
		stat = os.Lstat
	}
	info, err := stat(path)
	if err == nil {
		if atomic && !info.Mode().IsRegular() {
			return false, fmt.Errorf("existing path is not a regular file: %w", os.ErrInvalid)
		}
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

// writeFileExclusive stages a complete file beside the target and publishes it
// without replacing a path created concurrently.
func (w *EntityFileWriter) writeFileExclusive(path string, data []byte) (os.FileInfo, bool, error) {
	tempPath, fileInfo, err := w.stageEntityFile(filepath.Dir(path), data)
	if err != nil {
		return nil, false, err
	}

	rename := renameNoReplace
	if w.ioHooks != nil && w.ioHooks.rename != nil {
		rename = w.ioHooks.rename
	}
	err = rename(tempPath, path)
	if err == nil {
		return fileInfo, false, nil
	}
	if !errors.Is(err, ErrRenameUnsupported) {
		return w.handlePublicationError(tempPath, path, err)
	}
	link := os.Link
	if w.ioHooks != nil && w.ioHooks.link != nil {
		link = w.ioHooks.link
	}
	if err := link(tempPath, path); err != nil {
		return w.handlePublicationError(tempPath, path, err)
	}

	if err := removeStagedFile(tempPath); err != nil {
		slog.Warn("failed to remove staged entity file after publication", "path", tempPath, "error", err)
	}

	return fileInfo, false, nil
}

// ErrRenameUnsupported indicates native no-replace publication is unavailable for this operation or filesystem.
var ErrRenameUnsupported = errors.New("native no-replace rename unsupported")

func (w *EntityFileWriter) handlePublicationError(tempPath, path string, err error) (os.FileInfo, bool, error) {
	cleanupErr := removeStagedFile(tempPath)
	if errors.Is(err, os.ErrExist) {
		if verifyErr := w.verifyExistingTarget(path); verifyErr != nil {
			return nil, false, errors.Join(verifyErr, cleanupErr)
		}
		if cleanupErr != nil {
			slog.Warn("failed to remove staged entity file after concurrent publication", "path", tempPath, "error", cleanupErr)
		}
		return nil, true, nil
	}
	return nil, false, errors.Join(fmt.Errorf("failed to publish staged file: %w", err), cleanupErr)
}

func (w *EntityFileWriter) verifyExistingTarget(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("failed to verify existing target: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("concurrently published target is not a regular file: %w", os.ErrInvalid)
	}
	return nil
}

const stagedFileNameAttempts = 10

func (w *EntityFileWriter) stageEntityFile(dir string, data []byte) (string, os.FileInfo, error) {
	file, tempPath, err := openStagedFile(dir)
	if err != nil {
		return "", nil, err
	}

	fileInfo, writeErr := w.writeAndSyncStagedFile(file, data)
	closeErr := file.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("failed to close staged file: %w", closeErr)
	}
	if stageErr := errors.Join(writeErr, closeErr); stageErr != nil {
		if cleanupErr := removeStagedFile(tempPath); cleanupErr != nil {
			stageErr = errors.Join(stageErr, fmt.Errorf("failed to remove staged file: %w", cleanupErr))
		}
		return "", nil, stageErr
	}

	return tempPath, fileInfo, nil
}

func openStagedFile(dir string) (*os.File, string, error) {
	for range stagedFileNameAttempts {
		token := make([]byte, 16)
		if _, err := rand.Read(token); err != nil {
			return nil, "", fmt.Errorf("failed to generate staged file name: %w", err)
		}

		path := filepath.Join(dir, ".shark-entity-file-"+hex.EncodeToString(token))
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			return file, path, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, "", fmt.Errorf("failed to create staged file: %w", err)
		}
	}

	return nil, "", fmt.Errorf("failed to create staged file after %d name collisions", stagedFileNameAttempts)
}

func (w *EntityFileWriter) writeAndSyncStagedFile(file *os.File, data []byte) (os.FileInfo, error) {
	var written int
	var err error
	if w.ioHooks != nil && w.ioHooks.write != nil {
		written, err = w.ioHooks.write(file, data)
	} else {
		written, err = file.Write(data)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to write data: %w", err)
	}
	if written != len(data) {
		return nil, fmt.Errorf("failed to write data: %w", io.ErrShortWrite)
	}

	// Sync to disk for durability before publishing the staged file.
	if w.ioHooks != nil && w.ioHooks.sync != nil {
		err = w.ioHooks.sync(file)
	} else {
		err = file.Sync()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to sync file: %w", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat staged file: %w", err)
	}
	return fileInfo, nil
}

func removeStagedFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
