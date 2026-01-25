package keys

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

// Normalize converts a key to canonical uppercase format.
// This enables case-insensitive key handling throughout the CLI.
//
// Examples:
//
//	e01 -> E01
//	t-e04-f02-001 -> T-E04-F02-001
//	E01-FEATURE-NAME -> E01-FEATURE-NAME
func Normalize(key string) string {
	return strings.ToUpper(key)
}

// IsEpicKey validates if a string is a valid epic key format (E##)
// Case insensitive: e01, E01, and E-01 are all normalized to E01 before validation
// Returns true for valid epic keys like E01, e04, E99
// Returns false for invalid formats like E1, E001, etc.
func IsEpicKey(s string) bool {
	normalized := Normalize(s)
	return epicKeyPattern.MatchString(normalized)
}

// IsFeatureKey validates if a string is a valid feature key format (E##-F##)
// Case insensitive: e04-f01, E04-F01 are normalized before validation
// Returns true for valid feature keys like E04-F01, e01-f99
// Returns false for invalid formats like E04F01, E4-F01, etc.
func IsFeatureKey(s string) bool {
	normalized := Normalize(s)
	return featureKeyPattern.MatchString(normalized)
}

// IsFeatureKeySuffix validates if a string is a valid feature key suffix (F##)
// Case insensitive: f01, F01 are normalized before validation
// Returns true for valid suffixes like F01, f99
// Returns false for invalid formats like F1, etc.
func IsFeatureKeySuffix(s string) bool {
	normalized := Normalize(s)
	return featureSuffixPattern.MatchString(normalized)
}

// ParseFeatureKey parses a combined feature key format (E##-F##) into epic and feature parts
// Case insensitive: normalizes input to uppercase before parsing
// Returns (epic, feature, nil) for valid input like "E04-F01" or "e04-f01"
// Returns ("", "", error) for invalid input with clear error message
func ParseFeatureKey(s string) (epic, feature string, err error) {
	// Normalize to uppercase first
	normalized := Normalize(s)

	if !featureKeyPattern.MatchString(normalized) {
		return "", "", fmt.Errorf("invalid feature key format: %q", s)
	}

	// Split on hyphen - we know it's valid format at this point
	// Epic is first 3 chars (E##), Feature is last 3 chars (F##)
	epic = normalized[:3]
	feature = normalized[4:7]

	return epic, feature, nil
}

// IsTaskKey validates if a string is a valid task key format (T-E##-F##-###)
func IsTaskKey(s string) bool {
	// Task key format: T-E##-F##-###
	if len(s) < 13 {
		return false
	}
	if !strings.HasPrefix(s, "T-") {
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
	if len(s) >= 13 {
		taskNumStr := s[10:13]
		for _, ch := range taskNumStr {
			if ch < '0' || ch > '9' {
				return false
			}
		}
		return true
	}
	return false
}

// IsShortTaskKey validates if a string matches the short task key pattern (E##-F##-###)
// This is a helper function for NormalizeTaskKey to detect short format task keys.
// Short format omits the T- prefix for brevity: "E01-F02-001" instead of "T-E01-F02-001"
func IsShortTaskKey(s string) bool {
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
func NormalizeTaskKey(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("empty task key")
	}

	normalized := strings.ToUpper(input)

	// Already has T- prefix
	if strings.HasPrefix(normalized, "T-") {
		if IsTaskKey(normalized) {
			return normalized, nil
		}
		return "", fmt.Errorf("invalid task key format: %q", input)
	}

	// Check if it matches short format (E##-F##-###)
	if IsShortTaskKey(normalized) {
		return "T-" + normalized, nil
	}

	// Check for slugged short format (E##-F##-###-slug)
	parts := strings.SplitN(normalized, "-", 4)
	if len(parts) >= 4 {
		keyPart := strings.Join(parts[:3], "-")
		if IsShortTaskKey(keyPart) {
			return "T-" + normalized, nil
		}
	}

	return "", fmt.Errorf("invalid task key format: %q", input)
}

// ParseTaskNumber parses a task number string and validates it's in range 1-999
func ParseTaskNumber(s string) (int, error) {
	num := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid task number: %q (must be numeric 1-999)", s)
		}
		num = num*10 + int(ch-'0')
	}

	if num < 1 || num > 999 {
		return 0, fmt.Errorf("invalid task number: %q (must be between 1 and 999)", s)
	}

	return num, nil
}
