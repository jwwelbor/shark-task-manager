package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

func main() {
	ctx := context.Background()

	// Initialize database
	database, err := db.InitDB("shark-tasks.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init DB: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Create repository
	repoDb := repository.NewDB(database)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Test different key formats
	testKeys := []string{
		"E07-F11",                               // Full key
		"F11",                                   // Numeric key
		"f11",                                   // Lowercase numeric key
		"F11-slug-architecture-improvement",     // Slugged key
		"E07-F11-slug-architecture-improvement", // Full key with slug
	}

	for _, key := range testKeys {
		fmt.Printf("Testing key: %s... ", key)
		feature, err := featureRepo.GetByKey(ctx, key)
		if err != nil {
			fmt.Printf("FAILED: %v\n", err)
		} else {
			fmt.Printf("SUCCESS: Found feature %s (%s)\n", feature.Key, feature.Title)
		}
	}
}
