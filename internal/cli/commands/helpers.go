package commands

import (
	"fmt"
	"regexp"
)

// Compiled regex patterns for pattern matching
var (
	epicKeyPattern       = regexp.MustCompile(`^E\d{2}$`)
	featureKeyPattern    = regexp.MustCompile(`^E\d{2}-F\d{2}$`)
	featureSuffixPattern = regexp.MustCompile(`^F\d{2}$`)
)

// IsEpicKey validates if a string is a valid epic key format (E##)
// Returns true for valid epic keys like E01, E04, E99
// Returns false for invalid formats like E1, e04, E001, etc.
func IsEpicKey(s string) bool {
	return epicKeyPattern.MatchString(s)
}

// IsFeatureKey validates if a string is a valid feature key format (E##-F##)
// Returns true for valid feature keys like E04-F01, E01-F99
// Returns false for invalid formats like E04F01, E4-F01, etc.
func IsFeatureKey(s string) bool {
	return featureKeyPattern.MatchString(s)
}

// IsFeatureKeySuffix validates if a string is a valid feature key suffix (F##)
// Returns true for valid suffixes like F01, F99
// Returns false for invalid formats like F1, f01, etc.
func IsFeatureKeySuffix(s string) bool {
	return featureSuffixPattern.MatchString(s)
}

// ParseFeatureKey parses a combined feature key format (E##-F##) into epic and feature parts
// Returns (epic, feature, nil) for valid input like "E04-F01"
// Returns ("", "", error) for invalid input with clear error message
func ParseFeatureKey(s string) (epic, feature string, err error) {
	if !IsFeatureKey(s) {
		return "", "", fmt.Errorf("invalid feature key format: %q (expected E##-F##)", s)
	}

	// Split on hyphen - we know it's valid format at this point
	// Epic is first 3 chars (E##), Feature is last 3 chars (F##)
	epic = s[:3]
	feature = s[4:7]

	return epic, feature, nil
}

// ParseFeatureListArgs parses positional arguments for feature list command
// Returns (epicKey, nil) if valid, or (nil, error) if invalid
// Handles 0 or 1 positional arguments
func ParseFeatureListArgs(args []string) (*string, error) {
	if len(args) == 0 {
		// No filter
		return nil, nil
	}

	if len(args) > 1 {
		return nil, fmt.Errorf("too many positional arguments: feature list accepts at most 1 positional argument (got %d). Use --help for syntax examples", len(args))
	}

	// Check if first arg is a valid epic key
	epicKey := args[0]
	if !IsEpicKey(epicKey) {
		return nil, fmt.Errorf("invalid epic key format: %q (expected E##, e.g., E04). Use --help for syntax examples", epicKey)
	}

	return &epicKey, nil
}

// ParseTaskListArgs parses positional arguments for task list command
// Supports 0-2 arguments with multiple syntaxes:
// - No args: list all tasks
// - 1 arg (E##): filter by epic
// - 1 arg (E##-F##): filter by epic and feature
// - 2 args (E## and F## or E##-F##): filter by epic and feature
// Returns (epicKey, featureKey, nil) on success, or (nil, nil, error) on failure
func ParseTaskListArgs(args []string) (*string, *string, error) {
	if len(args) == 0 {
		// No filter
		return nil, nil, nil
	}

	if len(args) > 2 {
		return nil, nil, fmt.Errorf("too many positional arguments: task list accepts at most 2 positional arguments (got %d). Use --help for syntax examples", len(args))
	}

	// Single argument case
	if len(args) == 1 {
		arg := args[0]

		// Check if it's a combined feature key (E##-F##)
		if IsFeatureKey(arg) {
			epic, feature, err := ParseFeatureKey(arg)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid feature key format: %q (expected E##-F##, e.g., E04-F01). Use --help for syntax examples", arg)
			}
			return &epic, &feature, nil
		}

		// Check if it's just an epic key (E##)
		if IsEpicKey(arg) {
			return &arg, nil, nil
		}

		// Invalid format
		return nil, nil, fmt.Errorf("invalid key format: %q (expected E## or E##-F##, e.g., E04 or E04-F01). Use --help for syntax examples", arg)
	}

	// Two argument case
	arg1 := args[0]
	arg2 := args[1]

	// First argument must be an epic key
	if !IsEpicKey(arg1) {
		return nil, nil, fmt.Errorf("invalid epic key format: %q (expected E##, e.g., E04). Use --help for syntax examples", arg1)
	}

	// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
	if IsFeatureKeySuffix(arg2) {
		// Just feature suffix - arg2 is already F##
		return &arg1, &arg2, nil
	}

	if IsFeatureKey(arg2) {
		// Full feature key - extract the feature suffix
		_, featureSuffix, err := ParseFeatureKey(arg2)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid feature key format: %q (expected F## or E##-F##). Use --help for syntax examples", arg2)
		}
		return &arg1, &featureSuffix, nil
	}

	// Invalid feature format
	return nil, nil, fmt.Errorf("invalid feature key format: %q (expected F## or E##-F##, e.g., F01 or E04-F01). Use --help for syntax examples", arg2)
}

// Deprecated: Use IsEpicKey instead
// isValidEpicKey validates epic key format (E##)
func isValidEpicKey(key string) bool {
	return IsEpicKey(key)
}
