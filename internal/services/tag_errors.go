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
