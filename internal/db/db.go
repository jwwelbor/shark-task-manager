package db

import (
	"database/sql"
	"fmt"

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
    status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'completed', 'archived')),
    priority TEXT NOT NULL CHECK (priority IN ('high', 'medium', 'low')),
    business_value TEXT CHECK (business_value IN ('high', 'medium', 'low')),
    file_path TEXT,
    custom_folder_path TEXT,
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
    status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'completed', 'archived')),
    progress_pct REAL NOT NULL DEFAULT 0.0 CHECK (progress_pct >= 0.0 AND progress_pct <= 100.0),
    execution_order INTEGER NULL,
    file_path TEXT,
    custom_folder_path TEXT,
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
    status TEXT NOT NULL CHECK (status IN ('todo', 'in_progress', 'blocked', 'ready_for_review', 'completed', 'archived')),
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

	// Check if epics table has custom_folder_path column; if not, add it
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'custom_folder_path'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check epics schema for custom_folder_path: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE epics ADD COLUMN custom_folder_path TEXT;`); err != nil {
			return fmt.Errorf("failed to add custom_folder_path to epics: %w", err)
		}
		if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_epics_custom_folder_path ON epics(custom_folder_path);`); err != nil {
			return fmt.Errorf("failed to create epics custom_folder_path index: %w", err)
		}
	}

	// Check if features table has custom_folder_path column; if not, add it
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'custom_folder_path'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check features schema for custom_folder_path: %w", err)
	}

	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE features ADD COLUMN custom_folder_path TEXT;`); err != nil {
			return fmt.Errorf("failed to add custom_folder_path to features: %w", err)
		}
		if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_features_custom_folder_path ON features(custom_folder_path);`); err != nil {
			return fmt.Errorf("failed to create features custom_folder_path index: %w", err)
		}
	}

	// Create indexes on new columns that might not have existed before
	// These are created here after migrations ensure the columns exist
	newIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_epics_file_path ON epics(file_path);`,
		`CREATE INDEX IF NOT EXISTS idx_epics_custom_folder_path ON epics(custom_folder_path);`,
		`CREATE INDEX IF NOT EXISTS idx_features_file_path ON features(file_path);`,
		`CREATE INDEX IF NOT EXISTS idx_features_custom_folder_path ON features(custom_folder_path);`,
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
			CREATE TABLE task_criteria (
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
