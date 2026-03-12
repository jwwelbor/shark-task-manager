package db

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB initializes the SQLite database with complete schema
func InitDB(filepath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filepath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure SQLite for optimal performance and data integrity
	if err := configureSQLite(db); err != nil {
		return nil, fmt.Errorf("failed to configure SQLite: %w", err)
	}

	// Use version-checked migration: skip all DDL when schema is already current.
	// This avoids running 22+ migration checks on every command invocation.
	// Fresh installs (no schema_version table) still apply everything automatically.
	if _, err := ApplySchemaIfNeeded(db); err != nil {
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	}

	return db, nil
}

// configureSQLite sets SQLite PRAGMA settings for optimal operation
func configureSQLite(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",       // Enable foreign key constraints
		"PRAGMA journal_mode = WAL;",      // Use Write-Ahead Logging for better concurrency
		"PRAGMA busy_timeout = 5000;",     // 5 second timeout for locks
		"PRAGMA synchronous = NORMAL;",    // Balance safety and performance
		"PRAGMA cache_size = -64000;",     // 64MB cache
		"PRAGMA temp_store = MEMORY;",     // Store temp tables in memory
		"PRAGMA mmap_size = 30000000000;", // Use memory-mapped I/O
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute %q: %w", pragma, err)
		}
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&fkEnabled); err != nil {
		return fmt.Errorf("failed to verify foreign_keys: %w", err)
	}
	if fkEnabled != 1 {
		return fmt.Errorf("foreign_keys not enabled")
	}

	return nil
}

// createSchema creates all tables, indexes, and triggers
func createSchema(db *sql.DB) error {
	// First, create tables and triggers without indexes on new columns
	// These new column indexes will be created after migrations add the columns
	schema := `
-- ============================================================================
-- Table: epics
-- ============================================================================
CREATE TABLE IF NOT EXISTS epics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    priority TEXT NOT NULL CHECK (priority IN ('high', 'medium', 'low')),
    business_value TEXT CHECK (business_value IN ('high', 'medium', 'low')),
    file_path TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for epics (basic indexes only - new column indexes created after migrations)
CREATE UNIQUE INDEX IF NOT EXISTS idx_epics_key ON epics(key);
CREATE INDEX IF NOT EXISTS idx_epics_status ON epics(status);

-- Trigger to auto-update updated_at for epics
CREATE TRIGGER IF NOT EXISTS epics_updated_at
AFTER UPDATE ON epics
FOR EACH ROW
BEGIN
    UPDATE epics SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- ============================================================================
-- Table: features
-- ============================================================================
CREATE TABLE IF NOT EXISTS features (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    epic_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    progress_pct REAL NOT NULL DEFAULT 0.0 CHECK (progress_pct >= 0.0 AND progress_pct <= 100.0),
    execution_order INTEGER NULL,
    file_path TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE
);

-- Indexes for features (basic indexes only - new column indexes created after migrations)
CREATE UNIQUE INDEX IF NOT EXISTS idx_features_key ON features(key);
CREATE INDEX IF NOT EXISTS idx_features_epic_id ON features(epic_id);
CREATE INDEX IF NOT EXISTS idx_features_status ON features(status);

-- Trigger to auto-update updated_at for features
CREATE TRIGGER IF NOT EXISTS features_updated_at
AFTER UPDATE ON features
FOR EACH ROW
BEGIN
    UPDATE features SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- ============================================================================
-- Table: tasks
-- ============================================================================
CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    feature_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    agent_type TEXT,
    priority INTEGER NOT NULL DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    depends_on TEXT,
    assigned_agent TEXT,
    file_path TEXT,
    blocked_reason TEXT,
    execution_order INTEGER NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    blocked_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE
);

-- Indexes for tasks
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_key ON tasks(key);
CREATE INDEX IF NOT EXISTS idx_tasks_feature_id ON tasks(feature_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_agent_type ON tasks(agent_type);
CREATE INDEX IF NOT EXISTS idx_tasks_status_priority ON tasks(status, priority);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);
CREATE INDEX IF NOT EXISTS idx_tasks_file_path ON tasks(file_path);

-- Trigger to auto-update updated_at for tasks
CREATE TRIGGER IF NOT EXISTS tasks_updated_at
AFTER UPDATE ON tasks
FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- ============================================================================
-- Table: task_history
-- ============================================================================
CREATE TABLE IF NOT EXISTS task_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    old_status TEXT,
    new_status TEXT NOT NULL,
    agent TEXT,
    notes TEXT,
    forced BOOLEAN DEFAULT FALSE,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Indexes for task_history
CREATE INDEX IF NOT EXISTS idx_task_history_task_id ON task_history(task_id);
CREATE INDEX IF NOT EXISTS idx_task_history_timestamp ON task_history(timestamp DESC);

-- ============================================================================
-- Table: task_notes
-- ============================================================================
CREATE TABLE IF NOT EXISTS task_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    note_type TEXT CHECK (note_type IN (
        'comment',         -- General observation
        'decision',        -- Why we chose X over Y
        'blocker',         -- What's blocking progress
        'solution',        -- How we solved a problem
        'reference',       -- External links, documentation
        'implementation',  -- What we actually built
        'testing',         -- Test results, coverage
        'future',          -- Future improvements / TODO
        'question',        -- Unanswered questions
        'rejection'        -- Rejection reason for backward transitions
    )) NOT NULL,
    content TEXT NOT NULL,
    created_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Indexes for task_notes
CREATE INDEX IF NOT EXISTS idx_task_notes_task_id ON task_notes(task_id);
CREATE INDEX IF NOT EXISTS idx_task_notes_type ON task_notes(note_type);
CREATE INDEX IF NOT EXISTS idx_task_notes_created_at ON task_notes(created_at);
CREATE INDEX IF NOT EXISTS idx_task_notes_task_type ON task_notes(task_id, note_type);

-- ============================================================================
-- Table: task_relationships
-- ============================================================================
CREATE TABLE IF NOT EXISTS task_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_task_id INTEGER NOT NULL,
    to_task_id INTEGER NOT NULL,
    relationship_type TEXT CHECK (relationship_type IN (
        'depends_on',    -- Task from_task depends on to_task completing (hard dependency)
        'blocks',        -- Task from_task blocks to_task from proceeding (explicit blocker)
        'related_to',    -- Tasks share common code/concerns (soft relationship)
        'follows',       -- Task from_task naturally follows to_task (sequence, not blocking)
        'spawned_from',  -- Task from_task was created from UAT/bugs in to_task
        'duplicates',    -- Tasks represent duplicate work (should merge)
        'references'     -- Task from_task consults/uses output of to_task
    )) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (from_task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (to_task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    UNIQUE(from_task_id, to_task_id, relationship_type)
);

-- Indexes for task_relationships (bidirectional queries)
CREATE INDEX IF NOT EXISTS idx_task_relationships_from ON task_relationships(from_task_id);
CREATE INDEX IF NOT EXISTS idx_task_relationships_to ON task_relationships(to_task_id);
CREATE INDEX IF NOT EXISTS idx_task_relationships_type ON task_relationships(relationship_type);
CREATE INDEX IF NOT EXISTS idx_task_relationships_from_type ON task_relationships(from_task_id, relationship_type);
CREATE INDEX IF NOT EXISTS idx_task_relationships_to_type ON task_relationships(to_task_id, relationship_type);

-- ============================================================================
-- Table: documents
-- ============================================================================
CREATE TABLE IF NOT EXISTS documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    file_path TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(title, file_path)
);

-- Indexes for documents
CREATE INDEX IF NOT EXISTS idx_documents_title ON documents(title);
CREATE INDEX IF NOT EXISTS idx_documents_file_path ON documents(file_path);

-- ============================================================================
-- Table: epic_documents
-- ============================================================================
CREATE TABLE IF NOT EXISTS epic_documents (
    epic_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (epic_id, document_id),
    FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

-- Indexes for epic_documents
CREATE INDEX IF NOT EXISTS idx_epic_documents_epic_id ON epic_documents(epic_id);
CREATE INDEX IF NOT EXISTS idx_epic_documents_document_id ON epic_documents(document_id);

-- ============================================================================
-- Table: feature_documents
-- ============================================================================
CREATE TABLE IF NOT EXISTS feature_documents (
    feature_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (feature_id, document_id),
    FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

-- Indexes for feature_documents
CREATE INDEX IF NOT EXISTS idx_feature_documents_feature_id ON feature_documents(feature_id);
CREATE INDEX IF NOT EXISTS idx_feature_documents_document_id ON feature_documents(document_id);

-- ============================================================================
-- Table: task_documents
-- ============================================================================
CREATE TABLE IF NOT EXISTS task_documents (
    task_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (task_id, document_id),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

-- Indexes for task_documents
CREATE INDEX IF NOT EXISTS idx_task_documents_task_id ON task_documents(task_id);
CREATE INDEX IF NOT EXISTS idx_task_documents_document_id ON task_documents(document_id);

-- ============================================================================
-- Table: ideas
-- ============================================================================
CREATE TABLE IF NOT EXISTS ideas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,                          -- Format: I-YYYY-MM-DD-xx
    title TEXT NOT NULL,
    description TEXT,
    created_date TIMESTAMP NOT NULL,                   -- Date for key generation
    priority INTEGER CHECK (priority >= 1 AND priority <= 10),
    display_order INTEGER,                             -- Order for sorting ideas
    notes TEXT,
    related_docs TEXT,                                 -- JSON array of document paths
    dependencies TEXT,                                 -- JSON array of idea keys
    status TEXT NOT NULL CHECK (status IN ('new', 'on_hold', 'converted', 'archived')) DEFAULT 'new',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Conversion tracking (for E08-F03)
    converted_to_type TEXT CHECK (converted_to_type IN ('epic', 'feature', 'task')),
    converted_to_key TEXT,
    converted_at TIMESTAMP
);

-- Indexes for ideas
CREATE UNIQUE INDEX IF NOT EXISTS idx_ideas_key ON ideas(key);
CREATE INDEX IF NOT EXISTS idx_ideas_status ON ideas(status);
CREATE INDEX IF NOT EXISTS idx_ideas_created_date ON ideas(created_date DESC);
CREATE INDEX IF NOT EXISTS idx_ideas_priority ON ideas(priority);

-- Trigger to auto-update updated_at for ideas
CREATE TRIGGER IF NOT EXISTS ideas_updated_at
AFTER UPDATE ON ideas
FOR EACH ROW
BEGIN
    UPDATE ideas SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
`

	_, err := db.Exec(schema)
	return err
}

// ApplySchemaAndMigrations applies the database schema and migrations to an existing connection.
// This is used for Turso/cloud databases where the connection is already established.
// For local SQLite databases, use InitDB() instead which handles opening the connection.
func ApplySchemaAndMigrations(db *sql.DB) error {
	// Note: configureSQLite() is skipped for Turso as some PRAGMAs may not be supported
	// Turso handles configuration server-side

	// Create all tables, indexes, and triggers
	if err := createSchema(db); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Run migrations for backwards compatibility
	if err := runMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Record schema version after successful apply
	if err := setSchemaVersion(db, CurrentSchemaVersion); err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}

	return nil
}

// CurrentSchemaVersion is incremented whenever schema or migrations change.
// Bump this when adding new tables, columns, indexes, or migrations.
const CurrentSchemaVersion = 6

// ApplySchemaIfNeeded checks the schema version and only applies schema/migrations
// if the database is not at the current version. This avoids ~2s of DDL overhead
// on cloud databases (Turso) where each statement is a network round trip.
// Returns true if schema was applied, false if skipped.
func ApplySchemaIfNeeded(db *sql.DB) (bool, error) {
	version, err := getSchemaVersion(db)
	if err != nil {
		// If we can't read the version (table doesn't exist yet), apply everything
		if err2 := ApplySchemaAndMigrations(db); err2 != nil {
			return false, err2
		}
		return true, nil
	}

	if version >= CurrentSchemaVersion {
		return false, nil // Already up to date
	}

	// Version is behind, apply schema and migrations
	if err := ApplySchemaAndMigrations(db); err != nil {
		return false, err
	}
	return true, nil
}

// getSchemaVersion reads the current schema version from the database.
// Returns an error if the schema_version table doesn't exist.
func getSchemaVersion(db *sql.DB) (int, error) {
	var version int
	err := db.QueryRow(`SELECT version FROM schema_version ORDER BY version DESC LIMIT 1`).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to read schema version: %w", err)
	}
	return version, nil
}

// setSchemaVersion records the current schema version in the database.
// Creates the schema_version table if it doesn't exist.
func setSchemaVersion(db *sql.DB, version int) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER NOT NULL,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}

	_, err = db.Exec(`INSERT INTO schema_version (version) VALUES (?)`, version)
	if err != nil {
		return fmt.Errorf("failed to insert schema version: %w", err)
	}

	return nil
}

// CheckIntegrity runs PRAGMA integrity_check on the database
func CheckIntegrity(db *sql.DB) error {
	var result string
	if err := db.QueryRow("PRAGMA integrity_check;").Scan(&result); err != nil {
		return fmt.Errorf("failed to run integrity_check: %w", err)
	}
	if result != "ok" {
		return fmt.Errorf("database integrity check failed: %s", result)
	}
	return nil
}

// runMigrations runs all pending migrations for backwards compatibility
func runMigrations(db *sql.DB) error {
	// Check if epics table has file_path column; if not, add it
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'file_path'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check epics schema: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE epics ADD COLUMN file_path TEXT;`); err != nil {
			return fmt.Errorf("failed to add file_path to epics: %w", err)
		}
		if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_epics_file_path ON epics(file_path);`); err != nil {
			return fmt.Errorf("failed to create epics file_path index: %w", err)
		}
	}

	// Check if features table has file_path column; if not, add it
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'file_path'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check features schema: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE features ADD COLUMN file_path TEXT;`); err != nil {
			return fmt.Errorf("failed to add file_path to features: %w", err)
		}
		if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_features_file_path ON features(file_path);`); err != nil {
			return fmt.Errorf("failed to create features file_path index: %w", err)
		}
	}

	// Check if tasks table has file_path column; if not, add it
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name = 'file_path'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check tasks schema: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN file_path TEXT;`); err != nil {
			return fmt.Errorf("failed to add file_path to tasks: %w", err)
		}
		if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_file_path ON tasks(file_path);`); err != nil {
			return fmt.Errorf("failed to create tasks file_path index: %w", err)
		}
	}

	// Check if tasks table has execution_order column; if not, add it
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name = 'execution_order'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check tasks schema for execution_order: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN execution_order INTEGER NULL;`); err != nil {
			return fmt.Errorf("failed to add execution_order to tasks: %w", err)
		}
	}

	// Check if features table has execution_order column; if not, add it
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'execution_order'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check features schema for execution_order: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE features ADD COLUMN execution_order INTEGER NULL;`); err != nil {
			return fmt.Errorf("failed to add execution_order to features: %w", err)
		}
	}

	// NOTE: custom_folder_path columns removed in E07-F19 (File Path Flag Standardization)
	// Migration that previously added these columns has been removed.
	// See migrateDropCustomFolderPath() for the removal migration.

	// Migrate slug columns for E07-F11
	if err := migrateSlugColumns(db); err != nil {
		return fmt.Errorf("failed to migrate slug columns: %w", err)
	}

	// Create indexes on new columns that might not have existed before
	// These are created here after migrations ensure the columns exist
	// NOTE: custom_folder_path indexes removed in E07-F19
	newIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_epics_file_path ON epics(file_path);`,
		`CREATE INDEX IF NOT EXISTS idx_features_file_path ON features(file_path);`,
		`CREATE INDEX IF NOT EXISTS idx_epics_slug ON epics(slug);`,
		`CREATE INDEX IF NOT EXISTS idx_features_slug ON features(slug);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_slug ON tasks(slug);`,
	}

	for _, idx := range newIndexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Run document tables migration
	if err := migrateDocumentTables(db); err != nil {
		return fmt.Errorf("failed to migrate document tables: %w", err)
	}

	// Run completion metadata migration
	if err := migrateCompletionMetadata(db); err != nil {
		return fmt.Errorf("failed to migrate completion metadata: %w", err)
	}

	// Run task criteria and search migration
	if err := migrateTaskCriteriaAndSearch(db); err != nil {
		return fmt.Errorf("failed to migrate task criteria and search: %w", err)
	}

	// Run work sessions and context data migration
	if err := migrateWorkSessionsAndContext(db); err != nil {
		return fmt.Errorf("failed to migrate work sessions and context data: %w", err)
	}

	// Run status CHECK constraint removal migration
	// This allows workflow-defined statuses from config instead of hardcoded values
	if err := MigrateRemoveStatusCheckConstraints(db); err != nil {
		return fmt.Errorf("failed to remove status CHECK constraints: %w", err)
	}

	// Run task_history foreign key fix migration
	// This fixes databases where the tasks table was migrated but task_history
	// still references the old "tasks_old" table
	if err := migrateTaskHistoryForeignKey(db); err != nil {
		return fmt.Errorf("failed to fix task_history foreign key: %w", err)
	}

	// Run features_old foreign key fix migration
	// This fixes databases where tasks or feature_documents still reference
	// the old "features_old" table instead of "features"
	if err := MigrateFixFeaturesOldForeignKeys(db); err != nil {
		return fmt.Errorf("failed to fix features_old foreign keys: %w", err)
	}

	// Run task_notes foreign key fix migration
	// This fixes databases where task_notes references tasks_old
	needsTaskNotesFix, err := needsTaskNotesFKFix(db)
	if err != nil {
		return fmt.Errorf("failed to check if task_notes FK fix needed: %w", err)
	}
	if needsTaskNotesFix {
		if err := fixTaskNotesTasksOldFK(db); err != nil {
			return fmt.Errorf("failed to fix task_notes foreign key: %w", err)
		}
	}

	// Run task_criteria foreign key fix migration
	// This fixes databases where task_criteria references tasks_old
	needsTaskCriteriaFix, err := needsTaskCriteriaFKFix(db)
	if err != nil {
		return fmt.Errorf("failed to check if task_criteria FK fix needed: %w", err)
	}
	if needsTaskCriteriaFix {
		if err := fixTaskCriteriaTasksOldFK(db); err != nil {
			return fmt.Errorf("failed to fix task_criteria foreign key: %w", err)
		}
	}

	// Run work_sessions foreign key fix migration
	// This fixes databases where work_sessions references tasks_old
	needsWorkSessionsFix, err := needsWorkSessionsFKFix(db)
	if err != nil {
		return fmt.Errorf("failed to check if work_sessions FK fix needed: %w", err)
	}
	if needsWorkSessionsFix {
		if err := fixWorkSessionsTasksOldFK(db); err != nil {
			return fmt.Errorf("failed to fix work_sessions foreign key: %w", err)
		}
	}

	// Run status_override column migration for cascading status calculation (E07-F14)
	if err := migrateStatusOverrideColumn(db); err != nil {
		return fmt.Errorf("failed to migrate status_override column: %w", err)
	}

	// Run ideas table order column rename migration (E08-F02)
	if err := migrateIdeasOrderColumn(db); err != nil {
		return fmt.Errorf("failed to migrate ideas order column: %w", err)
	}

	// Run custom_folder_path column removal migration (E07-F19)
	if err := migrateDropCustomFolderPath(db); err != nil {
		return fmt.Errorf("failed to drop custom_folder_path columns: %w", err)
	}

	// Run task_notes metadata column migration (E07-F22)
	if err := migrateTaskNotesMetadata(db); err != nil {
		return fmt.Errorf("failed to migrate task_notes metadata: %w", err)
	}

	// Run task_history rejection_reason column migration (E07-F22)
	if err := migrateTaskHistoryRejectionReason(db); err != nil {
		return fmt.Errorf("failed to migrate task_history rejection_reason: %w", err)
	}

	// Run task_documents link_type column migration (E07-F22)
	if err := migrateTaskDocumentsLinkType(db); err != nil {
		return fmt.Errorf("failed to migrate task_documents link_type: %w", err)
	}

	// Run task_notes note_type CHECK constraint migration to include 'rejection' (E07-F22)
	if err := migrateTaskNotesNoteTypeConstraint(db); err != nil {
		return fmt.Errorf("failed to migrate task_notes note_type constraint: %w", err)
	}

	// Run entity_notes migration (E16-F04) - polymorphic notes table
	if err := migrateEntityNotes(db); err != nil {
		return fmt.Errorf("failed to migrate entity_notes: %w", err)
	}

	// Run context_data columns migration for epics and features (E16-F04)
	if err := migrateEpicFeatureContextData(db); err != nil {
		return fmt.Errorf("failed to migrate epic/feature context_data: %w", err)
	}

	// Run feature_relationships and epic_relationships table migrations (E07-F29)
	if err := migrateRelationshipTables(db); err != nil {
		return fmt.Errorf("failed to migrate relationship tables: %w", err)
	}

	// Create epic_display_data view for single-query epic detail retrieval
	if err := migrateEpicDisplayDataView(db); err != nil {
		return fmt.Errorf("failed to create epic_display_data view: %w", err)
	}

	// Create feature_display_data view for single-query feature detail retrieval
	if err := migrateFeatureDisplayDataView(db); err != nil {
		return fmt.Errorf("failed to create feature_display_data view: %w", err)
	}

	// Create task_display_data view for single-query task detail retrieval
	if err := migrateTaskDisplayDataView(db); err != nil {
		return fmt.Errorf("failed to create task_display_data view: %w", err)
	}

	// Create bugs and change_cards tables (E18-F01)
	if err := migrateBugAndChangeCardTables(db); err != nil {
		return fmt.Errorf("failed to migrate bug and change_card tables: %w", err)
	}

	// Expand entity_notes entity_type CHECK constraint for bug/change (E18-F01)
	if err := migrateEntityNotesExpandEntityTypes(db); err != nil {
		return fmt.Errorf("failed to expand entity_notes entity_type constraint: %w", err)
	}

	// Recreate display views dropped by migrateEntityNotesExpandEntityTypes (E18-F02 fix)
	if err := migrateEpicDisplayDataView(db); err != nil {
		return fmt.Errorf("failed to recreate epic_display_data view: %w", err)
	}
	if err := migrateFeatureDisplayDataView(db); err != nil {
		return fmt.Errorf("failed to recreate feature_display_data view: %w", err)
	}
	if err := migrateTaskDisplayDataView(db); err != nil {
		return fmt.Errorf("failed to recreate task_display_data view: %w", err)
	}

	// Migrate bugs table to use linked_entity_type/linked_entity_key/context_data (E18-F02)
	if err := migrateBugsLinkedEntityColumns(db); err != nil {
		return fmt.Errorf("failed to migrate bugs linked entity columns: %w", err)
	}

	// Add context_data column to change_cards table (E18-F03)
	if err := migrateChangeCardContextData(db); err != nil {
		return fmt.Errorf("failed to migrate change_cards context_data column: %w", err)
	}

	// Add bug_documents and change_card_documents junction tables (E07-F32/F33)
	if err := migrateBugAndChangeCardDocuments(db); err != nil {
		return fmt.Errorf("failed to migrate bug/change_card document tables: %w", err)
	}

	return nil
}

// migrateStatusOverrideColumn adds status_override column to features table
// for supporting manual override of calculated status (E07-F14)
func migrateStatusOverrideColumn(db *sql.DB) error {
	// Check if features table has status_override column
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'status_override'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check features schema for status_override: %w", err)
	}

	if columnExists == 0 {
		// Add status_override column with default false (auto-calculation)
		if _, err := db.Exec(`ALTER TABLE features ADD COLUMN status_override BOOLEAN DEFAULT 0;`); err != nil {
			return fmt.Errorf("failed to add status_override to features: %w", err)
		}
		// Create index for efficient queries
		if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_features_status_override ON features(status_override);`); err != nil {
			return fmt.Errorf("failed to create features status_override index: %w", err)
		}
	}

	return nil
}

// migrateSlugColumns adds slug columns to epics, features, and tasks tables
// This migration supports E07-F11: Slug Architecture Improvement
func migrateSlugColumns(db *sql.DB) error {
	// Check if epics table has slug column; if not, add it
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'slug'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check epics schema for slug: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE epics ADD COLUMN slug TEXT;`); err != nil {
			return fmt.Errorf("failed to add slug to epics: %w", err)
		}
	}

	// Check if features table has slug column; if not, add it
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'slug'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check features schema for slug: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE features ADD COLUMN slug TEXT;`); err != nil {
			return fmt.Errorf("failed to add slug to features: %w", err)
		}
	}

	// Check if tasks table has slug column; if not, add it
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name = 'slug'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check tasks schema for slug: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN slug TEXT;`); err != nil {
			return fmt.Errorf("failed to add slug to tasks: %w", err)
		}
	}

	return nil
}

// migrateDocumentTables handles any future migrations to the document tables
func migrateDocumentTables(db *sql.DB) error {
	// Currently, the document tables are created by createSchema with IF NOT EXISTS.
	// This function is a placeholder for future migrations such as adding new columns.
	// Check if tables exist to ensure schema was created
	var tablesExist int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name IN ('documents', 'epic_documents', 'feature_documents', 'task_documents')
	`).Scan(&tablesExist)
	if err != nil {
		return fmt.Errorf("failed to check document tables: %w", err)
	}

	if tablesExist != 4 {
		return fmt.Errorf("document tables not created: expected 4 tables, found %d", tablesExist)
	}

	return nil
}

// migrateCompletionMetadata adds completion metadata columns to tasks table
func migrateCompletionMetadata(db *sql.DB) error {
	// Check if tasks table has completed_by column; if not, add completion metadata columns
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name = 'completed_by'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check tasks schema for completed_by: %w", err)
	}

	if columnExists == 0 {
		// Add all completion metadata columns
		migrations := []string{
			`ALTER TABLE tasks ADD COLUMN completed_by TEXT;`,
			`ALTER TABLE tasks ADD COLUMN completion_notes TEXT;`,
			`ALTER TABLE tasks ADD COLUMN files_changed TEXT;`, // JSON array
			`ALTER TABLE tasks ADD COLUMN tests_passed BOOLEAN DEFAULT 0;`,
			`ALTER TABLE tasks ADD COLUMN verification_status TEXT CHECK(verification_status IN ('pending', 'verified', 'needs_rework')) DEFAULT 'pending';`,
			`ALTER TABLE tasks ADD COLUMN time_spent_minutes INTEGER;`,
		}

		for _, migration := range migrations {
			if _, err := db.Exec(migration); err != nil {
				return fmt.Errorf("failed to execute migration %q: %w", migration, err)
			}
		}

		// Create indexes
		indexes := []string{
			`CREATE INDEX IF NOT EXISTS idx_tasks_completed_by ON tasks(completed_by);`,
			`CREATE INDEX IF NOT EXISTS idx_tasks_verification_status ON tasks(verification_status);`,
		}

		for _, idx := range indexes {
			if _, err := db.Exec(idx); err != nil {
				return fmt.Errorf("failed to create index: %w", err)
			}
		}
	}

	return nil
}

// migrateTaskCriteriaAndSearch adds task_criteria table and FTS5 virtual table for search
func migrateTaskCriteriaAndSearch(db *sql.DB) error {
	// Check if task_criteria table exists
	var tableExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task_criteria'
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check task_criteria table: %w", err)
	}

	if tableExists == 0 {
		// Create task_criteria table
		_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS task_criteria (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				task_id INTEGER NOT NULL,
				criterion TEXT NOT NULL,
				status TEXT CHECK (status IN ('pending', 'in_progress', 'complete', 'failed', 'na')) DEFAULT 'pending',
				verified_at TIMESTAMP,
				verification_notes TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
			);
		`)
		if err != nil {
			return fmt.Errorf("failed to create task_criteria table: %w", err)
		}

		// Create indexes for task_criteria
		indexes := []string{
			`CREATE INDEX IF NOT EXISTS idx_task_criteria_task_id ON task_criteria(task_id);`,
			`CREATE INDEX IF NOT EXISTS idx_task_criteria_status ON task_criteria(status);`,
		}
		for _, idx := range indexes {
			if _, err := db.Exec(idx); err != nil {
				return fmt.Errorf("failed to create task_criteria index: %w", err)
			}
		}
	}

	// Check if task_search_fts table exists
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task_search_fts'
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check task_search_fts table: %w", err)
	}

	if tableExists == 0 {
		// Create FTS5 virtual table for full-text search (optional - skip if FTS5 not available)
		_, err := db.Exec(`
			CREATE VIRTUAL TABLE task_search_fts USING fts5(
				task_key UNINDEXED,
				title,
				description,
				note_content,
				criterion_text,
				metadata_text,
				tokenize='porter unicode61'
			);
		`)
		if err != nil {
			// FTS5 not available - skip this migration (search feature will be limited)
			// This is acceptable for development environments
			fmt.Printf("Warning: FTS5 not available, skipping full-text search table: %v\n", err)
		}
	}

	return nil
}

// migrateWorkSessionsAndContext adds work_sessions table and context_data column to tasks
func migrateWorkSessionsAndContext(db *sql.DB) error {
	// Check if work_sessions table exists
	var tableExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='work_sessions'
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check work_sessions table: %w", err)
	}

	if tableExists == 0 {
		// Create work_sessions table
		_, err := db.Exec(`
			CREATE TABLE work_sessions (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				task_id INTEGER NOT NULL,
				agent_id TEXT,
				started_at TIMESTAMP NOT NULL,
				ended_at TIMESTAMP,
				outcome TEXT CHECK (outcome IN ('completed', 'paused', 'blocked')),
				session_notes TEXT,
				context_snapshot TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
			);
		`)
		if err != nil {
			return fmt.Errorf("failed to create work_sessions table: %w", err)
		}

		// Create indexes for work_sessions
		indexes := []string{
			`CREATE INDEX IF NOT EXISTS idx_work_sessions_task_id ON work_sessions(task_id);`,
			`CREATE INDEX IF NOT EXISTS idx_work_sessions_agent_id ON work_sessions(agent_id);`,
			`CREATE INDEX IF NOT EXISTS idx_work_sessions_started_at ON work_sessions(started_at);`,
			// Partial index for active sessions (ended_at IS NULL)
			`CREATE INDEX IF NOT EXISTS idx_work_sessions_active ON work_sessions(task_id, ended_at) WHERE ended_at IS NULL;`,
		}
		for _, idx := range indexes {
			if _, err := db.Exec(idx); err != nil {
				return fmt.Errorf("failed to create work_sessions index: %w", err)
			}
		}
	}

	// Check if tasks table has context_data column; if not, add it
	var columnExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name = 'context_data'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check tasks schema for context_data: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN context_data TEXT;`); err != nil {
			return fmt.Errorf("failed to add context_data to tasks: %w", err)
		}
	}

	return nil
}

// BackupDatabase creates a timestamped backup of the database file and associated WAL files
// Returns the backup file path on success, or an error if the backup fails
func BackupDatabase(dbPath string) (string, error) {
	// Check if database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return "", fmt.Errorf("database file does not exist: %s", dbPath)
	}

	// Generate timestamp-based backup filename
	timestamp := time.Now().Format("20060102_150405")
	dir := filepath.Dir(dbPath)
	baseName := filepath.Base(dbPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := baseName[:len(baseName)-len(ext)]

	backupPath := filepath.Join(dir, fmt.Sprintf("%s_%s_backup%s", nameWithoutExt, timestamp, ext))

	// Copy main database file
	if err := copyFile(dbPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to backup database: %w", err)
	}

	// Copy WAL files if they exist (SQLite Write-Ahead Log files)
	walFiles := []string{
		dbPath + "-wal",
		dbPath + "-shm",
	}

	for _, walFile := range walFiles {
		if _, err := os.Stat(walFile); err == nil {
			// WAL file exists, copy it
			walBackupPath := backupPath + filepath.Ext(walFile)
			if err := copyFile(walFile, walBackupPath); err != nil {
				// Log warning but don't fail the backup
				fmt.Fprintf(os.Stderr, "Warning: Failed to backup WAL file %s: %v\n", walFile, err)
			}
		}
	}

	return backupPath, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// migrateIdeasOrderColumn renames the "order" column to "display_order" in the ideas table
// This avoids potential conflicts with the SQL reserved keyword "order"
func migrateIdeasOrderColumn(db *sql.DB) error {
	// Check if ideas table exists
	var tableExists int
	err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='ideas'`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check ideas table: %w", err)
	}

	// If table doesn't exist, nothing to migrate
	if tableExists == 0 {
		return nil
	}

	// Check if old "order" column exists
	var orderColumnExists int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('ideas') WHERE name = 'order'`).Scan(&orderColumnExists)
	if err != nil {
		return fmt.Errorf("failed to check for order column: %w", err)
	}

	// Check if new "display_order" column already exists
	var displayOrderColumnExists int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('ideas') WHERE name = 'display_order'`).Scan(&displayOrderColumnExists)
	if err != nil {
		return fmt.Errorf("failed to check for display_order column: %w", err)
	}

	// If old column exists and new column doesn't exist, we need to migrate
	if orderColumnExists > 0 && displayOrderColumnExists == 0 {
		// SQLite doesn't support ALTER TABLE RENAME COLUMN directly in older versions
		// We need to use the table recreation pattern

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() { _ = tx.Rollback() }()

		// Create new table with display_order column
		_, err = tx.Exec(`
			CREATE TABLE ideas_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				key TEXT NOT NULL UNIQUE,
				title TEXT NOT NULL,
				description TEXT,
				created_date TIMESTAMP NOT NULL,
				priority INTEGER CHECK (priority >= 1 AND priority <= 10),
				display_order INTEGER,
				notes TEXT,
				related_docs TEXT,
				dependencies TEXT,
				status TEXT NOT NULL CHECK (status IN ('new', 'on_hold', 'converted', 'archived')) DEFAULT 'new',
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				converted_to_type TEXT CHECK (converted_to_type IN ('epic', 'feature', 'task')),
				converted_to_key TEXT,
				converted_at TIMESTAMP
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create new ideas table: %w", err)
		}

		// Copy data from old table to new table (renaming order to display_order)
		_, err = tx.Exec(`
			INSERT INTO ideas_new (
				id, key, title, description, created_date, priority, display_order,
				notes, related_docs, dependencies, status, created_at, updated_at,
				converted_to_type, converted_to_key, converted_at
			)
			SELECT
				id, key, title, description, created_date, priority, "order",
				notes, related_docs, dependencies, status, created_at, updated_at,
				converted_to_type, converted_to_key, converted_at
			FROM ideas
		`)
		if err != nil {
			return fmt.Errorf("failed to copy ideas data: %w", err)
		}

		// Drop old table
		_, err = tx.Exec(`DROP TABLE ideas`)
		if err != nil {
			return fmt.Errorf("failed to drop old ideas table: %w", err)
		}

		// Rename new table to original name
		_, err = tx.Exec(`ALTER TABLE ideas_new RENAME TO ideas`)
		if err != nil {
			return fmt.Errorf("failed to rename ideas_new to ideas: %w", err)
		}

		// Recreate indexes
		indexes := []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_ideas_key ON ideas(key)`,
			`CREATE INDEX IF NOT EXISTS idx_ideas_status ON ideas(status)`,
			`CREATE INDEX IF NOT EXISTS idx_ideas_created_date ON ideas(created_date DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_ideas_priority ON ideas(priority)`,
		}

		for _, idx := range indexes {
			if _, err := tx.Exec(idx); err != nil {
				return fmt.Errorf("failed to create index: %w", err)
			}
		}

		// Recreate trigger
		_, err = tx.Exec(`
			CREATE TRIGGER IF NOT EXISTS ideas_updated_at
			AFTER UPDATE ON ideas
			FOR EACH ROW
			BEGIN
				UPDATE ideas SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
			END
		`)
		if err != nil {
			return fmt.Errorf("failed to create ideas_updated_at trigger: %w", err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
}

// migrateDropCustomFolderPath removes custom_folder_path columns from epics and features tables
// This migration supports E07-F19: File Path Flag Standardization
// The custom_folder_path columns were stored but never used in path calculations
func migrateDropCustomFolderPath(db *sql.DB) error {
	// Check if epics table has custom_folder_path column
	var epicColumnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'custom_folder_path'
	`).Scan(&epicColumnExists)
	if err != nil {
		return fmt.Errorf("failed to check epics schema for custom_folder_path: %w", err)
	}

	// Drop column and index from epics table if it exists
	if epicColumnExists > 0 {
		// Drop index first (required before dropping column)
		_, err = db.Exec(`DROP INDEX IF EXISTS idx_epics_custom_folder_path`)
		if err != nil {
			return fmt.Errorf("failed to drop epics custom_folder_path index: %w", err)
		}

		// Drop column
		_, err = db.Exec(`ALTER TABLE epics DROP COLUMN custom_folder_path`)
		if err != nil {
			return fmt.Errorf("failed to drop custom_folder_path from epics: %w", err)
		}
	}

	// Check if features table has custom_folder_path column
	var featureColumnExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'custom_folder_path'
	`).Scan(&featureColumnExists)
	if err != nil {
		return fmt.Errorf("failed to check features schema for custom_folder_path: %w", err)
	}

	// Drop column and index from features table if it exists
	if featureColumnExists > 0 {
		// Drop index first (required before dropping column)
		_, err = db.Exec(`DROP INDEX IF EXISTS idx_features_custom_folder_path`)
		if err != nil {
			return fmt.Errorf("failed to drop features custom_folder_path index: %w", err)
		}

		// Drop column
		_, err = db.Exec(`ALTER TABLE features DROP COLUMN custom_folder_path`)
		if err != nil {
			return fmt.Errorf("failed to drop custom_folder_path from features: %w", err)
		}
	}

	return nil
}

// migrateTaskNotesMetadata adds metadata column to task_notes table for storing JSON metadata
// This supports rejection note tracking with history_id, status transitions, and document paths (E07-F22)
func migrateTaskNotesMetadata(db *sql.DB) error {
	// Check if task_notes table has metadata column
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('task_notes') WHERE name = 'metadata'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check task_notes schema for metadata: %w", err)
	}

	if columnExists == 0 {
		// Add metadata column for storing JSON data
		if _, err := db.Exec(`ALTER TABLE task_notes ADD COLUMN metadata TEXT;`); err != nil {
			return fmt.Errorf("failed to add metadata column to task_notes: %w", err)
		}
	}

	// Create composite index on (note_type, task_id) for efficient rejection note queries
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_task_notes_type_task ON task_notes(note_type, task_id);`); err != nil {
		return fmt.Errorf("failed to create index on task_notes(note_type, task_id): %w", err)
	}

	return nil
}

// migrateTaskHistoryRejectionReason adds rejection_reason column to task_history table
// This column stores rejection reasons when tasks are rejected during review/QA (E07-F22)
func migrateTaskHistoryRejectionReason(db *sql.DB) error {
	// Check if task_history table has rejection_reason column
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('task_history') WHERE name = 'rejection_reason'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check task_history schema for rejection_reason: %w", err)
	}

	if columnExists == 0 {
		// Add rejection_reason column for storing rejection reasons
		if _, err := db.Exec(`ALTER TABLE task_history ADD COLUMN rejection_reason TEXT;`); err != nil {
			return fmt.Errorf("failed to add rejection_reason column to task_history: %w", err)
		}
	}

	// Create index on rejection_reason for filtering rejection records
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_task_history_rejection_reason ON task_history(rejection_reason) WHERE rejection_reason IS NOT NULL;`); err != nil {
		return fmt.Errorf("failed to create index on task_history(rejection_reason): %w", err)
	}

	return nil
}

// migrateTaskDocumentsLinkType adds link_type column to task_documents table
// for specifying the type of link between task and document (e.g., rejection_reason) (E07-F22)
func migrateTaskDocumentsLinkType(db *sql.DB) error {
	// Check if task_documents table has link_type column
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('task_documents') WHERE name = 'link_type'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check task_documents schema for link_type: %w", err)
	}

	if columnExists == 0 {
		// Add link_type column for categorizing document links
		if _, err := db.Exec(`ALTER TABLE task_documents ADD COLUMN link_type TEXT DEFAULT 'general';`); err != nil {
			return fmt.Errorf("failed to add link_type column to task_documents: %w", err)
		}
	}

	// Create index on link_type for filtering by link type
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_task_documents_link_type ON task_documents(link_type);`); err != nil {
		return fmt.Errorf("failed to create index on task_documents(link_type): %w", err)
	}

	return nil
}

// migrateTaskNotesNoteTypeConstraint updates the note_type CHECK constraint in task_notes table
// to include 'rejection' for storing rejection reasons during backward task transitions (E07-F22)
// This migration recreates the table with the updated constraint for existing databases
func migrateTaskNotesNoteTypeConstraint(db *sql.DB) error {
	// Check if task_notes table exists
	var tableExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task_notes'
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check task_notes table: %w", err)
	}

	// If table doesn't exist, nothing to migrate
	if tableExists == 0 {
		return nil
	}

	// Check if table already allows 'rejection' note type by trying to insert a test value
	// This is a non-destructive check - the insertion will rollback
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for constraint check: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Try to insert a dummy rejection note to test if the constraint allows it
	var taskIDExists int
	err = tx.QueryRow(`SELECT COUNT(*) FROM tasks LIMIT 1`).Scan(&taskIDExists)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check tasks table: %w", err)
	}

	// If no tasks exist, we can't test, so assume migration needed
	// (or check the constraint is already updated by trying to create a test entry)
	if taskIDExists == 0 {
		// No tasks to test with, just return - the constraint will be enforced on first use
		return nil
	}

	// For databases with tasks, we need to recreate the table if constraint is old
	// Get a task ID to use for testing
	var testTaskID int64
	err = tx.QueryRow(`SELECT id FROM tasks LIMIT 1`).Scan(&testTaskID)
	if err != nil {
		return nil // No tasks, nothing to check
	}

	// Try inserting a rejection note
	result := tx.QueryRow(`
		INSERT INTO task_notes (task_id, note_type, content)
		VALUES (?, 'rejection', 'test')
		RETURNING id;
	`, testTaskID)

	var noteID int64
	err = result.Scan(&noteID)

	// If insertion succeeded, constraint already allows 'rejection', so we're done
	if err == nil {
		// Clean up the test insertion by rolling back the transaction
		return nil
	}

	// If we got a constraint error, we need to migrate the table
	// Rollback the transaction
	_ = tx.Rollback()

	// Now perform the actual migration using table recreation
	tx, err = db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Create new table with updated note_type constraint
	_, err = tx.Exec(`
		CREATE TABLE task_notes_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER NOT NULL,
			note_type TEXT CHECK (note_type IN (
				'comment',
				'decision',
				'blocker',
				'solution',
				'reference',
				'implementation',
				'testing',
				'future',
				'question',
				'rejection'
			)) NOT NULL,
			content TEXT NOT NULL,
			created_by TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			metadata TEXT,
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create new task_notes table: %w", err)
	}

	// Copy data from old table to new table
	_, err = tx.Exec(`
		INSERT INTO task_notes_new (id, task_id, note_type, content, created_by, created_at, metadata)
		SELECT id, task_id, note_type, content, created_by, created_at, metadata
		FROM task_notes
	`)
	if err != nil {
		return fmt.Errorf("failed to copy task_notes data: %w", err)
	}

	// Drop old table
	_, err = tx.Exec(`DROP TABLE task_notes`)
	if err != nil {
		return fmt.Errorf("failed to drop old task_notes table: %w", err)
	}

	// Rename new table to original name
	_, err = tx.Exec(`ALTER TABLE task_notes_new RENAME TO task_notes`)
	if err != nil {
		return fmt.Errorf("failed to rename task_notes_new to task_notes: %w", err)
	}

	// Recreate indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_task_notes_task_id ON task_notes(task_id);`,
		`CREATE INDEX IF NOT EXISTS idx_task_notes_type ON task_notes(note_type);`,
		`CREATE INDEX IF NOT EXISTS idx_task_notes_created_at ON task_notes(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_task_notes_task_type ON task_notes(task_id, note_type);`,
		`CREATE INDEX IF NOT EXISTS idx_task_notes_type_task ON task_notes(note_type, task_id);`,
	}

	for _, idx := range indexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	return nil
}

// migrateEntityNotes creates the polymorphic entity_notes table and migrates data from task_notes (E16-F04)
func migrateEntityNotes(db *sql.DB) error {
	// Check if entity_notes table already exists
	var tableExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='entity_notes'
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check entity_notes table: %w", err)
	}

	if tableExists > 0 {
		// Already migrated
		return nil
	}

	// Create entity_notes table
	_, err = db.Exec(`
		CREATE TABLE entity_notes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_type TEXT NOT NULL CHECK (entity_type IN ('epic', 'feature', 'task')),
			entity_id INTEGER NOT NULL,
			note_type TEXT CHECK (note_type IN (
				'comment', 'decision', 'blocker', 'solution', 'reference',
				'implementation', 'testing', 'future', 'question', 'rejection'
			)) NOT NULL,
			content TEXT NOT NULL,
			created_by TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			metadata TEXT
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create entity_notes table: %w", err)
	}

	// Create indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_entity_notes_type ON entity_notes(note_type);`,
		`CREATE INDEX IF NOT EXISTS idx_entity_notes_created_at ON entity_notes(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_entity_notes_entity_type ON entity_notes(entity_type, entity_id, note_type);`,
		`CREATE INDEX IF NOT EXISTS idx_entity_notes_type_entity ON entity_notes(note_type, entity_type, entity_id);`,
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create entity_notes index: %w", err)
		}
	}

	// Create cascade delete triggers for entity_notes
	// Since entity_notes is polymorphic, we can't use FK constraints directly.
	// Instead, use triggers to cascade deletes when tasks, features, or epics are deleted.
	cascadeTriggers := []string{
		`CREATE TRIGGER IF NOT EXISTS entity_notes_cascade_delete_task
			AFTER DELETE ON tasks
			FOR EACH ROW
			BEGIN
				DELETE FROM entity_notes WHERE entity_type = 'task' AND entity_id = OLD.id;
			END;`,
		`CREATE TRIGGER IF NOT EXISTS entity_notes_cascade_delete_feature
			AFTER DELETE ON features
			FOR EACH ROW
			BEGIN
				DELETE FROM entity_notes WHERE entity_type = 'feature' AND entity_id = OLD.id;
			END;`,
		`CREATE TRIGGER IF NOT EXISTS entity_notes_cascade_delete_epic
			AFTER DELETE ON epics
			FOR EACH ROW
			BEGIN
				DELETE FROM entity_notes WHERE entity_type = 'epic' AND entity_id = OLD.id;
			END;`,
	}
	for _, trigger := range cascadeTriggers {
		if _, err := db.Exec(trigger); err != nil {
			return fmt.Errorf("failed to create entity_notes cascade trigger: %w", err)
		}
	}

	// Check if task_notes table exists and has data to migrate
	var taskNotesExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task_notes'
	`).Scan(&taskNotesExists)
	if err != nil {
		return fmt.Errorf("failed to check task_notes table: %w", err)
	}

	if taskNotesExists > 0 {
		// Migrate data from task_notes to entity_notes
		_, err = db.Exec(`
			INSERT INTO entity_notes (id, entity_type, entity_id, note_type, content, created_by, created_at, metadata)
			SELECT id, 'task', task_id, note_type, content, created_by, created_at, metadata
			FROM task_notes
		`)
		if err != nil {
			return fmt.Errorf("failed to migrate task_notes data to entity_notes: %w", err)
		}

		// Rename old table to backup
		_, err = db.Exec(`ALTER TABLE task_notes RENAME TO task_notes_backup`)
		if err != nil {
			// If backup already exists (from a previous partial migration), just drop task_notes
			var backupExists int
			_ = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task_notes_backup'`).Scan(&backupExists)
			if backupExists > 0 {
				_, _ = db.Exec(`DROP TABLE IF EXISTS task_notes`)
			} else {
				return fmt.Errorf("failed to rename task_notes to task_notes_backup: %w", err)
			}
		}
	}

	return nil
}

// migrateEpicFeatureContextData adds context_data TEXT column to epics and features tables (E16-F04)
func migrateEpicFeatureContextData(db *sql.DB) error {
	// Check if epics table has context_data column
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'context_data'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check epics schema for context_data: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE epics ADD COLUMN context_data TEXT;`); err != nil {
			return fmt.Errorf("failed to add context_data to epics: %w", err)
		}
	}

	// Check if features table has context_data column
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'context_data'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check features schema for context_data: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE features ADD COLUMN context_data TEXT;`); err != nil {
			return fmt.Errorf("failed to add context_data to features: %w", err)
		}
	}

	return nil
}

// migrateRelationshipTables creates feature_relationships and epic_relationships tables (E07-F29)
func migrateRelationshipTables(db *sql.DB) error {
	// Check if feature_relationships table exists
	var featureTableExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='feature_relationships'
	`).Scan(&featureTableExists)
	if err != nil {
		return fmt.Errorf("failed to check for feature_relationships table: %w", err)
	}

	if featureTableExists == 0 {
		// Create feature_relationships table
		_, err = db.Exec(`
			CREATE TABLE feature_relationships (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				from_feature_id INTEGER NOT NULL,
				to_feature_id INTEGER NOT NULL,
				relationship_type TEXT CHECK (relationship_type IN (
					'depends_on', 'blocks', 'related_to', 'follows',
					'spawned_from', 'duplicates', 'references'
				)) NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (from_feature_id) REFERENCES features(id) ON DELETE CASCADE,
				FOREIGN KEY (to_feature_id) REFERENCES features(id) ON DELETE CASCADE,
				UNIQUE(from_feature_id, to_feature_id, relationship_type)
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create feature_relationships table: %w", err)
		}

		// Create indexes for feature_relationships
		indexes := []string{
			`CREATE INDEX IF NOT EXISTS idx_feature_relationships_from ON feature_relationships(from_feature_id)`,
			`CREATE INDEX IF NOT EXISTS idx_feature_relationships_to ON feature_relationships(to_feature_id)`,
			`CREATE INDEX IF NOT EXISTS idx_feature_relationships_type ON feature_relationships(relationship_type)`,
		}

		for _, idx := range indexes {
			if _, err := db.Exec(idx); err != nil {
				return fmt.Errorf("failed to create feature_relationships index: %w", err)
			}
		}
	}

	// Check if epic_relationships table exists
	var epicTableExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='epic_relationships'
	`).Scan(&epicTableExists)
	if err != nil {
		return fmt.Errorf("failed to check for epic_relationships table: %w", err)
	}

	if epicTableExists == 0 {
		// Create epic_relationships table
		_, err = db.Exec(`
			CREATE TABLE epic_relationships (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				from_epic_id INTEGER NOT NULL,
				to_epic_id INTEGER NOT NULL,
				relationship_type TEXT CHECK (relationship_type IN (
					'depends_on', 'blocks', 'related_to', 'follows',
					'spawned_from', 'duplicates', 'references'
				)) NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (from_epic_id) REFERENCES epics(id) ON DELETE CASCADE,
				FOREIGN KEY (to_epic_id) REFERENCES epics(id) ON DELETE CASCADE,
				UNIQUE(from_epic_id, to_epic_id, relationship_type)
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create epic_relationships table: %w", err)
		}

		// Create indexes for epic_relationships
		indexes := []string{
			`CREATE INDEX IF NOT EXISTS idx_epic_relationships_from ON epic_relationships(from_epic_id)`,
			`CREATE INDEX IF NOT EXISTS idx_epic_relationships_to ON epic_relationships(to_epic_id)`,
			`CREATE INDEX IF NOT EXISTS idx_epic_relationships_type ON epic_relationships(relationship_type)`,
		}

		for _, idx := range indexes {
			if _, err := db.Exec(idx); err != nil {
				return fmt.Errorf("failed to create epic_relationships index: %w", err)
			}
		}
	}

	return nil
}

// migrateEpicDisplayDataView creates the epic_display_data SQL view that aggregates
// all epic-related data (features, task breakdown, blocked tasks, documents, notes)
// into a single queryable view for efficient epic detail retrieval.
func migrateEpicDisplayDataView(db *sql.DB) error {
	_, err := db.Exec(`
CREATE VIEW IF NOT EXISTS epic_display_data AS
SELECT
  e.*,

  -- Features (includes progress_pct from write-through cache)
  (SELECT COALESCE(json_group_array(json_object(
    'id', f.id, 'key', f.key, 'title', f.title, 'slug', f.slug,
    'description', f.description,
    'status', f.status, 'status_override', COALESCE(f.status_override, 0),
    'progress_pct', f.progress_pct, 'execution_order', f.execution_order,
    'file_path', f.file_path, 'context_data', f.context_data,
    'created_at', f.created_at, 'updated_at', f.updated_at
  )), '[]') FROM features f WHERE f.epic_id = e.id
  ) AS features_json,

  -- Task status breakdown: [{feature_id, status, cnt}, ...]
  (SELECT COALESCE(json_group_array(json_object(
    'feature_id', sub.feature_id, 'status', sub.status, 'cnt', sub.cnt
  )), '[]') FROM (
    SELECT t.feature_id, t.status, COUNT(*) as cnt
    FROM tasks t JOIN features f2 ON t.feature_id = f2.id
    WHERE f2.epic_id = e.id
    GROUP BY t.feature_id, t.status
  ) sub
  ) AS task_breakdown_json,

  -- Blocked tasks with details
  (SELECT COALESCE(json_group_array(json_object(
    'id', t.id, 'feature_id', t.feature_id, 'key', t.key, 'title', t.title,
    'status', t.status, 'blocked_reason', t.blocked_reason,
    'blocked_at', t.blocked_at
  )), '[]') FROM tasks t JOIN features f3 ON t.feature_id = f3.id
  WHERE f3.epic_id = e.id AND t.status = 'blocked'
  ) AS blocked_tasks_json,

  -- Related documents
  (SELECT COALESCE(json_group_array(json_object(
    'id', d.id, 'title', d.title, 'file_path', d.file_path
  )), '[]') FROM documents d JOIN epic_documents ed ON d.id = ed.document_id
  WHERE ed.epic_id = e.id
  ) AS documents_json,

  -- Entity notes
  (SELECT COALESCE(json_group_array(json_object(
    'id', n.id, 'note_type', n.note_type, 'content', n.content,
    'created_by', n.created_by, 'metadata', n.metadata, 'created_at', n.created_at
  )), '[]') FROM entity_notes n
  WHERE n.entity_type = 'epic' AND n.entity_id = e.id
  ) AS notes_json

FROM epics e;
`)
	return err
}

// migrateFeatureDisplayDataView creates the feature_display_data SQL view that aggregates
// all feature-related data (tasks, task breakdown, documents, notes) into a single
// queryable view for efficient feature detail retrieval.
func migrateFeatureDisplayDataView(db *sql.DB) error {
	_, err := db.Exec(`DROP VIEW IF EXISTS feature_display_data`)
	if err != nil {
		return fmt.Errorf("failed to drop old feature_display_data view: %w", err)
	}

	_, err = db.Exec(`
CREATE VIEW IF NOT EXISTS feature_display_data AS
SELECT
  f.*,

  -- Tasks (includes all task details needed for display)
  (SELECT COALESCE(json_group_array(json_object(
    'id', t.id, 'key', t.key, 'title', t.title, 'slug', t.slug,
    'description', t.description,
    'status', t.status, 'agent_type', t.agent_type,
    'priority', t.priority, 'execution_order', t.execution_order,
    'file_path', t.file_path, 'context_data', t.context_data,
    'blocked_reason', t.blocked_reason, 'blocked_at', t.blocked_at,
    'created_at', t.created_at, 'updated_at', t.updated_at
  )), '[]') FROM tasks t WHERE t.feature_id = f.id
  ) AS tasks_json,

  -- Task status breakdown: [{status, cnt}, ...]
  (SELECT COALESCE(json_group_array(json_object(
    'status', sub.status, 'cnt', sub.cnt
  )), '[]') FROM (
    SELECT t.status, COUNT(*) as cnt
    FROM tasks t
    WHERE t.feature_id = f.id
    GROUP BY t.status
  ) sub
  ) AS task_breakdown_json,

  -- Related documents
  (SELECT COALESCE(json_group_array(json_object(
    'id', d.id, 'title', d.title, 'file_path', d.file_path
  )), '[]') FROM documents d JOIN feature_documents fd ON d.id = fd.document_id
  WHERE fd.feature_id = f.id
  ) AS documents_json,

  -- Entity notes
  (SELECT COALESCE(json_group_array(json_object(
    'id', n.id, 'note_type', n.note_type, 'content', n.content,
    'created_by', n.created_by, 'metadata', n.metadata, 'created_at', n.created_at
  )), '[]') FROM entity_notes n
  WHERE n.entity_type = 'feature' AND n.entity_id = f.id
  ) AS notes_json

FROM features f;
`)
	return err
}

// migrateTaskDisplayDataView creates the task_display_data SQL view that aggregates
// all task-related data (blocked_by, blocks, dependencies, documents, notes) into a single
// queryable view for efficient task detail retrieval.
func migrateTaskDisplayDataView(db *sql.DB) error {
	_, err := db.Exec(`DROP VIEW IF EXISTS task_display_data`)
	if err != nil {
		return fmt.Errorf("failed to drop old task_display_data view: %w", err)
	}

	_, err = db.Exec(`
CREATE VIEW IF NOT EXISTS task_display_data AS
SELECT
  t.*,

  -- Blocked-by: outgoing depends_on relationships (tasks this task depends on)
  (SELECT COALESCE(json_group_array(json_object(
    'relationship_type', 'depends_on', 'direction', 'outgoing',
    'task_key', t2.key, 'task_title', t2.title, 'task_status', t2.status
  )), '[]') FROM task_relationships tr JOIN tasks t2 ON t2.id = tr.to_task_id
  WHERE tr.from_task_id = t.id AND tr.relationship_type = 'depends_on'
  ) AS blocked_by_json,

  -- Blocks: incoming depends_on + outgoing blocks (tasks blocked by this task)
  (SELECT COALESCE(json_group_array(json_object(
    'relationship_type', sub.rt, 'direction', sub.dir,
    'task_key', sub.key, 'task_title', sub.title, 'task_status', sub.status
  )), '[]') FROM (
    SELECT 'depends_on' as rt, 'incoming' as dir, t2.key, t2.title, t2.status
    FROM task_relationships tr JOIN tasks t2 ON t2.id = tr.from_task_id
    WHERE tr.to_task_id = t.id AND tr.relationship_type = 'depends_on'
    UNION ALL
    SELECT 'blocks' as rt, 'outgoing' as dir, t2.key, t2.title, t2.status
    FROM task_relationships tr JOIN tasks t2 ON t2.id = tr.to_task_id
    WHERE tr.from_task_id = t.id AND tr.relationship_type = 'blocks'
  ) sub
  ) AS blocks_json,

  -- Dependencies: all related tasks (both directions, distinct)
  (SELECT COALESCE(json_group_array(json_object(
    'key', sub2.key, 'title', sub2.title, 'status', sub2.status
  )), '[]') FROM (
    SELECT DISTINCT t2.key, t2.title, t2.status
    FROM task_relationships tr
    JOIN tasks t2 ON (
      (tr.from_task_id = t.id AND tr.to_task_id = t2.id) OR
      (tr.to_task_id = t.id AND tr.from_task_id = t2.id)
    )
    WHERE t2.id != t.id
  ) sub2
  ) AS dependencies_json,

  -- Related documents
  (SELECT COALESCE(json_group_array(json_object(
    'id', d.id, 'title', d.title, 'file_path', d.file_path
  )), '[]') FROM documents d JOIN task_documents td ON d.id = td.document_id
  WHERE td.task_id = t.id
  ) AS documents_json,

  -- Entity notes
  (SELECT COALESCE(json_group_array(json_object(
    'id', n.id, 'note_type', n.note_type, 'content', n.content,
    'created_by', n.created_by, 'metadata', n.metadata, 'created_at', n.created_at
  )), '[]') FROM entity_notes n
  WHERE n.entity_type = 'task' AND n.entity_id = t.id
  ) AS notes_json

FROM tasks t;
`)
	return err
}

// migrateBugAndChangeCardTables creates the bugs and change_cards tables (E18-F01).
func migrateBugAndChangeCardTables(db *sql.DB) error {
	// Check if bugs table already exists
	var bugsExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='bugs'
	`).Scan(&bugsExists)
	if err != nil {
		return fmt.Errorf("failed to check bugs table: %w", err)
	}

	if bugsExists == 0 {
		_, err = db.Exec(`
			CREATE TABLE bugs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				key TEXT NOT NULL UNIQUE,
				title TEXT NOT NULL,
				slug TEXT,
				description TEXT,
				status TEXT NOT NULL DEFAULT 'reported',
				severity TEXT NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low')) DEFAULT 'medium',
				linked_entity_type TEXT,
				linked_entity_key TEXT,
				context_data TEXT,
				file_path TEXT,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
		`)
		if err != nil {
			return fmt.Errorf("failed to create bugs table: %w", err)
		}

		// Create indexes for bugs
		bugIndexes := []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_bugs_key ON bugs(key);`,
			`CREATE INDEX IF NOT EXISTS idx_bugs_status ON bugs(status);`,
			`CREATE INDEX IF NOT EXISTS idx_bugs_severity ON bugs(severity);`,
			`CREATE INDEX IF NOT EXISTS idx_bugs_linked_entity_type ON bugs(linked_entity_type);`,
			`CREATE INDEX IF NOT EXISTS idx_bugs_slug ON bugs(slug);`,
		}
		for _, idx := range bugIndexes {
			if _, err := db.Exec(idx); err != nil {
				return fmt.Errorf("failed to create bugs index: %w", err)
			}
		}

		// Create updated_at trigger for bugs
		_, err = db.Exec(`
			CREATE TRIGGER IF NOT EXISTS bugs_updated_at
			AFTER UPDATE ON bugs
			FOR EACH ROW
			BEGIN
				UPDATE bugs SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
			END;
		`)
		if err != nil {
			return fmt.Errorf("failed to create bugs updated_at trigger: %w", err)
		}
	}

	// Check if change_cards table already exists
	var changeCardsExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='change_cards'
	`).Scan(&changeCardsExists)
	if err != nil {
		return fmt.Errorf("failed to check change_cards table: %w", err)
	}

	if changeCardsExists == 0 {
		_, err = db.Exec(`
			CREATE TABLE change_cards (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				key TEXT NOT NULL UNIQUE,
				title TEXT NOT NULL,
				description TEXT,
				status TEXT NOT NULL DEFAULT 'proposed',
				priority INTEGER CHECK (priority >= 1 AND priority <= 10) DEFAULT 5,
				requested_by TEXT,
				assigned_to TEXT,
				epic_id INTEGER REFERENCES epics(id),
				feature_id INTEGER REFERENCES features(id),
				related_task_id INTEGER REFERENCES tasks(id),
				justification TEXT,
				impact_analysis TEXT,
				rollback_plan TEXT,
				slug TEXT,
				file_path TEXT,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
		`)
		if err != nil {
			return fmt.Errorf("failed to create change_cards table: %w", err)
		}

		// Create indexes for change_cards
		changeCardIndexes := []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_change_cards_key ON change_cards(key);`,
			`CREATE INDEX IF NOT EXISTS idx_change_cards_status ON change_cards(status);`,
			`CREATE INDEX IF NOT EXISTS idx_change_cards_epic_id ON change_cards(epic_id);`,
			`CREATE INDEX IF NOT EXISTS idx_change_cards_feature_id ON change_cards(feature_id);`,
			`CREATE INDEX IF NOT EXISTS idx_change_cards_slug ON change_cards(slug);`,
		}
		for _, idx := range changeCardIndexes {
			if _, err := db.Exec(idx); err != nil {
				return fmt.Errorf("failed to create change_cards index: %w", err)
			}
		}

		// Create updated_at trigger for change_cards
		_, err = db.Exec(`
			CREATE TRIGGER IF NOT EXISTS change_cards_updated_at
			AFTER UPDATE ON change_cards
			FOR EACH ROW
			BEGIN
				UPDATE change_cards SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
			END;
		`)
		if err != nil {
			return fmt.Errorf("failed to create change_cards updated_at trigger: %w", err)
		}
	}

	return nil
}

// migrateEntityNotesExpandEntityTypes expands entity_notes CHECK constraint to include 'bug' and 'change' (E18-F01).
// SQLite does not support ALTER TABLE to modify CHECK constraints, so we recreate the table.
func migrateEntityNotesExpandEntityTypes(db *sql.DB) error {
	// Check if entity_notes table exists
	var tableExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='entity_notes'
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check entity_notes table: %w", err)
	}

	if tableExists == 0 {
		// Table doesn't exist yet; it will be created by migrateEntityNotes with the old constraint.
		// That's fine -- we'll handle it on next migration run.
		return nil
	}

	// Check current CHECK constraint by examining table SQL
	var tableSql string
	err = db.QueryRow(`
		SELECT sql FROM sqlite_master WHERE type='table' AND name='entity_notes'
	`).Scan(&tableSql)
	if err != nil {
		return fmt.Errorf("failed to get entity_notes schema: %w", err)
	}

	// If the table already has 'bug' in the CHECK constraint, skip
	if strings.Contains(tableSql, "'bug'") {
		return nil
	}

	// Recreate the table with expanded CHECK constraint using SQLite table recreation pattern
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	steps := []string{
		// Create new table with expanded CHECK
		`CREATE TABLE entity_notes_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_type TEXT NOT NULL CHECK (entity_type IN ('epic', 'feature', 'task', 'bug', 'change')),
			entity_id INTEGER NOT NULL,
			note_type TEXT CHECK (note_type IN (
				'comment', 'decision', 'blocker', 'solution', 'reference',
				'implementation', 'testing', 'future', 'question', 'rejection'
			)) NOT NULL,
			content TEXT NOT NULL,
			created_by TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			metadata TEXT
		);`,
		// Copy data
		`INSERT INTO entity_notes_new SELECT * FROM entity_notes;`,
		// Drop views that reference entity_notes (they'll be recreated after)
		`DROP VIEW IF EXISTS epic_display_data;`,
		`DROP VIEW IF EXISTS feature_display_data;`,
		`DROP VIEW IF EXISTS task_display_data;`,
		// Drop cascade triggers that reference the old entity_notes table
		`DROP TRIGGER IF EXISTS entity_notes_cascade_delete_task;`,
		`DROP TRIGGER IF EXISTS entity_notes_cascade_delete_feature;`,
		`DROP TRIGGER IF EXISTS entity_notes_cascade_delete_epic;`,
		// Drop old table
		`DROP TABLE entity_notes;`,
		// Rename new table
		`ALTER TABLE entity_notes_new RENAME TO entity_notes;`,
		// Recreate indexes
		`CREATE INDEX IF NOT EXISTS idx_entity_notes_type ON entity_notes(note_type);`,
		`CREATE INDEX IF NOT EXISTS idx_entity_notes_created_at ON entity_notes(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_entity_notes_entity_type ON entity_notes(entity_type, entity_id, note_type);`,
		`CREATE INDEX IF NOT EXISTS idx_entity_notes_type_entity ON entity_notes(note_type, entity_type, entity_id);`,
		// Recreate cascade delete triggers
		`CREATE TRIGGER IF NOT EXISTS entity_notes_cascade_delete_task
			AFTER DELETE ON tasks
			FOR EACH ROW
			BEGIN
				DELETE FROM entity_notes WHERE entity_type = 'task' AND entity_id = OLD.id;
			END;`,
		`CREATE TRIGGER IF NOT EXISTS entity_notes_cascade_delete_feature
			AFTER DELETE ON features
			FOR EACH ROW
			BEGIN
				DELETE FROM entity_notes WHERE entity_type = 'feature' AND entity_id = OLD.id;
			END;`,
		`CREATE TRIGGER IF NOT EXISTS entity_notes_cascade_delete_epic
			AFTER DELETE ON epics
			FOR EACH ROW
			BEGIN
				DELETE FROM entity_notes WHERE entity_type = 'epic' AND entity_id = OLD.id;
			END;`,
		// Add cascade triggers for bugs and change_cards
		`CREATE TRIGGER IF NOT EXISTS entity_notes_cascade_delete_bug
			AFTER DELETE ON bugs
			FOR EACH ROW
			BEGIN
				DELETE FROM entity_notes WHERE entity_type = 'bug' AND entity_id = OLD.id;
			END;`,
		`CREATE TRIGGER IF NOT EXISTS entity_notes_cascade_delete_change
			AFTER DELETE ON change_cards
			FOR EACH ROW
			BEGIN
				DELETE FROM entity_notes WHERE entity_type = 'change' AND entity_id = OLD.id;
			END;`,
	}

	for _, step := range steps {
		if _, err := tx.Exec(step); err != nil {
			return fmt.Errorf("failed to migrate entity_notes: %w (step: %s)", err, step[:min(len(step), 60)])
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit entity_notes migration: %w", err)
	}

	return nil
}

// migrateBugsLinkedEntityColumns adds linked_entity_type, linked_entity_key, and context_data
// columns to an existing bugs table. The original schema used epic_id/feature_id/related_task_id
// FK columns, but the repository uses the more flexible linked_entity_type/key/context_data
// pattern (E18-F02).
func migrateBugsLinkedEntityColumns(db *sql.DB) error {
	// Only run if bugs table exists
	var bugsExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='bugs'
	`).Scan(&bugsExists)
	if err != nil {
		return fmt.Errorf("failed to check bugs table: %w", err)
	}
	if bugsExists == 0 {
		// Table doesn't exist yet; migrateBugAndChangeCardTables will create it with correct schema
		return nil
	}

	// Add linked_entity_type if missing
	var hasLinkedEntityType int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('bugs') WHERE name = 'linked_entity_type'
	`).Scan(&hasLinkedEntityType); err != nil {
		return fmt.Errorf("failed to check bugs linked_entity_type column: %w", err)
	}
	if hasLinkedEntityType == 0 {
		if _, err := db.Exec(`ALTER TABLE bugs ADD COLUMN linked_entity_type TEXT;`); err != nil {
			return fmt.Errorf("failed to add linked_entity_type to bugs: %w", err)
		}
	}

	// Add linked_entity_key if missing
	var hasLinkedEntityKey int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('bugs') WHERE name = 'linked_entity_key'
	`).Scan(&hasLinkedEntityKey); err != nil {
		return fmt.Errorf("failed to check bugs linked_entity_key column: %w", err)
	}
	if hasLinkedEntityKey == 0 {
		if _, err := db.Exec(`ALTER TABLE bugs ADD COLUMN linked_entity_key TEXT;`); err != nil {
			return fmt.Errorf("failed to add linked_entity_key to bugs: %w", err)
		}
	}

	// Add context_data if missing
	var hasContextData int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('bugs') WHERE name = 'context_data'
	`).Scan(&hasContextData); err != nil {
		return fmt.Errorf("failed to check bugs context_data column: %w", err)
	}
	if hasContextData == 0 {
		if _, err := db.Exec(`ALTER TABLE bugs ADD COLUMN context_data TEXT;`); err != nil {
			return fmt.Errorf("failed to add context_data to bugs: %w", err)
		}
	}

	return nil
}

// migrateChangeCardContextData adds context_data column to change_cards table (E18-F03).
// This column stores JSON context data for AI agent session management on change-cards.
func migrateChangeCardContextData(db *sql.DB) error {
	// Only run if change_cards table exists
	var tableExists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='change_cards'
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check change_cards table: %w", err)
	}
	if tableExists == 0 {
		// Table doesn't exist yet; migrateBugAndChangeCardTables will create it with correct schema
		return nil
	}

	// Add context_data if missing
	var hasContextData int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('change_cards') WHERE name = 'context_data'
	`).Scan(&hasContextData); err != nil {
		return fmt.Errorf("failed to check change_cards context_data column: %w", err)
	}
	if hasContextData == 0 {
		if _, err := db.Exec(`ALTER TABLE change_cards ADD COLUMN context_data TEXT;`); err != nil {
			return fmt.Errorf("failed to add context_data to change_cards: %w", err)
		}
	}

	return nil
}

// migrateBugAndChangeCardDocuments creates bug_documents and change_card_documents
// junction tables for linking documents to bugs and change-cards (E07-F32/F33).
func migrateBugAndChangeCardDocuments(db *sql.DB) error {
	// Create bug_documents junction table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS bug_documents (
			bug_id INTEGER NOT NULL,
			document_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

			PRIMARY KEY (bug_id, document_id),
			FOREIGN KEY (bug_id) REFERENCES bugs(id) ON DELETE CASCADE,
			FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create bug_documents table: %w", err)
	}

	// Indexes for bug_documents
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_bug_documents_bug_id ON bug_documents(bug_id);`); err != nil {
		return fmt.Errorf("failed to create index on bug_documents(bug_id): %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_bug_documents_document_id ON bug_documents(document_id);`); err != nil {
		return fmt.Errorf("failed to create index on bug_documents(document_id): %w", err)
	}

	// Create change_card_documents junction table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS change_card_documents (
			change_card_id INTEGER NOT NULL,
			document_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

			PRIMARY KEY (change_card_id, document_id),
			FOREIGN KEY (change_card_id) REFERENCES change_cards(id) ON DELETE CASCADE,
			FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create change_card_documents table: %w", err)
	}

	// Indexes for change_card_documents
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_change_card_documents_change_card_id ON change_card_documents(change_card_id);`); err != nil {
		return fmt.Errorf("failed to create index on change_card_documents(change_card_id): %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_change_card_documents_document_id ON change_card_documents(document_id);`); err != nil {
		return fmt.Errorf("failed to create index on change_card_documents(document_id): %w", err)
	}

	return nil
}
