package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: cleanup <db-path> <epic-key>")
		fmt.Println("Example: cleanup shark-tasks.db E06")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	epicKey := os.Args[2]

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Get epic info before deletion
	var epicID int
	var epicTitle string
	err = tx.QueryRow("SELECT id, title FROM epics WHERE key = ?", epicKey).Scan(&epicID, &epicTitle)
	if err == sql.ErrNoRows {
		slog.Error("Epic not found", "key", epicKey)
		os.Exit(1)
	} else if err != nil {
		slog.Error("Failed to query epic", "error", err)
		os.Exit(1)
	}

	// Count features
	var featureCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM features WHERE epic_id = ?", epicID).Scan(&featureCount)
	if err != nil {
		slog.Error("Failed to count features", "error", err)
		os.Exit(1)
	}

	// Count tasks
	var taskCount int
	err = tx.QueryRow(`
		SELECT COUNT(*) FROM tasks
		WHERE feature_id IN (SELECT id FROM features WHERE epic_id = ?)
	`, epicID).Scan(&taskCount)
	if err != nil {
		slog.Error("Failed to count tasks", "error", err)
		os.Exit(1)
	}

	// Show what will be deleted
	fmt.Printf("\n🗑️  Cleanup Summary for Epic: %s\n", epicKey)
	fmt.Printf("─────────────────────────────────────────\n")
	fmt.Printf("Epic:     %s - %s\n", epicKey, epicTitle)
	fmt.Printf("Features: %d\n", featureCount)
	fmt.Printf("Tasks:    %d\n", taskCount)
	fmt.Printf("─────────────────────────────────────────\n\n")

	// Delete the epic (cascade will handle features and tasks)
	result, err := tx.Exec("DELETE FROM epics WHERE key = ?", epicKey)
	if err != nil {
		slog.Error("Failed to delete epic", "error", err)
		os.Exit(1)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		slog.Error("No epic found with key", "key", epicKey)
		os.Exit(1)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit transaction", "error", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully deleted epic %s and all associated data\n", epicKey)
}
