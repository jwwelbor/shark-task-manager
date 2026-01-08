package commands

import (
	"fmt"
	"regexp"
	"strings"
)

// Compiled regex patterns for pattern matching
var (
	epicKeyPattern       = regexp.MustCompile(`^E\d{2}$`)
	featureKeyPattern    = regexp.MustCompile(`^E\d{2}-F\d{2}$`)
	featureSuffixPattern = regexp.MustCompile(`^F\d{2}$`)
	// shortTaskKeyPattern matches task keys without the T- prefix (E##-F##-###)
	// This enables users to use "E01-F02-001" instead of "T-E01-F02-001"
	shortTaskKeyPattern = regexp.MustCompile(`^E\d{2}-F\d{2}-\d{3}$`)
)

// NormalizeKey converts a key to canonical uppercase format.
// This enables case-insensitive key handling throughout the CLI.
//
// Examples:
//
//	e01 -> E01
//	t-e04-f02-001 -> T-E04-F02-001
//	E01-FEATURE-NAME -> E01-FEATURE-NAME
func NormalizeKey(key string) string {
	return strings.ToUpper(key)
}

// IsEpicKey validates if a string is a valid epic key format (E##)
// Case insensitive: e01, E01, and E-01 are all normalized to E01 before validation
// Returns true for valid epic keys like E01, e04, E99
// Returns false for invalid formats like E1, E001, etc.
func IsEpicKey(s string) bool {
	normalized := NormalizeKey(s)
	return epicKeyPattern.MatchString(normalized)
}

// IsFeatureKey validates if a string is a valid feature key format (E##-F##)
// Case insensitive: e04-f01, E04-F01 are normalized before validation
// Returns true for valid feature keys like E04-F01, e01-f99
// Returns false for invalid formats like E04F01, E4-F01, etc.
func IsFeatureKey(s string) bool {
	normalized := NormalizeKey(s)
	return featureKeyPattern.MatchString(normalized)
}

// IsFeatureKeySuffix validates if a string is a valid feature key suffix (F##)
// Case insensitive: f01, F01 are normalized before validation
// Returns true for valid suffixes like F01, f99
// Returns false for invalid formats like F1, etc.
func IsFeatureKeySuffix(s string) bool {
	normalized := NormalizeKey(s)
	return featureSuffixPattern.MatchString(normalized)
}

// ParseFeatureKey parses a combined feature key format (E##-F##) into epic and feature parts
// Case insensitive: normalizes input to uppercase before parsing
// Returns (epic, feature, nil) for valid input like "E04-F01" or "e04-f01"
// Returns ("", "", error) for invalid input with clear error message
func ParseFeatureKey(s string) (epic, feature string, err error) {
	// Normalize to uppercase first
	normalized := NormalizeKey(s)

	if !featureKeyPattern.MatchString(normalized) {
		return "", "", InvalidFeatureKeyError(s)
	}

	// Split on hyphen - we know it's valid format at this point
	// Epic is first 3 chars (E##), Feature is last 3 chars (F##)
	epic = normalized[:3]
	feature = normalized[4:7]

	return epic, feature, nil
}

// ParseFeatureListArgs parses positional arguments for feature list command
// Case insensitive: normalizes epic key to uppercase
// Returns (epicKey, nil) if valid, or (nil, error) if invalid
// Handles 0 or 1 positional arguments
func ParseFeatureListArgs(args []string) (*string, error) {
	if len(args) == 0 {
		// No filter
		return nil, nil
	}

	if len(args) > 1 {
		return nil, TooManyArgumentsError(1, len(args))
	}

	// Normalize the epic key
	epicKey := NormalizeKey(args[0])

	// Validate format
	if !IsEpicKey(epicKey) {
		return nil, InvalidEpicKeyError(args[0])
	}

	return &epicKey, nil
}

// ParseTaskListArgs parses positional arguments for task list command
// Case insensitive: normalizes all keys to uppercase
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
		return nil, nil, TooManyArgumentsError(2, len(args))
	}

	// Single argument case
	if len(args) == 1 {
		normalized := NormalizeKey(args[0])

		// Check if it's a combined feature key (E##-F##)
		if IsFeatureKey(normalized) {
			epic, feature, err := ParseFeatureKey(normalized)
			if err != nil {
				return nil, nil, err
			}
			return &epic, &feature, nil
		}

		// Check if it's just an epic key (E##)
		if IsEpicKey(normalized) {
			return &normalized, nil, nil
		}

		// Check if it looks like it was trying to be an epic key (starts with E)
		if len(normalized) > 0 && normalized[0] == 'E' {
			return nil, nil, InvalidEpicKeyError(args[0])
		}

		// Check if it looks like it was trying to be a feature key (contains dash or starts with F)
		if strings.Contains(normalized, "-") || (len(normalized) > 0 && normalized[0] == 'F') {
			return nil, nil, InvalidFeatureKeyError(args[0])
		}

		// Generic invalid format
		return nil, nil, InvalidPositionalArgsError("task list",
			fmt.Sprintf("invalid key format %q - expected E## or E##-F##", args[0]),
			[]string{
				"shark task list E07",
				"shark task list E07-F01",
				"shark task list e07",
			})
	}

	// Two argument case
	epicNormalized := NormalizeKey(args[0])
	featureNormalized := NormalizeKey(args[1])

	// First argument must be an epic key
	if !IsEpicKey(epicNormalized) {
		return nil, nil, InvalidEpicKeyError(args[0])
	}

	// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
	if IsFeatureKeySuffix(featureNormalized) {
		// Just feature suffix
		return &epicNormalized, &featureNormalized, nil
	}

	if IsFeatureKey(featureNormalized) {
		// Full feature key - extract the feature suffix
		_, featureSuffix, err := ParseFeatureKey(featureNormalized)
		if err != nil {
			return nil, nil, err
		}
		return &epicNormalized, &featureSuffix, nil
	}

	// Invalid feature format
	return nil, nil, InvalidFeatureKeyError(args[1])
}

// ParseListArgs parses positional arguments for the list command dispatcher
// Case insensitive: normalizes all keys to uppercase
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
		return "", nil, nil, TooManyArgumentsError(2, len(args))
	}

	// Single argument case
	if len(args) == 1 {
		normalized := NormalizeKey(args[0])

		// Check if it's a combined feature key (E##-F##)
		if IsFeatureKey(normalized) {
			epic, feature, err := ParseFeatureKey(normalized)
			if err != nil {
				return "", nil, nil, err
			}
			return "task", &epic, &feature, nil
		}

		// Check if it's just an epic key (E##)
		if IsEpicKey(normalized) {
			return "feature", &normalized, nil, nil
		}

		// Check if it looks like it was trying to be an epic key (starts with E)
		if len(normalized) > 0 && normalized[0] == 'E' {
			return "", nil, nil, InvalidEpicKeyError(args[0])
		}

		// Check if it looks like it was trying to be a feature key (contains dash or starts with F)
		if strings.Contains(normalized, "-") || (len(normalized) > 0 && normalized[0] == 'F') {
			return "", nil, nil, InvalidFeatureKeyError(args[0])
		}

		// Generic invalid format
		return "", nil, nil, InvalidPositionalArgsError("list",
			fmt.Sprintf("invalid key format %q - expected E## or E##-F##", args[0]),
			[]string{
				"shark list E07",
				"shark list E07-F01",
			})
	}

	// Two argument case
	epicNormalized := NormalizeKey(args[0])
	featureNormalized := NormalizeKey(args[1])

	// First argument must be an epic key
	if !IsEpicKey(epicNormalized) {
		return "", nil, nil, InvalidEpicKeyError(args[0])
	}

	// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
	if IsFeatureKeySuffix(featureNormalized) {
		// Just feature suffix
		return "task", &epicNormalized, &featureNormalized, nil
	}

	if IsFeatureKey(featureNormalized) {
		// Full feature key - extract the feature suffix
		_, featureSuffix, err := ParseFeatureKey(featureNormalized)
		if err != nil {
			return "", nil, nil, err
		}
		return "task", &epicNormalized, &featureSuffix, nil
	}

	// Invalid feature format
	return "", nil, nil, InvalidFeatureKeyError(args[1])
}

// ParseGetArgs parses positional arguments for the get command dispatcher
// Case insensitive: normalizes all keys to uppercase
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
		return "", "", MissingArgumentsError(1, 0, []string{
			"shark get E07",
			"shark get E07-F01",
			"shark get T-E07-F01-001",
		})
	}

	if len(args) > 3 {
		return "", "", TooManyArgumentsError(3, len(args))
	}

	// Single argument case
	if len(args) == 1 {
		normalized := NormalizeKey(args[0])

		// Check if it's a task key (T-E##-F##-###)
		if isTaskKey(normalized) {
			return "task", normalized, nil
		}

		// Check if it's a combined feature key (E##-F##)
		if IsFeatureKey(normalized) {
			return "feature", normalized, nil
		}

		// Check if it's just an epic key (E##)
		if IsEpicKey(normalized) {
			return "epic", normalized, nil
		}

		// Check if it looks like it was trying to be a task key (starts with T)
		if len(normalized) > 0 && normalized[0] == 'T' {
			return "", "", InvalidTaskKeyError(args[0])
		}

		// Check if it looks like it was trying to be an epic key (starts with E but no dash)
		if len(normalized) > 0 && normalized[0] == 'E' && !strings.Contains(normalized, "-") {
			return "", "", InvalidEpicKeyError(args[0])
		}

		// Check if it looks like it was trying to be a feature key (contains dash or starts with F)
		if strings.Contains(normalized, "-") || (len(normalized) > 0 && normalized[0] == 'F') {
			return "", "", InvalidFeatureKeyError(args[0])
		}

		// Generic invalid format
		return "", "", InvalidPositionalArgsError("get",
			fmt.Sprintf("invalid key format %q - expected E##, E##-F##, or T-E##-F##-###", args[0]),
			[]string{
				"shark get E07",
				"shark get E07-F01",
				"shark get T-E07-F01-001",
			})
	}

	// Two argument case - must be epic + feature
	if len(args) == 2 {
		epicNormalized := NormalizeKey(args[0])
		featureNormalized := NormalizeKey(args[1])

		// First argument must be an epic key
		if !IsEpicKey(epicNormalized) {
			return "", "", InvalidEpicKeyError(args[0])
		}

		// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
		var featureSuffix string
		if IsFeatureKeySuffix(featureNormalized) {
			featureSuffix = featureNormalized
		} else if IsFeatureKey(featureNormalized) {
			// Full feature key - extract the feature suffix
			_, suffix, err := ParseFeatureKey(featureNormalized)
			if err != nil {
				return "", "", err
			}
			featureSuffix = suffix
		} else {
			return "", "", InvalidFeatureKeyError(args[1])
		}

		// Construct full feature key
		fullFeatureKey := epicNormalized + "-" + featureSuffix
		return "feature", fullFeatureKey, nil
	}

	// Three argument case - must be epic + feature + task number
	epicNormalized := NormalizeKey(args[0])
	featureNormalized := NormalizeKey(args[1])
	arg3 := args[2]

	// First argument must be an epic key
	if !IsEpicKey(epicNormalized) {
		return "", "", InvalidEpicKeyError(args[0])
	}

	// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
	var featureSuffix string
	if IsFeatureKeySuffix(featureNormalized) {
		featureSuffix = featureNormalized
	} else if IsFeatureKey(featureNormalized) {
		// Full feature key - extract the feature suffix
		_, suffix, err := ParseFeatureKey(featureNormalized)
		if err != nil {
			return "", "", err
		}
		featureSuffix = suffix
	} else {
		return "", "", InvalidFeatureKeyError(args[1])
	}

	// Third argument must be a task number (1-999)
	taskNum, err := parseTaskNumber(arg3)
	if err != nil {
		return "", "", err
	}

	// Construct full task key
	fullTaskKey := fmt.Sprintf("T-%s-%s-%03d", epicNormalized, featureSuffix, taskNum)
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

// isShortTaskKey validates if a string matches the short task key pattern (E##-F##-###)
// This is a helper function for NormalizeTaskKey to detect short format task keys.
// Short format omits the T- prefix for brevity: "E01-F02-001" instead of "T-E01-F02-001"
func isShortTaskKey(s string) bool {
	return shortTaskKeyPattern.MatchString(s)
}

// NormalizeTaskKey converts a task key to canonical format with T- prefix.
// Accepts both full format (T-E##-F##-###) and short format (E##-F##-###).
// This enables users to type shorter commands while maintaining backward compatibility.
//
// Examples:
//
//	T-E01-F02-001 → T-E01-F02-001 (no change)
//	e01-f02-001 → T-E01-F02-001 (add prefix, uppercase)
//	E01-F02-001 → T-E01-F02-001 (add prefix)
//	e01-f02-001-task-name → T-E01-F02-001-TASK-NAME (slugged, add prefix)
//
// This function is part of T-E07-F20-006: Add short task key pattern and normalization
func NormalizeTaskKey(input string) (string, error) {
	if input == "" {
		return "", InvalidTaskKeyError("")
	}

	// First normalize case
	normalized := strings.ToUpper(input)

	// Already has T- prefix - validate and return
	if strings.HasPrefix(normalized, "T-") {
		// Remove T- prefix temporarily to validate the rest
		withoutPrefix := strings.TrimPrefix(normalized, "T-")
		// Extract key part (first 3 components: E##-F##-###)
		parts := strings.SplitN(withoutPrefix, "-", 4)
		if len(parts) >= 3 {
			keyPart := strings.Join(parts[:3], "-")
			if isShortTaskKey(keyPart) {
				return normalized, nil
			}
		}
		return "", InvalidTaskKeyError(input)
	}

	// Check if it matches short format (E##-F##-###) exactly
	if isShortTaskKey(normalized) {
		return "T-" + normalized, nil
	}

	// Check for slugged short format (E##-F##-###-slug)
	// We need to extract the key part and check if it's valid
	parts := strings.SplitN(normalized, "-", 4)
	if len(parts) >= 4 {
		// Parts: [E##, F##, ###, slug...]
		keyPart := strings.Join(parts[:3], "-")
		if isShortTaskKey(keyPart) {
			// Valid short key with slug - add T- prefix
			return "T-" + normalized, nil
		}
	}

	// Invalid format - return error with helpful message
	return "", InvalidTaskKeyError(input)
}

// ParseFeatureCreateArgs parses positional arguments for feature create command
// Supports: shark feature create [EPIC] "TITLE"
// Returns (epicKey, title, nil) on success, or (nil, nil, error) on failure
// Case insensitive: normalizes epic key to uppercase
func ParseFeatureCreateArgs(args []string) (*string, *string, error) {
	// Expected: 2 arguments (EPIC TITLE)
	if len(args) < 2 {
		return nil, nil, MissingArgumentsError(2, len(args), []string{
			"shark feature create E07 \"Feature Title\"",
			"shark feature create E04 \"User Management\"",
		})
	}

	if len(args) > 2 {
		return nil, nil, TooManyArgumentsError(2, len(args))
	}

	// Parse and normalize epic key
	epicKey := NormalizeKey(args[0])

	// Validate epic key format
	if !IsEpicKey(epicKey) {
		return nil, nil, InvalidEpicKeyError(args[0])
	}

	// Title is the second argument (taken as-is, no normalization)
	title := args[1]

	return &epicKey, &title, nil
}

// ParseTaskCreateArgs parses positional arguments for task create command
// Supports:
//   - shark task create [EPIC] [FEATURE] "TITLE" (3 arguments)
//   - shark task create [EPIC-FEATURE] "TITLE" (2 arguments)
//
// Returns (epicKey, featureKey, title, nil) on success, or (nil, nil, nil, error) on failure
// Case insensitive: normalizes epic and feature keys to uppercase
func ParseTaskCreateArgs(args []string) (*string, *string, *string, error) {
	// Expected: 2 or 3 arguments
	if len(args) < 2 {
		return nil, nil, nil, MissingArgumentsError(2, len(args), []string{
			"shark task create E07 F01 \"Task Title\"",
			"shark task create E07-F01 \"Task Title\"",
		})
	}

	if len(args) > 3 {
		return nil, nil, nil, TooManyArgumentsError(3, len(args))
	}

	// Case 1: 3 arguments (EPIC FEATURE TITLE)
	if len(args) == 3 {
		epicKey := NormalizeKey(args[0])
		featureArg := NormalizeKey(args[1])
		title := args[2]

		// Validate epic key format
		if !IsEpicKey(epicKey) {
			return nil, nil, nil, InvalidEpicKeyError(args[0])
		}

		// Feature can be either F## (suffix) or E##-F## (full key)
		var featureKey string
		if IsFeatureKeySuffix(featureArg) {
			// Just the suffix (F##)
			featureKey = featureArg
		} else if IsFeatureKey(featureArg) {
			// Full feature key (E##-F##) - extract suffix
			_, suffix, err := ParseFeatureKey(featureArg)
			if err != nil {
				return nil, nil, nil, InvalidFeatureKeyError(args[1])
			}
			featureKey = suffix
		} else {
			return nil, nil, nil, InvalidFeatureKeyError(args[1])
		}

		return &epicKey, &featureKey, &title, nil
	}

	// Case 2: 2 arguments (EPIC-FEATURE TITLE)
	combinedKey := NormalizeKey(args[0])
	title := args[1]

	// Must be a valid feature key (E##-F##)
	if !IsFeatureKey(combinedKey) {
		return nil, nil, nil, InvalidFeatureKeyError(args[0])
	}

	// Parse the combined key
	epicKey, featureKey, err := ParseFeatureKey(combinedKey)
	if err != nil {
		return nil, nil, nil, err
	}

	return &epicKey, &featureKey, &title, nil
}
