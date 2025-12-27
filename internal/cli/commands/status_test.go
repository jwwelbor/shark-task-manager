package commands

import (
	"testing"
)

// TestStatusCommand_BasicExecution tests basic execution of status command
func TestStatusCommand_BasicExecution(t *testing.T) {
	t.Skip("Test needs refactoring - status command creates new DB connection, bypassing test data")
	// TODO: Refactor status command to accept database connection for testability
	// The current implementation calls db.InitDB() which opens the production database,
	// ignoring the test database setup. This test will be re-enabled once the command
	// supports dependency injection of the database connection.
}
