package commands

import (
	"fmt"
	"strings"
)

// InvalidEpicKeyError returns a user-friendly error for invalid epic keys
func InvalidEpicKeyError(key string) error {
	return fmt.Errorf(`Error: invalid epic key format: %q (expected E##)

Expected: two-digit epic key (case insensitive)

Valid syntax:
  shark epic create "Epic Title"        # Creates next epic (E08, E09, etc.)
  shark epic get E07                    # Get epic by key
  shark epic get e04                    # Case insensitive
  shark feature create E07 "Feature"    # Use epic key
  shark feature create E04 "Another"    # Different epic

Note: Epic keys must be exactly two-digit (E01-E99).`, key)
}

// InvalidFeatureKeyError returns a user-friendly error for invalid feature keys
func InvalidFeatureKeyError(key string) error {
	return fmt.Errorf(`Error: invalid feature key format: %q

Expected: two-digit feature key, with optional epic prefix (case insensitive)

Valid syntax:
  shark feature create E07 "Feature Title"     # Creates next feature (F01, F02, etc.)
  shark feature get F01                        # Short format (context-dependent)
  shark feature get E07-F01                    # Full format (recommended)
  shark feature get e07-f01                    # Case insensitive
  shark task create E07 F01 "Task"             # Use in task commands
  shark task create E04 F01 "Task"             # Different epic

Note: Feature keys must be two-digit (F01-F99).
      Epic keys must be two-digit (E01-E99).`, key)
}

// InvalidTaskKeyError returns a user-friendly error for invalid task keys
func InvalidTaskKeyError(key string) error {
	return fmt.Errorf(`Error: invalid task key format: %q

Expected: Task key with epic, feature, and task number (case insensitive)

Valid syntax:
  Full format:
    T-E07-F01-001                   # Standard format
    t-e07-f01-001                   # Case insensitive

  Short format (new):
    E07-F01-001                     # Without 'T-' prefix
    e07-f01-001                     # Case insensitive

Note: Epic and feature must be two-digit (E01-E99, F01-F99).
      Task number must be three-digit (001-999).`, key)
}

// MissingArgumentsError returns a user-friendly error for missing arguments
func MissingArgumentsError(expected, got int, examples []string) error {
	var exampleStr string
	if len(examples) > 0 {
		exampleStr = "\n\nValid syntax:\n"
		for _, ex := range examples {
			exampleStr += fmt.Sprintf("  %s\n", ex)
		}
	}

	return fmt.Errorf(`Error: missing required arguments

Expected: %d arguments
Got: %d arguments%s

Run with --help for full usage information.`, expected, got, exampleStr)
}

// TooManyArgumentsError returns a user-friendly error for too many arguments
func TooManyArgumentsError(expected, got int) error {
	return fmt.Errorf(`Error: too many arguments provided

Expected: %d arguments
Got: %d arguments

Note: Make sure to quote multi-word titles and strings.
      Example: shark epic create "My Epic Title"

Run with --help for full usage information.`, expected, got)
}

// InvalidPositionalArgsError returns a user-friendly error for invalid positional argument combinations
func InvalidPositionalArgsError(command, reason string, examples []string) error {
	var exampleStr string
	if len(examples) > 0 {
		exampleStr = "\n\nValid syntax:\n"
		for _, ex := range examples {
			exampleStr += fmt.Sprintf("  %s\n", ex)
		}
	}

	return fmt.Errorf(`Error: invalid arguments for %s

Reason: %s%s

Run with --help for full usage information.`, command, reason, exampleStr)
}

// AmbiguousKeyError returns a user-friendly error when a key matches multiple entities
func AmbiguousKeyError(key string, suggestions []string) error {
	var suggestionStr string
	if len(suggestions) > 0 {
		suggestionStr = "\n\nDid you mean:\n"
		for _, s := range suggestions {
			suggestionStr += fmt.Sprintf("  %s\n", s)
		}
	}

	return fmt.Errorf(`Error: ambiguous key %q - multiple matches found%s

Tip: Use a more specific key format to avoid ambiguity.
     Example: Use E07-F01-001 instead of 001`, key, suggestionStr)
}

// NotFoundError returns a user-friendly error when an entity is not found
func NotFoundError(entityType, key string) error {
	var suggestions string
	switch strings.ToLower(entityType) {
	case "epic":
		suggestions = `
Suggestions:
  - List all epics: shark epic list
  - Check epic key format (must be E## like E07)
  - Verify epic exists in database`
	case "feature":
		suggestions = `
Suggestions:
  - List all features: shark feature list
  - List features in epic: shark feature list E07
  - Check feature key format (F## or E##-F## like F01 or E07-F01)
  - Verify feature exists in database`
	case "task":
		suggestions = `
Suggestions:
  - List all tasks: shark task list
  - List tasks in feature: shark task list E07 F01
  - Check task key format (T-E##-F##-### or E##-F##-### like T-E07-F01-001)
  - Verify task exists in database`
	}

	return fmt.Errorf(`Error: %s not found: %q%s`, entityType, key, suggestions)
}

// InvalidStatusTransitionError returns a user-friendly error for invalid status transitions
func InvalidStatusTransitionError(currentStatus, targetStatus string, allowedTransitions []string) error {
	var transitionStr string
	if len(allowedTransitions) > 0 {
		transitionStr = "\n\nAllowed transitions from " + currentStatus + ":\n"
		for _, t := range allowedTransitions {
			transitionStr += fmt.Sprintf("  - %s\n", t)
		}
	}

	return fmt.Errorf(`Error: invalid status transition

Current status: %s
Target status: %s%s

Note: Use --force flag to bypass validation (use with caution).
      Example: shark task start <key> --force`, currentStatus, targetStatus, transitionStr)
}
