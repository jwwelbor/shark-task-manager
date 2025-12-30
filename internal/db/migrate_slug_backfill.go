package db

import (
	"database/sql"
	"fmt"
)

// BackfillSlugsFromFilePaths extracts slugs from existing file_path values
// and populates the slug column for epics, features, and tasks
func BackfillSlugsFromFilePaths(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Backfill epics
	rows, err := tx.Query("SELECT id, file_path FROM epics WHERE file_path IS NOT NULL AND file_path != ''")
	if err != nil {
		return fmt.Errorf("failed to query epics: %w", err)
	}

	var epicUpdates []struct {
		id   int64
		slug string
	}

	for rows.Next() {
		var id int64
		var filePath string
		if err := rows.Scan(&id, &filePath); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan epic row: %w", err)
		}
		slug := extractEpicSlugFromPath(filePath)
		if slug != "" {
			epicUpdates = append(epicUpdates, struct {
				id   int64
				slug string
			}{id, slug})
		}
	}
	rows.Close()

	for _, update := range epicUpdates {
		_, err := tx.Exec("UPDATE epics SET slug = ? WHERE id = ?", update.slug, update.id)
		if err != nil {
			return fmt.Errorf("failed to update epic slug: %w", err)
		}
	}

	// Backfill features
	rows, err = tx.Query("SELECT id, file_path FROM features WHERE file_path IS NOT NULL AND file_path != ''")
	if err != nil {
		return fmt.Errorf("failed to query features: %w", err)
	}

	var featureUpdates []struct {
		id   int64
		slug string
	}

	for rows.Next() {
		var id int64
		var filePath string
		if err := rows.Scan(&id, &filePath); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan feature row: %w", err)
		}
		slug := extractFeatureSlugFromPath(filePath)
		if slug != "" {
			featureUpdates = append(featureUpdates, struct {
				id   int64
				slug string
			}{id, slug})
		}
	}
	rows.Close()

	for _, update := range featureUpdates {
		_, err := tx.Exec("UPDATE features SET slug = ? WHERE id = ?", update.slug, update.id)
		if err != nil {
			return fmt.Errorf("failed to update feature slug: %w", err)
		}
	}

	// Backfill tasks
	rows, err = tx.Query("SELECT id, file_path FROM tasks WHERE file_path IS NOT NULL AND file_path != ''")
	if err != nil {
		return fmt.Errorf("failed to query tasks: %w", err)
	}

	var taskUpdates []struct {
		id   int64
		slug string
	}

	for rows.Next() {
		var id int64
		var filePath string
		if err := rows.Scan(&id, &filePath); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan task row: %w", err)
		}
		slug := extractTaskSlugFromPath(filePath)
		if slug != "" {
			taskUpdates = append(taskUpdates, struct {
				id   int64
				slug string
			}{id, slug})
		}
	}
	rows.Close()

	for _, update := range taskUpdates {
		_, err := tx.Exec("UPDATE tasks SET slug = ? WHERE id = ?", update.slug, update.id)
		if err != nil {
			return fmt.Errorf("failed to update task slug: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// extractEpicSlugFromPath extracts the slug from an epic file path
// Example: "docs/plan/E05-task-mgmt-cli-capabilities/epic.md" -> "task-mgmt-cli-capabilities"
func extractEpicSlugFromPath(filePath string) string {
	if filePath == "" {
		return ""
	}

	// Find "E##-" pattern
	epicKeyStart := -1
	for i := 0; i < len(filePath)-2; i++ {
		if filePath[i] == 'E' && filePath[i+2] == '-' {
			// Check if next char is a digit
			if filePath[i+1] >= '0' && filePath[i+1] <= '9' {
				epicKeyStart = i
				break
			}
		}
		// Also check for E### pattern (3 digits)
		if i < len(filePath)-3 && filePath[i] == 'E' && filePath[i+3] == '-' {
			if filePath[i+1] >= '0' && filePath[i+1] <= '9' && filePath[i+2] >= '0' && filePath[i+2] <= '9' {
				epicKeyStart = i
				break
			}
		}
	}

	if epicKeyStart == -1 {
		return ""
	}

	// Find the first '-' after 'E##'
	slugStart := -1
	for i := epicKeyStart; i < len(filePath); i++ {
		if filePath[i] == '-' {
			slugStart = i + 1
			break
		}
	}

	if slugStart == -1 {
		return ""
	}

	// Find "/epic.md" to determine the end
	epicMdIndex := -1
	searchPattern := "/epic.md"
	for i := 0; i <= len(filePath)-len(searchPattern); i++ {
		if filePath[i:i+len(searchPattern)] == searchPattern {
			epicMdIndex = i
			break
		}
	}

	if epicMdIndex == -1 {
		return ""
	}

	if slugStart >= epicMdIndex {
		return ""
	}

	return filePath[slugStart:epicMdIndex]
}

// extractFeatureSlugFromPath extracts the slug from a feature file path
// Example: "docs/plan/E06-intelligent-scanning/E06-F04-incremental-sync-engine/prd.md" -> "incremental-sync-engine"
func extractFeatureSlugFromPath(filePath string) string {
	if filePath == "" {
		return ""
	}

	// Find the last occurrence of "F##-" pattern
	featureKeyStart := -1
	for i := len(filePath) - 1; i >= 2; i-- {
		if filePath[i] == '-' && i >= 2 {
			// Check if previous chars are "F##"
			if filePath[i-2] == 'F' && filePath[i-1] >= '0' && filePath[i-1] <= '9' {
				featureKeyStart = i + 1
				break
			}
			// Also check for F### pattern (3 digits)
			if i >= 3 && filePath[i-3] == 'F' && filePath[i-2] >= '0' && filePath[i-2] <= '9' && filePath[i-1] >= '0' && filePath[i-1] <= '9' {
				featureKeyStart = i + 1
				break
			}
		}
	}

	if featureKeyStart == -1 {
		return ""
	}

	// Find either "/prd.md" or "/feature.md" to determine the end
	endIndex := -1

	// Check for "/prd.md"
	prdPattern := "/prd.md"
	for i := 0; i <= len(filePath)-len(prdPattern); i++ {
		if filePath[i:i+len(prdPattern)] == prdPattern {
			endIndex = i
			break
		}
	}

	// If not found, check for "/feature.md"
	if endIndex == -1 {
		featurePattern := "/feature.md"
		for i := 0; i <= len(filePath)-len(featurePattern); i++ {
			if filePath[i:i+len(featurePattern)] == featurePattern {
				endIndex = i
				break
			}
		}
	}

	if endIndex == -1 || featureKeyStart >= endIndex {
		return ""
	}

	return filePath[featureKeyStart:endIndex]
}

// extractTaskSlugFromPath extracts the slug from a task file path
// Example: "docs/plan/E04-epic/E04-F01-feature/tasks/T-E04-F01-001-some-task-description.md" -> "some-task-description"
func extractTaskSlugFromPath(filePath string) string {
	if filePath == "" {
		return ""
	}

	// Find ".md" extension
	if len(filePath) < 3 || filePath[len(filePath)-3:] != ".md" {
		return ""
	}

	// Find the last '/' to get the filename
	lastSlash := -1
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' {
			lastSlash = i
			break
		}
	}

	filename := filePath
	if lastSlash != -1 {
		filename = filePath[lastSlash+1:]
	}

	// Remove ".md" extension
	filename = filename[:len(filename)-3]

	// Find the task key pattern "T-E##-F##-###"
	// Look for the fourth hyphen after the task key
	hyphenCount := 0
	slugStart := -1

	for i := 0; i < len(filename); i++ {
		if filename[i] == '-' {
			hyphenCount++
			if hyphenCount == 4 {
				slugStart = i + 1
				break
			}
		}
	}

	// If we found the fourth hyphen, extract the slug
	if slugStart != -1 && slugStart < len(filename) {
		return filename[slugStart:]
	}

	// No slug found (task key only, like "T-E04-F01-001.md")
	return ""
}
