package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/db"
)

func main() {
	dbPath := flag.String("db", "shark-tasks.db", "Path to database file")
	flag.Parse()

	fmt.Printf("Running migration on database: %s\n", *dbPath)

	// Check if database exists
	if _, err := os.Stat(*dbPath); os.IsNotExist(err) {
		slog.Error("Database file does not exist", "path", *dbPath)
		os.Exit(1)
	}

	// Open database
	database, err := db.InitDB(*dbPath)
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Run migration
	fmt.Println("Removing agent_type CHECK constraint...")
	if err := db.MigrateRemoveAgentTypeConstraint(database); err != nil {
		slog.Error("Migration failed", "error", err)
		os.Exit(1)
	}

	fmt.Println("Migration completed successfully!")
	fmt.Println("agent_type column now accepts any string value.")
}
