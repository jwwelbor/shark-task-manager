package sync

import (
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ConflictResolver applies resolution strategies to conflicts
type ConflictResolver struct {
	// No state needed
}

// NewConflictResolver creates a new ConflictResolver
func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{}
}

// ResolveConflicts applies the resolution strategy and returns a resolved task model
// Returns a copy of the database task with conflicts resolved according to strategy
//
// Strategy behavior:
// - file-wins: Use file value for all conflicting fields
// - database-wins: Keep database value for all conflicting fields
// - newer-wins: Compare file.ModifiedAt with db.UpdatedAt, use newer source
// - manual: Prompt user interactively for each conflict
//
// Database-only fields are ALWAYS preserved from the database:
// - status, priority, agent_type, depends_on, assigned_agent, timestamps
func (r *ConflictResolver) ResolveConflicts(
	conflicts []Conflict,
	fileData *TaskMetadata,
	dbTask *models.Task,
	strategy ConflictStrategy,
) (*models.Task, error) {
	// Handle manual strategy separately
	if strategy == ConflictStrategyManual {
		manualResolver := NewManualResolver()
		return manualResolver.ResolveConflictsManually(conflicts, fileData, dbTask)
	}

	// Create a copy of the database task to avoid modifying the original
	resolved := r.copyTask(dbTask)

	// If no conflicts, return copy of database task
	if len(conflicts) == 0 {
		return resolved, nil
	}

	// Determine which source to use based on strategy
	useFileValues := false
	switch strategy {
	case ConflictStrategyFileWins:
		useFileValues = true
	case ConflictStrategyDatabaseWins:
		useFileValues = false
	case ConflictStrategyNewerWins:
		// Compare timestamps: file.ModifiedAt vs db.UpdatedAt
		// Use file if file is newer
		useFileValues = fileData.ModifiedAt.After(dbTask.UpdatedAt)
	}

	// Apply resolution to each conflict
	if useFileValues {
		// Apply file values to conflicting fields
		for _, conflict := range conflicts {
			switch conflict.Field {
			case "title":
				resolved.Title = fileData.Title
			case "description":
				if fileData.Description != nil {
					// Copy the string value
					desc := *fileData.Description
					resolved.Description = &desc
				}
				// If fileData.Description is nil, keep database value
			case "file_path":
				path := fileData.FilePath
				resolved.FilePath = &path
			}
		}
	} else {
		// Keep database values (already in resolved copy)
		// No action needed since we copied the database task
	}

	return resolved, nil
}

// copyTask creates a deep copy of a task model
func (r *ConflictResolver) copyTask(task *models.Task) *models.Task {
	copy := &models.Task{
		ID:          task.ID,
		FeatureID:   task.FeatureID,
		Key:         task.Key,
		Title:       task.Title,
		Status:      task.Status,
		Priority:    task.Priority,
		CreatedAt:   task.CreatedAt,
		StartedAt:   task.StartedAt,
		CompletedAt: task.CompletedAt,
		BlockedAt:   task.BlockedAt,
		UpdatedAt:   task.UpdatedAt,
	}

	// Copy pointer fields
	if task.Description != nil {
		desc := *task.Description
		copy.Description = &desc
	}

	if task.AgentType != nil {
		agentType := *task.AgentType
		copy.AgentType = &agentType
	}

	if task.DependsOn != nil {
		dependsOn := *task.DependsOn
		copy.DependsOn = &dependsOn
	}

	if task.AssignedAgent != nil {
		assignedAgent := *task.AssignedAgent
		copy.AssignedAgent = &assignedAgent
	}

	if task.FilePath != nil {
		filePath := *task.FilePath
		copy.FilePath = &filePath
	}

	if task.BlockedReason != nil {
		blockedReason := *task.BlockedReason
		copy.BlockedReason = &blockedReason
	}

	return copy
}
