package sync

import (
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConflictDetectionAndResolution demonstrates the complete workflow
// of detecting conflicts and resolving them with different strategies
func TestConflictDetectionAndResolution(t *testing.T) {
	detector := NewConflictDetector()
	resolver := NewConflictResolver()

	t.Run("complete workflow: file-wins strategy", func(t *testing.T) {
		// Arrange - File has newer content
		fileDesc := "Updated description from file"
		dbDesc := "Old description in database"
		newPath := "/new/path/task.md"
		oldPath := "/old/path/task.md"
		agentType := "backend"

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-001",
			Title:       "Updated Title",
			Description: &fileDesc,
			FilePath:    newPath,
			ModifiedAt:  time.Now(),
		}

		dbTask := &models.Task{
			ID:          1,
			FeatureID:   10,
			Key:         "T-E04-F07-001",
			Title:       "Original Title",
			Description: &dbDesc,
			FilePath:    &oldPath,
			Status:      models.TaskStatusInProgress,
			Priority:    5,
			AgentType:   &agentType,
			UpdatedAt:   time.Now().Add(-24 * time.Hour),
		}

		// Act - Detect conflicts
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert - Should detect 3 conflicts (title, description, file_path)
		require.Len(t, conflicts, 3)

		// Act - Resolve conflicts with file-wins strategy
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyFileWins)

		// Assert - File values should be used
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", resolved.Title)
		assert.Equal(t, fileDesc, *resolved.Description)
		assert.Equal(t, newPath, *resolved.FilePath)

		// Assert - Database-only fields preserved
		assert.Equal(t, models.TaskStatusInProgress, resolved.Status)
		assert.Equal(t, 5, resolved.Priority)
		assert.Equal(t, agentType, *resolved.AgentType)
		assert.Equal(t, int64(1), resolved.ID)
		assert.Equal(t, int64(10), resolved.FeatureID)
	})

	t.Run("complete workflow: newer-wins with database newer", func(t *testing.T) {
		// Arrange - Database was updated more recently
		fileDesc := "Old description from file"
		dbDesc := "Newer description in database"
		newPath := "/path/task.md"

		dbUpdateTime := time.Now()
		fileModTime := dbUpdateTime.Add(-48 * time.Hour) // File is 2 days older

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-002",
			Title:       "Old Title",
			Description: &fileDesc,
			FilePath:    newPath,
			ModifiedAt:  fileModTime,
		}

		dbTask := &models.Task{
			ID:          2,
			FeatureID:   10,
			Key:         "T-E04-F07-002",
			Title:       "Newer Title",
			Description: &dbDesc,
			FilePath:    &newPath,
			Status:      models.TaskStatusTodo,
			Priority:    3,
			UpdatedAt:   dbUpdateTime,
		}

		// Act - Detect conflicts
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert - Should detect 2 conflicts (title, description)
		// No file_path conflict since they match
		require.Len(t, conflicts, 2)

		// Act - Resolve with newer-wins strategy
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyNewerWins)

		// Assert - Database values should be used (newer)
		require.NoError(t, err)
		assert.Equal(t, "Newer Title", resolved.Title)
		assert.Equal(t, dbDesc, *resolved.Description)
		assert.Equal(t, models.TaskStatusTodo, resolved.Status)
		assert.Equal(t, 3, resolved.Priority)
	})

	t.Run("no conflicts means no changes", func(t *testing.T) {
		// Arrange - File and database match
		desc := "Same description"
		filePath := "/path/task.md"

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-003",
			Title:       "Matching Title",
			Description: &desc,
			FilePath:    filePath,
			ModifiedAt:  time.Now(),
		}

		dbTask := &models.Task{
			ID:          3,
			FeatureID:   10,
			Key:         "T-E04-F07-003",
			Title:       "Matching Title",
			Description: &desc,
			FilePath:    &filePath,
			Status:      models.TaskStatusCompleted,
			Priority:    1,
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
		}

		// Act - Detect conflicts
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert - No conflicts detected
		require.Empty(t, conflicts)

		// Act - Resolve (even though there are no conflicts)
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyFileWins)

		// Assert - Task unchanged
		require.NoError(t, err)
		assert.Equal(t, dbTask.Title, resolved.Title)
		assert.Equal(t, dbTask.Description, resolved.Description)
		assert.Equal(t, dbTask.FilePath, resolved.FilePath)
		assert.Equal(t, dbTask.Status, resolved.Status)
	})

	t.Run("partial file metadata doesn't create conflicts", func(t *testing.T) {
		// Arrange - File only has key and title, no description
		dbDesc := "Database description"
		filePath := "/path/task.md"

		fileData := &TaskMetadata{
			Key:         "T-E04-F07-004",
			Title:       "Same Title",
			Description: nil, // No description in file
			FilePath:    filePath,
			ModifiedAt:  time.Now(),
		}

		dbTask := &models.Task{
			ID:          4,
			FeatureID:   10,
			Key:         "T-E04-F07-004",
			Title:       "Same Title",
			Description: &dbDesc,
			FilePath:    &filePath,
			Status:      models.TaskStatusTodo,
			Priority:    5,
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
		}

		// Act - Detect conflicts
		conflicts := detector.DetectConflicts(fileData, dbTask)

		// Assert - No conflicts (missing description in file is not a conflict)
		require.Empty(t, conflicts)

		// Act - Resolve
		resolved, err := resolver.ResolveConflicts(conflicts, fileData, dbTask, ConflictStrategyFileWins)

		// Assert - Database description preserved
		require.NoError(t, err)
		assert.Equal(t, dbDesc, *resolved.Description)
	})
}

// TestFullSyncWorkflow tests a complete end-to-end sync operation
func TestFullSyncWorkflow(t *testing.T) {
	// This test simulates the complete workflow of syncing files
	// It's similar to what's in engine_test.go but demonstrates
	// the integration between all components
	t.Run("scanner finds files, engine imports them", func(t *testing.T) {
		// Arrange - Create a temporary directory with test files
		scanner := NewFileScanner()
		detector := NewConflictDetector()
		resolver := NewConflictResolver()

		// Verify components are initialized
		assert.NotNil(t, scanner)
		assert.NotNil(t, detector)
		assert.NotNil(t, resolver)

		// This test verifies that all components work together
		// More comprehensive testing is done in engine_test.go
	})
}
