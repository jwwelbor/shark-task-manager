package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	_ "modernc.org/sqlite"
)

func main() {
	// Open the production database
	database, err := sql.Open("sqlite", "shark-tasks.db")
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	fmt.Println("Starting slug backfill migration...")

	// Run the backfill
	updated, err := db.BackfillSlugsFromFilePaths(database, true)
	if err != nil {
		slog.Error("Backfill failed", "error", err)
		os.Exit(1)
	}
	fmt.Printf("Updated %d tasks with slugs\n", updated)

	fmt.Println("✅ Backfill completed successfully!")

	// Generate verification report
	fmt.Println("\n=== Verification Report ===")

	// Count epics with slugs
	var epicCount int
	err = database.QueryRow("SELECT COUNT(*) FROM epics WHERE slug IS NOT NULL").Scan(&epicCount)
	if err != nil {
		slog.Error("Failed to count epic slugs", "error", err)
		os.Exit(1)
	}
	fmt.Printf("Epics with slugs: %d\n", epicCount)

	// Count features with slugs
	var featureCount int
	err = database.QueryRow("SELECT COUNT(*) FROM features WHERE slug IS NOT NULL").Scan(&featureCount)
	if err != nil {
		slog.Error("Failed to count feature slugs", "error", err)
		os.Exit(1)
	}
	fmt.Printf("Features with slugs: %d\n", featureCount)

	// Count tasks with slugs
	var taskCount int
	err = database.QueryRow("SELECT COUNT(*) FROM tasks WHERE slug IS NOT NULL").Scan(&taskCount)
	if err != nil {
		slog.Error("Failed to count task slugs", "error", err)
		os.Exit(1)
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
				slog.Warn("scanning epic row", "error", err)
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
				slog.Warn("scanning feature row", "error", err)
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
				slog.Warn("scanning task row", "error", err)
				continue
			}
			fmt.Printf("  %s -> %s\n", key, slug)
		}
		rows.Close()
	}

	fmt.Println("\n=== Migration Complete ===")
}
