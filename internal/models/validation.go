package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Validation errors
var (
	ErrInvalidEpicKey                    = errors.New("invalid epic key format: must match ^E\\d{2}$")
	ErrInvalidFeatureKey                 = errors.New("invalid feature key format: must match ^E\\d{2}-F\\d{2}$")
	ErrInvalidTaskKey                    = errors.New("invalid task key format: must match ^T-E\\d{2}-F\\d{2}-\\d{3}$")
	ErrInvalidSprintKey                  = errors.New("invalid sprint key format: must match ^S\\d{3}$ (e.g., S001, S024, S999)")
	ErrInvalidSprintAssignmentEntityType = errors.New(
		"invalid sprint assignment entity_type: must be task, bug, change_card, tech_debt, epic, or feature")
	ErrInvalidEpicStatus    = errors.New("invalid epic status")
	ErrInvalidFeatureStatus = errors.New("invalid feature status")
	// ErrInvalidTaskStatus is deprecated - error messages are now generated dynamically based on workflow config
	ErrInvalidTaskStatus  = errors.New("invalid task status")
	ErrInvalidAgentType   = errors.New("invalid agent type: cannot be empty or whitespace-only")
	ErrInvalidPriority    = errors.New("invalid priority: must be between 1 and 10")
	ErrInvalidProgressPct = errors.New("invalid progress_pct: must be between 0.0 and 100.0")
	ErrInvalidDependsOn   = errors.New("invalid depends_on: must be a valid JSON array of strings")
	ErrEmptyTitle         = errors.New("title cannot be empty")
	ErrEmptyNewStatus     = errors.New("new_status cannot be empty")
	// ErrInvalidNoteType is the sentinel for an invalid note type. The
	// human-readable list of valid types is appended dynamically by
	// ValidateNoteType from the validNoteTypes slice (single source of
	// truth), so the error message can never drift from the validator.
	ErrInvalidNoteType         = errors.New("invalid note type")
	ErrInvalidTaskID           = errors.New("task_id must be greater than 0")
	ErrEmptyContent            = errors.New("content cannot be empty")
	ErrInvalidRelationshipType = errors.New("invalid relationship type: must be depends_on, blocks, related_to, follows, spawned_from, duplicates, references, linked_to, or question_blocks")
	ErrSelfRelationship        = errors.New("task cannot have a relationship with itself")
	ErrCircularDependency      = errors.New("circular dependency detected")
	ErrInvalidFeatureID        = errors.New("feature_id must be greater than 0")
	ErrInvalidEpicID           = errors.New("epic_id must be greater than 0")
	ErrInvalidSessionOutcome   = errors.New("invalid session outcome: must be completed, paused, or blocked")
	ErrInvalidTimestamp        = errors.New("invalid timestamp: cannot be zero value")
	ErrEmptyKey                = errors.New("key cannot be empty")
	ErrInvalidJSON             = errors.New("invalid JSON format")
	// NOTE: ErrInvalidSize is declared in size.go (E07-F42).
	// Use ValidateSize, ParseSize, and SizeLabel from that file.
	// Pattern mirrors ValidateNoteType / ValidateRelationshipType above.
)

// Key format regex patterns
var (
	epicKeyPattern    = regexp.MustCompile(`^E\d{2}$`)
	featureKeyPattern = regexp.MustCompile(`^E\d{2}-F\d{2}$`)
	taskKeyPattern    = regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}$`)
	sprintKeyPattern  = regexp.MustCompile(`^S\d{3}$`)
)

// ValidateEpicKey validates the epic key format
func ValidateEpicKey(key string) error {
	if !epicKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: got %q", ErrInvalidEpicKey, key)
	}
	return nil
}

// ValidateFeatureKey validates the feature key format
func ValidateFeatureKey(key string) error {
	if !featureKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: got %q", ErrInvalidFeatureKey, key)
	}
	return nil
}

// ValidateTaskKey validates the task key format
func ValidateTaskKey(key string) error {
	if !taskKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: got %q", ErrInvalidTaskKey, key)
	}
	return nil
}

// ValidateSprintKey validates the sprint key format (S### where ### is 3 digits).
//
// The validator receives an already-normalised (uppercase) key — callers are
// responsible for normalising via keys.Normalize() upstream. A lowercase key
// such as "s024" therefore fails here even though it would pass the
// case-insensitive keys.IsSprintKey helper.
func ValidateSprintKey(key string) error {
	if !sprintKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: got %q", ErrInvalidSprintKey, key)
	}
	return nil
}

// ValidateSprintAssignmentEntityType validates the polymorphic entity_type
// column on sprint_assignments. The allowed values are task, bug, change_card,
// tech_debt, epic, feature.
//
// Per the post-B018 convention (see internal/db/db.go:436-444 and the
// `feedback_entity_type_check_constraints` user-feedback memory), there is
// NO matching CHECK constraint on the underlying sprint_assignments table.
// Adding another assignable entity type later requires updating only this
// function — no DB migration is needed.
func ValidateSprintAssignmentEntityType(entityType string) error {
	valid := map[string]bool{
		"task":        true,
		"bug":         true,
		"change_card": true,
		"tech_debt":   true,
		"epic":        true,
		"feature":     true,
	}
	if !valid[entityType] {
		return fmt.Errorf("%w: got %q", ErrInvalidSprintAssignmentEntityType, entityType)
	}
	return nil
}

// ValidateEpicStatus performs basic validation on an epic status string.
// It only checks that the status is non-empty after trimming whitespace.
//
// This function does NOT validate against workflow-defined statuses because
// the models package cannot import the workflow package (circular dependency).
// For workflow-aware validation, callers at the CLI/command layer should use:
//
//	cli.GetWorkflowService().ForLevel("epic").ValidateStatus(status)
func ValidateEpicStatus(status string) error {
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("%w: status cannot be empty", ErrInvalidEpicStatus)
	}
	return nil
}

// ValidateFeatureStatus performs basic validation on a feature status string.
// It only checks that the status is non-empty after trimming whitespace.
//
// This function does NOT validate against workflow-defined statuses because
// the models package cannot import the workflow package (circular dependency).
// For workflow-aware validation, callers at the CLI/command layer should use:
//
//	cli.GetWorkflowService().ForLevel("feature").ValidateStatus(status)
func ValidateFeatureStatus(status string) error {
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("%w: status cannot be empty", ErrInvalidFeatureStatus)
	}
	return nil
}

// ValidateTaskStatus performs basic validation on a task status string.
// It only checks that the status is non-empty after trimming whitespace.
//
// This function does NOT validate against workflow-defined statuses because
// the models package cannot import the workflow package (circular dependency).
// For workflow-aware validation, callers at the CLI/command layer should use:
//
//	cli.GetWorkflowService().ValidateStatus(status)
//
// This basic check is still useful at the model layer to catch obviously
// invalid data (empty strings) before it reaches the database.
func ValidateTaskStatus(status string) error {
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("%w: status cannot be empty", ErrInvalidTaskStatus)
	}
	return nil
}

// ValidateAgentType validates the agent type
// Accepts any non-empty string after trimming whitespace
// Maximum length: 100 characters
func ValidateAgentType(agentType string) error {
	trimmed := strings.TrimSpace(agentType)

	if trimmed == "" {
		return fmt.Errorf("agent type cannot be empty or whitespace-only")
	}

	if len(trimmed) > 100 {
		return fmt.Errorf("agent type too long: maximum 100 characters, got %d", len(trimmed))
	}

	return nil
}

// ValidatePriority validates the priority level (for Epic and other entities)
func ValidatePriority(priority string) error {
	validPriorities := map[string]bool{
		"high":   true,
		"medium": true,
		"low":    true,
	}
	if !validPriorities[priority] {
		return fmt.Errorf("invalid priority: must be high, medium, or low, got %q", priority)
	}
	return nil
}

// ValidateDependsOn validates the JSON format of the depends_on field
func ValidateDependsOn(dependsOn string) error {
	if dependsOn == "" || dependsOn == "null" {
		return nil // Empty or null is valid
	}

	var deps []string
	if err := json.Unmarshal([]byte(dependsOn), &deps); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDependsOn, err)
	}

	// Optionally validate each dependency is a valid task key
	for _, dep := range deps {
		if err := ValidateTaskKey(dep); err != nil {
			return fmt.Errorf("invalid task key in depends_on: %w", err)
		}
	}

	return nil
}

// validNoteTypes is the canonical, ordered allowlist of note types.
// This is the SINGLE SOURCE OF TRUTH consumed by both ValidateNoteType
// (for membership checks) and the ErrInvalidNoteType error message
// (rendered dynamically), so the two cannot drift. To add or remove a
// note type, edit this slice only.
//
// Note on "review": records a code-review outcome (PASS/FAIL). The
// canonical code-review workflow emits --type=review for every entity
// type that goes through review (tasks, features, bugs, change-cards,
// …); see B027.
var validNoteTypes = []string{
	"comment",
	"decision",
	"blocker",
	"solution",
	"reference",
	"implementation",
	"testing",
	"future",
	"question",
	"rejection",
	"requirement",
	"review",
	// review-finding: one structured finding from a review gate
	// (code_review/qa/uat), with gate/round/severity/defect_class/
	// fingerprint carried in the note's metadata JSON so review
	// effectiveness is queryable instead of buried in free text.
	"review-finding",
}

// ValidNoteTypes returns a defensive copy of the canonical note-type
// allowlist for callers that need to enumerate or display valid types
// (e.g. CLI help, API schema generation).
func ValidNoteTypes() []string {
	out := make([]string, len(validNoteTypes))
	copy(out, validNoteTypes)
	return out
}

// ValidateNoteType validates the note type enum.
//
// On failure, the returned error wraps ErrInvalidNoteType and includes
// the full allowlist so the human-readable hint cannot drift from the
// validator.
func ValidateNoteType(noteType string) error {
	if !slices.Contains(validNoteTypes, noteType) {
		return fmt.Errorf("%w: must be one of [%s]; got %q",
			ErrInvalidNoteType, strings.Join(validNoteTypes, ", "), noteType)
	}
	return nil
}

// ValidateRelationshipType validates the relationship type enum
func ValidateRelationshipType(relType string) error {
	if !ValidEntityRelationshipTypeSet[EntityRelationshipType(relType)] {
		return fmt.Errorf("%w: got %q", ErrInvalidRelationshipType, relType)
	}
	return nil
}

// ValidateSessionOutcome validates a session outcome structurally. Outcomes
// carry workflow vocabulary (pass/fail/blocked/…) plus lease lifecycle values
// (released/superseded/expired), and workflow vocabulary is config-driven —
// so this validator only rejects empty/whitespace values, never enforces an
// allowlist (business validation belongs to the workflow layer).
func ValidateSessionOutcome(outcome string) error {
	if strings.TrimSpace(outcome) == "" {
		return fmt.Errorf("%w: got %q", ErrInvalidSessionOutcome, outcome)
	}
	return nil
}

// ValidateJSONArray validates that a string is a valid JSON array of strings
func ValidateJSONArray(jsonStr string) error {
	if jsonStr == "" || jsonStr == "null" {
		return nil // Empty or null is valid
	}

	var arr []string
	if err := json.Unmarshal([]byte(jsonStr), &arr); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	return nil
}
