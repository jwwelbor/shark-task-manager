package taskcreation

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*repository.DB, func()) {
	t.Helper()

	// Create temp database file
	tmpFile := t.TempDir() + "/test.db"

	// Initialize database with schema using db.InitDB
	sqlDB, err := sql.Open("sqlite3", tmpFile+"?_foreign_keys=on")
	require.NoError(t, err, "Failed to create database")

	// Configure SQLite and create schema
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON;")
	require.NoError(t, err, "Failed to enable foreign keys")

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
			execution_order INTEGER NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			blocked_at TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE
		);
	`
	_, err = sqlDB.Exec(schema)
	require.NoError(t, err, "Failed to create schema")

	db := repository.NewDB(sqlDB)

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func createTestEpic(t *testing.T, db *repository.DB, key string) *models.Epic {
	t.Helper()

	epicRepo := repository.NewEpicRepository(db)
	description := "Test Description"
	businessValue := models.PriorityHigh
	epic := &models.Epic{
		Key:           key,
		Title:         "Test Epic",
		Description:   &description,
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &businessValue,
	}

	err := epicRepo.Create(context.Background(), epic)
	require.NoError(t, err, "Failed to create test epic")

	return epic
}

func createTestFeature(t *testing.T, db *repository.DB, epicID int64, key string) *models.Feature {
	t.Helper()

	featureRepo := repository.NewFeatureRepository(db)
	description := "Test Description"
	feature := &models.Feature{
		EpicID:      epicID,
		Key:         key,
		Title:       "Test Feature",
		Description: &description,
		Status:      models.FeatureStatusActive,
		ProgressPct: 0.0,
	}

	err := featureRepo.Create(context.Background(), feature)
	require.NoError(t, err, "Failed to create test feature")

	return feature
}

func createTestTask(t *testing.T, db *repository.DB, featureID int64, key, title string) *models.Task {
	t.Helper()

	taskRepo := repository.NewTaskRepository(db)
	agentType := models.AgentTypeGeneral
	task := &models.Task{
		FeatureID:   featureID,
		Key:         key,
		Title:       title,
		Description: nil,
		Status:      models.TaskStatusTodo,
		AgentType:   &agentType,
		Priority:    5,
		DependsOn:   nil,
		FilePath:    nil,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err := taskRepo.Create(context.Background(), task)
	require.NoError(t, err, "Failed to create test task")

	return task
}

func TestKeyGenerator_GenerateTaskKey_FirstTask(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	_ = createTestFeature(t, db, epic.ID, "E01-F01")

	// Create key generator
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	kg := NewKeyGenerator(taskRepo, featureRepo)

	// Test
	key, err := kg.GenerateTaskKey(context.Background(), "E01", "F01")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "T-E01-F01-001", key)
}

func TestKeyGenerator_GenerateTaskKey_SecondTask(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	feature := createTestFeature(t, db, epic.ID, "E01-F02")
	createTestTask(t, db, feature.ID, "T-E01-F02-001", "First Task")

	// Create key generator
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	kg := NewKeyGenerator(taskRepo, featureRepo)

	// Test
	key, err := kg.GenerateTaskKey(context.Background(), "E01", "F02")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "T-E01-F02-002", key)
}

func TestKeyGenerator_GenerateTaskKey_MultipleTasksWithGaps(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup - create tasks with gaps (001, 003, 005)
	epic := createTestEpic(t, db, "E01")
	feature := createTestFeature(t, db, epic.ID, "E01-F03")
	createTestTask(t, db, feature.ID, "T-E01-F03-001", "First Task")
	createTestTask(t, db, feature.ID, "T-E01-F03-003", "Third Task")
	createTestTask(t, db, feature.ID, "T-E01-F03-005", "Fifth Task")

	// Create key generator
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	kg := NewKeyGenerator(taskRepo, featureRepo)

	// Test - should generate 006 (one after the highest)
	key, err := kg.GenerateTaskKey(context.Background(), "E01", "F03")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "T-E01-F03-006", key)
}

func TestKeyGenerator_GenerateTaskKey_NormalizeFeatureKey(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	_ = createTestFeature(t, db, epic.ID, "E01-F04")

	// Create key generator
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	kg := NewKeyGenerator(taskRepo, featureRepo)

	// Test with short form (F04 instead of E01-F04)
	key, err := kg.GenerateTaskKey(context.Background(), "E01", "F04")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "T-E01-F04-001", key)
}

func TestKeyGenerator_GenerateTaskKey_FullFeatureKey(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E02")
	_ = createTestFeature(t, db, epic.ID, "E02-F05")

	// Create key generator
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	kg := NewKeyGenerator(taskRepo, featureRepo)

	// Test with full form (E02-F05)
	key, err := kg.GenerateTaskKey(context.Background(), "E02", "E02-F05")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "T-E02-F05-001", key)
}

func TestKeyGenerator_GenerateTaskKey_NonExistentFeature(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup - create epic but no feature
	createTestEpic(t, db, "E01")

	// Create key generator
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	kg := NewKeyGenerator(taskRepo, featureRepo)

	// Test
	key, err := kg.GenerateTaskKey(context.Background(), "E01", "F99")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "feature E01-F99 does not exist")
	assert.Equal(t, "", key)
}

func TestKeyGenerator_GenerateTaskKey_MaxTaskCount(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	feature := createTestFeature(t, db, epic.ID, "E01-F06")

	// Create a task with key 999 to simulate max
	createTestTask(t, db, feature.ID, "T-E01-F06-999", "Task 999")

	// Create key generator
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	kg := NewKeyGenerator(taskRepo, featureRepo)

	// Test
	key, err := kg.GenerateTaskKey(context.Background(), "E01", "F06")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has reached maximum task count (999)")
	assert.Equal(t, "", key)
}

func TestKeyGenerator_GenerateTaskKey_ThreeDigitNumbers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	feature := createTestFeature(t, db, epic.ID, "E01-F07")

	// Create task 099
	createTestTask(t, db, feature.ID, "T-E01-F07-099", "Task 99")

	// Create key generator
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	kg := NewKeyGenerator(taskRepo, featureRepo)

	// Test - should generate 100
	key, err := kg.GenerateTaskKey(context.Background(), "E01", "F07")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "T-E01-F07-100", key)
}

func TestExtractNumberFromKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected int
	}{
		{
			name:     "Single digit",
			key:      "T-E01-F02-001",
			expected: 1,
		},
		{
			name:     "Two digits",
			key:      "T-E01-F02-042",
			expected: 42,
		},
		{
			name:     "Three digits",
			key:      "T-E01-F02-999",
			expected: 999,
		},
		{
			name:     "Invalid format",
			key:      "INVALID",
			expected: 0,
		},
		{
			name:     "Missing task prefix",
			key:      "E01-F02-001",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNumberFromKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeFeatureKey(t *testing.T) {
	tests := []struct {
		name       string
		epicKey    string
		featureKey string
		expected   string
	}{
		{
			name:       "Short form",
			epicKey:    "E01",
			featureKey: "F02",
			expected:   "E01-F02",
		},
		{
			name:       "Full form",
			epicKey:    "E01",
			featureKey: "E01-F02",
			expected:   "E01-F02",
		},
		{
			name:       "Different epic in full form",
			epicKey:    "E02",
			featureKey: "E01-F02",
			expected:   "E01-F02",
		},
		{
			name:       "Invalid format",
			epicKey:    "E01",
			featureKey: "INVALID",
			expected:   "INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeFeatureKey(tt.epicKey, tt.featureKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFeaturePart(t *testing.T) {
	tests := []struct {
		name             string
		fullFeatureKey   string
		expected         string
	}{
		{
			name:           "Standard format",
			fullFeatureKey: "E01-F02",
			expected:       "F02",
		},
		{
			name:           "Different epic",
			fullFeatureKey: "E99-F42",
			expected:       "F42",
		},
		{
			name:           "Short form",
			fullFeatureKey: "F05",
			expected:       "F05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFeaturePart(tt.fullFeatureKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}
