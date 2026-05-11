package template

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// epicKeyPattern matches an epic key segment like E07 or E12.
var epicKeyPattern = regexp.MustCompile(`^E\d+$`)

// featureKeyPattern matches a feature key segment like E07-F01 or E12-F03.
var featureKeyPattern = regexp.MustCompile(`^E\d+-F\d+$`)

// ParseEpicKeyFromEntityKey extracts the epic key (E##) from a task or feature key.
// E.g., "T-E07-F01-001" -> "E07", "E07-F01" -> "E07", "E07" -> "E07"
// Returns empty string if no epic key can be extracted.
func ParseEpicKeyFromEntityKey(entityKey string) string {
	return parseEpicKeyFromEntityKey(entityKey)
}

func parseEpicKeyFromEntityKey(entityKey string) string {
	if entityKey == "" {
		return ""
	}

	// Strip "T-" prefix if present
	key := entityKey
	if strings.HasPrefix(strings.ToUpper(key), "T-") {
		key = key[2:]
	}

	// Split on "-" and find the first segment matching E\d+
	parts := strings.Split(key, "-")
	for _, part := range parts {
		upper := strings.ToUpper(part)
		if epicKeyPattern.MatchString(upper) {
			return upper
		}
	}

	return ""
}

// ParseFeatureKeyFromTaskKey extracts the feature key (E##-F##) from a task key.
// E.g., "T-E07-F01-001" -> "E07-F01", "E07-F01-001" -> "E07-F01"
// Returns empty string if no feature key can be extracted.
func ParseFeatureKeyFromTaskKey(taskKey string) string {
	return parseFeatureKeyFromTaskKey(taskKey)
}

func parseFeatureKeyFromTaskKey(taskKey string) string {
	if taskKey == "" {
		return ""
	}

	// Strip "T-" prefix if present
	key := taskKey
	if strings.HasPrefix(strings.ToUpper(key), "T-") {
		key = key[2:]
	}

	// Split on "-" and look for E##-F## pattern in the first segments
	parts := strings.Split(key, "-")
	if len(parts) < 2 {
		return ""
	}

	// Try combining first two parts to form E##-F##
	candidate := strings.ToUpper(parts[0]) + "-" + strings.ToUpper(parts[1])
	if featureKeyPattern.MatchString(candidate) {
		return candidate
	}

	return ""
}

// deriveReviewBase computes the review-base directory for an entity from its
// file_path, matching the convention documented in the code-review / QA /
// UAT process partials: replace the leading "docs/plan/" with "docs/review/",
// drop the entity's filename, and strip a trailing "tasks/" segment if
// present (so a task's review base lives alongside its feature's, not under
// a per-task tasks/ subdirectory).
//
// Examples:
//
//	docs/plan/E19-sprint/E19-F04-analytics/tasks/T-E19-F04-001.md
//	  → docs/review/E19-sprint/E19-F04-analytics/
//	docs/plan/E19-sprint/E19-F04-analytics/feature.md
//	  → docs/review/E19-sprint/E19-F04-analytics/
//	docs/plan/bugs/B025.md
//	  → docs/review/bugs/
//
// Returns the empty string if filePath is empty so callers can decide
// whether to omit the placeholder entirely. The result always ends with a
// trailing slash when non-empty, matching the partials' usage as a
// path prefix (e.g. <review-base>/code_review/...).
func deriveReviewBase(filePath string) string {
	if filePath == "" {
		return ""
	}

	p := strings.TrimPrefix(filePath, "./")
	// Rewrite the plan-tree root to the review-tree root. We only rewrite the
	// leading occurrence — paths that don't live under docs/plan/ pass through
	// unchanged so callers see the original prefix in the result (useful for
	// custom layouts and for surfacing misconfiguration).
	if strings.HasPrefix(p, "docs/plan/") {
		p = "docs/review/" + strings.TrimPrefix(p, "docs/plan/")
	}

	// Drop the filename.
	if idx := strings.LastIndex(p, "/"); idx >= 0 {
		p = p[:idx+1]
	}

	// Collapse a trailing "tasks/" segment so task reviews live in the
	// feature's review directory (the partials' convention).
	if strings.HasSuffix(p, "/tasks/") {
		p = strings.TrimSuffix(p, "tasks/")
	}

	return p
}

// applySizePlaceholders populates the "size" and "size_label" keys in the
// placeholder map from the entity's Size field.
//
// When size is non-nil, "size" is set to the decimal representation of the
// integer (e.g., "5") and "size_label" is set to the canonical t-shirt label
// (e.g., "L") via models.SizeLabel. When size is nil, both keys are explicitly
// set to the empty string so that templates render cleanly without producing
// "<nil>" output.
//
// This helper is called from every per-entity placeholder builder
// (TaskPlaceholders, FeaturePlaceholders, EpicPlaceholders, BugPlaceholders,
// ChangeCardPlaceholders) to keep the size-placeholder logic in one place
// (REQ-F-011, REQ-F-012, Decision D6 — complexity_tier is independent).
//
// Part of Epic E07-F42.
func applySizePlaceholders(size *int, placeholders map[string]string) {
	if size != nil {
		placeholders["size"] = strconv.Itoa(*size)
		if label, err := models.SizeLabel(*size); err == nil {
			placeholders["size_label"] = label
		} else {
			// Defensive: SizeLabel only fails for non-canonical values.
			// Populate with empty string to avoid template noise.
			placeholders["size_label"] = ""
		}
	} else {
		placeholders["size"] = ""
		placeholders["size_label"] = ""
	}
}

// EntityPlaceholders creates a map of template placeholders from any Entity
// interface value. It extracts the shared fields common to all entity types:
// id, key, entity_type, title, status, created_at, updated_at, and optionally
// slug, description, and file_path.
//
// Entity-specific placeholder functions (TaskPlaceholders, FeaturePlaceholders,
// etc.) call this base function first, then add their unique fields.
//
// Returns an empty map if entity is nil.
func EntityPlaceholders(entity models.Entity) map[string]string {
	if entity == nil {
		return make(map[string]string)
	}
	m := map[string]string{
		"id":          entity.GetKey(),
		"key":         entity.GetKey(),
		"entity_type": string(entity.GetEntityType()),
		"title":       entity.GetTitle(),
		"status":      entity.GetStatus(),
		"created_at":  entity.GetCreatedAt().Format(time.RFC3339),
		"updated_at":  entity.GetUpdatedAt().Format(time.RFC3339),
	}

	// Optional shared fields via Entity interface
	if slug := entity.GetSlug(); slug != "" {
		m["slug"] = slug
	}
	if desc := entity.GetDescription(); desc != "" {
		m["description"] = desc
	}
	if fp := entity.GetFilePath(); fp != "" {
		m["file_path"] = fp
		if rb := deriveReviewBase(fp); rb != "" {
			m["review_base"] = rb
		}
	}

	return m
}

// TaskPlaceholders creates a map of template placeholders from a Task.
// Returns a map suitable for use with PopulateTemplate.
// Returns an empty map if task is nil.
func TaskPlaceholders(task *models.Task) map[string]string {
	if task == nil {
		return make(map[string]string)
	}
	m := EntityPlaceholders(task)

	// Task-specific key aliases
	epicKey := parseEpicKeyFromEntityKey(task.Key)
	featureKey := parseFeatureKeyFromTaskKey(task.Key)
	m["task_key"] = task.Key
	m["epic_key"] = epicKey
	m["feature_key"] = featureKey
	// Backward-compatible aliases (deprecated, prefer canonical names above)
	m["task_id"] = task.Key
	m["epic_id"] = epicKey
	m["feature_id"] = featureKey

	// Task-specific fields
	m["priority"] = fmt.Sprintf("%d", task.Priority)

	// Optional task-specific pointer fields
	if task.AgentType != nil {
		m["agent_type"] = *task.AgentType
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

	// Size placeholders (REQ-F-011): independent of complexity_tier (REQ-F-012 / D6).
	applySizePlaceholders(task.Size, m)

	return m
}

// FeaturePlaceholders creates a map of template placeholders from a Feature.
// Returns a map suitable for use with PopulateTemplate.
// Returns an empty map if feature is nil.
func FeaturePlaceholders(feature *models.Feature) map[string]string {
	if feature == nil {
		return make(map[string]string)
	}
	m := EntityPlaceholders(feature)

	// Feature-specific key aliases
	epicKey := parseEpicKeyFromEntityKey(feature.Key)
	m["epic_key"] = epicKey
	// Backward-compatible aliases (deprecated)
	m["feature_id"] = feature.Key
	m["epic_id"] = epicKey

	// Optional feature-specific pointer fields
	if feature.ExecutionOrder != nil {
		m["execution_order"] = fmt.Sprintf("%d", *feature.ExecutionOrder)
	}

	// Size placeholders (REQ-F-011): independent of complexity_tier (REQ-F-012 / D6).
	applySizePlaceholders(feature.Size, m)

	return m
}

// EpicPlaceholders creates a map of template placeholders from an Epic.
// Returns a map suitable for use with PopulateTemplate.
// Returns an empty map if epic is nil.
func EpicPlaceholders(epic *models.Epic) map[string]string {
	if epic == nil {
		return make(map[string]string)
	}
	m := EntityPlaceholders(epic)

	// Epic-specific key aliases
	// Backward-compatible alias (deprecated)
	m["epic_id"] = epic.Key

	// Epic-specific fields
	m["priority"] = string(epic.Priority)

	// Optional epic-specific pointer fields
	if epic.BusinessValue != nil {
		m["business_value"] = string(*epic.BusinessValue)
	}

	// Size placeholders (REQ-F-011): independent of complexity_tier (REQ-F-012 / D6).
	applySizePlaceholders(epic.Size, m)

	return m
}

// BugPlaceholders creates a map of template placeholders from a Bug.
func BugPlaceholders(bug *models.Bug) map[string]string {
	if bug == nil {
		return make(map[string]string)
	}
	m := EntityPlaceholders(bug)

	// Bug-specific fields
	m["severity"] = string(bug.Severity)

	// Optional bug-specific pointer fields
	if bug.LinkedEntityType != nil {
		m["linked_entity_type"] = *bug.LinkedEntityType
	}
	if bug.LinkedEntityKey != nil {
		m["linked_entity_key"] = *bug.LinkedEntityKey
	}

	// Size placeholders (REQ-F-011): independent of complexity_tier (REQ-F-012 / D6).
	applySizePlaceholders(bug.Size, m)

	return m
}

// TechDebtPlaceholders creates a map of template placeholders from a TechDebt.
// Returns a map suitable for use with PopulateTemplate.
// Returns an empty map if techDebt is nil.
func TechDebtPlaceholders(td *models.TechDebt) map[string]string {
	if td == nil {
		return make(map[string]string)
	}
	m := EntityPlaceholders(td)

	// TechDebt-specific fields
	m["category"] = string(td.Category)
	m["severity"] = string(td.Severity)

	// Optional tech-debt-specific pointer fields
	if td.EffortEstimate != nil {
		m["effort_estimate"] = *td.EffortEstimate
	}

	return m
}

// ChangeCardPlaceholders creates a map of template placeholders from a ChangeCard.
func ChangeCardPlaceholders(card *models.ChangeCard) map[string]string {
	if card == nil {
		return make(map[string]string)
	}
	m := EntityPlaceholders(card)

	// ChangeCard-specific fields
	m["priority"] = fmt.Sprintf("%d", card.Priority)

	// Optional change-card-specific pointer fields
	if card.RequestedBy != nil {
		m["requested_by"] = *card.RequestedBy
	}
	if card.AssignedTo != nil {
		m["assigned_to"] = *card.AssignedTo
	}
	if card.Justification != nil {
		m["justification"] = *card.Justification
	}
	if card.ImpactAnalysis != nil {
		m["impact_analysis"] = *card.ImpactAnalysis
	}
	if card.RollbackPlan != nil {
		m["rollback_plan"] = *card.RollbackPlan
	}

	// Size placeholders (REQ-F-011): independent of complexity_tier (REQ-F-012 / D6).
	applySizePlaceholders(card.Size, m)

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

// FeaturePlaceholdersWithRelated extends FeaturePlaceholders with relationship data.
// It adds two new placeholders:
// - related_docs: comma-separated file paths of documents linked to the feature
// - related_features: comma-separated keys of related features from entity_relationships table
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
	docRepoForFeature interface {
		ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error)
	},
	featureRelRepo FeatureRelationshipRepository,
	enrichment *TemplateEnrichmentData,
) map[string]string {
	// Start with basic placeholders
	placeholders := FeaturePlaceholders(feature)

	// If feature is nil, return empty placeholders
	if feature == nil {
		return placeholders
	}

	// Add related_docs
	if docRepoForFeature != nil {
		docs, err := docRepoForFeature.ListForFeature(ctx, feature.ID)
		if err != nil {
			slog.Warn("Failed to fetch related docs for feature", "feature", feature.Key, "error", err)
			placeholders["related_docs"] = ""
		} else {
			placeholders["related_docs"] = formatDocPathsAsCSV(docs)
		}
	} else {
		placeholders["related_docs"] = ""
	}

	// Add related_features from relationship table
	if featureRelRepo != nil {
		relatedKeys, err := featureRelRepo.ListRelatedFeatures(ctx, feature.ID)
		if err != nil {
			slog.Warn("Failed to fetch related features", "feature", feature.Key, "error", err)
			placeholders["related_features"] = ""
		} else {
			placeholders["related_features"] = strings.Join(relatedKeys, ",")
		}
	} else {
		placeholders["related_features"] = ""
	}

	// Extract ALL metadata and structured fields from ContextData
	extractContextDataFields(feature.ContextData, placeholders)

	// Also extract complexity_tier from entity Metadata for backward compatibility
	if feature.Metadata != nil {
		if tier, ok := feature.Metadata["complexity_tier"].(string); ok {
			if _, exists := placeholders["complexity_tier"]; !exists {
				placeholders["complexity_tier"] = tier
			}
		}
	}

	// Ensure complexity_tier has a value (empty string if not set anywhere)
	if _, exists := placeholders["complexity_tier"]; !exists {
		placeholders["complexity_tier"] = ""
	}

	// Apply enrichment data (nil-safe)
	ApplyEnrichmentData(enrichment, placeholders)

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

// TaskRelationshipRepository interface for accessing task-to-task relationships.
// Implementations must support querying related tasks by task ID.
type TaskRelationshipRepository interface {
	ListRelatedTaskKeys(ctx context.Context, taskID int64) ([]string, error)
}

// TaskPlaceholdersWithRelated extends TaskPlaceholders with relationship data.
// It adds two new placeholders:
// - related_docs: comma-separated file paths of documents linked to the task
// - related_tasks: comma-separated keys from entity_relationships table
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
// - Task with docs, no relationships (partial data)
// - Task with docs, relationship query error (graceful degradation)
// - Nil task pointer (no panic)
// - Document repo query failure (graceful degradation)
//
// REFACTORED: Now uses TaskRelationshipRepository.ListRelatedTaskKeys() instead of
// parsing task.ContextData JSON. This aligns with the proven repository pattern
// used for features and epics (T-E07-F29-029).
func TaskPlaceholdersWithRelated(
	ctx context.Context,
	task *models.Task,
	docRepo DocumentRepository,
	taskRelRepo TaskRelationshipRepository,
	enrichment *TemplateEnrichmentData,
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
		slog.Warn("Failed to fetch related docs for task", "task", task.Key, "error", err)
		placeholders["related_docs"] = ""
	} else {
		placeholders["related_docs"] = formatDocPathsAsCSV(docs)
	}

	// Add related_tasks from entity_relationships table via TaskRelationshipRepository adapter
	relatedKeys, err := taskRelRepo.ListRelatedTaskKeys(ctx, task.ID)
	if err != nil {
		slog.Warn("Failed to fetch related tasks", "task", task.Key, "error", err)
		placeholders["related_tasks"] = ""
	} else {
		placeholders["related_tasks"] = strings.Join(relatedKeys, ",")
	}

	// Extract ALL metadata and structured fields from ContextData
	extractContextDataFields(task.ContextData, placeholders)

	// Also extract complexity_tier from entity Metadata for backward compatibility
	if task.Metadata != nil {
		if tier, ok := task.Metadata["complexity_tier"].(string); ok {
			if _, exists := placeholders["complexity_tier"]; !exists {
				placeholders["complexity_tier"] = tier
			}
		}
	}

	// Ensure complexity_tier has a value (empty string if not set anywhere)
	if _, exists := placeholders["complexity_tier"]; !exists {
		placeholders["complexity_tier"] = ""
	}

	// Apply enrichment data (nil-safe)
	ApplyEnrichmentData(enrichment, placeholders)

	return placeholders
}

// EpicPlaceholdersWithRelated extends EpicPlaceholders with relationship data.
// It adds two new placeholders:
// - related_docs: comma-separated file paths of documents linked to the epic
// - related_epics: comma-separated keys of related epics from entity_relationships table
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
	docRepoForEpic interface {
		ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error)
	},
	epicRelRepo EpicRelationshipRepository,
	ctx context.Context,
	enrichment *TemplateEnrichmentData,
) map[string]string {
	// Start with basic placeholders
	placeholders := EpicPlaceholders(epic)

	// If epic is nil, return empty placeholders
	if epic == nil {
		return placeholders
	}

	// Add related_docs
	if docRepoForEpic != nil {
		docs, err := docRepoForEpic.ListForEpic(ctx, epic.ID)
		if err != nil {
			slog.Warn("Failed to fetch related docs for epic", "epic", epic.Key, "error", err)
			placeholders["related_docs"] = ""
		} else {
			placeholders["related_docs"] = formatDocPathsAsCSV(docs)
		}
	} else {
		placeholders["related_docs"] = ""
	}

	// Add related_epics from relationship table
	if epicRelRepo != nil {
		relatedKeys, err := epicRelRepo.ListRelatedEpics(ctx, epic.ID)
		if err != nil {
			slog.Warn("Failed to fetch related epics", "epic", epic.Key, "error", err)
			placeholders["related_epics"] = ""
		} else {
			placeholders["related_epics"] = formatEpicKeysAsCSV(relatedKeys)
		}
	} else {
		placeholders["related_epics"] = ""
	}

	// Extract ALL metadata and structured fields from ContextData
	extractContextDataFields(epic.ContextData, placeholders)

	// Apply enrichment data (nil-safe)
	ApplyEnrichmentData(enrichment, placeholders)

	return placeholders
}

// stringifyMetadataValue converts a metadata value of common types to a string
// representation suitable for template placeholders.
// Supports string, int, int64, float64, and bool types.
// Returns empty string for unsupported types or nil values.
func stringifyMetadataValue(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case float64:
		// JSON numbers are always float64
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return ""
	}
}

// extractContextDataFields parses the context_data JSON string and extracts
// all metadata fields and structured fields into the placeholder map.
// Metadata key-value pairs are flattened using stringifyMetadataValue.
// Structured fields (Progress, OpenQuestions, Blockers, ImplementationDecisions)
// are extracted into well-known placeholder keys.
// Metadata is extracted first; structured fields do NOT overwrite existing keys.
// Gracefully handles nil, empty, or malformed JSON by silently returning.
func extractContextDataFields(contextData *string, placeholders map[string]string) {
	if contextData == nil || *contextData == "" {
		return
	}

	var cd models.ContextData
	if err := json.Unmarshal([]byte(*contextData), &cd); err != nil {
		// Gracefully skip on malformed JSON
		return
	}

	// Step 1: Flatten Metadata key-value pairs (runs first)
	for key, value := range cd.Metadata {
		str := stringifyMetadataValue(value)
		if str != "" {
			placeholders[key] = str
		}
	}

	// Step 2: Extract structured Progress fields (do not overwrite existing keys)
	if cd.Progress != nil {
		if cd.Progress.CurrentStep != nil {
			if _, exists := placeholders["current_step"]; !exists {
				placeholders["current_step"] = *cd.Progress.CurrentStep
			}
		}
		if len(cd.Progress.CompletedSteps) > 0 {
			if _, exists := placeholders["completed_steps"]; !exists {
				placeholders["completed_steps"] = strings.Join(cd.Progress.CompletedSteps, ", ")
			}
		}
		if len(cd.Progress.RemainingSteps) > 0 {
			if _, exists := placeholders["remaining_steps"]; !exists {
				placeholders["remaining_steps"] = strings.Join(cd.Progress.RemainingSteps, ", ")
			}
		}
		if _, exists := placeholders["completed_steps_count"]; !exists {
			placeholders["completed_steps_count"] = fmt.Sprintf("%d", len(cd.Progress.CompletedSteps))
		}
		if _, exists := placeholders["remaining_steps_count"]; !exists {
			placeholders["remaining_steps_count"] = fmt.Sprintf("%d", len(cd.Progress.RemainingSteps))
		}
	}

	// Step 3: Extract open questions
	if len(cd.OpenQuestions) > 0 {
		if _, exists := placeholders["open_questions"]; !exists {
			placeholders["open_questions"] = strings.Join(cd.OpenQuestions, "; ")
		}
		if _, exists := placeholders["open_questions_count"]; !exists {
			placeholders["open_questions_count"] = fmt.Sprintf("%d", len(cd.OpenQuestions))
		}
	}

	// Step 4: Extract blockers summary
	if len(cd.Blockers) > 0 {
		if _, exists := placeholders["blockers_count"]; !exists {
			placeholders["blockers_count"] = fmt.Sprintf("%d", len(cd.Blockers))
		}
		if _, exists := placeholders["latest_blocker"]; !exists {
			placeholders["latest_blocker"] = cd.Blockers[len(cd.Blockers)-1].Description
		}
	}

	// Step 5: Extract implementation decisions count
	if len(cd.ImplementationDecisions) > 0 {
		if _, exists := placeholders["decisions_count"]; !exists {
			placeholders["decisions_count"] = fmt.Sprintf("%d", len(cd.ImplementationDecisions))
		}
	}
}

// TemplateEnrichmentData contains pre-fetched enrichment data for template rendering.
// All fields are optional (zero-value means "not fetched" or "not applicable").
// This struct is constructed by the service layer and passed to *PlaceholdersWithRelated().
type TemplateEnrichmentData struct {
	// Previous status from task_history (tasks only in v1)
	PreviousStatus string

	// Parent entity titles for hierarchical context
	ParentTitle      string // feature.title for tasks, epic.title for features
	GrandparentTitle string // epic.title for tasks (empty for features/epics)

	// Latest note from entity_notes
	LatestNoteContent string
	LatestNoteType    string

	// Note counts
	NotesCount     int
	RejectionCount int

	// Sibling progress (children of the same parent)
	SiblingTotal     int
	SiblingCompleted int
	SiblingBlocked   int
}

// TemplateEnrichmentRepository provides consolidated enrichment data
// for template variable population. Implementations should fetch all
// data in a single query to minimize Turso round-trips.
type TemplateEnrichmentRepository interface {
	GetTaskEnrichment(ctx context.Context, taskID int64) (*TemplateEnrichmentData, error)
	GetFeatureEnrichment(ctx context.Context, featureID int64) (*TemplateEnrichmentData, error)
	GetEpicEnrichment(ctx context.Context, epicID int64) (*TemplateEnrichmentData, error)
}

// ApplyEnrichmentData merges enrichment data into the placeholder map.
// If enrichment is nil, this is a no-op (nil-safe).
// This is exported for use by services that need to apply enrichment data
// to basic placeholders (when *PlaceholdersWithRelated is not used).
func ApplyEnrichmentData(enrichment *TemplateEnrichmentData, placeholders map[string]string) {
	if enrichment == nil {
		return
	}

	if enrichment.PreviousStatus != "" {
		placeholders["previous_status"] = enrichment.PreviousStatus
	}
	if enrichment.ParentTitle != "" {
		placeholders["parent_title"] = enrichment.ParentTitle
	}
	if enrichment.GrandparentTitle != "" {
		placeholders["grandparent_title"] = enrichment.GrandparentTitle
	}
	if enrichment.LatestNoteContent != "" {
		placeholders["latest_note"] = enrichment.LatestNoteContent
		placeholders["latest_note_type"] = enrichment.LatestNoteType
	}

	placeholders["notes_count"] = fmt.Sprintf("%d", enrichment.NotesCount)
	placeholders["rejection_count"] = fmt.Sprintf("%d", enrichment.RejectionCount)
	placeholders["sibling_total"] = fmt.Sprintf("%d", enrichment.SiblingTotal)
	placeholders["sibling_completed"] = fmt.Sprintf("%d", enrichment.SiblingCompleted)
	placeholders["sibling_blocked"] = fmt.Sprintf("%d", enrichment.SiblingBlocked)
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
