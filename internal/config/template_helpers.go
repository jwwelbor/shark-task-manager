package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskPlaceholders creates a map of template placeholders from a Task.
// Returns a map suitable for use with PopulateTemplate.
// Returns an empty map if task is nil.
func TaskPlaceholders(task *models.Task) map[string]string {
	if task == nil {
		return make(map[string]string)
	}
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
// Returns an empty map if feature is nil.
func FeaturePlaceholders(feature *models.Feature) map[string]string {
	if feature == nil {
		return make(map[string]string)
	}
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
// Returns an empty map if epic is nil.
func EpicPlaceholders(epic *models.Epic) map[string]string {
	if epic == nil {
		return make(map[string]string)
	}
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

// formatDocPathsAsCSV formats a slice of documents as comma-separated file paths.
// Returns an empty string if the slice is nil or empty.
// Handles documents with spaces in paths without escaping.
//
// Test cases (from test plan TC-FMT-01 through TC-FMT-06):
// - nil slice: returns ""
// - empty slice: returns ""
// - single doc: returns "docs/a.md"
// - three docs: returns "docs/a.md,docs/b.md,docs/c.md"
// - docs with spaces: returns "docs/My Doc.md,docs/Other.md"
// - 50+ docs: includes all paths without truncation
func formatDocPathsAsCSV(docs []*models.Document) string {
	if len(docs) == 0 {
		return ""
	}

	paths := make([]string, 0, len(docs))
	for _, doc := range docs {
		if doc != nil && doc.FilePath != "" {
			paths = append(paths, doc.FilePath)
		}
	}

	return strings.Join(paths, ",")
}

// extractRelatedTasksFromContext parses RelatedTasks from context_data JSON.
// Returns a comma-separated CSV string of task keys.
// Returns empty string if contextData is nil, empty, or contains no related_tasks.
// Gracefully handles malformed JSON by logging a warning and returning empty string.
//
// Test cases (from test plan TC-CTX-01 through TC-CTX-07):
// - nil context: returns ""
// - empty string: returns ""
// - valid JSON with 2 tasks: returns "E01-F01,E02-F01"
// - valid JSON with empty array: returns ""
// - valid JSON without related_tasks field: returns ""
// - malformed JSON: returns "" (warning logged)
// - valid JSON with null related_tasks: returns ""
func extractRelatedTasksFromContext(contextData *string) string {
	if contextData == nil || *contextData == "" {
		return ""
	}

	// Parse JSON without validation for graceful degradation
	var contextObj models.ContextData
	if err := json.Unmarshal([]byte(*contextData), &contextObj); err != nil {
		log.Printf("WARNING: Failed to parse context_data JSON (returning empty): %v", err)
		return ""
	}

	if len(contextObj.RelatedTasks) == 0 {
		return ""
	}

	return strings.Join(contextObj.RelatedTasks, ",")
}

// FeaturePlaceholdersWithRelated extends FeaturePlaceholders with relationship data.
// It adds two new placeholders:
// - related_docs: comma-separated file paths of documents linked to the feature
// - related_features: comma-separated keys of related features from feature_relationships table
//
// Accepts context and repository dependencies for database queries.
// Gracefully degrades on errors: returns empty strings for missing relationships
// rather than propagating errors (required for template population reliability).
//
// Implements AC-4a.1 through AC-4a.6 from test plan Story 4a.
//
// Test cases (from test plan TC-FPH-01 through TC-FPH-05):
// - Feature with 2 docs and 3 related features (happy path)
// - Feature with 0 docs and 0 related features (empty data)
// - Nil feature pointer (no panic)
// - Document repo query failure (graceful degradation)
// - Cross-epic feature relationships (supported)
func FeaturePlaceholdersWithRelated(
	ctx context.Context,
	feature *models.Feature,
	docRepo DocumentRepository,
	featureRelRepo FeatureRelationshipRepository,
) map[string]string {
	// Start with basic placeholders
	placeholders := FeaturePlaceholders(feature)

	// If feature is nil, return empty placeholders
	if feature == nil {
		return placeholders
	}

	// Add related_docs
	docs, err := docRepo.ListForFeature(ctx, feature.ID)
	if err != nil {
		log.Printf("WARNING: Failed to fetch related docs for feature %s: %v", feature.Key, err)
		placeholders["related_docs"] = ""
	} else {
		placeholders["related_docs"] = formatDocPathsAsCSV(docs)
	}

	// Add related_features from relationship table
	relatedKeys, err := featureRelRepo.ListRelatedFeatures(ctx, feature.ID)
	if err != nil {
		log.Printf("WARNING: Failed to fetch related features for %s: %v", feature.Key, err)
		placeholders["related_features"] = ""
	} else {
		placeholders["related_features"] = strings.Join(relatedKeys, ",")
	}

	return placeholders
}

// DocumentRepository interface for accessing documents linked to entities.
// Implementations must support querying documents by feature, task, or epic ID.
type DocumentRepository interface {
	ListForTask(ctx context.Context, taskID int64) ([]*models.Document, error)
	ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error)
	ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error)
}

// FeatureRelationshipRepository interface for accessing feature-to-feature relationships.
// Implementations must support querying related features by feature ID.
type FeatureRelationshipRepository interface {
	ListRelatedFeatures(ctx context.Context, featureID int64) ([]string, error)
}

// EpicRelationshipRepository interface for accessing epic-to-epic relationships.
// Implementations must support querying related epics by epic ID.
type EpicRelationshipRepository interface {
	ListRelatedEpics(ctx context.Context, epicID int64) ([]string, error)
}

// TaskPlaceholdersWithRelated extends TaskPlaceholders with relationship data.
// It adds two new placeholders:
// - related_docs: comma-separated file paths of documents linked to the task
// - related_tasks: comma-separated keys from task.ContextData.RelatedTasks
//
// Accepts context and repository dependencies for database queries.
// Gracefully degrades on errors: returns empty strings for missing relationships
// rather than propagating errors (required for template population reliability).
//
// Implements AC-1.1 through AC-1.4 from test plan Story 1.
//
// Test cases (from test plan TC-PH-01 through TC-PH-06):
// - Task with 2 docs and 2 related tasks (happy path)
// - Task with 0 docs and 0 related tasks (empty data)
// - Task with docs, nil context_data (partial data)
// - Task with docs, malformed JSON context (JSON error handling)
// - Nil task pointer (no panic)
// - Document repo query failure (graceful degradation)
func TaskPlaceholdersWithRelated(
	task *models.Task,
	docRepo DocumentRepository,
	ctx context.Context,
) map[string]string {
	// Start with basic placeholders
	placeholders := TaskPlaceholders(task)

	// If task is nil, return empty placeholders
	if task == nil {
		return placeholders
	}

	// Add related_docs
	docs, err := docRepo.ListForTask(ctx, task.ID)
	if err != nil {
		log.Printf("WARNING: Failed to fetch related docs for task %s: %v", task.Key, err)
		placeholders["related_docs"] = ""
	} else {
		placeholders["related_docs"] = formatDocPathsAsCSV(docs)
	}

	// Add related_tasks from context data
	placeholders["related_tasks"] = extractRelatedTasksFromContext(task.ContextData)

	return placeholders
}

// EpicPlaceholdersWithRelated extends EpicPlaceholders with relationship data.
// It adds two new placeholders:
// - related_docs: comma-separated file paths of documents linked to the epic
// - related_epics: comma-separated keys of related epics from epic_relationships table
//
// Accepts context and repository dependencies for database queries.
// Gracefully degrades on errors: returns empty strings for missing relationships
// rather than propagating errors (required for template population reliability).
//
// Implements AC-4b.1 through AC-4b.5 from test plan Story 4b.
//
// Test cases (from test plan TC-EPH-01 through TC-EPH-04):
// - Epic with 2 docs and 2 related epics (happy path)
// - Epic with 0 docs and 0 related epics (empty data)
// - Nil epic pointer (no panic)
// - Epic repo query failure (graceful degradation)
func EpicPlaceholdersWithRelated(
	epic *models.Epic,
	docRepo DocumentRepository,
	epicRelRepo EpicRelationshipRepository,
	ctx context.Context,
) map[string]string {
	// Start with basic placeholders
	placeholders := EpicPlaceholders(epic)

	// If epic is nil, return empty placeholders
	if epic == nil {
		return placeholders
	}

	// Add related_docs
	docs, err := docRepo.ListForEpic(ctx, epic.ID)
	if err != nil {
		log.Printf("WARNING: Failed to fetch related docs for epic %s: %v", epic.Key, err)
		placeholders["related_docs"] = ""
	} else {
		placeholders["related_docs"] = formatDocPathsAsCSV(docs)
	}

	// Add related_epics from relationship table
	relatedKeys, err := epicRelRepo.ListRelatedEpics(ctx, epic.ID)
	if err != nil {
		log.Printf("WARNING: Failed to fetch related epics for %s: %v", epic.Key, err)
		placeholders["related_epics"] = ""
	} else {
		placeholders["related_epics"] = formatEpicKeysAsCSV(relatedKeys)
	}

	return placeholders
}

// formatFeatureKeysAsCSV formats a slice of feature keys as comma-separated values.
// Returns an empty string if the slice is nil or empty.
//
// Test cases (from test plan TC-FKCSV-01 through TC-FKCSV-05):
// - nil slice: returns ""
// - empty slice: returns ""
// - single key: returns "E07-F05"
// - three keys: returns "E07-F05,E07-F21,E10-F05"
// - cross-epic keys: returns "E01-F01,E07-F05,E10-F05" (maintains order)
func formatFeatureKeysAsCSV(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	return strings.Join(keys, ",")
}

// formatEpicKeysAsCSV formats a slice of epic keys as comma-separated values.
// Returns an empty string if the slice is nil or empty.
//
// Test cases (from test plan TC-EKCSV-01 through TC-EKCSV-04):
// - nil slice: returns ""
// - empty slice: returns ""
// - single key: returns "E01"
// - three keys: returns "E01,E05,E07"
func formatEpicKeysAsCSV(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	return strings.Join(keys, ",")
}
