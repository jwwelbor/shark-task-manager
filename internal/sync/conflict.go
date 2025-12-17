package sync

import (
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ConflictDetector compares file metadata with database records to detect conflicts
type ConflictDetector struct {
	// No state needed
}

// NewConflictDetector creates a new ConflictDetector
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{}
}

// DetectConflicts compares file metadata with database record field-by-field
// Returns list of detected conflicts for metadata fields only
//
// Conflict Detection Rules:
// - Title: Conflict if file has title (non-empty) AND differs from database
// - Description: Conflict if both file and database have description AND they differ
// - File Path: Conflict if database path is nil OR differs from actual file path
// - Database-only fields (status, priority, agent_type, depends_on, assigned_agent): Never conflict
func (d *ConflictDetector) DetectConflicts(fileData *TaskMetadata, dbTask *models.Task) []Conflict {
	conflicts := []Conflict{}

	// 1. Title conflict: only if file has title AND differs from database
	if fileData.Title != "" && fileData.Title != dbTask.Title {
		conflicts = append(conflicts, Conflict{
			TaskKey:       dbTask.Key,
			Field:         "title",
			FileValue:     fileData.Title,
			DatabaseValue: dbTask.Title,
		})
	}

	// 2. Description conflict: only if both exist AND differ
	if fileData.Description != nil && dbTask.Description != nil {
		if *fileData.Description != *dbTask.Description {
			conflicts = append(conflicts, Conflict{
				TaskKey:       dbTask.Key,
				Field:         "description",
				FileValue:     *fileData.Description,
				DatabaseValue: *dbTask.Description,
			})
		}
	}

	// 3. File path conflict: always update to actual location
	// Conflict if database path is nil OR differs from actual path
	actualPath := fileData.FilePath
	if dbTask.FilePath == nil || *dbTask.FilePath != actualPath {
		dbValue := ""
		if dbTask.FilePath != nil {
			dbValue = *dbTask.FilePath
		}
		conflicts = append(conflicts, Conflict{
			TaskKey:       dbTask.Key,
			Field:         "file_path",
			FileValue:     actualPath,
			DatabaseValue: dbValue,
		})
	}

	// Note: We do NOT detect conflicts for database-only fields:
	// - status, priority, agent_type, depends_on, assigned_agent
	// These fields are managed by the database/CLI and never come from files

	return conflicts
}
