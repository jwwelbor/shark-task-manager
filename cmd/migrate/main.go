package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/db"
)

func main() {
	dbPath := flag.String("db", "shark-tasks.db", "Path to database file")
	flag.Parse()

	fmt.Printf("Running migration on database: %s\n", *dbPath)

	// Check if database exists
	if _, err := os.Stat(*dbPath); os.IsNotExist(err) {
		log.Fatalf("Database file does not exist: %s", *dbPath)
	}

	// Open database
	database, err := db.InitDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Run migration
	fmt.Println("Removing agent_type CHECK constraint...")
	if err := db.MigrateRemoveAgentTypeConstraint(database); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migration completed successfully!")
	fmt.Println("agent_type column now accepts any string value.")
}
