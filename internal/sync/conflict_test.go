package sync

import (
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestConflictDetector_DetectConflicts(t *testing.T) {
	detector := NewConflictDetector()

	t.Run("no conflicts when file and database match", func(t *testing.T) {
		// Arrange
		desc := "Test description"
		filePath := "/path/to/task.md"
		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "Test Task",
			Description: &desc,
			FilePath:    filePath,
			ModifiedAt:  time.Now(),
		}

		dbTask := &models.Task{
			ID:          1,
			Key:         "T-E04-F07-001",
			Title:       "Test Task",
			Description: &desc,
			FilePath:    &filePath,
			Status:      models.TaskStatusTodo,
			Priority:    5,
		}

		// Act
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert
		assert.Empty(t, conflicts, "Expected no conflicts when file and database match")
	})

	t.Run("detects title conflict", func(t *testing.T) {
		// Arrange
		filePath := "/path/to/task.md"
		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "New Title from File",
			FilePath:   filePath,
			ModifiedAt: time.Now(),
		}

		dbTask := &models.Task{
			ID:       1,
			Key:      "T-E04-F07-001",
			Title:    "Old Title in Database",
			FilePath: &filePath,
			Status:   models.TaskStatusTodo,
			Priority: 5,
		}

		// Act
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert
		assert.Len(t, conflicts, 1, "Expected one conflict for title mismatch")
		assert.Equal(t, "title", conflicts[0].Field)
		assert.Equal(t, "New Title from File", conflicts[0].FileValue)
		assert.Equal(t, "Old Title in Database", conflicts[0].DatabaseValue)
		assert.Equal(t, "T-E04-F07-001", conflicts[0].TaskKey)
	})

	t.Run("no title conflict when file title is empty", func(t *testing.T) {
		// Arrange
		filePath := "/path/to/task.md"
		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "", // Empty title in file
			FilePath:   filePath,
			ModifiedAt: time.Now(),
		}

		dbTask := &models.Task{
			ID:       1,
			Key:      "T-E04-F07-001",
			Title:    "Database Title",
			FilePath: &filePath,
			Status:   models.TaskStatusTodo,
			Priority: 5,
		}

		// Act
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert
		assert.Empty(t, conflicts, "Expected no conflict when file title is empty")
	})

	t.Run("detects description conflict when both exist", func(t *testing.T) {
		// Arrange
		fileDesc := "New description from file"
		dbDesc := "Old description in database"
		filePath := "/path/to/task.md"

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "Test Task",
			Description: &fileDesc,
			FilePath:    filePath,
			ModifiedAt:  time.Now(),
		}

		dbTask := &models.Task{
			ID:          1,
			Key:         "T-E04-F07-001",
			Title:       "Test Task",
			Description: &dbDesc,
			FilePath:    &filePath,
			Status:      models.TaskStatusTodo,
			Priority:    5,
		}

		// Act
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert
		assert.Len(t, conflicts, 1, "Expected one conflict for description mismatch")
		assert.Equal(t, "description", conflicts[0].Field)
		assert.Equal(t, fileDesc, conflicts[0].FileValue)
		assert.Equal(t, dbDesc, conflicts[0].DatabaseValue)
	})

	t.Run("no description conflict when file description is nil", func(t *testing.T) {
		// Arrange
		dbDesc := "Database description"
		filePath := "/path/to/task.md"

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "Test Task",
			Description: nil, // Nil description in file
			FilePath:    filePath,
			ModifiedAt:  time.Now(),
		}

		dbTask := &models.Task{
			ID:          1,
			Key:         "T-E04-F07-001",
			Title:       "Test Task",
			Description: &dbDesc,
			FilePath:    &filePath,
			Status:      models.TaskStatusTodo,
			Priority:    5,
		}

		// Act
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert
		assert.Empty(t, conflicts, "Expected no conflict when file description is nil")
	})

	t.Run("no description conflict when database description is nil", func(t *testing.T) {
		// Arrange
		fileDesc := "File description"
		filePath := "/path/to/task.md"

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "Test Task",
			Description: &fileDesc,
			FilePath:    filePath,
			ModifiedAt:  time.Now(),
		}

		dbTask := &models.Task{
			ID:          1,
			Key:         "T-E04-F07-001",
			Title:       "Test Task",
			Description: nil, // Nil description in database
			FilePath:    &filePath,
			Status:      models.TaskStatusTodo,
			Priority:    5,
		}

		// Act
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert
		assert.Empty(t, conflicts, "Expected no conflict when database description is nil")
	})

	t.Run("detects file_path conflict when database path is different", func(t *testing.T) {
		// Arrange
		actualPath := "/new/path/to/task.md"
		oldPath := "/old/path/to/task.md"

		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "Test Task",
			FilePath:   actualPath,
			ModifiedAt: time.Now(),
		}

		dbTask := &models.Task{
			ID:       1,
			Key:      "T-E04-F07-001",
			Title:    "Test Task",
			FilePath: &oldPath,
			Status:   models.TaskStatusTodo,
			Priority: 5,
		}

		// Act
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert
		assert.Len(t, conflicts, 1, "Expected one conflict for file_path mismatch")
		assert.Equal(t, "file_path", conflicts[0].Field)
		assert.Equal(t, actualPath, conflicts[0].FileValue)
		assert.Equal(t, oldPath, conflicts[0].DatabaseValue)
	})

	t.Run("detects file_path conflict when database path is nil", func(t *testing.T) {
		// Arrange
		actualPath := "/path/to/task.md"

		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "Test Task",
			FilePath:   actualPath,
			ModifiedAt: time.Now(),
		}

		dbTask := &models.Task{
			ID:       1,
			Key:      "T-E04-F07-001",
			Title:    "Test Task",
			FilePath: nil, // Nil path in database
			Status:   models.TaskStatusTodo,
			Priority: 5,
		}

		// Act
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert
		assert.Len(t, conflicts, 1, "Expected one conflict when database path is nil")
		assert.Equal(t, "file_path", conflicts[0].Field)
		assert.Equal(t, actualPath, conflicts[0].FileValue)
		assert.Equal(t, "", conflicts[0].DatabaseValue)
	})

	t.Run("detects multiple conflicts", func(t *testing.T) {
		// Arrange
		fileDesc := "File description"
		dbDesc := "Database description"
		actualPath := "/new/path/task.md"
		oldPath := "/old/path/task.md"

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "New Title",
			Description: &fileDesc,
			FilePath:    actualPath,
			ModifiedAt:  time.Now(),
		}

		dbTask := &models.Task{
			ID:          1,
			Key:         "T-E04-F07-001",
			Title:       "Old Title",
			Description: &dbDesc,
			FilePath:    &oldPath,
			Status:      models.TaskStatusTodo,
			Priority:    5,
		}

		// Act
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert
		assert.Len(t, conflicts, 3, "Expected three conflicts (title, description, file_path)")

		// Check that all three conflicts are present
		fields := make(map[string]bool)
		for _, c := range conflicts {
			fields[c.Field] = true
		}
		assert.True(t, fields["title"], "Expected title conflict")
		assert.True(t, fields["description"], "Expected description conflict")
		assert.True(t, fields["file_path"], "Expected file_path conflict")
	})

	t.Run("does not detect conflicts for database-only fields", func(t *testing.T) {
		// Arrange
		filePath := "/path/to/task.md"
		agentType := "backend"
		assignedAgent := "agent-1"

		fileData := &TaskMetadata{
			Key:        "T-E04-F07-001",
			Title:      "Test Task",
			FilePath:   filePath,
			ModifiedAt: time.Now(),
		}

		dbTask := &models.Task{
			ID:            1,
			Key:           "T-E04-F07-001",
			Title:         "Test Task",
			FilePath:      &filePath,
			Status:        models.TaskStatusInProgress, // Database-only field
			Priority:      10,                          // Database-only field
			AgentType:     &agentType,                  // Database-only field
			AssignedAgent: &assignedAgent,              // Database-only field
		}

		// Act
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert
		assert.Empty(t, conflicts, "Expected no conflicts for database-only fields")
	})
}
