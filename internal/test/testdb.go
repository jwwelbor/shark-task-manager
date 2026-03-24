package test

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
)

var (
	testDB    *sql.DB
	dbOnce    sync.Once
	dbDirOnce sync.Once
	dbDir     string
)

// initDBDir initialises dbDir to a unique temp directory per test-binary
// invocation.  Using os.MkdirTemp guarantees that concurrent test binaries
// (i.e. packages running in parallel under go test -p N) each get their own
// private directory and never share the same SQLite file.
func initDBDir() {
	dbDirOnce.Do(func() {
		var err error
		dbDir, err = os.MkdirTemp("", "shark-test-*")
		if err != nil {
			panic("test: failed to create temp DB directory: " + err.Error())
		}
	})
}

// GetTestDB returns a shared test database for the current test-binary
// invocation.  The database lives in a temp directory that is unique per
// binary, so packages running in parallel (go test -p N) never share the
// same SQLite file and do not interfere with each other.
//
// Within a single test-binary all calls return the same *sql.DB connection,
// preserving the original "shared singleton" semantics that existing tests
// rely on for setup/tear-down via DELETE statements.
func GetTestDB() *sql.DB {
	initDBDir()
	dbOnce.Do(func() {
		dbPath := filepath.Join(dbDir, "test-shark-tasks.db")

		var err error
		testDB, err = db.InitDB(dbPath)
		if err != nil {
			panic("Failed to initialize test database: " + err.Error())
		}
	})
	return testDB
}

// SeedTestData populates the test database with sample data using SQL
// Returns epic_id, feature_id for use in tests
func SeedTestData() (int64, int64) {
	database := GetTestDB()

	// Clean up any existing E99 data to ensure fresh state
	// Delete in reverse order of dependencies (tasks -> features -> epics)
	_, _ = database.Exec("DELETE FROM tasks WHERE key LIKE 'T-E99-%'")
	_, _ = database.Exec("DELETE FROM features WHERE key LIKE 'E99-%'")
	_, _ = database.Exec("DELETE FROM epics WHERE key = 'E99'")

	// Create epic via SQL to avoid import cycle
	result, err := database.Exec(`
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E99', 'Test Epic', 'Test epic', 'active', 'high')
	`)
	if err != nil {
		panic(fmt.Sprintf("Failed to insert epic: %v", err))
	}

	epicID, err := result.LastInsertId()
	if err != nil {
		panic(fmt.Sprintf("Failed to get epic LastInsertId: %v", err))
	}

	if epicID == 0 {
		panic("Failed to get valid epic ID after insert")
	}

	// Create feature
	result, err = database.Exec(`
		INSERT INTO features (epic_id, key, title, slug, description, status)
		VALUES (?, 'E99-F99', 'Test Feature', 'test-feature', 'Test feature', 'active')
	`, epicID)
	if err != nil {
		panic(fmt.Sprintf("Failed to insert feature: %v", err))
	}

	featureID, err := result.LastInsertId()
	if err != nil {
		panic(fmt.Sprintf("Failed to get feature LastInsertId: %v", err))
	}

	if featureID == 0 {
		panic("Failed to get valid feature ID after insert")
	}

	// Create test tasks
	_, err = database.Exec(`
		INSERT OR IGNORE INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
		VALUES
			(?, 'T-E99-F99-001', 'Completed Task', 'completed', 'backend', 1, '[]'),
			(?, 'T-E99-F99-002', 'Todo Task', 'todo', 'backend', 2, '[]'),
			(?, 'T-E99-F99-003', 'Task with Dependency', 'todo', 'backend', 3, '["T-E99-F99-001"]'),
			(?, 'T-E99-F99-004', 'Task with Incomplete Dependency', 'todo', 'backend', 4, '["T-E99-F99-002"]')
	`, featureID, featureID, featureID, featureID)
	if err != nil {
		// In parallel tests, E99-F99 feature might be deleted by another test between our INSERT and this point
		// FK constraint errors are acceptable here since tests that need this data will fail anyway
		// Don't panic on FK errors, just skip the task creation
		if !strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			panic(fmt.Sprintf("Failed to insert test tasks: %v", err))
		}
	}

	// Create E04 epic and feature for sync tests
	result, err = database.Exec(`INSERT OR IGNORE INTO epics (key, title, description, status, priority) VALUES ('E04', 'Task Management CLI Core', 'Core CLI functionality', 'active', 'high')`)
	if err != nil {
		panic(fmt.Sprintf("Failed to insert E04 epic: %v", err))
	}

	e04ID, err := result.LastInsertId()
	if err != nil {
		panic(fmt.Sprintf("Failed to get E04 epic LastInsertId: %v", err))
	}

	// If INSERT OR IGNORE didn't insert (already exists), query for existing ID
	if e04ID == 0 {
		err = database.QueryRow("SELECT id FROM epics WHERE key = 'E04'").Scan(&e04ID)
		if err != nil {
			panic(fmt.Sprintf("Failed to find epic E04: %v", err))
		}
	}

	_, err = database.Exec(`INSERT OR IGNORE INTO features (epic_id, key, title, description, status) VALUES (?, 'E04-F05', 'Task File Management', 'Task CRUD operations', 'active')`, e04ID)
	if err != nil {
		// In parallel tests, E04 epic might be deleted by another test between our INSERT and this point
		// FK constraint errors are acceptable here since E04-F05 is optional test data
		// Don't panic on FK errors, just skip the feature creation
		if !strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			panic(fmt.Sprintf("Failed to insert E04-F05 feature: %v", err))
		}
	}

	return epicID, featureID
}

// NewIsolatedTestDB creates a per-test isolated SQLite database in a temp directory.
//
// Unlike GetTestDB (which returns a single shared database), each call to
// NewIsolatedTestDB returns a fresh, empty database with the full schema applied.
// The database file is automatically deleted when the test finishes via t.Cleanup.
//
// Use this function when:
//   - Your test calls t.Parallel() and needs an independent database
//   - Your test mutates shared rows (e.g. E99, E04) that other tests also use
//   - You want true isolation without DELETE-before/after boilerplate
//
// Example:
//
//	func TestMyRepo_Create(t *testing.T) {
//	    t.Parallel()
//	    database := test.NewIsolatedTestDB(t)
//	    repo := NewMyRepository(NewDB(database))
//	    // test body – no cleanup needed
//	}
func NewIsolatedTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dir := t.TempDir() // automatically removed by t.Cleanup
	dbFile := filepath.Join(dir, "test-isolated.db")

	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("NewIsolatedTestDB: failed to initialise database: %v", err)
	}

	t.Cleanup(func() {
		if closeErr := database.Close(); closeErr != nil {
			t.Logf("NewIsolatedTestDB: error closing database: %v", closeErr)
		}
	})

	return database
}

// StringPtr returns a pointer to a string
func StringPtr(s string) *string {
	return &s
}

// PriorityPtr returns a pointer to a string as Priority type
func PriorityPtr(p string) *string {
	return &p
}

// GenerateUniqueKey generates a unique task key for testing
// Expects epicFeature like "E04-F05" and returns valid task keys
func GenerateUniqueKey(epicFeature string, i int) string {
	return fmt.Sprintf("T-%s-%03d", epicFeature, i)
}

// SeedTestDataWithKeys creates test data with custom epic and feature keys
// Returns epic_id, feature_id for use in tests
func SeedTestDataWithKeys(epicKey, featureKey string) (int64, int64) {
	database := GetTestDB()

	// Clean up any existing data with these keys to ensure fresh state
	_, _ = database.Exec("DELETE FROM tasks WHERE key LIKE ?", epicKey+"-%")
	_, _ = database.Exec("DELETE FROM features WHERE key = ?", featureKey)
	_, _ = database.Exec("DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic
	result, err := database.Exec(`
		INSERT INTO epics (key, title, description, status, priority)
		VALUES (?, 'Test Epic', 'Test epic', 'active', 'high')
	`, epicKey)
	if err != nil {
		panic(fmt.Sprintf("Failed to insert epic with key %s: %v", epicKey, err))
	}

	epicID, err := result.LastInsertId()
	if err != nil {
		panic(fmt.Sprintf("Failed to get epic LastInsertId: %v", err))
	}

	if epicID == 0 {
		panic("Failed to get valid epic ID after insert")
	}

	// Create feature
	result, err = database.Exec(`
		INSERT INTO features (epic_id, key, title, slug, description, status)
		VALUES (?, ?, 'Test Feature', 'test-feature', 'Test feature', 'active')
	`, epicID, featureKey)
	if err != nil {
		panic(fmt.Sprintf("Failed to insert feature with key %s: %v", featureKey, err))
	}

	featureID, err := result.LastInsertId()
	if err != nil {
		panic(fmt.Sprintf("Failed to get feature LastInsertId: %v", err))
	}

	if featureID == 0 {
		panic("Failed to get valid feature ID after insert")
	}

	return epicID, featureID
}
