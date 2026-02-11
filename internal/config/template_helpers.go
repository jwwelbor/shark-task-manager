package config

import (
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskPlaceholders creates a map of template placeholders from a Task.
// Returns a map suitable for use with PopulateTemplate.
func TaskPlaceholders(task *models.Task) map[string]string {
	m := map[string]string{
		"id":         task.Key,
		"task_id":    task.Key,
		"epic_id":    task.Key,
		"feature_id": task.Key,
		"title":      task.Title,
		"status":     string(task.Status),
		"priority":   fmt.Sprintf("%d", task.Priority),
		"created_at": task.CreatedAt.Format(time.RFC3339),
		"updated_at": task.UpdatedAt.Format(time.RFC3339),
	}

	// Optional pointer fields
	if task.Slug != nil {
		m["slug"] = *task.Slug
	}
	if task.FilePath != nil {
		m["file_path"] = *task.FilePath
	}
	if task.AgentType != nil {
		m["agent_type"] = *task.AgentType
	}
	if task.Description != nil {
		m["description"] = *task.Description
	}
	if task.ExecutionOrder != nil {
		m["execution_order"] = fmt.Sprintf("%d", *task.ExecutionOrder)
	}
	if task.BlockedReason != nil {
		m["blocked_reason"] = *task.BlockedReason
	}
	if task.DependsOn != nil {
		m["depends_on"] = *task.DependsOn
	}
	if task.CompletionNotes != nil {
		m["completion_notes"] = *task.CompletionNotes
	}
	if task.FilesChanged != nil {
		m["files_changed"] = *task.FilesChanged
	}

	return m
}

// FeaturePlaceholders creates a map of template placeholders from a Feature.
// Returns a map suitable for use with PopulateTemplate.
func FeaturePlaceholders(feature *models.Feature) map[string]string {
	m := map[string]string{
		"id":         feature.Key,
		"feature_id": feature.Key,
		"title":      feature.Title,
		"status":     string(feature.Status),
		"created_at": feature.CreatedAt.Format(time.RFC3339),
		"updated_at": feature.UpdatedAt.Format(time.RFC3339),
	}

	// Optional pointer fields
	if feature.Slug != nil {
		m["slug"] = *feature.Slug
	}
	if feature.Description != nil {
		m["description"] = *feature.Description
	}
	if feature.FilePath != nil {
		m["file_path"] = *feature.FilePath
	}
	if feature.ExecutionOrder != nil {
		m["execution_order"] = fmt.Sprintf("%d", *feature.ExecutionOrder)
	}

	return m
}

// EpicPlaceholders creates a map of template placeholders from an Epic.
// Returns a map suitable for use with PopulateTemplate.
func EpicPlaceholders(epic *models.Epic) map[string]string {
	m := map[string]string{
		"id":         epic.Key,
		"epic_id":    epic.Key,
		"title":      epic.Title,
		"status":     string(epic.Status),
		"priority":   string(epic.Priority),
		"created_at": epic.CreatedAt.Format(time.RFC3339),
		"updated_at": epic.UpdatedAt.Format(time.RFC3339),
	}

	// Optional pointer fields
	if epic.Slug != nil {
		m["slug"] = *epic.Slug
	}
	if epic.Description != nil {
		m["description"] = *epic.Description
	}
	if epic.FilePath != nil {
		m["file_path"] = *epic.FilePath
	}
	if epic.BusinessValue != nil {
		m["business_value"] = string(*epic.BusinessValue)
	}

	return m
}
