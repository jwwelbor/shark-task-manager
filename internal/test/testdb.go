package test

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/db"
)

var (
	testDB   *sql.DB
	dbOnce   sync.Once
	dbPath   = "test-shark-tasks.db"
)

// GetTestDB returns a shared test database
func GetTestDB() *sql.DB {
	dbOnce.Do(func() {
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
		VALUES ('E-TEST-01', 'Test Epic', 'Test epic', 'active', 'high')
	`)
	epicID, _ := result.LastInsertId()
	if epicID == 0 {
		database.QueryRow("SELECT id FROM epics WHERE key = 'E-TEST-01'").Scan(&epicID)
	}

	// Create feature
	result, _ = database.Exec(`
		INSERT OR IGNORE INTO features (epic_id, key, title, description, status)
		VALUES (?, 'F-TEST-01', 'Test Feature', 'Test feature', 'active')
	`, epicID)
	featureID, _ := result.LastInsertId()
	if featureID == 0 {
		database.QueryRow("SELECT id FROM features WHERE key = 'F-TEST-01'").Scan(&featureID)
	}

	// Create test tasks
	database.Exec(`
		INSERT OR IGNORE INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
		VALUES
			(?, 'T-TEST-001', 'Completed Task', 'completed', 'backend', 1, '[]'),
			(?, 'T-TEST-002', 'Todo Task', 'todo', 'backend', 2, '[]'),
			(?, 'T-TEST-003', 'Task with Dependency', 'todo', 'backend', 3, '["T-TEST-001"]'),
			(?, 'T-TEST-004', 'Task with Incomplete Dependency', 'todo', 'backend', 4, '["T-TEST-002"]')
	`, featureID, featureID, featureID, featureID)

	// Create E04 epic and feature for sync tests
	database.Exec(`INSERT OR IGNORE INTO epics (key, title, description, status, priority) VALUES ('E04', 'Task Management CLI Core', 'Core CLI functionality', 'active', 'high')`)
	var e04ID int64
	database.QueryRow("SELECT id FROM epics WHERE key = 'E04'").Scan(&e04ID)

	database.Exec(`INSERT OR IGNORE INTO features (epic_id, key, title, description, status) VALUES (?, 'E04-F05', 'Task File Management', 'Task CRUD operations', 'active')`, e04ID)

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
func GenerateUniqueKey(prefix string, i int) string {
	return fmt.Sprintf("%s-%03d", prefix, i)
}
