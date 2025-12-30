package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/jwwelbor/shark-task-manager/internal/db"
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
	database.QueryRow("SELECT COUNT(*) FROM epics WHERE slug IS NOT NULL AND slug != ''").Scan(&epicsBefore)
	database.QueryRow("SELECT COUNT(*) FROM features WHERE slug IS NOT NULL AND slug != ''").Scan(&featuresCount)
	database.QueryRow("SELECT COUNT(*) FROM tasks WHERE slug IS NOT NULL AND slug != ''").Scan(&tasksCount)

	var totalEpics, totalFeatures, totalTasks int
	database.QueryRow("SELECT COUNT(*) FROM epics").Scan(&totalEpics)
	database.QueryRow("SELECT COUNT(*) FROM features").Scan(&totalFeatures)
	database.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&totalTasks)

	fmt.Printf("\nResults:\n")
	fmt.Printf("  Epics: %d/%d (%.1f%%)\n", epicsBefore, totalEpics, float64(epicsBefore)/float64(totalEpics)*100)
	fmt.Printf("  Features: %d/%d (%.1f%%)\n", featuresCount, totalFeatures, float64(featuresCount)/float64(totalFeatures)*100)
	fmt.Printf("  Tasks: %d/%d (%.1f%%)\n", tasksCount, totalTasks, float64(tasksCount)/float64(totalTasks)*100)

	// Show epics to verify no F## slugs
	fmt.Println("\nEpic slugs:")
	rows, _ := database.Query("SELECT key, COALESCE(slug, '<null>') as slug FROM epics ORDER BY key")
	for rows.Next() {
		var key, slug string
		rows.Scan(&key, &slug)
		fmt.Printf("  %s: %s\n", key, slug)
	}
	rows.Close()
}
