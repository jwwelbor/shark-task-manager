package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRestoreDump_RebuildsFTS(t *testing.T) {
	dir := t.TempDir()
	dumpPath := filepath.Join(dir, "dump.sql")
	outputPath := filepath.Join(dir, "restored.db")

	dumpSQL := `
PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE epics (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	key TEXT NOT NULL UNIQUE,
	title TEXT NOT NULL,
	description TEXT,
	status TEXT NOT NULL,
	priority TEXT NOT NULL,
	business_value TEXT,
	file_path TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE features (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	epic_id INTEGER NOT NULL,
	key TEXT NOT NULL UNIQUE,
	title TEXT NOT NULL,
	description TEXT,
	status TEXT NOT NULL,
	progress_pct REAL NOT NULL DEFAULT 0.0,
	execution_order INTEGER NULL,
	file_path TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE tasks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	feature_id INTEGER NOT NULL,
	key TEXT NOT NULL UNIQUE,
	title TEXT NOT NULL,
	description TEXT,
	status TEXT NOT NULL,
	agent_type TEXT,
	priority INTEGER NOT NULL DEFAULT 5,
	depends_on TEXT,
	assigned_agent TEXT,
	file_path TEXT,
	blocked_reason TEXT,
	execution_order INTEGER NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	started_at TIMESTAMP,
	completed_at TIMESTAMP,
	blocked_at TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE schema_version (
	version INTEGER NOT NULL,
	applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO epics (id, key, title, status, priority) VALUES (1, 'E01', 'Epic', 'todo', 'high');
INSERT INTO features (id, epic_id, key, title, status) VALUES (1, 1, 'E01-F01', 'Feature', 'todo');
INSERT INTO tasks (id, feature_id, key, title, description, status, priority) VALUES (1, 1, 'T-E01-F01-001', 'Task', 'Backfill restore body', 'todo', 5);
INSERT INTO schema_version (version) VALUES (25);
COMMIT;
`
	if err := os.WriteFile(dumpPath, []byte(dumpSQL), 0o644); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	if err := restoreDump(dumpPath, outputPath); err != nil {
		t.Fatalf("restoreDump: %v", err)
	}

	db, err := sql.Open("sqlite", outputPath)
	if err != nil {
		t.Fatalf("open restored db: %v", err)
	}
	defer db.Close()

	var ftsCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='entity_search_fts'`).Scan(&ftsCount); err != nil {
		t.Fatalf("query entity_search_fts existence: %v", err)
	}
	if ftsCount != 1 {
		t.Fatalf("entity_search_fts existence count = %d, want 1", ftsCount)
	}

	var matchCount int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM entity_search_fts
		WHERE key = 'T-E01-F01-001'
		  AND entity_search_fts MATCH '"backfill"'
	`).Scan(&matchCount); err != nil {
		t.Fatalf("query restored FTS match: %v", err)
	}
	if matchCount != 1 {
		t.Fatalf("FTS backfill match count = %d, want 1", matchCount)
	}
}
