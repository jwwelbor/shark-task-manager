package sync

import (
	"time"
)

// TaskFileInfo represents metadata about a discovered task file
type TaskFileInfo struct {
	FilePath    string      // Absolute path to file
	FileName    string      // Filename (e.g., T-E04-F07-001.md or 01-setup.md)
	EpicKey     string      // Inferred epic key (e.g., E04)
	FeatureKey  string      // Inferred feature key (e.g., E04-F07 or E04-P01-F02)
	ModifiedAt  time.Time   // File modified timestamp
	PatternType PatternType // Pattern type that matched this file (task or prp)
}

// TaskMetadata represents metadata parsed from a task file
// Used for conflict detection and resolution
type TaskMetadata struct {
	Key          string    // Task key (required)
	Title        string    // Task title (optional in file, conflicts with DB if present)
	Description  *string   // Task description (optional)
	FilePath     string    // Actual file path (used for conflict detection)
	ModifiedAt   time.Time // File modified timestamp (for newer-wins strategy)
}

// ConflictStrategy defines how to resolve conflicts between file and database
type ConflictStrategy string

const (
	// ConflictStrategyFileWins always uses file value when there's a conflict
	ConflictStrategyFileWins ConflictStrategy = "file-wins"

	// ConflictStrategyDatabaseWins always keeps database value when there's a conflict
	ConflictStrategyDatabaseWins ConflictStrategy = "database-wins"

	// ConflictStrategyNewerWins compares timestamps and uses newer value
	ConflictStrategyNewerWins ConflictStrategy = "newer-wins"

	// ConflictStrategyManual prompts user interactively for each conflict
	ConflictStrategyManual ConflictStrategy = "manual"
)

// Conflict represents a detected difference between file and database
type Conflict struct {
	TaskKey       string // Task key that has the conflict
	Field         string // Field name (title, description, file_path)
	FileValue     string // Value from file
	DatabaseValue string // Value from database
}

// SyncOptions contains configuration for sync operations
type SyncOptions struct {
	DBPath        string           // Database file path
	FolderPath    string           // Folder to sync (default: docs/plan)
	DryRun        bool             // Preview changes only
	Strategy      ConflictStrategy // Conflict resolution strategy
	CreateMissing bool             // Auto-create missing epics/features
	Cleanup       bool             // Delete orphaned database tasks
	LastSyncTime  *time.Time       // Last sync time for incremental filtering (nil = full scan)
	ForceFullScan bool             // Force full scan even if LastSyncTime is set
}

// SyncReport contains the results of a sync operation
type SyncReport struct {
	DryRun            bool                   `json:"dry_run"`
	FilesScanned      int                    `json:"files_scanned"`
	FilesFiltered     int                    `json:"files_filtered"`     // Files processed after incremental filtering
	FilesSkipped      int                    `json:"files_skipped"`      // Files skipped due to incremental filtering
	TasksImported     int                    `json:"tasks_imported"`
	TasksUpdated      int                    `json:"tasks_updated"`
	TasksDeleted      int                    `json:"tasks_deleted"`
	ConflictsResolved int                    `json:"conflicts_resolved"`
	KeysGenerated     int                    `json:"keys_generated"`     // Number of task keys generated for PRP files
	PatternMatches    map[string]int         `json:"pattern_matches"`    // Count of files matched by each pattern
	Warnings          []string               `json:"warnings"`
	Errors            []string               `json:"errors"`
	Conflicts         []Conflict             `json:"conflicts"`
}
