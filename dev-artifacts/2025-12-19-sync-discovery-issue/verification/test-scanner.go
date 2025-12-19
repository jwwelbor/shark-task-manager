package main

import (
	"fmt"
	"regexp"
)

func main() {
	// Test the updated pattern
	pattern := regexp.MustCompile(`^T-(E\d{2})-(F\d{2})-\d{3}.*\.md$`)

	testFiles := []string{
		"T-E07-F08-001.md",
		"T-E07-F08-001-database-schema-migration.md",
		"T-E07-F08-002-repository-methods.md",
		"T-E07-F08-003-validation-reuse.md",
		"T-E07-F08-004-epic-cli-flags.md",
		"T-E07-F08-005-feature-cli-flags.md",
		"T-E07-F08-006-documentation-updates.md",
		"T-E07-F09-001.md",
		"T-E07-F09-002.md",
		"prd.md",
		"not-a-task.md",
	}

	fmt.Println("Testing updated scanner pattern:")
	matchCount := 0
	for _, file := range testFiles {
		matches := pattern.MatchString(file)
		status := "✗"
		if matches {
			status = "✓"
			matchCount++
		}
		fmt.Printf("  %s %s\n", status, file)
	}
	fmt.Printf("\nMatched: %d/%d\n", matchCount, len(testFiles))
}
