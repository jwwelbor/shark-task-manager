package db

import (
	"database/sql"
	"fmt"
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

	// Step 2: Create new table without agent_type constraint
	_, err = tx.Exec(`
CREATE TABLE tasks (
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
