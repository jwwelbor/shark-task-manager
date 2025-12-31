package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	database, err := sql.Open("sqlite3", "./shark-tasks.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	fmt.Println("Running slug backfill...")
	stats, err := db.BackfillSlugsFromFilePaths(database, false)
	if err != nil {
		log.Fatal("Backfill failed:", err)
	}

	fmt.Printf("Backfill completed successfully! Updated %d epics, %d features, %d tasks.\n",
		stats.EpicsUpdated, stats.FeaturesUpdated, stats.TasksUpdated)

	// Show results
	var epicsBefore, featuresCount, tasksCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM epics WHERE slug IS NOT NULL AND slug != ''").Scan(&epicsBefore); err != nil {
		log.Printf("Error counting epics: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM features WHERE slug IS NOT NULL AND slug != ''").Scan(&featuresCount); err != nil {
		log.Printf("Error counting features: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM tasks WHERE slug IS NOT NULL AND slug != ''").Scan(&tasksCount); err != nil {
		log.Printf("Error counting tasks: %v", err)
	}

	var totalEpics, totalFeatures, totalTasks int
	if err := database.QueryRow("SELECT COUNT(*) FROM epics").Scan(&totalEpics); err != nil {
		log.Printf("Error counting total epics: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM features").Scan(&totalFeatures); err != nil {
		log.Printf("Error counting total features: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&totalTasks); err != nil {
		log.Printf("Error counting total tasks: %v", err)
	}

	fmt.Printf("\nResults:\n")
	fmt.Printf("  Epics: %d/%d (%.1f%%)\n", epicsBefore, totalEpics, float64(epicsBefore)/float64(totalEpics)*100)
	fmt.Printf("  Features: %d/%d (%.1f%%)\n", featuresCount, totalFeatures, float64(featuresCount)/float64(totalFeatures)*100)
	fmt.Printf("  Tasks: %d/%d (%.1f%%)\n", tasksCount, totalTasks, float64(tasksCount)/float64(totalTasks)*100)

	// Show epics to verify no F## slugs
	fmt.Println("\nEpic slugs:")
	rows, err := database.Query("SELECT key, COALESCE(slug, '<null>') as slug FROM epics ORDER BY key")
	if err != nil {
		log.Printf("Error querying epic slugs: %v", err)
	} else {
		for rows.Next() {
			var key, slug string
			if err := rows.Scan(&key, &slug); err != nil {
				log.Printf("Error scanning epic row: %v", err)
				continue
			}
			fmt.Printf("  %s: %s\n", key, slug)
		}
		rows.Close()
	}
}
