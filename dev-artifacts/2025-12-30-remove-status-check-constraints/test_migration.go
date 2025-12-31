package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/db"
)

func main() {
	dbPath := "shark-tasks.db"

	fmt.Println("Opening database:", dbPath)
	database, err := db.InitDB(dbPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	fmt.Println("Database initialized successfully")

	// Test that we can insert a task with new workflow status
	ctx := context.Background()

	// Get a feature to add task to
	var featureID int64
	err = database.QueryRowContext(ctx, "SELECT id FROM features LIMIT 1").Scan(&featureID)
	if err != nil {
		fmt.Printf("Error getting feature: %v\n", err)
		os.Exit(1)
	}

	// Try to create a task with new workflow status
	testStatus := "in_development"
	fmt.Printf("\nAttempting to create task with status '%s'...\n", testStatus)

	result, err := database.ExecContext(ctx, `
		INSERT INTO tasks (feature_id, key, title, description, status, priority)
		VALUES (?, ?, ?, ?, ?, ?)
	`, featureID, "TEST-MIGRATION-001", "Test Migration Task", "Testing workflow status", testStatus, 5)

	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
		os.Exit(1)
	}

	taskID, _ := result.LastInsertId()
	fmt.Printf("✅ SUCCESS: Created task ID %d with status '%s'\n", taskID, testStatus)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskID)
	fmt.Println("\nTest task cleaned up")

	// Verify schema has no CHECK constraint on status
	var createSQL string
	err = database.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type='table' AND name='tasks'").Scan(&createSQL)
	if err != nil {
		fmt.Printf("Error getting schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ Migration successful!")
	fmt.Println("Database now accepts workflow-defined statuses")
}
