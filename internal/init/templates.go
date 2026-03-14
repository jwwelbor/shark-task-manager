package init

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	stm "github.com/jwwelbor/shark-task-manager"
)

// copyTemplates copies embedded templates to the configured template directory.
// All files under shark-templates/ (entity templates, orchestrator .tmpl files,
// partials, etc.) are copied preserving their directory structure.
// Returns count of templates copied.
func (i *Initializer) copyTemplates(force bool, templateDir string) (int, error) {
	if templateDir == "" {
		templateDir = "shark-templates"
	}
	count := 0

	// Walk all embedded templates under shark-templates/
	err := fs.WalkDir(stm.EmbeddedSharkTemplates, "shark-templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Read embedded file
		data, err := stm.EmbeddedSharkTemplates.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded template %s: %w", path, err)
		}

		// Compute target path: strip "shark-templates/" prefix, prepend templateDir
		relPath, _ := filepath.Rel("shark-templates", path)
		targetPath := filepath.Join(templateDir, relPath)

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
