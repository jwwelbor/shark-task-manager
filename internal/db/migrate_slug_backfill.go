package db

import (
	"database/sql"
	"fmt"
)

// MigrationStats contains statistics about slug backfill migration
type MigrationStats struct {
	EpicsTotal      int
	EpicsWithSlugs  int
	EpicsUpdated    int
	FeaturesTotal   int
	FeaturesWithSlugs int
	FeaturesUpdated int
	TasksTotal      int
	TasksWithSlugs  int
	TasksUpdated    int
}

// BackfillSlugsFromFilePaths extracts slugs from existing file_path values
// and populates the slug column for epics, features, and tasks.
// Uses a three-phase approach for maximum coverage:
//   Phase 1: Extract epic/feature slugs from task paths (highest coverage - 278 tasks)
//   Phase 2: Extract epic slugs from feature paths (fill gaps - 11 features)
//   Phase 3: Extract slugs from own file_path (current behavior)
//
// If dryRun is true, changes are not committed and statistics are returned showing
// what WOULD be updated. This allows preview before running actual migration.
func BackfillSlugsFromFilePaths(db *sql.DB, dryRun bool) (*MigrationStats, error) {
	stats := &MigrationStats{}

	// Get total counts before migration
	if err := db.QueryRow("SELECT COUNT(*) FROM epics").Scan(&stats.EpicsTotal); err != nil {
		return nil, fmt.Errorf("failed to count epics: %w", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM features").Scan(&stats.FeaturesTotal); err != nil {
		return nil, fmt.Errorf("failed to count features: %w", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&stats.TasksTotal); err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Get counts with existing slugs
	if err := db.QueryRow("SELECT COUNT(*) FROM epics WHERE slug IS NOT NULL AND slug != ''").Scan(&stats.EpicsWithSlugs); err != nil {
		return nil, fmt.Errorf("failed to count epics with slugs: %w", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM features WHERE slug IS NOT NULL AND slug != ''").Scan(&stats.FeaturesWithSlugs); err != nil {
		return nil, fmt.Errorf("failed to count features with slugs: %w", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE slug IS NOT NULL AND slug != ''").Scan(&stats.TasksWithSlugs); err != nil {
		return nil, fmt.Errorf("failed to count tasks with slugs: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// PHASE 1: Extract epic and feature slugs from task paths
	// This provides the highest coverage since all tasks have file_path
	rows, err := tx.Query(`
		SELECT t.id, t.file_path, t.feature_id, f.epic_id
		FROM tasks t
		JOIN features f ON t.feature_id = f.id
		WHERE t.file_path IS NOT NULL AND t.file_path != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks for phase 1: %w", err)
	}

	type ParentUpdate struct {
		epicID    int64
		epicSlug  string
		featureID int64
		featSlug  string
	}
	parentUpdates := make(map[int64]ParentUpdate) // key: epic_id or feature_id

	for rows.Next() {
		var taskID, featureID, epicID int64
		var filePath string
		if err := rows.Scan(&taskID, &filePath, &featureID, &epicID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan task row in phase 1: %w", err)
		}

		// Extract epic slug from task path
		epicSlug := extractEpicSlugFromPath(filePath)
		if epicSlug != "" {
			if existing, ok := parentUpdates[epicID]; !ok || existing.epicSlug == "" {
				update := parentUpdates[epicID]
				update.epicID = epicID
				update.epicSlug = epicSlug
				parentUpdates[epicID] = update
			}
		}

		// Extract feature slug from task path
		featureSlug := extractFeatureSlugFromPath(filePath)
		if featureSlug != "" {
			key := featureID + 1000000 // Offset to avoid collision with epic IDs
			if existing, ok := parentUpdates[key]; !ok || existing.featSlug == "" {
				update := parentUpdates[key]
				update.featureID = featureID
				update.featSlug = featureSlug
				parentUpdates[key] = update
			}
		}
	}
	rows.Close()

	// Apply epic updates from phase 1
	for _, update := range parentUpdates {
		if update.epicSlug != "" {
			result, err := tx.Exec("UPDATE epics SET slug = ? WHERE id = ? AND (slug IS NULL OR slug = '')", update.epicSlug, update.epicID)
			if err != nil {
				return nil, fmt.Errorf("failed to update epic slug in phase 1: %w", err)
			}
			rows, _ := result.RowsAffected()
			stats.EpicsUpdated += int(rows)
		}
		if update.featSlug != "" {
			result, err := tx.Exec("UPDATE features SET slug = ? WHERE id = ? AND (slug IS NULL OR slug = '')", update.featSlug, update.featureID)
			if err != nil {
				return nil, fmt.Errorf("failed to update feature slug in phase 1: %w", err)
			}
			rows, _ := result.RowsAffected()
			stats.FeaturesUpdated += int(rows)
		}
	}

	// PHASE 2: Extract epic slugs from feature paths
	// This fills gaps for epics that don't have tasks yet
	rows, err = tx.Query(`
		SELECT f.id, f.file_path, f.epic_id
		FROM features f
		WHERE f.file_path IS NOT NULL AND f.file_path != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query features for phase 2: %w", err)
	}

	var phase2EpicUpdates []struct {
		id   int64
		slug string
	}

	for rows.Next() {
		var featureID, epicID int64
		var filePath string
		if err := rows.Scan(&featureID, &filePath, &epicID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan feature row in phase 2: %w", err)
		}

		epicSlug := extractEpicSlugFromPath(filePath)
		if epicSlug != "" {
			phase2EpicUpdates = append(phase2EpicUpdates, struct {
				id   int64
				slug string
			}{epicID, epicSlug})
		}
	}
	rows.Close()

	for _, update := range phase2EpicUpdates {
		result, err := tx.Exec("UPDATE epics SET slug = ? WHERE id = ? AND (slug IS NULL OR slug = '')", update.slug, update.id)
		if err != nil {
			return nil, fmt.Errorf("failed to update epic slug in phase 2: %w", err)
		}
		rowsAff, _ := result.RowsAffected()
		stats.EpicsUpdated += int(rowsAff)
	}

	// PHASE 3: Extract slugs from own file_path (original behavior)

	// Backfill epics from their own paths
	rows, err = tx.Query("SELECT id, file_path FROM epics WHERE file_path IS NOT NULL AND file_path != '' AND (slug IS NULL OR slug = '')")
	if err != nil {
		return nil, fmt.Errorf("failed to query epics for phase 3: %w", err)
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
			return nil, fmt.Errorf("failed to scan epic row in phase 3: %w", err)
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
		result, err := tx.Exec("UPDATE epics SET slug = ? WHERE id = ?", update.slug, update.id)
		if err != nil {
			return nil, fmt.Errorf("failed to update epic slug in phase 3: %w", err)
		}
		rowsAff, _ := result.RowsAffected()
		stats.EpicsUpdated += int(rowsAff)
	}

	// Backfill features from their own paths
	rows, err = tx.Query("SELECT id, file_path FROM features WHERE file_path IS NOT NULL AND file_path != '' AND (slug IS NULL OR slug = '')")
	if err != nil {
		return nil, fmt.Errorf("failed to query features for phase 3: %w", err)
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
			return nil, fmt.Errorf("failed to scan feature row in phase 3: %w", err)
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
		result, err := tx.Exec("UPDATE features SET slug = ? WHERE id = ?", update.slug, update.id)
		if err != nil {
			return nil, fmt.Errorf("failed to update feature slug in phase 3: %w", err)
		}
		rowsAff, _ := result.RowsAffected()
		stats.FeaturesUpdated += int(rowsAff)
	}

	// Backfill tasks from their own paths
	rows, err = tx.Query("SELECT id, file_path FROM tasks WHERE file_path IS NOT NULL AND file_path != ''")
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks for phase 3: %w", err)
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
			return nil, fmt.Errorf("failed to scan task row in phase 3: %w", err)
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
		result, err := tx.Exec("UPDATE tasks SET slug = ? WHERE id = ?", update.slug, update.id)
		if err != nil {
			return nil, fmt.Errorf("failed to update task slug in phase 3: %w", err)
		}
		rowsAff, _ := result.RowsAffected()
		stats.TasksUpdated += int(rowsAff)
	}

	// In dry-run mode, rollback transaction and return stats
	if dryRun {
		// Don't commit - just return stats
		return stats, nil
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return stats, nil
}

// extractEpicSlugFromPath extracts the slug from an epic file path
// Works with epic.md paths, feature paths, and task paths
// Examples:
//   "docs/plan/E05-task-mgmt-cli-capabilities/epic.md" -> "task-mgmt-cli-capabilities"
//   "docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/feature.md" -> "task-mgmt-cli-core"
//   "docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/tasks/T-E04-F01-001.md" -> "task-mgmt-cli-core"
func extractEpicSlugFromPath(filePath string) string {
	if filePath == "" {
		return ""
	}

	// Find "E##-" pattern (first occurrence is the epic key)
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

	// Find the first '-' after 'E##' to get slug start
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

	// Find the end of the epic slug by looking for:
	// 1. "/epic.md" (epic's own path)
	// 2. Next '/' after epic key (feature or task path - must contain E##-F## pattern after)

	// First, try to find "/epic.md"
	epicMdPattern := "/epic.md"
	epicMdIndex := -1
	for i := 0; i <= len(filePath)-len(epicMdPattern); i++ {
		if filePath[i:i+len(epicMdPattern)] == epicMdPattern {
			epicMdIndex = i
			break
		}
	}

	if epicMdIndex != -1 && slugStart < epicMdIndex {
		// Epic's own path
		return filePath[slugStart:epicMdIndex]
	}

	// Not an epic.md path, so look for the next '/' after slug start
	// But only if the path contains a feature key (E##-F##) or tasks/ folder
	// This ensures we're extracting from valid feature/task paths
	hasFeatureKey := false
	hasTasks := false

	// Check if path contains feature key pattern (E##-F##)
	// Look for /E##-F##- pattern (folder name)
	for i := 1; i < len(filePath)-6; i++ {
		if filePath[i-1] == '/' && filePath[i] == 'E' {
			// Check for E##-F## or E###-F## patterns
			// E#-F#-
			if i+5 < len(filePath) && filePath[i+2] == '-' && filePath[i+3] == 'F' && filePath[i+5] == '-' {
				hasFeatureKey = true
				break
			}
			// E##-F#-
			if i+6 < len(filePath) && filePath[i+3] == '-' && filePath[i+4] == 'F' && filePath[i+6] == '-' {
				hasFeatureKey = true
				break
			}
			// E#-F##-
			if i+6 < len(filePath) && filePath[i+2] == '-' && filePath[i+3] == 'F' && filePath[i+6] == '-' {
				hasFeatureKey = true
				break
			}
			// E##-F##-
			if i+7 < len(filePath) && filePath[i+3] == '-' && filePath[i+4] == 'F' && filePath[i+7] == '-' {
				hasFeatureKey = true
				break
			}
		}
	}

	// Check if path contains /tasks/
	tasksPattern := "/tasks/"
	for i := 0; i <= len(filePath)-len(tasksPattern); i++ {
		if filePath[i:i+len(tasksPattern)] == tasksPattern {
			hasTasks = true
			break
		}
	}

	if hasFeatureKey || hasTasks {
		// Valid feature or task path - extract epic slug before next '/'
		for i := slugStart; i < len(filePath); i++ {
			if filePath[i] == '/' {
				slug := filePath[slugStart:i]

				// VALIDATION: Reject if slug matches feature key pattern (F## or F###)
				// This happens when epic folder has no slug (e.g., "E08/E08-F01/...")
				// and we incorrectly extract "F01" as the epic slug
				if isFeatureKeyPattern(slug) {
					return ""
				}

				return slug
			}
		}
	}

	// Not a valid path for epic slug extraction
	return ""
}

// isFeatureKeyPattern checks if a string matches feature key pattern (F## or F###)
// Examples: "F01", "F05", "F123" return true
// Examples: "F01-migrations", "feature-name" return false for partial matches
func isFeatureKeyPattern(s string) bool {
	if len(s) < 2 || s[0] != 'F' {
		return false
	}

	// Check for F## pattern (2 digits)
	if len(s) == 3 && s[1] >= '0' && s[1] <= '9' && s[2] >= '0' && s[2] <= '9' {
		return true
	}

	// Check for F### pattern (3 digits)
	if len(s) == 4 && s[1] >= '0' && s[1] <= '9' && s[2] >= '0' && s[2] <= '9' && s[3] >= '0' && s[3] <= '9' {
		return true
	}

	// Also check for F##-something or F###-something (feature folder without epic slug)
	// Examples: "F01-migrations", "F05-slug-architecture"
	if len(s) > 3 && s[1] >= '0' && s[1] <= '9' {
		// F#-...
		if s[2] == '-' {
			return true
		}
		// F##-...
		if len(s) > 4 && s[2] >= '0' && s[2] <= '9' && s[3] == '-' {
			return true
		}
		// F###-...
		if len(s) > 5 && s[2] >= '0' && s[2] <= '9' && s[3] >= '0' && s[3] <= '9' && s[4] == '-' {
			return true
		}
	}

	return false
}

// extractFeatureSlugFromPath extracts the slug from a feature file path
// Works with feature's own paths (prd.md, feature.md) and task paths
// Examples:
//   "docs/plan/E06-intelligent-scanning/E06-F04-incremental-sync-engine/prd.md" -> "incremental-sync-engine"
//   "docs/plan/E04-epic/E04-F01-database-schema/tasks/T-E04-F01-001.md" -> "database-schema"
func extractFeatureSlugFromPath(filePath string) string {
	if filePath == "" {
		return ""
	}

	// Find the feature key pattern "E##-F##-" in the path
	// We look for the complete pattern to distinguish from task keys like "T-E##-F##-###"
	// The feature folder pattern is: /E##-F##-slug/
	featureKeyStart := -1

	// Search for E##-F## pattern where it's preceded by '/' (folder name, not task key)
	for i := 1; i < len(filePath)-6; i++ {
		if filePath[i-1] == '/' && filePath[i] == 'E' {
			// Check for E##-F## pattern
			if i+6 < len(filePath) {
				// E##-F##
				if filePath[i+2] == '-' && filePath[i+3] == 'F' && filePath[i+5] == '-' {
					if filePath[i+1] >= '0' && filePath[i+1] <= '9' && filePath[i+4] >= '0' && filePath[i+4] <= '9' {
						featureKeyStart = i + 6 // After "E#-F#-"
						break
					}
				}
				// E###-F## (3-digit epic)
				if i+7 < len(filePath) && filePath[i+3] == '-' && filePath[i+4] == 'F' && filePath[i+6] == '-' {
					if filePath[i+1] >= '0' && filePath[i+1] <= '9' && filePath[i+2] >= '0' && filePath[i+2] <= '9' &&
						filePath[i+5] >= '0' && filePath[i+5] <= '9' {
						featureKeyStart = i + 7 // After "E##-F#-"
						break
					}
				}
				// E##-F### (3-digit feature)
				if i+7 < len(filePath) && filePath[i+2] == '-' && filePath[i+3] == 'F' && filePath[i+6] == '-' {
					if filePath[i+1] >= '0' && filePath[i+1] <= '9' &&
						filePath[i+4] >= '0' && filePath[i+4] <= '9' && filePath[i+5] >= '0' && filePath[i+5] <= '9' {
						featureKeyStart = i + 7 // After "E#-F##-"
						break
					}
				}
				// E###-F### (both 3-digit)
				if i+8 < len(filePath) && filePath[i+3] == '-' && filePath[i+4] == 'F' && filePath[i+7] == '-' {
					if filePath[i+1] >= '0' && filePath[i+1] <= '9' && filePath[i+2] >= '0' && filePath[i+2] <= '9' &&
						filePath[i+5] >= '0' && filePath[i+5] <= '9' && filePath[i+6] >= '0' && filePath[i+6] <= '9' {
						featureKeyStart = i + 8 // After "E##-F##-"
						break
					}
				}
			}
		}
	}

	if featureKeyStart == -1 {
		return ""
	}

	// Find the end of the feature slug by looking for:
	// 1. "/prd.md" (feature's own path)
	// 2. "/feature.md" (feature's own path)
	// 3. "/tasks/" (task path) - extract slug before this
	// 4. Other .md files - only valid if it's prd.md or feature.md

	// Check for "/prd.md"
	prdPattern := "/prd.md"
	for i := 0; i <= len(filePath)-len(prdPattern); i++ {
		if filePath[i:i+len(prdPattern)] == prdPattern {
			if featureKeyStart < i {
				return filePath[featureKeyStart:i]
			}
		}
	}

	// Check for "/feature.md"
	featurePattern := "/feature.md"
	for i := 0; i <= len(filePath)-len(featurePattern); i++ {
		if filePath[i:i+len(featurePattern)] == featurePattern {
			if featureKeyStart < i {
				return filePath[featureKeyStart:i]
			}
		}
	}

	// Check for "/tasks/" (task path)
	tasksPattern := "/tasks/"
	for i := 0; i <= len(filePath)-len(tasksPattern); i++ {
		if filePath[i:i+len(tasksPattern)] == tasksPattern {
			if featureKeyStart < i {
				return filePath[featureKeyStart:i]
			}
		}
	}

	// If path ends with other .md files (not prd.md or feature.md), it's not valid
	// We only extract from known valid paths
	return ""
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
