// Package task_test verifies that internal/test/testdb.go resolves the correct
// path to the shared test database when tests run from the task sub-package directory.
package task_test

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestGetTestDB_PathResolves verifies that test.GetTestDB() resolves the correct
// path from the task sub-package location (internal/repository/task/).
// The testdb.go path heuristic uses os.Stat("../../internal/repository") which
// resolves correctly from this sub-package location.
func TestGetTestDB_PathResolves(t *testing.T) {
	db := test.GetTestDB()
	if db == nil {
		t.Fatal("GetTestDB returned nil — path resolution failed from sub-package location")
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("test DB ping failed: %v", err)
	}
}
