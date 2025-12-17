package sync

import (
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConflictResolver_ResolveConflicts(t *testing.T) {
	resolver := NewConflictResolver()

	t.Run("file-wins strategy updates all conflicting fields", func(t *testing.T) {
		// Arrange
		fileDesc := "New description from file"
		dbDesc := "Old description in database"
		newPath := "/new/path/task.md"
		oldPath := "/old/path/task.md"
		agentType := models.AgentTypeBackend

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "New Title",
			Description: &fileDesc,
			FilePath:    newPath,
			ModifiedAt:  time.Now(),
		}

		dbTask := &models.Task{
			ID:          1,
			FeatureID:   10,
			Key:         "T-E04-F07-001",
			Title:       "Old Title",
			Description: &dbDesc,
			FilePath:    &oldPath,
			Status:      models.TaskStatusInProgress,
			Priority:    5,
			AgentType:   &agentType,
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
		}

		conflicts := []Conflict{
			{TaskKey: "T-E04-F07-001", Field: "title", FileValue: "New Title", DatabaseValue: "Old Title"},
			{TaskKey: "T-E04-F07-001", Field: "description", FileValue: fileDesc, DatabaseValue: dbDesc},
			{TaskKey: "T-E04-F07-001", Field: "file_path", FileValue: newPath, DatabaseValue: oldPath},
		}

		// Act
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyFileWins)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "New Title", resolved.Title, "Title should be from file")
		assert.Equal(t, fileDesc, *resolved.Description, "Description should be from file")
		assert.Equal(t, newPath, *resolved.FilePath, "FilePath should be from file")

		// Database-only fields should be preserved
		assert.Equal(t, models.TaskStatusInProgress, resolved.Status, "Status should be preserved")
		assert.Equal(t, 5, resolved.Priority, "Priority should be preserved")
		assert.Equal(t, agentType, *resolved.AgentType, "AgentType should be preserved")
		assert.Equal(t, int64(1), resolved.ID, "ID should be preserved")
		assert.Equal(t, int64(10), resolved.FeatureID, "FeatureID should be preserved")
	})

	t.Run("database-wins strategy keeps all database values", func(t *testing.T) {
		// Arrange
		fileDesc := "New description from file"
		dbDesc := "Old description in database"
		newPath := "/new/path/task.md"
		oldPath := "/old/path/task.md"

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "New Title",
			Description: &fileDesc,
			FilePath:    newPath,
			ModifiedAt:  time.Now(),
		}

		dbTask := &models.Task{
			ID:          1,
			FeatureID:   10,
			Key:         "T-E04-F07-001",
			Title:       "Old Title",
			Description: &dbDesc,
			FilePath:    &oldPath,
			Status:      models.TaskStatusTodo,
			Priority:    3,
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
		}

		conflicts := []Conflict{
			{TaskKey: "T-E04-F07-001", Field: "title", FileValue: "New Title", DatabaseValue: "Old Title"},
			{TaskKey: "T-E04-F07-001", Field: "description", FileValue: fileDesc, DatabaseValue: dbDesc},
			{TaskKey: "T-E04-F07-001", Field: "file_path", FileValue: newPath, DatabaseValue: oldPath},
		}

		// Act
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyDatabaseWins)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "Old Title", resolved.Title, "Title should be from database")
		assert.Equal(t, dbDesc, *resolved.Description, "Description should be from database")
		assert.Equal(t, oldPath, *resolved.FilePath, "FilePath should be from database")
		assert.Equal(t, models.TaskStatusTodo, resolved.Status, "Status should be preserved")
		assert.Equal(t, 3, resolved.Priority, "Priority should be preserved")
	})

	t.Run("newer-wins strategy uses file when file is newer", func(t *testing.T) {
		// Arrange
		fileDesc := "New description from file"
		dbDesc := "Old description in database"
		newPath := "/new/path/task.md"
		oldPath := "/old/path/task.md"

		// File modified time is newer than database updated time
		fileModTime := time.Now()
		dbUpdateTime := fileModTime.Add(-2 * time.Hour)

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "New Title",
			Description: &fileDesc,
			FilePath:    newPath,
			ModifiedAt:  fileModTime,
		}

		dbTask := &models.Task{
			ID:          1,
			FeatureID:   10,
			Key:         "T-E04-F07-001",
			Title:       "Old Title",
			Description: &dbDesc,
			FilePath:    &oldPath,
			Status:      models.TaskStatusTodo,
			Priority:    3,
			UpdatedAt:   dbUpdateTime,
		}

		conflicts := []Conflict{
			{TaskKey: "T-E04-F07-001", Field: "title", FileValue: "New Title", DatabaseValue: "Old Title"},
			{TaskKey: "T-E04-F07-001", Field: "description", FileValue: fileDesc, DatabaseValue: dbDesc},
			{TaskKey: "T-E04-F07-001", Field: "file_path", FileValue: newPath, DatabaseValue: oldPath},
		}

		// Act
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyNewerWins)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "New Title", resolved.Title, "Title should be from file (newer)")
		assert.Equal(t, fileDesc, *resolved.Description, "Description should be from file (newer)")
		assert.Equal(t, newPath, *resolved.FilePath, "FilePath should be from file (newer)")
	})

	t.Run("newer-wins strategy uses database when database is newer", func(t *testing.T) {
		// Arrange
		fileDesc := "Old description from file"
		dbDesc := "New description in database"
		newPath := "/new/path/task.md"
		oldPath := "/old/path/task.md"

		// Database updated time is newer than file modified time
		dbUpdateTime := time.Now()
		fileModTime := dbUpdateTime.Add(-2 * time.Hour)

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "Old Title",
			Description: &fileDesc,
			FilePath:    newPath,
			ModifiedAt:  fileModTime,
		}

		dbTask := &models.Task{
			ID:          1,
			FeatureID:   10,
			Key:         "T-E04-F07-001",
			Title:       "New Title",
			Description: &dbDesc,
			FilePath:    &oldPath,
			Status:      models.TaskStatusTodo,
			Priority:    3,
			UpdatedAt:   dbUpdateTime,
		}

		conflicts := []Conflict{
			{TaskKey: "T-E04-F07-001", Field: "title", FileValue: "Old Title", DatabaseValue: "New Title"},
			{TaskKey: "T-E04-F07-001", Field: "description", FileValue: fileDesc, DatabaseValue: dbDesc},
			{TaskKey: "T-E04-F07-001", Field: "file_path", FileValue: newPath, DatabaseValue: oldPath},
		}

		// Act
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyNewerWins)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "New Title", resolved.Title, "Title should be from database (newer)")
		assert.Equal(t, dbDesc, *resolved.Description, "Description should be from database (newer)")
		assert.Equal(t, oldPath, *resolved.FilePath, "FilePath should be from database (newer)")
	})

	t.Run("file-wins with nil description in file preserves database description", func(t *testing.T) {
		// Arrange
		dbDesc := "Database description"
		filePath := "/path/task.md"

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "New Title",
			Description: nil, // Nil description in file
			FilePath:    filePath,
			ModifiedAt:  time.Now(),
		}

		dbTask := &models.Task{
			ID:          1,
			FeatureID:   10,
			Key:         "T-E04-F07-001",
			Title:       "Old Title",
			Description: &dbDesc,
			FilePath:    &filePath,
			Status:      models.TaskStatusTodo,
			Priority:    3,
		}

		conflicts := []Conflict{
			{TaskKey: "T-E04-F07-001", Field: "title", FileValue: "New Title", DatabaseValue: "Old Title"},
		}

		// Act
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyFileWins)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "New Title", resolved.Title, "Title should be from file")
		assert.Equal(t, dbDesc, *resolved.Description, "Description should be preserved from database")
	})

	t.Run("resolves empty conflict list without error", func(t *testing.T) {
		// Arrange
		filePath := "/path/task.md"
		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "Test Task",
			FilePath:   filePath,
			ModifiedAt: time.Now(),
		}

		dbTask := &models.Task{
			ID:        1,
			FeatureID: 10,
			Key:       "T-E04-F07-001",
			Title:     "Test Task",
			FilePath:  &filePath,
			Status:    models.TaskStatusTodo,
			Priority:  3,
		}

		conflicts := []Conflict{} // No conflicts

		// Act
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyFileWins)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, dbTask.Title, resolved.Title)
		assert.Equal(t, dbTask.Status, resolved.Status)
	})

	t.Run("returns copy of database task, not original", func(t *testing.T) {
		// Arrange
		filePath := "/path/task.md"
		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "New Title",
			FilePath:   filePath,
			ModifiedAt: time.Now(),
		}

		dbTask := &models.Task{
			ID:        1,
			FeatureID: 10,
			Key:       "T-E04-F07-001",
			Title:     "Old Title",
			FilePath:  &filePath,
			Status:    models.TaskStatusTodo,
			Priority:  3,
		}

		conflicts := []Conflict{
			{TaskKey: "T-E04-F07-001", Field: "title", FileValue: "New Title", DatabaseValue: "Old Title"},
		}

		// Act
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyFileWins)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "New Title", resolved.Title)
		assert.Equal(t, "Old Title", dbTask.Title, "Original database task should not be modified")
	})

	t.Run("preserves all database-only fields", func(t *testing.T) {
		// Arrange
		filePath := "/path/task.md"
		agentType := models.AgentTypeAPI
		assignedAgent := "test-agent"
		dependsOn := `["T-E04-F07-001"]`
		blockedReason := "Waiting for API"

		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "New Title",
			FilePath:   filePath,
			ModifiedAt: time.Now(),
		}

		dbTask := &models.Task{
			ID:            1,
			FeatureID:     10,
			Key:           "T-E04-F07-001",
			Title:         "Old Title",
			FilePath:      &filePath,
			Status:        models.TaskStatusBlocked,
			Priority:      7,
			AgentType:     &agentType,
			AssignedAgent: &assignedAgent,
			DependsOn:     &dependsOn,
			BlockedReason: &blockedReason,
			CreatedAt:     time.Now().Add(-24 * time.Hour),
			UpdatedAt:     time.Now().Add(-1 * time.Hour),
		}

		conflicts := []Conflict{
			{TaskKey: "T-E04-F07-001", Field: "title", FileValue: "New Title", DatabaseValue: "Old Title"},
		}

		// Act
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyFileWins)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, models.TaskStatusBlocked, resolved.Status)
		assert.Equal(t, 7, resolved.Priority)
		assert.Equal(t, agentType, *resolved.AgentType)
		assert.Equal(t, assignedAgent, *resolved.AssignedAgent)
		assert.Equal(t, dependsOn, *resolved.DependsOn)
		assert.Equal(t, blockedReason, *resolved.BlockedReason)
		assert.Equal(t, dbTask.CreatedAt, resolved.CreatedAt)
		assert.Equal(t, dbTask.UpdatedAt, resolved.UpdatedAt)
	})
}
