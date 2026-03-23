package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate-exec-order <database-path>")
		fmt.Println("Example: migrate-exec-order ./shark-tasks.db")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		slog.Error("Database does not exist", "path", dbPath)
		os.Exit(1)
	}

	// Open database
	database, err := db.InitDB(dbPath)
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Check if migration is needed
	var hasExecutionOrder bool
	err = database.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('features')
		WHERE name = 'execution_order'
	`).Scan(&hasExecutionOrder)
	if err != nil {
		slog.Error("Failed to check schema", "error", err)
		os.Exit(1)
	}

	if hasExecutionOrder {
		fmt.Println("Migration already applied: execution_order column exists")
		os.Exit(0)
	}

	// Run migration
	fmt.Println("Applying migration: adding execution_order column...")
	if err := db.MigrateAddExecutionOrder(database); err != nil {
		slog.Error("Migration failed", "error", err)
		os.Exit(1)
	}

	fmt.Println("Migration completed successfully!")
	fmt.Println("- Added execution_order column to features table")
	fmt.Println("- Added execution_order column to tasks table")
}
