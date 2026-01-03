package db

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	// Create all tables, indexes, and triggers
	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Run migrations for backwards compatibility
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
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
        'question'         -- Unanswered questions
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
