package services

import "fmt"

// ValidationError is returned by TagService when user-supplied input fails
// normalization or name-format validation (REQ-F-003, REQ-F-007).
//
// CLI exit code: 3.
// Error() format: "invalid <Field>: <Message>"
type ValidationError struct {
	// Field names the input field that failed validation, e.g. "tag name",
	// "old name", or "new name".
	Field string
	// Message explains the validation constraint that was violated.
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Message)
}

// NotFoundError is returned by TagService when a tag looked up by name does
// not exist in the vocabulary (REQ-F-005, REQ-F-006, REQ-F-007).
//
// CLI exit code: 1.
// Error() format: "tag not found: <Name>"
type NotFoundError struct {
	// Name is the normalized tag name that was not found.
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("tag not found: %s", e.Name)
}

// ConflictError is returned by TagService when an AddTag or RenameTag
// operation would create a duplicate name (REQ-F-006, REQ-F-007).
//
// CLI exit code: 3.
// Error() format: "tag already exists: <Name>"
type ConflictError struct {
	// Name is the normalized tag name that already exists.
	Name string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("tag already exists: %s", e.Name)
}

// TagInUseError is returned by TagService.RemoveTag when the tag still has
// entity associations and the caller did not pass force=true (REQ-F-005, ADR-9).
//
// CLI exit code: 3.
// Error() format: "tag %q is in use by %d entities; re-run with --force to delete it and its associations"
type TagInUseError struct {
	// Name is the normalized tag name that is still in use.
	Name string
	// Count is the total number of entity_tags rows that reference this tag.
	Count int64
}

func (e *TagInUseError) Error() string {
	return fmt.Sprintf(
		"tag %q is in use by %d entities; re-run with --force to delete it and its associations",
		e.Name, e.Count,
	)
}

// UnregisteredTagError is returned by TagService.AttachMany when a name
// passes structural validation but is not present in the vocabulary
// (REQ-F-004, AC-2, AC-28). It is distinct from NotFoundError (which is
// used by vocabulary-management paths in F03) so CLI command layers can
// produce the SC-2 error shape without overloading NotFoundError's
// meaning (see ADR-F04-3).
//
// CLI exit code: 3 (maps to "unregistered_tag" per REQ-F-016).
// Error() format: "tag is not registered: <Name>"
type UnregisteredTagError struct {
	// Name is the normalized (lowercased + trimmed) tag name that was
	// not found in the vocabulary on the attach path.
	Name string
}

func (e *UnregisteredTagError) Error() string {
	return fmt.Sprintf("tag is not registered: %s", e.Name)
}

// TagRequiredError is returned by TagService.EnforceRequired when the
// given entity type is listed in Config.TagRequiredFor but the provided
// name slice is empty (REQ-F-003, AC-9, AC-28b).
//
// CLI exit code: 3 (maps to "tag_required" per REQ-F-016).
// Error() format: "at least one tag is required for <EntityType>"
type TagRequiredError struct {
	// EntityType is the string form of the entity type
	// (e.g. "task", "feature", "epic", "bug", "change", "idea")
	// matching models.EntityType.String() output.
	EntityType string
}

func (e *TagRequiredError) Error() string {
	return fmt.Sprintf("at least one tag is required for %s", e.EntityType)
}
