package sync

import (
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConflictDetectionWithLastSyncTime tests enhanced conflict detection
// that considers both file.mtime and db.updated_at relative to last_sync_time
func TestConflictDetectionWithLastSyncTime(t *testing.T) {
	detector := NewConflictDetector()
	now := time.Now()

	t.Run("no conflict when only file modified since last sync", func(t *testing.T) {
		// File modified after last sync, DB not modified
		lastSync := now.Add(-2 * time.Hour)
		fileMTime := now.Add(-1 * time.Hour)    // Modified 1 hour ago
		dbUpdateTime := now.Add(-3 * time.Hour) // Modified 3 hours ago (before last sync)

		filePath := "/path/to/task.md"
		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "New Title",
			FilePath:   filePath,
			ModifiedAt: fileMTime,
		}

		dbTask := &models.Task{
			ID:        1,
			Key:       "T-E04-F07-001",
			Title:     "Old Title",
			FilePath:  &filePath,
			Status:    models.TaskStatusTodo,
			Priority:  5,
			UpdatedAt: dbUpdateTime,
		}

		// Act - detect conflicts with last sync time
		conflicts := detector.DetectConflictsWithSync(fileData, dbTask, &lastSync)

		// Assert - no conflict because only file changed (this is expected file update)
		// Only file_path should be detected as needing update (not a true conflict)
		assert.Empty(t, conflicts, "Expected no conflicts when only file modified")
	})

	t.Run("no conflict when only DB modified since last sync", func(t *testing.T) {
		// DB modified after last sync, file not modified
		lastSync := now.Add(-2 * time.Hour)
		fileMTime := now.Add(-3 * time.Hour)    // Modified 3 hours ago (before last sync)
		dbUpdateTime := now.Add(-1 * time.Hour) // Modified 1 hour ago

		filePath := "/path/to/task.md"
		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "Old Title",
			FilePath:   filePath,
			ModifiedAt: fileMTime,
		}

		dbTask := &models.Task{
			ID:        1,
			Key:       "T-E04-F07-001",
			Title:     "Old Title",
			FilePath:  &filePath,
			Status:    models.TaskStatusTodo,
			Priority:  5,
			UpdatedAt: dbUpdateTime,
		}

		// Act - detect conflicts with last sync time
		conflicts := detector.DetectConflictsWithSync(fileData, dbTask, &lastSync)

		// Assert - no conflict because only DB changed (skip file, DB is current)
		assert.Empty(t, conflicts, "Expected no conflicts when only DB modified")
	})

	t.Run("no conflict when both modified but metadata identical", func(t *testing.T) {
		// Both modified after last sync, but values match
		lastSync := now.Add(-3 * time.Hour)
		fileMTime := now.Add(-1 * time.Hour)
		dbUpdateTime := now.Add(-2 * time.Hour)

		filePath := "/path/to/task.md"
		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "Same Title",
			FilePath:   filePath,
			ModifiedAt: fileMTime,
		}

		dbTask := &models.Task{
			ID:        1,
			Key:       "T-E04-F07-001",
			Title:     "Same Title",
			FilePath:  &filePath,
			Status:    models.TaskStatusTodo,
			Priority:  5,
			UpdatedAt: dbUpdateTime,
		}

		// Act
		conflicts := detector.DetectConflictsWithSync(fileData, dbTask, &lastSync)

		// Assert - no conflict because values are identical (only timestamps differ)
		assert.Empty(t, conflicts, "Expected no conflicts when metadata identical")
	})

	t.Run("conflict detected when both modified and title differs", func(t *testing.T) {
		// Both file and DB modified after last sync with different values
		lastSync := now.Add(-3 * time.Hour)
		fileMTime := now.Add(-1 * time.Hour)
		dbUpdateTime := now.Add(-2 * time.Hour)

		filePath := "/path/to/task.md"
		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "File Title",
			FilePath:   filePath,
			ModifiedAt: fileMTime,
		}

		dbTask := &models.Task{
			ID:        1,
			Key:       "T-E04-F07-001",
			Title:     "DB Title",
			FilePath:  &filePath,
			Status:    models.TaskStatusTodo,
			Priority:  5,
			UpdatedAt: dbUpdateTime,
		}

		// Act
		conflicts := detector.DetectConflictsWithSync(fileData, dbTask, &lastSync)

		// Assert - TRUE conflict: both modified since last sync AND values differ
		require.Len(t, conflicts, 1, "Expected one conflict")
		assert.Equal(t, "title", conflicts[0].Field)
		assert.Equal(t, "File Title", conflicts[0].FileValue)
		assert.Equal(t, "DB Title", conflicts[0].DatabaseValue)
	})

	t.Run("multiple conflicts when both modified with different values", func(t *testing.T) {
		lastSync := now.Add(-3 * time.Hour)
		fileMTime := now.Add(-1 * time.Hour)
		dbUpdateTime := now.Add(-2 * time.Hour)

		fileDesc := "File description"
		dbDesc := "DB description"
		filePath := "/path/to/task.md"

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "File Title",
			Description: &fileDesc,
			FilePath:    filePath,
			ModifiedAt:  fileMTime,
		}

		dbTask := &models.Task{
			ID:          1,
			Key:         "T-E04-F07-001",
			Title:       "DB Title",
			Description: &dbDesc,
			FilePath:    &filePath,
			Status:      models.TaskStatusTodo,
			Priority:    5,
			UpdatedAt:   dbUpdateTime,
		}

		// Act
		conflicts := detector.DetectConflictsWithSync(fileData, dbTask, &lastSync)

		// Assert
		assert.Len(t, conflicts, 2, "Expected two conflicts (title, description)")

		fields := make(map[string]bool)
		for _, c := range conflicts {
			fields[c.Field] = true
		}
		assert.True(t, fields["title"], "Expected title conflict")
		assert.True(t, fields["description"], "Expected description conflict")
	})

	t.Run("clock skew tolerance applied", func(t *testing.T) {
		// Test clock skew buffer (±60 seconds)
		// File modified 59 seconds before last sync (within tolerance window)
		// Both file and DB were modified around the same time, so conflicts should be detected
		lastSync := now.Add(-1 * time.Hour)
		fileMTime := lastSync.Add(-59 * time.Second) // Just within tolerance
		dbUpdateTime := now.Add(-30 * time.Minute)   // Definitely after last sync

		filePath := "/path/to/task.md"
		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "File Title",
			FilePath:   filePath,
			ModifiedAt: fileMTime,
		}

		dbTask := &models.Task{
			ID:        1,
			Key:       "T-E04-F07-001",
			Title:     "DB Title",
			FilePath:  &filePath,
			Status:    models.TaskStatusTodo,
			Priority:  5,
			UpdatedAt: dbUpdateTime,
		}

		// Act - with clock skew tolerance, both file and DB are considered modified
		conflicts := detector.DetectConflictsWithSync(fileData, dbTask, &lastSync)

		// Assert - conflict detected because both were modified and titles differ
		assert.NotEmpty(t, conflicts, "Expected conflicts because both file and DB modified with different titles")
		assert.Equal(t, 1, len(conflicts), "Expected one title conflict")
		assert.Equal(t, "title", conflicts[0].Field)
	})

	t.Run("falls back to basic detection when last_sync_time is nil", func(t *testing.T) {
		// When no last sync time provided, use basic field comparison
		filePath := "/path/to/task.md"
		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "New Title",
			FilePath:   filePath,
			ModifiedAt: time.Now(),
		}

		dbTask := &models.Task{
			ID:        1,
			Key:       "T-E04-F07-001",
			Title:     "Old Title",
			FilePath:  &filePath,
			Status:    models.TaskStatusTodo,
			Priority:  5,
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		}

		// Act - nil last sync time means full scan mode
		conflicts := detector.DetectConflictsWithSync(fileData, dbTask, nil)

		// Assert - should detect title conflict as before
		require.Len(t, conflicts, 1, "Expected one conflict")
		assert.Equal(t, "title", conflicts[0].Field)
	})
}

// TestManualConflictResolution tests the manual resolution strategy
func TestManualConflictResolution(t *testing.T) {
	t.Run("manual strategy with simulated user input", func(t *testing.T) {
		_ = NewConflictResolver() // Note: using ManualConflictResolver directly in this test

		filePath := "/path/to/task.md"
		fileDesc := "File description"
		dbDesc := "DB description"

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "File Title",
			Description: &fileDesc,
			FilePath:    filePath,
			ModifiedAt:  time.Now(),
		}

		dbTask := &models.Task{
			ID:          1,
			FeatureID:   10,
			Key:         "T-E04-F07-001",
			Title:       "DB Title",
			Description: &dbDesc,
			FilePath:    &filePath,
			Status:      models.TaskStatusTodo,
			Priority:    5,
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
		}

		conflicts := []Conflict{
			{TaskKey: "T-E04-F07-001", Field: "title", FileValue: "File Title", DatabaseValue: "DB Title"},
			{TaskKey: "T-E04-F07-001", Field: "description", FileValue: fileDesc, DatabaseValue: dbDesc},
		}

		// Create manual resolver with simulated user choices
		// Choices: file, db (choose file for title, db for description)
		userChoices := []string{"file", "db"}
		manualResolver := &ManualConflictResolver{
			choices: userChoices,
			index:   0,
		}

		// Act
		resolved, report, err := manualResolver.ResolveManually(conflicts, fileData, dbTask)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "File Title", resolved.Title, "Title should be from file (user chose 'file')")
		assert.Equal(t, dbDesc, *resolved.Description, "Description should be from DB (user chose 'db')")

		// Check report records decisions
		require.Len(t, report, 2)
		assert.Equal(t, "title", report[0].Field)
		assert.Equal(t, "file", report[0].Resolution)
		assert.Equal(t, "description", report[1].Field)
		assert.Equal(t, "db", report[1].Resolution)
	})
}

// ManualConflictResolver is a test helper that simulates user input
type ManualConflictResolver struct {
	choices []string
	index   int
}

// ConflictResolution records a manual resolution decision
type ConflictResolution struct {
	Field      string
	Resolution string // "file" or "db"
}

// ResolveManually simulates manual conflict resolution
func (m *ManualConflictResolver) ResolveManually(
	conflicts []Conflict,
	fileData *TaskMetadata,
	dbTask *models.Task,
) (*models.Task, []ConflictResolution, error) {
	// Create copy of database task
	resolved := &models.Task{
		ID:          dbTask.ID,
		FeatureID:   dbTask.FeatureID,
		Key:         dbTask.Key,
		Title:       dbTask.Title,
		Status:      dbTask.Status,
		Priority:    dbTask.Priority,
		CreatedAt:   dbTask.CreatedAt,
		UpdatedAt:   dbTask.UpdatedAt,
	}

	if dbTask.Description != nil {
		desc := *dbTask.Description
		resolved.Description = &desc
	}

	if dbTask.FilePath != nil {
		path := *dbTask.FilePath
		resolved.FilePath = &path
	}

	if dbTask.AgentType != nil {
		agentType := *dbTask.AgentType
		resolved.AgentType = &agentType
	}

	report := []ConflictResolution{}

	// For each conflict, use simulated user choice
	for _, conflict := range conflicts {
		if m.index >= len(m.choices) {
			break
		}

		choice := m.choices[m.index]
		m.index++

		if choice == "file" {
			// Apply file value
			switch conflict.Field {
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
		// If choice == "db", keep database value (already in resolved)

		report = append(report, ConflictResolution{
			Field:      conflict.Field,
			Resolution: choice,
		})
	}

	return resolved, report, nil
}
