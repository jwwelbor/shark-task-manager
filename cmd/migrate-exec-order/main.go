package main

import (
	"fmt"
	"log"
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
		log.Fatalf("Database does not exist: %s", dbPath)
	}

	// Open database
	database, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
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
		log.Fatalf("Failed to check schema: %v", err)
	}

	if hasExecutionOrder {
		fmt.Println("Migration already applied: execution_order column exists")
		os.Exit(0)
	}

	// Run migration
	fmt.Println("Applying migration: adding execution_order column...")
	if err := db.MigrateAddExecutionOrder(database); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migration completed successfully!")
	fmt.Println("- Added execution_order column to features table")
	fmt.Println("- Added execution_order column to tasks table")
}
