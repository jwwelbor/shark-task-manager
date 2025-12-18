package sync

import (
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

const (
	// clockSkewBuffer is the acceptable clock skew buffer zone (±60 seconds)
	clockSkewBuffer = 60 * time.Second
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
	return d.DetectConflictsWithSync(fileData, dbTask, nil)
}

// DetectConflictsWithSync detects conflicts considering last sync time
//
// Enhanced conflict detection that checks:
// 1. file.mtime > last_sync_time (file modified since last sync)
// 2. db.updated_at > last_sync_time (DB modified since last sync)
// 3. metadata differs (actual conflict in values)
//
// Conflict is only reported if ALL three conditions are true.
// If lastSyncTime is nil, falls back to basic field comparison (full scan mode).
//
// Clock Skew Handling:
// - Uses ±60 second buffer zone to handle clock differences between systems
// - File/DB timestamps within buffer of last_sync are considered "not modified"
func (d *ConflictDetector) DetectConflictsWithSync(fileData *TaskMetadata, dbTask *models.Task, lastSyncTime *time.Time) []Conflict {
	// If no last sync time, use basic detection (full scan mode)
	if lastSyncTime == nil {
		return d.detectBasicConflicts(fileData, dbTask)
	}

	// Check if file was modified since last sync (with clock skew tolerance)
	fileModified := fileData.ModifiedAt.After(lastSyncTime.Add(-clockSkewBuffer))

	// Check if database was modified since last sync (with clock skew tolerance)
	dbModified := dbTask.UpdatedAt.After(lastSyncTime.Add(-clockSkewBuffer))

	// If only file modified: not a conflict, just a regular file update
	if fileModified && !dbModified {
		// Return only file_path update (not a true conflict)
		return d.detectFilePathConflict(fileData, dbTask)
	}

	// If only DB modified: not a conflict, DB is current (skip file)
	if dbModified && !fileModified {
		return []Conflict{} // No conflicts, DB wins by default
	}

	// If neither modified: no conflicts
	if !fileModified && !dbModified {
		return []Conflict{}
	}

	// Both modified since last sync: check for actual metadata differences
	return d.detectBasicConflicts(fileData, dbTask)
}

// detectBasicConflicts performs basic field-by-field comparison
func (d *ConflictDetector) detectBasicConflicts(fileData *TaskMetadata, dbTask *models.Task) []Conflict {
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
	filePathConflicts := d.detectFilePathConflict(fileData, dbTask)
	conflicts = append(conflicts, filePathConflicts...)

	// Note: We do NOT detect conflicts for database-only fields:
	// - status, priority, agent_type, depends_on, assigned_agent
	// These fields are managed by the database/CLI and never come from files

	return conflicts
}

// detectFilePathConflict checks if file path needs updating
func (d *ConflictDetector) detectFilePathConflict(fileData *TaskMetadata, dbTask *models.Task) []Conflict {
	actualPath := fileData.FilePath

	// Conflict if database path is nil OR differs from actual path
	if dbTask.FilePath == nil || *dbTask.FilePath != actualPath {
		dbValue := ""
		if dbTask.FilePath != nil {
			dbValue = *dbTask.FilePath
		}
		return []Conflict{
			{
				TaskKey:       dbTask.Key,
				Field:         "file_path",
				FileValue:     actualPath,
				DatabaseValue: dbValue,
			},
		}
	}

	return []Conflict{}
}
