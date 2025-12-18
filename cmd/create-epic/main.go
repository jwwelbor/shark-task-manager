package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: create-epic <db-path> <key> <title> <description>")
		fmt.Println("Example: create-epic shark-tasks.db E06 'Intelligent Scanning' 'Pattern-based discovery system'")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	key := os.Args[2]
	title := os.Args[3]
	description := os.Args[4]

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert epic
	result, err := db.Exec(`
		INSERT INTO epics (key, title, description, status, priority)
		VALUES (?, ?, ?, 'draft', 'medium')
	`, key, title, description)
	if err != nil {
		log.Fatalf("Failed to insert epic: %v", err)
	}

	id, _ := result.LastInsertId()
	fmt.Printf("✅ Successfully created epic %s (ID: %d)\n", key, id)
	fmt.Printf("   Title: %s\n", title)
	fmt.Printf("   Description: %s\n", description)
}
