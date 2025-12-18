package sync

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ManualResolver handles interactive conflict resolution
type ManualResolver struct {
	scanner *bufio.Scanner
}

// NewManualResolver creates a new ManualResolver
func NewManualResolver() *ManualResolver {
	return &ManualResolver{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// ResolveConflictsManually prompts the user for each conflict and applies their choice
func (m *ManualResolver) ResolveConflictsManually(
	conflicts []Conflict,
	fileData *TaskMetadata,
	dbTask *models.Task,
) (*models.Task, error) {
	// Create a copy of the database task
	resolver := NewConflictResolver()
	resolved := resolver.copyTask(dbTask)

	// If no conflicts, return copy
	if len(conflicts) == 0 {
		return resolved, nil
	}

	fmt.Println("\n=== Manual Conflict Resolution ===")
	fmt.Printf("Task: %s\n\n", dbTask.Key)

	// For each conflict, prompt user
	for i, conflict := range conflicts {
		fmt.Printf("Conflict %d/%d - Field: %s\n", i+1, len(conflicts), conflict.Field)
		fmt.Println("----------------------------------------")
		fmt.Printf("  Database value: %q\n", conflict.DatabaseValue)
		fmt.Printf("  File value:     %q\n", conflict.FileValue)
		fmt.Println()

		// Prompt for choice
		choice, err := m.promptForChoice()
		if err != nil {
			return nil, fmt.Errorf("failed to get user input: %w", err)
		}

		// Apply the choice
		if choice == "file" {
			m.applyFileValue(resolved, conflict.Field, fileData)
		}
		// If choice == "db", keep database value (already in resolved)

		fmt.Printf("  Resolution: Using %s value\n\n", choice)
	}

	fmt.Println("=== Manual Resolution Complete ===")

	return resolved, nil
}

// promptForChoice prompts the user to choose between file and database values
func (m *ManualResolver) promptForChoice() (string, error) {
	for {
		fmt.Print("Choose resolution (file/db): ")

		if !m.scanner.Scan() {
			if err := m.scanner.Err(); err != nil {
				return "", fmt.Errorf("scanner error: %w", err)
			}
			return "", fmt.Errorf("unexpected end of input")
		}

		choice := strings.TrimSpace(strings.ToLower(m.scanner.Text()))

		if choice == "file" || choice == "db" {
			return choice, nil
		}

		fmt.Println("Invalid choice. Please enter 'file' or 'db'.")
	}
}

// applyFileValue applies the file value to the resolved task for the given field
func (m *ManualResolver) applyFileValue(resolved *models.Task, field string, fileData *TaskMetadata) {
	switch field {
	case "title":
		resolved.Title = fileData.Title
	case "description":
		if fileData.Description != nil {
			desc := *fileData.Description
			resolved.Description = &desc
		}
	case "file_path":
		path := fileData.FilePath
		resolved.FilePath = &path
	}
}
