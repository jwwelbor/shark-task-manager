package main

import (
	"fmt"
	"github.com/jwwelbor/shark-task-manager/internal/discovery"
)

func main() {
	parser := discovery.NewIndexParser()
	
	epics, features, err := parser.Parse("docs/plan/epic-index.md")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("✓ Successfully parsed epic-index.md\n\n")
	fmt.Printf("Found %d epics and %d features\n\n", len(epics), len(features))
	
	fmt.Println("EPICS:")
	fmt.Println("------")
	for _, epic := range epics {
		fmt.Printf("  %s: %s\n", epic.Key, epic.Title)
	}
	
	fmt.Println("\nFEATURES:")
	fmt.Println("---------")
	for _, feature := range features {
		fmt.Printf("  %s: %s (parent: %s)\n", feature.Key, feature.Title, feature.EpicKey)
	}
}
