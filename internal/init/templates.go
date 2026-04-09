package init

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	stm "github.com/jwwelbor/shark-task-manager"
)

// templateCopyResult summarizes a templates sync.
type templateCopyResult struct {
	// Copied is files that did not exist on disk and were written fresh.
	Copied int
	// Refreshed is files that existed but differed from the embedded version
	// and were overwritten because force=true.
	Refreshed int
	// Differed is files that exist on disk and differ from the embedded version
	// but were left untouched because force=false. Callers should surface these
	// to the user as a warning so they know an upgrade is available.
	Differed []string
}

// copyTemplates syncs the embedded shark-templates/ tree into templateDir.
//
// Behavior per file:
//   - If the file does not exist on disk → copy it.
//   - If the file exists and matches the embedded bytes → silently skip.
//   - If the file exists and differs from the embedded bytes:
//   - force=true  → overwrite (counts as Refreshed).
//   - force=false → leave the user's copy alone, record it in Differed
//     so the caller can warn that `--force` is needed to upgrade.
//
// This protects user customization (no clobbering by default) while still
// surfacing template updates from a newer shark binary.
func (i *Initializer) copyTemplates(force bool, templateDir string) (templateCopyResult, error) {
	if templateDir == "" {
		templateDir = "shark-templates"
	}
	result := templateCopyResult{}

	walkErr := fs.WalkDir(stm.EmbeddedSharkTemplates, "shark-templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		embeddedData, err := stm.EmbeddedSharkTemplates.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded template %s: %w", path, err)
		}

		relPath, _ := filepath.Rel("shark-templates", path)
		targetPath := filepath.Join(templateDir, relPath)

		existingData, statErr := os.ReadFile(targetPath)
		switch {
		case os.IsNotExist(statErr):
			// New file — write it.
			if err := writeTemplateFile(targetPath, embeddedData); err != nil {
				return err
			}
			result.Copied++
			return nil
		case statErr != nil:
			return fmt.Errorf("failed to read existing template %s: %w", targetPath, statErr)
		}

		if bytes.Equal(existingData, embeddedData) {
			// Up to date — nothing to do.
			return nil
		}

		// User's copy differs from shipped version.
		if !force {
			result.Differed = append(result.Differed, targetPath)
			return nil
		}

		if err := writeTemplateFile(targetPath, embeddedData); err != nil {
			return err
		}
		result.Refreshed++
		return nil
	})

	return result, walkErr
}

// writeTemplateFile creates the parent directory and writes data to path.
func writeTemplateFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write template %s: %w", path, err)
	}
	return nil
}
