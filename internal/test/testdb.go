package test

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/db"
)

var (
	testDB *sql.DB
	dbOnce sync.Once
	dbPath string
)

// init determines the test database path
func init() {
	// Try to find the project root by looking for internal/repository directory
	// If it doesn't exist, create a temp directory
	if _, err := os.Stat("internal/repository"); err == nil {
		dbPath = "internal/repository/test-shark-tasks.db"
	} else if _, err := os.Stat("../../internal/repository"); err == nil {
		dbPath = "../../internal/repository/test-shark-tasks.db"
	} else {
		// Fallback to temp directory
		dbPath = filepath.Join(os.TempDir(), "shark-test-tasks.db")
	}
}

// GetTestDB returns a shared test database
func GetTestDB() *sql.DB {
	dbOnce.Do(func() {
		// Ensure directory exists
		dir := filepath.Dir(dbPath)
		_ = os.MkdirAll(dir, 0755)

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

	// Create epic via SQL to avoid import cycle
	result, err := database.Exec(`
		INSERT OR IGNORE INTO epics (key, title, description, status, priority)
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
		err = database.QueryRow("SELECT id FROM epics WHERE key = 'E99'").Scan(&epicID)
		if err != nil {
			panic(fmt.Sprintf("Failed to find epic E99: %v", err))
		}
	}

	// Create feature
	result, err = database.Exec(`
		INSERT OR IGNORE INTO features (epic_id, key, title, slug, description, status)
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
		err = database.QueryRow("SELECT id FROM features WHERE key = 'E99-F99'").Scan(&featureID)
		if err != nil {
			panic(fmt.Sprintf("Failed to find feature E99-F99: %v", err))
		}
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
		if err.Error() != "FOREIGN KEY constraint failed" {
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
		if err.Error() != "FOREIGN KEY constraint failed" {
			panic(fmt.Sprintf("Failed to insert E04-F05 feature: %v", err))
		}
	}

	return epicID, featureID
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
