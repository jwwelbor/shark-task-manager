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
	result, _ := database.Exec(`
		INSERT OR IGNORE INTO epics (key, title, description, status, priority)
		VALUES ('E99', 'Test Epic', 'Test epic', 'active', 'high')
	`)
	epicID, _ := result.LastInsertId()
	if epicID == 0 {
		_ = database.QueryRow("SELECT id FROM epics WHERE key = 'E99'").Scan(&epicID)
	}

	// Create feature
	result, _ = database.Exec(`
		INSERT OR IGNORE INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E99-F99', 'Test Feature', 'Test feature', 'active')
	`, epicID)
	featureID, _ := result.LastInsertId()
	if featureID == 0 {
		_ = database.QueryRow("SELECT id FROM features WHERE key = 'E99-F99'").Scan(&featureID)
	}

	// Create test tasks
	_, _ = database.Exec(`
		INSERT OR IGNORE INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
		VALUES
			(?, 'T-E99-F99-001', 'Completed Task', 'completed', 'backend', 1, '[]'),
			(?, 'T-E99-F99-002', 'Todo Task', 'todo', 'backend', 2, '[]'),
			(?, 'T-E99-F99-003', 'Task with Dependency', 'todo', 'backend', 3, '["T-E99-F99-001"]'),
			(?, 'T-E99-F99-004', 'Task with Incomplete Dependency', 'todo', 'backend', 4, '["T-E99-F99-002"]')
	`, featureID, featureID, featureID, featureID)

	// Create E04 epic and feature for sync tests
	_, _ = database.Exec(`INSERT OR IGNORE INTO epics (key, title, description, status, priority) VALUES ('E04', 'Task Management CLI Core', 'Core CLI functionality', 'active', 'high')`)
	var e04ID int64
	_ = database.QueryRow("SELECT id FROM epics WHERE key = 'E04'").Scan(&e04ID)

	_, _ = database.Exec(`INSERT OR IGNORE INTO features (epic_id, key, title, description, status) VALUES (?, 'E04-F05', 'Task File Management', 'Task CRUD operations', 'active')`, e04ID)

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
