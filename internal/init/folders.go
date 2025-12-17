package init

import (
	"fmt"
	"os"
	"path/filepath"
)

// createFolders creates required folder structure
// Returns list of folders created (empty if all existed)
func (i *Initializer) createFolders() ([]string, error) {
	folders := []string{
		"docs/plan",
		"templates",
	}

	created := []string{} // Initialize to empty slice, not nil

	for _, folder := range folders {
		// Check if folder exists
		if _, err := os.Stat(folder); err == nil {
			// Folder exists, skip
			continue
		}

		// Create folder
		if err := os.MkdirAll(folder, 0755); err != nil {
			return created, fmt.Errorf("failed to create folder %s: %w", folder, err)
		}

		absPath, _ := filepath.Abs(folder)
		created = append(created, absPath)
	}

	return created, nil
}
