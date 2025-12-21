package sync

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

const (
	// clockSkewTolerance is the acceptable clock skew in seconds (±60 seconds)
	clockSkewTolerance = 60 * time.Second
)

// IncrementalFilter filters files based on modification time
type IncrementalFilter struct {
	taskRepo *repository.TaskRepository
}

// NewIncrementalFilter creates a new IncrementalFilter
func NewIncrementalFilter(taskRepo *repository.TaskRepository) *IncrementalFilter {
	return &IncrementalFilter{
		taskRepo: taskRepo,
	}
}

// FilterOptions contains options for file filtering
type FilterOptions struct {
	LastSyncTime  *time.Time // nil triggers full scan
	ForceFullScan bool       // Bypass incremental filtering
}

// FilterResult contains statistics about the filtering operation
type FilterResult struct {
	TotalFiles    int
	FilteredFiles int
	SkippedFiles  int
	NewFiles      int
	Warnings      []string
}

// Filter filters a list of files based on modification time
// Returns only files that have been modified since lastSyncTime
func (f *IncrementalFilter) Filter(ctx context.Context, files []TaskFileInfo, opts FilterOptions) ([]TaskFileInfo, *FilterResult, error) {
	result := &FilterResult{
		TotalFiles: len(files),
		Warnings:   []string{},
	}

	// If force full scan is enabled, return all files
	if opts.ForceFullScan {
		result.FilteredFiles = len(files)
		return files, result, nil
	}

	// If no last sync time, perform full scan
	if opts.LastSyncTime == nil {
		result.FilteredFiles = len(files)
		return files, result, nil
	}

	lastSyncTime := *opts.LastSyncTime

	// Get all existing task file paths from database for "new file" detection
	existingFiles, err := f.getExistingFilePaths(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get existing files from database: %w", err)
	}

	var filteredFiles []TaskFileInfo

	for _, file := range files {
		// Check if file is new (not in database)
		isNewFile := !existingFiles[file.FilePath]

		// If file is new, always include it
		if isNewFile {
			filteredFiles = append(filteredFiles, file)
			result.NewFiles++
			result.FilteredFiles++
			continue
		}

		// Get file modification time
		fileInfo, err := os.Stat(file.FilePath)
		if err != nil {
			// File doesn't exist or can't be accessed - skip it
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Cannot stat file %s: %v", file.FilePath, err))
			result.SkippedFiles++
			continue
		}

		mtime := fileInfo.ModTime()

		// Handle clock skew (future mtime)
		if mtime.After(time.Now().Add(clockSkewTolerance)) {
			// File mtime is significantly in the future (>60 seconds)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("File %s has future mtime, possible clock skew", file.FilePath))
			// Still process the file (treat as changed)
		}

		// Compare mtime with last sync time
		// Include file if mtime > lastSyncTime (file modified since last sync)
		if mtime.After(lastSyncTime) {
			filteredFiles = append(filteredFiles, file)
			result.FilteredFiles++
		} else {
			// File unchanged since last sync - skip it
			result.SkippedFiles++
		}
	}

	// Log filtering statistics
	if opts.LastSyncTime != nil {
		log.Printf("Incremental filter: %d total files, %d changed, %d skipped, %d new",
			result.TotalFiles, result.FilteredFiles, result.SkippedFiles, result.NewFiles)
	}

	return filteredFiles, result, nil
}

// getExistingFilePaths queries the database for all known task file paths
// Returns a map for O(1) lookup: map[filePath]true
func (f *IncrementalFilter) getExistingFilePaths(ctx context.Context) (map[string]bool, error) {
	// Query all tasks that have a file_path set
	tasks, err := f.taskRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks from database: %w", err)
	}

	// Build lookup map
	existingFiles := make(map[string]bool, len(tasks))
	for _, task := range tasks {
		if task.FilePath != nil && *task.FilePath != "" {
			existingFiles[*task.FilePath] = true
		}
	}

	return existingFiles, nil
}
