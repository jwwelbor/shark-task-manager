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

// ParseListArgs parses positional arguments for the list command dispatcher
// Returns (command, epicKey, featureKey, error)
// - command: "epic", "feature", or "task"
// - epicKey: pointer to epic key if applicable
// - featureKey: pointer to feature key if applicable
// Supports:
// - 0 args: list epics
// - 1 arg (E##): list features in epic
// - 1 arg (E##-F##): list tasks in feature
// - 2 args (E## F## or E## E##-F##): list tasks in epic+feature
func ParseListArgs(args []string) (command string, epicKey, featureKey *string, err error) {
	if len(args) == 0 {
		// No args: list epics
		return "epic", nil, nil, nil
	}

	if len(args) > 2 {
		return "", nil, nil, fmt.Errorf("too many positional arguments: list accepts at most 2 positional arguments (got %d). Use --help for syntax examples", len(args))
	}

	// Single argument case
	if len(args) == 1 {
		arg := args[0]

		// Check if it's a combined feature key (E##-F##)
		if IsFeatureKey(arg) {
			epic, feature, err := ParseFeatureKey(arg)
			if err != nil {
				return "", nil, nil, err
			}
			return "task", &epic, &feature, nil
		}

		// Check if it's just an epic key (E##)
		if IsEpicKey(arg) {
			return "feature", &arg, nil, nil
		}

		// Invalid format
		return "", nil, nil, fmt.Errorf("invalid key format: %q (expected E## or E##-F##, e.g., E04 or E04-F01). Use --help for syntax examples", arg)
	}

	// Two argument case
	arg1 := args[0]
	arg2 := args[1]

	// First argument must be an epic key
	if !IsEpicKey(arg1) {
		return "", nil, nil, fmt.Errorf("invalid epic key format: %q (expected E##, e.g., E04). Use --help for syntax examples", arg1)
	}

	// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
	if IsFeatureKeySuffix(arg2) {
		// Just feature suffix
		return "task", &arg1, &arg2, nil
	}

	if IsFeatureKey(arg2) {
		// Full feature key - extract the feature suffix
		_, featureSuffix, err := ParseFeatureKey(arg2)
		if err != nil {
			return "", nil, nil, err
		}
		return "task", &arg1, &featureSuffix, nil
	}

	// Invalid feature format
	return "", nil, nil, fmt.Errorf("invalid feature key format: %q (expected F## or E##-F##, e.g., F01 or E04-F01). Use --help for syntax examples", arg2)
}

// ParseGetArgs parses positional arguments for the get command dispatcher
// Returns (command, key, error)
// - command: "epic", "feature", or "task"
// - key: The full key to pass to the get command (E10, E10-F01, T-E10-F01-001)
// Supports:
// - 1 arg (E##): get epic
// - 1 arg (E##-F##): get feature
// - 1 arg (T-E##-F##-###): get task
// - 2 args (E## F## or E## E##-F##): get feature
// - 3 args (E## F## ### or E## F## #): get task
func ParseGetArgs(args []string) (command string, key string, err error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("missing argument: get requires at least one argument. Use --help for syntax examples")
	}

	if len(args) > 3 {
		return "", "", fmt.Errorf("too many positional arguments: get accepts at most 3 positional arguments (got %d). Use --help for syntax examples", len(args))
	}

	// Single argument case
	if len(args) == 1 {
		arg := args[0]

		// Check if it's a task key (T-E##-F##-###)
		if isTaskKey(arg) {
			return "task", arg, nil
		}

		// Check if it's a combined feature key (E##-F##)
		if IsFeatureKey(arg) {
			return "feature", arg, nil
		}

		// Check if it's just an epic key (E##)
		if IsEpicKey(arg) {
			return "epic", arg, nil
		}

		// Invalid format
		return "", "", fmt.Errorf("invalid key format: %q (expected E##, E##-F##, or T-E##-F##-###). Use --help for syntax examples", arg)
	}

	// Two argument case - must be epic + feature
	if len(args) == 2 {
		arg1 := args[0]
		arg2 := args[1]

		// First argument must be an epic key
		if !IsEpicKey(arg1) {
			return "", "", fmt.Errorf("invalid epic key format: %q (expected E##, e.g., E04). Use --help for syntax examples", arg1)
		}

		// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
		var featureSuffix string
		if IsFeatureKeySuffix(arg2) {
			featureSuffix = arg2
		} else if IsFeatureKey(arg2) {
			// Full feature key - extract the feature suffix
			_, suffix, err := ParseFeatureKey(arg2)
			if err != nil {
				return "", "", err
			}
			featureSuffix = suffix
		} else {
			return "", "", fmt.Errorf("invalid feature key format: %q (expected F## or E##-F##, e.g., F01 or E04-F01). Use --help for syntax examples", arg2)
		}

		// Construct full feature key
		fullFeatureKey := arg1 + "-" + featureSuffix
		return "feature", fullFeatureKey, nil
	}

	// Three argument case - must be epic + feature + task number
	arg1 := args[0]
	arg2 := args[1]
	arg3 := args[2]

	// First argument must be an epic key
	if !IsEpicKey(arg1) {
		return "", "", fmt.Errorf("invalid epic key format: %q (expected E##, e.g., E04). Use --help for syntax examples", arg1)
	}

	// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
	var featureSuffix string
	if IsFeatureKeySuffix(arg2) {
		featureSuffix = arg2
	} else if IsFeatureKey(arg2) {
		// Full feature key - extract the feature suffix
		_, suffix, err := ParseFeatureKey(arg2)
		if err != nil {
			return "", "", err
		}
		featureSuffix = suffix
	} else {
		return "", "", fmt.Errorf("invalid feature key format: %q (expected F## or E##-F##, e.g., F01 or E04-F01). Use --help for syntax examples", arg2)
	}

	// Third argument must be a task number (1-999)
	taskNum, err := parseTaskNumber(arg3)
	if err != nil {
		return "", "", err
	}

	// Construct full task key
	fullTaskKey := fmt.Sprintf("T-%s-%s-%03d", arg1, featureSuffix, taskNum)
	return "task", fullTaskKey, nil
}

// isTaskKey validates if a string is a valid task key format (T-E##-F##-###)
func isTaskKey(s string) bool {
	// Task key format: T-E##-F##-###
	// Example: T-E04-F01-001
	// Length: T(1) -(1) E##(3) -(1) F##(3) -(1) ###(3) = 13 characters
	if len(s) != 13 {
		return false
	}
	if s[0] != 'T' || s[1] != '-' {
		return false
	}
	// Check epic part (E##)
	if !IsEpicKey(s[2:5]) {
		return false
	}
	if s[5] != '-' {
		return false
	}
	// Check feature part (F##)
	if !IsFeatureKeySuffix(s[6:9]) {
		return false
	}
	if s[9] != '-' {
		return false
	}
	// Check task number part (###)
	taskNumStr := s[10:]
	if len(taskNumStr) != 3 {
		return false
	}
	// All characters must be digits
	for _, ch := range taskNumStr {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// parseTaskNumber parses a task number string and validates it's in range 1-999
func parseTaskNumber(s string) (int, error) {
	// Parse as integer
	num := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid task number: %q (must be numeric 1-999)", s)
		}
		num = num*10 + int(ch-'0')
	}

	// Validate range
	if num < 1 || num > 999 {
		return 0, fmt.Errorf("invalid task number: %q (must be between 1 and 999)", s)
	}

	return num, nil
}

// Deprecated: Use IsEpicKey instead
// isValidEpicKey validates epic key format (E##)
func isValidEpicKey(key string) bool {
	return IsEpicKey(key)
}
