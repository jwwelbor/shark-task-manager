package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// MigrateAddExecutionOrder adds execution_order column to features and tasks tables
func MigrateAddExecutionOrder(db *sql.DB) error {
	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Add execution_order to features table
	_, err = tx.Exec(`ALTER TABLE features ADD COLUMN execution_order INTEGER NULL;`)
	if err != nil {
		return fmt.Errorf("failed to add execution_order to features: %w", err)
	}

	// Add execution_order to tasks table
	_, err = tx.Exec(`ALTER TABLE tasks ADD COLUMN execution_order INTEGER NULL;`)
	if err != nil {
		return fmt.Errorf("failed to add execution_order to tasks: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// MigrateRemoveAgentTypeConstraint removes the CHECK constraint on agent_type
// This allows any string value for agent_type instead of limiting to predefined types
func MigrateRemoveAgentTypeConstraint(db *sql.DB) error {
	// SQLite doesn't support ALTER TABLE to modify constraints directly
	// We need to recreate the table

	// Temporarily disable foreign key constraints for migration
	_, err := db.Exec("PRAGMA foreign_keys = OFF;")
	if err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}
	defer func() {
		_, _ = db.Exec("PRAGMA foreign_keys = ON;")
	}()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Step 1: Rename old table
	_, err = tx.Exec(`ALTER TABLE tasks RENAME TO tasks_old;`)
	if err != nil {
		return fmt.Errorf("failed to rename tasks table: %w", err)
	}

	// Step 2: Create new table without agent_type or status constraints
	_, err = tx.Exec(`
CREATE TABLE tasks (
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
);`)
	if err != nil {
		return fmt.Errorf("failed to create new tasks table: %w", err)
	}

	// Step 3: Copy data from old table
	_, err = tx.Exec(`
INSERT INTO tasks (
    id, feature_id, key, title, description, status, agent_type, priority,
    depends_on, assigned_agent, file_path, blocked_reason, execution_order, created_at,
    started_at, completed_at, blocked_at, updated_at
)
SELECT
    id, feature_id, key, title, description, status, agent_type, priority,
    depends_on, assigned_agent, file_path, blocked_reason, execution_order, created_at,
    started_at, completed_at, blocked_at, updated_at
FROM tasks_old;`)
	if err != nil {
		return fmt.Errorf("failed to copy data from old table: %w", err)
	}

	// Step 4: Recreate indexes
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_key ON tasks(key);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_feature_id ON tasks(feature_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_agent_type ON tasks(agent_type);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status_priority ON tasks(status, priority);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_file_path ON tasks(file_path);`,
	}

	for _, idx := range indexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Step 5: Recreate trigger
	// Drop old trigger first
	_, _ = tx.Exec(`DROP TRIGGER IF EXISTS tasks_updated_at;`)

	_, err = tx.Exec(`
CREATE TRIGGER IF NOT EXISTS tasks_updated_at
AFTER UPDATE ON tasks
FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;`)
	if err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	// Step 6: Drop old table
	_, err = tx.Exec(`DROP TABLE tasks_old;`)
	if err != nil {
		return fmt.Errorf("failed to drop old table: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// MigrateRemoveStatusCheckConstraints removes CHECK constraints on status columns
// from epics, features, and tasks tables to allow workflow-defined statuses
func MigrateRemoveStatusCheckConstraints(db *sql.DB) error {
	// Check if migrations are needed by inspecting table schemas
	needsMigration, err := needsStatusConstraintRemoval(db)
	if err != nil {
		return fmt.Errorf("failed to check if status constraint removal needed: %w", err)
	}

	if !needsMigration {
		// Already migrated or created with new schema
		return nil
	}

	// Migrate tasks table
	if err := migrateTasksStatusConstraint(db); err != nil {
		return fmt.Errorf("failed to migrate tasks status constraint: %w", err)
	}

	// Migrate epics table
	if err := migrateEpicsStatusConstraint(db); err != nil {
		return fmt.Errorf("failed to migrate epics status constraint: %w", err)
	}

	// Migrate features table
	if err := migrateFeaturesStatusConstraint(db); err != nil {
		return fmt.Errorf("failed to migrate features status constraint: %w", err)
	}

	return nil
}

// needsStatusConstraintRemoval checks if any table has CHECK constraints on status column
func needsStatusConstraintRemoval(db *sql.DB) (bool, error) {
	tables := []string{"tasks", "epics", "features"}

	for _, table := range tables {
		var sql string
		err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&sql)
		if err != nil {
			return false, fmt.Errorf("failed to get schema for %s: %w", table, err)
		}

		// Check if the CREATE TABLE statement contains a CHECK constraint specifically on "status TEXT"
		// Must look for "status TEXT NOT NULL CHECK" pattern to avoid matching verification_status
		if strings.Contains(sql, "status TEXT NOT NULL CHECK") {
			return true, nil
		}
	}

	return false, nil
}

// migrateTaskHistoryForeignKey fixes the task_history foreign key reference
// This migration handles databases where the tasks table was already migrated
// but task_history still references the old "tasks_old" table
func migrateTaskHistoryForeignKey(db *sql.DB) error {
	// Check if task_history has the wrong foreign key
	var schema string
	err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='task_history'").Scan(&schema)
	if err != nil {
		return fmt.Errorf("failed to get task_history schema: %w", err)
	}

	// If schema references tasks_old, we need to fix it
	if !strings.Contains(schema, "tasks_old") {
		// Already fixed
		return nil
	}

	// Temporarily disable foreign key constraints
	_, err = db.Exec("PRAGMA foreign_keys = OFF;")
	if err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}
	defer func() {
		_, _ = db.Exec("PRAGMA foreign_keys = ON;")
	}()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Rename task_history to task_history_old
	_, err = tx.Exec(`ALTER TABLE task_history RENAME TO task_history_old;`)
	if err != nil {
		return fmt.Errorf("failed to rename task_history table: %w", err)
	}

	// Create new task_history table with correct foreign key
	_, err = tx.Exec(`
CREATE TABLE task_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    old_status TEXT,
    new_status TEXT NOT NULL,
    agent TEXT,
    notes TEXT,
    forced BOOLEAN DEFAULT FALSE,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);`)
	if err != nil {
		return fmt.Errorf("failed to create new task_history table: %w", err)
	}

	// Copy history data from old table
	_, err = tx.Exec(`
INSERT INTO task_history (id, task_id, old_status, new_status, agent, notes, forced, timestamp)
SELECT id, task_id, old_status, new_status, agent, notes, forced, timestamp
FROM task_history_old;`)
	if err != nil {
		return fmt.Errorf("failed to copy task_history data: %w", err)
	}

	// Recreate indexes for task_history
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_task_history_task_id ON task_history(task_id);`,
		`CREATE INDEX IF NOT EXISTS idx_task_history_timestamp ON task_history(timestamp DESC);`,
	}
	for _, idx := range indexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("failed to create task_history index: %w", err)
		}
	}

	// Drop old table
	_, err = tx.Exec(`DROP TABLE task_history_old;`)
	if err != nil {
		return fmt.Errorf("failed to drop old task_history table: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// migrateTasksStatusConstraint removes status CHECK constraint from tasks table
func migrateTasksStatusConstraint(db *sql.DB) error {
	// Get current table schema to check if it has the constraint
	var createSQL string
	err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='tasks'").Scan(&createSQL)
	if err != nil {
		return fmt.Errorf("failed to get tasks schema: %w", err)
	}

	// If no CHECK constraint on status column, nothing to do
	// Look for "status TEXT NOT NULL CHECK" pattern specifically
	if !strings.Contains(createSQL, "status TEXT NOT NULL CHECK") {
		return nil
	}

	// Temporarily disable foreign key constraints for migration
	_, err = db.Exec("PRAGMA foreign_keys = OFF;")
	if err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}
	defer func() {
		_, _ = db.Exec("PRAGMA foreign_keys = ON;")
	}()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Step 1: Rename old table
	_, err = tx.Exec(`ALTER TABLE tasks RENAME TO tasks_old;`)
	if err != nil {
		return fmt.Errorf("failed to rename tasks table: %w", err)
	}

	// Step 2: Create new table without status constraint
	// Get column list from old table to ensure we copy all columns
	_, err = tx.Exec(`
CREATE TABLE tasks (
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
    completed_by TEXT,
    completion_notes TEXT,
    files_changed TEXT,
    tests_passed BOOLEAN DEFAULT 0,
    verification_status TEXT CHECK(verification_status IN ('pending', 'verified', 'needs_rework')) DEFAULT 'pending',
    time_spent_minutes INTEGER,
    context_data TEXT,
    slug TEXT,

    FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE
);`)
	if err != nil {
		return fmt.Errorf("failed to create new tasks table: %w", err)
	}

	// Step 3: Copy data from old table - explicitly list columns
	_, err = tx.Exec(`
INSERT INTO tasks (id, feature_id, key, title, description, status, agent_type, priority, depends_on, assigned_agent,
                   file_path, blocked_reason, execution_order, created_at, started_at, completed_at, blocked_at, updated_at,
                   completed_by, completion_notes, files_changed, tests_passed, verification_status, time_spent_minutes,
                   context_data, slug)
SELECT id, feature_id, key, title, description, status, agent_type, priority, depends_on, assigned_agent,
       file_path, blocked_reason, execution_order, created_at, started_at, completed_at, blocked_at, updated_at,
       completed_by, completion_notes, files_changed, tests_passed, verification_status, time_spent_minutes,
       context_data, slug
FROM tasks_old;`)
	if err != nil {
		return fmt.Errorf("failed to copy data from old table: %w", err)
	}

	// Step 4: Recreate indexes
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_key ON tasks(key);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_feature_id ON tasks(feature_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_agent_type ON tasks(agent_type);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status_priority ON tasks(status, priority);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_file_path ON tasks(file_path);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_completed_by ON tasks(completed_by);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_verification_status ON tasks(verification_status);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_slug ON tasks(slug);`,
	}

	for _, idx := range indexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Step 5: Recreate trigger
	_, _ = tx.Exec(`DROP TRIGGER IF EXISTS tasks_updated_at;`)
	_, err = tx.Exec(`
CREATE TRIGGER IF NOT EXISTS tasks_updated_at
AFTER UPDATE ON tasks
FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;`)
	if err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	// Step 6: Recreate task_history table with correct foreign key
	// First, rename task_history to task_history_old
	_, err = tx.Exec(`ALTER TABLE task_history RENAME TO task_history_old;`)
	if err != nil {
		return fmt.Errorf("failed to rename task_history table: %w", err)
	}

	// Create new task_history table with correct foreign key
	_, err = tx.Exec(`
CREATE TABLE task_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    old_status TEXT,
    new_status TEXT NOT NULL,
    agent TEXT,
    notes TEXT,
    forced BOOLEAN DEFAULT FALSE,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);`)
	if err != nil {
		return fmt.Errorf("failed to create new task_history table: %w", err)
	}

	// Copy history data from old table
	_, err = tx.Exec(`
INSERT INTO task_history (id, task_id, old_status, new_status, agent, notes, forced, timestamp)
SELECT id, task_id, old_status, new_status, agent, notes, forced, timestamp
FROM task_history_old;`)
	if err != nil {
		return fmt.Errorf("failed to copy task_history data: %w", err)
	}

	// Recreate indexes for task_history
	historyIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_task_history_task_id ON task_history(task_id);`,
		`CREATE INDEX IF NOT EXISTS idx_task_history_timestamp ON task_history(timestamp DESC);`,
	}
	for _, idx := range historyIndexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("failed to create task_history index: %w", err)
		}
	}

	// Step 7: Drop old tables
	_, err = tx.Exec(`DROP TABLE task_history_old;`)
	if err != nil {
		return fmt.Errorf("failed to drop old task_history table: %w", err)
	}

	_, err = tx.Exec(`DROP TABLE tasks_old;`)
	if err != nil {
		return fmt.Errorf("failed to drop old tasks table: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// migrateEpicsStatusConstraint removes status CHECK constraint from epics table
func migrateEpicsStatusConstraint(db *sql.DB) error {
	// Get current table schema
	var createSQL string
	err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='epics'").Scan(&createSQL)
	if err != nil {
		return fmt.Errorf("failed to get epics schema: %w", err)
	}

	// If no CHECK constraint on status column, nothing to do
	// Look for "status TEXT NOT NULL CHECK" pattern specifically
	if !strings.Contains(createSQL, "status TEXT NOT NULL CHECK") {
		return nil
	}

	// Temporarily disable foreign key constraints
	_, err = db.Exec("PRAGMA foreign_keys = OFF;")
	if err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}
	defer func() {
		_, _ = db.Exec("PRAGMA foreign_keys = ON;")
	}()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Rename old table
	_, err = tx.Exec(`ALTER TABLE epics RENAME TO epics_old;`)
	if err != nil {
		return fmt.Errorf("failed to rename epics table: %w", err)
	}

	// Create new table without status constraint
	_, err = tx.Exec(`
CREATE TABLE epics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    priority TEXT NOT NULL CHECK (priority IN ('high', 'medium', 'low')),
    business_value TEXT CHECK (business_value IN ('high', 'medium', 'low')),
    file_path TEXT,
    custom_folder_path TEXT,
    slug TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);`)
	if err != nil {
		return fmt.Errorf("failed to create new epics table: %w", err)
	}

	// Copy data - explicitly list columns to handle any schema differences
	_, err = tx.Exec(`
INSERT INTO epics (id, key, title, description, status, priority, business_value, file_path, custom_folder_path, slug, created_at, updated_at)
SELECT id, key, title, description, status, priority, business_value, file_path, custom_folder_path, slug, created_at, updated_at
FROM epics_old;`)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Recreate indexes
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_epics_key ON epics(key);`,
		`CREATE INDEX IF NOT EXISTS idx_epics_status ON epics(status);`,
		`CREATE INDEX IF NOT EXISTS idx_epics_file_path ON epics(file_path);`,
		`CREATE INDEX IF NOT EXISTS idx_epics_custom_folder_path ON epics(custom_folder_path);`,
		`CREATE INDEX IF NOT EXISTS idx_epics_slug ON epics(slug);`,
	}
	for _, idx := range indexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Recreate trigger
	_, _ = tx.Exec(`DROP TRIGGER IF EXISTS epics_updated_at;`)
	_, err = tx.Exec(`
CREATE TRIGGER IF NOT EXISTS epics_updated_at
AFTER UPDATE ON epics
FOR EACH ROW
BEGIN
    UPDATE epics SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;`)
	if err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	// Drop old table
	_, err = tx.Exec(`DROP TABLE epics_old;`)
	if err != nil {
		return fmt.Errorf("failed to drop old table: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// migrateFeaturesStatusConstraint removes status CHECK constraint from features table
func migrateFeaturesStatusConstraint(db *sql.DB) error {
	// Get current table schema
	var createSQL string
	err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='features'").Scan(&createSQL)
	if err != nil {
		return fmt.Errorf("failed to get features schema: %w", err)
	}

	// If no CHECK constraint on status column, nothing to do
	// Look for "status TEXT NOT NULL CHECK" pattern specifically
	if !strings.Contains(createSQL, "status TEXT NOT NULL CHECK") {
		return nil
	}

	// Temporarily disable foreign key constraints
	_, err = db.Exec("PRAGMA foreign_keys = OFF;")
	if err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}
	defer func() {
		_, _ = db.Exec("PRAGMA foreign_keys = ON;")
	}()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Rename old table
	_, err = tx.Exec(`ALTER TABLE features RENAME TO features_old;`)
	if err != nil {
		return fmt.Errorf("failed to rename features table: %w", err)
	}

	// Create new table without status constraint
	_, err = tx.Exec(`
CREATE TABLE features (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    epic_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    progress_pct REAL NOT NULL DEFAULT 0.0 CHECK (progress_pct >= 0.0 AND progress_pct <= 100.0),
    execution_order INTEGER NULL,
    file_path TEXT,
    custom_folder_path TEXT,
    slug TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE
);`)
	if err != nil {
		return fmt.Errorf("failed to create new features table: %w", err)
	}

	// Copy data - explicitly list columns to handle any schema differences
	_, err = tx.Exec(`
INSERT INTO features (id, epic_id, key, title, description, status, progress_pct, execution_order, file_path, custom_folder_path, slug, created_at, updated_at)
SELECT id, epic_id, key, title, description, status, progress_pct, execution_order, file_path, custom_folder_path, slug, created_at, updated_at
FROM features_old;`)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Recreate indexes
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_features_key ON features(key);`,
		`CREATE INDEX IF NOT EXISTS idx_features_epic_id ON features(epic_id);`,
		`CREATE INDEX IF NOT EXISTS idx_features_status ON features(status);`,
		`CREATE INDEX IF NOT EXISTS idx_features_file_path ON features(file_path);`,
		`CREATE INDEX IF NOT EXISTS idx_features_custom_folder_path ON features(custom_folder_path);`,
		`CREATE INDEX IF NOT EXISTS idx_features_slug ON features(slug);`,
	}
	for _, idx := range indexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Recreate trigger
	_, _ = tx.Exec(`DROP TRIGGER IF EXISTS features_updated_at;`)
	_, err = tx.Exec(`
CREATE TRIGGER IF NOT EXISTS features_updated_at
AFTER UPDATE ON features
FOR EACH ROW
BEGIN
    UPDATE features SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;`)
	if err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	// Drop old table
	_, err = tx.Exec(`DROP TABLE features_old;`)
	if err != nil {
		return fmt.Errorf("failed to drop old table: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
