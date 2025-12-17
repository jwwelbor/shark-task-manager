package init

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed templates/*
var embeddedTemplates embed.FS

// copyTemplates copies embedded templates to templates/ folder
// Returns count of templates copied
func (i *Initializer) copyTemplates(force bool) (int, error) {
	targetDir := "templates"
	count := 0

	// Walk embedded templates
	err := fs.WalkDir(embeddedTemplates, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Read embedded file
		data, err := embeddedTemplates.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded template %s: %w", path, err)
		}

		// Compute target path
		relPath, _ := filepath.Rel("templates", path)
		targetPath := filepath.Join(targetDir, relPath)

		// Check if target exists
		if _, err := os.Stat(targetPath); err == nil && !force {
			// Skip existing template
			return nil
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
		}

		// Write file
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write template %s: %w", targetPath, err)
		}

		count++
		return nil
	})

	return count, err
}
