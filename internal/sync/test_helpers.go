package sync

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/require"
)

// setupTestDatabase creates and initializes a test database
func setupTestDatabase(tb testing.TB, dbPath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	require.NoError(tb, err)

	// Create schema
	schema := `
		CREATE TABLE IF NOT EXISTS epics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL,
			priority TEXT NOT NULL,
			business_value TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS features (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			epic_id INTEGER NOT NULL,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL,
			progress_pct REAL NOT NULL DEFAULT 0.0,
			execution_order INTEGER,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS tasks (
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
			execution_order INTEGER,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			blocked_at TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS task_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER NOT NULL,
			status_from TEXT,
			status_to TEXT,
			changed_by TEXT NOT NULL,
			change_description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (task_id) REFERENCES tasks(id)
		);
	`
	_, err = db.Exec(schema)
	require.NoError(tb, err)

	return db
}

// setupTestEpicAndFeature creates a test epic and feature
func setupTestEpicAndFeature(tb testing.TB, db *sql.DB) {
	ctx := context.Background()
	repoDb := repository.NewDB(db)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Create epic
	epic := &models.Epic{
		Key:      "E01",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(tb, err)

	// Create feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E01-F01",
		Title:  "Test Feature",
		Status: models.FeatureStatusActive,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(tb, err)
}

// createTestTaskFile creates a task file with the given key and title
func createTestTaskFile(tb testing.TB, filePath, taskKey, title string) {
	content := fmt.Sprintf(`---
task_key: %s
status: todo
---

# Task: %s

## Description

Test task description for %s.
`, taskKey, title, taskKey)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(tb, err)
}

// updateTestTaskFile updates a task file with new content
func updateTestTaskFile(tb testing.TB, filePath, taskKey, title string) {
	content := fmt.Sprintf(`---
task_key: %s
status: todo
---

# Task: %s

## Description

Updated test task description for %s.
`, taskKey, title, taskKey)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(tb, err)
}
