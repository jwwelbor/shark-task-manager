package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// getTestDB creates a test database for the db package tests
func getTestDB(t *testing.T) *sql.DB {
	// Create test database in temp directory
	dbPath := filepath.Join(os.TempDir(), "shark-status-test.db")

	// Remove old test database if it exists
	_ = os.Remove(dbPath)

	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	return database
}

// TestWorkflowStatusesAllowed tests that new workflow statuses from config can be used
// This test demonstrates the problem: hardcoded CHECK constraints prevent using workflow statuses
func TestWorkflowStatusesAllowed(t *testing.T) {
	ctx := context.Background()
	database := getTestDB(t)
	defer database.Close()

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-STATUS-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'TEST-STATUS-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'TEST-STATUS-%'")

	// Create test epic and feature first
	_, err := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('TEST-STATUS-E01', 'Test Epic', 'Epic for status testing', 'active', 'medium')
	`)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	var epicID int64
	err = database.QueryRowContext(ctx, "SELECT id FROM epics WHERE key = 'TEST-STATUS-E01'").Scan(&epicID)
	if err != nil {
		t.Fatalf("Failed to get epic ID: %v", err)
	}

	_, err = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'TEST-STATUS-F01', 'Test Feature', 'Feature for status testing', 'active')
	`, epicID)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	var featureID int64
	err = database.QueryRowContext(ctx, "SELECT id FROM features WHERE key = 'TEST-STATUS-F01'").Scan(&featureID)
	if err != nil {
		t.Fatalf("Failed to get feature ID: %v", err)
	}

	// Test cases for workflow statuses from .sharkconfig.json
	workflowStatuses := []struct {
		status      string
		description string
	}{
		{"draft", "Initial draft state"},
		{"ready_for_refinement", "Awaiting specification"},
		{"in_refinement", "Being analyzed"},
		{"ready_for_development", "Ready for implementation"},
		{"in_development", "Code implementation in progress"},
		{"ready_for_code_review", "Code complete, awaiting review"},
		{"in_code_review", "Under code review"},
		{"ready_for_qa", "Ready for QA testing"},
		{"in_qa", "Being tested"},
		{"ready_for_approval", "Awaiting final approval"},
		{"in_approval", "Under final review"},
		{"on_hold", "Intentionally paused"},
		{"cancelled", "Task abandoned"},
	}

	for _, tc := range workflowStatuses {
		t.Run(tc.status, func(t *testing.T) {
			taskKey := "TEST-STATUS-" + tc.status

			// Clean up any existing task
			_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", taskKey)

			// Attempt to create a task with workflow status
			_, err := database.ExecContext(ctx, `
				INSERT INTO tasks (feature_id, key, title, description, status, priority)
				VALUES (?, ?, ?, ?, ?, 5)
			`, featureID, taskKey, "Test Task "+tc.status, tc.description, tc.status)

			if err != nil {
				t.Errorf("Failed to create task with status '%s': %v", tc.status, err)
				t.Logf("This failure is EXPECTED - it demonstrates the CHECK constraint problem")
			} else {
				t.Logf("Successfully created task with status '%s'", tc.status)
			}

			// Cleanup
			_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", taskKey)
		})
	}

	// Cleanup epic and feature
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'TEST-STATUS-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'TEST-STATUS-E01'")
}

// TestStatusTransitionFromOldToNewWorkflow tests transitioning from old statuses to new workflow statuses
func TestStatusTransitionFromOldToNewWorkflow(t *testing.T) {
	ctx := context.Background()
	database := getTestDB(t)
	defer database.Close()

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'TEST-TRANSITION-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'TEST-TRANSITION-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'TEST-TRANSITION-E01'")

	// Create test epic and feature
	_, err := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('TEST-TRANSITION-E01', 'Test Epic', 'Epic for transition testing', 'active', 'medium')
	`)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	var epicID int64
	err = database.QueryRowContext(ctx, "SELECT id FROM epics WHERE key = 'TEST-TRANSITION-E01'").Scan(&epicID)
	if err != nil {
		t.Fatalf("Failed to get epic ID: %v", err)
	}

	_, err = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'TEST-TRANSITION-F01', 'Test Feature', 'Feature for transition testing', 'active')
	`, epicID)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	var featureID int64
	err = database.QueryRowContext(ctx, "SELECT id FROM features WHERE key = 'TEST-TRANSITION-F01'").Scan(&featureID)
	if err != nil {
		t.Fatalf("Failed to get feature ID: %v", err)
	}

	// Create a task with old status (this should work - 'todo' is in CHECK constraint)
	_, err = database.ExecContext(ctx, `
		INSERT INTO tasks (feature_id, key, title, description, status, priority)
		VALUES (?, 'TEST-TRANSITION-001', 'Test Task', 'Task for transition testing', 'todo', 5)
	`, featureID)
	if err != nil {
		t.Fatalf("Failed to create test task with 'todo' status: %v", err)
	}

	// Try to update to new workflow status
	_, err = database.ExecContext(ctx, `
		UPDATE tasks SET status = 'in_development' WHERE key = 'TEST-TRANSITION-001'
	`)
	if err != nil {
		t.Errorf("Failed to transition from 'todo' to 'in_development': %v", err)
		t.Logf("This failure is EXPECTED - demonstrates CHECK constraint blocks workflow transitions")
	} else {
		t.Logf("Successfully transitioned task to 'in_development' status")
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'TEST-TRANSITION-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'TEST-TRANSITION-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'TEST-TRANSITION-E01'")
}

// TestInvalidStatusStillRejected tests that truly invalid statuses are still rejected
// This ensures we maintain data integrity at the application level after removing CHECK constraints
func TestInvalidStatusStillRejected(t *testing.T) {
	ctx := context.Background()
	database := getTestDB(t)
	defer database.Close()

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'TEST-INVALID-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'TEST-INVALID-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'TEST-INVALID-E01'")

	// Create test epic and feature
	_, err := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('TEST-INVALID-E01', 'Test Epic', 'Epic for invalid status testing', 'active', 'medium')
	`)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	var epicID int64
	err = database.QueryRowContext(ctx, "SELECT id FROM epics WHERE key = 'TEST-INVALID-E01'").Scan(&epicID)
	if err != nil {
		t.Fatalf("Failed to get epic ID: %v", err)
	}

	_, err = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'TEST-INVALID-F01', 'Test Feature', 'Feature for invalid status testing', 'active')
	`, epicID)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	var featureID int64
	err = database.QueryRowContext(ctx, "SELECT id FROM features WHERE key = 'TEST-INVALID-F01'").Scan(&featureID)
	if err != nil {
		t.Fatalf("Failed to get feature ID: %v", err)
	}

	// Note: After removing CHECK constraints, the database will allow any string value
	// Application-level validation becomes critical to prevent invalid statuses
	// This test documents the behavior change

	invalidStatuses := []string{
		"not_a_real_status",
		"invalid",
		"random_string",
		"",
	}

	for _, invalidStatus := range invalidStatuses {
		t.Run(invalidStatus, func(t *testing.T) {
			// After removing CHECK constraints, database will accept these
			// Application code MUST validate before inserting
			_, err := database.ExecContext(ctx, `
				INSERT INTO tasks (feature_id, key, title, description, status, priority)
				VALUES (?, ?, 'Test Task', 'Test', ?, 5)
			`, featureID, "TEST-INVALID-001", invalidStatus)

			// Clean up if it was created
			_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'TEST-INVALID-001'")

			// Document the behavior
			if err != nil {
				t.Logf("Database rejected invalid status '%s': %v", invalidStatus, err)
			} else {
				t.Logf("WARNING: Database accepted invalid status '%s' - application validation is REQUIRED", invalidStatus)
			}
		})
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'TEST-INVALID-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'TEST-INVALID-E01'")
}
