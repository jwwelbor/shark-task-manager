package sync

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/mattn/go-sqlite3"
)

// TestConcurrentFileAndDatabaseChanges tests the complete scenario where:
// 1. A task exists in both file and database
// 2. Last sync happened at T0
// 3. File was modified at T1 (after T0)
// 4. Database was modified at T2 (after T0)
// 5. Sync detects conflict and applies resolution strategy
func TestConcurrentFileAndDatabaseChanges(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "shark-conflict-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test database
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	require.NoError(t, err)
	defer db.Close()

	// Initialize database schema
	err = initTestSchema(db)
	require.NoError(t, err)

	ctx := context.Background()

	// Create repositories
	repoDb := repository.NewDB(db)
	taskRepo := repository.NewTaskRepository(repoDb)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Step 1: Create epic and feature
	epic := &models.Epic{
		Key:      "E04",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err = epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E04-F07",
		Title:  "Test Feature",
		Status: models.FeatureStatusActive,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Step 2: Create initial task in database at T0
	t0 := time.Now().Add(-3 * time.Hour) // 3 hours ago
	initialTitle := "Initial Title"
	initialDesc := "Initial Description"
	filePath := filepath.Join(tempDir, "T-E04-F07-001.md")

	task := &models.Task{
		FeatureID:   feature.ID,
		Key:         "T-E04-F07-001",
		Title:       initialTitle,
		Description: &initialDesc,
		Status:      models.TaskStatusTodo,
		Priority:    5,
		FilePath:    &filePath,
	}

	// Manually insert with specific timestamps
	_, err = db.ExecContext(ctx, `
		INSERT INTO tasks (feature_id, key, title, description, status, priority, file_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.FeatureID, task.Key, task.Title, task.Description, task.Status, task.Priority, task.FilePath, t0, t0)
	require.NoError(t, err)

	// Step 3: Last sync happened at T0 (same time as initial creation)
	lastSyncTime := t0

	// Step 4: Modify file at T1 (1 hour ago)
	t1 := time.Now().Add(-1 * time.Hour)
	fileTitle := "File Modified Title"
	fileDesc := "File Modified Description"

	// Create task file with new content
	fileContent := `---
task_key: T-E04-F07-001
status: todo
---

# Task: ` + fileTitle + `

` + fileDesc + `
`
	err = os.WriteFile(filePath, []byte(fileContent), 0644)
	require.NoError(t, err)

	// Set file mtime to T1
	err = os.Chtimes(filePath, t1, t1)
	require.NoError(t, err)

	// Step 5: Modify database at T2 (30 minutes ago)
	t2 := time.Now().Add(-30 * time.Minute)
	dbTitle := "Database Modified Title"
	dbDesc := "Database Modified Description"

	_, err = db.ExecContext(ctx, `
		UPDATE tasks
		SET title = ?, description = ?, updated_at = ?
		WHERE key = ?
	`, dbTitle, dbDesc, t2, "T-E04-F07-001")
	require.NoError(t, err)

	// Step 6: Run conflict detection
	detector := NewConflictDetector()

	// Get current task from database
	dbTask, err := taskRepo.GetByKey(ctx, "T-E04-F07-001")
	require.NoError(t, err)

	// Parse file metadata
	fileData := &TaskMetadata{
		Key:         "T-E04-F07-001",
		Title:       fileTitle,
		Description: &fileDesc,
		FilePath:    filePath,
		ModifiedAt:  t1,
	}

	// Detect conflicts with last sync time
	conflicts := detector.DetectConflictsWithSync(fileData, dbTask, &lastSyncTime)

	// Step 7: Verify conflict detection
	assert.Len(t, conflicts, 2, "Expected 2 conflicts (title and description)")

	// Check that conflicts include title and description
	conflictFields := make(map[string]Conflict)
	for _, c := range conflicts {
		conflictFields[c.Field] = c
	}

	// Verify title conflict
	titleConflict, hasTitleConflict := conflictFields["title"]
	assert.True(t, hasTitleConflict, "Expected title conflict")
	if hasTitleConflict {
		assert.Equal(t, fileTitle, titleConflict.FileValue)
		assert.Equal(t, dbTitle, titleConflict.DatabaseValue)
	}

	// Verify description conflict
	descConflict, hasDescConflict := conflictFields["description"]
	assert.True(t, hasDescConflict, "Expected description conflict")
	if hasDescConflict {
		assert.Equal(t, fileDesc, descConflict.FileValue)
		assert.Equal(t, dbDesc, descConflict.DatabaseValue)
	}

	// Step 8: Test file-wins resolution
	t.Run("file-wins resolution", func(t *testing.T) {
		resolver := NewConflictResolver()
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyFileWins)

		require.NoError(t, err)
		assert.Equal(t, fileTitle, resolved.Title, "Title should be from file")
		assert.Equal(t, fileDesc, *resolved.Description, "Description should be from file")
		assert.Equal(t, models.TaskStatusTodo, resolved.Status, "Status should be preserved from DB")
		assert.Equal(t, 5, resolved.Priority, "Priority should be preserved from DB")
	})

	// Step 9: Test database-wins resolution
	t.Run("database-wins resolution", func(t *testing.T) {
		resolver := NewConflictResolver()
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyDatabaseWins)

		require.NoError(t, err)
		assert.Equal(t, dbTitle, resolved.Title, "Title should be from database")
		assert.Equal(t, dbDesc, *resolved.Description, "Description should be from database")
	})

	// Step 10: Test newer-wins resolution (file is newer: T1 > T2? No, T1 is older)
	t.Run("newer-wins resolution", func(t *testing.T) {
		resolver := NewConflictResolver()
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyNewerWins)

		require.NoError(t, err)

		// Database was updated at T2 (30 min ago), file at T1 (1 hour ago)
		// So database is newer, should use database values
		assert.Equal(t, dbTitle, resolved.Title, "Title should be from database (newer)")
		assert.Equal(t, dbDesc, *resolved.Description, "Description should be from database (newer)")
	})

	// Step 11: Test that without last_sync_time, conflicts are still detected
	t.Run("detection without last_sync_time", func(t *testing.T) {
		conflicts := detector.DetectConflictsWithSync(fileData, dbTask, nil)

		// Should still detect conflicts based on field differences
		assert.Len(t, conflicts, 2, "Expected conflicts even without last_sync_time")
	})
}

// TestNoConflictWhenOnlyFileModified tests that when only file is modified,
// no conflict is reported (normal file update scenario)
func TestNoConflictWhenOnlyFileModified(t *testing.T) {
	detector := NewConflictDetector()
	now := time.Now()

	lastSync := now.Add(-2 * time.Hour)
	fileMTime := now.Add(-1 * time.Hour)    // File modified after last sync
	dbUpdateTime := now.Add(-3 * time.Hour) // DB not modified since last sync

	filePath := "/path/to/task.md"
	fileData := &TaskMetadata{
		Key:        "T-E04-F07-001",
		Title:      "New Title",
		FilePath:   filePath,
		ModifiedAt: fileMTime,
	}

	dbTask := &models.Task{
		ID:        1,
		Key:       "T-E04-F07-001",
		Title:     "Old Title",
		FilePath:  &filePath,
		Status:    models.TaskStatusTodo,
		Priority:  5,
		UpdatedAt: dbUpdateTime,
	}

	conflicts := detector.DetectConflictsWithSync(fileData, dbTask, &lastSync)

	// No conflict - this is just a file update
	assert.Empty(t, conflicts, "Expected no conflicts when only file modified")
}

// TestNoConflictWhenOnlyDatabaseModified tests that when only database is modified,
// no conflict is reported (DB is current, skip file)
func TestNoConflictWhenOnlyDatabaseModified(t *testing.T) {
	detector := NewConflictDetector()
	now := time.Now()

	lastSync := now.Add(-2 * time.Hour)
	fileMTime := now.Add(-3 * time.Hour)    // File not modified since last sync
	dbUpdateTime := now.Add(-1 * time.Hour) // DB modified after last sync

	filePath := "/path/to/task.md"
	fileData := &TaskMetadata{
		Key:        "T-E04-F07-001",
		Title:      "Old Title",
		FilePath:   filePath,
		ModifiedAt: fileMTime,
	}

	dbTask := &models.Task{
		ID:        1,
		Key:       "T-E04-F07-001",
		Title:     "New Title",
		FilePath:  &filePath,
		Status:    models.TaskStatusTodo,
		Priority:  5,
		UpdatedAt: dbUpdateTime,
	}

	conflicts := detector.DetectConflictsWithSync(fileData, dbTask, &lastSync)

	// No conflict - DB is current, file is stale
	assert.Empty(t, conflicts, "Expected no conflicts when only database modified")
}

// initTestSchema creates the minimal database schema for testing
func initTestSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS epics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL,
		priority TEXT NOT NULL,
		business_value TEXT,
		file_path TEXT,
		custom_folder_path TEXT,
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
		execution_order INTEGER NULL,
		file_path TEXT,
		custom_folder_path TEXT,
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
	`

	_, err := db.Exec(schema)
	return err
}
