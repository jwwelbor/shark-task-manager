package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Open the production database
	database, err := sql.Open("sqlite3", "shark-tasks.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	fmt.Println("Starting slug backfill migration...")

	// Run the backfill
	updated, err := db.BackfillSlugsFromFilePaths(database, true)
	if err != nil {
		log.Fatalf("Backfill failed: %v", err)
	}
	fmt.Printf("Updated %d tasks with slugs\n", updated)

	fmt.Println("✅ Backfill completed successfully!")

	// Generate verification report
	fmt.Println("\n=== Verification Report ===")

	// Count epics with slugs
	var epicCount int
	err = database.QueryRow("SELECT COUNT(*) FROM epics WHERE slug IS NOT NULL").Scan(&epicCount)
	if err != nil {
		log.Fatalf("Failed to count epic slugs: %v", err)
	}
	fmt.Printf("Epics with slugs: %d\n", epicCount)

	// Count features with slugs
	var featureCount int
	err = database.QueryRow("SELECT COUNT(*) FROM features WHERE slug IS NOT NULL").Scan(&featureCount)
	if err != nil {
		log.Fatalf("Failed to count feature slugs: %v", err)
	}
	fmt.Printf("Features with slugs: %d\n", featureCount)

	// Count tasks with slugs
	var taskCount int
	err = database.QueryRow("SELECT COUNT(*) FROM tasks WHERE slug IS NOT NULL").Scan(&taskCount)
	if err != nil {
		log.Fatalf("Failed to count task slugs: %v", err)
	}
	fmt.Printf("Tasks with slugs: %d\n", taskCount)

	// Sample slugs
	fmt.Println("\n=== Sample Slugs ===")

	// Sample epic slugs
	rows, err := database.Query("SELECT key, slug FROM epics WHERE slug IS NOT NULL LIMIT 3")
	if err == nil {
		fmt.Println("\nEpics:")
		for rows.Next() {
			var key, slug string
			if err := rows.Scan(&key, &slug); err != nil {
				log.Printf("Error scanning epic row: %v", err)
				continue
			}
			fmt.Printf("  %s -> %s\n", key, slug)
		}
		rows.Close()
	}

	// Sample feature slugs
	rows, err = database.Query("SELECT key, slug FROM features WHERE slug IS NOT NULL LIMIT 3")
	if err == nil {
		fmt.Println("\nFeatures:")
		for rows.Next() {
			var key, slug string
			if err := rows.Scan(&key, &slug); err != nil {
				log.Printf("Error scanning feature row: %v", err)
				continue
			}
			fmt.Printf("  %s -> %s\n", key, slug)
		}
		rows.Close()
	}

	// Sample task slugs
	rows, err = database.Query("SELECT key, slug FROM tasks WHERE slug IS NOT NULL LIMIT 5")
	if err == nil {
		fmt.Println("\nTasks:")
		for rows.Next() {
			var key, slug string
			if err := rows.Scan(&key, &slug); err != nil {
				log.Printf("Error scanning task row: %v", err)
				continue
			}
			fmt.Printf("  %s -> %s\n", key, slug)
		}
		rows.Close()
	}

	fmt.Println("\n=== Migration Complete ===")
}
